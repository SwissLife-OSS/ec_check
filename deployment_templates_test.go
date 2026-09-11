package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_tierSizesFromTemplate(t *testing.T) {
	template := DeploymentTemplate{
		ID: "test-template",
		InstanceConfigurations: []InstanceConfiguration{
			{
				ID:                "azure.es.datahot.edsv4",
				StorageMultiplier: 35.0,
				DiscreteSizes: DiscreteSizes{
					Sizes: []int{1024, 2048, 61440},
				},
			},
			{
				// Not a data tier, must be ignored.
				ID: "azure.kibana.edsv4",
				DiscreteSizes: DiscreteSizes{
					Sizes: []int{1024},
				},
			},
			{
				// No discrete sizes, must be ignored instead of panicking.
				ID: "azure.es.datawarm.edsv4",
			},
		},
	}

	tierSizes := tierSizesFromTemplate(template)

	require.Len(t, tierSizes, 1)
	require.NotContains(t, tierSizes, tierWarm)

	hot := tierSizes[tierHot]
	require.Len(t, hot, 3+maxNodesPerTier-1)

	// Partial nodes are sized from the discrete sizes.
	require.Equal(t, 1024.0*mibMultiplier, hot[0].Memory)
	require.Equal(t, 1024.0*35.0*mibMultiplier, hot[0].Disk)

	// Full nodes are multiples of the largest discrete size, for memory as
	// well as for disk.
	require.Equal(t, 61440.0*2*mibMultiplier, hot[3].Memory)
	require.Equal(t, 61440.0*2*35.0*mibMultiplier, hot[3].Disk)

	require.Equal(t, 61440.0*maxNodesPerTier*mibMultiplier, hot[len(hot)-1].Memory)
	require.Equal(t, 61440.0*maxNodesPerTier*35.0*mibMultiplier, hot[len(hot)-1].Disk)
}

func Test_tierSizesFromTemplateWithoutDataTiers(t *testing.T) {
	// An error response from the API unmarshals into an empty template, which
	// must not be mistaken for a valid sizing configuration.
	require.Empty(t, tierSizesFromTemplate(DeploymentTemplate{}))
}
