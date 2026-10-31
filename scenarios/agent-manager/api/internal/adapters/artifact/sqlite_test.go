package artifact_test

import (
	"bytes"
	"context"
	"path/filepath"
	"testing"
	"time"

	"agent-manager/internal/adapters/artifact"

	"github.com/google/uuid"
	"github.com/jmoiron/sqlx"
	coredb "github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	corestorage "github.com/vrooli/api-core/storage"
	_ "modernc.org/sqlite"
)

func newSQLiteCollector(t *testing.T) (*artifact.SQLiteCollector, *filerouting.RoutedRoots) {
	t.Helper()
	db, err := sqlx.Open("sqlite", "file:artifact-retention?mode=memory&cache=shared")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(artifact.Schema()); err != nil {
		t.Fatalf("schema: %v", err)
	}
	roots := filerouting.New(corestorage.Paths{DataDir: filepath.Join(t.TempDir(), "primary")})
	if err := roots.InstallTestRoots(corestorage.Paths{DataDir: filepath.Join(t.TempDir(), "test")}, "artifact-test", time.Minute); err != nil {
		t.Fatal(err)
	}
	return artifact.NewSQLiteCollector(db, roots), roots
}

func TestSQLiteCollectorStoresInRoutedDataRootAndPrunesByAge(t *testing.T) {
	collector, roots := newSQLiteCollector(t)
	ctx := coredb.WithTestMode(context.Background())
	runID := uuid.New()
	stored, err := collector.Store(ctx, artifact.StoreRequest{RunID: runID, Type: artifact.ArtifactTypeLog, Name: "output.log", Content: bytes.NewBufferString("artifact contents"), ContentType: "text/plain"})
	if err != nil {
		t.Fatalf("Store: %v", err)
	}
	if got := roots.LeaseStats(); got.TestRootWrites != 1 || got.PrimaryWritesDuringTestMode != 0 {
		t.Fatalf("write stats = %+v", got)
	}
	read, err := collector.Read(ctx, stored.ID)
	if err != nil {
		t.Fatalf("Read: %v", err)
	}
	_ = read.Close()
	if _, err := collector.DeleteBefore(ctx, time.Now().Add(time.Hour), 1); err != nil {
		t.Fatalf("DeleteBefore: %v", err)
	}
	if artifacts, err := collector.List(ctx, runID, artifact.ListOptions{}); err != nil || len(artifacts) != 0 {
		t.Fatalf("artifacts after retention = %+v, %v", artifacts, err)
	}
}
