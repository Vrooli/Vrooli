package system

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/vrooli/vrooli/internal/hostpressure"
	"github.com/vrooli/vrooli/internal/resources"
	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/reporoot"
)

// resourceLifecycle is the control-plane surface used by host-pressure
// reclaim. Keeping this narrow makes the safety policy testable and ensures
// the check never grows a process signal or service-manager implementation.
type resourceLifecycle interface {
	EnabledResourceNames() ([]string, error)
	Status(name string, fast bool) (resources.Status, error)
	Run(name string, args []string, stdout, stderr io.Writer) error
}

// NewProductionHostPressureReclaimer returns the governed lifecycle callback
// used by the production check. It resolves resources through the control
// plane and recycles at most one declared, non-serving resource per action.
// An unavailable repository or home deliberately yields a refusal callback,
// not a callback that guesses ownership.
func NewProductionHostPressureReclaimer() func(context.Context) (string, error) {
	root := reporoot.ResolveFromOS()
	home, homeErr := os.UserHomeDir()
	if strings.TrimSpace(root) == "" || homeErr != nil || strings.TrimSpace(home) == "" {
		return func(context.Context) (string, error) {
			return "", fmt.Errorf("control-plane resource lifecycle is unavailable")
		}
	}
	return NewHostPressureReclaimer(resources.NewController(root, home), readHostPressureThresholds().CPUPercent)
}

// NewHostPressureReclaimer builds the policy around an injected lifecycle
// surface. Tests can therefore prove that saturation, ownership, serving, and
// one-service limits hold without starting or stopping a real resource.
func NewHostPressureReclaimer(controller resourceLifecycle, cpuThresholdPercent float64) func(context.Context) (string, error) {
	return newHostPressureReclaimer(controller, cpuThresholdPercent, func(ctx context.Context) hostpressure.PressureSnapshot {
		return hostpressure.Collect(ctx, hostpressure.Options{})
	})
}

func newHostPressureReclaimer(controller resourceLifecycle, cpuThresholdPercent float64, collect func(context.Context) hostpressure.PressureSnapshot) func(context.Context) (string, error) {
	if cpuThresholdPercent <= 0 {
		cpuThresholdPercent = fallbackHostPressureThresholds.CPUPercent
	}
	if collect == nil {
		collect = func(ctx context.Context) hostpressure.PressureSnapshot {
			return hostpressure.Collect(ctx, hostpressure.Options{})
		}
	}
	return func(ctx context.Context) (string, error) {
		if controller == nil {
			return "", fmt.Errorf("control-plane resource lifecycle is unavailable")
		}
		names, err := controller.EnabledResourceNames()
		if err != nil {
			return "", fmt.Errorf("resolve enabled resource declarations: %w", err)
		}
		allowed := make(map[string]struct{}, len(names))
		for _, name := range names {
			name = strings.TrimSpace(name)
			if name != "" {
				allowed[name] = struct{}{}
			}
		}

		snapshot := collect(ctx)
		candidates := make([]hostpressure.ReclaimCandidate, 0)
		for _, process := range snapshot.Processes {
			for name := range allowed {
				if processMatchesResource(process.Name, name) {
					candidates = append(candidates, hostpressure.ReclaimCandidate{Service: name, Process: process})
				}
			}
		}
		decision, err := hostpressure.ReclaimOne(ctx, snapshot.Processes, candidates, hostpressure.ReclaimPolicy{
			SwapToResident: 2,
			MinimumSwapped: 500 * 1024 * 1024,
			Saturated: func(context.Context) (bool, error) {
				value, ok := snapshot.CPUPressure.Number()
				return ok && value >= cpuThresholdPercent, nil
			},
			Managed: func(_ context.Context, name string) (bool, error) {
				_, ok := allowed[name]
				return ok, nil
			},
			Serving: func(ctx context.Context, name string) (bool, error) {
				status, statusErr := controller.Status(name, true)
				if statusErr != nil {
					return true, fmt.Errorf("refuse reclaim of %s because serving state is unreadable: %w", name, statusErr)
				}
				// Unknown serving state is treated as serving. Reclaim must never
				// turn an observation failure into a destructive restart.
				return status.Serving == nil || *status.Serving, nil
			},
			Recycle: func(_ context.Context, name string) error {
				return controller.Run(name, []string{"restart"}, io.Discard, io.Discard)
			},
		})
		if err != nil {
			return "", err
		}
		if decision.Selected == nil {
			return decision.HeldReason, nil
		}
		return fmt.Sprintf("recycled one idle managed resource: %s (pid %d)", decision.Selected.Service, decision.Selected.Process.PID), nil
	}
}

func processMatchesResource(processName, resourceName string) bool {
	processName = strings.ToLower(strings.TrimSpace(processName))
	resourceName = strings.ToLower(strings.TrimSpace(resourceName))
	if processName == "" || resourceName == "" {
		return false
	}
	return strings.Contains(processName, resourceName)
}
