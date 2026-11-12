package resources

import (
	"context"
	"errors"
	"runtime"

	platform "github.com/vrooli/platform-go"
)

// PublishOpenRGBNativeUpdateArtifact publishes only the currently selected,
// already acquired exact release. The caller holds the lighting intent lock.
// Source choice is distinct from the retained serving selection; publication
// neither adopts the target nor changes the current device policy or service.
func (c *Controller) PublishOpenRGBNativeUpdateArtifact(ctx context.Context) error {
	if c == nil || ctx == nil {
		return errors.New("native artifact publication requires a controller and context")
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	if runtime.GOOS != "darwin" && runtime.GOOS != "windows" {
		return platform.ErrUnsupported
	}
	if err := c.ValidateOpenRGBNativeResourceStorage(ctx); err != nil {
		return err
	}
	if err := c.ValidateNativeOpenRGBManagementHome(ctx); err != nil {
		return err
	}
	manifest, err := c.managedServiceArtifactManifest(openRGBResourceName)
	if err != nil {
		return err
	}
	release, err := acquireManagedLifecycleLocks(ctx, manifest)
	if err != nil {
		return err
	}
	defer release()
	if err := ensureManagedUpdateLifecycleAvailable(openRGBResourceName, "native artifact publication"); err != nil {
		return err
	}
	artifact, err := c.InspectManagedServiceArtifact(ctx, openRGBResourceName)
	if err != nil {
		return err
	}
	_, err = c.PrepareOpenRGBAppImageStage(ctx, artifact)
	return err
}
