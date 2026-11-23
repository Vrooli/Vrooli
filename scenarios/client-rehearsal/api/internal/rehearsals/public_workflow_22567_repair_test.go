package rehearsals

import (
	"strings"
	"testing"
	"time"
)

func TestPublic22567SeededSplitSelectorDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/22567.json")
	seededWorkflow := readFixture(t, "repairs/22567-seeded.json")
	requireRepairFixtureHash(t, "public", publicWorkflow, "31df6c3f90e7d91672ab7c004064c43904c2c502a4804d2adf52c7c6c2a5f702")
	requireRepairFixtureHash(t, "seeded defect", seededWorkflow, "7567df90debaf0ac1fe02e057531884c0ba53d47895614589d239f98fd4887b6")
	profile := readFixture(t, "public-workflow-profiles.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 10, 23, 40, 0, 0, time.UTC) })
	failed, corrected := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public invoice import with seeded Split Out selector defect", WorkflowJSON: seededWorkflow, TriggerNodeID: "c09-starter-start", SimulatorProfile: "public-invoice-and-payment-catalog", SimulatorSourceJSON: profile}, publicWorkflow, "Corrected public invoice Split Out selector")
	if !strings.Contains(failed.FaultSummary, "Split Out field assignment.missingInvoices is missing") {
		t.Fatalf("seeded record-loss defect was not detected: %#v", failed)
	}
	if !traceHasNode(corrected.Trace, "Split invoices") || !traceHasOperation(corrected.Trace, "GET /v1/invoices-and-payments") || !strings.Contains(corrected.FaultSummary, "failed during execution") || !strings.Contains(corrected.RepairSummary, "no output-path assertions were configured") {
		t.Fatalf("corrected public workflow did not produce a repair handoff: %#v", corrected)
	}
}
