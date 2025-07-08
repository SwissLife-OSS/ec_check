package main

import (
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strconv"
	"strings"

	"github.com/docker/go-units"
)

// Allocation represents the disk allocation information as returned from the
// Elasticsearch _cat API.
type Allocation struct {
	DiskUsed  string `json:"disk.used"`
	DiskTotal string `json:"disk.total"`
	NodeRole  string `json:"node.role"`
}

// tierConfig contains the specification as well as the current usage for a Tier.
// This is calculated from the current allocation information and the TierSize
// information.
type tierConfig struct {
	NodeSizeIndex        int
	NodeSizeDiskConfig   float64
	NodeSizeMemoryConfig float64
	NodeCount            int
	TotalDiskUsage       float64
}

// Recommendation contains the final result of the calculation with the
// current sizing and the proposed sizing, if a downscaling is recommended.
type Recommendation struct {
	tier Tier

	currentNodes            int
	currentDiskPerNode      float64
	currentMemoryPerNode    float64
	currentDiskTotal        float64
	currentConsumption      float64
	requiredHeadroomPercent float64

	smallerNodes                int
	smallerDiskPerNode          float64
	smallerMemoryPerNode        float64
	smallerDiskTotal            float64
	smallerFreeAfterDownsize    float64
	smallerFreeAfterDownsizePct float64

	isAlreadySmallest        bool
	isDownscalingRecommended bool
}

func (r Recommendation) String() string {
	str := strings.Builder{}

	str.WriteString(fmt.Sprintf("Tier: %s\n", r.tier))
	str.WriteString(fmt.Sprintf("Current Config: %d nodes with %s disk (%s memory) each = %s total\n", r.currentNodes, units.BytesSize(r.currentDiskPerNode), units.BytesSize(r.currentMemoryPerNode), units.BytesSize(r.currentDiskTotal)))

	str.WriteString(fmt.Sprintf("Current Consumption: %s\n", units.BytesSize(r.currentConsumption)))

	if r.isAlreadySmallest {
		str.WriteString("Already on smallest size of the tier, no downsizing possible.")
		return str.String()
	}

	str.WriteString(fmt.Sprintf("Next smaller: %d nodes with %s disk (%s memory) each = %s total\n", r.smallerNodes, units.BytesSize(r.smallerDiskPerNode), units.BytesSize(r.smallerMemoryPerNode), units.BytesSize(r.smallerDiskTotal)))

	freeAfterDownsize := "does not fit"
	freeAfterDownsizePct := "- %"
	if r.smallerFreeAfterDownsize > 0 {
		freeAfterDownsize = units.BytesSize(r.smallerFreeAfterDownsize)
		freeAfterDownsizePct = fmt.Sprintf("%.1f%%", r.smallerFreeAfterDownsizePct)
	}

	str.WriteString(fmt.Sprintf("Free space after downsize: %s (%s)\n", freeAfterDownsize, freeAfterDownsizePct))

	str.WriteString(fmt.Sprintf("Downsize of tier recommended: %t\n", r.isDownscalingRecommended))

	return str.String()
}

// Recommendations contains a Recommendation for each Tier.
type Recommendations map[Tier]Recommendation

// IsDownscalingRecommended returns true, if for at least one of the included Tiers
// downscaling is recommended. Otherwise false is returned.
func (r Recommendations) IsDownscalingRecommended() bool {
	for _, recommendation := range r {
		if recommendation.isDownscalingRecommended {
			return true
		}
	}
	return false
}

func (r Recommendations) String() string {
	str := strings.Builder{}
	for _, recommendation := range mapOrderedByKey(r) {
		str.WriteString(recommendation.String())
		str.WriteByte('\n')
	}

	return str.String()
}

func getAllocationInformation(baseURL string) ([]Allocation, error) {
	resp, err := http.Get(baseURL + "/_cat/allocation?h=disk.used,disk.total,node.role&bytes=b&format=json")
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

	var allocations []Allocation
	err = json.Unmarshal(body, &allocations)
	if err != nil {
		return nil, err
	}

	return allocations, nil
}

func calcDownscaleRecommendation(allocations []Allocation, tierSizes TierSizes, headroomPercent float64, recommendZoneChange bool) Recommendations {
	if recommendZoneChange {
		return calcDownscaleRecommendationWithZoneChange(allocations, tierSizes, headroomPercent)
	}

	tiers := tierConfigMapping(allocations, tierSizes)

	recommendations := make(map[Tier]Recommendation, 4)
	for tier, tierCfg := range tiers {
		recommend := Recommendation{
			tier:                    tier,
			currentNodes:            tierCfg.NodeCount,
			currentDiskPerNode:      tierCfg.NodeSizeDiskConfig,
			currentDiskTotal:        float64(tierCfg.NodeCount) * tierCfg.NodeSizeDiskConfig,
			currentMemoryPerNode:    tierCfg.NodeSizeMemoryConfig,
			currentConsumption:      tierCfg.TotalDiskUsage,
			requiredHeadroomPercent: headroomPercent,

			smallerNodes:         tierCfg.NodeCount,
			smallerDiskPerNode:   tierCfg.NodeSizeDiskConfig,
			smallerMemoryPerNode: tierCfg.NodeSizeMemoryConfig,
		}

		if tierCfg.NodeSizeIndex == 0 {
			recommend.isAlreadySmallest = true
			recommendations[tier] = recommend
			continue
		}

		steps := 1
		for {
			optimizedSmallerNodeCount := float64(tierCfg.NodeCount)
			optimizedSmallerSizeDisk := tierSizes[tier][tierCfg.NodeSizeIndex-steps].Disk
			optimizedSmallerSizeMemory := tierSizes[tier][tierCfg.NodeSizeIndex-steps].Memory
			optimizedFreeAfterDownsize := (optimizedSmallerNodeCount * optimizedSmallerSizeDisk) - tierCfg.TotalDiskUsage
			optimizedFreeAfterDownsizePct := 100.0 / (optimizedSmallerNodeCount * optimizedSmallerSizeDisk) * optimizedFreeAfterDownsize

			if optimizedFreeAfterDownsizePct < recommend.requiredHeadroomPercent {
				recommend.isDownscalingRecommended = steps > 1
				break
			}

			recommend.smallerNodes = int(optimizedSmallerNodeCount)
			recommend.smallerDiskPerNode = optimizedSmallerSizeDisk
			recommend.smallerMemoryPerNode = optimizedSmallerSizeMemory
			recommend.smallerDiskTotal = optimizedSmallerNodeCount * optimizedSmallerSizeDisk

			recommend.smallerFreeAfterDownsize = optimizedFreeAfterDownsize
			recommend.smallerFreeAfterDownsizePct = optimizedFreeAfterDownsizePct

			steps++

			if tierCfg.NodeSizeIndex-steps < 0 {
				recommend.isDownscalingRecommended = true
				break
			}
		}

		recommendations[tier] = recommend
	}

	return recommendations
}

func calcDownscaleRecommendationWithZoneChange(allocations []Allocation, tierSizes TierSizes, headroomPercent float64) Recommendations {
	tiers := tierConfigMapping(allocations, tierSizes)

	recommendations := make(map[Tier]Recommendation, 4)
	for tier, tierCfg := range tiers {
		recommend := Recommendation{
			tier:                    tier,
			currentNodes:            tierCfg.NodeCount,
			currentDiskPerNode:      tierCfg.NodeSizeDiskConfig,
			currentDiskTotal:        float64(tierCfg.NodeCount) * tierCfg.NodeSizeDiskConfig,
			currentMemoryPerNode:    tierCfg.NodeSizeMemoryConfig,
			currentConsumption:      tierCfg.TotalDiskUsage,
			requiredHeadroomPercent: headroomPercent,

			smallerNodes:         tierCfg.NodeCount,
			smallerDiskPerNode:   tierCfg.NodeSizeDiskConfig,
			smallerMemoryPerNode: tierCfg.NodeSizeMemoryConfig,
		}

		if tierCfg.NodeSizeIndex == 0 && tierCfg.NodeCount <= 2 {
			recommend.isAlreadySmallest = true
			recommendations[tier] = recommend
			continue
		}

		steps := 0
		if tierCfg.NodeCount%2 == 0 {
			steps = 1
		}
		for {
			var optimizedSmallerNodeCount float64
			var optimizedSmallerSizeDisk float64
			var optimizedSmallerSizeMemory float64
			switch {
			case recommend.smallerNodes == 1:
				// if current node count == 1, optimal next smaller is same smaller size with 1 node
				optimizedSmallerNodeCount = 1.0
				optimizedSmallerSizeDisk = tierSizes[tier][tierCfg.NodeSizeIndex-steps].Disk
				optimizedSmallerSizeMemory = tierSizes[tier][tierCfg.NodeSizeIndex-steps].Memory
			case recommend.smallerNodes%2 == 0:
				// if current node count is even, optimal next smaller is next smaller size with 3 nodes
				optimizedSmallerNodeCount = 3.0
				optimizedSmallerSizeDisk = tierSizes[tier][tierCfg.NodeSizeIndex-steps].Disk
				optimizedSmallerSizeMemory = tierSizes[tier][tierCfg.NodeSizeIndex-steps].Memory
			default:
				// if current node count is odd, optimal next smaller is same node size with 2 nodes
				optimizedSmallerNodeCount = 2.0
				optimizedSmallerSizeDisk = tierSizes[tier][tierCfg.NodeSizeIndex-steps].Disk
				optimizedSmallerSizeMemory = tierSizes[tier][tierCfg.NodeSizeIndex-steps].Memory
			}

			optimizedFreeAfterDownsize := (optimizedSmallerNodeCount * optimizedSmallerSizeDisk) - tierCfg.TotalDiskUsage
			optimizedFreeAfterDownsizePct := 100.0 / (optimizedSmallerNodeCount * optimizedSmallerSizeDisk) * optimizedFreeAfterDownsize

			if optimizedFreeAfterDownsizePct < recommend.requiredHeadroomPercent {
				recommend.isDownscalingRecommended = (steps > 1 || recommend.currentNodes != recommend.smallerNodes)
				break
			}

			recommend.smallerNodes = int(optimizedSmallerNodeCount)
			recommend.smallerDiskPerNode = optimizedSmallerSizeDisk
			recommend.smallerMemoryPerNode = optimizedSmallerSizeMemory
			recommend.smallerDiskTotal = optimizedSmallerNodeCount * optimizedSmallerSizeDisk

			recommend.smallerFreeAfterDownsize = optimizedFreeAfterDownsize
			recommend.smallerFreeAfterDownsizePct = optimizedFreeAfterDownsizePct

			// If current node count is not a multiple of 3, the next smaller needs to try the smaller node size.
			// FIXME: a 6 node cluster might get detected wrongly, because it is not
			// clear, if it is 2 zones 3 nodes each or 3 zones 2 nodes each.
			if recommend.smallerNodes%3 != 0 {
				steps++
			}

			if tierCfg.NodeSizeIndex-steps < 0 && recommend.smallerNodes < 3 {
				recommend.isDownscalingRecommended = true
				break
			}
		}

		recommendations[tier] = recommend
	}

	return recommendations
}

// tierConfigMapping maps the current allocations to the Tiers and determines
// the current instance configuration based on the total disk size.
// The total disk size is used here, since the allocations API does not provide
// details about the memory configuration, which would allow for an exact
// matching. I practice, this approach has not caused any issues.
func tierConfigMapping(allocations []Allocation, tierSizes TierSizes) map[Tier]tierConfig {
	tiersAllocations := make(map[Tier][]Allocation, 10)
	tiers := make(map[Tier]tierConfig)
	for _, alloc := range allocations {
		if alloc.NodeRole == "" {
			continue
		}

		tier := tierFromNodeRole(alloc.NodeRole)

		diskTotal, _ := strconv.ParseFloat(alloc.DiskTotal, 64)

		sizeIndex := tierSize(tierSizes, tier, diskTotal)

		allocs, ok := tiersAllocations[tier]
		if !ok {
			allocs = []Allocation{}
		}

		allocs = append(allocs, alloc)

		tiersAllocations[tier] = allocs

		diskUsage, _ := strconv.ParseFloat(alloc.DiskUsed, 64)

		tierCfg := tiers[tier]
		tierCfg.NodeSizeIndex = sizeIndex
		tierCfg.NodeSizeDiskConfig = tierSizes[tier][sizeIndex].Disk
		tierCfg.NodeSizeMemoryConfig = tierSizes[tier][sizeIndex].Memory
		tierCfg.NodeCount++
		tierCfg.TotalDiskUsage += diskUsage
		tiers[tier] = tierCfg
	}

	return tiers
}

func tierFromNodeRole(nodeRole string) (tier Tier) {
	switch {
	case strings.Contains(nodeRole, "h"):
		return Tier("hot")
	case strings.Contains(nodeRole, "w"):
		return Tier("warm")
	case strings.Contains(nodeRole, "c"):
		return Tier("cold")
	case strings.Contains(nodeRole, "f"):
		return Tier("frozen")
	default:
		panic(fmt.Sprintf("tier undefined for %q", nodeRole))
	}
}

func tierSize(tierSizes TierSizes, tier Tier, diskTotalBytes float64) int {
	minDelta := math.MaxFloat64
	index := 0
	sizes, ok := tierSizes[tier]
	if !ok {
		return 0
	}

	for i, size := range sizes {
		d := delta(size.Disk, diskTotalBytes)
		if d < minDelta {
			minDelta = d
			index = i
		}
	}

	return index
}

func delta(a, b float64) float64 {
	if a > b {
		return a - b
	}

	return b - a
}
