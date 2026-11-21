//go:build !linux

package localworkbridge

import "context"

// Process-group isolation is qualified on Linux only. Other platforms retain
// default-disabled compilation and fail closed rather than launching a child.
func (r *ProcessRunner) Run(context.Context, Request) (Result, error) {
	if !r.enabled {
		return Result{}, ErrDisabled
	}
	return Result{}, ErrProcess
}
