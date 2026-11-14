package rehearsals

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"
)

func TestPublic16081SeededSwitchDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/16081.json")
	seededWorkflow := readFixture(t, "repairs/16081-seeded.json")
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(publicWorkflow))); got != "ee642568c470fc3763c5d467b6535c775eb3843b8ba8ee68dc56d20e57618532" {
		t.Fatalf("public workflow fingerprint changed: %s", got)
	}
	if got := fmt.Sprintf("%x", sha256.Sum256([]byte(seededWorkflow))); got != "e590d547912cf26f6c609bcc17057407166c66242df5572ed287e2103e5678bb" {
		t.Fatalf("seeded defect fixture changed: %s", got)
	}
	profile := readFixture(t, "public-workflow-profiles.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC) })
	failed, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Public PostgreSQL webhook with seeded list routing defect", WorkflowJSON: seededWorkflow, TriggerNodeID: "3ebe4490-ec6f-46af-9cab-0090337add9f", SimulatorProfile: "public-workflow-postgres-catalog", SimulatorSourceJSON: profile, RequiredOutputPaths: []string{"rows"}})
	if err != nil {
		t.Fatalf("run seeded defect: %v", err)
	}
	if failed.State != StateFailed || failed.FaultSummary != "Required output paths missing: rows" || traceHasOperation(failed.Trace, "executeQuery.select") {
		t.Fatalf("seeded list-route defect was not detected before data read: %#v", failed)
	}
	corrected, err := service.CreateAndRun(context.Background(), CreateInput{Title: "Corrected public PostgreSQL webhook list routing", WorkflowJSON: publicWorkflow, TriggerNodeID: "3ebe4490-ec6f-46af-9cab-0090337add9f", SimulatorProfile: "public-workflow-postgres-catalog", SimulatorSourceJSON: profile, RequiredOutputPaths: []string{"rows"}, RepairOfID: failed.ID})
	if err != nil {
		t.Fatalf("run corrected public workflow revision: %v", err)
	}
	if corrected.State != StateNeedsReview || !traceHasOperation(corrected.Trace, "executeQuery.select") || !traceHasOperation(corrected.Trace, "repair lineage") || !strings.Contains(corrected.HandoffMarkdown, "Human review is required") {
		t.Fatalf("corrected revision did not produce a repair handoff: %#v", corrected)
	}
}
