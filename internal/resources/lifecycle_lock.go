package resources

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

// withManagedServiceLifecycleLock serializes all Vrooli-owned lifecycle
// mutations for one resource across both threads and independent control-plane
// processes. It deliberately uses a different key from artifact convergence:
// acquisition may run while a service is stopped, whereas migration,
// start/restart, stop, and uninstall must form one exclusive control-plane
// lifecycle transaction.
func withManagedServiceLifecycleLock(ctx context.Context, resource string, fn func() error) error {
	if fn == nil {
		return fmt.Errorf("managed-service lifecycle callback is required")
	}
	release, err := acquireManagedServiceLifecycleLock(ctx, resource)
	if err != nil {
		return fmt.Errorf("acquire managed-service lifecycle lock for %s: %w", resource, err)
	}
	defer release()
	if err := ctx.Err(); err != nil {
		return err
	}
	return fn()
}

func acquireManagedServiceLifecycleLock(ctx context.Context, resource string) (func(), error) {
	return artifactlock.AcquireContext(ctx, "resource-lifecycle:"+resource)
}

func managedServiceLifecycleAction(action string) bool {
	switch action {
	case "start", "restart", brokerTransportStop, "uninstall":
		return true
	default:
		return false
	}
}
