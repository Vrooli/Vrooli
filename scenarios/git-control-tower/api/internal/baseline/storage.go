package baseline

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"

	"github.com/vrooli/api-core/storage"

	"git-control-tower/internal/git"
)

const maxPathSnapshotRepositoryBytes int64 = 64 << 20

// ErrNotFound is returned when a baseline does not exist.
var ErrNotFound = errors.New("baseline not found")

// ErrSnapshotQuota is returned before writing a path snapshot that would make
// the repository's retained source evidence exceed its bounded object budget.
var ErrSnapshotQuota = errors.New("path snapshot storage quota exceeded")

// ErrAlreadyExists is returned by Save (create mode) when a baseline with the
// same scenario+branch+name already exists.
var ErrAlreadyExists = errors.New("baseline already exists")

// Legacy migration errors are explicit because choosing a run from incomplete
// or mixed V1 pointers would manufacture a baseline identity that never
// existed.
var (
	ErrLegacyMixedRuns  = errors.New("legacy baseline references multiple runs")
	ErrLegacyIncomplete = errors.New("legacy baseline is incomplete")
)

// Storage is the branch-scoped, flock-protected baseline manifest store.
//
// Layout (Decision 2 — branch is the scoping axis):
//
//	data/<repoID>/baselines/<scenario>/<branch>/
//	  ├── <name>.json
//	  └── .locks/<name>.lock
//
// All reads/writes for a given manifest serialize through an advisory flock on
// its sibling lock file, so concurrent agents on one box cannot corrupt or race
// the same baseline. Locks are per-baseline-name, not per-scenario, so distinct
// baselines never contend.
type Storage struct {
	resolver     *storage.Resolver
	scenarioD    string // storage ScenarioID for the resolver (always "git-control-tower")
	rootOverride string // test seam: forces class roots under this dir; empty in prod
	// writeLifecycle is a narrow fault-injection seam for transition tests.
	// Production leaves it nil and uses the normal atomic writer.
	writeLifecycle func(path string, value any) error
	// writeJSON is the general durable-write seam. Tests can fail any record
	// boundary (intent, manifest, audit, diff, collection, or snapshot) without
	// bypassing the same atomic writer production uses.
	writeJSON func(path string, value any) error
}

// NewStorage builds a Storage over the given api-core resolver.
func NewStorage(resolver *storage.Resolver) *Storage {
	return &Storage{resolver: resolver, scenarioD: "git-control-tower"}
}

// NewStorageAt builds a Storage whose class roots are forced under root. Test
// seam — production uses NewStorage.
func NewStorageAt(resolver *storage.Resolver, root string) *Storage {
	return &Storage{resolver: resolver, scenarioD: "git-control-tower", rootOverride: root}
}

func (s *Storage) opts() storage.Options {
	return storage.Options{ScenarioID: s.scenarioD, RootOverride: s.rootOverride}
}

// ResolveStorageBranch maps a git.State to the filesystem branch segment.
// Detached HEAD uses the synthetic name detached-<sha[:8]> (Decision 2);
// an empty/unknown branch falls back to "HEAD".
func ResolveStorageBranch(st git.State) string {
	if st.Detached || st.Branch == "" {
		if len(st.Sha) >= 8 {
			return "detached-" + st.Sha[:8]
		}
		if st.Detached {
			return "detached-unknown"
		}
		return "HEAD"
	}
	return st.Branch
}

// sanitizeSegment makes a branch or baseline name safe as a single path
// segment. Slashes become "__"; other separators and control chars are
// stripped. The manifest's own Branch/Name fields remain the source of truth,
// so this need not be reversible.
func sanitizeSegment(s string) string {
	s = strings.ReplaceAll(s, "/", "__")
	s = strings.ReplaceAll(s, "\\", "__")
	s = strings.ReplaceAll(s, "\x00", "")
	s = strings.TrimSpace(s)
	// Refuse path-traversal segments outright.
	if s == "." || s == ".." || s == "" {
		return "_"
	}
	return s
}

func (s *Storage) branchDir(repoID int64, scenario, branch string) (string, error) {
	return s.resolver.Path(
		s.opts(),
		storage.ClassData,
		fmt.Sprintf("%d/baselines/%s/%s", repoID, sanitizeSegment(scenario), sanitizeSegment(branch)),
	)
}

func (s *Storage) scenarioDir(repoID int64, scenario string) (string, error) {
	return s.resolver.Path(
		s.opts(),
		storage.ClassData,
		fmt.Sprintf("%d/baselines/%s", repoID, sanitizeSegment(scenario)),
	)
}

func (s *Storage) baselinesDir(repoID int64) (string, error) {
	return s.resolver.Path(
		s.opts(),
		storage.ClassData,
		fmt.Sprintf("%d/baselines", repoID),
	)
}

func (s *Storage) collectionDir(repoID int64, branch string) (string, error) {
	return s.resolver.Path(s.opts(), storage.ClassData, fmt.Sprintf("%d/baseline-collections/%s", repoID, sanitizeSegment(branch)))
}

func (s *Storage) collectionPath(dir, name string) string {
	return filepath.Join(dir, sanitizeSegment(name)+".json")
}

func (s *Storage) collectionDiffPath(dir, name, operationID string) string {
	return filepath.Join(dir, ".diffs", sanitizeSegment(name)+"__"+sanitizeSegment(operationID)+".json")
}

func (s *Storage) pathSnapshotRepoDir(repoID int64) (string, error) {
	return s.resolver.Path(s.opts(), storage.ClassData, fmt.Sprintf("%d/path-snapshots", repoID))
}

func (s *Storage) pathSnapshotDir(repoID int64, branch string) (string, error) {
	root, err := s.pathSnapshotRepoDir(repoID)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, sanitizeSegment(branch)), nil
}

func (s *Storage) pathSnapshotPath(dir, name string) string {
	return filepath.Join(dir, sanitizeSegment(name)+".json")
}

func (s *Storage) pathSnapshotObjectDir(repoID int64) (string, error) {
	root, err := s.pathSnapshotRepoDir(repoID)
	if err != nil {
		return "", err
	}
	return filepath.Join(root, "objects"), nil
}

func (s *Storage) pathSnapshotObjectPath(repoID int64, digest string) (string, error) {
	if len(digest) != 64 {
		return "", fmt.Errorf("invalid path snapshot content digest")
	}
	for _, r := range digest {
		if !((r >= '0' && r <= '9') || (r >= 'a' && r <= 'f')) {
			return "", fmt.Errorf("invalid path snapshot content digest")
		}
	}
	dir, err := s.pathSnapshotObjectDir(repoID)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, digest), nil
}

func (s *Storage) manifestPath(dir, name string) string {
	return filepath.Join(dir, sanitizeSegment(name)+".json")
}

// diffCachePath is where a computed diff result is cached, keyed by baseline
// name + the current run it was computed against. Keeping it under the baseline
// branch dir (in a .diffs/ subdir) co-locates it with the manifest it diffs.
func (s *Storage) diffCachePath(dir, name, runID string) string {
	return filepath.Join(dir, ".diffs", sanitizeSegment(name)+"__"+sanitizeSegment(runID)+".json")
}

func (s *Storage) snapshotIntentPath(dir, name, runID string) string {
	return filepath.Join(dir, ".snapshots", sanitizeSegment(name)+"__"+sanitizeSegment(runID)+".json")
}

func (s *Storage) lifecyclePath(dir, name string) string {
	return filepath.Join(dir, ".lifecycle", sanitizeSegment(name)+".json")
}

func (s *Storage) lifecycleAuditPath(dir, name string, createdAt time.Time) string {
	return filepath.Join(dir, ".lifecycle-audit", sanitizeSegment(name)+"__"+createdAt.UTC().Format("20060102T150405.000000000Z")+".json")
}

func (s *Storage) diffIntentPath(dir, name, runID string) string {
	return filepath.Join(dir, ".diffs", ".intents", sanitizeSegment(name)+"__"+sanitizeSegment(runID)+".json")
}

// CachedDiff is the persisted outcome of a finalized diff: the computed result
// (ready) or the failure reason (failed). Cached so GetDiffResult returns the
// verdict instantly, surviving client disconnect and server restart.
type CachedDiff struct {
	Status     string      `json:"status"` // ready | failed
	Result     *DiffResult `json:"result,omitempty"`
	Error      string      `json:"error,omitempty"`
	RunID      string      `json:"run_id"`
	ComputedAt time.Time   `json:"computed_at"`
}

// DiffIntent is the durable StartDiff handoff. It exists before the verdict
// cache, so interrupted clients can recover the latest run id for a baseline.
type DiffIntent struct {
	Status     string           `json:"status"` // pending | ready | failed
	Error      string           `json:"error,omitempty"`
	RepoID     int64            `json:"repo_id"`
	Scenario   string           `json:"scenario"`
	Branch     string           `json:"branch"`
	Name       string           `json:"name"`
	Manifest   BaselineManifest `json:"manifest"`
	CurrentGit git.State        `json:"current_git"`
	Staleness  Staleness        `json:"staleness"`
	BaseRunID  string           `json:"base_run_id"`
	CurRunID   string           `json:"cur_run_id"`
	CreatedAt  time.Time        `json:"created_at"`
	UpdatedAt  time.Time        `json:"updated_at"`
}

// SnapshotIntent is the durable handoff between SnapshotForBaseline's
// return-fast start and the later pin+manifest finalize. It exists so a GCT API
// restart cannot lose the only in-memory goroutine that knew which run should
// become which baseline.
type SnapshotIntent struct {
	Status       string           `json:"status"` // pending | ready | failed
	Error        string           `json:"error,omitempty"`
	RepoID       int64            `json:"repo_id"`
	RepoDir      string           `json:"repo_dir"`
	Scenario     string           `json:"scenario"`
	Branch       string           `json:"branch"`
	Name         string           `json:"name"`
	CreatedBy    string           `json:"created_by,omitempty"`
	Reason       string           `json:"reason,omitempty"`
	Manifest     BaselineManifest `json:"manifest"`
	Run          RunHandle        `json:"run"`
	DirtyWarning string           `json:"dirty_warning,omitempty"`
	CreatedAt    time.Time        `json:"created_at"`
	UpdatedAt    time.Time        `json:"updated_at"`
}

// PendingCapture reconstructs the finalize input from the durable intent.
func (i SnapshotIntent) PendingCapture() PendingCapture {
	return PendingCapture{
		Manifest: i.Manifest,
		Req: CreateRequest{
			RepoID:    i.RepoID,
			RepoDir:   i.RepoDir,
			Scenario:  i.Scenario,
			Name:      i.Name,
			Branch:    i.Branch,
			CreatedBy: i.CreatedBy,
			Reason:    i.Reason,
		},
		Run:          i.Run,
		DirtyWarning: i.DirtyWarning,
	}
}

// PendingDiff reconstructs the finalize input from the durable intent.
func (i DiffIntent) PendingDiff() PendingDiff {
	return PendingDiff{
		RepoID:     i.RepoID,
		Scenario:   i.Scenario,
		Branch:     i.Branch,
		Name:       i.Name,
		Manifest:   i.Manifest,
		CurrentGit: i.CurrentGit,
		Staleness:  i.Staleness,
		BaseRunID:  i.BaseRunID,
		CurRunID:   i.CurRunID,
	}
}

// SaveSnapshotIntent persists a snapshot intent. It overwrites the status for
// the same (baseline name, run id), which lets finalize mark a pending intent
// ready/failed without changing its identity.
func (s *Storage) SaveSnapshotIntent(repoID int64, intent SnapshotIntent) error {
	dir, err := s.branchDir(repoID, intent.Scenario, intent.Branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, intent.Name, func() error {
		path := s.snapshotIntentPath(dir, intent.Name, intent.Run.RunID)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create snapshot intent dir: %w", err)
		}
		return s.writeRecord(path, intent)
	})
}

// SaveLifecycle is the sole durable authority for whether an intent may
// publish a baseline. It is deliberately retained after deletion.
func (s *Storage) SaveLifecycle(repoID int64, record LifecycleRecord) error {
	if err := record.Validate(); err != nil {
		return err
	}
	dir, err := s.branchDir(repoID, record.Scenario, record.Branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, record.Name, func() error {
		path := s.lifecyclePath(dir, record.Name)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create baseline lifecycle dir: %w", err)
		}
		if s.writeLifecycle != nil {
			return s.writeLifecycle(path, record)
		}
		return s.writeRecord(path, record)
	})
}

func (s *Storage) LoadLifecycle(repoID int64, scenario, branch, name string) (LifecycleRecord, error) {
	dir, err := s.branchDir(repoID, scenario, branch)
	if err != nil {
		return LifecycleRecord{}, err
	}
	var record LifecycleRecord
	err = s.withLock(dir, name, func() error {
		data, err := os.ReadFile(s.lifecyclePath(dir, name))
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read baseline lifecycle: %w", err)
		}
		if err := json.Unmarshal(data, &record); err != nil {
			return fmt.Errorf("decode baseline lifecycle: %w", err)
		}
		return record.Validate()
	})
	return record, err
}

func (s *Storage) SaveLifecycleAudit(repoID int64, entry LifecycleAuditEntry) error {
	if strings.TrimSpace(entry.Scenario) == "" || strings.TrimSpace(entry.Branch) == "" || strings.TrimSpace(entry.Name) == "" || strings.TrimSpace(entry.Action) == "" {
		return fmt.Errorf("baseline lifecycle audit identity and action are required")
	}
	if entry.CreatedAt.IsZero() {
		return fmt.Errorf("baseline lifecycle audit timestamp is required")
	}
	dir, err := s.branchDir(repoID, entry.Scenario, entry.Branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, entry.Name, func() error {
		path := s.lifecycleAuditPath(dir, entry.Name, entry.CreatedAt)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create baseline lifecycle audit dir: %w", err)
		}
		return s.writeRecord(path, entry)
	})
}

// LoadSnapshotIntent reads one durable snapshot intent.
func (s *Storage) LoadSnapshotIntent(repoID int64, scenario, branch, name, runID string) (SnapshotIntent, bool, error) {
	dir, err := s.branchDir(repoID, scenario, branch)
	if err != nil {
		return SnapshotIntent{}, false, err
	}
	var intent SnapshotIntent
	found := false
	err = s.withLock(dir, name, func() error {
		data, rerr := os.ReadFile(s.snapshotIntentPath(dir, name, runID))
		if rerr != nil {
			if os.IsNotExist(rerr) {
				return nil
			}
			return fmt.Errorf("read snapshot intent: %w", rerr)
		}
		found = true
		return json.Unmarshal(data, &intent)
	})
	if err != nil {
		return SnapshotIntent{}, false, err
	}
	return intent, found, nil
}

// ListSnapshotIntents lists durable snapshot intents for a scenario. Empty
// branch lists across every branch; empty name lists every baseline name.
func (s *Storage) ListSnapshotIntents(repoID int64, scenario, branch, name string) ([]SnapshotIntent, error) {
	var dirs []string
	if branch != "" {
		dir, err := s.branchDir(repoID, scenario, branch)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, dir)
	} else {
		root, err := s.scenarioDir(repoID, scenario)
		if err != nil {
			return nil, err
		}
		entries, rerr := os.ReadDir(root)
		if rerr != nil {
			if os.IsNotExist(rerr) {
				return []SnapshotIntent{}, nil
			}
			return nil, fmt.Errorf("read scenario baselines: %w", rerr)
		}
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(root, e.Name()))
			}
		}
	}

	var out []SnapshotIntent
	for _, dir := range dirs {
		entries, err := os.ReadDir(filepath.Join(dir, ".snapshots"))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read snapshot intents: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			data, rerr := os.ReadFile(filepath.Join(dir, ".snapshots", e.Name()))
			if rerr != nil {
				continue
			}
			var intent SnapshotIntent
			if json.Unmarshal(data, &intent) != nil {
				continue
			}
			if name != "" && intent.Name != name {
				continue
			}
			out = append(out, intent)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].UpdatedAt.After(out[b].UpdatedAt)
	})
	return out, nil
}

// ListAllSnapshotIntents lists every durable snapshot intent for a repository.
func (s *Storage) ListAllSnapshotIntents(repoID int64) ([]SnapshotIntent, error) {
	root, err := s.baselinesDir(repoID)
	if err != nil {
		return nil, err
	}
	scenarios, rerr := os.ReadDir(root)
	if rerr != nil {
		if os.IsNotExist(rerr) {
			return []SnapshotIntent{}, nil
		}
		return nil, fmt.Errorf("read baselines root: %w", rerr)
	}
	var out []SnapshotIntent
	for _, sc := range scenarios {
		if !sc.IsDir() {
			continue
		}
		intents, err := s.ListSnapshotIntents(repoID, sc.Name(), "", "")
		if err != nil {
			return nil, err
		}
		out = append(out, intents...)
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].UpdatedAt.After(out[b].UpdatedAt)
	})
	return out, nil
}

// SaveDiffResult persists a finalized diff under (repoID, scenario, branch,
// name, runID). Writes are atomic (tmp + rename) and serialized through the same
// per-baseline lock as the manifest.
func (s *Storage) SaveDiffResult(repoID int64, scenario, branch, name, runID string, cd CachedDiff) error {
	dir, err := s.branchDir(repoID, scenario, branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, name, func() error {
		path := s.diffCachePath(dir, name, runID)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create diff cache dir: %w", err)
		}
		return s.writeRecord(path, cd)
	})
}

// LoadDiffResult reads a cached diff. ok=false (no error) when none is cached.
func (s *Storage) LoadDiffResult(repoID int64, scenario, branch, name, runID string) (CachedDiff, bool, error) {
	dir, err := s.branchDir(repoID, scenario, branch)
	if err != nil {
		return CachedDiff{}, false, err
	}
	var cd CachedDiff
	found := false
	err = s.withLock(dir, name, func() error {
		data, rerr := os.ReadFile(s.diffCachePath(dir, name, runID))
		if rerr != nil {
			if os.IsNotExist(rerr) {
				return nil
			}
			return fmt.Errorf("read diff cache: %w", rerr)
		}
		found = true
		return json.Unmarshal(data, &cd)
	})
	if err != nil {
		return CachedDiff{}, false, err
	}
	return cd, found, nil
}

// SaveDiffIntent persists the StartDiff handoff before the RPC returns.
func (s *Storage) SaveDiffIntent(repoID int64, intent DiffIntent) error {
	dir, err := s.branchDir(repoID, intent.Scenario, intent.Branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, intent.Name, func() error {
		path := s.diffIntentPath(dir, intent.Name, intent.CurRunID)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			return fmt.Errorf("create diff intent dir: %w", err)
		}
		return s.writeRecord(path, intent)
	})
}

// LoadDiffIntent reconstructs one durable child diff finalizer after a server
// restart or a collection-level wait attachment.
func (s *Storage) LoadDiffIntent(repoID int64, scenario, branch, name, runID string) (DiffIntent, bool, error) {
	dir, err := s.branchDir(repoID, scenario, branch)
	if err != nil {
		return DiffIntent{}, false, err
	}
	var intent DiffIntent
	found := false
	err = s.withLock(dir, name, func() error {
		data, err := os.ReadFile(s.diffIntentPath(dir, name, runID))
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return fmt.Errorf("read diff intent: %w", err)
		}
		found = true
		return json.Unmarshal(data, &intent)
	})
	if err != nil {
		return DiffIntent{}, false, err
	}
	return intent, found, nil
}

// LatestDiffIntent returns the newest StartDiff intent for a baseline.
func (s *Storage) LatestDiffIntent(repoID int64, scenario, branch, name string) (DiffIntent, bool, error) {
	dir, err := s.branchDir(repoID, scenario, branch)
	if err != nil {
		return DiffIntent{}, false, err
	}
	intentDir := filepath.Join(dir, ".diffs", ".intents")
	entries, rerr := os.ReadDir(intentDir)
	if rerr != nil {
		if os.IsNotExist(rerr) {
			return DiffIntent{}, false, nil
		}
		return DiffIntent{}, false, fmt.Errorf("read diff intents: %w", rerr)
	}
	var latest DiffIntent
	found := false
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		data, ferr := os.ReadFile(filepath.Join(intentDir, e.Name()))
		if ferr != nil {
			continue
		}
		var intent DiffIntent
		if json.Unmarshal(data, &intent) != nil {
			continue
		}
		if intent.Name != name {
			continue
		}
		if !found || intent.UpdatedAt.After(latest.UpdatedAt) {
			latest = intent
			found = true
		}
	}
	return latest, found, nil
}

func (s *Storage) lockPath(dir, name string) string {
	return filepath.Join(dir, ".locks", sanitizeSegment(name)+".lock")
}

// withLock runs fn while holding an exclusive advisory lock for (dir, name).
func (s *Storage) withLock(dir, name string, fn func() error) error {
	lockPath := s.lockPath(dir, name)
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return fmt.Errorf("create lock dir: %w", err)
	}
	lf, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return fmt.Errorf("open baseline lock: %w", err)
	}
	defer lf.Close()
	if err := syscall.Flock(int(lf.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("acquire baseline lock: %w", err)
	}
	defer func() { _ = syscall.Flock(int(lf.Fd()), syscall.LOCK_UN) }()
	return fn()
}

// SaveMode controls Save's create-vs-overwrite behavior.
type SaveMode int

const (
	// CreateOnly fails with ErrAlreadyExists if the baseline exists.
	CreateOnly SaveMode = iota
	// Overwrite replaces an existing baseline (used by Edit).
	Overwrite
)

// Save persists a manifest. The manifest's Branch field selects the storage
// branch segment; callers resolve detached HEAD via ResolveStorageBranch
// before populating it.
func (s *Storage) Save(repoID int64, m BaselineManifest, mode SaveMode) error {
	if err := m.Validate(); err != nil {
		return err
	}
	dir, err := s.branchDir(repoID, m.Scenario, m.Branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, m.Name, func() error {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create baseline dir: %w", err)
		}
		path := s.manifestPath(dir, m.Name)
		if mode == CreateOnly {
			if _, statErr := os.Stat(path); statErr == nil {
				return ErrAlreadyExists
			} else if !os.IsNotExist(statErr) {
				return fmt.Errorf("stat baseline: %w", statErr)
			}
		}
		return s.writeRecord(path, m)
	})
}

// Load reads a manifest by scenario+branch+name. Returns ErrNotFound if absent.
func (s *Storage) Load(repoID int64, scenario, branch, name string) (BaselineManifest, error) {
	dir, err := s.branchDir(repoID, scenario, branch)
	if err != nil {
		return BaselineManifest{}, err
	}
	var m BaselineManifest
	err = s.withLock(dir, name, func() error {
		data, rerr := os.ReadFile(s.manifestPath(dir, name))
		if rerr != nil {
			if os.IsNotExist(rerr) {
				return ErrNotFound
			}
			return fmt.Errorf("read baseline: %w", rerr)
		}
		decoded, migrated, derr := decodeManifest(data, s.nowUTC())
		if derr != nil {
			return derr
		}
		m = decoded
		if migrated {
			if werr := s.writeRecord(s.manifestPath(dir, name), m); werr != nil {
				return fmt.Errorf("persist migrated baseline: %w", werr)
			}
		}
		return nil
	})
	if err != nil {
		return BaselineManifest{}, err
	}
	return m, nil
}

// Delete removes a manifest. Missing baselines return ErrNotFound.
func (s *Storage) Delete(repoID int64, scenario, branch, name string) error {
	dir, err := s.branchDir(repoID, scenario, branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, name, func() error {
		path := s.manifestPath(dir, name)
		if _, statErr := os.Stat(path); os.IsNotExist(statErr) {
			return ErrNotFound
		}
		return os.Remove(path)
	})
}

// SaveCollection writes a durable aggregate of existing per-scenario anchors.
// The collection lock is separate from member locks: it never blocks snapshot
// finalization for an individual scenario, and member state can be reconciled
// independently after restart.
func (s *Storage) SaveCollection(repoID int64, collection CollectionManifest, mode SaveMode) error {
	collection = collection.Normalized()
	if err := collection.Validate(); err != nil {
		return err
	}
	dir, err := s.collectionDir(repoID, collection.Branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, collection.Name, func() error {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create collection dir: %w", err)
		}
		path := s.collectionPath(dir, collection.Name)
		if mode == CreateOnly {
			if _, err := os.Stat(path); err == nil {
				return ErrAlreadyExists
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat collection: %w", err)
			}
		}
		return s.writeRecord(path, collection)
	})
}

func (s *Storage) LoadCollection(repoID int64, branch, name string) (CollectionManifest, error) {
	dir, err := s.collectionDir(repoID, branch)
	if err != nil {
		return CollectionManifest{}, err
	}
	var collection CollectionManifest
	err = s.withLock(dir, name, func() error {
		data, err := os.ReadFile(s.collectionPath(dir, name))
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read collection: %w", err)
		}
		if err := json.Unmarshal(data, &collection); err != nil {
			return fmt.Errorf("decode collection: %w", err)
		}
		collection = collection.Normalized()
		return collection.Validate()
	})
	if err != nil {
		return CollectionManifest{}, err
	}
	return collection, nil
}

func (s *Storage) DeleteCollection(repoID int64, branch, name string) error {
	dir, err := s.collectionDir(repoID, branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, name, func() error {
		if err := os.Remove(s.collectionPath(dir, name)); os.IsNotExist(err) {
			return ErrNotFound
		} else if err != nil {
			return fmt.Errorf("delete collection: %w", err)
		}
		// Collection diff operations are independent durable records. They retain
		// their immutable collection snapshot so deletion of the mutable
		// collection cannot strand a detached validation or erase its audit trail.
		// Their lifecycle is governed by operation retention, not collection
		// deletion.
		return nil
	})
}

func (s *Storage) SaveCollectionDiffOperation(repoID int64, operation CollectionDiffOperation, mode SaveMode) error {
	if strings.TrimSpace(operation.ID) == "" || strings.TrimSpace(operation.Collection) == "" || strings.TrimSpace(operation.Branch) == "" {
		return fmt.Errorf("collection diff operation id, collection, and branch are required")
	}
	dir, err := s.collectionDir(repoID, operation.Branch)
	if err != nil {
		return err
	}
	return s.withLock(dir, operation.Collection, func() error {
		path := s.collectionDiffPath(dir, operation.Collection, operation.ID)
		if mode == CreateOnly {
			if _, err := os.Stat(path); err == nil {
				return ErrAlreadyExists
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat collection diff operation: %w", err)
			}
		}
		return s.writeRecord(path, operation)
	})
}

func (s *Storage) LoadCollectionDiffOperation(repoID int64, branch, name, operationID string) (CollectionDiffOperation, error) {
	dir, err := s.collectionDir(repoID, branch)
	if err != nil {
		return CollectionDiffOperation{}, err
	}
	var operation CollectionDiffOperation
	err = s.withLock(dir, name, func() error {
		data, err := os.ReadFile(s.collectionDiffPath(dir, name, operationID))
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read collection diff operation: %w", err)
		}
		if err := json.Unmarshal(data, &operation); err != nil {
			return fmt.Errorf("decode collection diff operation: %w", err)
		}
		return nil
	})
	if err != nil {
		return CollectionDiffOperation{}, err
	}
	return operation, nil
}

// ListCollectionDiffOperations returns every durable collection-diff parent for
// a repository. It is used only by the server-owned reconciler; callers still
// use LoadCollectionDiffOperation for an individual operation.
func (s *Storage) ListCollectionDiffOperations(repoID int64) ([]CollectionDiffOperation, error) {
	root, err := s.resolver.Path(s.opts(), storage.ClassData, fmt.Sprintf("%d/baseline-collections", repoID))
	if err != nil {
		return nil, err
	}
	branches, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []CollectionDiffOperation{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read collection diff root: %w", err)
	}
	var operations []CollectionDiffOperation
	for _, branch := range branches {
		if !branch.IsDir() {
			continue
		}
		diffDir := filepath.Join(root, branch.Name(), ".diffs")
		entries, readErr := os.ReadDir(diffDir)
		if os.IsNotExist(readErr) {
			continue
		}
		if readErr != nil {
			return nil, fmt.Errorf("read collection diff directory: %w", readErr)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			data, readErr := os.ReadFile(filepath.Join(diffDir, entry.Name()))
			if readErr != nil {
				return nil, fmt.Errorf("read collection diff operation: %w", readErr)
			}
			var operation CollectionDiffOperation
			if err := json.Unmarshal(data, &operation); err != nil {
				return nil, fmt.Errorf("decode collection diff operation: %w", err)
			}
			operations = append(operations, operation)
		}
	}
	sort.SliceStable(operations, func(i, j int) bool { return operations[i].UpdatedAt.After(operations[j].UpdatedAt) })
	return operations, nil
}

// ListCollections returns every durable collection manifest for a repository.
// It is deliberately a server-owned reconciliation seam; callers that know a
// collection identity should use LoadCollection instead.
func (s *Storage) ListCollections(repoID int64) ([]CollectionManifest, error) {
	root, err := s.resolver.Path(s.opts(), storage.ClassData, fmt.Sprintf("%d/baseline-collections", repoID))
	if err != nil {
		return nil, err
	}
	branches, err := os.ReadDir(root)
	if os.IsNotExist(err) {
		return []CollectionManifest{}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read collection root: %w", err)
	}
	var collections []CollectionManifest
	for _, branch := range branches {
		if !branch.IsDir() {
			continue
		}
		entries, readErr := os.ReadDir(filepath.Join(root, branch.Name()))
		if readErr != nil {
			return nil, fmt.Errorf("read collection directory: %w", readErr)
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			data, readErr := os.ReadFile(filepath.Join(root, branch.Name(), entry.Name()))
			if readErr != nil {
				return nil, fmt.Errorf("read collection: %w", readErr)
			}
			var collection CollectionManifest
			if err := json.Unmarshal(data, &collection); err != nil {
				return nil, fmt.Errorf("decode collection: %w", err)
			}
			collection = collection.Normalized()
			if err := collection.Validate(); err != nil {
				return nil, fmt.Errorf("validate collection: %w", err)
			}
			collections = append(collections, collection)
		}
	}
	sort.SliceStable(collections, func(i, j int) bool { return collections[i].UpdatedAt.After(collections[j].UpdatedAt) })
	return collections, nil
}

// UpdateCollectionDiffOperation is the compare-and-write boundary for
// multi-process dispatch claims. Callers must put any decision derived from a
// child lease inside mutate; separate Load/Save calls are not safe across two
// GCT servers.
func (s *Storage) UpdateCollectionDiffOperation(repoID int64, branch, name, operationID string, mutate func(*CollectionDiffOperation) error) (CollectionDiffOperation, error) {
	dir, err := s.collectionDir(repoID, branch)
	if err != nil {
		return CollectionDiffOperation{}, err
	}
	var operation CollectionDiffOperation
	err = s.withLock(dir, name, func() error {
		data, readErr := os.ReadFile(s.collectionDiffPath(dir, name, operationID))
		if os.IsNotExist(readErr) {
			return ErrNotFound
		}
		if readErr != nil {
			return fmt.Errorf("read collection diff operation: %w", readErr)
		}
		if err := json.Unmarshal(data, &operation); err != nil {
			return fmt.Errorf("decode collection diff operation: %w", err)
		}
		if err := mutate(&operation); err != nil {
			return err
		}
		return s.writeRecord(s.collectionDiffPath(dir, name, operationID), operation)
	})
	return operation, err
}

// SavePathSnapshot atomically commits a manifest and its content-addressed
// text objects. The store lock serializes creation, deletion, and garbage
// collection so a deleted snapshot cannot race a new object into being swept.
// Objects are write-once and verify their SHA-256 reference before retention.
func (s *Storage) SavePathSnapshot(repoID int64, snapshot PathSnapshot, objects map[string][]byte, mode SaveMode) error {
	if err := snapshot.Validate(); err != nil {
		return err
	}
	root, err := s.pathSnapshotRepoDir(repoID)
	if err != nil {
		return err
	}
	return s.withLock(root, ".path-snapshot-store", func() error {
		dir, err := s.pathSnapshotDir(repoID, snapshot.Branch)
		if err != nil {
			return err
		}
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("create path snapshot directory: %w", err)
		}
		path := s.pathSnapshotPath(dir, snapshot.Name)
		if mode == CreateOnly {
			if _, err := os.Stat(path); err == nil {
				return ErrAlreadyExists
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat path snapshot: %w", err)
			}
		}
		refs := snapshotContentRefs(snapshot)
		var needed int64
		for ref := range refs {
			object, ok := objects[ref]
			if !ok {
				return fmt.Errorf("path snapshot object %q is missing", ref)
			}
			digest := sha256.Sum256(object)
			if fmt.Sprintf("%x", digest) != ref {
				return fmt.Errorf("path snapshot object %q has wrong digest", ref)
			}
			objectPath, err := s.pathSnapshotObjectPath(repoID, ref)
			if err != nil {
				return err
			}
			if _, err := os.Stat(objectPath); os.IsNotExist(err) {
				needed += int64(len(object))
			} else if err != nil {
				return fmt.Errorf("stat path snapshot object: %w", err)
			}
		}
		used, err := s.pathSnapshotObjectBytes(repoID)
		if err != nil {
			return err
		}
		if used+needed > maxPathSnapshotRepositoryBytes {
			return fmt.Errorf("%w: limit is %d bytes", ErrSnapshotQuota, maxPathSnapshotRepositoryBytes)
		}
		for ref := range refs {
			objectPath, err := s.pathSnapshotObjectPath(repoID, ref)
			if err != nil {
				return err
			}
			if _, err := os.Stat(objectPath); err == nil {
				continue
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("stat path snapshot object: %w", err)
			}
			if err := os.MkdirAll(filepath.Dir(objectPath), 0o755); err != nil {
				return fmt.Errorf("create path snapshot object directory: %w", err)
			}
			if err := writeBytesAtomic(objectPath, objects[ref], 0o600); err != nil {
				return fmt.Errorf("write path snapshot object: %w", err)
			}
		}
		return s.writeRecord(path, snapshot)
	})
}

// LoadPathSnapshot reads immutable source evidence. It deliberately returns no
// file bytes; transport callers can expose only redacted summaries and bounded
// diffs built by this package.
func (s *Storage) LoadPathSnapshot(repoID int64, branch, name string) (PathSnapshot, error) {
	root, err := s.pathSnapshotRepoDir(repoID)
	if err != nil {
		return PathSnapshot{}, err
	}
	var snapshot PathSnapshot
	err = s.withLock(root, ".path-snapshot-store", func() error {
		dir, err := s.pathSnapshotDir(repoID, branch)
		if err != nil {
			return err
		}
		data, err := os.ReadFile(s.pathSnapshotPath(dir, name))
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read path snapshot: %w", err)
		}
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return fmt.Errorf("decode path snapshot: %w", err)
		}
		return snapshot.Validate()
	})
	if err != nil {
		return PathSnapshot{}, err
	}
	return snapshot, nil
}

// DeletePathSnapshot deletes one manifest and runs reference-count-free GC.
// A full manifest scan is bounded by retention policy and avoids a second,
// crash-prone mutable refcount index.
func (s *Storage) DeletePathSnapshot(repoID int64, branch, name string) error {
	root, err := s.pathSnapshotRepoDir(repoID)
	if err != nil {
		return err
	}
	return s.withLock(root, ".path-snapshot-store", func() error {
		dir, err := s.pathSnapshotDir(repoID, branch)
		if err != nil {
			return err
		}
		if err := os.Remove(s.pathSnapshotPath(dir, name)); os.IsNotExist(err) {
			return ErrNotFound
		} else if err != nil {
			return fmt.Errorf("delete path snapshot: %w", err)
		}
		return s.collectPathSnapshotObjectsLocked(repoID)
	})
}

// SweepExpiredPathSnapshots removes expired manifests then reclaims any newly
// unreferenced content objects. It uses the same store lock as writes/deletes,
// so a crash leaves either the old manifest or a recoverable object scan — not
// a partially shared object refcount.
func (s *Storage) SweepExpiredPathSnapshots(repoID int64, now time.Time) (int, error) {
	root, err := s.pathSnapshotRepoDir(repoID)
	if err != nil {
		return 0, err
	}
	removed := 0
	err = s.withLock(root, ".path-snapshot-store", func() error {
		var expired []string
		err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.Contains(filepath.ToSlash(path), "/.locks/") {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			var snapshot PathSnapshot
			if json.Unmarshal(data, &snapshot) != nil {
				return nil // preserve corrupt evidence for manual recovery
			}
			if !snapshot.ExpiresAt.After(now.UTC()) {
				expired = append(expired, path)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		for _, path := range expired {
			if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
				return fmt.Errorf("remove expired path snapshot: %w", err)
			}
			removed++
		}
		return s.collectPathSnapshotObjectsLocked(repoID)
	})
	return removed, err
}

func snapshotContentRefs(snapshot PathSnapshot) map[string]struct{} {
	refs := make(map[string]struct{})
	for _, entry := range snapshot.Entries {
		if entry.ContentRef != "" {
			refs[entry.ContentRef] = struct{}{}
		}
	}
	return refs
}

func (s *Storage) pathSnapshotObjectBytes(repoID int64) (int64, error) {
	dir, err := s.pathSnapshotObjectDir(repoID)
	if err != nil {
		return 0, err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return 0, nil
	}
	if err != nil {
		return 0, fmt.Errorf("read path snapshot objects: %w", err)
	}
	var total int64
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return 0, fmt.Errorf("stat path snapshot object: %w", err)
		}
		total += info.Size()
	}
	return total, nil
}

func (s *Storage) collectPathSnapshotObjectsLocked(repoID int64) error {
	root, err := s.pathSnapshotRepoDir(repoID)
	if err != nil {
		return err
	}
	refs := map[string]struct{}{}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") || strings.Contains(filepath.ToSlash(path), "/.locks/") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		var snapshot PathSnapshot
		if err := json.Unmarshal(data, &snapshot); err != nil {
			return fmt.Errorf("decode path snapshot during garbage collection: %w", err)
		}
		for ref := range snapshotContentRefs(snapshot) {
			refs[ref] = struct{}{}
		}
		return nil
	})
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("scan path snapshots for garbage collection: %w", err)
	}
	dir, err := s.pathSnapshotObjectDir(repoID)
	if err != nil {
		return err
	}
	entries, err := os.ReadDir(dir)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("read path snapshot objects for garbage collection: %w", err)
	}
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		if _, retained := refs[entry.Name()]; retained {
			continue
		}
		if err := os.Remove(filepath.Join(dir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("remove unreferenced path snapshot object: %w", err)
		}
	}
	return nil
}

// UpdateCollectionMember applies one member transition while holding the
// collection's lock. Returning ErrNotFound is intentional: a deleted
// collection must not be silently recreated by a late server-owned finalizer.
func (s *Storage) UpdateCollectionMember(repoID int64, branch, name, scenario string, update func(*CollectionMember) error) (CollectionManifest, error) {
	dir, err := s.collectionDir(repoID, branch)
	if err != nil {
		return CollectionManifest{}, err
	}
	var out CollectionManifest
	err = s.withLock(dir, name, func() error {
		data, err := os.ReadFile(s.collectionPath(dir, name))
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read collection: %w", err)
		}
		if err := json.Unmarshal(data, &out); err != nil {
			return fmt.Errorf("decode collection: %w", err)
		}
		out = out.Normalized()
		found := false
		for i := range out.Members {
			if out.Members[i].Scenario != scenario {
				continue
			}
			found = true
			if err := update(&out.Members[i]); err != nil {
				return err
			}
			break
		}
		if !found {
			return fmt.Errorf("collection member %q not found", scenario)
		}
		out.UpdatedAt = time.Now().UTC()
		out = out.Normalized()
		if err := out.Validate(); err != nil {
			return err
		}
		return s.writeRecord(s.collectionPath(dir, name), out)
	})
	if err != nil {
		return CollectionManifest{}, err
	}
	return out, nil
}

// AppendCollectionMembers is the sole mutation path for collection membership.
// It is deliberately append-only so historical before-state coverage cannot be
// narrowed or silently rewritten after implementation begins.
func (s *Storage) AppendCollectionMembers(repoID int64, branch, name string, targets []CollectionTarget, now time.Time) (CollectionManifest, error) {
	dir, err := s.collectionDir(repoID, branch)
	if err != nil {
		return CollectionManifest{}, err
	}
	if len(targets) == 0 {
		return CollectionManifest{}, fmt.Errorf("collection extension requires targets")
	}
	var out CollectionManifest
	err = s.withLock(dir, name, func() error {
		data, err := os.ReadFile(s.collectionPath(dir, name))
		if os.IsNotExist(err) {
			return ErrNotFound
		}
		if err != nil {
			return fmt.Errorf("read collection: %w", err)
		}
		if err := json.Unmarshal(data, &out); err != nil {
			return fmt.Errorf("decode collection: %w", err)
		}
		out = out.Normalized()
		existing := make(map[string]struct{}, len(out.Members))
		for _, member := range out.Members {
			existing[member.Scenario] = struct{}{}
		}
		added := make(map[string]struct{}, len(targets))
		for _, target := range targets {
			scenario, baseline := strings.TrimSpace(target.Scenario), strings.TrimSpace(target.BaselineName)
			if scenario == "" || baseline == "" {
				return fmt.Errorf("collection extension member scenario and baseline name are required")
			}
			if _, found := existing[scenario]; found {
				return fmt.Errorf("collection %q already contains scenario %q; member replacement is forbidden", name, scenario)
			}
			if _, duplicate := added[scenario]; duplicate {
				return fmt.Errorf("collection extension contains duplicate scenario %q", scenario)
			}
			added[scenario] = struct{}{}
			out.Members = append(out.Members, CollectionMember{Scenario: scenario, BaselineName: baseline, Required: target.Required, Status: CollectionMemberPending, UpdatedAt: now.UTC()})
		}
		out.UpdatedAt = now.UTC()
		out = out.Normalized()
		if err := out.Validate(); err != nil {
			return err
		}
		return s.writeRecord(s.collectionPath(dir, name), out)
	})
	if err != nil {
		return CollectionManifest{}, err
	}
	return out, nil
}

// List returns manifests for a scenario. An empty branch lists across every
// branch; a non-empty branch restricts to that branch. Results are sorted
// newest-first by CreatedAt.
func (s *Storage) List(repoID int64, scenario, branch string) ([]BaselineManifest, error) {
	var dirs []string
	if branch != "" {
		dir, err := s.branchDir(repoID, scenario, branch)
		if err != nil {
			return nil, err
		}
		dirs = append(dirs, dir)
	} else {
		root, err := s.scenarioDir(repoID, scenario)
		if err != nil {
			return nil, err
		}
		entries, rerr := os.ReadDir(root)
		if rerr != nil {
			if os.IsNotExist(rerr) {
				return []BaselineManifest{}, nil
			}
			return nil, fmt.Errorf("read scenario baselines: %w", rerr)
		}
		for _, e := range entries {
			if e.IsDir() {
				dirs = append(dirs, filepath.Join(root, e.Name()))
			}
		}
	}

	var out []BaselineManifest
	for _, dir := range dirs {
		entries, err := os.ReadDir(dir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("read baseline dir: %w", err)
		}
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
				continue
			}
			name := strings.TrimSuffix(e.Name(), ".json")
			var m BaselineManifest
			if err := s.withLock(dir, name, func() error {
				path := filepath.Join(dir, e.Name())
				data, rerr := os.ReadFile(path)
				if rerr != nil {
					return rerr
				}
				decoded, migrated, derr := decodeManifest(data, s.nowUTC())
				if derr != nil {
					return derr
				}
				m = decoded
				if migrated {
					return s.writeRecord(path, m)
				}
				return nil
			}); err != nil {
				return nil, fmt.Errorf("load baseline %s: %w", e.Name(), err)
			}
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].CreatedAt.After(out[b].CreatedAt)
	})
	return out, nil
}

func (s *Storage) nowUTC() time.Time { return time.Now().UTC() }

type legacySurfacePointerV1 struct {
	Kind       string    `json:"kind"`
	Ref        string    `json:"ref"`
	CapturedAt time.Time `json:"captured_at"`
}

type legacyManifestV1 struct {
	Name          string                            `json:"name"`
	Scenario      string                            `json:"scenario"`
	Branch        string                            `json:"branch"`
	CreatedAt     time.Time                         `json:"created_at"`
	CreatedBy     string                            `json:"created_by,omitempty"`
	Git           git.State                         `json:"git"`
	Surfaces      map[string]legacySurfacePointerV1 `json:"surfaces"`
	Skipped       map[string]string                 `json:"skipped,omitempty"`
	SchemaVersion int                               `json:"schema_version"`
}

// decodeManifest is the only V1-aware boundary. Successful legacy reads are
// immediately rewritten as V2; the rest of GCT never receives a surface map.
func decodeManifest(data []byte, migratedAt time.Time) (BaselineManifest, bool, error) {
	var version struct {
		SchemaVersion int `json:"schema_version"`
	}
	if err := json.Unmarshal(data, &version); err != nil {
		return BaselineManifest{}, false, fmt.Errorf("decode baseline schema: %w", err)
	}
	if version.SchemaVersion == SchemaVersion {
		var m BaselineManifest
		if err := json.Unmarshal(data, &m); err != nil {
			return BaselineManifest{}, false, fmt.Errorf("decode baseline v2: %w", err)
		}
		if err := m.Validate(); err != nil {
			return BaselineManifest{}, false, err
		}
		return m, false, nil
	}
	if version.SchemaVersion != 1 {
		return BaselineManifest{}, false, fmt.Errorf("unsupported baseline schema version %d", version.SchemaVersion)
	}

	var legacy legacyManifestV1
	if err := json.Unmarshal(data, &legacy); err != nil {
		return BaselineManifest{}, false, fmt.Errorf("decode legacy baseline: %w", err)
	}
	if len(legacy.Skipped) > 0 || len(legacy.Surfaces) != 5 {
		return BaselineManifest{}, false, fmt.Errorf("%w: baseline %q has %d/5 surface pointers and %d skipped entries; recapture it as a comprehensive baseline", ErrLegacyIncomplete, legacy.Name, len(legacy.Surfaces), len(legacy.Skipped))
	}
	runIDs := map[string]struct{}{}
	capturedAt := legacy.CreatedAt
	for surface, pointer := range legacy.Surfaces {
		if strings.TrimSpace(pointer.Ref) == "" || (pointer.Kind != "" && pointer.Kind != "test-genie-run") {
			return BaselineManifest{}, false, fmt.Errorf("%w: baseline %q surface %q has no usable Test Genie run; recapture it", ErrLegacyIncomplete, legacy.Name, surface)
		}
		runIDs[pointer.Ref] = struct{}{}
		if pointer.CapturedAt.After(capturedAt) {
			capturedAt = pointer.CapturedAt
		}
	}
	if len(runIDs) != 1 {
		ids := make([]string, 0, len(runIDs))
		for id := range runIDs {
			ids = append(ids, id)
		}
		sort.Strings(ids)
		return BaselineManifest{}, false, fmt.Errorf("%w: baseline %q points at %s; automatic selection would be dishonest, so recapture it", ErrLegacyMixedRuns, legacy.Name, strings.Join(ids, ", "))
	}
	var runID string
	for id := range runIDs {
		runID = id
	}
	m := BaselineManifest{
		Name: legacy.Name, Scenario: legacy.Scenario, Branch: legacy.Branch,
		CreatedAt: legacy.CreatedAt, CreatedBy: legacy.CreatedBy, Git: legacy.Git,
		Run: RunAnchor{RunID: runID, CapturedAt: capturedAt, CaptureProfile: CaptureProfile},
		Migration: &MigrationInfo{
			FromSchemaVersion: 1,
			MigratedAt:        migratedAt,
			DegradedReasons: []string{
				"legacy baseline has no captured tree digest",
				"legacy baseline has no captured phase-set digest",
				"legacy baseline has no descriptor snapshot identity",
			},
		},
		SchemaVersion: SchemaVersion,
	}
	if err := m.Validate(); err != nil {
		return BaselineManifest{}, false, err
	}
	return m, true, nil
}

func writeManifestAtomic(path string, m BaselineManifest) error {
	return writeJSONAtomic(path, m)
}

func (s *Storage) writeRecord(path string, value any) error {
	if s.writeJSON != nil {
		return s.writeJSON(path, value)
	}
	return writeJSONAtomic(path, value)
}

func writeJSONAtomic(path string, value any) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal baseline: %w", err)
	}
	return writeBytesAtomic(path, data, 0o644)
}

func writeBytesAtomic(path string, data []byte, mode os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create atomic-write directory: %w", err)
	}
	tmp := path + ".tmp"
	f, err := os.OpenFile(tmp, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, mode)
	if err != nil {
		return fmt.Errorf("open baseline tmp: %w", err)
	}
	if _, err = f.Write(data); err == nil {
		err = f.Sync()
	}
	closeErr := f.Close()
	if err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("write baseline tmp: %w", err)
	}
	if closeErr != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("close baseline tmp: %w", closeErr)
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return fmt.Errorf("replace baseline: %w", err)
	}
	if dir, err := os.Open(filepath.Dir(path)); err == nil {
		_ = dir.Sync()
		_ = dir.Close()
	}
	return nil
}
