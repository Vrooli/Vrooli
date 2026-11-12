package runtimeapp

import (
	"context"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/resources"
)

// While lighting is enabled, explicit prepare may only publish qualified
// target content. Existing boot/account/device/launcher provisioning is not
// repeated. Missing or uncertain installed ownership remains a setup error.
func prepareNativeLightingUpdateArtifact(ctx context.Context, c *resources.Controller) error {
	if err := validateNativeLightingPublisher(ctx, c); err != nil {
		return err
	}
	if err := c.ValidateNativeOpenRGBManagementHome(ctx); err != nil {
		return err
	}
	image, err := platform.OpenInstalledNativeSupervisorImage(ctx)
	if err != nil {
		return err
	}
	defer image.Close()
	return c.PublishOpenRGBNativeUpdateArtifact(ctx)
}
