package baseline

import (
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

// ErrNotFound is returned when a baseline does not exist.
var ErrNotFound = errors.New("baseline not found")

// ErrAlreadyExists is returned by Save (create mode) when a baseline with the
// same scenario+branch+name already exists.
var ErrAlreadyExists = errors.New("baseline already exists")

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
	Surface    string           `json:"surface,omitempty"`
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
	Include      []string         `json:"include,omitempty"`
	Fast         bool             `json:"fast"`
	CreatedBy    string           `json:"created_by,omitempty"`
	Reason       string           `json:"reason,omitempty"`
	Want         []string         `json:"want,omitempty"`
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
			Include:   i.Include,
			Fast:      i.Fast,
			Capture:   true,
			CreatedBy: i.CreatedBy,
			Reason:    i.Reason,
		},
		Want:         i.Want,
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
		Surface:    i.Surface,
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
		data, err := json.MarshalIndent(intent, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal snapshot intent: %w", err)
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			return fmt.Errorf("write snapshot intent tmp: %w", err)
		}
		if err := os.Rename(tmp, path); err != nil {
			return fmt.Errorf("replace snapshot intent: %w", err)
		}
		return nil
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
		data, err := json.MarshalIndent(cd, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal diff cache: %w", err)
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			return fmt.Errorf("write diff cache tmp: %w", err)
		}
		if err := os.Rename(tmp, path); err != nil {
			return fmt.Errorf("replace diff cache: %w", err)
		}
		return nil
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
		data, err := json.MarshalIndent(intent, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal diff intent: %w", err)
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			return fmt.Errorf("write diff intent tmp: %w", err)
		}
		if err := os.Rename(tmp, path); err != nil {
			return fmt.Errorf("replace diff intent: %w", err)
		}
		return nil
	})
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
		data, err := json.MarshalIndent(m, "", "  ")
		if err != nil {
			return fmt.Errorf("marshal baseline: %w", err)
		}
		tmp := path + ".tmp"
		if err := os.WriteFile(tmp, data, 0o644); err != nil {
			return fmt.Errorf("write baseline tmp: %w", err)
		}
		if err := os.Rename(tmp, path); err != nil {
			return fmt.Errorf("replace baseline: %w", err)
		}
		return nil
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
		return json.Unmarshal(data, &m)
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
			data, rerr := os.ReadFile(filepath.Join(dir, e.Name()))
			if rerr != nil {
				continue
			}
			var m BaselineManifest
			if json.Unmarshal(data, &m) != nil {
				continue
			}
			out = append(out, m)
		}
	}
	sort.SliceStable(out, func(a, b int) bool {
		return out[a].CreatedAt.After(out[b].CreatedAt)
	})
	return out, nil
}
