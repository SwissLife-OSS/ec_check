package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sort"

	"github.com/urfave/cli/v3"
)

type DeploymentTemplates []DeploymentTemplate

func listProfiles(ctx context.Context, cmd *cli.Command) error {
	region := cmd.String("region")
	if _, _, ok := regionProviderParts(region); !ok {
		return fmt.Errorf("region %q is not a known Elastic Cloud region, use %q to list the known regions", region, "ec_check regions")
	}

	deploymentTemplatesURL := fmt.Sprintf("https://api.elastic-cloud.com/api/v1/deployments/templates?region=%s", region)

	resp, err := http.Get(deploymentTemplatesURL)
	if err != nil {
		return err
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var deploymentTemplates DeploymentTemplates
	err = json.Unmarshal(body, &deploymentTemplates)
	if err != nil {
		return err
	}

	sort.Slice(deploymentTemplates, func(i, j int) bool {
		return deploymentTemplates[i].ID < deploymentTemplates[j].ID
	})

	fmt.Fprintf(cmd.Writer, "Profiles for %q:\n", region)
	for _, dt := range deploymentTemplates {
		fmt.Fprintf(cmd.Writer, "%s\n", dt.ID)
	}

	return nil
}
