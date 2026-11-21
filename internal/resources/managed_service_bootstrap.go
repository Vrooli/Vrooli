package resources

import (
	"context"
	"fmt"

	resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"
)

// NativeManagedSharedBootstrap is the shared-provider adapter for managed
// services whose upstream process has no first-run initialization protocol.
// The supervisor and generic health contract already establish ownership and
// readiness; the adapter exists so the control-plane managed-shared policy
// does not silently fall back to a private-only provider.
func nativeManagedSharedBootstrap(_ context.Context, _ *UserResourceHost, instance ManagedInstance, appScope string) error {
	if instance.Provider != resourcedeployment.ProviderManagedShared {
		return fmt.Errorf("native managed bootstrap requires managed-shared provider")
	}
	if instance.Resource == "" || appScope == "" {
		return fmt.Errorf("native managed bootstrap requires resource and scope")
	}
	return nil
}

func init() {
	registerManagedSharedBootstrapper("minio", nativeManagedSharedBootstrap)
	registerManagedSharedBootstrapper("qdrant", nativeManagedSharedBootstrap)
}
