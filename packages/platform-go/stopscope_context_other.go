//go:build !linux

package platform

import "context"

func stopScopeContext(ctx context.Context, ref ScopeRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return stopScope(ref)
}

func scopeProcessesContext(ctx context.Context, ref ScopeRef) ([]int, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return scopeProcesses(ref)
}
