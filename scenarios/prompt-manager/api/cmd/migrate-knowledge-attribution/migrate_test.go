package main

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMigrateStore_GoldenFile is the load-bearing end-to-end check: the
// migration tool, given testdata/basic/input as a fresh tree, must produce
// every file in testdata/basic/expected exactly. Any future change to the
// canonical KnowledgeEntry shape, AttributionInfo zero-value, or team.json
// insertion logic surfaces here as a focused diff.
//
// The cutoff date must match the date hard-coded into the expected
// team.json fixtures; do not parameterize.
func TestMigrateStore_GoldenFile(t *testing.T) {
	const cutoff = "2026-05-04"

	tmp := t.TempDir()
	copyTree(t, "testdata/basic/input", tmp)

	stats, err := migrateStore(migrateOpts{
		StoreRoot:  tmp,
		CutoffDate: cutoff,
		DryRun:     false,
	})
	if err != nil {
		t.Fatalf("migrateStore: %v", err)
	}

	// Stats sanity check; specific values here are the contract for the
	// fixture set. Update both fixtures and these expectations in lockstep.
	//
	// Per-team breakdown (4 teams, 6 entries total scanned):
	//   alpha: 3 legacy entries          -> migrated; team.json migrated; 2 backups
	//   beta:  0 entries (empty file)    -> skipped; team.json migrated; 1 backup
	//   gamma: 1 already-migrated entry  -> skipped; team.json skipped (already had cutoff)
	//   delta: 1 legacy + 1 migrated     -> 1 migrated 1 skipped; team.json migrated; 2 backups
	wantStats := migrationStats{
		TeamsScanned:    4,
		TeamsMigrated:   3,
		TeamsSkipped:    1,
		EntriesScanned:  6,
		EntriesMigrated: 4,
		EntriesSkipped:  2,
		BackupsWritten:  5,
	}

	if stats != wantStats {
		t.Errorf("stats mismatch:\n  got:  %+v\n  want: %+v", stats, wantStats)
	}

	assertTreesEqual(t, "testdata/basic/expected", tmp)
}

// TestMigrateStore_Idempotent: re-running the migration over an
// already-migrated tree (testdata/basic/expected) is a perfect no-op:
// every file is byte-identical to its starting state, no .backup files
// land, and stats report zero changes.
func TestMigrateStore_Idempotent(t *testing.T) {
	const cutoff = "2026-05-04"

	tmp := t.TempDir()
	copyTree(t, "testdata/basic/expected", tmp)

	// Snapshot every file's bytes before the run; we'll compare after.
	snapshot := snapshotTree(t, tmp)

	stats, err := migrateStore(migrateOpts{
		StoreRoot:  tmp,
		CutoffDate: cutoff,
		DryRun:     false,
	})
	if err != nil {
		t.Fatalf("migrateStore: %v", err)
	}

	if stats.TeamsMigrated != 0 {
		t.Errorf("expected zero teams migrated on idempotent run; got %d", stats.TeamsMigrated)
	}
	if stats.EntriesMigrated != 0 {
		t.Errorf("expected zero entries migrated on idempotent run; got %d", stats.EntriesMigrated)
	}
	if stats.BackupsWritten != 0 {
		t.Errorf("expected zero backups on idempotent run; got %d", stats.BackupsWritten)
	}

	after := snapshotTree(t, tmp)
	if !sameTreeSnapshots(snapshot, after) {
		t.Error("idempotent run mutated files; expected byte-perfect preservation")
		diffSnapshots(t, snapshot, after)
	}

	// Also assert no .backup files exist anywhere.
	walkErr := filepath.WalkDir(tmp, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if strings.HasSuffix(path, ".backup") {
			t.Errorf("idempotent run wrote unexpected backup: %s", path)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk: %v", walkErr)
	}
}

// TestMigrateStore_DryRun: --dry-run reports what would change without
// touching the filesystem. After a dry-run the input tree is byte-identical
// to its starting state, and no .backup files exist.
func TestMigrateStore_DryRun(t *testing.T) {
	const cutoff = "2026-05-04"

	tmp := t.TempDir()
	copyTree(t, "testdata/basic/input", tmp)
	beforeSnapshot := snapshotTree(t, tmp)

	stats, err := migrateStore(migrateOpts{
		StoreRoot:  tmp,
		CutoffDate: cutoff,
		DryRun:     true,
	})
	if err != nil {
		t.Fatalf("migrateStore: %v", err)
	}

	// Stats should still report what the run *would* do, modulo backups
	// (which are filesystem side effects only).
	if stats.TeamsMigrated == 0 {
		t.Error("expected dry-run to report teams that would be migrated; got 0")
	}
	if stats.EntriesMigrated == 0 {
		t.Error("expected dry-run to report entries that would be migrated; got 0")
	}
	if stats.BackupsWritten != 0 {
		t.Errorf("dry-run must not write backups; got %d", stats.BackupsWritten)
	}

	afterSnapshot := snapshotTree(t, tmp)
	if !sameTreeSnapshots(beforeSnapshot, afterSnapshot) {
		t.Error("dry-run mutated the filesystem")
		diffSnapshots(t, beforeSnapshot, afterSnapshot)
	}
}

// TestMigrateStore_BackupContents: every .backup file emitted by the
// migration must contain the exact pre-migration content of its
// corresponding original file.
func TestMigrateStore_BackupContents(t *testing.T) {
	const cutoff = "2026-05-04"

	src := "testdata/basic/input"
	tmp := t.TempDir()
	copyTree(t, src, tmp)
	originals := snapshotTree(t, src)

	if _, err := migrateStore(migrateOpts{StoreRoot: tmp, CutoffDate: cutoff}); err != nil {
		t.Fatalf("migrateStore: %v", err)
	}

	walkErr := filepath.WalkDir(tmp, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".backup") {
			return nil
		}
		// Map .backup file back to its original source path so we can
		// compare against the input fixture content.
		rel, err := filepath.Rel(tmp, path)
		if err != nil {
			return err
		}
		origRel := strings.TrimSuffix(rel, ".backup")
		want, ok := originals[origRel]
		if !ok {
			t.Errorf("backup %s has no matching original in input fixtures", rel)
			return nil
		}
		got, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		if !bytes.Equal(got, want) {
			t.Errorf("backup %s does not match pre-migration original\n  got:  %q\n  want: %q",
				rel, got, want)
		}
		return nil
	})
	if walkErr != nil {
		t.Fatalf("walk: %v", walkErr)
	}
}

// TestMigrateStore_BackupNotOverwrittenOnRerun: a .backup written by the
// first migration must survive a hand-edit of the migrated file plus a
// second migration. The .backup is the original-original; subsequent runs
// never clobber it.
func TestMigrateStore_BackupNotOverwrittenOnRerun(t *testing.T) {
	const cutoff = "2026-05-04"

	tmp := t.TempDir()
	copyTree(t, "testdata/basic/input", tmp)

	// First run: produces .backup files.
	if _, err := migrateStore(migrateOpts{StoreRoot: tmp, CutoffDate: cutoff}); err != nil {
		t.Fatalf("first migrateStore: %v", err)
	}

	backupPath := filepath.Join(tmp, "teams", "team-alpha", "shared", "knowledge.jsonl.backup")
	originalBackup, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("read backup: %v", err)
	}

	// Append a new legacy-shaped entry to the migrated knowledge.jsonl so a
	// second migration has work to do (otherwise the no-op idempotent path
	// short-circuits before reaching writeWithBackup).
	jsonlPath := filepath.Join(tmp, "teams", "team-alpha", "shared", "knowledge.jsonl")
	jsonlContent, err := os.ReadFile(jsonlPath)
	if err != nil {
		t.Fatalf("read jsonl: %v", err)
	}
	addedLine := []byte(`{"id":"knw-a-4","at":"2026-04-04T12:00:00Z","by":"editor","topic":"snapshot/04","content":"appended legacy"}` + "\n")
	if err := os.WriteFile(jsonlPath, append(jsonlContent, addedLine...), 0o644); err != nil {
		t.Fatalf("rewrite jsonl: %v", err)
	}

	// Second run: should migrate the new line but leave the existing
	// .backup file untouched.
	if _, err := migrateStore(migrateOpts{StoreRoot: tmp, CutoffDate: cutoff}); err != nil {
		t.Fatalf("second migrateStore: %v", err)
	}

	got, err := os.ReadFile(backupPath)
	if err != nil {
		t.Fatalf("re-read backup: %v", err)
	}
	if !bytes.Equal(got, originalBackup) {
		t.Errorf("backup overwritten on rerun:\n  got:  %q\n  want: %q", got, originalBackup)
	}
}

// TestMigrateStore_RejectsMissingTeamsDir surfaces a clear error when the
// caller points --root at a directory that doesn't contain teams/. Without
// this guard, the tool would silently report "0 teams migrated" — a
// pleasant-looking failure mode that masks operator typos.
func TestMigrateStore_RejectsMissingTeamsDir(t *testing.T) {
	tmp := t.TempDir() // no teams/ subdir
	_, err := migrateStore(migrateOpts{StoreRoot: tmp, CutoffDate: "2026-05-04"})
	if err == nil {
		t.Fatal("expected error when teams/ is missing; got nil")
	}
	if !strings.Contains(err.Error(), "teams") {
		t.Errorf("error message should mention teams dir; got %q", err)
	}
}

// --- insertTeamAttributionValidFrom -----------------------------------

func TestInsertTeamAttributionValidFrom_AppendsField(t *testing.T) {
	src := []byte("{\n  \"id\": \"foo\",\n  \"enabled\": true\n}\n")
	out, changed, err := insertTeamAttributionValidFrom(src, "2026-05-04")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true on first migration")
	}
	want := "{\n  \"id\": \"foo\",\n  \"enabled\": true,\n  \"attributionValidFrom\": \"2026-05-04\"\n}\n"
	if string(out) != want {
		t.Errorf("unexpected output:\n  got:  %q\n  want: %q", out, want)
	}
}

func TestInsertTeamAttributionValidFrom_PreservesNoTrailingNewline(t *testing.T) {
	src := []byte("{\"id\":\"foo\"}")
	out, _, err := insertTeamAttributionValidFrom(src, "2026-05-04")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if bytes.HasSuffix(out, []byte("\n")) {
		t.Errorf("trailing newline added unexpectedly: %q", out)
	}
}

func TestInsertTeamAttributionValidFrom_Idempotent(t *testing.T) {
	src := []byte("{\n  \"id\": \"foo\",\n  \"attributionValidFrom\": \"2026-04-15\"\n}\n")
	out, changed, err := insertTeamAttributionValidFrom(src, "2026-05-04")
	if err != nil {
		t.Fatalf("insert: %v", err)
	}
	if changed {
		t.Error("expected changed=false when field already present")
	}
	if !bytes.Equal(out, src) {
		t.Errorf("output mutated despite changed=false:\n  got:  %q\n  want: %q", out, src)
	}
}

func TestInsertTeamAttributionValidFrom_RejectsEmpty(t *testing.T) {
	_, _, err := insertTeamAttributionValidFrom([]byte("  \n"), "2026-05-04")
	if err == nil {
		t.Fatal("expected error on empty input")
	}
}

func TestInsertTeamAttributionValidFrom_RejectsNonObject(t *testing.T) {
	_, _, err := insertTeamAttributionValidFrom([]byte("[1,2,3]"), "2026-05-04")
	if err == nil {
		t.Fatal("expected error on JSON-array input")
	}
}

// --- migrateKnowledgeLine ---------------------------------------------

// TestMigrateKnowledgeLine_LegacyFidelity pins the same on-disk shape the
// store-package canon test (TestKnowledgeEntry_LegacyMigrationFidelity)
// requires, but exercised through the migration tool's transformer
// rather than via direct struct construction. If this test diverges from
// the canon test, P3.2 is producing a shape ruleActualWriterUndeclared
// (P3.6) won't recognize.
func TestMigrateKnowledgeLine_LegacyFidelity(t *testing.T) {
	in := []byte(`{"id":"knw-l-1","at":"2026-04-01T12:00:00Z","by":"director","topic":"snapshot/01","content":"hello"}`)
	out, changed, err := migrateKnowledgeLine(in)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true for legacy entry")
	}
	for _, want := range []string{
		`"caller":"legacy:director"`,
		`"caller_note":"director"`,
		`"kind":"legacy"`,
		`"spawn_origin":"legacy"`,
		`"member_id":null`,
		`"team_id":null`,
		`"run_id":null`,
		`"source_skill_id":null`,
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("missing %s in output: %s", want, out)
		}
	}
	if bytes.Contains(out, []byte(`"by":`)) {
		t.Errorf("legacy 'by' field must be dropped from migrated entry: %s", out)
	}
}

func TestMigrateKnowledgeLine_LegacyEmptyByValue(t *testing.T) {
	in := []byte(`{"id":"knw-empty","at":"2026-04-01T12:00:00Z","by":"","topic":"t","content":"c"}`)
	out, changed, err := migrateKnowledgeLine(in)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if !changed {
		t.Fatal("expected changed=true")
	}
	// caller is "legacy:" even when the original by-value was empty —
	// signals "no historical attribution available" while preserving
	// the legacy-prefix invariant readers depend on.
	if !bytes.Contains(out, []byte(`"caller":"legacy:"`)) {
		t.Errorf(`expected caller="legacy:" for empty-by entry; got %s`, out)
	}
	// Empty caller_note round-trips to omitted (omitempty struct tag);
	// validators must not over-fit on the field's presence.
	if bytes.Contains(out, []byte(`"caller_note"`)) {
		t.Errorf("expected caller_note omitted for empty by-value; got %s", out)
	}
}

func TestMigrateKnowledgeLine_PreservesOptionalFields(t *testing.T) {
	in := []byte(`{"id":"knw-1","at":"2026-04-01T12:00:00Z","by":"x","topic":"t","content":"c","source":"https://e.example","supersedes":"knw-0"}`)
	out, _, err := migrateKnowledgeLine(in)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	for _, want := range []string{
		`"source":"https://e.example"`,
		`"supersedes":"knw-0"`,
	} {
		if !bytes.Contains(out, []byte(want)) {
			t.Errorf("missing %s in output: %s", want, out)
		}
	}
}

func TestMigrateKnowledgeLine_SkipsAlreadyMigrated(t *testing.T) {
	in := []byte(`{"id":"knw-2","at":"2026-05-04T12:00:00Z","topic":"t","content":"c","caller":"operator","attribution":{"kind":"operator-direct","member_id":null,"team_id":null,"run_id":null,"spawn_origin":"operator-cli","source_skill_id":null}}`)
	out, changed, err := migrateKnowledgeLine(in)
	if err != nil {
		t.Fatalf("migrate: %v", err)
	}
	if changed {
		t.Error("expected changed=false for already-migrated entry")
	}
	if !bytes.Equal(out, in) {
		t.Errorf("already-migrated line should round-trip verbatim:\n  got:  %s\n  want: %s", out, in)
	}
}

func TestMigrateKnowledgeLine_RejectsMalformed(t *testing.T) {
	cases := map[string][]byte{
		"neither by nor attribution": []byte(`{"id":"x","at":"y","topic":"t","content":"c"}`),
		"not JSON":                   []byte(`not-json`),
		"by-not-a-string":            []byte(`{"id":"x","by":42}`),
	}
	for name, in := range cases {
		t.Run(name, func(t *testing.T) {
			_, _, err := migrateKnowledgeLine(in)
			if err == nil {
				t.Fatal("expected error; got nil")
			}
		})
	}
}

// --- helpers ----------------------------------------------------------

// copyTree copies every regular file under src into dst, creating dirs as
// needed. Failures abort the test.
func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		return copyFile(path, out)
	})
	if err != nil {
		t.Fatalf("copyTree %s -> %s: %v", src, dst, err)
	}
}

func copyFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	info, err := in.Stat()
	if err != nil {
		return err
	}
	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	if _, err := io.Copy(out, in); err != nil {
		return err
	}
	return nil
}

// snapshotTree returns a path -> content map for every regular file
// under root, indexed by path relative to root. Used to compare
// before/after states for byte-perfect equality checks.
func snapshotTree(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := make(map[string][]byte)
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		out[rel] = data
		return nil
	})
	if err != nil {
		t.Fatalf("snapshotTree %s: %v", root, err)
	}
	return out
}

func sameTreeSnapshots(a, b map[string][]byte) bool {
	if len(a) != len(b) {
		return false
	}
	for k, va := range a {
		vb, ok := b[k]
		if !ok {
			return false
		}
		if !bytes.Equal(va, vb) {
			return false
		}
	}
	return true
}

func diffSnapshots(t *testing.T, before, after map[string][]byte) {
	t.Helper()
	keys := make(map[string]bool)
	for k := range before {
		keys[k] = true
	}
	for k := range after {
		keys[k] = true
	}
	for k := range keys {
		bv := before[k]
		av := after[k]
		if !bytes.Equal(bv, av) {
			t.Logf("file %s changed:\n  before: %s\n  after:  %s", k, bv, av)
		}
	}
}

// assertTreesEqual walks every file under `want` and compares it byte-for-byte
// against the same relative path under `got`. Extra files in `got` are also
// flagged. This is the load-bearing assertion for the golden-file test.
func assertTreesEqual(t *testing.T, want, got string) {
	t.Helper()
	wantSnap := snapshotTree(t, want)
	gotSnap := snapshotTree(t, got)

	// Allow .backup and .tmp files in `got` that are not present in `want`
	// — those are migration artifacts, not part of the canonical end state.
	filtered := make(map[string][]byte, len(gotSnap))
	for k, v := range gotSnap {
		if strings.HasSuffix(k, ".backup") || strings.HasSuffix(k, ".tmp") {
			continue
		}
		filtered[k] = v
	}

	if len(filtered) != len(wantSnap) {
		var extra, missing []string
		for k := range filtered {
			if _, ok := wantSnap[k]; !ok {
				extra = append(extra, k)
			}
		}
		for k := range wantSnap {
			if _, ok := filtered[k]; !ok {
				missing = append(missing, k)
			}
		}
		t.Errorf("file count mismatch: got %d, want %d\n  extra:   %v\n  missing: %v",
			len(filtered), len(wantSnap), extra, missing)
	}

	for rel, wantData := range wantSnap {
		gotData, ok := filtered[rel]
		if !ok {
			continue
		}
		if !bytes.Equal(gotData, wantData) {
			t.Errorf("file %s differs from expected:\n  got:  %s\n  want: %s",
				rel, gotData, wantData)
		}
	}
}

// stripWhitespace exists in migrate.go for live use; we re-validate it here
// so future callers can rely on the contract.
func TestStripWhitespace(t *testing.T) {
	cases := map[string]string{
		"":            "",
		"abc":         "abc",
		"  a b\tc\n":  "abc",
		"\r\n\t  ":    "",
		"x\ny\rz\t  ": "xyz",
	}
	for in, want := range cases {
		if got := stripWhitespace(in); got != want {
			t.Errorf("stripWhitespace(%q) = %q, want %q", in, got, want)
		}
	}
}

// shadow imports of unused stdlib symbols can sneak in via golden tests;
// keep this asserting-error to surface them.
var (
	_ = errors.New
	_ = fmt.Sprintf
)
