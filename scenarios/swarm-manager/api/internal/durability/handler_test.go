package durability

import (
	"encoding/json"
	"testing"

	"swarm-manager/internal/eventlog"
	"swarm-manager/internal/records"
)

func TestPushbackTransitionRecognizesFailureBeforeCompletion(t *testing.T) {
	raw, err := json.Marshal(eventlog.StatusChangePayload{From: "failed", To: "completed"})
	if err != nil {
		t.Fatal(err)
	}
	if !pushbackTransition(raw) {
		t.Fatal("failed to completed should retain pushback evidence")
	}
	clean, _ := json.Marshal(eventlog.StatusChangePayload{From: "in_progress", To: "completed"})
	if pushbackTransition(clean) {
		t.Fatal("clean completion must not be pushback")
	}
}

// Subjects are "kind:value" tokens, so the record's scenario has to be found
// inside the token. Comparing the other way round never matched anything and
// silently reported every run as free of rework.
func TestRecordMatchesFindsScenarioInsideSubjectToken(t *testing.T) {
	record := records.Record{Scenario: "agent-manager"}
	for _, subject := range []string{
		"path:scenarios/agent-manager/api/internal/durability/projector.go",
		"tool:agent-manager",
		"tool:agent-manager run recent",
	} {
		if !recordMatches(record, "run-1", []string{subject}) {
			t.Fatalf("subject %q should match scenario %q", subject, record.Scenario)
		}
	}
}

// Whole-segment matching keeps a scenario from claiming a sibling's work.
func TestRecordMatchesRejectsUnrelatedAndSiblingScenarios(t *testing.T) {
	for _, tc := range []struct{ scenario, subject string }{
		{"agent-manager", "path:scenarios/swarm-manager/api/main.go"},
		{"agent-manager", "path:scenarios/agent-manager-retired/api/main.go"},
		{"manager", "path:scenarios/agent-manager/api/main.go"},
	} {
		if recordMatches(records.Record{Scenario: tc.scenario}, "run-1", []string{tc.subject}) {
			t.Fatalf("scenario %q must not match subject %q", tc.scenario, tc.subject)
		}
	}
}

// A record with no scenario or backlog ref has nothing to match on. Treating
// its empty scenario as a substring hit would match every subject there is.
func TestRecordMatchesIgnoresEmptyRecordIdentifiers(t *testing.T) {
	if recordMatches(records.Record{}, "run-1", []string{"path:scenarios/agent-manager/api/main.go"}) {
		t.Fatal("a record with no identifiers must not match a subject")
	}
}

func TestRecordMatchesUsesBacklogRef(t *testing.T) {
	record := records.Record{BacklogRef: "fix/durability-subject-join"}
	if !recordMatches(record, "run-1", []string{"backlog:fix/durability-subject-join"}) {
		t.Fatal("backlog ref should match its own subject token")
	}
}

// Scoping matters: an unscoped browse still returns the corpus, but a request
// scoped to one run must not adopt every fix record as that run's rework.
func TestRecordMatchesScopesEmptySubjectsByRun(t *testing.T) {
	record := records.Record{Scenario: "agent-manager"}
	if !recordMatches(record, "", nil) {
		t.Fatal("an unscoped request should still return the corpus")
	}
	if recordMatches(record, "run-1", nil) {
		t.Fatal("a run-scoped request with no subjects must not match every record")
	}
}
