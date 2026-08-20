package domains

import (
	"context"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

// SurfaceProvider supplies the code-facts-owned view of scenario execution
// surfaces and parse units. Domain extractors use this seam so they do not
// hardcode which top-level surfaces exist.
type SurfaceProvider interface {
	Inspect(ctx context.Context, scenarioDir string) (SurfaceInventory, error)
}

type surfaceInspectionCacheContextKey struct{}

type surfaceInspectionCache struct {
	mu      sync.Mutex
	entries map[string]surfaceInspectionResult
}

type surfaceInspectionResult struct {
	inventory SurfaceInventory
	err       error
}

func withSurfaceInspectionCache(ctx context.Context) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, surfaceInspectionCacheContextKey{}, &surfaceInspectionCache{
		entries: make(map[string]surfaceInspectionResult),
	})
}

func inspectSurfaceCached(ctx context.Context, scenarioDir string, load func() (SurfaceInventory, error)) (SurfaceInventory, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	cache, _ := ctx.Value(surfaceInspectionCacheContextKey{}).(*surfaceInspectionCache)
	if cache == nil {
		return load()
	}

	key := filepath.Clean(scenarioDir)
	cache.mu.Lock()
	defer cache.mu.Unlock()
	if result, ok := cache.entries[key]; ok {
		return result.inventory, result.err
	}
	result := surfaceInspectionResult{}
	result.inventory, result.err = load()
	cache.entries[key] = result
	return result.inventory, result.err
}

func inspectSurfaceProvider(ctx context.Context, provider SurfaceProvider, scenarioDir string) (SurfaceInventory, error) {
	return inspectSurfaceCached(ctx, scenarioDir, func() (SurfaceInventory, error) {
		return provider.Inspect(ctx, scenarioDir)
	})
}

// SurfaceInventory is the narrow code-facts shape cartographer needs for
// domain derivation.
type SurfaceInventory struct {
	Surfaces   []Surface
	ParseUnits []ParseUnit
	Warnings   []ExtractionWarning
}

// Surface is one execution surface reported by code-facts.
type Surface struct {
	ID     string
	Kind   string
	Path   string
	Status string
}

// ParseUnit is one analyzer unit reported by code-facts.
type ParseUnit struct {
	ID         string
	Language   string
	RootPath   string
	ConfigPath string
	Status     string
}

// LocalSurfaceProvider is the explicit degraded path used when code-facts is
// unavailable. It mirrors code-facts' filesystem discovery narrowly enough for
// domain extraction tests and offline operation.
type LocalSurfaceProvider struct{}

func NewLocalSurfaceProvider() *LocalSurfaceProvider { return &LocalSurfaceProvider{} }

var _ SurfaceProvider = (*LocalSurfaceProvider)(nil)

func (*LocalSurfaceProvider) Inspect(_ context.Context, scenarioDir string) (SurfaceInventory, error) {
	inv := SurfaceInventory{}
	for _, spec := range []struct {
		id   string
		kind string
	}{
		{id: "api", kind: "api"},
		{id: "cli", kind: "cli"},
		{id: "ui", kind: "ui"},
	} {
		path := filepath.Join(scenarioDir, spec.id)
		inv.Surfaces = append(inv.Surfaces, Surface{
			ID:     spec.id,
			Kind:   spec.kind,
			Path:   path,
			Status: surfaceStatus(path),
		})
	}
	for _, spec := range []struct {
		id   string
		kind string
	}{
		{id: "sidecar", kind: "sidecar"},
		{id: "sidecars", kind: "sidecar"},
		{id: "workers", kind: "worker"},
		{id: "jobs", kind: "job"},
	} {
		path := filepath.Join(scenarioDir, spec.id)
		if info, err := os.Stat(path); err == nil && info.IsDir() {
			inv.Surfaces = append(inv.Surfaces, Surface{
				ID:     spec.id,
				Kind:   spec.kind,
				Path:   path,
				Status: surfaceStatus(path),
			})
		}
	}
	inv.ParseUnits = localParseUnits(scenarioDir)
	sort.Slice(inv.Surfaces, func(i, j int) bool { return inv.Surfaces[i].ID < inv.Surfaces[j].ID })
	sort.Slice(inv.ParseUnits, func(i, j int) bool { return inv.ParseUnits[i].ID < inv.ParseUnits[j].ID })
	return inv, nil
}

func surfaceStatus(path string) string {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return "known"
	}
	return "missing"
}

func localParseUnits(root string) []ParseUnit {
	var out []ParseUnit
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			if path != root && shouldSkipSurfaceWalkDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		switch d.Name() {
		case "go.mod":
			dir := filepath.Dir(path)
			out = append(out, ParseUnit{
				ID:         relSlash(root, dir),
				Language:   "go",
				RootPath:   dir,
				ConfigPath: path,
				Status:     "proven",
			})
		case "tsconfig.json":
			dir := filepath.Dir(path)
			out = append(out, ParseUnit{
				ID:         relSlash(root, dir),
				Language:   "typescript",
				RootPath:   dir,
				ConfigPath: path,
				Status:     "proven",
			})
		}
		return nil
	})
	return out
}

func shouldSkipSurfaceWalkDir(name string) bool {
	switch name {
	case ".git", "node_modules", "dist", "build", "coverage", ".turbo", ".vite":
		return true
	default:
		return false
	}
}

func relSlash(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == "." {
		return "."
	}
	return filepath.ToSlash(rel)
}

func surfaceByID(inv SurfaceInventory, id string) (Surface, bool) {
	for _, surface := range inv.Surfaces {
		if surface.ID == id && surface.Status != "missing" {
			return surface, true
		}
	}
	return Surface{}, false
}

func surfaceRel(scenarioDir string, path string) string {
	rel, err := filepath.Rel(scenarioDir, path)
	if err != nil {
		return ""
	}
	return filepath.ToSlash(rel)
}

func readChildDirs(root string) ([]os.DirEntry, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, fmt.Errorf("read %s: %w", filepath.ToSlash(root), err)
	}
	out := entries[:0]
	for _, ent := range entries {
		if ent.IsDir() && strings.TrimSpace(ent.Name()) != "" {
			out = append(out, ent)
		}
	}
	return out, nil
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}
