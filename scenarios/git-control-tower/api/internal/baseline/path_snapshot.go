package baseline

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	PathSnapshotSchemaVersion = 1
	maxSnapshotFileBytes      = 1 << 20 // 1 MiB; larger files are metadata only.
	maxSnapshotObjectBytes    = 8 << 20 // A single snapshot may retain at most 8 MiB of text.
	defaultPathSnapshotLease  = 7 * 24 * time.Hour
)

type PathEntryState string

const (
	PathEntryPresent    PathEntryState = "present"
	PathEntryExcluded   PathEntryState = "excluded"
	PathEntryUnreadable PathEntryState = "unreadable"
)

// PathSnapshot is immutable source evidence. It is not a Test Genie result and
// must always be rendered/consumed as informational.
type PathSnapshot struct {
	Name          string      `json:"name"`
	Branch        string      `json:"branch"`
	CreatedAt     time.Time   `json:"created_at"`
	ExpiresAt     time.Time   `json:"expires_at"`
	SchemaVersion int         `json:"schema_version"`
	Selections    []string    `json:"selections"`
	Entries       []PathEntry `json:"entries"`
}

type PathEntry struct {
	Path       string         `json:"path"`
	Mode       fs.FileMode    `json:"mode"`
	Type       string         `json:"type"`
	Size       int64          `json:"size"`
	Digest     string         `json:"digest,omitempty"`
	ContentRef string         `json:"content_ref,omitempty"`
	State      PathEntryState `json:"state"`
	Detail     string         `json:"detail,omitempty"`
}

type SourceDelta struct {
	Path   string
	Status string // added | deleted | modified | unchanged | excluded | unreadable
	Before *PathEntry
	After  *PathEntry
}

// Validate protects the durable manifest boundary. Content bytes are stored
// separately and are accepted only when their digest matches their reference.
func (s PathSnapshot) Validate() error {
	if strings.TrimSpace(s.Name) == "" || strings.TrimSpace(s.Branch) == "" {
		return fmt.Errorf("path snapshot name and branch are required")
	}
	if s.SchemaVersion != PathSnapshotSchemaVersion {
		return fmt.Errorf("unsupported path snapshot schema version %d", s.SchemaVersion)
	}
	if s.CreatedAt.IsZero() || s.ExpiresAt.IsZero() || !s.ExpiresAt.After(s.CreatedAt) {
		return fmt.Errorf("path snapshot requires a future retention expiry")
	}
	if _, err := normalizeSnapshotSelections(s.Selections); err != nil {
		return err
	}
	seen := make(map[string]struct{}, len(s.Entries))
	for _, entry := range s.Entries {
		if entry.Path == "" || filepath.IsAbs(entry.Path) || strings.Contains(filepath.ToSlash(entry.Path), "../") {
			return fmt.Errorf("unsafe snapshot entry path %q", entry.Path)
		}
		if _, duplicate := seen[entry.Path]; duplicate {
			return fmt.Errorf("duplicate snapshot entry path %q", entry.Path)
		}
		seen[entry.Path] = struct{}{}
		if entry.State == PathEntryPresent && entry.Type == "file" && entry.Digest == "" {
			return fmt.Errorf("present file %q is missing digest", entry.Path)
		}
		if entry.ContentRef != "" && entry.ContentRef != entry.Digest {
			return fmt.Errorf("snapshot entry %q content reference must equal digest", entry.Path)
		}
	}
	return nil
}

var deniedSnapshotPathPrefixes = []string{".git/", ".env", "secrets/", "credentials/"}

// CapturePathSnapshot captures the actual current tree, including dirty files.
// It rejects traversal/absolute patterns, excludes sensitive paths and symlinks,
// and only retains bytes for bounded text files. Callers can compare exact
// digests later without treating the result as behavioral evidence.
func CapturePathSnapshot(root, name, branch string, selections []string, now time.Time) (PathSnapshot, map[string][]byte, error) {
	return CapturePathSnapshotWithLease(root, name, branch, selections, now, defaultPathSnapshotLease)
}

// CapturePathSnapshotWithLease captures source evidence under an explicit,
// bounded retention lease. Callers cannot retain bytes indefinitely by omitting
// policy; the default wrapper above uses the safe seven-day lease.
func CapturePathSnapshotWithLease(root, name, branch string, selections []string, now time.Time, lease time.Duration) (PathSnapshot, map[string][]byte, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(branch) == "" {
		return PathSnapshot{}, nil, fmt.Errorf("path snapshot root, name, and branch are required")
	}
	patterns, err := normalizeSnapshotSelections(selections)
	if err != nil {
		return PathSnapshot{}, nil, err
	}
	if lease <= 0 || lease > 30*24*time.Hour {
		return PathSnapshot{}, nil, fmt.Errorf("path snapshot retention lease must be between one nanosecond and 30 days")
	}
	created := now.UTC()
	snapshot := PathSnapshot{Name: strings.TrimSpace(name), Branch: strings.TrimSpace(branch), CreatedAt: created, ExpiresAt: created.Add(lease), SchemaVersion: PathSnapshotSchemaVersion, Selections: patterns}
	objects := map[string][]byte{}
	entries := map[string]PathEntry{}
	for _, pattern := range patterns {
		matches, err := doublestar.Glob(os.DirFS(root), pattern)
		if err != nil {
			return PathSnapshot{}, nil, fmt.Errorf("expand path selection %q: %w", pattern, err)
		}
		for _, match := range matches {
			match = filepath.ToSlash(match)
			if _, seen := entries[match]; seen {
				continue
			}
			entry, bytes, include := capturePathEntry(root, match)
			if !include {
				continue
			}
			entries[match] = entry
			if entry.ContentRef != "" {
				objects[entry.ContentRef] = bytes
			}
		}
	}
	for _, entry := range entries {
		snapshot.Entries = append(snapshot.Entries, entry)
	}
	sort.Slice(snapshot.Entries, func(i, j int) bool { return snapshot.Entries[i].Path < snapshot.Entries[j].Path })
	var retained int
	for _, object := range objects {
		retained += len(object)
	}
	if retained > maxSnapshotObjectBytes {
		return PathSnapshot{}, nil, fmt.Errorf("path snapshot retained content exceeds %d-byte limit", maxSnapshotObjectBytes)
	}
	if err := snapshot.Validate(); err != nil {
		return PathSnapshot{}, nil, err
	}
	return snapshot, objects, nil
}

func normalizeSnapshotSelections(selections []string) ([]string, error) {
	seen := map[string]struct{}{}
	out := make([]string, 0, len(selections))
	for _, selection := range selections {
		selection = filepath.ToSlash(strings.TrimSpace(selection))
		if selection == "" || filepath.IsAbs(selection) || strings.Contains(selection, "../") || selection == ".." {
			return nil, fmt.Errorf("unsafe path selection %q", selection)
		}
		if snapshotPathDenied(strings.TrimSuffix(selection, "**")) {
			return nil, fmt.Errorf("path selection %q is denied by source-evidence safety policy", selection)
		}
		if _, exists := seen[selection]; !exists {
			seen[selection] = struct{}{}
			out = append(out, selection)
		}
	}
	if len(out) == 0 {
		return nil, fmt.Errorf("path snapshot requires at least one selection")
	}
	sort.Strings(out)
	return out, nil
}

func snapshotPathDenied(path string) bool {
	path = strings.TrimPrefix(filepath.ToSlash(path), "./")
	for _, prefix := range deniedSnapshotPathPrefixes {
		if path == strings.TrimSuffix(prefix, "/") || strings.HasPrefix(path, prefix) || strings.Contains(path, "/"+prefix) {
			return true
		}
	}
	return false
}

func capturePathEntry(root, rel string) (PathEntry, []byte, bool) {
	if snapshotPathDenied(rel) {
		return PathEntry{Path: rel, State: PathEntryExcluded, Detail: "denied by source-evidence safety policy"}, nil, true
	}
	full := filepath.Join(root, filepath.FromSlash(rel))
	info, err := os.Lstat(full)
	if err != nil {
		return PathEntry{Path: rel, State: PathEntryUnreadable, Detail: err.Error()}, nil, true
	}
	if info.IsDir() {
		return PathEntry{}, nil, false
	}
	entry := PathEntry{Path: rel, Mode: info.Mode(), Size: info.Size(), State: PathEntryPresent, Type: "file"}
	if info.Mode()&os.ModeSymlink != 0 {
		entry.Type, entry.State, entry.Detail = "symlink", PathEntryExcluded, "symlinks are not retained"
		return entry, nil, true
	}
	data, err := os.ReadFile(full)
	if err != nil {
		entry.State, entry.Detail = PathEntryUnreadable, err.Error()
		return entry, nil, true
	}
	digest := sha256.Sum256(data)
	entry.Digest = hex.EncodeToString(digest[:])
	if len(data) > maxSnapshotFileBytes {
		entry.Detail = "oversized file retained as metadata only"
		return entry, nil, true
	}
	if bytes.IndexByte(data, 0) >= 0 {
		entry.Type, entry.Detail = "binary", "binary file retained as metadata only"
		return entry, nil, true
	}
	entry.ContentRef = entry.Digest
	return entry, data, true
}

func DiffPathSnapshots(before, after PathSnapshot) []SourceDelta {
	left, right := map[string]PathEntry{}, map[string]PathEntry{}
	for _, entry := range before.Entries {
		left[entry.Path] = entry
	}
	for _, entry := range after.Entries {
		right[entry.Path] = entry
	}
	// A unique deleted/added file pair with the same retained digest is a
	// rename, not two unrelated source changes. Ambiguous duplicates remain
	// explicit add/delete entries so this informational view never guesses.
	deletedByDigest, addedByDigest := map[string][]string{}, map[string][]string{}
	for path, entry := range left {
		if _, stillPresent := right[path]; !stillPresent && entry.State == PathEntryPresent && entry.Type == "file" && entry.Digest != "" {
			deletedByDigest[entry.Digest] = append(deletedByDigest[entry.Digest], path)
		}
	}
	for path, entry := range right {
		if _, existed := left[path]; !existed && entry.State == PathEntryPresent && entry.Type == "file" && entry.Digest != "" {
			addedByDigest[entry.Digest] = append(addedByDigest[entry.Digest], path)
		}
	}
	renamedTo, renamedFrom := map[string]string{}, map[string]string{}
	for digest, deleted := range deletedByDigest {
		added := addedByDigest[digest]
		if len(deleted) == 1 && len(added) == 1 {
			renamedTo[added[0]] = deleted[0]
			renamedFrom[deleted[0]] = added[0]
		}
	}
	paths := map[string]struct{}{}
	for path := range left {
		paths[path] = struct{}{}
	}
	for path := range right {
		paths[path] = struct{}{}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	sort.Strings(ordered)
	out := make([]SourceDelta, 0, len(ordered))
	for _, path := range ordered {
		if _, isRenameSource := renamedFrom[path]; isRenameSource {
			continue
		}
		beforeEntry, beforeOK := left[path]
		afterEntry, afterOK := right[path]
		if oldPath, isRenameTarget := renamedTo[path]; isRenameTarget {
			beforeEntry, beforeOK = left[oldPath], true
		}
		delta := SourceDelta{Path: path}
		if beforeOK {
			delta.Before = &beforeEntry
		}
		if afterOK {
			delta.After = &afterEntry
		}
		switch {
		case renamedTo[path] != "":
			delta.Status = "renamed"
		case !beforeOK:
			delta.Status = "added"
		case !afterOK:
			delta.Status = "deleted"
		case beforeEntry.State == PathEntryExcluded || afterEntry.State == PathEntryExcluded:
			delta.Status = "excluded"
		case beforeEntry.State == PathEntryUnreadable || afterEntry.State == PathEntryUnreadable:
			delta.Status = "unreadable"
		case beforeEntry.Digest != afterEntry.Digest || beforeEntry.Mode != afterEntry.Mode:
			delta.Status = "modified"
		default:
			delta.Status = "unchanged"
		}
		out = append(out, delta)
	}
	return out
}

// FilterSourceDeltas narrows an already-computed informational diff to safe
// repo-relative glob selections. A rename matches either its destination or
// source path so a phase cannot lose evidence merely because a file moved.
func FilterSourceDeltas(deltas []SourceDelta, selections []string) ([]SourceDelta, error) {
	if len(selections) == 0 {
		return deltas, nil
	}
	patterns, err := normalizeSnapshotSelections(selections)
	if err != nil {
		return nil, err
	}
	out := make([]SourceDelta, 0, len(deltas))
	for _, delta := range deltas {
		if sourceDeltaMatches(patterns, delta) {
			out = append(out, delta)
		}
	}
	return out, nil
}

func sourceDeltaMatches(patterns []string, delta SourceDelta) bool {
	paths := []string{delta.Path}
	if delta.Before != nil {
		paths = append(paths, delta.Before.Path)
	}
	for _, pattern := range patterns {
		for _, path := range paths {
			if matched, err := doublestar.Match(pattern, filepath.ToSlash(path)); err == nil && matched {
				return true
			}
		}
	}
	return false
}
