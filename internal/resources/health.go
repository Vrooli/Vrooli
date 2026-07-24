package resources

import (
	"context"
	"fmt"
	"os/exec"

	runtimehealth "github.com/vrooli/vrooli/internal/resources/runtime/health"
)

type HealthResult struct {
	Healthy bool
	Message string
}

func (c *Controller) runResourceHealthChecks(ctx context.Context, manifest ResourceManifest) (HealthResult, error) {
	env := resourceEnvForResource(c.Root, c.Home, manifest.Name)
	for _, port := range manifest.Ports {
		if port.Host > 0 {
			env = setEnvValue(env, managedServicePortEnvName(port.Name), fmt.Sprintf("%d", port.Host))
		}
	}
	result, err := runtimehealth.RunChecks(ctx, manifest.HealthChecks, runtimehealth.Config{
		Root: c.Root,
		Env:  env,
		Runner: func(ctx context.Context, cmd *exec.Cmd) ([]byte, error) {
			result := runCommandResource(ctx, cmd)
			return result.output, result.err
		},
	})
	return HealthResult(result), err
}
