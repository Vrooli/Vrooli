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
	if err := ensureManagedUpdateLifecycleAvailable(resource, "owner lifecycle"); err != nil {
		return err
	}
	return fn()
}

func acquireManagedServiceLifecycleLock(ctx context.Context, resource string) (func(), error) {
	if nativeOpenRGBUsesPrivateLocks(resource) {
		return acquireOpenRGBNativeOwnerLock(ctx, "resource-lifecycle:"+resource)
	}
	return artifactlock.AcquireContext(ctx, "resource-lifecycle:"+resource)
}

// Group before resource is the single lock order, including stop/restart. A
// completed start publishes supervised state before another member may inspect.
func acquireManagedLifecycleLocks(ctx context.Context, manifest ResourceManifest) (func(), error) {
	var groupRelease func()
	if group := manifest.Lifecycle.ExclusiveGroup; group != "" {
		var err error
		if nativeOpenRGBUsesPrivateLocks(manifest.Name) {
			groupRelease, err = acquireOpenRGBNativeOwnerLock(ctx, "resource-exclusive-group:"+group)
		} else {
			groupRelease, err = artifactlock.AcquireContext(ctx, "resource-exclusive-group:"+group)
		}
		if err != nil {
			return nil, err
		}
	}
	resourceRelease, err := acquireManagedServiceLifecycleLock(ctx, manifest.Name)
	if err != nil {
		if groupRelease != nil {
			groupRelease()
		}
		return nil, err
	}
	return func() {
		resourceRelease()
		if groupRelease != nil {
			groupRelease()
		}
	}, nil
}

func managedServiceLifecycleAction(action string) bool {
	switch action {
	case "start", "restart", brokerTransportStop, "uninstall":
		return true
	default:
		return false
	}
}
