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
		wantClass     ReconcileClassification
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
			name:          "expired supervised running heartbeat is stale",
			instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning, HostBootID: "boot-current", SupervisorID: "sup-alpha", HeartbeatDeadlineAt: ptrTime(now.Add(-time.Second))},
			wantClass:     ReconcileStaleInstance,
			authoritative: false,
		},
		{
			name:          "expired lifecycle-owned running heartbeat remains rollout-compatible",
			instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning, HostBootID: "boot-current", HeartbeatDeadlineAt: ptrTime(now.Add(-time.Second))},
			wantClass:     ReconcileVerifiedRunning,
			authoritative: true,
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

func TestReconcileRuntimeDoesNotMakeReservedClaimsAuthoritative(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	got := ReconcileRuntime(ReconcileInput{
		Now:           now,
		CurrentBootID: "boot-current",
		Instance:      Instance{InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning, HostBootID: "boot-current"},
		Claims: []PortClaim{
			{ClaimID: "claim-api", InstanceID: "inst-alpha", Scenario: "alpha", PortName: "api", EnvVar: "API_PORT", Port: 18080, Status: ClaimStatusReserved},
		},
	})
	if !got.Authoritative || got.Classification != ReconcileVerifiedRunning {
		t.Fatalf("runtime = %+v, want authoritative verified instance", got)
	}
	if len(got.Claims) != 1 {
		t.Fatalf("len(Claims) = %d, want 1", len(got.Claims))
	}
	if got.Claims[0].Authoritative {
		t.Fatalf("reserved claim = %+v, want non-authoritative", got.Claims[0])
	}
	if got.Claims[0].Classification != ReconcileStaleClaim {
		t.Fatalf("reserved claim classification = %q, want %q", got.Claims[0].Classification, ReconcileStaleClaim)
	}
}

func TestRuntimeEvidenceHelpersKeepHostInspectionAtTheSeam(t *testing.T) {
	pid := 1234
	refs := []ProcessRef{{InstanceID: "inst-alpha", PID: &pid}}
	processes := ProcessEvidenceFromRefs(refs, func(got int) bool {
		if got != pid {
			t.Fatalf("pid = %d, want %d", got, pid)
		}
		return true
	})
	if !processes[strconv.Itoa(pid)].Known || !processes[strconv.Itoa(pid)].Running {
		t.Fatalf("processes = %#v, want known running pid", processes)
	}

	listeners := ListenerEvidenceFromClaims(
		[]PortClaim{
			{ClaimID: "claim-api", Port: 18080, Status: ClaimStatusBound},
			{ClaimID: "claim-ws", Port: 18081, Status: ClaimStatusReserved},
		},
		refs,
		func(port int) ListenerEvidence {
			if port != 18080 {
				t.Fatalf("inspected port = %d, want only bound claim port 18080", port)
			}
			return ListenerEvidence{Known: true, Listening: true}
		},
	)
	if len(listeners) != 1 || !listeners[18080].Known || !listeners[18080].Listening {
		t.Fatalf("listeners = %#v, want bound port listener evidence", listeners)
	}
}

func ptrTime(t time.Time) *time.Time {
	return &t
}

// A supervisor-owned instance has owner_pid = NULL by design — the owner is a
// session, not a process on this row. Reading that NULL as proof of death put
// the entire fleet one lapsed heartbeat away from being classified stale, so a
// supervisor restart was indistinguishable from every scenario dying at once.
func TestReconcileDoesNotCondemnSupervisedInstanceOnMissingOwnerPID(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	lapsed := now.Add(-time.Minute)
	livePID := 4242

	instance := Instance{
		InstanceID:          "inst-alpha",
		Scenario:            "alpha",
		Status:              StatusRunning,
		OwnerKind:           OwnerKindSupervisor,
		OwnerPID:            nil,
		SupervisorID:        "sup-alpha",
		HostBootID:          "boot-current",
		HeartbeatDeadlineAt: &lapsed,
	}
	result := ReconcileRuntime(ReconcileInput{
		Now:           now,
		CurrentBootID: "boot-current",
		Instance:      instance,
		Claims:        []PortClaim{{ClaimID: "claim-a", Port: 8080, Status: ClaimStatusBound}},
		ProcessRefs:   []ProcessRef{{RefID: "ref-a", PID: &livePID}},
		Processes:     map[string]ProcessEvidence{"4242": {Known: true, Running: true}},
		Listeners:     map[int]ListenerEvidence{8080: {Known: true, Listening: true}},
	})
	if !result.Authoritative {
		t.Fatalf("supervised instance with a live process and a listening port was condemned: %s (%s)",
			result.Classification, result.Reason)
	}
}

// The exemption must not become a licence to keep a dead scenario authoritative:
// positive evidence of death still expires it.
func TestReconcileStillCondemnsSupervisedInstanceWithDeadProcessAndNoListener(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	lapsed := now.Add(-time.Minute)
	deadPID := 4242

	result := ReconcileRuntime(ReconcileInput{
		Now:           now,
		CurrentBootID: "boot-current",
		Instance: Instance{
			InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusRunning,
			OwnerKind: OwnerKindSupervisor, SupervisorID: "sup-alpha",
			HostBootID: "boot-current", HeartbeatDeadlineAt: &lapsed,
		},
		Claims:      []PortClaim{{ClaimID: "claim-a", Port: 8080, Status: ClaimStatusBound}},
		ProcessRefs: []ProcessRef{{RefID: "ref-a", PID: &deadPID}},
		Processes:   map[string]ProcessEvidence{"4242": {Known: true, Running: false}},
		Listeners:   map[int]ListenerEvidence{8080: {Known: true, Listening: false}},
	})
	if result.Authoritative || result.Classification != ReconcileStaleInstance {
		t.Fatalf("classification = %s (authoritative=%v), want stale_instance", result.Classification, result.Authoritative)
	}
}

// Lifecycle ownership keeps the old rule: there, a missing PID means the row is
// malformed or the starter vanished, which IS evidence.
func TestReconcileStillCondemnsLifecycleInstanceWithMissingOwnerPID(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	lapsed := now.Add(-time.Minute)

	result := ReconcileRuntime(ReconcileInput{
		Now:           now,
		CurrentBootID: "boot-current",
		Instance: Instance{
			InstanceID: "inst-alpha", Scenario: "alpha", Status: StatusStarting,
			OwnerKind: OwnerKindLifecycle, OwnerPID: nil,
			HostBootID: "boot-current", HeartbeatDeadlineAt: &lapsed,
		},
	})
	if result.Authoritative || result.Classification != ReconcileStaleInstance {
		t.Fatalf("classification = %s (authoritative=%v), want stale_instance", result.Classification, result.Authoritative)
	}
}
