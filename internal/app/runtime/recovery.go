package runtimeapp

import (
	"context"
	"fmt"
	"io"
	"strconv"
	"time"

	"github.com/vrooli/vrooli/internal/cli/rootcli"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

const (
	recoveryParameterA = 2
)

func (app *App) runRuntimeRecovery(ctx *CommandContext, args []string) error {
	if len(args) == 0 || commandWantsHelp(args) {
		_, _ = io.WriteString(ctx.Stdout, "Usage:\n  vrooli runtime recovery policy set <scenario> --critical --enabled --tier <n> --retry-budget <n> [--variant <name>] [--opt-out]\n  vrooli runtime recovery policy list\n  vrooli runtime recovery inspect [--limit <n>] [--json]\n")
		return nil
	}
	if args[0] != "policy" && args[0] != "inspect" {
		return rootcli.UsageErrorf("runtime recovery", "expected policy or inspect")
	}
	home, err := ctx.HomeDir()
	if err != nil {
		return err
	}
	store, err := scenarioruntime.NewSQLiteStore(context.Background(), scenarioruntime.Config{HomeDir: home})
	if err != nil {
		return err
	}
	defer store.Close()
	if args[0] == "inspect" {
		return inspectRuntimeRecovery(ctx, store, args[1:])
	}
	if len(args) < recoveryParameterA {
		return rootcli.UsageErrorf("runtime recovery policy", "expected set or list")
	}
	switch args[1] {
	case "list":
		if len(args) != recoveryParameterA {
			return rootcli.UsageErrorf("runtime recovery policy list", "list does not accept positional arguments")
		}
		policies, err := store.ListRecoveryPolicies(context.Background(), scenarioruntime.RecoveryPolicyFilter{})
		if err != nil {
			return err
		}
		for _, policy := range policies {
			_, _ = fmt.Fprintf(ctx.Stdout, "%s@%s critical=%t enabled=%t opt_out=%t tier=%d retry_budget=%d updated_at=%s\n", policy.Scenario, policy.Variant, policy.Critical, policy.Enabled, policy.OptOut, policy.DependencyTier, policy.RetryBudget, policy.UpdatedAt.Format(time.RFC3339))
		}
		return nil
	case "set":
		return setRuntimeRecoveryPolicy(ctx, store, args[2:])
	default:
		return rootcli.UsageErrorf("runtime recovery policy", "unknown policy command: %s", args[1])
	}
}

func inspectRuntimeRecovery(ctx *CommandContext, store *scenarioruntime.SQLiteStore, args []string) error {
	limit := 50
	jsonOutput := ctx.Globals.JSON
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--json":
			jsonOutput = true
		case "--limit":
			if i+1 >= len(args) {
				return rootcli.UsageErrorf("runtime recovery inspect", "--limit requires a value")
			}
			value, err := strconv.Atoi(args[i+1])
			i++
			if err != nil || value < 1 || value > 1000 {
				return rootcli.UsageErrorf("runtime recovery inspect", "--limit must be between 1 and 1000")
			}
			limit = value
		default:
			return rootcli.UsageErrorf("runtime recovery inspect", "unknown option: %s", args[i])
		}
	}
	epochs, err := store.ListPressureEpochs(context.Background(), limit)
	if err != nil {
		return err
	}
	decisions, err := store.ListRecoveryDecisions(context.Background(), scenarioruntime.RecoveryDecisionFilter{Limit: limit})
	if err != nil {
		return err
	}
	if jsonOutput {
		return cliout.WriteJSONValue(ctx.Stdout, struct {
			Epochs    []scenarioruntime.PressureEpoch    `json:"epochs"`
			Decisions []scenarioruntime.RecoveryDecision `json:"decisions"`
		}{epochs, decisions})
	}
	for _, epoch := range epochs {
		_, _ = fmt.Fprintf(ctx.Stdout, "epoch %s status=%s source=%s detected_at=%s cleared_at=%s reason=%s\n", epoch.EpochID, epoch.Status, epoch.Source, epoch.DetectedAt.Format(time.RFC3339), formatOptionalRecoveryTime(epoch.ClearedAt), epoch.DetailsJSON)
	}
	for _, decision := range decisions {
		_, _ = fmt.Fprintf(ctx.Stdout, "decision %s epoch=%s %s@%s state=%s attempt=%d cooldown_until=%s reason=%s\n", decision.DecisionID, decision.EpochID, decision.Scenario, decision.Variant, decision.State, decision.Attempt, formatOptionalRecoveryTime(decision.CooldownUntil), decision.Reason)
	}
	return nil
}

func formatOptionalRecoveryTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}

func setRuntimeRecoveryPolicy(ctx *CommandContext, store *scenarioruntime.SQLiteStore, args []string) error {
	if len(args) == 0 {
		return rootcli.UsageErrorf("runtime recovery policy set", "scenario is required")
	}
	policy := scenarioruntime.RecoveryPolicy{Scenario: args[0]}
	for i := 1; i < len(args); i++ {
		flag := args[i]
		switch flag {
		case "--critical":
			policy.Critical = true
		case "--not-critical":
			policy.Critical = false
		case "--enabled":
			policy.Enabled = true
		case "--disabled":
			policy.Enabled = false
		case "--opt-out":
			policy.OptOut = true
		case "--clear-opt-out":
			policy.OptOut = false
		case "--variant", "--tier", "--retry-budget":
			if i+1 >= len(args) {
				return rootcli.UsageErrorf("runtime recovery policy set", "%s requires a value", flag)
			}
			value := args[i+1]
			i++
			switch flag {
			case "--variant":
				policy.Variant = value
			case "--tier":
				tier, err := strconv.Atoi(value)
				if err != nil || tier < 0 {
					return rootcli.UsageErrorf("runtime recovery policy set", "--tier must be a non-negative integer")
				}
				policy.DependencyTier = tier
			case "--retry-budget":
				budget, err := strconv.Atoi(value)
				if err != nil || budget < 0 {
					return rootcli.UsageErrorf("runtime recovery policy set", "--retry-budget must be a non-negative integer")
				}
				policy.RetryBudget = budget
			}
		default:
			return rootcli.UsageErrorf("runtime recovery policy set", "unknown option: %s", flag)
		}
	}
	updated, err := store.UpsertRecoveryPolicy(context.Background(), policy)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(ctx.Stdout, "Recovery policy saved: %s@%s critical=%t enabled=%t opt_out=%t tier=%d retry_budget=%d\n", updated.Scenario, updated.Variant, updated.Critical, updated.Enabled, updated.OptOut, updated.DependencyTier, updated.RetryBudget)
	return nil
}
