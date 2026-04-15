package resources

import (
	"context"
	"os/exec"

	runtimehealth "github.com/vrooli/vrooli/internal/resources/runtime/health"
)

type HealthResult struct {
	Healthy bool
	Message string
}

func (c *Controller) runResourceHealthChecks(ctx context.Context, manifest ResourceManifest) (HealthResult, error) {
	result, err := runtimehealth.RunChecks(ctx, manifest.HealthChecks, runtimehealth.Config{
		Root: c.Root,
		Env:  resourceEnv(c.Root, c.Home),
		Runner: func(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
			result := runCommandResource(ctx, cmd)
			return result.output, result.err
		},
	})
	return HealthResult(result), err
}

func (c *Controller) runResourceHealthCheck(ctx context.Context, check ResourceHealthCheck) (HealthResult, error) {
	result, err := runtimehealth.RunCheck(ctx, check, runtimehealth.Config{
		Root: c.Root,
		Env:  resourceEnv(c.Root, c.Home),
		Runner: func(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
			result := runCommandResource(ctx, cmd)
			return result.output, result.err
		},
	})
	return HealthResult(result), err
}
