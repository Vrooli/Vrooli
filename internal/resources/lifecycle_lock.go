package resources

import (
	"fmt"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

// withManagedServiceLifecycleLock serializes all Vrooli-owned lifecycle
// mutations for one resource across both threads and independent control-plane
// processes. It deliberately uses a different key from artifact convergence:
// acquisition may run while a service is stopped, whereas migration,
// start/restart, stop, and uninstall must form one exclusive control-plane
// lifecycle transaction.
func withManagedServiceLifecycleLock(resource string, fn func() error) error {
	if fn == nil {
		return fmt.Errorf("managed-service lifecycle callback is required")
	}
	release, err := acquireManagedServiceLifecycleLock(resource)
	if err != nil {
		return fmt.Errorf("acquire managed-service lifecycle lock for %s: %w", resource, err)
	}
	defer release()
	return fn()
}

func acquireManagedServiceLifecycleLock(resource string) (func(), error) {
	return artifactlock.Acquire("resource-lifecycle:" + resource)
}

func managedServiceLifecycleAction(action string) bool {
	switch action {
	case "start", "restart", "stop", "uninstall":
		return true
	default:
		return false
	}
}
