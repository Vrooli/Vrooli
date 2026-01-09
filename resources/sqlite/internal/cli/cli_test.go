package cli

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/resources/sqlite/internal/config"
	"github.com/vrooli/resources/sqlite/internal/sqlite"
)

func TestHandleStatsGracefulWhenDbstatMissing(t *testing.T) {
	cfg := config.Config{CLITimeout: time.Second}
	cli := &CLI{
		Service: &stubService{showStatsErr: sqlite.ErrDbstatUnavailable},
		Config:  cfg,
	}

	code, out := runWithCapture(func() int {
		return cli.handleStats([]string{"show", "db"})
	})

	if code != 0 {
		t.Fatalf("expected exit code 0, got %d (output: %s)", code, out)
	}
	if !containsString(out, "dbstat extension not available") {
		t.Fatalf("expected graceful message, got: %s", out)
	}
}

func TestManageInstallInvokesRebuilderAndCreatesDirs(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		DataRoot:      root,
		DatabasePath:  filepath.Join(root, "db"),
		BackupPath:    filepath.Join(root, "backups"),
		ReplicaPath:   filepath.Join(root, "replicas"),
		MigrationPath: filepath.Join(root, "migrations"),
		CLITimeout:    time.Second,
	}
	stub := &stubService{}
	rebuilt := false
	cli := &CLI{
		Service:   stub,
		Config:    cfg,
		Rebuilder: func() error { rebuilt = true; return nil },
	}

	code, _ := runWithCapture(func() int {
		return cli.handleManage([]string{"install"})
	})

	if code != 0 {
		t.Fatalf("expected success code, got %d", code)
	}
	if !rebuilt {
		t.Fatalf("rebuilder was not invoked")
	}
	for _, dir := range []string{cfg.DatabasePath, cfg.BackupPath, cfg.ReplicaPath, cfg.MigrationPath} {
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("expected dir created: %s (err %v)", dir, err)
		}
	}
}

func TestManageInstallWarnsButSucceedsWhenRebuilderFails(t *testing.T) {
	root := t.TempDir()
	cfg := config.Config{
		DataRoot:      root,
		DatabasePath:  filepath.Join(root, "db"),
		BackupPath:    filepath.Join(root, "backups"),
		ReplicaPath:   filepath.Join(root, "replicas"),
		MigrationPath: filepath.Join(root, "migrations"),
		CLITimeout:    time.Second,
	}
	cli := &CLI{
		Service:   &stubService{},
		Config:    cfg,
		Rebuilder: func() error { return fmt.Errorf("boom") },
	}

	code, out := runWithCapture(func() int {
		return cli.handleManage([]string{"install"})
	})

	if code != 0 {
		t.Fatalf("expected success code, got %d", code)
	}
	if !containsString(out, "Warning:") {
		t.Fatalf("expected warning in output, got: %s", out)
	}
}

func TestReplicateAddParsesFlags(t *testing.T) {
	stub := &stubService{}
	cli := &CLI{
		Service: stub,
		Config:  config.Config{CLITimeout: time.Second},
	}

	code, _ := runWithCapture(func() int {
		return cli.handleReplicate([]string{"add", "--database", "db1", "--target", "/tmp/rep.db", "--interval", "120"})
	})

	if code != 0 {
		t.Fatalf("expected success, got %d", code)
	}
	if stub.addDatabase != "db1" || stub.addTarget != "/tmp/rep.db" || stub.addInterval != 120*time.Second {
		t.Fatalf("unexpected add args: %+v", stub)
	}
}

func TestReplicateSyncAllReportsFailures(t *testing.T) {
	stub := &stubService{syncAllResult: [3]int{1, 1, 0}}
	cli := &CLI{
		Service: stub,
		Config:  config.Config{CLITimeout: time.Second},
	}

	code, out := runWithCapture(func() int {
		return cli.handleReplicate([]string{"sync", "--all"})
	})

	if code != 1 {
		t.Fatalf("expected failure exit due to failed replicas, got %d", code)
	}
	if !containsString(out, "1 failed") {
		t.Fatalf("expected failure summary, got %s", out)
	}
}

// Minimal stub satisfying SQLiteService; defaults to no-ops unless overridden.
type stubService struct {
	showStatsErr   error
	addDatabase    string
	addTarget      string
	addInterval    time.Duration
	syncAllResult  [3]int // ok, fail, errFlag (0 ok, 1 error)
	syncSingleCall struct {
		db    string
		force bool
	}
}

func (s *stubService) Status(ctx context.Context) (*sqlite.StatusInfo, error) {
	return &sqlite.StatusInfo{}, nil
}
func (s *stubService) CreateDatabase(ctx context.Context, name string) (string, error) {
	return "", nil
}
func (s *stubService) Execute(ctx context.Context, name, query string) (*sqlite.QueryResult, error) {
	return nil, nil
}
func (s *stubService) ListDatabases() ([]sqlite.DBInfo, error) { return nil, nil }
func (s *stubService) GetDatabaseInfo(ctx context.Context, name, query string) (*sqlite.GetResult, error) {
	return nil, nil
}
func (s *stubService) BackupDatabase(ctx context.Context, name string) (string, error) {
	return "", nil
}
func (s *stubService) RestoreDatabase(ctx context.Context, name, backup string, force bool) (string, error) {
	return "", nil
}
func (s *stubService) RemoveDatabase(name string, force bool) error { return nil }
func (s *stubService) Batch(ctx context.Context, name string, reader io.Reader) (*sqlite.BatchResult, error) {
	return nil, nil
}
func (s *stubService) ImportCSV(ctx context.Context, dbName, table, csvPath string, hasHeader bool, columnOverride []string) (int, error) {
	return 0, nil
}
func (s *stubService) ExportCSV(ctx context.Context, dbName, table, outputPath string) (int, error) {
	return 0, nil
}
func (s *stubService) EncryptDatabase(dbName, password string) error { return nil }
func (s *stubService) DecryptDatabase(dbName, password string) error { return nil }
func (s *stubService) AddReplica(dbName, target string, interval time.Duration) error {
	s.addDatabase = dbName
	s.addTarget = target
	s.addInterval = interval
	return nil
}
func (s *stubService) RemoveReplica(dbName, target string) error { return nil }
func (s *stubService) ListReplicas() ([]sqlite.Replica, error)   { return nil, nil }
func (s *stubService) SyncReplica(ctx context.Context, dbName string, force bool) (int, int, error) {
	s.syncSingleCall.db = dbName
	s.syncSingleCall.force = force
	return 0, 0, nil
}
func (s *stubService) SyncDueReplicas(ctx context.Context, force bool) (int, int, error) {
	if s.syncAllResult[2] != 0 {
		return 0, 0, fmt.Errorf("sync error")
	}
	return s.syncAllResult[0], s.syncAllResult[1], nil
}
func (s *stubService) VerifyReplicas(ctx context.Context, dbName string) ([]string, error) {
	return nil, nil
}
func (s *stubService) ToggleReplica(dbName, target string, enabled bool) error { return nil }
func (s *stubService) InitMigrations(name string) error                        { return nil }
func (s *stubService) CreateMigration(name string) (string, error)             { return "", nil }
func (s *stubService) ApplyMigrations(ctx context.Context, dbName string, targetVersion string) (int, error) {
	return 0, nil
}
func (s *stubService) MigrationStatus(ctx context.Context, dbName string) (applied []string, pending []string, err error) {
	return nil, nil, nil
}
func (s *stubService) QuerySelect(ctx context.Context, dbName, table string, where string, order string, limit int) (*sqlite.QueryResult, error) {
	return nil, nil
}
func (s *stubService) QueryInsert(ctx context.Context, dbName, table string, pairs []string) error {
	return nil
}
func (s *stubService) QueryUpdate(ctx context.Context, dbName, table string, pairs []string, where string) error {
	return nil
}
func (s *stubService) EnableStats(ctx context.Context, dbName string) error { return nil }
func (s *stubService) ShowStats(ctx context.Context, dbName string) (*sqlite.QueryResult, error) {
	return nil, s.showStatsErr
}
func (s *stubService) Analyze(ctx context.Context, dbName string) error { return nil }
func (s *stubService) Vacuum(ctx context.Context, dbName string) error  { return nil }
func runWithCapture(fn func() int) (int, string) {
	origOut, origErr := os.Stdout, os.Stderr
	rOut, wOut, _ := os.Pipe()
	rErr, wErr, _ := os.Pipe()
	os.Stdout, os.Stderr = wOut, wErr

	code := fn()

	_ = wOut.Close()
	_ = wErr.Close()
	os.Stdout, os.Stderr = origOut, origErr

	var buf bytes.Buffer
	_, _ = buf.ReadFrom(rOut)
	_, _ = buf.ReadFrom(rErr)
	return code, buf.String()
}

func containsString(haystack, needle string) bool {
	return bytes.Contains([]byte(haystack), []byte(needle))
}
