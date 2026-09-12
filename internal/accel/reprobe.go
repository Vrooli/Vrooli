package accel

import (
	"fmt"
	"slices"
	"strings"
)

// ReprobeModes are what to do when host accelerator readiness transitions from
// not-ready to ready. They are duplicated from internal/capacity's policy
// vocabulary rather than imported, because accel must not depend on the claim
// ledger to decide what a transition means.
const (
	ReprobeOff     = "off"
	ReprobeReport  = "report"
	ReprobeRestart = "restart"
)

// DriftedResource is one resource the watcher may act on: it declared a backend
// it is not on, and the host has just become able to give it one.
type DriftedResource struct {
	// Name is the resource slug.
	Name string `json:"name"`
	// Declared is the backend it asked for.
	Declared Backend `json:"declared"`
	// Observed is the backend it is on.
	Observed Backend `json:"observed"`
	// Active is true when the resource is currently doing work. An active
	// resource is never restarted; the next transition retries it.
	Active bool `json:"active"`
}

// ReprobeAction is what the watcher decided for one transition.
type ReprobeAction struct {
	// Mode is the policy that produced this decision.
	Mode string `json:"mode"`
	// Transitioned is true when readiness actually moved from not-ready to
	// ready. Every other observation is a no-op.
	Transitioned bool `json:"transitioned"`
	// Backends are the backends that became reachable.
	Backends []Backend `json:"backends,omitempty"`
	// Restart names the resources to restart, in stable order.
	Restart []string `json:"restart,omitempty"`
	// Deferred names drifted resources left alone because they are working.
	Deferred []string `json:"deferred,omitempty"`
	// Reason is one line an operator can read.
	Reason string `json:"reason"`
}

// ReadinessWatcher tracks host accelerator readiness across observations and
// decides what to do when it changes.
//
// It holds only the previous observation, so it is safe to keep for the life of
// a supervisor. It performs no I/O: the caller supplies the snapshot and the
// drifted-resource list, which keeps this decidable in a unit test on a host
// with no accelerator.
type ReadinessWatcher struct {
	// observed is false until the first Observe call, so a supervisor that
	// starts on an already-ready host does not report a transition that did not
	// happen.
	observed bool
	previous []Backend
}

// Observe records one readiness observation and returns what to do about it.
//
// A transition is a backend becoming reachable that was not reachable before.
// The CPU backend is always reachable and is never a transition.
func (w *ReadinessWatcher) Observe(mode string, reachable []Backend, drifted []DriftedResource) ReprobeAction {
	action := ReprobeAction{Mode: normaliseReprobeMode(mode)}

	accelerated := make([]Backend, 0, len(reachable))
	for _, backend := range reachable {
		if backend != BackendCPU {
			accelerated = append(accelerated, backend)
		}
	}

	first := !w.observed
	gained := make([]Backend, 0, len(accelerated))
	for _, backend := range accelerated {
		if !slices.Contains(w.previous, backend) {
			gained = append(gained, backend)
		}
	}
	w.observed = true
	w.previous = accelerated

	if first {
		action.Reason = "first observation; nothing to compare against"
		return action
	}
	if len(gained) == 0 {
		action.Reason = "no accelerator backend became reachable since the last observation"
		return action
	}

	action.Transitioned = true
	action.Backends = gained
	names := make([]string, 0, len(gained))
	for _, backend := range gained {
		names = append(names, string(backend))
	}

	if action.Mode == ReprobeOff {
		action.Reason = fmt.Sprintf("%s became reachable; accel_reprobe is off, so nothing was done", strings.Join(names, ", "))
		return action
	}
	if len(drifted) == 0 {
		action.Reason = fmt.Sprintf("%s became reachable; no resource is running below its declared backend", strings.Join(names, ", "))
		return action
	}

	for _, resource := range drifted {
		if resource.Active {
			action.Deferred = append(action.Deferred, resource.Name)
			continue
		}
		action.Restart = append(action.Restart, resource.Name)
	}
	slices.Sort(action.Restart)
	slices.Sort(action.Deferred)

	if action.Mode == ReprobeReport {
		// Report mode records the list and restarts nothing. Restarting is a
		// lifecycle action; doing one by default would surprise an operator
		// whose resource is mid-request.
		action.Reason = fmt.Sprintf("%s became reachable; %d resource(s) are running below their declared backend: %s", strings.Join(names, ", "), len(drifted), strings.Join(driftedNames(drifted), ", "))
		action.Restart = nil
		action.Deferred = driftedNames(drifted)
		return action
	}

	action.Reason = fmt.Sprintf("%s became reachable; restarting %d drifted resource(s), deferring %d that are actively working", strings.Join(names, ", "), len(action.Restart), len(action.Deferred))
	return action
}

func driftedNames(drifted []DriftedResource) []string {
	names := make([]string, 0, len(drifted))
	for _, resource := range drifted {
		names = append(names, resource.Name)
	}
	slices.Sort(names)
	return names
}

func normaliseReprobeMode(mode string) string {
	normalised := strings.ToLower(strings.TrimSpace(mode))
	switch normalised {
	case ReprobeOff, ReprobeReport, ReprobeRestart:
		return normalised
	}
	return ReprobeReport
}
