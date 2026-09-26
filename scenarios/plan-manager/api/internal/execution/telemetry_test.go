package execution

import (
	"testing"
	"time"
)

func TestTelemetryReportKeepsReuseRecoveryAndUnknownsTyped(t *testing.T) {
	base := time.Date(2026, 9, 5, 12, 0, 0, 0, time.UTC)
	sink := &MemoryTelemetrySink{}
	events := []ExecutionTelemetryEvent{
		{ID: "request-1", Kind: TelemetryRequest, OccurredAt: base, TaskID: "task-1", PlanID: "plan-1", ValidationID: "validation-1"},
		{ID: "execute-1", Kind: TelemetryExecution, OccurredAt: base.Add(time.Second), TaskID: "task-1", PlanID: "plan-1", ValidationID: "validation-1", Duration: 3 * time.Second, PolicyIdentity: "policy-1", ContentIdentity: "content-1"},
		{ID: "reuse-2", Kind: TelemetryReuse, OccurredAt: base.Add(2 * time.Second), TaskID: "task-2", PlanID: "plan-1", ValidationID: "validation-2", PolicyIdentity: "policy-1", ContentIdentity: "content-1"},
		{ID: "recovery-1", Kind: TelemetryRecovery, OccurredAt: base.Add(3 * time.Second), TaskID: "task-1", PlanID: "plan-1", Reason: "operator reattached"},
		{ID: "family-1", Kind: TelemetryExecution, OccurredAt: base.Add(4 * time.Second), TaskID: "task-1", PlanID: "plan-1", FamilyID: "family-1"},
		{ID: "family-2", Kind: TelemetryCompletion, OccurredAt: base.Add(9 * time.Second), TaskID: "task-1", PlanID: "plan-1", FamilyID: "family-1"},
	}
	for _, event := range events {
		if err := sink.Append(event); err != nil {
			t.Fatal(err)
		}
	}
	report := BuildTelemetryReport(sink.Events())
	if report.Validation.Requested != 1 || report.Validation.Executed != 1 || report.Validation.Reused != 1 || report.Validation.Unknown != 1 || report.Validation.Duration != 3*time.Second {
		t.Fatalf("validation report = %+v", report.Validation)
	}
	if report.ManualRecovery != 1 {
		t.Fatalf("manual recovery = %d", report.ManualRecovery)
	}
	if report.CriticalPath.Families != 1 || report.CriticalPath.Known != 1 || report.CriticalPath.Duration != 5*time.Second {
		t.Fatalf("critical path = %+v", report.CriticalPath)
	}
	if report.Coordination.Unknown <= 0 {
		t.Fatalf("missing unknown coordination interval: %+v", report.Coordination)
	}
}

func TestTelemetryRejectsEventsWithoutOwnerIdentity(t *testing.T) {
	if err := (&MemoryTelemetrySink{}).Append(ExecutionTelemetryEvent{ID: "event-1", Kind: TelemetryRequest, OccurredAt: time.Now()}); err == nil {
		t.Fatal("accepted ownerless telemetry")
	}
}
