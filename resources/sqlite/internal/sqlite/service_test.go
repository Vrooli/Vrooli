package sqlite

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/vrooli/resources/sqlite/internal/config"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	ctx := context.Background()

	if _, err := svc.CreateDatabase(ctx, "test"); err != nil {
		t.Fatalf("CreateDatabase: %v", err)
	}
	dbPath, _ := svc.databasePath("test")

	if svc.IsEncrypted(dbPath) {
		t.Fatalf("fresh database incorrectly reported as encrypted")
	}

	if err := svc.EncryptDatabase("test", "password"); err != nil {
		t.Fatalf("EncryptDatabase: %v", err)
	}
	if !svc.IsEncrypted(dbPath) {
		t.Fatalf("encrypted database not detected")
	}

	if err := svc.DecryptDatabase("test", "password"); err != nil {
		t.Fatalf("DecryptDatabase: %v", err)
	}
	if svc.IsEncrypted(dbPath) {
		t.Fatalf("database still marked encrypted after decrypt")
	}

	if _, err := svc.Execute(ctx, "test", "PRAGMA integrity_check;"); err != nil {
		t.Fatalf("post-decrypt integrity check failed: %v", err)
	}
}

func TestIsEncryptedDetectionForPrefixedFile(t *testing.T) {
	cfg := testConfig(t)
	svc := NewService(cfg)
	path := filepath.Join(cfg.DatabasePath, "custom.db")
	if err := os.MkdirAll(cfg.DatabasePath, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(encMagic+"12345"), cfg.FilePermissions); err != nil {
		t.Fatalf("write: %v", err)
	}
	if !svc.IsEncrypted(path) {
		t.Fatalf("encMagic-prefixed file should be treated as encrypted")
	}
}

func testConfig(t *testing.T) config.Config {
	t.Helper()
	root := t.TempDir()
	return config.Config{
		DataRoot:            root,
		DatabasePath:        filepath.Join(root, "databases"),
		BackupPath:          filepath.Join(root, "backups"),
		ReplicaPath:         filepath.Join(root, "replicas"),
		JournalMode:         "WAL",
		BusyTimeout:         10 * time.Second,
		CacheSize:           2000,
		PageSize:            4096,
		Synchronous:         "NORMAL",
		TempStore:           "MEMORY",
		MmapSize:            268435456,
		FilePermissions:     0o600,
		CLITimeout:          5 * time.Second,
		BackupRetentionDays: 7,
	}
}
