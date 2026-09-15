//go:build !linux

package resources

import resourcedeployment "github.com/vrooli/vrooli/packages/resource-deployment"

func applyManagedServiceProcessLimits(_ int, _ *resourcedeployment.ProcessLimits) error {
	return nil
}
