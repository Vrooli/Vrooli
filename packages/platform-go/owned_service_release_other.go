//go:build !linux

package platform

import "context"

func ownedServiceReleased(context.Context, ServiceRef, string) (bool, error) {
	return false, ErrUnsupported
}

func ownedServiceIntentAbsent(context.Context, ServiceIntent) (bool, error) {
	return false, ErrUnsupported
}

func ownedWorkloadReleased(context.Context, OperationRef, string) (bool, error) {
	return false, ErrUnsupported
}
func ownedWorkloadIntentAbsent(context.Context, OperationIntent) (bool, error) {
	return false, ErrUnsupported
}
