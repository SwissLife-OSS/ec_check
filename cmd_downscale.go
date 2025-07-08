package main

import (
	"context"
	"fmt"
	"strings"

	"github.com/urfave/cli/v3"
)

func downscale(ctx context.Context, cmd *cli.Command) error {
	deployment := cmd.String("deployment")
	region := cmd.String("region")
	profile := cmd.String("profile")
	username := cmd.String("username")
	password := cmd.String("password")
	headroomPercent := cmd.Float64("headroom-pct")
	recommendZoneChange := cmd.Bool("recommend-zone-change")
	exitCode := cmd.Bool("exit-code")

	if !isRegionValid(region) {
		return fmt.Errorf("region %q is not a known Elastic Cloud region", region)
	}

	regionParts := strings.Split(region, "-")
	if len(regionParts) != 2 {
		return fmt.Errorf(`invalid region, expected format "<provider>-<region>", e.g. "azure-westeurope"`)
	}

	provider := regionParts[0]
	providerRegion := regionParts[1]

	tierDiskSizes, err := getTierSizes(region, profile)
	if err != nil {
		return err
	}

	verbosef(cmd, "%s", tierDiskSizes)

	usernamePassword := ""
	if username != "" && password != "" {
		usernamePassword = fmt.Sprintf("%s:%s@", username, password)
	}

	deploymentURL := fmt.Sprintf("https://%s%s.es.%s.%s.elastic-cloud.com", usernamePassword, deployment, providerRegion, provider)

	allocations, err := getAllocationInformation(deploymentURL)
	if err != nil {
		return err
	}

	recommendations := calcDownscaleRecommendation(allocations, tierDiskSizes, headroomPercent, recommendZoneChange)

	fmt.Fprintf(cmd.Writer, "%s", recommendations)

	if exitCode && recommendations.IsDownscalingRecommended() {
		return cli.Exit("Downscaling for at least one tier is recommended", 2)
	}

	return nil
}

func verbosef(cmd *cli.Command, format string, args ...any) {
	if cmd.Bool("verbose") {
		fmt.Fprintf(cmd.Writer, format, args...)
	}
}
