package retention

import (
	"fmt"
	"log/slog"
	"time"
)

// BuiltinConfig supplies what the framework pruners need beyond the manifest:
// the resolved absolute path of each target and, for a SQLite target, the open
// database handle.
type BuiltinConfig struct {
	// ResolvePath returns the absolute on-disk path for a target. Required.
	// Callers pass the api-core/storage resolution so a shadow variant resolves
	// under its own namespace.
	ResolvePath func(Target) (string, error)
	// OpenDatabase returns a handle to the SQLite database at path. Required
	// only when a sqlite_table budget is declared.
	OpenDatabase func(path string) (Execer, error)
	// BatchSize is rows per delete statement. Defaults to DefaultBatchSize.
	BatchSize int
	// ReclaimPercent is how far below its ceiling a byte-bound prune reduces a
	// target. Defaults to DefaultReclaimPercent.
	ReclaimPercent int
	// BatchPause is how long the pruner waits between delete batches, yielding
	// the database to the serving path.
	BatchPause time.Duration
	// AllowFullVacuum permits the one-time full VACUUM that converts a
	// database to incremental auto-vacuum. Off by default; it belongs to an
	// explicit operator command, never to startup.
	AllowFullVacuum bool
	// MaxDuration bounds one Prune call's wall clock. Zero means unbounded.
	MaxDuration time.Duration
	// Now supplies the current time. Defaults to time.Now.
	Now func() time.Time
	// FreeSpace probes available bytes. Defaults to FreeSpace.
	FreeSpace FreeSpaceFunc
	// Logger receives pruner detail. Defaults to slog.Default.
	Logger *slog.Logger
}

// NewBuiltinFactory returns the BuiltinFactory an Engine uses for budgets that
// declare pruner "builtin". This is the whole no-Go-code path: a component
// declares a budget, the engine builds the pruner, and the component writes
// nothing.
func NewBuiltinFactory(cfg BuiltinConfig) (BuiltinFactory, error) {
	if cfg.ResolvePath == nil {
		return nil, fmt.Errorf("builtin retention pruners: ResolvePath is required")
	}
	return func(spec Spec) (Pruner, error) {
		path, err := cfg.ResolvePath(spec.Target)
		if err != nil {
			return nil, fmt.Errorf("resolve %s target: %w", spec.Target.Kind, err)
		}
		switch spec.Target.Kind {
		case TargetSQLiteTable:
			if cfg.OpenDatabase == nil {
				return nil, fmt.Errorf("builtin retention pruners: OpenDatabase is required for sqlite_table budget %q", spec.Budget.Name)
			}
			db, err := cfg.OpenDatabase(path)
			if err != nil {
				return nil, fmt.Errorf("open %s: %w", path, err)
			}
			return NewSQLiteTablePruner(SQLiteTableConfig{
				DB:              db,
				Path:            path,
				Table:           spec.Target.Table,
				TimeColumn:      spec.Target.TimeColumn,
				BatchSize:       cfg.BatchSize,
				ReclaimPercent:  cfg.ReclaimPercent,
				BatchPause:      cfg.BatchPause,
				Now:             cfg.Now,
				FreeSpace:       cfg.FreeSpace,
				AllowFullVacuum: cfg.AllowFullVacuum,
				MaxDuration:     cfg.MaxDuration,
				Logger:          cfg.Logger,
			})
		case TargetDirectory:
			return NewDirectoryPruner(DirectoryConfig{
				Path:   path,
				Now:    cfg.Now,
				Logger: cfg.Logger,
			})
		case TargetFile:
			return NewFilePruner(FileConfig{Path: path, Now: cfg.Now})
		default:
			return nil, fmt.Errorf("%w: no builtin pruner for kind %q", ErrInvalidTarget, spec.Target.Kind)
		}
	}, nil
}
