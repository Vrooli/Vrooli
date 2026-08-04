package graph

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"architecture-cartographer/internal/clock"
)

// TestPayloadCodec_RoundTripIsByteExact is the correctness bar for compression:
// what goes in must come out unchanged, bit for bit.
func TestPayloadCodec_RoundTripIsByteExact(t *testing.T) {
	cases := map[string][]byte{
		"empty":            {},
		"tiny":             []byte("{}"),
		"json object":      []byte(`{"files":[{"path":"a/b.go","symbols":42}],"imports":[]}`),
		"repetitive":       bytes.Repeat([]byte(`{"path":"internal/graph/sqlite.go"},`), 5000),
		"binary-ish bytes": {0x00, 0xff, 0x7f, 0x80, 0x01, 0xfe, 0x00, 0x00},
		"utf8":             []byte(`{"note":"naïve café — 日本語 🎉"}`),
	}

	for name, original := range cases {
		t.Run(name, func(t *testing.T) {
			encoded, codec, err := encodePayload(original)
			if err != nil {
				t.Fatalf("encodePayload: %v", err)
			}
			if codec != codecGzip {
				t.Errorf("codec = %q, want %q", codec, codecGzip)
			}

			decoded, err := decodePayload(encoded, codec)
			if err != nil {
				t.Fatalf("decodePayload: %v", err)
			}
			if !bytes.Equal(decoded, original) {
				t.Errorf("round trip changed the payload: got %d bytes, want %d", len(decoded), len(original))
			}
		})
	}
}

// TestPayloadCodec_CompressesRepetitiveJSON documents the measured benefit.
//
// Real production snapshots measured 33x (106 MB to 3.2 MB) and 30x (10 MB to
// 335 KB). This asserts a far weaker bound so the test is about the mechanism
// working, not about a specific ratio that legitimately varies with content.
func TestPayloadCodec_CompressesRepetitiveJSON(t *testing.T) {
	original := bytes.Repeat([]byte(`{"path":"internal/graph/sqlite.go","kind":"func","name":"SaveSnapshot"},`), 2000)

	encoded, _, err := encodePayload(original)
	if err != nil {
		t.Fatalf("encodePayload: %v", err)
	}
	if len(encoded) >= len(original)/2 {
		t.Errorf("compressed %d bytes to %d; expected at least a 2x reduction on repetitive JSON", len(original), len(encoded))
	}
}

// TestPayloadCodec_LegacyRowsReadUnchanged asserts a row written before
// compression still decodes.
//
// This is the whole compatibility story: existing rows get an empty codec from
// the ALTER TABLE default, and the read path returns them untouched. Nothing
// rewrites 238 multi-hundred-megabyte payloads in a blocking migration.
func TestPayloadCodec_LegacyRowsReadUnchanged(t *testing.T) {
	legacy := []byte(`{"files":[{"path":"legacy.go"}]}`)

	decoded, err := decodePayload(legacy, codecNone)
	if err != nil {
		t.Fatalf("decodePayload for a legacy row: %v", err)
	}
	if !bytes.Equal(decoded, legacy) {
		t.Error("a legacy raw-JSON payload was altered on read")
	}
}

// TestPayloadCodec_RejectsUnknownCodec asserts an unrecognised marker is an
// error rather than a silent fallback to raw bytes, which would surface as a
// confusing JSON parse failure far from the real cause.
func TestPayloadCodec_RejectsUnknownCodec(t *testing.T) {
	if _, err := decodePayload([]byte("whatever"), "brotli"); err == nil {
		t.Error("an unknown codec decoded successfully")
	}
}

// TestPayloadCodec_RejectsCorruptCompressedData asserts corruption is reported
// rather than returning garbage.
func TestPayloadCodec_RejectsCorruptCompressedData(t *testing.T) {
	if _, err := decodePayload([]byte("this is not gzip"), codecGzip); err == nil {
		t.Error("corrupt compressed data decoded successfully")
	}
}

// TestSnapshotStorage_MixedEncodingsCoexist is the end-to-end proof: a legacy
// row and a newly written row live in the same table and both read correctly.
func TestSnapshotStorage_MixedEncodingsCoexist(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{})
	ctx := context.Background()

	// A legacy row: raw JSON, empty codec, written the way the old code did.
	legacyPayload, err := json.Marshal(snapshotPayload{
		Files: []FileNode{{Path: "legacy/main.go"}},
	})
	if err != nil {
		t.Fatalf("marshal legacy payload: %v", err)
	}
	if _, err := db.ExecContext(ctx,
		`INSERT INTO graph_snapshots (id, scenario, content_hash, source_fingerprint, payload, payload_codec, extracted_at, extraction_ms)
		 VALUES (?,?,?,?,?,?,?,?)`,
		"legacy-1", "demo", "hash-legacy", "", legacyPayload, "",
		time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC).Format(snapshotTimeFormat), 1,
	); err != nil {
		t.Fatalf("insert legacy row: %v", err)
	}

	// A new row written through the repository, which compresses.
	saved, err := repo.SaveSnapshot(ctx, GraphSnapshot{
		ID:          "modern-1",
		Scenario:    "demo",
		ContentHash: "hash-modern",
		ExtractedAt: time.Date(2026, 7, 2, 0, 0, 0, 0, time.UTC),
		Files:       []FileNode{{Path: "modern/main.go"}},
	})
	if err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	// The new row must actually be stored compressed.
	var codec string
	if err := db.QueryRowContext(ctx, `SELECT payload_codec FROM graph_snapshots WHERE id = ?`, saved.ID).Scan(&codec); err != nil {
		t.Fatalf("read codec: %v", err)
	}
	if codec != codecGzip {
		t.Errorf("new row stored with codec %q, want %q", codec, codecGzip)
	}

	// Both read back correctly.
	legacyRead, err := repo.GetSnapshot(ctx, "legacy-1")
	if err != nil {
		t.Fatalf("read legacy row: %v", err)
	}
	if len(legacyRead.Files) != 1 || legacyRead.Files[0].Path != "legacy/main.go" {
		t.Errorf("legacy row decoded to %+v", legacyRead.Files)
	}

	modernRead, err := repo.GetSnapshot(ctx, "modern-1")
	if err != nil {
		t.Fatalf("read modern row: %v", err)
	}
	if len(modernRead.Files) != 1 || modernRead.Files[0].Path != "modern/main.go" {
		t.Errorf("modern row decoded to %+v", modernRead.Files)
	}

	// Listing spans both encodings in one query.
	page, err := repo.ListSnapshots(ctx, ListSnapshotsFilter{Scenario: "demo"})
	if err != nil {
		t.Fatalf("ListSnapshots: %v", err)
	}
	if len(page.Snapshots) != 2 {
		t.Fatalf("listed %d snapshots, want 2", len(page.Snapshots))
	}
}

// TestSnapshotStorage_CompressionShrinksStoredRows asserts the stored bytes are
// genuinely smaller than the JSON they encode.
func TestSnapshotStorage_CompressionShrinksStoredRows(t *testing.T) {
	db, _ := newRetentionDB(t)
	repo := NewSQLiteRepository(db, clock.System{})
	ctx := context.Background()

	// A snapshot shaped like a real one: many similar file records.
	files := make([]FileNode, 0, 3000)
	for i := 0; i < 3000; i++ {
		files = append(files, FileNode{Path: fmt.Sprintf("internal/graph/generated_%04d.go", i)})
	}
	snap := GraphSnapshot{
		ID:          "big-1",
		Scenario:    "demo",
		ContentHash: "hash-big",
		ExtractedAt: time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC),
		Files:       files,
	}

	rawJSON, err := json.Marshal(snapshotPayload{Files: snap.Files})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if _, err := repo.SaveSnapshot(ctx, snap); err != nil {
		t.Fatalf("SaveSnapshot: %v", err)
	}

	var storedBytes int
	if err := db.QueryRowContext(ctx, `SELECT length(payload) FROM graph_snapshots WHERE id = ?`, "big-1").Scan(&storedBytes); err != nil {
		t.Fatalf("read stored length: %v", err)
	}

	if storedBytes >= len(rawJSON) {
		t.Errorf("stored %d bytes for %d bytes of JSON; compression saved nothing", storedBytes, len(rawJSON))
	}
	t.Logf("stored %d bytes for %d bytes of JSON (%.1fx)", storedBytes, len(rawJSON), float64(len(rawJSON))/float64(storedBytes))

	// And it still round-trips.
	read, err := repo.GetSnapshot(ctx, "big-1")
	if err != nil {
		t.Fatalf("GetSnapshot: %v", err)
	}
	if len(read.Files) != len(files) {
		t.Errorf("read back %d files, want %d", len(read.Files), len(files))
	}
}

// TestPayloadCodecMigration_IsIdempotent asserts the column migration is safe
// to re-run, which it must be because it runs on every repository call path.
func TestPayloadCodecMigration_IsIdempotent(t *testing.T) {
	db, _ := newRetentionDB(t)
	ctx := context.Background()

	repo := NewSQLiteRepository(db, clock.System{}).(*sqliteRepository)
	for i := 0; i < 3; i++ {
		repo.sourceFingerprintReady.Store(false)
		if err := repo.ensureSourceFingerprintColumn(ctx); err != nil {
			t.Fatalf("migration run %d: %v", i, err)
		}
	}

	if !columnExists(t, db, "graph_snapshots", "payload_codec") {
		t.Error("payload_codec column missing after migration")
	}
}

func columnExists(t *testing.T, db *sql.DB, table, column string) bool {
	t.Helper()
	rows, err := db.QueryContext(context.Background(), fmt.Sprintf("PRAGMA table_info(%s)", table))
	if err != nil {
		t.Fatalf("table_info: %v", err)
	}
	defer rows.Close()

	for rows.Next() {
		var (
			cid           int
			name, colType string
			notNull, pk   int
			dfltValue     sql.NullString
		)
		if err := rows.Scan(&cid, &name, &colType, &notNull, &dfltValue, &pk); err != nil {
			t.Fatalf("scan table_info: %v", err)
		}
		if name == column {
			return true
		}
	}
	return false
}
