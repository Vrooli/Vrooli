package persistence

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"test-genie/internal/storage/sqlitedb"

	"github.com/vrooli/api-core/database"
	_ "modernc.org/sqlite"
)

const DatabaseCutoverConfirmation = "ARCHIVE_TEST_GENIE_EVIDENCE"

var ErrDatabaseCutoverNotConfirmed = errors.New("database cutover is not confirmed")

type DatabaseCutoverPlan struct {
	LivePath    string `json:"livePath"`
	ArchivePath string `json:"archivePath"`
	Bytes       int64  `json:"bytes"`
	Digest      string `json:"digest"`
}
type databaseCutoverReceipt struct {
	ConfirmedAt time.Time           `json:"confirmedAt"`
	Plan        DatabaseCutoverPlan `json:"plan"`
	Integrity   string              `json:"integrity"`
}

// PlanDatabaseCutover inventories one offline SQLite file. Active WAL/SHM
// sidecars are rejected: a cutover must start from a stopped, checkpointed store.
func PlanDatabaseCutover(livePath, archivePath string) (DatabaseCutoverPlan, error) {
	livePath, archivePath = strings.TrimSpace(livePath), strings.TrimSpace(archivePath)
	if livePath == "" || archivePath == "" {
		return DatabaseCutoverPlan{}, fmt.Errorf("live and archive database paths are required")
	}
	var err error
	if livePath, err = filepath.Abs(livePath); err != nil {
		return DatabaseCutoverPlan{}, err
	}
	if archivePath, err = filepath.Abs(archivePath); err != nil {
		return DatabaseCutoverPlan{}, err
	}
	if livePath == archivePath {
		return DatabaseCutoverPlan{}, fmt.Errorf("distinct live and archive database paths are required")
	}
	info, err := os.Stat(livePath)
	if err != nil || !info.Mode().IsRegular() {
		return DatabaseCutoverPlan{}, fmt.Errorf("inspect live database: %w", err)
	}
	for _, sidecar := range []string{livePath + "-wal", livePath + "-shm"} {
		if _, err := os.Stat(sidecar); err == nil {
			return DatabaseCutoverPlan{}, fmt.Errorf("database has active SQLite sidecar %s; stop and checkpoint Test Genie first", sidecar)
		} else if !os.IsNotExist(err) {
			return DatabaseCutoverPlan{}, err
		}
	}
	digest, err := databaseDigest(livePath)
	if err != nil {
		return DatabaseCutoverPlan{}, err
	}
	return DatabaseCutoverPlan{LivePath: livePath, ArchivePath: archivePath, Bytes: info.Size(), Digest: digest}, nil
}

// ApplyDatabaseCutover moves the reviewed legacy store into a rollback archive,
// builds an empty canonical store through ApplySchema, and verifies both files.
func ApplyDatabaseCutover(plan DatabaseCutoverPlan, confirmation string) error {
	if confirmation != DatabaseCutoverConfirmation {
		return ErrDatabaseCutoverNotConfirmed
	}
	current, err := PlanDatabaseCutover(plan.LivePath, plan.ArchivePath)
	if err != nil {
		return err
	}
	if current.Bytes != plan.Bytes || current.Digest != plan.Digest {
		return fmt.Errorf("database inventory changed since review")
	}
	if _, err := os.Stat(plan.ArchivePath); !os.IsNotExist(err) {
		return fmt.Errorf("database archive destination already exists")
	}
	if err := os.MkdirAll(filepath.Dir(plan.ArchivePath), 0o755); err != nil {
		return err
	}
	if err := os.Rename(plan.LivePath, plan.ArchivePath); err != nil {
		return fmt.Errorf("archive database: %w", err)
	}
	rollback := func(cause error) error {
		_ = os.Remove(plan.LivePath)
		if err := os.Rename(plan.ArchivePath, plan.LivePath); err != nil {
			return fmt.Errorf("%w; rollback database rename: %v", cause, err)
		}
		return cause
	}
	if err := verifySQLite(plan.ArchivePath); err != nil {
		return rollback(fmt.Errorf("verify archived database: %w", err))
	}
	if err := createEmptyStore(plan.LivePath); err != nil {
		return rollback(fmt.Errorf("create compact replacement database: %w", err))
	}
	if err := verifySQLite(plan.LivePath); err != nil {
		return rollback(fmt.Errorf("verify replacement database: %w", err))
	}
	return writeJSONAtomic(plan.ArchivePath+".cutover-receipt.json", databaseCutoverReceipt{ConfirmedAt: time.Now().UTC(), Plan: plan, Integrity: "ok"})
}

// RestoreDatabaseCutover is the compensating action used only by a combined
// cutover when a later preflighted filesystem step fails. It refuses to
// overwrite a nonempty replacement and leaves the archive untouched on error.
func RestoreDatabaseCutover(plan DatabaseCutoverPlan) error {
	if err := verifySQLite(plan.LivePath); err != nil {
		return fmt.Errorf("verify replacement before rollback: %w", err)
	}
	if err := os.Remove(plan.LivePath); err != nil {
		return fmt.Errorf("remove replacement database: %w", err)
	}
	if err := os.Rename(plan.ArchivePath, plan.LivePath); err != nil {
		return fmt.Errorf("restore archived database: %w", err)
	}
	_ = os.Remove(plan.ArchivePath + ".cutover-receipt.json")
	return nil
}

func createEmptyStore(path string) error {
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          sqlitedb.BuildDSN(path),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return err
	}
	defer db.Close()
	return ApplySchema(db, false)
}

func verifySQLite(path string) error {
	db, err := database.Connect(context.Background(), database.Config{
		Driver:       database.DriverSQLite,
		DSN:          sqlitedb.BuildDSN(path),
		MaxOpenConns: 1,
		MaxIdleConns: 1,
	})
	if err != nil {
		return err
	}
	defer db.Close()
	var integrity string
	if err := db.QueryRowContext(context.Background(), `PRAGMA integrity_check`).Scan(&integrity); err != nil {
		return err
	}
	if integrity != "ok" {
		return fmt.Errorf("integrity_check = %q", integrity)
	}
	return nil
}

func databaseDigest(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	h := sha256.New()
	if _, err = io.Copy(h, f); err != nil {
		return "", err
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeJSONAtomic(path string, value any) error {
	payload, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".cutover-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if _, err := tmp.Write(append(payload, '\n')); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}
