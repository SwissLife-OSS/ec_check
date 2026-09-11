package main

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func Test_regionProviderParts(t *testing.T) {
	tests := []struct {
		region string

		wantProvider       string
		wantProviderRegion string
		wantOK             bool
	}{
		{
			region:             "azure-westeurope",
			wantProvider:       "azure",
			wantProviderRegion: "westeurope",
			wantOK:             true,
		},
		{
			// Provider regions containing dashes must be kept intact.
			region:             "aws-eu-central-1",
			wantProvider:       "aws",
			wantProviderRegion: "eu-central-1",
			wantOK:             true,
		},
		{
			// Legacy AWS regions carry no provider prefix.
			region:             "us-east-1",
			wantProvider:       "aws",
			wantProviderRegion: "us-east-1",
			wantOK:             true,
		},
		{
			region:             "gcp-northamerica-northeast1",
			wantProvider:       "gcp",
			wantProviderRegion: "northamerica-northeast1",
			wantOK:             true,
		},
		{
			region: "not-a-region",
			wantOK: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.region, func(t *testing.T) {
			provider, providerRegion, ok := regionProviderParts(tc.region)

			require.Equal(t, tc.wantOK, ok, "region known")
			require.Equal(t, tc.wantProvider, provider, "provider")
			require.Equal(t, tc.wantProviderRegion, providerRegion, "provider region")
		})
	}
}

func Test_regionProviderPartsCoversAllListedRegions(t *testing.T) {
	for _, regions := range [][]region{awsRegions, gcpRegions, azureRegions} {
		for _, r := range regions {
			_, providerRegion, ok := regionProviderParts(r.region)

			require.True(t, ok, "region %q must be resolvable", r.region)
			require.NotEmpty(t, providerRegion, "region %q must have a provider region", r.region)
		}
	}
}
