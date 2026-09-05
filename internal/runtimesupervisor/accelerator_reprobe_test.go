package runtimesupervisor

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/accel"
)

// Feature: the supervisor ends a silent CPU session without an operator
//
//	As an operator who rebooted a host
//	I want resources that started before the accelerator appeared to be noticed
//	So that a fifteen-hour silent CPU session becomes a one-interval correction.

// stubAcceleratorProbe is a scripted host: each call to ReachableBackends pops
// the next observation.
type stubAcceleratorProbe struct {
	observations [][]accel.Backend
	drifted      []accel.DriftedResource
	mode         string
	restarted    []string
	restartErr   map[string]error
	probeErr     error
	driftErr     error
}

func (p *stubAcceleratorProbe) ReachableBackends(context.Context) ([]accel.Backend, error) {
	if p.probeErr != nil {
		return nil, p.probeErr
	}
	if len(p.observations) == 0 {
		return []accel.Backend{accel.BackendCPU}, nil
	}
	next := p.observations[0]
	p.observations = p.observations[1:]
	return next, nil
}

func (p *stubAcceleratorProbe) DriftedResources(context.Context) ([]accel.DriftedResource, error) {
	if p.driftErr != nil {
		return nil, p.driftErr
	}
	return p.drifted, nil
}

func (p *stubAcceleratorProbe) ReprobeMode(context.Context) string { return p.mode }

func (p *stubAcceleratorProbe) Restart(_ context.Context, resource string) error {
	if err, ok := p.restartErr[resource]; ok {
		return err
	}
	p.restarted = append(p.restarted, resource)
	return nil
}

func serviceWithProbe(probe AcceleratorProbe) *Service {
	return &Service{cfg: Config{AcceleratorProbe: probe}}
}

// Scenario: a device appearing after boot restarts the drifted resources.
func TestReprobeRestartsDriftedResourcesWhenTheDeviceAppears(t *testing.T) {
	// Given a host that boots with no accelerator and later gains CUDA
	probe := &stubAcceleratorProbe{
		observations: [][]accel.Backend{
			{accel.BackendCPU},
			{accel.BackendCUDA, accel.BackendCPU},
		},
		drifted: []accel.DriftedResource{
			{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU},
			{Name: "reranker", Declared: accel.BackendCUDA, Observed: accel.BackendCPU},
		},
		mode: accel.ReprobeRestart,
	}
	service := serviceWithProbe(probe)

	// When the supervisor observes it twice
	first := service.reprobeAccelerators(context.Background())
	second := service.reprobeAccelerators(context.Background())

	// Then the first observation is only a baseline
	if first.Action.Transitioned {
		t.Fatalf("first observation transitioned: %+v", first)
	}
	// And the second restarts both drifted resources
	if !second.Action.Transitioned {
		t.Fatalf("second observation did not transition: %+v", second)
	}
	if !slices.Equal(probe.restarted, []string{"reranker", "whisper"}) {
		t.Fatalf("restarted = %v, want [reranker whisper]", probe.restarted)
	}
	if !slices.Equal(second.Restarted, []string{"reranker", "whisper"}) {
		t.Fatalf("report.Restarted = %v, want [reranker whisper]", second.Restarted)
	}
}

// Scenario: an actively working resource is never interrupted.
func TestReprobeDefersAnActiveResource(t *testing.T) {
	// Given a drifted resource that is currently doing work
	probe := &stubAcceleratorProbe{
		observations: [][]accel.Backend{
			{accel.BackendCPU},
			{accel.BackendCUDA, accel.BackendCPU},
		},
		drifted: []accel.DriftedResource{
			{Name: "ollama", Declared: accel.BackendCUDA, Observed: accel.BackendCPU, Active: true},
		},
		mode: accel.ReprobeRestart,
	}
	service := serviceWithProbe(probe)

	// When CUDA becomes reachable
	service.reprobeAccelerators(context.Background())
	report := service.reprobeAccelerators(context.Background())

	// Then it is deferred rather than restarted mid-request
	if len(probe.restarted) != 0 {
		t.Fatalf("restarted = %v, want none; an active resource must not be interrupted", probe.restarted)
	}
	if !slices.Contains(report.Action.Deferred, "ollama") {
		t.Fatalf("Deferred = %v, want ollama", report.Action.Deferred)
	}
}

// Scenario: a failing probe does not manufacture a transition.
//
// Treating "could not read the host" as "the accelerator went away" would
// produce a spurious transition on the next successful read, restarting the
// fleet for no reason.
func TestReprobeSkipsTheObservationWhenTheProbeFails(t *testing.T) {
	// Given a host whose accelerator probe fails
	probe := &stubAcceleratorProbe{probeErr: errors.New("collector timed out"), mode: accel.ReprobeRestart}
	service := serviceWithProbe(probe)

	// When the supervisor ticks
	report := service.reprobeAccelerators(context.Background())

	// Then the failure is recorded and no transition is claimed
	if report.Action.Transitioned {
		t.Fatalf("Transitioned = true on a failed probe: %+v", report)
	}
	if !strings.Contains(report.Error, "collector timed out") {
		t.Fatalf("Error = %q, want the underlying probe failure", report.Error)
	}

	// And a later successful read is treated as the baseline, not a transition
	probe.probeErr = nil
	probe.observations = [][]accel.Backend{{accel.BackendCUDA, accel.BackendCPU}}
	probe.drifted = []accel.DriftedResource{{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU}}
	after := service.reprobeAccelerators(context.Background())
	if after.Action.Transitioned {
		t.Fatalf("a failed probe followed by a successful one manufactured a transition: %+v", after)
	}
}

// Scenario: a restart that fails is recorded, and the others still run.
func TestReprobeRecordsARestartFailureWithoutStopping(t *testing.T) {
	// Given two drifted resources, one of which cannot be restarted
	probe := &stubAcceleratorProbe{
		observations: [][]accel.Backend{
			{accel.BackendCPU},
			{accel.BackendCUDA, accel.BackendCPU},
		},
		drifted: []accel.DriftedResource{
			{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU},
			{Name: "reranker", Declared: accel.BackendCUDA, Observed: accel.BackendCPU},
		},
		mode:       accel.ReprobeRestart,
		restartErr: map[string]error{"reranker": errors.New("artifact checksum mismatch")},
	}
	service := serviceWithProbe(probe)

	// When CUDA becomes reachable
	service.reprobeAccelerators(context.Background())
	report := service.reprobeAccelerators(context.Background())

	// Then the working restart still happened
	if !slices.Equal(report.Restarted, []string{"whisper"}) {
		t.Fatalf("Restarted = %v, want [whisper]", report.Restarted)
	}
	// And the failure is named rather than swallowed
	if report.Failures["reranker"] == "" {
		t.Fatalf("Failures = %v, want the reranker failure recorded", report.Failures)
	}
}

// Scenario: no probe configured means the pass does nothing at all.
func TestReprobeIsInertWithoutAProbe(t *testing.T) {
	// Given a supervisor with no accelerator probe
	service := serviceWithProbe(nil)

	// When it ticks
	report := service.reprobeAccelerators(context.Background())

	// Then nothing is claimed and nothing fails
	if report.Action.Transitioned || report.Error != "" || len(report.Restarted) != 0 {
		t.Fatalf("report = %+v, want a zero report", report)
	}
}
