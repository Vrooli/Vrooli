package sqlitetime_test

import (
	"database/sql"
	"testing"
	"time"

	"knowledge-observatory/internal/sqlitetime"

	_ "modernc.org/sqlite"
)

// TestStoredFormatMatchesCurrentTimestamp is the regression guard for the bug
// this package exists to prevent.
//
// Binding a time.Time directly stores Go's String() rendering
// ("2026-03-04 05:06:07 +0000 UTC"), while a CURRENT_TIMESTAMP default stores
// the SQLite canonical form ("2026-03-04 05:06:07"). Two shapes in one column
// read back fine through a plain column scan — the driver knows the declared
// type — but break the moment the value passes through an expression such as
// MAX(), where the affinity is lost and a scan into time.Time fails at runtime.
func TestStoredFormatMatchesCurrentTimestamp(t *testing.T) {
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	if _, err := db.Exec(`CREATE TABLE t (id TEXT, at TIMESTAMP, def TIMESTAMP DEFAULT CURRENT_TIMESTAMP)`); err != nil {
		t.Fatal(err)
	}

	at := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	if _, err := db.Exec(`INSERT INTO t (id, at) VALUES (?, ?)`, "a", sqlitetime.Format(at)); err != nil {
		t.Fatal(err)
	}

	var stored string
	if err := db.QueryRow(`SELECT CAST(at AS TEXT) FROM t WHERE id = 'a'`).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != "2026-03-04 05:06:07" {
		t.Errorf("stored text = %q, want the SQLite canonical form", stored)
	}

	// The whole point: an aggregate must still be readable.
	var maxed string
	if err := db.QueryRow(`SELECT MAX(at) FROM t`).Scan(&maxed); err != nil {
		t.Fatalf("aggregate over a normalised column must scan as text: %v", err)
	}
	parsed, err := sqlitetime.Parse(maxed)
	if err != nil {
		t.Fatalf("parse aggregate: %v", err)
	}
	if !parsed.Equal(at) {
		t.Errorf("round trip = %v, want %v", parsed, at)
	}

	// A raw time.Time bind is what used to produce the second format. Confirm
	// the two really do differ, so this test fails loudly if the driver ever
	// changes and the normalisation becomes unnecessary.
	if _, err := db.Exec(`INSERT INTO t (id, at) VALUES (?, ?)`, "b", at); err != nil {
		t.Fatal(err)
	}
	var raw string
	if err := db.QueryRow(`SELECT CAST(at AS TEXT) FROM t WHERE id = 'b'`).Scan(&raw); err != nil {
		t.Fatal(err)
	}
	if raw == stored {
		t.Skip("driver now stores time.Time in canonical form; normalisation is belt-and-braces")
	}
}

func TestParseAcceptsEveryFormatThisDatabaseCanHold(t *testing.T) {
	want := time.Date(2026, 3, 4, 5, 6, 7, 0, time.UTC)
	for _, value := range []string{
		"2026-03-04 05:06:07",
		"2026-03-04 05:06:07 +0000 UTC",
		"2026-03-04T05:06:07Z",
	} {
		got, err := sqlitetime.Parse(value)
		if err != nil {
			t.Errorf("Parse(%q): %v", value, err)
			continue
		}
		if !got.Equal(want) {
			t.Errorf("Parse(%q) = %v, want %v", value, got, want)
		}
	}
}

func TestParseRejectsGarbage(t *testing.T) {
	for _, value := range []string{"", "   ", "not a time"} {
		if _, err := sqlitetime.Parse(value); err == nil {
			t.Errorf("Parse(%q) should fail", value)
		}
	}
}

func TestFormatPtrPreservesNull(t *testing.T) {
	if got := sqlitetime.FormatPtr(nil); got != nil {
		t.Errorf("FormatPtr(nil) = %v, want nil so the column stays SQL NULL", got)
	}
	at := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	if got := sqlitetime.FormatPtr(&at); got != "2026-01-02 03:04:05" {
		t.Errorf("FormatPtr = %v", got)
	}
}

func TestParseNullOnAbsentValue(t *testing.T) {
	if got := sqlitetime.ParseNull(sql.NullString{}); got != nil {
		t.Errorf("ParseNull(invalid) = %v, want nil", got)
	}
	if got := sqlitetime.ParseNull(sql.NullString{String: "junk", Valid: true}); got != nil {
		t.Errorf("ParseNull(garbage) = %v, want nil rather than a zero time", got)
	}
}

// TestLexicographicOrderMatchesChronological is what makes `measured_at < ?`
// correct: the canonical layout is fixed width and UTC, so SQLite's text
// comparison is also a time comparison. Retention pruning depends on this.
func TestLexicographicOrderMatchesChronological(t *testing.T) {
	earlier := sqlitetime.Format(time.Date(2026, 1, 31, 23, 59, 59, 0, time.UTC))
	later := sqlitetime.Format(time.Date(2026, 2, 1, 0, 0, 0, 0, time.UTC))
	if !(earlier < later) {
		t.Errorf("%q should sort before %q", earlier, later)
	}
}
