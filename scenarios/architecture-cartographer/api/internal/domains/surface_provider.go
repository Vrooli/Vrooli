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
	Surfaces []Surface
	Warnings []ExtractionWarning
}

// Surface is one execution surface reported by code-facts.
type Surface struct {
	ID     string
	Kind   string
	Path   string
	Status string
}

// LocalSurfaceProvider is the explicit degraded path used when code-facts is
// unavailable. It discovers execution surfaces while keeping offline domain
// extraction available; parse units remain owned by Code Facts.
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
	sort.Slice(inv.Surfaces, func(i, j int) bool { return inv.Surfaces[i].ID < inv.Surfaces[j].ID })
	return inv, nil
}

func surfaceStatus(path string) string {
	if info, err := os.Stat(path); err == nil && info.IsDir() {
		return "known"
	}
	return "missing"
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
