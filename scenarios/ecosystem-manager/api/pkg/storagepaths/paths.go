// Package storagepaths is the single production seam that resolves Ecosystem
// Manager's runtime storage locations through github.com/vrooli/api-core/storage.
//
// It is the ONLY production code allowed to call storage.NewResolver. Every
// other package receives already-resolved typed paths (queue dir, SQLite DB,
// system log dir, task-run log dir, settings path) so storage-class decisions
// stay reviewable and testable in one place. In particular, GetQueueDir on the
// task storage is not a license to derive sibling log/db/config paths — those
// are resolved here independently.
//
// Variant isolation: the scenario namespace flows through
// storage.ScenarioNamespace, so a Baseline Modes shadow engagement (which
// injects VROOLI_STORAGE_NAMESPACE) lands every storage class beside
// "ecosystem-manager_shadow" and never shares live's queue, database, or logs.
package storagepaths

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/api-core/storage"
)

// Scenario is the compile-time slug used as the storage namespace fallback when
// the lifecycle does not inject VROOLI_STORAGE_NAMESPACE (local runs, tests).
const Scenario = "ecosystem-manager"

// Paths holds the resolved runtime storage locations. Construct it via Resolve
// (production) or ForTest (tests). Parent directories are created at the point
// of use, not here, so a Paths value is cheap and side-effect free.
type Paths struct {
	// QueueDir is the root for file-backed task queue status directories,
	// under storage.ClassData.
	QueueDir string
	// DBPath is the SQLite database file, under storage.ClassData.
	DBPath string
	// SystemLogDir is the audit-log directory, under storage.ClassLogs.
	SystemLogDir string
	// TaskRunLogDir is the per-task-run execution log directory, under
	// storage.ClassLogs. It is resolved here, not derived from QueueDir.
	TaskRunLogDir string
	// SettingsPath is the mutable settings file, under storage.ClassConfig.
	SettingsPath string
}

// Resolve builds the production storage paths from api-core/storage using the
// variant-aware scenario namespace.
func Resolve() (Paths, error) {
	resolver, err := storage.NewResolver(storage.ResolverConfig{
		AppID:   "vrooli",
		Profile: storage.ProfileAuto,
	})
	if err != nil {
		return Paths{}, fmt.Errorf("create storage resolver: %w", err)
	}
	scenarioID, err := storage.ScenarioNamespace(Scenario)
	if err != nil {
		return Paths{}, fmt.Errorf("resolve %s storage namespace: %w", Scenario, err)
	}
	dirs, err := resolver.Resolve(storage.Options{ScenarioID: scenarioID})
	if err != nil {
		return Paths{}, fmt.Errorf("resolve %s storage classes: %w", Scenario, err)
	}
	return ForDirs(dirs.DataDir, dirs.LogsDir, dirs.ConfigDir), nil
}

// ForDirs assembles typed paths from the three storage-class roots. Exported so
// tests can build a Paths from arbitrary directories without a resolver.
func ForDirs(dataDir, logsDir, configDir string) Paths {
	return Paths{
		QueueDir:      filepath.Join(dataDir, "queue"),
		DBPath:        filepath.Join(dataDir, Scenario+".db"),
		SystemLogDir:  logsDir,
		TaskRunLogDir: filepath.Join(logsDir, "task-runs"),
		SettingsPath:  filepath.Join(configDir, "settings.json"),
	}
}

// ForTest points every storage class at a subdirectory of root (typically a
// t.TempDir()), so a test exercises the same path seam as production without
// touching the real storage root.
func ForTest(root string) Paths {
	return ForDirs(
		filepath.Join(root, "data"),
		filepath.Join(root, "logs"),
		filepath.Join(root, "config"),
	)
}

// SQLiteDSN resolves the database file path and wraps it in the canonical
// modernc.org/sqlite DSN. Resolution order:
//
//  1. SQLITE_PATH env — the canonical operator/test override.
//  2. SQLITE_DB env — alias accepted for symmetry with other scenarios.
//  3. p.DBPath — the storage-resolved default.
//
// The parent directory is created as a side effect so the driver can open the
// file. A value already prefixed with "file:" is returned unchanged.
func (p Paths) SQLiteDSN() (string, error) {
	path := p.DBPath
	if v := strings.TrimSpace(os.Getenv("SQLITE_PATH")); v != "" {
		path = v
	} else if v := strings.TrimSpace(os.Getenv("SQLITE_DB")); v != "" {
		path = v
	}
	if strings.HasPrefix(path, "file:") {
		return path, nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", fmt.Errorf("prepare sqlite directory: %w", err)
	}
	return fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(ON)&_pragma=journal_mode(WAL)&_pragma=busy_timeout(10000)&_pragma=cache_size(-2000)&_pragma=synchronous(NORMAL)&_pragma=temp_store(MEMORY)",
		path,
	), nil
}
