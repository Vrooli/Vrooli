package devrouting

import (
	"context"
	"database/sql"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"connectrpc.com/connect"
	"github.com/vrooli/api-core/database"
	"github.com/vrooli/api-core/filerouting"
	"github.com/vrooli/api-core/storage"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/dev-routing/v1/routing"
	_ "modernc.org/sqlite"
)

func TestRestartCheckpointReattachesExactDatabaseAndFileTree(t *testing.T) {
	cache := t.TempDir()
	t.Setenv("XDG_CACHE_HOME", cache)
	primaryDB := filepath.Join(t.TempDir(), "primary.db")
	testDB := filepath.Join(t.TempDir(), "test.db")
	primaryRoots := makePrimaryRoots(t)
	db1 := openRestartDB(t, primaryDB)
	roots1 := filerouting.New(primaryRoots)
	service1 := NewService(db1, roots1)
	leaseID := "restart-paired-lease"
	if _, err := service1.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: testDB, LeaseId: leaseID})); err != nil {
		t.Fatalf("install: %v", err)
	}
	testCtx := database.WithTestMode(context.Background())
	if _, err := db1.ExecContext(testCtx, "CREATE TABLE persisted (value TEXT)"); err != nil {
		t.Fatalf("create test table: %v", err)
	}
	if _, err := db1.ExecContext(testCtx, "INSERT INTO persisted(value) VALUES (?)", "survives restart"); err != nil {
		t.Fatalf("insert test row: %v", err)
	}
	dataRoot, err := roots1.PickRequired(testCtx, storage.ClassData)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dataRoot, "payload.bin"), []byte("retained bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	roots1.RecordWrite(testCtx)

	prepared, err := service1.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatalf("prepare: %v", err)
	}
	ticket := prepared.Msg.GetRestartTicket()
	if len(ticket) != 64 || prepared.Msg.GetPreRestartStats().GetTestPoolRequests() == 0 || prepared.Msg.GetPreRestartStats().GetTestRootWrites() != 1 {
		t.Fatalf("incomplete checkpoint receipt: %+v", prepared.Msg)
	}
	encoded, _ := json.Marshal(prepared.Msg)
	if strings.Contains(string(encoded), testDB) || strings.Contains(string(encoded), dataRoot) {
		t.Fatal("public checkpoint response exposed a private binding")
	}
	path, err := checkpointPathForService(service1, leaseID)
	if err != nil {
		t.Fatal(err)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil || dirInfo.Mode().Perm() != 0o700 {
		t.Fatalf("checkpoint namespace mode = %v, err=%v; want 0700", dirInfo, err)
	}
	fileInfo, err := os.Stat(path)
	if err != nil || fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("checkpoint record mode = %v, err=%v; want 0600", fileInfo, err)
	}
	if _, err := service1.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); err == nil {
		t.Fatal("preparing service instance resumed its own active lease")
	}
	if err := db1.Close(); err != nil {
		t.Fatalf("close old owner database: %v", err)
	}

	// A fresh service object models the new target process after the harness has
	// independently verified that the former process exited.
	db2 := openRestartDB(t, primaryDB)
	roots2 := filerouting.New(primaryRoots)
	service2 := NewService(db2, roots2)
	if _, err := service2.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "replacement.db"), LeaseId: leaseID})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("ordinary same-ID install bypassed restart checkpoint: %v", err)
	}
	resumed, err := service2.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket}))
	if err != nil {
		t.Fatalf("resume: %v", err)
	}
	if resumed.Msg.GetActiveLeaseId() != leaseID || !resumed.Msg.GetFileRootsResumed() || resumed.Msg.GetPreRestartStats().GetTestRootWrites() != 1 {
		t.Fatalf("unexpected resume receipt: %+v", resumed.Msg)
	}
	var persisted string
	if err := db2.QueryRowContext(testCtx, "SELECT value FROM persisted").Scan(&persisted); err != nil {
		t.Fatalf("persisted database row missing: %v", err)
	}
	if persisted != "survives restart" {
		t.Fatalf("persisted database value = %q", persisted)
	}
	retainedRoot, err := roots2.PickRequired(testCtx, storage.ClassData)
	if err != nil {
		t.Fatal(err)
	}
	if got, err := os.ReadFile(filepath.Join(retainedRoot, "payload.bin")); err != nil || string(got) != "retained bytes" {
		t.Fatalf("retained file = %q, err=%v", got, err)
	}
	// A lost Resume response can be recovered only by this same live owner.
	retried, err := service2.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket}))
	if err != nil || retried.Msg.GetActiveLeaseId() != leaseID {
		t.Fatalf("same-owner resume retry failed: response=%v err=%v", retried, err)
	}
	cleared, err := service2.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket}))
	if err != nil {
		t.Fatalf("clear resumed binding: %v", err)
	}
	if cleared.Msg.GetStats().GetPrimaryDuringTestModeRequests() != 0 || cleared.Msg.GetStats().GetPrimaryRootWritesDuringTestMode() != 0 {
		t.Fatalf("primary routing counters were nonzero: %+v", cleared.Msg.GetStats())
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("checkpoint remains after clear: %v", err)
	}
	if _, err := service2.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); err == nil {
		t.Fatal("cleared checkpoint was replayed")
	}
}

func TestRestartResumePinsRelativeSQLiteURIToOriginalDirectory(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = os.Chdir(originalDir) }()
	dirA, dirB := t.TempDir(), t.TempDir()
	primaryRoots := makePrimaryRoots(t)
	primaryDB := filepath.Join(t.TempDir(), "primary.db")
	leaseID := "restart-relative-sqlite"
	if err := os.Chdir(dirA); err != nil {
		t.Fatal(err)
	}
	db1 := openRestartDB(t, primaryDB)
	service1 := NewService(db1, filerouting.New(primaryRoots))
	if _, err := service1.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: "file:lease.db", LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	if _, err := db1.ExecContext(database.WithTestMode(context.Background()), "CREATE TABLE relative_binding (value TEXT)"); err != nil {
		t.Fatal(err)
	}
	if _, err := db1.ExecContext(database.WithTestMode(context.Background()), "INSERT INTO relative_binding(value) VALUES ('original')"); err != nil {
		t.Fatal(err)
	}
	prepared, err := service1.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatal(err)
	}
	ticket := prepared.Msg.GetRestartTicket()
	if err := db1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dirB); err != nil {
		t.Fatal(err)
	}
	other, err := os.OpenFile("lease.db", os.O_CREATE|os.O_WRONLY, 0o600)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = other.WriteString("not the leased database")
	_ = other.Close()
	db2 := openRestartDB(t, primaryDB)
	service2 := NewService(db2, filerouting.New(primaryRoots))
	if _, err := service2.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); err != nil {
		t.Fatalf("resume from changed working directory: %v", err)
	}
	var value string
	if err := db2.QueryRowContext(database.WithTestMode(context.Background()), "SELECT value FROM relative_binding").Scan(&value); err != nil || value != "original" {
		t.Fatalf("reopened relative SQLite binding = %q, err=%v", value, err)
	}
	if _, err := service2.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); err != nil {
		t.Fatalf("clear resumed relative binding: %v", err)
	}
}

func TestRestartCheckpointRejectsWrongTargetAndChangedTree(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	leaseID := "restart-target-lease"
	primaryDB := filepath.Join(t.TempDir(), "primary.db")
	db1 := openRestartDB(t, primaryDB)
	primaryRoots := makePrimaryRoots(t)
	roots1 := filerouting.New(primaryRoots)
	service1 := NewService(db1, roots1)
	if _, err := service1.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "test.db"), LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	prepared, err := service1.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatal(err)
	}
	ticket := prepared.Msg.GetRestartTicket()
	if err := db1.Close(); err != nil {
		t.Fatal(err)
	}
	db2 := openRestartDB(t, filepath.Join(t.TempDir(), "different-primary.db"))
	roots2 := filerouting.New(makePrimaryRoots(t))
	service2 := NewService(db2, roots2)
	if _, err := service2.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("wrong target resume code = %v, want FailedPrecondition", connect.CodeOf(err))
	}

	// Reopen the exact expected target so this case reaches marker validation.
	db3 := openRestartDB(t, primaryDB)
	var checkpoint restartCheckpoint
	path, _ := checkpointPathForService(service1, leaseID)
	if err := readCheckpoint(path, ticket, &checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(checkpoint.Roots.LeaseRoot, rootMarkerName)); err != nil {
		t.Fatal(err)
	}
	service3 := NewService(db3, filerouting.New(primaryRoots))
	_, markerErr := service3.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket}))
	if connect.CodeOf(markerErr) != connect.CodeFailedPrecondition || !strings.Contains(markerErr.Error(), "retained leased root does not match checkpoint") {
		t.Fatalf("changed marker resume error = %v, want marker-specific FailedPrecondition", markerErr)
	}
	if _, err := service3.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unattached clear code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("failed resume/clear discarded its retry record: %v", err)
	}
	if _, err := os.Stat(checkpoint.Roots.LeaseRoot); err != nil {
		t.Fatalf("failed resume/clear orphaned its retained data: %v", err)
	}
	if err := writeRootMarker(checkpoint.Roots.LeaseRoot, rootMarker{TokenHash: checkpoint.TokenHash, LeaseID: leaseID, TargetFingerprint: checkpoint.TargetFingerprint}); err != nil {
		t.Fatal(err)
	}
	if _, err := service3.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); err != nil {
		t.Fatalf("resume after restoring exact marker: %v", err)
	}
	if _, err := service3.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); err != nil {
		t.Fatalf("clear restored checkpoint: %v", err)
	}
}

func TestRestartTicketAllowsOnlyOneFreshOwnerToResume(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	leaseID := "restart-one-owner"
	primaryDB, testDB := filepath.Join(t.TempDir(), "primary.db"), filepath.Join(t.TempDir(), "test.db")
	primaryRoots := makePrimaryRoots(t)
	oldDB := openRestartDB(t, primaryDB)
	old := NewService(oldDB, filerouting.New(primaryRoots))
	if _, err := old.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: testDB, LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	prepared, err := old.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatal(err)
	}
	if err := oldDB.Close(); err != nil {
		t.Fatal(err)
	}
	ticket := prepared.Msg.GetRestartTicket()
	services := []*Service{
		NewService(openRestartDB(t, primaryDB), filerouting.New(primaryRoots)),
		NewService(openRestartDB(t, primaryDB), filerouting.New(primaryRoots)),
	}
	type result struct {
		service *Service
		err     error
	}
	results := make(chan result, 2)
	for _, service := range services {
		go func(service *Service) {
			_, err := service.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket}))
			results <- result{service: service, err: err}
		}(service)
	}
	first, second := <-results, <-results
	if (first.err == nil) == (second.err == nil) {
		t.Fatalf("expected exactly one resume winner: first=%v second=%v", first.err, second.err)
	}
	winner := first.service
	if first.err != nil {
		winner = second.service
	}
	if _, err := winner.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); err != nil {
		t.Fatalf("clear winning resumed binding: %v", err)
	}
}

func TestRestartClearRefusesDifferentSameIDDatabaseBinding(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	leaseID := "restart-clear-binding"
	primaryDB := filepath.Join(t.TempDir(), "primary.db")
	primaryRoots := makePrimaryRoots(t)
	db := openRestartDB(t, primaryDB)
	service := NewService(db, filerouting.New(primaryRoots))
	originalTestDB, replacementDB := filepath.Join(t.TempDir(), "original.db"), filepath.Join(t.TempDir(), "replacement.db")
	if _, err := service.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: originalTestDB, LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	prepared, err := service.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatal(err)
	}
	// Simulate an owner-level same-ID replacement outside this service. The
	// checkpoint ticket must not authorize clearing the new pool generation.
	if err := db.InstallTestPool(context.Background(), replacementDB, leaseID, time.Minute); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: prepared.Msg.GetRestartTicket()})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("clear different same-ID binding code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	var value string
	if err := db.QueryRowContext(database.WithTestMode(context.Background()), "SELECT 1").Scan(&value); err != nil {
		t.Fatalf("replacement pool was cleared: %v", err)
	}
	path, _ := checkpointPathForService(service, leaseID)
	_ = db.ClearTestPool(leaseID)
	_ = service.roots.ClearTestRoots(leaseID)
	_ = clearCheckpoint(path)
}

func TestRestartMissingSQLiteFileIsNotRecreatedAndClearRetainsCheckpoint(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	leaseID := "restart-missing-db"
	primaryDB, testDB := filepath.Join(t.TempDir(), "primary.db"), filepath.Join(t.TempDir(), "test.db")
	primaryRoots := makePrimaryRoots(t)
	db1 := openRestartDB(t, primaryDB)
	service1 := NewService(db1, filerouting.New(primaryRoots))
	if _, err := service1.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: testDB, LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	prepared, err := service1.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatal(err)
	}
	path, err := checkpointPathForService(service1, leaseID)
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint restartCheckpoint
	if err := readCheckpoint(path, prepared.Msg.GetRestartTicket(), &checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := db1.Close(); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(testDB); err != nil {
		t.Fatal(err)
	}
	newService := NewService(openRestartDB(t, primaryDB), filerouting.New(primaryRoots))
	if _, err := newService.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: prepared.Msg.GetRestartTicket()})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("resume missing database code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	if _, err := os.Stat(testDB); !os.IsNotExist(err) {
		t.Fatalf("missing database was recreated: %v", err)
	}
	if _, err := newService.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: prepared.Msg.GetRestartTicket()})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("unattached cleanup code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("checkpoint lost after failed resume: %v", err)
	}
	if _, err := os.Stat(checkpoint.Roots.LeaseRoot); err != nil {
		t.Fatalf("retained root lost after failed resume: %v", err)
	}
	_ = clearCheckpoint(path)
	_ = os.RemoveAll(checkpoint.Roots.LeaseRoot)
}

func TestRestartClearRetryDeletesRecordAfterBothRoutesDetached(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	leaseID := "restart-clear-retry"
	db := openRestartDB(t, filepath.Join(t.TempDir(), "primary.db"))
	service := NewService(db, filerouting.New(makePrimaryRoots(t)))
	if _, err := service.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "test.db"), LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	prepared, err := service.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatal(err)
	}
	ticket := prepared.Msg.GetRestartTicket()
	path, err := checkpointPathForService(service, leaseID)
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint restartCheckpoint
	if err := readCheckpoint(path, ticket, &checkpoint); err != nil {
		t.Fatal(err)
	}
	checkpoint.Phase = "clearing"
	if err := writeCheckpoint(path, checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := db.ClearTestPool(leaseID); err != nil {
		t.Fatal(err)
	}
	if err := service.roots.ClearTestRoots(leaseID); err != nil {
		t.Fatal(err)
	}
	if _, err := service.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); err != nil {
		t.Fatalf("clear retry after route detach: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("checkpoint remained after retry: %v", err)
	}
}

func TestRestartCheckpointExpiryAndTicketValidation(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	leaseID := "restart-expiry-lease"
	db := openRestartDB(t, filepath.Join(t.TempDir(), "primary.db"))
	service := NewService(db, filerouting.New(makePrimaryRoots(t)))
	if _, err := service.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "test.db"), LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	prepared, err := service.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatal(err)
	}
	ticket := prepared.Msg.GetRestartTicket()
	if _, err := service.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: "../escape"})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("malformed ticket code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	path, _ := checkpointPathForService(service, leaseID)
	var checkpoint restartCheckpoint
	if err := readCheckpoint(path, ticket, &checkpoint); err != nil {
		t.Fatal(err)
	}
	checkpoint.ExpiresAt = time.Now().Add(-time.Second)
	checkpoint.Database.ExpiresAt = checkpoint.ExpiresAt
	checkpoint.Roots.ExpiresAt = checkpoint.ExpiresAt
	if err := writeCheckpoint(path, checkpoint); err != nil {
		t.Fatal(err)
	}
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}
	newDB := openRestartDB(t, filepath.Join(t.TempDir(), "primary.db"))
	newSvc := NewService(newDB, filerouting.New(makePrimaryRoots(t)))
	if _, err := newSvc.ResumeTestPool(context.Background(), connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: ticket})); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expired checkpoint code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	_ = service.roots.ClearTestRoots(leaseID)
	_ = clearCheckpoint(path)
}

func TestExpiredPreparedCheckpointCanBeClearedAfterOldOwnerExit(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	leaseID := "restart-expired-prepared-clear"
	primaryDB, testDB := filepath.Join(t.TempDir(), "primary.db"), filepath.Join(t.TempDir(), "test.db")
	primaryRoots := makePrimaryRoots(t)
	if err := os.WriteFile(filepath.Join(primaryRoots.DataDir, "sentinel"), []byte("primary"), 0o600); err != nil {
		t.Fatal(err)
	}
	oldDB := openRestartDB(t, primaryDB)
	oldRoots := filerouting.New(primaryRoots)
	old := NewService(oldDB, oldRoots)
	if _, err := old.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: testDB, LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	prepared, err := old.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if err != nil {
		t.Fatal(err)
	}
	path, err := checkpointPathForService(old, leaseID)
	if err != nil {
		t.Fatal(err)
	}
	var checkpoint restartCheckpoint
	if err := readCheckpoint(path, prepared.Msg.GetRestartTicket(), &checkpoint); err != nil {
		t.Fatal(err)
	}
	expired := time.Now().Add(-time.Second)
	checkpoint.ExpiresAt, checkpoint.Database.ExpiresAt, checkpoint.Roots.ExpiresAt = expired, expired, expired
	if err := writeCheckpoint(path, checkpoint); err != nil {
		t.Fatal(err)
	}
	retainedRoot := checkpoint.Roots.LeaseRoot
	clearRequest := connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID, RestartTicket: prepared.Msg.GetRestartTicket()})
	if _, err := old.ClearTestPool(context.Background(), clearRequest); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expired checkpoint on a still-live prepared owner code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	liveDB := openRestartDB(t, primaryDB)
	liveOwnerProbe := NewService(liveDB, filerouting.New(primaryRoots))
	if _, err := liveOwnerProbe.ClearTestPool(context.Background(), clearRequest); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("fresh service cleared checkpoint while old owner lock was held: %v", err)
	}
	if _, err := os.Stat(retainedRoot); err != nil {
		t.Fatalf("refused live-owner cleanup changed retained root: %v", err)
	}
	_ = liveDB.Close()
	if err := oldDB.Close(); err != nil {
		t.Fatal(err)
	}
	// A dead process releases its kernel lock while the sidecar remains. This
	// models that OS behavior without deleting the proof file.
	old.ownerLocks[leaseID].release()
	delete(old.ownerLocks, leaseID)
	newDB := openRestartDB(t, primaryDB)
	newRoots := filerouting.New(primaryRoots)
	newService := NewService(newDB, newRoots)
	request := connect.NewRequest(&routingv1.ResumeTestPoolRequest{LeaseId: leaseID, RestartTicket: prepared.Msg.GetRestartTicket()})
	if _, err := newService.ResumeTestPool(context.Background(), request); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("expired resume code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	markerPath := filepath.Join(retainedRoot, rootMarkerName)
	if err := os.Remove(markerPath); err != nil {
		t.Fatal(err)
	}
	if err := writeRootMarker(retainedRoot, rootMarker{TokenHash: "mismatch", LeaseID: leaseID, TargetFingerprint: checkpoint.TargetFingerprint}); err != nil {
		t.Fatal(err)
	}
	if _, err := newService.ClearTestPool(context.Background(), clearRequest); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("changed marker clear code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	if err := os.Remove(markerPath); err != nil {
		t.Fatal(err)
	}
	if err := writeRootMarker(retainedRoot, rootMarker{TokenHash: checkpoint.TokenHash, LeaseID: leaseID, TargetFingerprint: checkpoint.TargetFingerprint}); err != nil {
		t.Fatal(err)
	}
	otherLease := "restart-expired-prepared-other-lease"
	if _, err := newService.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "other.db"), LeaseId: otherLease})); err != nil {
		t.Fatal(err)
	}
	if _, err := newService.ClearTestPool(context.Background(), clearRequest); connect.CodeOf(err) != connect.CodeFailedPrecondition {
		t.Fatalf("other active lease clear code = %v, want FailedPrecondition", connect.CodeOf(err))
	}
	if newDB.ActiveLeaseID() != otherLease || newRoots.ActiveLeaseID() != otherLease {
		t.Fatal("refused cleanup changed the other active lease")
	}
	if _, err := newService.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: otherLease})); err != nil {
		t.Fatal(err)
	}
	if _, err := newService.ClearTestPool(context.Background(), clearRequest); err != nil {
		t.Fatalf("verified expired prepared checkpoint should clear after old owner exit: %v", err)
	}
	if _, err := os.Lstat(retainedRoot); !os.IsNotExist(err) {
		t.Fatalf("retained test root remains after clear: %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("checkpoint remains after clear: %v", err)
	}
	var value int
	if err := newDB.QueryRowContext(context.Background(), "SELECT 1").Scan(&value); err != nil || value != 1 {
		t.Fatalf("primary DB route changed during cleanup: value=%d err=%v", value, err)
	}
	if content, err := os.ReadFile(filepath.Join(primaryRoots.DataDir, "sentinel")); err != nil || string(content) != "primary" {
		t.Fatalf("primary file route changed during cleanup: content=%q err=%v", content, err)
	}
}

func TestPrepareRejectsUnavailablePrimaryFingerprint(t *testing.T) {
	t.Setenv("XDG_CACHE_HOME", t.TempDir())
	primaryPath := filepath.Join(t.TempDir(), "unresolved-primary.db")
	db, err := database.Open(context.Background(), database.Config{
		Driver: database.DriverSQLite,
		DSN:    primaryPath,
		Opener: func(driver, dsn string) (*sql.DB, error) {
			if dsn == primaryPath {
				return sql.Open(driver, ":memory:")
			}
			return sql.Open(driver, dsn)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if db.PrimaryFingerprint() != "" {
		t.Fatal("test setup expected an unavailable resolved primary file identity")
	}
	leaseID := "restart-no-primary-identity"
	service := NewService(db, filerouting.New(makePrimaryRoots(t)))
	if _, err := service.InstallTestPool(context.Background(), connect.NewRequest(&routingv1.InstallTestPoolRequest{Dsn: filepath.Join(t.TempDir(), "test.db"), LeaseId: leaseID})); err != nil {
		t.Fatal(err)
	}
	_, err = service.PrepareTestPoolRestart(context.Background(), connect.NewRequest(&routingv1.PrepareTestPoolRestartRequest{LeaseId: leaseID}))
	if connect.CodeOf(err) != connect.CodeFailedPrecondition || !strings.Contains(err.Error(), "restart target identity is unavailable") {
		t.Fatalf("prepare without a primary fingerprint = %v, want target-identity FailedPrecondition", err)
	}
	if _, err := service.ClearTestPool(context.Background(), connect.NewRequest(&routingv1.ClearTestPoolRequest{LeaseId: leaseID})); err != nil {
		t.Fatalf("clear test routing after rejected prepare: %v", err)
	}
}

func openRestartDB(t *testing.T, dsn string) *database.RoutedDB {
	t.Helper()
	db, err := database.Open(context.Background(), database.Config{Driver: database.DriverSQLite, DSN: dsn})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })
	return db
}

func makePrimaryRoots(t *testing.T) storage.Paths {
	t.Helper()
	base := t.TempDir()
	roots := storage.Paths{ConfigDir: filepath.Join(base, "config"), DataDir: filepath.Join(base, "data"), CacheDir: filepath.Join(base, "cache"), LogsDir: filepath.Join(base, "logs"), StateDir: filepath.Join(base, "state")}
	for _, path := range []string{roots.ConfigDir, roots.DataDir, roots.CacheDir, roots.LogsDir, roots.StateDir} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	return roots
}
