package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic22567SeededSplitSelectorDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/22567.json")
	seededWorkflow := readFixture(t, "repairs/22567-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(publicWorkflow))); got != "31df6c3f90e7d91672ab7c004064c43904c2c502a4804d2adf52c7c6c2a5f702" {
		t.Fatalf("public fixture hash changed: %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "7567df90debaf0ac1fe02e057531884c0ba53d47895614589d239f98fd4887b6" {
		t.Fatalf("seeded defect fixture hash changed: %s", got)
	}
	profile := readFixture(t, "public-workflow-profiles.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 10, 23, 40, 0, 0, time.UTC) })
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public invoice import with seeded Split Out selector defect", WorkflowJSON: seededWorkflow, TriggerNodeID: "c09-starter-start", SimulatorProfile: "public-invoice-and-payment-catalog", SimulatorSourceJSON: profile})
	if err != nil {
		t.Fatalf("run seeded Split Out defect: %v", err)
	}
	if failed.State != StateFailed || !strings.Contains(failed.FaultSummary, "Split Out field assignment.missingInvoices is missing") {
		t.Fatalf("seeded record-loss defect was not detected: %#v", failed)
	}
	corrected, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public invoice Split Out selector", WorkflowJSON: publicWorkflow, TriggerNodeID: "c09-starter-start", SimulatorProfile: "public-invoice-and-payment-catalog", SimulatorSourceJSON: profile, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected public graph: %v", err)
	}
	if corrected.State != StateNeedsReview || !traceHasNode(corrected.Trace, "Split invoices") || !traceHasOperation(corrected.Trace, "GET /v1/invoices-and-payments") || !traceHasOperation(corrected.Trace, "repair lineage") || !strings.Contains(corrected.FaultSummary, "failed during execution") || !strings.Contains(corrected.RepairSummary, "no output-path assertions were configured") {
		t.Fatalf("corrected public workflow did not produce a repair handoff: %#v", corrected)
	}
}
