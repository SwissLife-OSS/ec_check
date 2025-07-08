package main

import (
	"context"
	"fmt"

	"github.com/urfave/cli/v3"
)

type region struct {
	region string
	name   string
}

// Regions taken from https://www.elastic.co/docs/reference/cloud/cloud-hosted/regions
var awsRegions = []region{
	{
		region: "aws-af-south-1",
		name:   "Africa (Cape Town)",
	},
	{
		region: "aws-ap-east-1",
		name:   "Asia Pacific (Hong Kong)",
	},
	{
		region: "ap-northeast-1",
		name:   "Asia Pacific (Tokyo)",
	},
	{
		region: "aws-ap-northeast-2",
		name:   "Asia Pacific (Seoul)",
	},
	{
		region: "aws-ap-south-1",
		name:   "Asia Pacific (Mumbai)",
	},
	{
		region: "ap-southeast-1",
		name:   "Asia Pacific (Singapore)",
	},
	{
		region: "ap-southeast-2",
		name:   "Asia Pacific (Sydney)",
	},
	{
		region: "aws-ca-central-1",
		name:   "Canada (central)",
	},
	{
		region: "aws-eu-central-1",
		name:   "EU (Frankfurt)",
	},
	{
		region: "aws-eu-central-2",
		name:   "EU (Zurich)",
	},
	{
		region: "aws-eu-north-1",
		name:   "EU (Stockholm)",
	},
	{
		region: "aws-eu-south-1",
		name:   "EU (Milan)",
	},
	{
		region: "eu-west-1",
		name:   "EU (Ireland)",
	},
	{
		region: "aws-eu-west-2",
		name:   "EU (London)",
	},
	{
		region: "aws-eu-west-3",
		name:   "EU (Paris)",
	},
	{
		region: "aws-me-south-1",
		name:   "Middle East (Bahrain)",
	},
	{
		region: "sa-east-1",
		name:   "South America (São Paulo)",
	},
	{
		region: "us-east-1",
		name:   "US East (N. Virginia)",
	},
	{
		region: "aws-us-east-2",
		name:   "US East (Ohio)",
	},
	{
		region: "us-west-1",
		name:   "US West (N. California)",
	},
	{
		region: "us-west-2",
		name:   "US West (Oregon)",
	},
}

var gcpRegions = []region{
	{
		region: "gcp-asia-east1",
		name:   "Asia Pacific East 1 (Taiwan)",
	},
	{
		region: "gcp-asia-northeast1",
		name:   "Asia Pacific Northeast 1 (Tokyo)",
	},
	{
		region: "gcp-asia-northeast3",
		name:   "Asia Pacific Northeast 3 (Seoul)",
	},
	{
		region: "gcp-asia-south1",
		name:   "Asia Pacific South 1 (Mumbai)",
	},
	{
		region: "gcp-asia-southeast1",
		name:   "Asia Pacific Southeast 1 (Singapore)",
	},
	{
		region: "gcp-asia-southeast2",
		name:   "Asia Pacific Southeast 2 (Jakarta)",
	},
	{
		region: "gcp-australia-southeast1",
		name:   "Asia Pacific Southeast 1 (Sydney)",
	},
	{
		region: "gcp-europe-north1",
		name:   "Europe North 1 (Finland)",
	},
	{
		region: "gcp-europe-west1",
		name:   "Europe West 1 (Belgium)",
	},
	{
		region: "gcp-europe-west2",
		name:   "Europe West 2 (London)",
	},
	{
		region: "gcp-europe-west3",
		name:   "Europe West 3 (Frankfurt)",
	},
	{
		region: "gcp-europe-west4",
		name:   "Europe West 4 (Netherlands)",
	},
	{
		region: "gcp-europe-west9",
		name:   "Europe West 9 (Paris)",
	},
	{
		region: "gcp-me-west1",
		name:   "ME West 1 (Tel Aviv)",
	},
	{
		region: "gcp-northamerica-northeast1",
		name:   "North America Northeast 1 (Montreal)",
	},
	{
		region: "gcp-southamerica-east1",
		name:   "South America East 1 (Sao Paulo)",
	},
	{
		region: "gcp-us-central1",
		name:   "US Central 1 (Iowa)",
	},
	{
		region: "gcp-us-east1",
		name:   "US East 1 (South Carolina)",
	},
	{
		region: "gcp-us-east4",
		name:   "US East 4 (N. Virginia)",
	},
	{
		region: "gcp-us-west1",
		name:   "US West 1 (Oregon)",
	},
}

var azureRegions = []region{
	{
		region: "azure-australiaeast",
		name:   "Australia East (New South Wales)",
	},
	{
		region: "azure-brazilsouth",
		name:   "Brazil South (São Paulo)",
	},
	{
		region: "azure-canadacentral",
		name:   "Canada Central (Toronto)",
	},
	{
		region: "azure-centralindia",
		name:   "Central India (Pune)",
	},
	{
		region: "azure-centralus",
		name:   "Central US (Iowa)",
	},
	{
		region: "azure-eastus",
		name:   "East US (Virginia)",
	},
	{
		region: "azure-eastus2",
		name:   "East US 2 (Virginia)",
	},
	{
		region: "azure-francecentral",
		name:   "France Central (Paris)",
	},
	{
		region: "azure-japaneast",
		name:   "Japan East (Tokyo, Saitama)",
	},
	{
		region: "azure-northeurope",
		name:   "North Europe (Ireland)",
	},
	{
		region: "azure-southafricanorth",
		name:   "South Africa North (Johannesburg)",
	},
	{
		region: "azure-southcentralus",
		name:   "South Central US (Texas)",
	},
	{
		region: "azure-southeastasia",
		name:   "South East Asia (Singapore)",
	},
	{
		region: "azure-uksouth",
		name:   "UK South (London)",
	},
	{
		region: "azure-westeurope",
		name:   "West Europe (Netherlands)",
	},
	{
		region: "azure-westus2",
		name:   "West US 2 (Washington)",
	},
}

var allRegions = append(awsRegions, append(gcpRegions, azureRegions...)...)

func listRegions(ctx context.Context, cmd *cli.Command) error {
	fmt.Fprintf(cmd.Writer, "Regions:\n")
	fmt.Fprintf(cmd.Writer, "  AWS:\n")
	for _, r := range awsRegions {
		fmt.Fprintf(cmd.Writer, "    %s - %s\n", r.region, r.name)
	}

	fmt.Fprintf(cmd.Writer, "  GCP:\n")
	for _, r := range gcpRegions {
		fmt.Fprintf(cmd.Writer, "    %s - %s\n", r.region, r.name)
	}

	fmt.Fprintf(cmd.Writer, "  Azure:\n")
	for _, r := range azureRegions {
		fmt.Fprintf(cmd.Writer, "    %s - %s\n", r.region, r.name)
	}

	return nil
}

func isRegionValid(region string) bool {
	for _, r := range allRegions {
		if r.name == region {
			return true
		}
	}

	return false
}
