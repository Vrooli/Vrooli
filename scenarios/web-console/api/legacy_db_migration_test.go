package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMigrateLegacyDB_CopiesWhenCanonicalAbsent(t *testing.T) {
	// XDG_DATA_HOME is the most-specific legacy candidate, so pointing it at a
	// temp dir makes the migration deterministic regardless of any real DB under
	// the developer's home.
	legacyHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", legacyHome)
	legacyDir := filepath.Join(legacyHome, "vrooli", "web-console")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	legacyDB := filepath.Join(legacyDir, "web-console.db")
	if err := os.WriteFile(legacyDB, []byte("legacy-data"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyDB+"-wal", []byte("wal-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(t.TempDir(), "vrooli", "web-console", "web-console.db")
	migrateLegacyDB(dbPath)

	got, err := os.ReadFile(dbPath)
	if err != nil {
		t.Fatalf("canonical DB not created by migration: %v", err)
	}
	if string(got) != "legacy-data" {
		t.Errorf("canonical DB content = %q, want %q", got, "legacy-data")
	}
	if wal, err := os.ReadFile(dbPath + "-wal"); err != nil || string(wal) != "wal-data" {
		t.Errorf("WAL sidecar not copied: content=%q err=%v", wal, err)
	}
}

func TestMigrateLegacyDB_NoopWhenCanonicalPresent(t *testing.T) {
	legacyHome := t.TempDir()
	t.Setenv("XDG_DATA_HOME", legacyHome)
	legacyDir := filepath.Join(legacyHome, "vrooli", "web-console")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "web-console.db"), []byte("legacy-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	dbPath := filepath.Join(t.TempDir(), "web-console.db")
	if err := os.WriteFile(dbPath, []byte("canonical-data"), 0o644); err != nil {
		t.Fatal(err)
	}

	migrateLegacyDB(dbPath)

	got, _ := os.ReadFile(dbPath)
	if string(got) != "canonical-data" {
		t.Errorf("canonical DB was overwritten by migration: %q", got)
	}
}

// Regression test for the 2026-05-27 hook-token data-loss companion bug:
// migrateLegacyDB shipped without a peer migration for the State-class files
// (hook-token.txt, voice/tts/wakeword configs). The canonical state dir was
// therefore empty after the ~/.vrooli relocation, loadOrCreateHookToken
// generated a fresh random token, and every claude-code hook POST 401'd —
// silently zeroing the conversation_events stream. State files live under
// XDG_STATE_HOME (~/.local/state), NOT XDG_DATA_HOME, so they require a
// separate legacy resolution path.
func TestMigrateLegacyStateFile_CopiesHookTokenWhenCanonicalAbsent(t *testing.T) {
	legacyHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", legacyHome)
	// Defensively clear HOME-rooted fallback so the test is hermetic — the
	// resolver consults ~/.local/state when XDG_STATE_HOME is unset.
	t.Setenv("HOME", t.TempDir())
	legacyDir := filepath.Join(legacyHome, "vrooli", "web-console")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	want := "legacy-hook-token-deadbeefcafef00d"
	if err := os.WriteFile(filepath.Join(legacyDir, "hook-token.txt"), []byte(want), 0o600); err != nil {
		t.Fatal(err)
	}

	canonical := filepath.Join(t.TempDir(), "vrooli", "web-console", "hook-token.txt")
	migrateLegacyStateFile(canonical, "hook-token.txt")

	got, err := os.ReadFile(canonical)
	if err != nil {
		t.Fatalf("canonical hook-token not created by migration: %v", err)
	}
	if string(got) != want {
		t.Errorf("canonical hook-token = %q, want %q", got, want)
	}
}

func TestMigrateLegacyStateFile_NoopWhenCanonicalPresent(t *testing.T) {
	legacyHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", legacyHome)
	t.Setenv("HOME", t.TempDir())
	legacyDir := filepath.Join(legacyHome, "vrooli", "web-console")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "hook-token.txt"), []byte("legacy"), 0o600); err != nil {
		t.Fatal(err)
	}

	canonical := filepath.Join(t.TempDir(), "hook-token.txt")
	if err := os.WriteFile(canonical, []byte("canonical"), 0o600); err != nil {
		t.Fatal(err)
	}

	migrateLegacyStateFile(canonical, "hook-token.txt")

	got, _ := os.ReadFile(canonical)
	if string(got) != "canonical" {
		t.Errorf("canonical hook-token was overwritten by migration: %q", got)
	}
}

// The token file is sensitive — confirm the migrated copy preserves owner-only
// permissions (0o600). A widened-mode copy would be a quiet credential leak.
func TestMigrateLegacyStateFile_PreservesTokenMode(t *testing.T) {
	legacyHome := t.TempDir()
	t.Setenv("XDG_STATE_HOME", legacyHome)
	t.Setenv("HOME", t.TempDir())
	legacyDir := filepath.Join(legacyHome, "vrooli", "web-console")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "hook-token.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatal(err)
	}

	canonical := filepath.Join(t.TempDir(), "hook-token.txt")
	migrateLegacyStateFile(canonical, "hook-token.txt")

	info, err := os.Stat(canonical)
	if err != nil {
		t.Fatal(err)
	}
	if mode := info.Mode().Perm(); mode != 0o600 {
		t.Errorf("migrated hook-token mode = %#o, want 0o600", mode)
	}
}
