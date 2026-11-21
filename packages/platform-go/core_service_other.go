//go:build !linux

package platform

import "context"

func inspectCoreSupervisor(context.Context) (CoreServiceState, error) {
	return CoreServiceState{}, ErrUnsupported
}
func pinCoreSupervisor(context.Context, CoreServiceState) (*CoreSupervisorFence, error) {
	return nil, ErrUnsupported
}
