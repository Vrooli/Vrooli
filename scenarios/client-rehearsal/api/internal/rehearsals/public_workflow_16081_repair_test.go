package rehearsals

import (
	"strings"
	"testing"
	"time"
)

func TestPublic16081SeededSwitchDefectIsDetectedAndCorrected(t *testing.T) {
	publicWorkflow := readFixture(t, "public-workflows/16081.json")
	seededWorkflow := readFixture(t, "repairs/16081-seeded.json")
	requireRepairFixtureHash(t, "public workflow", publicWorkflow, "ee642568c470fc3763c5d467b6535c775eb3843b8ba8ee68dc56d20e57618532")
	requireRepairFixtureHash(t, "seeded defect", seededWorkflow, "e590d547912cf26f6c609bcc17057407166c66242df5572ed287e2103e5678bb")
	profile := readFixture(t, "public-workflow-profiles.json")
	service := NewService(newMemoryRepository(), SyntheticEngine{}, func() time.Time { return time.Date(2026, 11, 10, 23, 0, 0, 0, time.UTC) })
	failed, corrected := runPublicWorkflowRepairPair(t, service, CreateInput{Title: "Public PostgreSQL webhook with seeded list routing defect", WorkflowJSON: seededWorkflow, TriggerNodeID: "3ebe4490-ec6f-46af-9cab-0090337add9f", SimulatorProfile: "public-workflow-postgres-catalog", SimulatorSourceJSON: profile, RequiredOutputPaths: []string{"rows"}}, publicWorkflow, "Corrected public PostgreSQL webhook list routing")
	if failed.FaultSummary != "Required output paths missing: rows" || traceHasOperation(failed.Trace, "executeQuery.select") {
		t.Fatalf("seeded list-route defect was not detected before data read: %#v", failed)
	}
	if !traceHasOperation(corrected.Trace, "executeQuery.select") || !strings.Contains(corrected.HandoffMarkdown, "Human review is required") {
		t.Fatalf("corrected revision did not produce a repair handoff: %#v", corrected)
	}
}
