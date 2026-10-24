package policy

import "context"

type ConservativeResolverPolicyAdapter struct{}

func (ConservativeResolverPolicyAdapter) Preview(_ context.Context, change Change) (AdapterPlan, error) {
	return AdapterPlan{
		Effects: []string{
			"Preview only; no resolver policy was changed.",
			"Live AdGuard Home filtering policy writes are not configured yet.",
			"Applying this plan will return unsupported until a resolver policy adapter confirms write support.",
		},
		RollbackSupported: false,
	}, nil
}

func (ConservativeResolverPolicyAdapter) Apply(context.Context, Change) (AdapterApplyResult, error) {
	return AdapterApplyResult{}, ErrUnsupported
}

func (ConservativeResolverPolicyAdapter) Rollback(context.Context, Change) (AdapterRollbackResult, error) {
	return AdapterRollbackResult{}, ErrUnsupported
}
