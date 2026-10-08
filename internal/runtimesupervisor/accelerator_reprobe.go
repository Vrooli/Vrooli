package runtimesupervisor

import (
	"context"
	"fmt"

	"github.com/vrooli/vrooli/internal/accel"
)

// AcceleratorProbe reads host accelerator readiness and the resources currently
// running below their declared backend. It is a seam so the supervisor's
// re-probe logic is testable on a host with no accelerator and no resources.
type AcceleratorProbe interface {
	// ReachableBackends is what the host can reach right now.
	ReachableBackends(ctx context.Context) ([]accel.Backend, error)
	// DriftedResources are the resources whose observed backend is below the
	// backend they declared, with their current activity state.
	DriftedResources(ctx context.Context) ([]accel.DriftedResource, error)
	// ReprobeMode is the operator's accel_reprobe policy value.
	ReprobeMode(ctx context.Context) string
	// Restart restarts one resource. It is only called in restart mode, for a
	// resource the probe reported as not active.
	Restart(ctx context.Context, resource string) error
}

// AcceleratorReprobeReport is what one supervisor tick decided about
// accelerator readiness.
type AcceleratorReprobeReport struct {
	Action    accel.ReprobeAction `json:"action"`
	Restarted []string            `json:"restarted,omitempty"`
	Failures  map[string]string   `json:"failures,omitempty"`
	Error     string              `json:"error,omitempty"`
}

// reprobeAccelerators is the supervisor's answer to a device that appears after
// the resources that need it have already started. It runs at the supervisor's
// existing interval and does nothing at all until readiness actually changes.
//
// It never escalates privilege and never restarts a resource that is working.
func (s *Service) reprobeAccelerators(ctx context.Context) AcceleratorReprobeReport {
	report := AcceleratorReprobeReport{}
	if s.cfg.AcceleratorProbe == nil {
		return report
	}

	reachable, err := s.cfg.AcceleratorProbe.ReachableBackends(ctx)
	if err != nil {
		// A probe that cannot run is not evidence that the accelerator went
		// away. Skip the observation entirely rather than recording a loss that
		// would produce a spurious transition on the next tick.
		report.Error = fmt.Sprintf("read accelerator readiness: %v", err)
		return report
	}

	mode := s.cfg.AcceleratorProbe.ReprobeMode(ctx)
	var drifted []accel.DriftedResource
	if mode != accel.ReprobeOff {
		drifted, err = s.cfg.AcceleratorProbe.DriftedResources(ctx)
		if err != nil {
			report.Error = fmt.Sprintf("list drifted resources: %v", err)
			// Still observe readiness, so the baseline advances and a later
			// tick with a working listing sees the correct previous state.
			report.Action = s.acceleratorWatcher.Observe(mode, reachable, nil)
			return report
		}
	}

	report.Action = s.acceleratorWatcher.Observe(mode, reachable, drifted)
	if !report.Action.Transitioned {
		return report
	}
	s.logf("accelerator readiness changed: %s", report.Action.Reason)
	if len(report.Action.Restart) == 0 {
		return report
	}

	for _, resource := range report.Action.Restart {
		if err := s.cfg.AcceleratorProbe.Restart(ctx, resource); err != nil {
			if report.Failures == nil {
				report.Failures = map[string]string{}
			}
			report.Failures[resource] = err.Error()
			s.logf("accelerator re-probe restart failed resource=%s err=%v", resource, err)
			continue
		}
		report.Restarted = append(report.Restarted, resource)
		s.logf("accelerator re-probe restarted resource=%s", resource)
	}
	return report
}
