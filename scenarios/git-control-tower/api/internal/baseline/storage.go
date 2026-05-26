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

func (s *Storage) manifestPath(dir, name string) string {
	return filepath.Join(dir, sanitizeSegment(name)+".json")
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
