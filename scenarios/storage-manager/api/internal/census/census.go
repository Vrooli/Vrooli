// Package census measures the storage surface without mutating it.
package census

import (
	"context"
	"crypto/sha256"
	"database/sql"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/vrooli/api-core/database"
	corestorage "github.com/vrooli/api-core/storage"
)

//go:embed schema.sql
var censusSchemaSQL string

type Entry struct {
	Owner    string `json:"owner"`
	Kind     string `json:"kind,omitempty"`
	Name     string `json:"name"`
	Path     string `json:"path"`
	Bytes    int64  `json:"bytes"`
	Declared bool   `json:"declared"`
}

type Finding struct {
	Code     string `json:"code"`
	Severity string `json:"severity"`
	Owner    string `json:"owner,omitempty"`
	Kind     string `json:"kind,omitempty"`
	Path     string `json:"path,omitempty"`
	Message  string `json:"message"`
}

type Report struct {
	SnapshotID              string         `json:"snapshot_id,omitempty"`
	ObservedAt              time.Time      `json:"observed_at,omitempty"`
	Root                    string         `json:"root"`
	MeasuredBytes           int64          `json:"measured_bytes"`
	AttributedBytes         int64          `json:"attributed_bytes"`
	UnattributedBytes       int64          `json:"unattributed_bytes"`
	Closed                  bool           `json:"closed"`
	AccountingIdentity      bool           `json:"accounting_identity"`
	Confidence              string         `json:"confidence"`
	GrowthSlopeBytesPerHour *float64       `json:"growth_slope_bytes_per_hour,omitempty"`
	OwnerCounts             map[string]int `json:"owner_counts,omitempty"`
	UnreadablePaths         []string       `json:"unreadable_paths,omitempty"`
	Findings                []Finding      `json:"findings,omitempty"`
	Entries                 []Entry        `json:"entries"`
	FrameworkRoots          []string       `json:"framework_roots,omitempty"`
}

func (r Report) MarshalJSON() ([]byte, error) { type alias Report; return json.Marshal(alias(r)) }

type Declaration struct {
	Name     string
	Path     string
	Budgeted bool
}

// Scan walks root and attributes each file to at most one declaration. It
// never creates, removes, renames, or opens a database for writing.
func Scan(root string, manifests map[string][]Declaration) (Report, error) {
	declarations := make([]resolvedDeclaration, 0)
	owners := make([]string, 0, len(manifests))
	for owner := range manifests {
		owners = append(owners, owner)
	}
	sort.Strings(owners)
	for _, owner := range owners {
		for _, declaration := range manifests[owner] {
			declarations = append(declarations, resolvedDeclaration{owner: owner, name: declaration.Name, path: declaration.Path})
		}
	}
	return scan(root, declarations, nil)
}

// ScanInventory measures all declarations loaded from scenarios, resources,
// tools, and safeguards. Relative scenario paths are rooted at the scenario;
// other relative paths are rooted beside their native manifest.
func ScanInventory(root string, inventory corestorage.OwnerInventory) (Report, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Report{}, fmt.Errorf("census: resolve root: %w", err)
	}
	declarations := make([]resolvedDeclaration, 0)
	boundedRoots := make([]string, 0)
	counts := map[string]int{}
	findings := make([]Finding, 0, len(inventory.Findings))
	for _, finding := range inventory.Findings {
		findings = append(findings, Finding{Code: finding.Code, Severity: finding.Severity, Owner: finding.OwnerID, Kind: string(finding.OwnerKind), Path: finding.ManifestPath, Message: finding.Message})
	}
	for _, owner := range inventory.Owners {
		kind := string(owner.Kind)
		counts[kind]++
		for _, declaration := range owner.StorageEntries {
			path, resolveErr := corestorage.ResolveOwnerStoragePath(root, owner, declaration, corestorage.Platform(runtime.GOOS), corestorage.PlatformSeams{})
			if resolveErr != nil {
				findings = append(findings, Finding{Code: "unresolvable_storage_path", Severity: "error", Owner: owner.ID, Kind: kind, Path: owner.ManifestPath, Message: declaration.Name + ": " + resolveErr.Error()})
				continue
			}
			declarations = append(declarations, resolvedDeclaration{owner: owner.ID, kind: kind, name: declaration.Name, path: path})
			boundedRoots = append(boundedRoots, path)
		}
		if owner.Kind == corestorage.OwnerScenario {
			// Candidate roots make missing and explicit-empty scenario declarations
			// observable without walking source, dependency, or VCS trees.
			boundedRoots = append(boundedRoots, scenarioCandidateRoots(root, owner)...)
		}
	}
	report, err := scanRoots(root, boundedRoots, declarations, findings)
	if err != nil {
		return Report{}, err
	}
	report.OwnerCounts = counts
	report.FrameworkRoots = existingRoots(boundedRoots)
	return report, nil
}

type resolvedDeclaration struct {
	owner string
	kind  string
	name  string
	path  string
}

func scan(root string, declarations []resolvedDeclaration, initialFindings []Finding) (Report, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return Report{}, err
	}
	return scanRoots(root, []string{root}, declarations, initialFindings)
}

func scanRoots(displayRoot string, roots []string, declarations []resolvedDeclaration, initialFindings []Finding) (Report, error) {
	displayRoot, err := filepath.Abs(displayRoot)
	if err != nil {
		return Report{}, err
	}
	files := make([]fileObservation, 0)
	unreadable := make([]string, 0)
	seenRoots := make(map[string]struct{}, len(roots))
	seenFiles := make(map[string]struct{})
	findings := append([]Finding(nil), initialFindings...)
	for _, rawRoot := range roots {
		root, absErr := filepath.Abs(rawRoot)
		if absErr != nil {
			return Report{}, absErr
		}
		root = filepath.Clean(root)
		if _, exists := seenRoots[root]; exists {
			continue
		}
		seenRoots[root] = struct{}{}
		if _, statErr := os.Stat(root); statErr != nil {
			if os.IsNotExist(statErr) {
				findings = append(findings, Finding{Code: "missing_scan_root", Severity: "warning", Path: root, Message: "declared or candidate storage root does not exist"})
				continue
			}
			findings = append(findings, Finding{Code: "unreadable_path", Severity: "error", Path: root, Message: statErr.Error()})
			unreadable = append(unreadable, root)
			continue
		}
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				unreadable = append(unreadable, path)
				if d != nil && d.IsDir() {
					return fs.SkipDir
				}
				return nil
			}
			if d.IsDir() {
				return nil
			}
			info, statErr := d.Info()
			if statErr != nil {
				unreadable = append(unreadable, path)
				return nil
			}
			if _, exists := seenFiles[path]; exists {
				return nil
			}
			seenFiles[path] = struct{}{}
			files = append(files, fileObservation{path: path, bytes: info.Size()})
			return nil
		})
		if walkErr != nil {
			return Report{}, fmt.Errorf("scan %s: %w", root, walkErr)
		}
	}
	sort.Slice(files, func(i, j int) bool { return files[i].path < files[j].path })
	sort.Slice(declarations, func(i, j int) bool {
		if declarations[i].path != declarations[j].path {
			return declarations[i].path < declarations[j].path
		}
		if declarations[i].owner != declarations[j].owner {
			return declarations[i].owner < declarations[j].owner
		}
		return declarations[i].name < declarations[j].name
	})
	declarationByPath := make(map[string][]int, len(declarations))
	for index, declaration := range declarations {
		declarationByPath[filepath.Clean(declaration.path)] = append(declarationByPath[filepath.Clean(declaration.path)], index)
	}

	entries := make([]Entry, len(declarations))
	for i, declaration := range declarations {
		entries[i] = Entry{Owner: declaration.owner, Kind: declaration.kind, Name: declaration.name, Path: declaration.path, Declared: true}
	}
	var measured, attributed int64
	for _, file := range files {
		measured += file.bytes
		matches := declarationMatches(file.path, declarationByPath)
		if len(matches) == 0 {
			continue
		}
		// Assign a file once to keep the accounting identity exact. The overlap
		// finding preserves the fact that more than one declaration matched.
		chosen := matches[0]
		entries[chosen].Bytes += file.bytes
		attributed += file.bytes
		if len(matches) > 1 {
			for _, index := range matches[1:] {
				findings = append(findings, Finding{Code: "overlap", Severity: "warning", Owner: entries[index].Owner, Kind: entries[index].Kind, Path: file.path, Message: fmt.Sprintf("file also matched declaration %s/%s", entries[chosen].Owner, entries[chosen].Name)})
			}
		}
	}
	unattributed := measured - attributed
	if unattributed > 0 {
		findings = append(findings, Finding{Code: "unattributed_storage", Severity: "warning", Path: displayRoot, Message: fmt.Sprintf("%d measured bytes have no declaration", unattributed)})
	}
	for _, path := range unreadable {
		findings = append(findings, Finding{Code: "unreadable_path", Severity: "error", Path: path, Message: "census could not read this path"})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].Owner != entries[j].Owner {
			return entries[i].Owner < entries[j].Owner
		}
		return entries[i].Name < entries[j].Name
	})
	sort.Slice(findings, func(i, j int) bool {
		if findings[i].Code != findings[j].Code {
			return findings[i].Code < findings[j].Code
		}
		if findings[i].Owner != findings[j].Owner {
			return findings[i].Owner < findings[j].Owner
		}
		return findings[i].Path < findings[j].Path
	})
	confidence := "high"
	if len(unreadable) > 0 || len(findings) > 0 {
		confidence = "degraded"
	}
	return Report{Root: displayRoot, MeasuredBytes: measured, AttributedBytes: attributed, UnattributedBytes: unattributed, Closed: len(unreadable) == 0 && unattributed == 0, AccountingIdentity: measured == attributed+unattributed, Confidence: confidence, UnreadablePaths: sortedStrings(unreadable), Findings: findings, Entries: entries}, nil
}

type fileObservation struct {
	path  string
	bytes int64
}

func existingRoots(roots []string) []string {
	seen := make(map[string]struct{}, len(roots))
	result := make([]string, 0, len(roots))
	for _, raw := range roots {
		path, err := filepath.Abs(raw)
		if err != nil {
			continue
		}
		path = filepath.Clean(path)
		if _, err := os.Stat(path); err != nil {
			continue
		}
		if _, exists := seen[path]; exists {
			continue
		}
		seen[path] = struct{}{}
		result = append(result, path)
	}
	sort.Strings(result)
	return result
}

func scenarioCandidateRoots(repoRoot string, owner corestorage.OwnerManifest) []string {
	scenarioRoot := filepath.Dir(filepath.Dir(owner.ManifestPath))
	if !filepath.IsAbs(scenarioRoot) {
		scenarioRoot = filepath.Join(repoRoot, scenarioRoot)
	}
	relative := []string{
		"data", "logs", "storage", "state", "uploads", "models", "cache", "runtime",
		filepath.Join("api", "data"), filepath.Join("api", "storage"),
	}
	result := make([]string, 0, len(relative))
	for _, name := range relative {
		path := filepath.Join(scenarioRoot, name)
		if _, err := os.Stat(path); err == nil {
			result = append(result, path)
		}
	}
	return result
}

func sortedStrings(values []string) []string { sort.Strings(values); return values }

// declarationMatches finds declarations that contain a file by walking the
// file's ancestor chain. The old implementation compared every file with
// every declaration, which made a legitimate whole-host declaration turn a
// census into O(files*declarations) work. Ancestor lookup keeps the same
// overlap semantics while making the hot path proportional to path depth.
func declarationMatches(path string, byPath map[string][]int) []int {
	matches := make([]int, 0, 1)
	for candidate := filepath.Clean(path); ; candidate = filepath.Dir(candidate) {
		if indexes := byPath[candidate]; len(indexes) > 0 {
			matches = append(matches, indexes...)
		}
		parent := filepath.Dir(candidate)
		if parent == candidate {
			break
		}
	}
	return matches
}

// Snapshot persistence is intentionally a small per-domain seam. The JSON
// payload keeps history forward-compatible while the indexed columns make
// the operator's common trend query cheap.
type SnapshotStore struct{ db *database.RoutedDB }

func NewSnapshotStore(db *database.RoutedDB) *SnapshotStore {
	if db == nil {
		return nil
	}
	return &SnapshotStore{db: db}
}

func (s *SnapshotStore) Save(ctx context.Context, report Report) (Report, error) {
	if s == nil {
		return report, nil
	}
	previous, err := s.latest(ctx, report.Root)
	if err != nil {
		return report, err
	}
	report.ObservedAt = time.Now().UTC()
	report.SnapshotID = snapshotID(report)
	if previous != nil && previous.ObservedAt.Before(report.ObservedAt) {
		hours := report.ObservedAt.Sub(previous.ObservedAt).Hours()
		if hours > 0 {
			slope := float64(report.MeasuredBytes-previous.MeasuredBytes) / hours
			report.GrowthSlopeBytesPerHour = &slope
		}
	}
	payload, err := json.Marshal(report)
	if err != nil {
		return Report{}, err
	}
	_, err = s.db.ExecContext(ctx, `INSERT INTO census_snapshots (id, observed_at, root, measured_bytes, attributed_bytes, unattributed_bytes, confidence, report_json) VALUES (?, ?, ?, ?, ?, ?, ?, ?)`, report.SnapshotID, report.ObservedAt.Format(time.RFC3339Nano), report.Root, report.MeasuredBytes, report.AttributedBytes, report.UnattributedBytes, report.Confidence, string(payload))
	if err != nil {
		return Report{}, fmt.Errorf("save census snapshot: %w", err)
	}
	return report, nil
}

func (s *SnapshotStore) latest(ctx context.Context, root string) (*Report, error) {
	var payload string
	err := s.db.QueryRowContext(ctx, `SELECT report_json FROM census_snapshots WHERE root = ? ORDER BY observed_at DESC LIMIT 1`, root).Scan(&payload)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("load census snapshot: %w", err)
	}
	var report Report
	if err := json.Unmarshal([]byte(payload), &report); err != nil {
		return nil, fmt.Errorf("decode census snapshot: %w", err)
	}
	return &report, nil
}

func (s *SnapshotStore) History(ctx context.Context, root string, limit int) ([]Report, error) {
	if s == nil {
		return []Report{}, nil
	}
	if limit < 1 || limit > 100 {
		limit = 20
	}
	rows, err := s.db.QueryContext(ctx, `SELECT report_json FROM census_snapshots WHERE root = ? ORDER BY observed_at DESC LIMIT ?`, root, limit)
	if err != nil {
		return nil, fmt.Errorf("load census history: %w", err)
	}
	defer func() { _ = rows.Close() }()
	result := make([]Report, 0, limit)
	for rows.Next() {
		var payload string
		if err := rows.Scan(&payload); err != nil {
			return nil, fmt.Errorf("scan census history: %w", err)
		}
		var report Report
		if err := json.Unmarshal([]byte(payload), &report); err != nil {
			return nil, fmt.Errorf("decode census history: %w", err)
		}
		result = append(result, report)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read census history: %w", err)
	}
	return result, nil
}

func snapshotID(report Report) string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%s/%d/%d/%d", report.Root, report.MeasuredBytes, report.AttributedBytes, report.ObservedAt.UnixNano())))
	return "census-" + hex.EncodeToString(hash[:8])
}

// Schema is embedded so the census domain participates in the same
// idempotent per-domain bootstrap as cleanup and fleet inventories.
func Schema() string { return censusSchemaSQL }
