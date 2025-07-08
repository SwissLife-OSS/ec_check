package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"

	"github.com/docker/go-units"
)

type Tier string

const (
	tierHot    Tier = "hot"
	tierWarm   Tier = "warm"
	tierCold   Tier = "cold"
	tierFrozen Tier = "frozen"
)

func (t Tier) String() string {
	return string(t)
}

type TierSizes map[Tier][]Size

type Size struct {
	Disk   float64
	Memory float64
}

func (t TierSizes) String() string {
	str := strings.Builder{}

	str.WriteString("Disk sizes per tier:\n")

	for tierName, tierSizes := range mapOrderedByKey(t) {
		str.WriteString(tierName.String())
		str.WriteString(":\n")

		first := true
		for _, size := range tierSizes {
			if !first {
				str.WriteString(", ")
			}

			str.WriteString("disk: ")
			str.WriteString(units.BytesSize(size.Disk))

			str.WriteString(" (memory: ")
			str.WriteString(units.BytesSize(size.Memory))
			str.WriteString(")")

			first = false
		}

		str.WriteByte('\n')
	}

	str.WriteByte('\n')

	return str.String()
}

type DeploymentTemplate struct {
	ID                     string                  `json:"id"`
	InstanceConfigurations []InstanceConfiguration `json:"instance_configurations"`
}

type InstanceConfiguration struct {
	ID                string            `json:"id"`
	Name              string            `json:"name"`
	ConfigVersion     int               `json:"config_version"`
	Description       string            `json:"description"`
	InstanceType      string            `json:"instance_type"`
	NodeTypes         []string          `json:"node_types"`
	DiscreteSizes     DiscreteSizes     `json:"discrete_sizes"`
	StorageMultiplier float64           `json:"storage_multiplier"`
	CPUMultiplier     float64           `json:"cpu_multiplier"`
	Metadata          map[string]string `json:"metadata"`
}

type DiscreteSizes struct {
	Sizes       []int  `json:"sizes"`
	DefaultSize int    `json:"default_size"`
	Resource    string `json:"resouce"`
}

const maxNodesPerTier = 32

const mibMultiplier = 1024 * 1024

var tierRegexp = regexp.MustCompile(`\.es\.data([^\.]+)\.`)

// getTierSizes extracts the instance configurations from the deployment templates
// provided by Elastic Cloud and returns the available sizing configurations
// (memory and disk) for each Tier.
func getTierSizes(region string, profile string) (TierSizes, error) {
	deploymentTemplateURL := fmt.Sprintf("https://api.elastic-cloud.com/api/v1/deployments/templates/%s?region=%s", profile, region)

	resp, err := http.Get(deploymentTemplateURL)
	if err != nil {
		return nil, err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var deploymentTemplate DeploymentTemplate
	err = json.Unmarshal(body, &deploymentTemplate)
	if err != nil {
		return nil, err
	}

	tierSizes := make(TierSizes, 4)
	for _, template := range deploymentTemplate.InstanceConfigurations {
		tierMatches := tierRegexp.FindStringSubmatch(template.ID)
		if len(tierMatches) != 2 {
			continue
		}

		tier := Tier(tierMatches[1])

		sizes := make([]Size, 0, len(template.DiscreteSizes.Sizes)+maxNodesPerTier-1)

		// Partial nodes with <= 64 MB of memory.
		for _, size := range template.DiscreteSizes.Sizes {
			sizes = append(sizes,
				Size{
					Memory: float64(size) * mibMultiplier,
					Disk:   float64(size) * template.StorageMultiplier * mibMultiplier,
				},
			)
		}

		fullNodeDiskSize := float64(template.DiscreteSizes.Sizes[len(template.DiscreteSizes.Sizes)-1]) * template.StorageMultiplier * mibMultiplier

		// Full nodes, adding adding 64 MB of memory each to the cluster.
		for size := 2.0; size <= maxNodesPerTier; size++ {
			sizes = append(sizes,
				Size{
					Memory: size,
					Disk:   size * fullNodeDiskSize,
				},
			)
		}

		tierSizes[tier] = sizes
	}

	return tierSizes, nil
}
