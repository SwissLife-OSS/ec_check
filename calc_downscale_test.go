package main

import (
	"fmt"
	"testing"

	"github.com/docker/go-units"
	"github.com/stretchr/testify/require"
)

var tierSizes = TierSizes{
	tierHot: []Size{
		{
			Memory: 1024 * mibMultiplier,
			Disk:   1024 * 35.0 * mibMultiplier,
		},
		{
			Memory: 2048 * mibMultiplier,
			Disk:   2048 * 35.0 * mibMultiplier,
		},
		{
			Memory: 4096 * mibMultiplier,
			Disk:   4096 * 35.0 * mibMultiplier,
		},
		{
			Memory: 8192 * mibMultiplier,
			Disk:   8192 * 35.0 * mibMultiplier,
		},
		{
			Memory: 15360 * mibMultiplier,
			Disk:   15360 * 35.0 * mibMultiplier,
		},
		{
			Memory: 30720 * mibMultiplier,
			Disk:   30720 * 35.0 * mibMultiplier,
		},
		{
			Memory: 61440 * mibMultiplier,
			Disk:   61440 * 35.0 * mibMultiplier,
		},
		{
			Memory: 61440 * 2 * mibMultiplier,
			Disk:   61440 * 2 * 35.0 * mibMultiplier,
		},
		{
			Memory: 61440 * 3 * mibMultiplier,
			Disk:   61440 * 3 * 35.0 * mibMultiplier,
		},
	},
}

func Test_calcDownscaleRecommendation(t *testing.T) {
	tests := []struct {
		zoneCount           int
		ram                 float64
		diskUsage           float64
		recommendZoneChange bool

		wantIsAlreadySmallest        bool
		wantIsDownscalingRecommended bool
		wantSmallerNodes             int
		wantSmallerMemoryPerNode     float64
	}{
		// Cluster with single node per zone.
		{
			zoneCount:           1,
			ram:                 1,
			diskUsage:           0.01,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        true,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 1,
			diskUsage:           0.01,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        true,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 1,
			diskUsage:           0.01,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        true,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             3,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           1,
			ram:                 4,
			diskUsage:           0.37,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     2048 * mibMultiplier,
		},
		{
			zoneCount:           1,
			ram:                 4,
			diskUsage:           0.38,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     4096 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 4,
			diskUsage:           0.37,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     2048 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 4,
			diskUsage:           0.38,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     4096 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 4,
			diskUsage:           0.37,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             3,
			wantSmallerMemoryPerNode:     2048 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 4,
			diskUsage:           0.38,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             3,
			wantSmallerMemoryPerNode:     4096 * mibMultiplier,
		},

		// Cluster with more than 1 node per zone.
		{
			zoneCount:           1,
			ram:                 180,
			diskUsage:           0.50,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     120 * 1024 * mibMultiplier,
		},
		{
			zoneCount:           1,
			ram:                 180,
			diskUsage:           0.51,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     180 * 1024 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 180,
			diskUsage:           0.50,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     120 * 1024 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 180,
			diskUsage:           0.51,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     180 * 1024 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 180,
			diskUsage:           0.50,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             3,
			wantSmallerMemoryPerNode:     120 * 1024 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 180,
			diskUsage:           0.51,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             3,
			wantSmallerMemoryPerNode:     180 * 1024 * mibMultiplier,
		},

		// Multiple downscale steps possible
		{
			zoneCount:           1,
			ram:                 15,
			diskUsage:           0.0001,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 15,
			diskUsage:           0.0001,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 15,
			diskUsage:           0.0001,
			recommendZoneChange: false,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             3,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},

		// Recommend Zone Change
		{
			zoneCount:           1,
			ram:                 1,
			diskUsage:           0.01,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        true,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 1,
			diskUsage:           0.01,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        true,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 1,
			diskUsage:           0.01,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           1,
			ram:                 4,
			diskUsage:           0.37,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     2048 * mibMultiplier,
		},
		{
			zoneCount:           1,
			ram:                 4,
			diskUsage:           0.38,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     4096 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 4,
			diskUsage:           0.56,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             3,
			wantSmallerMemoryPerNode:     2048 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 4,
			diskUsage:           0.57,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     4096 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 4,
			diskUsage:           0.50,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     4096 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 4,
			diskUsage:           0.51,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: false,
			wantSmallerNodes:             3,
			wantSmallerMemoryPerNode:     4096 * mibMultiplier,
		},

		// Recommend Zone Change - multiple downscale steps possible
		{
			zoneCount:           1,
			ram:                 15,
			diskUsage:           0.0001,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             1,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           2,
			ram:                 15,
			diskUsage:           0.0001,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
		{
			zoneCount:           3,
			ram:                 15,
			diskUsage:           0.0001,
			recommendZoneChange: true,

			wantIsAlreadySmallest:        false,
			wantIsDownscalingRecommended: true,
			wantSmallerNodes:             2,
			wantSmallerMemoryPerNode:     1024 * mibMultiplier,
		},
	}

	for _, tc := range tests {
		t.Run(fmt.Sprintf("%d zone deployment, %.0fGB ram with %.2f%% disk usage, recommend zone change: %t", tc.zoneCount, tc.ram, tc.diskUsage, tc.recommendZoneChange), func(t *testing.T) {
			allocations := make([]Allocation, 0, tc.zoneCount+1)
			allocations = append(allocations, Allocation{
				NodeRole: "", // no node role
			})
			for range tc.zoneCount {
				allocations = append(allocations, Allocation{
					NodeRole:  "h",
					DiskUsed:  fmt.Sprintf("%.0f", tc.ram*1024*35.0*mibMultiplier*tc.diskUsage),
					DiskTotal: fmt.Sprintf("%.0f", tc.ram*1024*35.0*mibMultiplier),
				})
			}

			recommendations := calcDownscaleRecommendation(allocations, tierSizes, 25.0, tc.recommendZoneChange)

			require.Equal(t, tc.wantIsAlreadySmallest, recommendations[tierHot].isAlreadySmallest, "already smallest")
			require.Equal(t, tc.wantIsDownscalingRecommended, recommendations[tierHot].isDownscalingRecommended, "is downscaling recommended")
			require.Equal(t, tc.wantSmallerNodes, recommendations[tierHot].smallerNodes, "smaller node size")
			require.Equal(t, tc.wantSmallerMemoryPerNode, recommendations[tierHot].smallerMemoryPerNode, "smaller memory per node want %s, got: %s", units.BytesSize(tc.wantSmallerMemoryPerNode), units.BytesSize(recommendations[tierHot].smallerMemoryPerNode))
		})
	}
}

func Test_tierFromNodeRole(t *testing.T) {
	tests := []struct {
		name     string
		nodeRole string

		wantTier Tier
	}{
		{
			name:     "hot",
			nodeRole: "h",

			wantTier: "hot",
		},
		{
			name:     "warm",
			nodeRole: "w",

			wantTier: "warm",
		},
		{
			name:     "cold",
			nodeRole: "c",

			wantTier: "cold",
		},
		{
			name:     "frozen",
			nodeRole: "f",

			wantTier: "frozen",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			gotNodeRole := tierFromNodeRole(tc.nodeRole)
			require.Equal(t, tc.wantTier, gotNodeRole)
		})
	}
}

func Test_tierFromNodeRoleInvalid(t *testing.T) {
	tests := []struct {
		name     string
		nodeRole string

		wantTier Tier
	}{
		{
			name:     "empty",
			nodeRole: "",
		},
		{
			name:     "unknown",
			nodeRole: "x",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.Panics(t, func() {
				tierFromNodeRole(tc.nodeRole)
			})
		})
	}
}
