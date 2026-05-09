package scenarioruntime

import (
	"strconv"
	"testing"
	"time"
)

func TestReconcileRuntimeClassifiesCrashRebootStates(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	deadPID := 424242
	livePID := 424243
	claim := PortClaim{ClaimID: "claim-api", InstanceID: "inst-alpha", Scenario: "alpha", PortName: "api", EnvVar: "API_PORT", Port: 18080, Status: ClaimStatusBound}
	tests := []struct {
		name          string
		instance      Instance
		refs          []ProcessRef
		processes     map[string]ProcessEvidence
		listeners     map[int]ListenerEvidence
		wantClass     string
		authoritative bool
	}{
		{
			name:          "current boot without pid dependency is authoritative",
			instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning, HostBootID: "boot-current"},
			wantClass:     ReconcileVerifiedRunning,
			authoritative: true,
		},
		{
			name:          "previous boot is stale even if claim remains bound",
			instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning, HostBootID: "boot-previous"},
			wantClass:     ReconcileStaleInstance,
			authoritative: false,
		},
		{
			name:          "schema v1 instance without boot identity is unverified",
			instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning},
			wantClass:     ReconcileUnverified,
			authoritative: false,
		},
		{
			name:          "stale starting heartbeat is stale",
			instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusStarting, HostBootID: "boot-current", HeartbeatDeadlineAt: ptrTime(now.Add(-time.Second))},
			wantClass:     ReconcileStaleInstance,
			authoritative: false,
		},
		{
			name:      "all dead process refs and no listeners is stale sudden stop",
			instance:  Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning, HostBootID: "boot-current"},
			refs:      []ProcessRef{{InstanceID: "inst-alpha", PID: &deadPID, Status: "running"}},
			processes: map[string]ProcessEvidence{strconv.Itoa(deadPID): {Known: true, Running: false}},
			listeners: map[int]ListenerEvidence{18080: {Known: true, Listening: false}},
			wantClass: ReconcileStaleInstance,
		},
		{
			name:          "dead process ref with listener remains diagnostic but authoritative",
			instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning, HostBootID: "boot-current"},
			refs:          []ProcessRef{{InstanceID: "inst-alpha", PID: &deadPID, Status: "running"}},
			processes:     map[string]ProcessEvidence{strconv.Itoa(deadPID): {Known: true, Running: false}},
			listeners:     map[int]ListenerEvidence{18080: {Known: true, Listening: true}},
			wantClass:     ReconcileVerifiedRunning,
			authoritative: true,
		},
		{
			name:          "live process ref is authoritative",
			instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning, HostBootID: "boot-current"},
			refs:          []ProcessRef{{InstanceID: "inst-alpha", PID: &livePID, Status: "running"}},
			processes:     map[string]ProcessEvidence{strconv.Itoa(livePID): {Known: true, Running: true}},
			wantClass:     ReconcileVerifiedRunning,
			authoritative: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ReconcileRuntime(ReconcileInput{
				Now:           now,
				CurrentBootID: "boot-current",
				Instance:      tt.instance,
				Claims:        []PortClaim{claim},
				ProcessRefs:   tt.refs,
				Processes:     tt.processes,
				Listeners:     tt.listeners,
			})
			if got.Classification != tt.wantClass {
				t.Fatalf("Classification = %q, want %q (%s)", got.Classification, tt.wantClass, got.Reason)
			}
			if got.Authoritative != tt.authoritative {
				t.Fatalf("Authoritative = %v, want %v (%s)", got.Authoritative, tt.authoritative, got.Reason)
			}
		})
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}
