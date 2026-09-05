package baseline

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/bmatcuk/doublestar/v4"
)

const (
	PathSnapshotSchemaVersion = 1
	// PathSnapshotPolicyVersion identifies the semantics used for a newly
	// captured snapshot independently from the storage schema. Zero remains a
	// readable legacy value; version one is the Git-aware metadata-default
	// policy introduced with the estimate API.
	PathSnapshotPolicyVersion = 1
	maxSnapshotFileBytes      = 1 << 20 // 1 MiB; larger files are metadata only.
	maxSnapshotObjectBytes    = 8 << 20 // A single snapshot may retain at most 8 MiB of text.
	defaultPathSnapshotLease  = 7 * 24 * time.Hour
)

// PathSnapshotPolicy is explicit capture policy. New snapshots are metadata
// only by default: a digest is evidence enough to compare a file later, while
// retaining its body is an intentionally separate privacy and quota decision.
type PathSnapshotPolicy struct {
	IncludeIgnored bool `json:"include_ignored,omitempty"`
	RetainContent  bool `json:"retain_content,omitempty"`
}

// PathSnapshotEstimate is the authoritative preflight result used by both
// direct callers and collection capture. Source evidence remains
// informational; this report never changes Test Genie behavior.
type PathSnapshotEstimate struct {
	PolicyVersion          int
	Selections             []string
	Policy                 PathSnapshotPolicy
	EligibleFiles          int
	EligibleBytes          int64
	ExcludedIgnoredFiles   int
	ExcludedIgnoredBytes   int64
	ExcludedSensitiveFiles int
	ExcludedBinaryFiles    int
	OversizedFiles         int
	RetainedContentBytes   int64
	TopContributors        []PathSnapshotContributor
	Issues                 []PathSnapshotIssue
	Recommendations        []PathSnapshotRecommendation
	entries                []resolvedPath
}

type PathSnapshotContributor struct {
	Path  string
	Files int
	Bytes int64
}

type PathSnapshotIssue struct {
	Code     string
	Severity string // info | warning | repair-required | error
	Detail   string
}

type PathSnapshotRecommendation struct {
	Selection string
	Reason    string
}

// PathSnapshotPolicyError is returned before any durable record is written
// when an estimate says that a capture must be narrowed. Keeping the full
// estimate allows both in-process callers and the RPC edge to give the author
// the same actionable repair report.
type PathSnapshotPolicyError struct{ Estimate PathSnapshotEstimate }

func (e *PathSnapshotPolicyError) Error() string {
	for _, issue := range e.Estimate.Issues {
		if issue.Severity == "repair-required" || issue.Severity == "error" {
			return "path snapshot selection requires repair: " + issue.Detail
		}
	}
	return "path snapshot selection requires repair"
}

func (e PathSnapshotEstimate) RequiresRepair() bool {
	for _, issue := range e.Issues {
		if issue.Severity == "repair-required" || issue.Severity == "error" {
			return true
		}
	}
	return false
}

type resolvedPath struct {
	entry PathEntry
	data  []byte
}

type PathEntryState string

const (
	PathEntryPresent    PathEntryState = "present"
	PathEntryExcluded   PathEntryState = "excluded"
	PathEntryUnreadable PathEntryState = "unreadable"
)

// PathSnapshot is immutable source evidence. It is not a Test Genie result and
// must always be rendered/consumed as informational.
type PathSnapshot struct {
	Name          string             `json:"name"`
	Branch        string             `json:"branch"`
	CreatedAt     time.Time          `json:"created_at"`
	ExpiresAt     time.Time          `json:"expires_at"`
	SchemaVersion int                `json:"schema_version"`
	PolicyVersion int                `json:"policy_version,omitempty"`
	Selections    []string           `json:"selections"`
	Policy        PathSnapshotPolicy `json:"policy,omitempty"`
	Entries       []PathEntry        `json:"entries"`
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
	if s.PolicyVersion != 0 && s.PolicyVersion != PathSnapshotPolicyVersion {
		return fmt.Errorf("unsupported path snapshot policy version %d", s.PolicyVersion)
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
	return CapturePathSnapshotWithPolicyAndLease(root, name, branch, selections, PathSnapshotPolicy{}, now, defaultPathSnapshotLease)
}

// CapturePathSnapshotWithLease captures source evidence under an explicit,
// bounded retention lease. Callers cannot retain bytes indefinitely by omitting
// policy; the default wrapper above uses the safe seven-day lease.
func CapturePathSnapshotWithLease(root, name, branch string, selections []string, now time.Time, lease time.Duration) (PathSnapshot, map[string][]byte, error) {
	return CapturePathSnapshotWithPolicyAndLease(root, name, branch, selections, PathSnapshotPolicy{}, now, lease)
}

// CapturePathSnapshotWithPolicyAndLease persists the exact eligible set from
// the resolver. Callers that want historical body retention must say so.
func CapturePathSnapshotWithPolicyAndLease(root, name, branch string, selections []string, policy PathSnapshotPolicy, now time.Time, lease time.Duration) (PathSnapshot, map[string][]byte, error) {
	if strings.TrimSpace(root) == "" || strings.TrimSpace(name) == "" || strings.TrimSpace(branch) == "" {
		return PathSnapshot{}, nil, fmt.Errorf("path snapshot root, name, and branch are required")
	}
	if lease <= 0 || lease > 30*24*time.Hour {
		return PathSnapshot{}, nil, fmt.Errorf("path snapshot retention lease must be between one nanosecond and 30 days")
	}
	estimate, err := EstimatePathSnapshot(root, selections, policy)
	if err != nil {
		return PathSnapshot{}, nil, err
	}
	if estimate.RequiresRepair() {
		return PathSnapshot{}, nil, &PathSnapshotPolicyError{Estimate: estimate}
	}
	created := now.UTC()
	snapshot := PathSnapshot{Name: strings.TrimSpace(name), Branch: strings.TrimSpace(branch), CreatedAt: created, ExpiresAt: created.Add(lease), SchemaVersion: PathSnapshotSchemaVersion, PolicyVersion: PathSnapshotPolicyVersion, Selections: estimate.Selections, Policy: policy}
	objects := map[string][]byte{}
	for _, resolved := range estimate.entries {
		entry := resolved.entry
		if policy.RetainContent && entry.Type == "file" && entry.Size <= maxSnapshotFileBytes && !isBinary(resolved.data) {
			entry.ContentRef = entry.Digest
			objects[entry.ContentRef] = resolved.data
		}
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

// EstimatePathSnapshot resolves tracked files and non-ignored untracked files
// through Git before touching content. Git failure is explicit: falling back to
// a raw walk would silently reintroduce ignored dependency capture.
func EstimatePathSnapshot(root string, selections []string, policy PathSnapshotPolicy) (PathSnapshotEstimate, error) {
	patterns, err := normalizeSnapshotSelections(selections)
	if err != nil {
		return PathSnapshotEstimate{}, err
	}
	tracked, err := gitPathSet(root, "ls-files", "-z", "--cached")
	if err != nil {
		return PathSnapshotEstimate{}, fmt.Errorf("enumerate tracked source evidence paths: %w", err)
	}
	untracked, err := gitPathSet(root, "ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return PathSnapshotEstimate{}, fmt.Errorf("enumerate untracked source evidence paths: %w", err)
	}
	ignored, err := gitPathSet(root, "ls-files", "-z", "--others", "--ignored", "--exclude-standard")
	if err != nil {
		return PathSnapshotEstimate{}, fmt.Errorf("enumerate ignored source evidence paths: %w", err)
	}
	eligible := make(map[string]struct{}, len(tracked)+len(untracked))
	for path := range tracked {
		eligible[path] = struct{}{}
	}
	for path := range untracked {
		eligible[path] = struct{}{}
	}
	if policy.IncludeIgnored {
		for path := range ignored {
			eligible[path] = struct{}{}
		}
	}
	estimate := PathSnapshotEstimate{PolicyVersion: PathSnapshotPolicyVersion, Selections: patterns, Policy: policy}
	for _, path := range sortedPaths(eligible) {
		if !matchesPath(patterns, path) {
			continue
		}
		if snapshotPathDenied(path) {
			estimate.ExcludedSensitiveFiles++
			continue
		}
		entry, data, include := capturePathEntry(root, path)
		if !include || entry.State != PathEntryPresent {
			continue
		}
		estimate.EligibleFiles++
		estimate.EligibleBytes += entry.Size
		if entry.Type == "binary" {
			estimate.ExcludedBinaryFiles++
		}
		if entry.Size > maxSnapshotFileBytes {
			estimate.OversizedFiles++
		}
		if policy.RetainContent && entry.Type == "file" && entry.Size <= maxSnapshotFileBytes && !isBinary(data) {
			estimate.RetainedContentBytes += int64(len(data))
		}
		estimate.entries = append(estimate.entries, resolvedPath{entry: entry, data: data})
	}
	if !policy.IncludeIgnored {
		for _, path := range sortedPaths(ignored) {
			if !matchesPath(patterns, path) {
				continue
			}
			info, statErr := os.Lstat(filepath.Join(root, filepath.FromSlash(path)))
			if statErr == nil && !info.IsDir() {
				estimate.ExcludedIgnoredFiles++
				estimate.ExcludedIgnoredBytes += info.Size()
			}
		}
	}
	estimate.TopContributors = contributors(estimate.entries)
	addEstimateIssues(&estimate)
	return estimate, nil
}

func gitPathSet(root string, args ...string) (map[string]struct{}, error) {
	cmd := exec.Command("git", append([]string{"--no-optional-locks", "-C", root}, args...)...)
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	paths := map[string]struct{}{}
	for _, raw := range bytes.Split(out, []byte{0}) {
		path := filepath.ToSlash(strings.TrimSpace(string(raw)))
		if path != "" {
			paths[path] = struct{}{}
		}
	}
	return paths, nil
}

func sortedPaths(set map[string]struct{}) []string {
	out := make([]string, 0, len(set))
	for path := range set {
		out = append(out, path)
	}
	sort.Strings(out)
	return out
}

func matchesPath(patterns []string, path string) bool {
	for _, pattern := range patterns {
		if ok, _ := doublestar.Match(pattern, path); ok {
			return true
		}
	}
	return false
}

func isBinary(data []byte) bool { return bytes.IndexByte(data, 0) >= 0 }

func contributors(entries []resolvedPath) []PathSnapshotContributor {
	byPath := map[string]*PathSnapshotContributor{}
	for _, resolved := range entries {
		parts := strings.Split(resolved.entry.Path, "/")
		key := parts[0]
		if len(parts) > 1 {
			key += "/" + parts[1]
		}
		if byPath[key] == nil {
			byPath[key] = &PathSnapshotContributor{Path: key}
		}
		byPath[key].Files++
		byPath[key].Bytes += resolved.entry.Size
	}
	out := make([]PathSnapshotContributor, 0, len(byPath))
	for _, item := range byPath {
		out = append(out, *item)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Bytes == out[j].Bytes {
			return out[i].Path < out[j].Path
		}
		return out[i].Bytes > out[j].Bytes
	})
	if len(out) > 8 {
		out = out[:8]
	}
	return out
}

func addEstimateIssues(estimate *PathSnapshotEstimate) {
	if estimate.Policy.RetainContent && estimate.RetainedContentBytes > maxSnapshotObjectBytes {
		estimate.Issues = append(estimate.Issues, PathSnapshotIssue{Code: "retained_content_over_budget", Severity: "error", Detail: fmt.Sprintf("retained content projects %d bytes, above the %d-byte per-snapshot limit", estimate.RetainedContentBytes, maxSnapshotObjectBytes)})
		estimate.Recommendations = append(estimate.Recommendations, contributorSelections(estimate.entries)...)
	}
	for _, selection := range estimate.Selections {
		if selection == "packages/proto/gen/**" {
			estimate.Issues = append(estimate.Issues, PathSnapshotIssue{Code: "generated_output_too_broad", Severity: "repair-required", Detail: "all generated proto output is broader than the affected namespaces"})
			estimate.Recommendations = append(estimate.Recommendations, generatedNamespaceSelections(estimate.entries)...)
		}
		if selection == "scenarios/**" {
			estimate.Issues = append(estimate.Issues, PathSnapshotIssue{Code: "scenarios_too_broad", Severity: "repair-required", Detail: "all scenarios are broader than the affected scenario boundary"})
			estimate.Recommendations = append(estimate.Recommendations, scenarioSelections(estimate.entries)...)
		}
	}
	if estimate.ExcludedIgnoredFiles > 0 {
		estimate.Issues = append(estimate.Issues, PathSnapshotIssue{Code: "ignored_files_excluded", Severity: "info", Detail: fmt.Sprintf("%d ignored files excluded by default", estimate.ExcludedIgnoredFiles)})
	}
	if estimate.Policy.IncludeIgnored {
		estimate.Issues = append(estimate.Issues, PathSnapshotIssue{Code: "ignored_files_included", Severity: "warning", Detail: "ignored files were explicitly included"})
	}
}

func generatedNamespaceSelections(entries []resolvedPath) []PathSnapshotRecommendation {
	return uniqueRecommendations(entries, func(path string) (string, bool) {
		parts := strings.Split(path, "/")
		if len(parts) < 5 || strings.Join(parts[:3], "/") != "packages/proto/gen" {
			return "", false
		}
		if parts[3] == "manifests" {
			return path, true
		}
		return strings.Join(parts[:5], "/") + "/**", true
	}, "select only this generated language/namespace present in the estimate")
}

func scenarioSelections(entries []resolvedPath) []PathSnapshotRecommendation {
	return uniqueRecommendations(entries, func(path string) (string, bool) {
		parts := strings.Split(path, "/")
		if len(parts) < 3 || parts[0] != "scenarios" {
			return "", false
		}
		return "scenarios/" + parts[1] + "/**", true
	}, "select this affected scenario present in the estimate")
}

func contributorSelections(entries []resolvedPath) []PathSnapshotRecommendation {
	return uniqueRecommendations(entries, func(path string) (string, bool) {
		parts := strings.Split(path, "/")
		if len(parts) == 1 {
			return path, true
		}
		return strings.Join(parts[:2], "/") + "/**", true
	}, "narrow retained-content capture to this estimated contributor, or keep metadata-only mode")
}

func uniqueRecommendations(entries []resolvedPath, selection func(string) (string, bool), reason string) []PathSnapshotRecommendation {
	seen := map[string]struct{}{}
	var out []PathSnapshotRecommendation
	for _, entry := range entries {
		candidate, ok := selection(entry.entry.Path)
		if !ok {
			continue
		}
		if _, exists := seen[candidate]; exists {
			continue
		}
		seen[candidate] = struct{}{}
		out = append(out, PathSnapshotRecommendation{Selection: candidate, Reason: reason})
		if len(out) == 8 {
			break
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Selection < out[j].Selection })
	return out
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
