package scenarioruntime

import (
	"fmt"
	"testing"
	"time"
)

func TestPlanSupervisionRenewsFreshCurrentBootRunningInstance(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	instance := supervisedRunningInstance("inst-alpha", "alpha", 1, "sup-alpha")

	plan := PlanSupervision(SupervisionPlanInput{
		Now:           now,
		CurrentBootID: "boot-current",
		SupervisorID:  "sup-alpha",
		Instances:     []Instance{instance},
	})

	if got := flattenRenewals(plan); len(got) != 1 || got[0].InstanceID != instance.InstanceID {
		t.Fatalf("renewals = %#v, want inst-alpha", got)
	}
	if len(plan.Expire) != 0 || len(plan.Unverified) != 0 {
		t.Fatalf("expire/unverified = %#v/%#v, want none", plan.Expire, plan.Unverified)
	}
}

func TestPlanSupervisionPlansHealthProbeFromHealthSnapshotCadence(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	instance := supervisedRunningInstance("inst-alpha", "alpha", 1, "sup-alpha")
	freshCheckedAt := now.Add(-10 * time.Second)
	staleCheckedAt := now.Add(-2 * time.Minute)

	fresh := PlanSupervision(SupervisionPlanInput{
		Now:           now,
		CurrentBootID: "boot-current",
		SupervisorID:  "sup-alpha",
		Instances:     []Instance{instance},
		HealthByInstance: map[string]HealthSnapshot{
			instance.InstanceID: {InstanceID: instance.InstanceID, Scenario: instance.Scenario, CheckedAt: &freshCheckedAt},
		},
		HealthInterval: time.Minute,
	})
	if len(fresh.HealthProbes) != 0 {
		t.Fatalf("fresh HealthProbes = %#v, want none", fresh.HealthProbes)
	}

	stale := PlanSupervision(SupervisionPlanInput{
		Now:           now,
		CurrentBootID: "boot-current",
		SupervisorID:  "sup-alpha",
		Instances:     []Instance{instance},
		HealthByInstance: map[string]HealthSnapshot{
			instance.InstanceID: {InstanceID: instance.InstanceID, Scenario: instance.Scenario, CheckedAt: &staleCheckedAt},
		},
		HealthInterval: time.Minute,
	})
	if len(stale.HealthProbes) != 1 || stale.HealthProbes[0] != instance.InstanceID {
		t.Fatalf("stale HealthProbes = %#v, want inst-alpha", stale.HealthProbes)
	}
}

func TestPlanSupervisionCanTakeOverExpiredPriorSupervisorAfterRevalidation(t *testing.T) {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	instance := supervisedRunningInstance("inst-alpha", "alpha", 1, "sup-old")
	expired := now.Add(-time.Minute)
	instance.HeartbeatDeadlineAt = &expired

	plan := PlanSupervision(SupervisionPlanInput{
		Now:           now,
		CurrentBootID: "boot-current",
		SupervisorID:  "sup-new",
		Instances:     []Instance{instance},
	})

	got := flattenRenewals(plan)
	if len(got) != 1 || got[0].InstanceID != instance.InstanceID || got[0].SupervisorID != "sup-new" {
		t.Fatalf("renewals = %#v, want takeover renewal by sup-new", got)
	}
	if len(plan.Expire) != 0 || len(plan.Unverified) != 0 {
		t.Fatalf("expire/unverified = %#v/%#v, want none", plan.Expire, plan.Unverified)
	}
}

func TestPlanSupervisionExpiresPreviousBootInstance(t *testing.T) {
	instance := supervisedRunningInstance("inst-alpha", "alpha", 1, "sup-alpha")
	instance.HostBootID = "boot-old"

	plan := PlanSupervision(SupervisionPlanInput{
		Now:           time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		CurrentBootID: "boot-current",
		Instances:     []Instance{instance},
	})

	if len(plan.Expire) != 1 || plan.Expire[0].Classification != ReconcileStaleInstance {
		t.Fatalf("expire = %#v, want stale previous boot", plan.Expire)
	}
	if got := flattenRenewals(plan); len(got) != 0 {
		t.Fatalf("renewals = %#v, want none", got)
	}
}

func TestPlanSupervisionDoesNotRenewDeadProcessWithoutListener(t *testing.T) {
	pid := 31337
	instance := supervisedRunningInstance("inst-alpha", "alpha", 1, "sup-alpha")
	claim := PortClaim{ClaimID: "claim-alpha-api", InstanceID: instance.InstanceID, Scenario: "alpha", PortName: "api", Port: 18080, Status: ClaimStatusBound}
	ref := ProcessRef{RefID: "proc-alpha", InstanceID: instance.InstanceID, PID: &pid, HostBootID: "boot-current"}

	plan := PlanSupervision(SupervisionPlanInput{
		Now:              time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		CurrentBootID:    "boot-current",
		Instances:        []Instance{instance},
		ClaimsByInstance: map[string][]PortClaim{instance.InstanceID: {claim}},
		RefsByInstance:   map[string][]ProcessRef{instance.InstanceID: {ref}},
		Processes:        map[string]ProcessEvidence{"31337": {Known: true, Running: false}},
		Listeners:        map[int]ListenerEvidence{18080: {Known: true, Listening: false}},
	})

	if len(plan.Expire) != 1 {
		t.Fatalf("expire = %#v, want dead process/no listener expiration", plan.Expire)
	}
	if got := flattenRenewals(plan); len(got) != 0 {
		t.Fatalf("renewals = %#v, want none", got)
	}
}

func TestPlanSupervisionMarksUnavailableListenerUnverified(t *testing.T) {
	pid := 31337
	instance := supervisedRunningInstance("inst-alpha", "alpha", 1, "sup-alpha")
	claim := PortClaim{ClaimID: "claim-alpha-api", InstanceID: instance.InstanceID, Scenario: "alpha", PortName: "api", Port: 18080, Status: ClaimStatusBound}
	ref := ProcessRef{RefID: "proc-alpha", InstanceID: instance.InstanceID, PID: &pid, HostBootID: "boot-current"}

	plan := PlanSupervision(SupervisionPlanInput{
		Now:              time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		CurrentBootID:    "boot-current",
		Instances:        []Instance{instance},
		ClaimsByInstance: map[string][]PortClaim{instance.InstanceID: {claim}},
		RefsByInstance:   map[string][]ProcessRef{instance.InstanceID: {ref}},
		Processes:        map[string]ProcessEvidence{"31337": {Known: true, Running: false}},
		Listeners:        map[int]ListenerEvidence{18080: {Known: false}},
	})

	if len(plan.Unverified) != 1 || plan.Unverified[0].Classification != ReconcileUnverified {
		t.Fatalf("unverified = %#v, want unavailable listener unverified", plan.Unverified)
	}
	if got := flattenRenewals(plan); len(got) != 0 {
		t.Fatalf("renewals = %#v, want none", got)
	}
}

func TestPlanSupervisionIgnoresTerminalInstances(t *testing.T) {
	instance := supervisedRunningInstance("inst-alpha", "alpha", 1, "sup-alpha")
	instance.Status = StatusStopped

	plan := PlanSupervision(SupervisionPlanInput{
		Now:           time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		CurrentBootID: "boot-current",
		Instances:     []Instance{instance},
	})

	if got := flattenRenewals(plan); len(got) != 0 || len(plan.Expire) != 0 || len(plan.Unverified) != 0 {
		t.Fatalf("plan = %#v, want terminal instance ignored", plan)
	}
}

func TestPlanSupervisionBatchesDeterministicallyAtScale(t *testing.T) {
	instances := make([]Instance, 0, 1000)
	for i := 0; i < 1000; i++ {
		scenario := fmt.Sprintf("scenario-%04d", i)
		instances = append(instances, supervisedRunningInstance("inst-"+scenario, scenario, 1, "sup-alpha"))
	}

	plan := PlanSupervision(SupervisionPlanInput{
		Now:           time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC),
		CurrentBootID: "boot-current",
		Instances:     instances,
		BatchSize:     250,
	})

	if len(plan.RenewalBatches) != 4 {
		t.Fatalf("len(RenewalBatches) = %d, want 4", len(plan.RenewalBatches))
	}
	for i, batch := range plan.RenewalBatches {
		if len(batch) != 250 {
			t.Fatalf("batch %d len = %d, want 250", i, len(batch))
		}
	}
	if first := plan.RenewalBatches[0][0].InstanceID; first != "inst-scenario-0000" {
		t.Fatalf("first renewal = %q, want deterministic scenario order", first)
	}
	if last := plan.RenewalBatches[3][249].InstanceID; last != "inst-scenario-0999" {
		t.Fatalf("last renewal = %q, want deterministic scenario order", last)
	}
}

func supervisedRunningInstance(instanceID string, scenario string, generation int64, supervisorID string) Instance {
	now := time.Date(2026, 5, 9, 12, 0, 0, 0, time.UTC)
	deadline := now.Add(time.Minute)
	return Instance{
		InstanceID:          instanceID,
		Scenario:            scenario,
		Generation:          generation,
		Status:              StatusRunning,
		StartedAt:           now.Add(-time.Minute),
		UpdatedAt:           now,
		LastHeartbeatAt:     &now,
		HeartbeatDeadlineAt: &deadline,
		OwnerKind:           OwnerKindSupervisor,
		HostBootID:          "boot-current",
		HostSessionID:       "session-current",
		SupervisorID:        supervisorID,
		SupervisionPolicy:   SupervisionPolicyManaged,
		SchemaVersion:       SchemaVersion,
	}
}

func flattenRenewals(plan SupervisionPlan) []SupervisionClaim {
	var out []SupervisionClaim
	for _, batch := range plan.RenewalBatches {
		out = append(out, batch...)
	}
	return out
}
