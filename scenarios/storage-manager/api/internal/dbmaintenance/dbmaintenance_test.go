package dbmaintenance

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func bloatedDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "storage-manager.db"))
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	ctx := context.Background()
	for _, stmt := range []string{"PRAGMA journal_mode=WAL", "CREATE TABLE census_snapshots (id INTEGER PRIMARY KEY, report_json TEXT)"} {
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			t.Fatal(err)
		}
	}
	churn(t, db)
	return db
}

// churn writes and deletes ~8 MiB, leaving the pages on the freelist.
func churn(t *testing.T, db *sql.DB) {
	t.Helper()
	ctx := context.Background()
	blob := strings.Repeat("x", 256<<10)
	for i := 0; i < 32; i++ {
		if _, err := db.ExecContext(ctx, "INSERT INTO census_snapshots (report_json) VALUES (?)", blob); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := db.ExecContext(ctx, "DELETE FROM census_snapshots"); err != nil {
		t.Fatal(err)
	}
}

func testOptions(free int64, allow bool) Options {
	return Options{
		Policy:          Policy{FreeTriggerBytes: 1 << 20, FreeFloorBytes: 0, BatchPages: 64, TickBudget: 10 * time.Second, SpaceMarginBytes: 1 << 20},
		AllowConversion: allow,
		FreeSpace:       func(string) (int64, error) { return free, nil },
	}
}

func TestMaintainReturnsFreedPagesOnceConvertedAndOnEveryLaterPass(t *testing.T) {
	db := bloatedDatabase(t)
	ctx := context.Background()

	deferred, err := Maintain(ctx, db, testOptions(1<<40, false))
	if err != nil || deferred.Converted || deferred.After.AutoVacuum != "none" || deferred.Deferred == "" {
		t.Fatalf("conversion ran while retention still had queued work: %#v %v", deferred, err)
	}

	converted, err := Maintain(ctx, db, testOptions(1<<40, true))
	if err != nil {
		t.Fatal(err)
	}
	if !converted.Converted || converted.After.AutoVacuum != "incremental" || converted.After.FreePages != 0 || converted.After.PageCount >= converted.Before.PageCount {
		t.Fatalf("conversion did not return freed pages: %#v", converted)
	}

	// Later retention deletes are returned online, without another rewrite.
	churn(t, db)
	drained, err := Maintain(ctx, db, testOptions(1<<40, true))
	if err != nil {
		t.Fatal(err)
	}
	if drained.Converted || drained.ReturnedPages == 0 || drained.After.FreePages != 0 || drained.After.PageCount >= drained.Before.PageCount {
		t.Fatalf("incremental pass did not shrink the file: %#v", drained)
	}
}

func TestMaintainRefusesConversionWithoutRoomForTheTransientCopy(t *testing.T) {
	db := bloatedDatabase(t)
	result, err := Maintain(context.Background(), db, testOptions(1<<10, true))
	if !errors.Is(err, ErrInsufficientSpace) || result.Converted {
		t.Fatalf("conversion attempted on a full disk: %#v %v", result, err)
	}
	after, err := Measure(context.Background(), db)
	if err != nil || after.AutoVacuum != "none" {
		t.Fatalf("auto_vacuum changed after a refused conversion: %#v %v", after, err)
	}
}
