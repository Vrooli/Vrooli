package resources

import (
	"context"
	"io"
)

func driverStatus(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, fast bool) (Status, error) {
	driver, err := driverForManifest(manifest)
	if err != nil {
		return Status{}, err
	}
	return driver.Status(ctx, controller, item, manifest, fast)
}

func driverRun(ctx context.Context, controller *Controller, item Resource, manifest ResourceManifest, operation string, args []string, stdout, stderr io.Writer) error {
	driver, err := driverForManifest(manifest)
	if err != nil {
		return err
	}
	return driver.Run(ctx, controller, item, manifest, operation, args, stdout, stderr)
}
