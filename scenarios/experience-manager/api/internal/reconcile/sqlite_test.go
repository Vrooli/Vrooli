package reconcile

import (
	"context"
	"testing"

	apidb "github.com/vrooli/api-core/database"

	testdb "experience-manager/internal/testutil/db"
)

func TestSQLiteRepositorySavesAndListsEvidence(t *testing.T) {
	db := testdb.NewSQLite(t)
	if err := apidb.EnsureSchemas(context.Background(), db, apidb.SchemaProviderFunc(Schema)); err != nil {
		t.Fatalf("EnsureSchemas: %v", err)
	}
	repo := NewSQLiteRepository(db)
	ctx := context.Background()

	want := Evidence{
		ID:             "ev-test",
		Scenario:       "demo",
		PageID:         "home",
		Route:          "/",
		StateID:        "default",
		ViewportID:     "mobile",
		ViewportWidth:  390,
		ViewportHeight: 844,
		ClaimID:        "primary-present",
		ClaimType:      "element-present",
		Verdict:        "passed",
		CaptureRef:     "scenario=demo,path=/",
		AXNodeJSON:     `{"role":"button"}`,
		Message:        "claim proven",
		CheckedAt:      "2026-07-05T12:00:00Z",
	}
	if err := repo.SaveEvidence(ctx, want); err != nil {
		t.Fatalf("SaveEvidence: %v", err)
	}
	got, err := repo.ListEvidence(ctx, EvidenceFilter{Scenario: "demo", PageID: "home", ClaimID: "primary-present"})
	if err != nil {
		t.Fatalf("ListEvidence: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("evidence rows = %d, want 1: %+v", len(got), got)
	}
	if got[0] != want {
		t.Fatalf("evidence = %+v, want %+v", got[0], want)
	}
}
