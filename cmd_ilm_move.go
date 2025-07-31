package main

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/urfave/cli/v3"
)

func ilmMove(ctx context.Context, cmd *cli.Command) error {
	deployment := cmd.String("deployment")
	region := cmd.String("region")
	username := cmd.String("username")
	password := cmd.String("password")

	dryRun := cmd.Bool("dry-run")
	force := cmd.Bool("force")
	indexPattern := cmd.String("index-pattern")
	targetPhase := cmd.String("target-phase")

	if !isRegionValid(region) {
		return fmt.Errorf("region %q is not a known Elastic Cloud region", region)
	}

	var dryRunPrefix string
	if dryRun {
		dryRunPrefix = "(DRY RUN) "
	}

	phases := []string{"hot", "warm", "cold", "frozen", "delete"}
	if !slices.Contains(phases, targetPhase) {
		return fmt.Errorf("target-phase %q is invalid, valid values are: %v", targetPhase, phases)
	}

	regionParts := strings.Split(region, "-")
	if len(regionParts) != 2 {
		return fmt.Errorf(`invalid region, expected format "<provider>-<region>", e.g. "azure-westeurope"`)
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

	ilms, err := client.Ilm.ExplainLifecycle(indexPattern).OnlyManaged(true).Do(ctx)
	if err != nil {
		return err
	}

	policies, err := client.Ilm.GetLifecycle().Do(ctx)
	if err != nil {
		return err
	}

	for _, ilm := range ilms.Indices {
		managed, ok := ilm.(*types.LifecycleExplainManaged)
		if !ok {
			continue
		}

		if *managed.Phase == targetPhase {
			fmt.Printf("index %q is already in phase %q, skipping\n", managed.Index, *managed.Phase)
			continue
		}

		if !force && (*managed.Action != "complete" || *managed.Step != "complete") {
			fmt.Printf(`index %q is not in "complete" state (action: %q, step: %q) in its phase and --force is not given, skipping`+"\n", managed.Index, *managed.Action, *managed.Step)
			continue
		}

		policy, ok := policies[*managed.Policy]
		if !ok {
			return fmt.Errorf("policy %q not found", *managed.Policy)
		}

		var policyPhaseDefinition *types.Phase
		switch targetPhase {
		case "hot":
			policyPhaseDefinition = policy.Policy.Phases.Hot
		case "warm":
			policyPhaseDefinition = policy.Policy.Phases.Warm
		case "cold":
			policyPhaseDefinition = policy.Policy.Phases.Cold
		case "frozen":
			policyPhaseDefinition = policy.Policy.Phases.Frozen
		case "delete":
			policyPhaseDefinition = policy.Policy.Phases.Delete
		}
		if policyPhaseDefinition == nil {
			fmt.Printf("target phase %q is not defined in policy %q used by index %q\n", targetPhase, *managed.Phase, managed.Index)
			continue
		}

		fmt.Printf("%s move %q (phase: %q, action: %q, step: %q, policy: %q) to phase %q\n", dryRunPrefix, managed.Index, *managed.Phase, *managed.Action, *managed.Step, *managed.Policy, targetPhase)
		if dryRun {
			continue
		}

		resp, err := client.Ilm.MoveToStep(managed.Index).CurrentStep(&types.StepKey{
			Phase:  *managed.Phase,
			Action: managed.Action,
			Name:   managed.Step,
		}).NextStep(&types.StepKey{
			Phase: targetPhase,
		}).Do(ctx)
		if err != nil {
			return err
		}

		if !resp.Acknowledged {
			return fmt.Errorf("move operation for %q has not been acknowledged", managed.Index)
		}
	}

	return nil
}
