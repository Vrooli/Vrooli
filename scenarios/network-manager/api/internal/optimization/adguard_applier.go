package optimization

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"network-manager/internal/policy"
)

const adGuardOptimizationRule = "||vrooli-optimization-check.invalid^"

type AdGuardPolicyApplier struct {
	Adapter policy.ResolverPolicyAdapter
}

var _ Applier = AdGuardPolicyApplier{}

type adGuardOptimizationRollback struct {
	Target         string   `json:"target"`
	Action         string   `json:"action"`
	Values         []string `json:"values"`
	RollbackHandle string   `json:"rollback_handle"`
}

func (a AdGuardPolicyApplier) Apply(ctx context.Context, run Run, candidate Candidate) (ApplyResult, error) {
	if a.Adapter == nil {
		return ApplyResult{Evidence: []string{"AdGuard Home policy adapter is unavailable; no optimization change was made."}}, ErrManualRequired
	}
	if !isAdGuardDNSFilteringCandidate(candidate) {
		return ApplyResult{Evidence: []string{"Candidate is not an AdGuard DNS filtering optimization; no live change was made."}}, ErrManualRequired
	}
	change := policy.Change{
		ID:     "optimization-" + strings.TrimSpace(candidate.ID),
		Target: "network",
		Action: "blocklist",
		Status: "approved",
		Values: []string{adGuardOptimizationRule},
	}
	result, err := a.Adapter.Apply(ctx, change)
	if err != nil {
		if errors.Is(err, policy.ErrUnsupported) {
			return ApplyResult{Evidence: []string{"AdGuard Home policy adapter does not support this optimization candidate."}}, ErrManualRequired
		}
		return ApplyResult{}, err
	}
	if !result.RollbackSupported || strings.TrimSpace(result.RollbackHandle) == "" {
		return ApplyResult{Evidence: append(result.Effects, "AdGuard Home optimization apply did not return rollback state; operator action is required.")}, ErrManualRequired
	}
	handle, err := json.Marshal(adGuardOptimizationRollback{
		Target:         change.Target,
		Action:         change.Action,
		Values:         change.Values,
		RollbackHandle: result.RollbackHandle,
	})
	if err != nil {
		return ApplyResult{}, err
	}
	evidence := append([]string{}, result.Effects...)
	evidence = append(evidence,
		fmt.Sprintf("Applied AdGuard Home optimization safety rule %q through the policy rollback adapter.", adGuardOptimizationRule),
		"Rollback state was captured from the AdGuard Home policy adapter.",
	)
	return ApplyResult{Evidence: evidence, RollbackHandle: string(handle)}, nil
}

func (a AdGuardPolicyApplier) Rollback(ctx context.Context, run Run, candidate Candidate) (RollbackResult, error) {
	if a.Adapter == nil {
		return RollbackResult{Evidence: []string{"AdGuard Home policy adapter is unavailable; no optimization rollback was executed."}}, ErrManualRequired
	}
	var handle adGuardOptimizationRollback
	if err := json.Unmarshal([]byte(candidate.RollbackHandle), &handle); err != nil {
		return RollbackResult{}, fmt.Errorf("decode AdGuard optimization rollback handle: %w", err)
	}
	result, err := a.Adapter.Rollback(ctx, policy.Change{
		ID:                "optimization-" + strings.TrimSpace(candidate.ID),
		Target:            handle.Target,
		Action:            handle.Action,
		Status:            "applied",
		Values:            append([]string(nil), handle.Values...),
		RollbackSupported: true,
		RollbackHandle:    handle.RollbackHandle,
	})
	if err != nil {
		return RollbackResult{}, err
	}
	evidence := append([]string{}, result.Effects...)
	evidence = append(evidence, "Rolled back AdGuard Home optimization through the policy rollback adapter.")
	return RollbackResult{Evidence: evidence}, nil
}

func isAdGuardDNSFilteringCandidate(candidate Candidate) bool {
	id := strings.ToLower(strings.TrimSpace(candidate.ID))
	if strings.Contains(id, "adguard-home") && strings.Contains(id, "dns-filtering") {
		return true
	}
	for _, item := range candidate.Evidence {
		normalized := strings.ToLower(item)
		if strings.Contains(normalized, "adguard-home") && strings.Contains(normalized, "manage_dns_filtering") {
			return true
		}
	}
	return false
}
