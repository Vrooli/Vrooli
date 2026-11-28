package validation

import (
	"strings"
	"testing"
)

// TestIsoDatabasePath_DetectsGenericEnvironmentReads covers the exact shape the
// fleet carried before consolidation, plus the shapes that must NOT fire.
func TestIsoDatabasePath_DetectsGenericEnvironmentReads(t *testing.T) {
	a := isoDatabasePath{}
	cases := []struct {
		name    string
		path    string
		src     string
		wantHit bool
	}{
		{
			// The pre-consolidation fleet idiom, reproduced verbatim. This is
			// the pattern that put twelve scenarios on one database file.
			name: "fail_generic_sqlite_path",
			path: "api/main.go",
			src: `package main

import (
	"os"
	"strings"
)

func sqliteDSN() (string, error) {
	if path := strings.TrimSpace(os.Getenv("SQLITE_PATH")); path != "" {
		return path, nil
	}
	return "", nil
}
`,
			wantHit: true,
		},
		{
			name: "fail_generic_sqlite_db_lookup",
			path: "api/main.go",
			src: `package main

import "os"

func dbPath() string {
	if path, ok := os.LookupEnv("SQLITE_DB"); ok {
		return path
	}
	return ""
}
`,
			wantHit: true,
		},
		{
			// DATABASE_URL only carries this defect when the file accepts a
			// "file:" SQLite DSN through it — which is how three scenarios
			// took their database path from an inherited environment.
			name: "fail_database_url_used_as_sqlite_dsn",
			path: "api/main.go",
			src: `package main

import (
	"os"
	"strings"
)

func dsn() string {
	if v := os.Getenv("DATABASE_URL"); strings.HasPrefix(v, "file:") {
		return v
	}
	return ""
}
`,
			wantHit: true,
		},
		{
			// Reading DATABASE_URL for PostgreSQL is correct. Postgres isolates
			// per variant through the lifecycle-injected POSTGRES_DB, so the
			// URL is not a cross-scenario path. Flagging it would make this
			// analyzer noise, and a noisy analyzer gets switched off.
			name: "pass_database_url_for_postgres",
			path: "api/main.go",
			src: `package main

import (
	"os"
	"strings"
)

func resolveDatabaseURL() string {
	if raw := strings.TrimSpace(os.Getenv("DATABASE_URL")); raw != "" {
		return raw
	}
	return "postgres://" + os.Getenv("POSTGRES_HOST")
}
`,
			wantHit: false,
		},
		{
			// A scenario-scoped name carries its owner, so it cannot silently
			// capture a sibling. Flagging it would be a false positive.
			name: "pass_scenario_scoped_variable",
			path: "api/main.go",
			src: `package main

import "os"

func dsn() string { return os.Getenv("BAS_SQLITE_PATH") }
`,
			wantHit: false,
		},
		{
			// The storage-root levers redirect a whole tree, not one file, so
			// every scenario beneath one still resolves to its own path. They
			// are how a test harness isolates storage safely.
			name: "pass_storage_root_lever",
			path: "api/main.go",
			src: `package main

import "os"

func root() string {
	if v := os.Getenv("VROOLI_STORAGE_ROOT"); v != "" {
		return v
	}
	return os.Getenv("VROOLI_STORAGE_NAMESPACE")
}
`,
			wantHit: false,
		},
		{
			// The target state: identity in, path out, no environment read.
			name: "pass_resolves_through_the_seam",
			path: "api/main.go",
			src: `package main

import "github.com/vrooli/api-core/storage"

func dsn() (string, error) {
	return storage.SQLiteDSN(storage.SQLiteConfig{Scenario: "example-scenario"})
}
`,
			wantHit: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := a.analyzeSource(tc.src, tc.path)
			if got := len(findings) > 0; got != tc.wantHit {
				t.Fatalf("expected hit=%v, got %d findings: %+v", tc.wantHit, len(findings), findings)
			}
			for _, f := range findings {
				if f.Code != "DATABASE_PATH_FROM_ENVIRONMENT" {
					t.Fatalf("unexpected code %q", f.Code)
				}
				if f.Remediation == "" {
					t.Fatal("a finding must say how to fix it")
				}
			}
		})
	}
}

func TestIsoDatabasePath_DetectsHandRolledDSN(t *testing.T) {
	a := isoDatabasePath{}
	cases := []struct {
		name    string
		src     string
		wantHit bool
	}{
		{
			// The canonical fleet string, duplicated 60+ times.
			name: "fail_sprintf_pragma_dsn",
			src: `package main

import "fmt"

func fileDSN(path string) string {
	return fmt.Sprintf("file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)", path)
}
`,
			wantHit: true,
		},
		{
			// The drifted copy: pre-"_pragma" grammar the driver ignores, so
			// this scenario never actually ran in WAL mode.
			name: "fail_legacy_grammar_dsn",
			src: `package main

func dsn(path string) string {
	return "file:" + path + "?_journal_mode=WAL&_busy_timeout=5000"
}
`,
			wantHit: true,
		},
		{
			// An in-memory DSN names no file, so nothing can be captured and
			// no tuning can drift in a way that matters.
			name: "pass_in_memory_dsn",
			src: `package main

func dsn() string { return "file::memory:?_pragma=foreign_keys(ON)" }
`,
			wantHit: false,
		},
		{
			name: "pass_seam_with_typed_tuning",
			src: `package main

import "github.com/vrooli/api-core/storage"

func dsn(path string) (string, error) {
	return storage.SQLiteDSNAt(path, storage.SQLiteTuning{TxLock: "immediate"})
}
`,
			wantHit: false,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			findings := a.analyzeSource(tc.src, "api/main.go")
			if got := len(findings) > 0; got != tc.wantHit {
				t.Fatalf("expected hit=%v, got %d findings: %+v", tc.wantHit, len(findings), findings)
			}
		})
	}
}

// TestIsoDatabasePath_ExemptsAPICore keeps the analyzer from flagging the one
// package that legitimately owns this knowledge.
func TestIsoDatabasePath_ExemptsAPICore(t *testing.T) {
	a := isoDatabasePath{}
	src := `package storage

import "os"

func legacy() string { return os.Getenv("SQLITE_PATH") }
`
	if findings := a.analyzeSource(src, "packages/api-core/storage/sqlite.go"); len(findings) != 0 {
		t.Fatalf("api-core must be exempt, got %d findings", len(findings))
	}
}

// TestIsoDatabasePath_MessageNamesTheRisk guards the thing that makes this
// analyzer useful rather than annoying: an engineer who hits it must learn why
// the pattern is unsafe, not merely that it is banned.
func TestIsoDatabasePath_MessageNamesTheRisk(t *testing.T) {
	a := isoDatabasePath{}
	src := `package main

import "os"

func dsn() string { return os.Getenv("SQLITE_PATH") }
`
	findings := a.analyzeSource(src, "api/main.go")
	if len(findings) == 0 {
		t.Fatal("expected a finding")
	}
	msg := findings[0].Message
	for _, want := range []string{"inheritance", "supervisor", "scoped"} {
		if !strings.Contains(strings.ToLower(msg), want) {
			t.Errorf("message should explain %q: %s", want, msg)
		}
	}
	if !strings.Contains(findings[0].Remediation, "storage.SQLiteDSN") {
		t.Errorf("remediation should name the seam: %s", findings[0].Remediation)
	}
}
