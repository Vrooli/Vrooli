package integrations

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/schedule"
	_ "modernc.org/sqlite"
)

func TestFixtureConnectionSyncPreservesReadOnlyBoundary(t *testing.T) {
	db, err := sql.Open("sqlite", "file:integrations-fixture-test?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if _, err := db.Exec(Schema()); err != nil {
		t.Fatal(err)
	}
	clock := schedule.NewFake(time.Date(2026, 9, 19, 12, 0, 0, 0, time.UTC))
	repo := NewSQLiteRepository(database.NewFromPrimary(db), clock)
	service := NewService(repo)
	created, err := service.CreateFixture(context.Background(), "Sample")
	if err != nil {
		t.Fatal(err)
	}
	if !created.ReadOnly || created.SourceKind != "fixture" || created.Status != "connected" {
		t.Fatalf("created=%#v", created)
	}
	synced, err := service.Sync(context.Background(), SyncInput{ID: created.ID, ExpectedRevision: created.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if synced.Status != "synced" || synced.ImportedEventCount != 3 || synced.BusyMinutes != 120 || synced.Revision != 2 {
		t.Fatalf("synced=%#v", synced)
	}
	if _, err := service.Sync(context.Background(), SyncInput{ID: created.ID, ExpectedRevision: created.Revision}); err == nil {
		t.Fatal("expected revision conflict")
	}
	disconnected, err := service.Disconnect(context.Background(), SyncInput{ID: synced.ID, ExpectedRevision: synced.Revision})
	if err != nil {
		t.Fatal(err)
	}
	if disconnected.Status != "disconnected" || disconnected.ReadOnly != true {
		t.Fatalf("disconnected=%#v", disconnected)
	}
}
