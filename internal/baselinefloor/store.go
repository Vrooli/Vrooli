package baselinefloor

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/config"
)

// engagementDirPrefix is the per-engagement directory prefix under a scenario's
// cache dir: ~/.cache/vrooli/<scenario>/baseline-<slug>/.
const engagementDirPrefix = "baseline-"

// manifestFile is the floor-owned engagement manifest filename, co-located with
// the restore point inside the engagement dir.
const manifestFile = "engagement.json"

// restorePointDir is the restore-point copy subdirectory inside the engagement
// dir. Keeping it a subdir leaves the manifest a clean sibling.
const restorePointDir = "restore-point"

// migrationsSubdir is the managed per-engagement migration folder inside the
// engagement dir. Ordered *.sql scripts here are applied to live (transactional,
// dry-run-first) by promote — the storage-steer "migration scripts live in the
// managed per-baseline folder" contract (Baseline Modes §8).
const migrationsSubdir = "migrations"

// DefaultCacheRoot resolves the Baseline Modes cache root —
// ${XDG_CACHE_HOME:-<home>/.cache}/vrooli — sudo-aware via config.HomeDir so a
// `sudo vrooli` invocation still writes under the invoking user's home rather
// than /root. This is the single place the root is computed.
func DefaultCacheRoot() (string, error) {
	if xdg := strings.TrimSpace(os.Getenv("XDG_CACHE_HOME")); xdg != "" && filepath.IsAbs(xdg) {
		return filepath.Join(xdg, "vrooli"), nil
	}
	home, err := config.HomeDir()
	if err != nil {
		return "", fmt.Errorf("baselinefloor: resolve home: %w", err)
	}
	return filepath.Join(home, ".cache", "vrooli"), nil
}

// Store is the floor's filesystem handle: it resolves the engagement directory,
// restore point, and manifest paths under a cache root, and reads/writes the
// engagement manifests. Construct it with DefaultStore (production) or NewStore
// (tests, with an injected temp root).
//
// The Store holds NO open handles and NO mutable state; it is safe to construct
// freely and share. All running-process truth lives in the scenarioruntime
// registry, never here.
type Store struct {
	root string
}

// NewStore returns a Store rooted at an explicit cache root. Tests inject a temp
// dir; production code uses DefaultStore.
func NewStore(root string) *Store {
	return &Store{root: root}
}

// DefaultStore resolves the production cache root via DefaultCacheRoot.
func DefaultStore() (*Store, error) {
	root, err := DefaultCacheRoot()
	if err != nil {
		return nil, err
	}
	return NewStore(root), nil
}

// Root returns the cache root this Store writes under.
func (s *Store) Root() string {
	return s.root
}

// scenarioDir is ~/.cache/vrooli/<scenario>/.
func (s *Store) scenarioDir(scenario string) string {
	return filepath.Join(s.root, scenario)
}

// EngagementDir resolves ~/.cache/vrooli/<scenario>/baseline-<slug>/.
func (s *Store) EngagementDir(scenario, slug string) string {
	return filepath.Join(s.scenarioDir(scenario), engagementDirPrefix+slug)
}

// RestorePointPath resolves the restore-point copy directory inside an
// engagement: ~/.cache/vrooli/<scenario>/baseline-<slug>/restore-point/.
func (s *Store) RestorePointPath(scenario, slug string) string {
	return filepath.Join(s.EngagementDir(scenario, slug), restorePointDir)
}

// ManifestPath resolves the engagement.json path inside an engagement.
func (s *Store) ManifestPath(scenario, slug string) string {
	return filepath.Join(s.EngagementDir(scenario, slug), manifestFile)
}

// MigrationsPath resolves the managed per-engagement migration folder:
// ~/.cache/vrooli/<scenario>/baseline-<slug>/migrations/. Promote applies the
// ordered *.sql scripts here to live within a transaction (dry-run-first); a
// missing/empty folder is the shape-unchanged fast path.
func (s *Store) MigrationsPath(scenario, slug string) string {
	return filepath.Join(s.EngagementDir(scenario, slug), migrationsSubdir)
}

// Clean removes an entire engagement directory (restore point + manifest). It is
// idempotent: a missing engagement is not an error. This is the teardown
// primitive promote/abandon/gc call once an engagement closes.
func (s *Store) Clean(scenario, slug string) error {
	if err := os.RemoveAll(s.EngagementDir(scenario, slug)); err != nil {
		return fmt.Errorf("baselinefloor: clean engagement %s/%s: %w", scenario, slug, err)
	}
	return nil
}
