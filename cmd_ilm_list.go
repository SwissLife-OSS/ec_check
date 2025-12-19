package main

import (
	"context"
	"fmt"
	"hash/fnv"
	"os"
	"slices"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/docker/go-units"
	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/bytes"
	"github.com/olekukonko/tablewriter"
	"github.com/urfave/cli/v3"
)

func ilmList(ctx context.Context, cmd *cli.Command) error {
	deployment := cmd.String("deployment")
	region := cmd.String("region")
	username := cmd.String("username")
	password := cmd.String("password")

	phase := cmd.String("phase")
	ilmPolicy := cmd.String("ilm-policy")
	sortColumns := cmd.StringSlice("sort")
	minSizeStr := cmd.String("min-size")
	minAge := time.Duration(cmd.Int("min-age-days")) * 24 * time.Hour

	if !isRegionValid(region) {
		return fmt.Errorf("region %q is not a known Elastic Cloud region", region)
	}

	regionParts := strings.Split(region, "-")
	if len(regionParts) != 2 {
		return fmt.Errorf(`invalid region, expected format "<provider>-<region>", e.g. "azure-westeurope"`)
	}

	allowedSortColumns := []string{"age", "size"}
	for _, sortColumn := range sortColumns {
		if !slices.Contains(allowedSortColumns, sortColumn) {
			return fmt.Errorf("column %q is not allowed for sorting, use one of %v", sortColumn, allowedSortColumns)
		}
	}

	var minSize int64
	var err error
	if minSizeStr != "" {
		minSize, err = units.FromHumanSize(minSizeStr)
		if err != nil {
			return fmt.Errorf("failed to parse minimum size: %w", err)
		}
	}

	provider := regionParts[0]
	providerRegion := regionParts[1]

	deploymentURL := fmt.Sprintf("https://%s.es.%s.%s.elastic-cloud.com", deployment, providerRegion, provider)

	client, err := elasticsearch.NewTypedClient(elasticsearch.Config{
		Addresses: []string{
			deploymentURL,
		},
		Username: username,
		Password: password,
	})
	if err != nil {
		return err
	}

	indices, err := client.Cat.Indices().H("index", "store.size").Bytes(bytes.B).Do(ctx)
	if err != nil {
		return err
	}

	sizes := make(map[string]int64, len(indices))

	for _, index := range indices {
		size, err := strconv.ParseInt(*index.StoreSize, 10, 64)
		if err != nil {
			return err
		}

		sizes[*index.Index] = size
	}

	ilms, err := client.Ilm.ExplainLifecycle("_all").OnlyManaged(true).Do(ctx)
	if err != nil {
		return err
	}

	type indexDetails struct {
		name   string
		phase  string
		action string
		step   string
		policy string
		age    time.Duration
		size   int64
	}

	indexILM := make([]indexDetails, 0, len(ilms.Indices))

	for index, ilm := range ilms.Indices {
		managed, ok := ilm.(*types.LifecycleExplainManaged)
		if !ok {
			continue
		}

		if phase != "" && phase != *managed.Phase {
			continue
		}

		if ilmPolicy != "" && ilmPolicy != *managed.Policy {
			continue
		}

		size := sizes[index]

		if size < minSize {
			continue
		}

		age, err := parseESDuration(managed.Age)
		if err != nil {
			return err
		}

		if age < minAge {
			continue
		}

		indexILM = append(indexILM, indexDetails{name: index, phase: *managed.Phase, action: *managed.Action, step: *managed.Step, policy: *managed.Policy, age: age, size: size})
	}

	// Apply the sort criteria as less functions controlled by:
	// - might the result contain multiple phases
	// - provided sort columns, applied in order
	lessFuncs := []func(i, j int) (final, less bool){}
	if phase == "" {
		lessFuncs = append(lessFuncs, func(i, j int) (final bool, less bool) {
			if indexILM[i].phase != indexILM[j].phase {
				return true, phaseLess(indexILM[i].phase, indexILM[j].phase)
			}

			return false, false
		})
	}

	for _, sortColumn := range sortColumns {
		switch sortColumn {
		case "age":
			lessFuncs = append(lessFuncs, func(i, j int) (final bool, less bool) {
				if indexILM[i].age != indexILM[j].age {
					return true, indexILM[i].age > indexILM[j].age
				}

				return false, false
			})

		case "size":
			lessFuncs = append(lessFuncs, func(i, j int) (final bool, less bool) {
				if indexILM[i].size != indexILM[j].size {
					return true, indexILM[i].size > indexILM[j].size
				}

				return false, false
			})
		}
	}

	// Sort indices by the lessFuncs.
	sort.Slice(indexILM, func(i, j int) bool {
		for _, lessFunc := range lessFuncs {
			final, less := lessFunc(i, j)
			if final {
				return less
			}
		}

		return indexILM[i].name < indexILM[j].name
	})

	data := make([][]string, 0, len(indexILM))
	for _, item := range indexILM {
		data = append(data, []string{item.name, item.phase, item.action, item.step, item.policy, formatDuration(item.age), units.BytesSize(float64(item.size))})
	}

	table := tablewriter.NewWriter(os.Stdout)
	table.Header([]string{
		"Index", "Phase", "Action", "Step", "Policy", "Age", "Size",
	})
	err = table.Bulk(data)
	if err != nil {
		return err
	}

	err = table.Render()
	if err != nil {
		return err
	}

	return nil
}

// parseESDuration parses the duration format of Elasticsearch to a Go duration.
// ES duration does support days, which are not supported by Go durations. For
// simplicity reasons, days are just converted to 24 hours. This is not the correct
// thing to do in all cases (e.g. daylight saving), but these execeptions are
// accepted in this case, since it is not expected, that this difference will
// matter for the use case at hand.
func parseESDuration(esDuration types.Duration) (time.Duration, error) {
	durationStr, ok := esDuration.(string)
	if !ok {
		return 0, fmt.Errorf("unexpected type for types.Duration, got %T, want: string", esDuration)
	}

	if strings.HasSuffix(durationStr, "d") {
		// Days are not supported by go durations, handle it manually
		durationStr = strings.TrimSuffix(durationStr, "d")
		days, err := strconv.ParseFloat(durationStr, 64)
		if err != nil {
			return 0, err
		}

		return time.Duration(days) * 24 * time.Hour, nil
	}

	return time.ParseDuration(durationStr)
}

const (
	day  = time.Minute * 60 * 24
	year = 365 * day
)

// formatDuration returns a given formatDuration rounded as formatted string
// with a focus on days and years.
// The rounding has the following logic:
// - If the formatDuration is less than a day, use the standard formatting from Go (hours, minutes and seconds, leading units, which are 0 are omitted).
// - If the formatDuration more than a year, prepend the number of years. A year is always considered to be 365 days, leap years are ignored.
// - Append the remainder of days, omit any more fine grained units.
func formatDuration(d time.Duration) string {
	if d < day {
		return d.String()
	}

	var b strings.Builder
	if d >= year {
		years := d / year
		fmt.Fprintf(&b, "%dy", years)
		d -= years * year
	}

	days := d / day
	fmt.Fprintf(&b, "%dd", days)

	return b.String()
}

var phaseOrder = map[string]int64{
	"hot":    0,
	"warm":   1,
	"cold":   2,
	"frozen": 3,
}

// phaseLess returns if phase a has the lower order than phase b.
// Unknown phases are sorted at the end, same phases properly grouped together.
func phaseLess(a, b string) bool {
	var aa int64
	var bb int64
	var ok bool

	aa, ok = phaseOrder[a]
	if !ok {
		h := fnv.New32a()
		h.Write([]byte(a))
		aa = 1<<32 + int64(h.Sum32())
	}

	bb, ok = phaseOrder[b]
	if !ok {
		h := fnv.New32a()
		h.Write([]byte(b))
		bb = 1<<32 + int64(h.Sum32())
	}

	return aa < bb
}
