package resources

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/artifactlock"
)

// uninstallManagedServiceArtifact removes only the resource's managed runtime
// artifact store after the caller has stopped the service under its lifecycle
// lock. Operator data and independently installed host tools are outside this
// store and are deliberately preserved.
func uninstallManagedServiceArtifact(ctx context.Context, controller *Controller, manifest ResourceManifest) error {
	if manifest.ManagedService == nil || manifest.ManagedService.Acquisition == nil {
		return nil
	}
	if strings.EqualFold(strings.TrimSpace(manifest.ManagedService.Artifact.Verification), "host-tool") {
		return nil
	}
	name := strings.TrimSpace(manifest.Name)
	if name == "" || filepath.Base(name) != name || name == "." || name == ".." {
		return fmt.Errorf("refuse to uninstall unsafe resource artifact name %q", manifest.Name)
	}
	storeRoot, err := managedServiceArtifactStoreRoot(controller.Home)
	if err != nil {
		return err
	}
	resourceRoot := filepath.Join(storeRoot, name)
	relative, err := filepath.Rel(storeRoot, resourceRoot)
	if err != nil || relative != name || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refuse to uninstall artifact path outside managed store")
	}

	release, err := artifactlock.AcquireContext(ctx, "resource:"+name)
	if err != nil {
		return err
	}
	defer release()

	storeInfo, err := os.Lstat(storeRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect managed artifact store: %w", err)
	}
	if storeInfo.Mode()&os.ModeSymlink != 0 || !storeInfo.IsDir() {
		return fmt.Errorf("refuse to uninstall through non-directory or symlink artifact store %s", storeRoot)
	}
	resourceInfo, err := os.Lstat(resourceRoot)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("inspect resource artifact store: %w", err)
	}
	if resourceInfo.Mode()&os.ModeSymlink != 0 || !resourceInfo.IsDir() {
		return fmt.Errorf("refuse to uninstall non-directory or symlink resource artifact store %s", resourceRoot)
	}

	if state, running, err := func() (ManagedServiceState, bool, error) {
		supervisor, _, err := managedServiceSupervisorFor(name)
		if err != nil {
			return ManagedServiceState{}, false, err
		}
		return supervisor.Status()
	}(); err != nil {
		return fmt.Errorf("verify %s is stopped before uninstall: %w", name, err)
	} else if running {
		return fmt.Errorf("refuse to uninstall %s artifact version %s while its managed service is running", name, state.ArtifactVersion)
	}

	for version, inUse := range artifactVersionsInUse(resourceRoot) {
		if inUse {
			return fmt.Errorf("refuse to uninstall %s artifact version %s while a process still uses it", name, version)
		}
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := os.RemoveAll(resourceRoot); err != nil {
		return fmt.Errorf("remove managed %s artifacts: %w", name, err)
	}
	return nil
}
