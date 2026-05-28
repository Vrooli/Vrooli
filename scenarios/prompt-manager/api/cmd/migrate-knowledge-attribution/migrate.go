package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"prompt-manager/store"
	"sort"
	"strings"
)

// migrateOpts is the input contract for migrateStore — the single
// entry-point both the binary's main and the test suite use.
type migrateOpts struct {
	// StoreRoot is the prompt-manager store directory; the migration walks
	// <StoreRoot>/teams/<id>/team.json and <StoreRoot>/teams/<id>/shared/knowledge.jsonl.
	StoreRoot string
	// BackupRoot is the centralized .backup directory (paths.Roots.BackupFor
	// produces locations under <RuntimeData>/backups/). Empty means write
	// backups as siblings of the original — only used by tests that don't
	// model the runtime root.
	BackupRoot string
	// CutoffDate is the YYYY-MM-DD value written into each team.json's
	// attributionValidFrom field.
	CutoffDate string
	// DryRun reports proposed changes without touching the filesystem.
	DryRun bool
}

// resolveBackupPath returns the backup destination for a path being
// rewritten in place. Honors the centralized .backup contract (CD-3):
// when opts.BackupRoot is set, backups land under it mirroring the rel
// path beneath StoreRoot; otherwise (test fixtures) they fall back to a
// sibling.
func (opts migrateOpts) resolveBackupPath(path string) (string, error) {
	if strings.TrimSpace(opts.BackupRoot) == "" {
		return path + ".backup", nil
	}
	rel, err := filepath.Rel(opts.StoreRoot, path)
	if err != nil {
		return "", fmt.Errorf("rel backup path: %w", err)
	}
	return filepath.Join(opts.BackupRoot, rel+".backup"), nil
}

// migrationStats summarizes the work performed (or proposed in dry-run).
type migrationStats struct {
	TeamsScanned    int
	TeamsMigrated   int // had attributionValidFrom inserted
	TeamsSkipped    int // already migrated
	EntriesScanned  int // total non-blank knowledge.jsonl lines visited
	EntriesMigrated int // had attribution populated
	EntriesSkipped  int // already migrated
	BackupsWritten  int // count of .backup files written
}

// migrateStore is the testable entry point. It walks the store tree, runs
// per-team migration, and aggregates stats. Returns an error on the first
// per-team failure — partial migrations are not desirable here because the
// caller is one CLI invocation that the operator reviews atomically.
func migrateStore(opts migrateOpts) (migrationStats, error) {
	var stats migrationStats

	teamsDir := filepath.Join(opts.StoreRoot, "teams")
	entries, err := os.ReadDir(teamsDir)
	if err != nil {
		return stats, fmt.Errorf("read teams dir %s: %w", teamsDir, err)
	}

	// Sort for deterministic output across platforms.
	sort.Slice(entries, func(i, j int) bool { return entries[i].Name() < entries[j].Name() })

	for _, te := range entries {
		if !te.IsDir() {
			continue
		}
		teamID := te.Name()
		teamPath := filepath.Join(teamsDir, teamID, "team.json")
		if _, err := os.Stat(teamPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				// Partial scaffold (team directory without team.json); skip.
				continue
			}
			return stats, fmt.Errorf("stat %s: %w", teamPath, err)
		}
		stats.TeamsScanned++

		teamBackup, err := opts.resolveBackupPath(teamPath)
		if err != nil {
			return stats, fmt.Errorf("team.json backup path %s: %w", teamID, err)
		}
		migrated, wroteBackup, err := migrateTeamJSON(teamPath, teamBackup, opts.CutoffDate, opts.DryRun)
		if err != nil {
			return stats, fmt.Errorf("team.json %s: %w", teamID, err)
		}
		if migrated {
			stats.TeamsMigrated++
		} else {
			stats.TeamsSkipped++
		}
		if wroteBackup {
			stats.BackupsWritten++
		}

		jsonlPath := filepath.Join(teamsDir, teamID, "shared", "knowledge.jsonl")
		if _, err := os.Stat(jsonlPath); err != nil {
			if errors.Is(err, os.ErrNotExist) {
				continue
			}
			return stats, fmt.Errorf("stat %s: %w", jsonlPath, err)
		}
		jsonlBackup, err := opts.resolveBackupPath(jsonlPath)
		if err != nil {
			return stats, fmt.Errorf("knowledge.jsonl backup path %s: %w", teamID, err)
		}
		scanned, migratedCount, skipped, wroteJSONLBackup, err := migrateKnowledgeJSONL(jsonlPath, jsonlBackup, opts.DryRun)
		if err != nil {
			return stats, fmt.Errorf("knowledge.jsonl %s: %w", teamID, err)
		}
		stats.EntriesScanned += scanned
		stats.EntriesMigrated += migratedCount
		stats.EntriesSkipped += skipped
		if wroteJSONLBackup {
			stats.BackupsWritten++
		}
	}

	return stats, nil
}

// migrateTeamJSON ensures <path> has an `attributionValidFrom` top-level
// field. Idempotent: if the field is already present, returns
// (migrated=false, wroteBackup=false). Uses raw text insertion to keep the
// diff minimal and to avoid round-tripping the entire team.json through Go
// structs (which would reorder fields significantly).
func migrateTeamJSON(path, backupPath, cutoffDate string, dryRun bool) (migrated, wroteBackup bool, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return false, false, fmt.Errorf("read: %w", err)
	}

	out, changed, err := insertTeamAttributionValidFrom(src, cutoffDate)
	if err != nil {
		return false, false, fmt.Errorf("insert: %w", err)
	}
	if !changed {
		return false, false, nil
	}
	if dryRun {
		return true, false, nil
	}

	wroteBackup, err = writeWithBackup(path, backupPath, out)
	if err != nil {
		return false, false, fmt.Errorf("write: %w", err)
	}
	return true, wroteBackup, nil
}

// insertTeamAttributionValidFrom returns src with `attributionValidFrom`
// inserted as the final top-level field (after the existing last field,
// before the closing `}` of the outer object). Field-order preservation is
// the whole reason this uses raw text manipulation rather than struct
// round-trip — operators reviewing the diff see exactly one new line.
//
// changed=false when the field is already present (idempotent no-op) OR the
// input is empty / not a JSON object (malformed inputs surface as errors).
func insertTeamAttributionValidFrom(src []byte, cutoffDate string) (out []byte, changed bool, err error) {
	if len(bytes.TrimSpace(src)) == 0 {
		return nil, false, errors.New("empty team.json")
	}

	var probe map[string]json.RawMessage
	if err := json.Unmarshal(src, &probe); err != nil {
		return nil, false, fmt.Errorf("not a JSON object: %w", err)
	}
	if _, exists := probe["attributionValidFrom"]; exists {
		return src, false, nil
	}

	// Preserve trailing-newline convention so we round-trip POSIX-style EOFs.
	trailingNL := ""
	if bytes.HasSuffix(src, []byte("\n")) {
		trailingNL = "\n"
	}

	trimmed := bytes.TrimRight(src, " \t\r\n")
	if len(trimmed) == 0 || trimmed[len(trimmed)-1] != '}' {
		return nil, false, errors.New("expected JSON to end with '}'")
	}

	// Drop the closing `}` and any whitespace immediately preceding it.
	body := bytes.TrimRight(trimmed[:len(trimmed)-1], " \t\r\n")

	// json.Marshal a string so the value is correctly quoted/escaped (the
	// cutoff-date validator already constrained the format, but escaping
	// stays correct under any future widening of the cutoff vocabulary).
	enc, _ := json.Marshal(cutoffDate)

	var buf bytes.Buffer
	buf.Grow(len(src) + 64)
	buf.Write(body)
	buf.WriteString(",\n  \"attributionValidFrom\": ")
	buf.Write(enc)
	buf.WriteString("\n}")
	buf.WriteString(trailingNL)
	return buf.Bytes(), true, nil
}

// migrateKnowledgeJSONL rewrites <path> in place, transforming every
// pre-cutoff entry to the post-cutoff KnowledgeEntry shape. Idempotent
// per-line: lines whose entries already carry an `attribution` field are
// passed through verbatim. Atomic file-level swap: writes <path>.tmp,
// renames <path> → <path>.backup (only on first migration), then renames
// <path>.tmp → <path>.
func migrateKnowledgeJSONL(path, backupPath string, dryRun bool) (scanned, migrated, skipped int, wroteBackup bool, err error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return 0, 0, 0, false, fmt.Errorf("read: %w", err)
	}

	// Empty file is a valid steady state (some teams have not yet produced
	// knowledge entries); nothing to do.
	if len(src) == 0 {
		return 0, 0, 0, false, nil
	}

	var outBuf bytes.Buffer
	outBuf.Grow(len(src))

	scanner := bufio.NewScanner(bytes.NewReader(src))
	scanner.Buffer(make([]byte, 0, 64*1024), 4*1024*1024) // tolerate large entries

	anyChange := false
	for scanner.Scan() {
		line := scanner.Bytes()

		// Preserve blank lines verbatim — a JSONL file may legitimately have
		// terminator newlines we should not drop.
		if len(bytes.TrimSpace(line)) == 0 {
			outBuf.Write(line)
			outBuf.WriteByte('\n')
			continue
		}

		scanned++
		newLine, lineChanged, lineErr := migrateKnowledgeLine(line)
		if lineErr != nil {
			return scanned, migrated, skipped, false, fmt.Errorf("line %d: %w", scanned, lineErr)
		}
		if lineChanged {
			migrated++
			anyChange = true
		} else {
			skipped++
		}
		outBuf.Write(newLine)
		outBuf.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return scanned, migrated, skipped, false, fmt.Errorf("scan: %w", err)
	}

	if !anyChange {
		return scanned, migrated, skipped, false, nil
	}
	if dryRun {
		return scanned, migrated, skipped, false, nil
	}

	wroteBackup, err = writeWithBackup(path, backupPath, outBuf.Bytes())
	if err != nil {
		return scanned, migrated, skipped, false, fmt.Errorf("write: %w", err)
	}
	return scanned, migrated, skipped, wroteBackup, nil
}

// migrateKnowledgeLine transforms one pre-cutoff knowledge.jsonl line into
// its post-cutoff shape. The contract pinned by
// store.TestKnowledgeEntry_LegacyMigrationFidelity:
//
//   - caller       = "legacy:" + <original by value>
//   - caller_note  = <original by value>
//   - attribution  = {kind: "legacy", spawn_origin: "legacy", others nil}
//
// Idempotent: lines that already carry `attribution` (with no `by` field)
// are passed through unchanged. Lines with neither `by` nor `attribution`
// are surfaced as errors — the caller can investigate; the migration
// refuses to silently corrupt data.
func migrateKnowledgeLine(line []byte) (out []byte, changed bool, err error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(line, &raw); err != nil {
		return nil, false, fmt.Errorf("parse JSON: %w", err)
	}

	_, hasBy := raw["by"]
	_, hasAttribution := raw["attribution"]

	if hasAttribution {
		// Already migrated. Leave the line verbatim even if `by` is also
		// somehow still present — the operator can reconcile manually if
		// they care; the migration tool's contract is "preserve migrated
		// state untouched".
		return line, false, nil
	}
	if !hasBy {
		return nil, false, errors.New("entry has neither 'by' nor 'attribution' — cannot derive legacy attribution")
	}

	var byValue string
	if err := json.Unmarshal(raw["by"], &byValue); err != nil {
		return nil, false, fmt.Errorf("decode 'by': %w", err)
	}

	// Round-trip through the canonical KnowledgeEntry: the `by` field has
	// no struct destination and is silently dropped; the new fields are
	// then populated and re-marshaled. This guarantees the output shape
	// matches the canon test (TestKnowledgeEntry_LegacyMigrationFidelity).
	var entry store.KnowledgeEntry
	if err := json.Unmarshal(line, &entry); err != nil {
		return nil, false, fmt.Errorf("decode entry: %w", err)
	}
	entry.Caller = "legacy:" + byValue
	entry.CallerNote = byValue
	entry.Attribution = store.AttributionInfo{
		Kind:        store.KnowledgeKindLegacy,
		SpawnOrigin: store.SpawnOriginLegacy,
	}

	out, err = json.Marshal(entry)
	if err != nil {
		return nil, false, fmt.Errorf("re-marshal: %w", err)
	}
	return out, true, nil
}

// writeWithBackup performs an atomic file replacement that preserves the
// original pre-migration content in a sibling `.backup` file. First
// invocation writes the backup; subsequent invocations leave the existing
// `.backup` alone (so the operator's "what was the original?" answer
// remains stable across re-runs).
//
// Steps:
//  1. Write content to `<path>.tmp`.
//  2. If `<path>.backup` does not exist yet: hard-link the original to
//     `<path>.backup` (or copy on file systems where Link is unavailable).
//  3. Rename `<path>.tmp` to `<path>` (atomic on POSIX).
func writeWithBackup(path, backupPath string, content []byte) (wroteBackup bool, err error) {
	tmpPath := path + ".tmp"

	// Always clean up the temp file on error — there's no scenario where a
	// stale .tmp is useful to keep around.
	defer func() {
		if err != nil {
			_ = os.Remove(tmpPath)
		}
	}()

	if err := os.WriteFile(tmpPath, content, 0o644); err != nil {
		return false, fmt.Errorf("write tmp: %w", err)
	}

	// Backup is one-shot: only created on first migration of this file.
	if _, statErr := os.Stat(backupPath); errors.Is(statErr, os.ErrNotExist) {
		if err := os.MkdirAll(filepath.Dir(backupPath), 0o755); err != nil {
			return false, fmt.Errorf("create backup dir: %w", err)
		}
		if err := copyRegularFile(path, backupPath); err != nil {
			return false, fmt.Errorf("write backup: %w", err)
		}
		wroteBackup = true
	} else if statErr != nil {
		return false, fmt.Errorf("stat backup: %w", statErr)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return wroteBackup, fmt.Errorf("rename tmp -> path: %w", err)
	}
	return wroteBackup, nil
}

// copyRegularFile copies src to dst by streaming bytes; preserves the
// source mode bits. Refuses to clobber an existing dst.
func copyRegularFile(src, dst string) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()

	info, err := in.Stat()
	if err != nil {
		return err
	}

	out, err := os.OpenFile(dst, os.O_WRONLY|os.O_CREATE|os.O_EXCL, info.Mode().Perm())
	if err != nil {
		return err
	}
	defer func() {
		_ = out.Close()
	}()

	if _, err := io.Copy(out, in); err != nil {
		_ = os.Remove(dst)
		return err
	}
	return out.Close()
}

// stripWhitespace is a small helper used in tests for normalized comparisons.
// Kept here (not in the test file) so it can be reused by future tooling.
func stripWhitespace(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range s {
		if r == ' ' || r == '\t' || r == '\n' || r == '\r' {
			continue
		}
		b.WriteRune(r)
	}
	return b.String()
}
