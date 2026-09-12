package optimization

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	"network-manager/internal/policy"
)

func TestAdGuardPolicyApplierApplyAndRollback(t *testing.T) {
	// [REQ:NM-P0-005] AdGuard-backed optimization applies only through rollback-capable policy adapter state.
	adapter := &fakePolicyAdapter{
		apply: policy.AdapterApplyResult{
			Effects:           []string{"policy adapter applied test rule"},
			RollbackSupported: true,
			RollbackHandle:    `{"kind":"filtering_rules","user_rules":[]}`,
		},
		rollback: policy.AdapterRollbackResult{Effects: []string{"policy adapter restored rules"}},
	}
	applier := AdGuardPolicyApplier{Adapter: adapter}
	candidate := Candidate{
		ID:                "adguard-home-dns-filtering-stability",
		RollbackSupported: true,
		Evidence:          []string{"adguard-home supports manage_dns_filtering: test capability"},
	}

	applied, err := applier.Apply(context.Background(), Run{ID: "run-1"}, candidate)
	require.NoError(t, err)
	require.NotEmpty(t, applied.RollbackHandle)
	require.Contains(t, applied.Evidence, "policy adapter applied test rule")
	require.Contains(t, applied.Evidence, "Rollback state was captured from the AdGuard Home policy adapter.")
	require.Equal(t, "network", adapter.applied.Target)
	require.Equal(t, "blocklist", adapter.applied.Action)
	require.Equal(t, []string{adGuardOptimizationRule}, adapter.applied.Values)

	candidate.RollbackHandle = applied.RollbackHandle
	rolledBack, err := applier.Rollback(context.Background(), Run{ID: "run-1"}, candidate)
	require.NoError(t, err)
	require.Contains(t, rolledBack.Evidence, "policy adapter restored rules")
	require.Equal(t, `{"kind":"filtering_rules","user_rules":[]}`, adapter.rolledBack.RollbackHandle)
}

func TestAdGuardPolicyApplierRejectsNonAdGuardCandidate(t *testing.T) {
	// [REQ:NM-P0-005] Production optimization does not apply unsupported candidates through the AdGuard adapter.
	applier := AdGuardPolicyApplier{Adapter: &fakePolicyAdapter{}}

	result, err := applier.Apply(context.Background(), Run{}, Candidate{ID: "host-linux-read-only-baseline-compare"})
	require.ErrorIs(t, err, ErrManualRequired)
	require.Contains(t, result.Evidence, "Candidate is not an AdGuard DNS filtering optimization; no live change was made.")
}

func TestAdGuardPolicyApplierRequiresRollbackHandle(t *testing.T) {
	// [REQ:NM-P0-005] AdGuard optimization apply fails closed when adapter rollback state is unavailable.
	applier := AdGuardPolicyApplier{Adapter: &fakePolicyAdapter{apply: policy.AdapterApplyResult{Effects: []string{"applied without rollback"}}}}
	candidate := Candidate{ID: "adguard-home-dns-filtering-stability", Evidence: []string{"adguard-home supports manage_dns_filtering"}}

	result, err := applier.Apply(context.Background(), Run{}, candidate)
	require.ErrorIs(t, err, ErrManualRequired)
	require.Contains(t, result.Evidence, "AdGuard Home optimization apply did not return rollback state; operator action is required.")
}

func TestAdGuardPolicyApplierMapsUnsupportedToManualRequired(t *testing.T) {
	// [REQ:NM-P0-005] Unsupported AdGuard policy writes remain manual_required instead of being reported as applied.
	applier := AdGuardPolicyApplier{Adapter: &fakePolicyAdapter{applyErr: policy.ErrUnsupported}}
	candidate := Candidate{ID: "adguard-home-dns-filtering-stability", Evidence: []string{"adguard-home supports manage_dns_filtering"}}

	result, err := applier.Apply(context.Background(), Run{}, candidate)
	require.ErrorIs(t, err, ErrManualRequired)
	require.Contains(t, result.Evidence, "AdGuard Home policy adapter does not support this optimization candidate.")
}

type fakePolicyAdapter struct {
	apply       policy.AdapterApplyResult
	applyErr    error
	rollback    policy.AdapterRollbackResult
	rollbackErr error
	applied     policy.Change
	rolledBack  policy.Change
}

func (f *fakePolicyAdapter) Preview(context.Context, policy.Change) (policy.AdapterPlan, error) {
	return policy.AdapterPlan{}, nil
}

func (f *fakePolicyAdapter) Apply(_ context.Context, change policy.Change) (policy.AdapterApplyResult, error) {
	f.applied = change
	if f.applyErr != nil {
		return policy.AdapterApplyResult{}, f.applyErr
	}
	return f.apply, nil
}

func (f *fakePolicyAdapter) Rollback(_ context.Context, change policy.Change) (policy.AdapterRollbackResult, error) {
	f.rolledBack = change
	if f.rollbackErr != nil {
		return policy.AdapterRollbackResult{}, f.rollbackErr
	}
	if len(f.rollback.Effects) == 0 && f.rollbackErr == nil {
		return policy.AdapterRollbackResult{}, errors.New("rollback result not configured")
	}
	return f.rollback, nil
}

var _ policy.ResolverPolicyAdapter = (*fakePolicyAdapter)(nil)
