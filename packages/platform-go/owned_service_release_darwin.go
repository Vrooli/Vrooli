//go:build darwin

package platform

import "context"

func ownedServiceReleased(ctx context.Context, r ServiceRef, boot string) (bool, error) {
	return nativeDarwinOwnedBackend().released(ctx, OperationRef(r), boot)
}
func ownedServiceIntentAbsent(ctx context.Context, i ServiceIntent) (bool, error) {
	return nativeDarwinOwnedBackend().absent(ctx, OperationIntent(i))
}
func ownedWorkloadReleased(ctx context.Context, r OperationRef, boot string) (bool, error) {
	return nativeDarwinOwnedBackend().released(ctx, r, boot)
}
func ownedWorkloadIntentAbsent(ctx context.Context, i OperationIntent) (bool, error) {
	return nativeDarwinOwnedBackend().absent(ctx, i)
}
