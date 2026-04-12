package resources

import (
	"context"
	"io"
	"os/exec"

	internalcontrol "github.com/vrooli/vrooli/internal/control"
	catalogpkg "github.com/vrooli/vrooli/internal/resources/catalog"
	resourcecontrol "github.com/vrooli/vrooli/internal/resources/control"
)

type Status = resourcecontrol.Status

func (c *Controller) resourceControl() *resourcecontrol.Service {
	return &resourcecontrol.Service{
		DiscoverFn: func() ([]catalogpkg.Resource, error) {
			return c.Discover()
		},
		DiscoverOneFn: func(name string) (*catalogpkg.Resource, error) {
			return c.discoverResource(name)
		},
		IsDeprecatedFn:    c.IsDeprecated,
		IsBlueprintArchFn: c.IsBlueprintArchived,
		LoadManifestFn: func(path string) (ResourceManifest, error) {
			return c.loadResourceManifest(path)
		},
		DriverStatusFn: func(ctx context.Context, item catalogpkg.Resource, manifest ResourceManifest, fast bool) (resourcecontrol.Status, error) {
			return driverStatus(ctx, c, item, manifest, fast)
		},
		DriverRunFn: func(ctx context.Context, item catalogpkg.Resource, manifest ResourceManifest, operation string, args []string, stdout, stderr io.Writer) error {
			return driverRun(ctx, c, item, manifest, operation, args, stdout, stderr)
		},
		RunLegacyFn: func(name, operation string, args []string, stdout, stderr io.Writer) error {
			return c.runLegacyResourceCommand(name, operation, args, stdout, stderr)
		},
		CommandForResourceFn: c.commandForResource,
		RunCommandFn: func(ctx context.Context, cmd *exec.Cmd) resourcecontrol.CommandResult {
			result := runCommandResource(ctx, cmd)
			return resourcecontrol.CommandResult{Output: result.output, Err: result.err}
		},
	}
}

func (c *Controller) Status(name string, fast bool) (Status, error) {
	return c.resourceControl().Status(name, fast)
}

func (c *Controller) ListStatuses(fast bool, onlyEnabled bool) ([]Status, error) {
	return c.resourceControl().ListStatuses(fast, onlyEnabled)
}

func (c *Controller) Run(name string, args []string, stdout, stderr io.Writer) error {
	return c.resourceControl().Run(name, args, stdout, stderr)
}

func (c *Controller) StartAll(stdout, stderr io.Writer) (internalcontrol.StartReport, error) {
	return c.resourceControl().StartAll(stdout, stderr)
}

func (c *Controller) StopAll(stdout, stderr io.Writer) (internalcontrol.StopReport, error) {
	return c.resourceControl().StopAll(stdout, stderr)
}
