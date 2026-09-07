package runtimeapp

import (
	"context"
	"fmt"
	"io"
	"time"

	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
)

type RecoveryInspectOptions struct {
	Limit int
	JSON  bool
}

// RecoveryPolicyList renders all configured recovery policies.
func (app *Service) RecoveryPolicyList(operationCtx context.Context, home string, out io.Writer) error {
	if operationCtx == nil {
		return fmt.Errorf("runtime operation context is nil")
	}
	store, err := scenarioruntime.NewSQLiteStore(operationCtx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		return err
	}
	defer store.Close()
	policies, err := store.ListRecoveryPolicies(operationCtx, scenarioruntime.RecoveryPolicyFilter{})
	if err != nil {
		return err
	}
	for _, policy := range policies {
		_, _ = fmt.Fprintf(out, "%s@%s critical=%t enabled=%t opt_out=%t tier=%d retry_budget=%d updated_at=%s\n", policy.Scenario, policy.Variant, policy.Critical, policy.Enabled, policy.OptOut, policy.DependencyTier, policy.RetryBudget, policy.UpdatedAt.Format(time.RFC3339))
	}
	return nil
}

// RecoveryPolicySet stores one typed recovery policy.
func (app *Service) RecoveryPolicySet(operationCtx context.Context, home string, out io.Writer, policy scenarioruntime.RecoveryPolicy) error {
	if operationCtx == nil {
		return fmt.Errorf("runtime operation context is nil")
	}
	store, err := scenarioruntime.NewSQLiteStore(operationCtx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		return err
	}
	defer store.Close()
	updated, err := store.UpsertRecoveryPolicy(operationCtx, policy)
	if err != nil {
		return err
	}
	_, _ = fmt.Fprintf(out, "Recovery policy saved: %s@%s critical=%t enabled=%t opt_out=%t tier=%d retry_budget=%d\n", updated.Scenario, updated.Variant, updated.Critical, updated.Enabled, updated.OptOut, updated.DependencyTier, updated.RetryBudget)
	return nil
}

// RecoveryInspect renders recent pressure epochs and recovery decisions.
func (app *Service) RecoveryInspect(operationCtx context.Context, home string, out io.Writer, opts RecoveryInspectOptions) error {
	if operationCtx == nil {
		return fmt.Errorf("runtime operation context is nil")
	}
	limit := opts.Limit
	if limit == 0 {
		limit = 50
	}
	store, err := scenarioruntime.NewSQLiteStore(operationCtx, scenarioruntime.Config{HomeDir: home})
	if err != nil {
		return err
	}
	defer store.Close()
	epochs, err := store.ListPressureEpochs(operationCtx, limit)
	if err != nil {
		return err
	}
	decisions, err := store.ListRecoveryDecisions(operationCtx, scenarioruntime.RecoveryDecisionFilter{Limit: limit})
	if err != nil {
		return err
	}
	if opts.JSON {
		return cliout.WriteJSONValue(out, struct {
			Epochs    []scenarioruntime.PressureEpoch    `json:"epochs"`
			Decisions []scenarioruntime.RecoveryDecision `json:"decisions"`
		}{epochs, decisions})
	}
	for _, epoch := range epochs {
		_, _ = fmt.Fprintf(out, "epoch %s status=%s source=%s detected_at=%s cleared_at=%s reason=%s\n", epoch.EpochID, epoch.Status, epoch.Source, epoch.DetectedAt.Format(time.RFC3339), formatOptionalRecoveryTime(epoch.ClearedAt), epoch.DetailsJSON)
	}
	for _, decision := range decisions {
		_, _ = fmt.Fprintf(out, "decision %s epoch=%s %s@%s state=%s attempt=%d cooldown_until=%s reason=%s\n", decision.DecisionID, decision.EpochID, decision.Scenario, decision.Variant, decision.State, decision.Attempt, formatOptionalRecoveryTime(decision.CooldownUntil), decision.Reason)
	}
	return nil
}

func formatOptionalRecoveryTime(value *time.Time) string {
	if value == nil {
		return ""
	}
	return value.Format(time.RFC3339)
}
