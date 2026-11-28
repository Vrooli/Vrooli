package backlog

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestIsStaleUsesUpdatedAgeAndAcceptancePaths(t *testing.T) {
	now := time.Date(2026, 7, 20, 0, 0, 0, 0, time.UTC)
	fresh := BacklogItem{Updated: now.Add(-13 * 24 * time.Hour).Format(time.RFC3339)}
	if IsStale(fresh, t.TempDir(), now) {
		t.Fatal("fresh item marked stale")
	}
	old := fresh
	old.Updated = now.Add(-14 * 24 * time.Hour).Format(time.RFC3339)
	if !IsStale(old, t.TempDir(), now) {
		t.Fatal("old item was not marked stale")
	}
	reviewed := old
	reviewed.LastReview = &ReviewRecord{ReviewedAt: now.Add(-time.Hour).Format(time.RFC3339), SessionID: "session-1", ProposalID: "proposal-1", Rationale: "Still valid."}
	if IsStale(reviewed, t.TempDir(), now) {
		t.Fatal("recently reviewed item marked stale")
	}
	expiredReview := reviewed
	expiredReview.LastReview = &ReviewRecord{ReviewedAt: now.Add(-14 * 24 * time.Hour).Format(time.RFC3339), SessionID: "session-1", ProposalID: "proposal-1"}
	if !IsStale(expiredReview, t.TempDir(), now) {
		t.Fatal("item with expired review was not marked stale")
	}
	missing := fresh
	missing.AcceptanceAllow = []string{"missing/path/**"}
	if !IsStale(missing, t.TempDir(), now) {
		t.Fatal("missing acceptance path was not marked stale")
	}
}

func TestIsStaleResolvesRepoRelativeAcceptanceFromScenarioAnchor(t *testing.T) {
	projectRoot := t.TempDir()
	writeRepoContractFixture(t, projectRoot)
	scenarioRoot := filepath.Join(projectRoot, "scenarios", "swarm-manager")
	if err := os.MkdirAll(filepath.Join(projectRoot, "scenarios", "deployment-manager"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(scenarioRoot, 0o755); err != nil {
		t.Fatal(err)
	}

	now := time.Date(2026, 8, 29, 9, 0, 0, 0, time.UTC)
	item := BacklogItem{
		Updated:         now.Add(-time.Hour).Format(time.RFC3339),
		AcceptanceAllow: []string{"scenarios/deployment-manager/**"},
	}
	if IsStale(item, scenarioRoot, now) {
		t.Fatal("repo-relative acceptance path was resolved under the scenario directory")
	}

	item.AcceptanceAllow = []string{"scenarios/missing/**"}
	if !IsStale(item, scenarioRoot, now) {
		t.Fatal("missing repo-relative acceptance path was not marked stale")
	}
}
