package retention

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

// FileConfig configures the builtin pruner for one regenerable file.
type FileConfig struct {
	Path           string
	ProtectedRoots []string
	Now            func() time.Time
}

// FilePruner removes one declared file atomically when its retention bound is
// exceeded. It is intentionally narrow: it never truncates or rewrites a file,
// and a missing file is already within budget.
type FilePruner struct {
	path           string
	protectedRoots []string
	now            func() time.Time
}

func NewFilePruner(cfg FileConfig) (*FilePruner, error) {
	if cfg.Path == "" {
		return nil, fmt.Errorf("file pruner: Path is required")
	}
	if !filepath.IsAbs(cfg.Path) {
		return nil, fmt.Errorf("file pruner: Path %q must be absolute", cfg.Path)
	}
	protectedRoots, err := normalizeProtectedRoots(cfg.ProtectedRoots)
	if err != nil {
		return nil, fmt.Errorf("file pruner: %w", err)
	}
	if cfg.Now == nil {
		cfg.Now = time.Now
	}
	return &FilePruner{path: filepath.Clean(cfg.Path), protectedRoots: protectedRoots, now: cfg.Now}, nil
}

func (p *FilePruner) Measure(context.Context) (Usage, error) {
	info, err := os.Stat(p.path)
	if os.IsNotExist(err) {
		return Usage{}, nil
	}
	if err != nil {
		return Usage{}, fmt.Errorf("stat %s: %w", p.path, err)
	}
	if info.IsDir() {
		return Usage{}, fmt.Errorf("file target %s is a directory", p.path)
	}
	return Usage{Bytes: info.Size(), Items: 1}, nil
}

func (p *FilePruner) Prune(ctx context.Context, budget Budget) (Result, error) {
	before, err := p.Measure(ctx)
	if err != nil {
		return Result{Budget: budget.Name, Incomplete: false}, err
	}
	result := Result{Budget: budget.Name, Before: before, After: before}
	if before.Items == 0 {
		return result, nil
	}
	info, err := os.Stat(p.path)
	if err != nil {
		return result, fmt.Errorf("stat %s: %w", p.path, err)
	}
	overAge := budget.HasAgeBound() && info.ModTime().Before(p.now().Add(-budget.MaxAge))
	overBytes := budget.HasByteBound() && before.Bytes > budget.MaxBytes
	if !overAge && !overBytes {
		return result, nil
	}
	if err := ctx.Err(); err != nil {
		result.Incomplete = true
		return result, err
	}
	if protectedPathOverlap(p.path, p.protectedRoots) {
		return result, fmt.Errorf("refusing to remove protected path %s", p.path)
	}
	if err := os.Remove(p.path); err != nil {
		return result, fmt.Errorf("remove %s: %w", p.path, err)
	}
	result.Deleted = 1
	result.FreedBytes = before.Bytes
	result.After = Usage{}
	if overBytes {
		result.BoundBy = BoundBytes
	} else {
		result.BoundBy = BoundAge
	}
	return result, nil
}
