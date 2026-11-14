package accel_test

import (
	"slices"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/accel"
)

// Feature: a late-appearing device does not leave the fleet on the CPU
//
//	As the runtime supervisor
//	I want to notice when an accelerator backend becomes reachable
//	So that resources which started before the device existed stop running on
//	the CPU with every status surface reporting green.

// Scenario: the first observation is never a transition.
//
// A supervisor that starts on an already-ready host must not report that the
// accelerator "just appeared" and restart the fleet.
func TestReadinessWatcherTreatsTheFirstObservationAsABaseline(t *testing.T) {
	// Given a fresh watcher
	var watcher accel.ReadinessWatcher
	drifted := []accel.DriftedResource{{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU}}

	// When it first observes a host that can already reach CUDA
	action := watcher.Observe(accel.ReprobeRestart, []accel.Backend{accel.BackendCUDA, accel.BackendCPU}, drifted)

	// Then nothing transitioned and nothing is restarted
	if action.Transitioned {
		t.Fatalf("Transitioned = true on the first observation: %+v", action)
	}
	if len(action.Restart) != 0 {
		t.Fatalf("Restart = %v, want none", action.Restart)
	}
	if !strings.Contains(action.Reason, "first observation") {
		t.Fatalf("Reason = %q, want it to say this was the baseline", action.Reason)
	}
}

// Scenario: a backend becoming reachable restarts the drifted resources.
//
// This is the fifteen-hour silent CPU session, ended within one supervisor
// interval.
func TestReadinessWatcherRestartsDriftedResourcesOnTransition(t *testing.T) {
	// Given a watcher that has seen a host with no accelerator
	var watcher accel.ReadinessWatcher
	watcher.Observe(accel.ReprobeRestart, []accel.Backend{accel.BackendCPU}, nil)

	// And three resources running below their declared backend, one of them busy
	drifted := []accel.DriftedResource{
		{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU},
		{Name: "ollama", Declared: accel.BackendCUDA, Observed: accel.BackendCPU, Active: true},
		{Name: "reranker", Declared: accel.BackendCUDA, Observed: accel.BackendCPU},
	}

	// When CUDA becomes reachable
	action := watcher.Observe(accel.ReprobeRestart, []accel.Backend{accel.BackendCUDA, accel.BackendCPU}, drifted)

	// Then the transition is recognised and names the backend that appeared
	if !action.Transitioned {
		t.Fatalf("Transitioned = false: %+v", action)
	}
	if !slices.Equal(action.Backends, []accel.Backend{accel.BackendCUDA}) {
		t.Fatalf("Backends = %v, want [cuda]", action.Backends)
	}
	// And the idle drifted resources are restarted, in stable order
	if !slices.Equal(action.Restart, []string{"reranker", "whisper"}) {
		t.Fatalf("Restart = %v, want [reranker whisper]", action.Restart)
	}
	// And the busy one is deferred rather than interrupted mid-request
	if !slices.Equal(action.Deferred, []string{"ollama"}) {
		t.Fatalf("Deferred = %v, want [ollama]; an active resource must never be restarted", action.Deferred)
	}
}

// Scenario: report mode records the list and restarts nothing.
func TestReadinessWatcherReportModeRestartsNothing(t *testing.T) {
	// Given a watcher that has seen a host with no accelerator
	var watcher accel.ReadinessWatcher
	watcher.Observe(accel.ReprobeReport, []accel.Backend{accel.BackendCPU}, nil)
	drifted := []accel.DriftedResource{{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU}}

	// When CUDA becomes reachable under the default policy
	action := watcher.Observe(accel.ReprobeReport, []accel.Backend{accel.BackendCUDA, accel.BackendCPU}, drifted)

	// Then the transition is reported
	if !action.Transitioned {
		t.Fatalf("Transitioned = false: %+v", action)
	}
	// And nothing is restarted, because restarting is a lifecycle action
	if len(action.Restart) != 0 {
		t.Fatalf("Restart = %v, want none in report mode", action.Restart)
	}
	// And the drifted resource is still named, so an operator can act
	if !slices.Contains(action.Deferred, "whisper") || !strings.Contains(action.Reason, "whisper") {
		t.Fatalf("action = %+v, want whisper named", action)
	}
}

// Scenario: policy values outside the closed set fall back to report.
func TestReadinessWatcherHonoursTheReprobeMode(t *testing.T) {
	cases := []struct {
		scenario    string
		mode        string
		wantRestart bool
	}{
		{scenario: "Given accel_reprobe off, Then nothing is listed for restart", mode: accel.ReprobeOff, wantRestart: false},
		{scenario: "Given accel_reprobe report, Then nothing is listed for restart", mode: accel.ReprobeReport, wantRestart: false},
		{scenario: "Given accel_reprobe restart, Then the drifted resource is listed", mode: accel.ReprobeRestart, wantRestart: true},
		{scenario: "Given an unrecognised mode, Then it falls back to report", mode: "aggressive", wantRestart: false},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given a watcher that has seen a host with no accelerator
			var watcher accel.ReadinessWatcher
			watcher.Observe(tc.mode, []accel.Backend{accel.BackendCPU}, nil)
			drifted := []accel.DriftedResource{{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU}}

			// When CUDA becomes reachable
			action := watcher.Observe(tc.mode, []accel.Backend{accel.BackendCUDA, accel.BackendCPU}, drifted)

			// Then restarts happen only under the restart mode
			if got := len(action.Restart) > 0; got != tc.wantRestart {
				t.Fatalf("Restart = %v, want restart=%v (mode resolved to %q)", action.Restart, tc.wantRestart, action.Mode)
			}
			// And an unrecognised mode resolves to report rather than to restart
			if tc.mode == "aggressive" && action.Mode != accel.ReprobeReport {
				t.Fatalf("Mode = %q, want %q for an unrecognised value", action.Mode, accel.ReprobeReport)
			}
		})
	}
}

// Scenario: a backend staying reachable is not a transition.
func TestReadinessWatcherIgnoresASteadyState(t *testing.T) {
	// Given a watcher that has already seen CUDA reachable
	var watcher accel.ReadinessWatcher
	watcher.Observe(accel.ReprobeRestart, []accel.Backend{accel.BackendCUDA, accel.BackendCPU}, nil)
	drifted := []accel.DriftedResource{{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU}}

	// When it observes the same host again
	action := watcher.Observe(accel.ReprobeRestart, []accel.Backend{accel.BackendCUDA, accel.BackendCPU}, drifted)

	// Then nothing transitioned, so a permanently drifted resource is not
	// restarted on every supervisor tick
	if action.Transitioned || len(action.Restart) != 0 {
		t.Fatalf("action = %+v, want no transition on a steady state", action)
	}
	if !strings.Contains(action.Reason, "no accelerator backend became reachable") {
		t.Fatalf("Reason = %q, want it to say nothing changed", action.Reason)
	}
}

// Scenario: losing a backend and regaining it transitions again.
func TestReadinessWatcherTransitionsAgainAfterALoss(t *testing.T) {
	// Given a watcher that saw CUDA, then saw it disappear
	var watcher accel.ReadinessWatcher
	watcher.Observe(accel.ReprobeRestart, []accel.Backend{accel.BackendCUDA, accel.BackendCPU}, nil)
	watcher.Observe(accel.ReprobeRestart, []accel.Backend{accel.BackendCPU}, nil)
	drifted := []accel.DriftedResource{{Name: "whisper", Declared: accel.BackendCUDA, Observed: accel.BackendCPU}}

	// When CUDA comes back
	action := watcher.Observe(accel.ReprobeRestart, []accel.Backend{accel.BackendCUDA, accel.BackendCPU}, drifted)

	// Then it is a transition again, because the device genuinely reappeared
	if !action.Transitioned || !slices.Contains(action.Restart, "whisper") {
		t.Fatalf("action = %+v, want a fresh transition restarting whisper", action)
	}
}
