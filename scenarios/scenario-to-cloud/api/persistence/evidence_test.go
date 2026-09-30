package persistence

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"
	"time"

	"scenario-to-cloud/evidence"

	_ "modernc.org/sqlite"
)

// TestEvidenceRecordsAreAppendOnlyAndPublicationsAreIdempotent
// [REQ:STC-P0-035] proves P17-A04 at the storage layer: a record id cannot
// be rewritten, a rerun is a second row, and a publication request key
// resolves to exactly one publication.
func TestEvidenceRecordsAreAppendOnlyAndPublicationsAreIdempotent(t *testing.T) {
	db, err := sql.Open("sqlite", "file:cloud-evidence-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	defer db.Close()
	ctx := context.Background()
	repo := NewRepository(db)
	if err := repo.InitSchemaOnDialect(ctx, db, "sqlite"); err != nil {
		t.Fatalf("initialize schema: %v", err)
	}
	if _, err := db.ExecContext(ctx, `INSERT INTO deployments (id, name, scenario_id, status, manifest) VALUES ($1, $2, $3, $4, $5)`, "dep-1", "demo", "demo", "deployed", `{}`); err != nil {
		t.Fatalf("insert deployment: %v", err)
	}
	release := "sha256:" + strings.Repeat("a", 64)
	at := time.Date(2026, 9, 9, 12, 0, 0, 0, time.UTC)
	failed := evidence.Record{
		SchemaVersion: 1, ID: "rec-1", ProfileID: evidence.ProfileCloudLaunchV1, CaseID: "GOV-01", Lane: "api", Disposition: evidence.DispositionFailed, Reason: "assertion failed",
		Binding: evidence.Binding{ProducerRef: evidence.ProducerRef, DeploymentID: "dep-1", ReleaseDigest: release, TargetKey: "machine:a", OperationID: "op-1"}, ObservedAt: at, RecordedAt: at,
	}
	if err := repo.AppendEvidenceRecord(ctx, failed); err != nil {
		t.Fatal(err)
	}
	rewrite := failed
	rewrite.Disposition = evidence.DispositionPassed
	if err := repo.AppendEvidenceRecord(ctx, rewrite); !errors.Is(err, ErrEvidenceRecordExists) {
		t.Fatalf("rewrite must be refused, got %v", err)
	}
	rerun := failed
	rerun.ID, rerun.Disposition, rerun.ReceiptRefs, rerun.ObservedAt = "rec-2", evidence.DispositionPassed, []string{"cloud-target:receipt:2"}, at.Add(time.Minute)
	if err := repo.AppendEvidenceRecord(ctx, rerun); err != nil {
		t.Fatal(err)
	}
	records, err := repo.ListEvidenceRecords(ctx, release, "dep-1")
	if err != nil || len(records) != 2 || records[0].ID != "rec-1" || records[0].Disposition != evidence.DispositionFailed || records[1].ID != "rec-2" {
		t.Fatalf("records = %+v err=%v", records, err)
	}
	if other, _ := repo.ListEvidenceRecords(ctx, "sha256:"+strings.Repeat("b", 64), ""); len(other) != 0 {
		t.Fatalf("foreign release must list nothing: %+v", other)
	}

	pub := evidence.Publication{ID: "pub-1", DeploymentID: "dep-1", RequestKey: "req-1", IdentityDigest: "sha256:id", ReleaseDigest: release, State: evidence.StateRequested, TargetKey: "machine:a"}
	stored, created, err := repo.CreatePublication(ctx, pub)
	if err != nil || !created || stored.ID != "pub-1" {
		t.Fatalf("create = %+v created=%v err=%v", stored, created, err)
	}
	replay := pub
	replay.ID = "pub-dup"
	stored, created, err = repo.CreatePublication(ctx, replay)
	if err != nil || created || stored.ID != "pub-1" {
		t.Fatalf("replay = %+v created=%v err=%v", stored, created, err)
	}
	stored.State = evidence.StatePublished
	published := at.Add(2 * time.Minute)
	stored.PublishedAt = &published
	stored.ActivatedReleaseDigest = release
	if err := repo.UpdatePublication(ctx, *stored); err != nil {
		t.Fatal(err)
	}
	list, err := repo.ListPublications(ctx, "dep-1")
	if err != nil || len(list) != 1 || list[0].State != evidence.StatePublished || list[0].PublishedAt == nil {
		t.Fatalf("list = %+v err=%v", list, err)
	}
	if got, _ := repo.GetPublication(ctx, "pub-1"); got == nil || got.ActivatedReleaseDigest != release {
		t.Fatalf("get = %+v", got)
	}
}
