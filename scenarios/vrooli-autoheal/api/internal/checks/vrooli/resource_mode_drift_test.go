package vrooli

import (
	"context"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// Feature: one check asks whether a resource is on the device it asked for
//
//	As an operator
//	I want a check that fails when a resource serves from the CPU while it
//	declared an accelerator
//	So that the state every other signal reports as healthy becomes visible.

// Scenario: the check reports what the control plane observed.
func TestModeDriftCheckClassifiesPlacement(t *testing.T) {
	cases := []struct {
		scenario   string
		json       string
		wantStatus checks.Status
		wantSaid   string
	}{
		{
			scenario:   "Given a resource on its declared backend, Then the check passes",
			json:       `{"success":true,"name":"ollama","running":true,"resource":{"running":true,"declared_mode":"cuda","observed_mode":"cuda","mode_drift":false}}`,
			wantStatus: checks.StatusOK,
			wantSaid:   "running on its declared cuda backend",
		},
		{
			scenario:   "Given a resource serving from the CPU while it declared cuda, Then the check warns and names both",
			json:       `{"success":true,"name":"whisper","running":true,"resource":{"running":true,"declared_mode":"cuda","observed_mode":"cpu","mode_drift":true,"mode_reason":"nvidia-smi lists no compute process for pid 2187104"}}`,
			wantStatus: checks.StatusWarning,
			wantSaid:   "declared cuda but is running on cpu",
		},
		{
			scenario:   "Given a resource that declares no accelerator, Then the check passes without opinion",
			json:       `{"success":true,"name":"postgres","running":true,"resource":{"running":true}}`,
			wantStatus: checks.StatusOK,
			wantSaid:   "declares no accelerator backend",
		},
		{
			scenario:   "Given a stopped resource, Then placement is not this check's business",
			json:       `{"success":true,"name":"kokoro","running":false,"resource":{"running":false,"declared_mode":"cuda"}}`,
			wantStatus: checks.StatusOK,
			wantSaid:   "not running, so it has no placement to verify",
		},
		{
			scenario:   "Given a running resource whose placement could not be read, Then the check warns rather than passing",
			json:       `{"success":true,"name":"reranker","running":true,"resource":{"running":true,"declared_mode":"cuda","observed_mode":"","mode_drift":false}}`,
			wantStatus: checks.StatusWarning,
			wantSaid:   "placement could not be read",
		},
	}

	for _, tc := range cases {
		t.Run(tc.scenario, func(t *testing.T) {
			// Given the control plane's answer for that resource
			check := NewModeDriftCheck("fixture", WithModeDriftExecutor(&stubExecutor{output: tc.json}))

			// When the check runs
			result := check.Run(context.Background())

			// Then the verdict and the message match what was observed
			if result.Status != tc.wantStatus {
				t.Fatalf("Status = %v, want %v (message = %q)", result.Status, tc.wantStatus, result.Message)
			}
			if !strings.Contains(result.Message, tc.wantSaid) {
				t.Fatalf("Message = %q, want it to contain %q", result.Message, tc.wantSaid)
			}
		})
	}
}

// Scenario: the check id and category place it beside the resource checks.
func TestModeDriftCheckIdentity(t *testing.T) {
	check := NewModeDriftCheck("whisper")

	// Then its id shares the resource- prefix, so it inherits resource defaults
	if got := check.ID(); got != "resource-whisper-mode-drift" {
		t.Fatalf("ID() = %q, want resource-whisper-mode-drift", got)
	}
	if !strings.HasPrefix(check.ID(), "resource-") {
		t.Fatal("the check id must start with resource- so auto-heal defaults apply to it")
	}
	if check.Category() != checks.CategoryResource {
		t.Fatalf("Category() = %v, want %v", check.Category(), checks.CategoryResource)
	}
}

// Scenario: the restart action is offered only while the resource is drifted.
//
// Restarting a resource that is already on its backend costs a cold start and
// changes nothing.
func TestModeDriftCheckOffersRestartOnlyWhileDrifted(t *testing.T) {
	check := NewModeDriftCheck("whisper")

	// Given a passing result
	clean := check.RecoveryActions(&checks.Result{Details: map[string]interface{}{"modeDrift": false}})
	if actionAvailable(clean, "restart") {
		t.Fatal("restart offered for a resource that is on its declared backend")
	}
	// And explain is always available, because reading is never destructive
	if !actionAvailable(clean, "explain") {
		t.Fatal("explain must always be available")
	}

	// Given a drifted result
	drifted := check.RecoveryActions(&checks.Result{Details: map[string]interface{}{"modeDrift": true}})
	if !actionAvailable(drifted, "restart") {
		t.Fatal("restart not offered for a drifted resource")
	}
}

func actionAvailable(actions []checks.RecoveryAction, id string) bool {
	for _, action := range actions {
		if action.ID == id {
			return action.Available
		}
	}
	return false
}

type stubExecutor struct {
	output string
	err    error
}

func (s *stubExecutor) CombinedOutput(context.Context, string, ...string) ([]byte, error) {
	return []byte(s.output), s.err
}

func (s *stubExecutor) Output(context.Context, string, ...string) ([]byte, error) {
	return []byte(s.output), s.err
}

func (s *stubExecutor) Run(context.Context, string, ...string) error { return s.err }
