package resources

import (
	"context"
	"fmt"
	"os/exec"

	"github.com/vrooli/vrooli/internal/accel"
	runtimehealth "github.com/vrooli/vrooli/internal/resources/runtime/health"
)

// HealthResult is the combined verdict of a resource's declared health checks
// and, for a resource that declares an accelerator, its observed placement.
type HealthResult struct {
	// Healthy is false whenever the resource is not fully meeting its contract.
	// Running below the declared accelerator backend makes it false.
	Healthy bool
	Message string
	// Serving is true whenever the resource can answer requests. A degraded
	// resource is serving; a resource whose readiness check failed is not.
	Serving bool
	// LivenessFailed names the liveness check that demoted this result, if any.
	LivenessFailed string
	// DeclaredMode is the accelerator backend the resource asked for.
	DeclaredMode string
	// ObservedMode is the backend the host says it is on. Empty means the
	// placement could not be read.
	ObservedMode string
	// ModeDrift is true when the resource is serving below its declared
	// backend.
	ModeDrift bool
	// ModeReason is the evidence behind ObservedMode.
	ModeReason string
	// PlacementUndetermined means the resource is serving but the accelerator
	// placement signal is not present yet. It must not make a health gate fail.
	PlacementUndetermined bool
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
	health := HealthResult{
		Healthy:        result.Healthy,
		Message:        result.Message,
		Serving:        result.Serving,
		LivenessFailed: result.LivenessFailed,
	}
	if err != nil {
		return health, err
	}
	return c.foldPlacementIntoHealth(ctx, manifest, health), nil
}

// foldPlacementIntoHealth makes "healthy" mean "up and on the backend it
// declared". A resource serving from the CPU while it declared CUDA is
// degraded, not healthy — which is the rule docs/resources/maturity-migration.md
// already states and nothing implemented.
//
// Placement never turns a serving resource into a stopped one, and an
// unreadable placement never demotes anything: unknown is reported as unknown.
func (c *Controller) foldPlacementIntoHealth(ctx context.Context, manifest ResourceManifest, health HealthResult) HealthResult {
	declaration := manifest.EffectiveAcceleration()
	if declaration == nil {
		return health
	}
	spec, accelerated := accelSpecFor(manifest)
	if !accelerated {
		// A CPU-only declaration still has a concrete placement contract. Keep
		// the operator-facing mode fields populated even though no accelerator
		// probe is required.
		health.DeclaredMode = string(accel.BackendCPU)
		if health.Serving {
			health.ObservedMode = string(accel.BackendCPU)
			health.ModeReason = "resource declares the cpu backend"
		}
		return health
	}
	health.DeclaredMode = string(spec.Backends[0])

	placement, err := observePlacement(ctx, c, manifest)
	if err != nil || placement == nil {
		if err != nil {
			health.ModeReason = err.Error()
			health.PlacementUndetermined = true
		}
		return health
	}
	health.DeclaredMode = string(placement.Declared)
	health.ObservedMode = string(placement.Observed)
	health.ModeReason = placement.Reason
	if placement.State == accel.BackendUndetermined {
		health.PlacementUndetermined = true
		return health
	}
	if placement.State != accel.StateDrift {
		return health
	}
	health.ModeDrift = true
	health.Healthy = false
	if health.Serving || health.Message == "" {
		health.Serving = true
		health.Message = fmt.Sprintf("degraded: declared %s but running on %s", placement.Declared, placement.Observed)
	}
	return health
}
