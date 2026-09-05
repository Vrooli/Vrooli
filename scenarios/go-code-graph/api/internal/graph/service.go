package graph

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/vrooli/api-core/schedule"
	"golang.org/x/tools/go/packages"
)

// Service orchestrates the Extract flow: validate input → acquire the
// per-path mutex → call the PackagesLoader seam → normalize → return.
//
// All non-trivial work flows through seams (PackagesLoader, PathMutex)
// so the Service itself is pure orchestration and exhaustively
// testable with FakeLoader.
type Service struct {
	loader                 PackagesLoader
	mu                     *PathMutex
	limiter                *extractionLimiter
	cache                  ExtractionCache
	environmentFingerprint string
	clock                  schedule.Clock
}

// ExtractionStats exposes phase timings without changing the domain graph
// itself. They are intended for diagnostics and benchmark assertions.
type ExtractionStats struct {
	Fingerprint time.Duration
	Load        time.Duration
	Normalize   time.Duration
	CacheHit    bool
}

// NewService wires the production Service. Both arguments are required;
// the caller (api/main.go) owns construction so the mutex can be shared
// with the rewrite domain (OT-P0-006).
func NewService(loader PackagesLoader, mu *PathMutex) *Service {
	return NewServiceWithMaxConcurrentAndClock(loader, mu, DefaultMaxConcurrentExtracts, schedule.System())
}

// NewServiceWithMaxConcurrent is the testable constructor for the production
// extraction guard. A non-positive max disables the global cap.
func NewServiceWithMaxConcurrent(loader PackagesLoader, mu *PathMutex, maxConcurrent int) *Service {
	return NewServiceWithMaxConcurrentAndClock(loader, mu, maxConcurrent, schedule.System())
}

// NewServiceWithMaxConcurrentAndClock is the constructor used by deterministic
// tests that need to control phase timing.
func NewServiceWithMaxConcurrentAndClock(loader PackagesLoader, mu *PathMutex, maxConcurrent int, clock schedule.Clock) *Service {
	if clock == nil {
		clock = schedule.System()
	}
	return &Service{
		loader:  loader,
		mu:      mu,
		limiter: newExtractionLimiter(maxConcurrent),
		clock:   clock,
	}
}

// NewServiceWithCache wires the optional best-effort extraction cache. Cache
// failures never make an otherwise valid extraction fail.
func NewServiceWithCache(loader PackagesLoader, mu *PathMutex, maxConcurrent int, cache ExtractionCache) *Service {
	return NewServiceWithCacheAndEnvironment(loader, mu, maxConcurrent, cache, "")
}

// NewServiceWithCacheAndEnvironment wires a cache and the loader environment
// fingerprint captured by the composition root.
func NewServiceWithCacheAndEnvironment(loader PackagesLoader, mu *PathMutex, maxConcurrent int, cache ExtractionCache, environmentFingerprint string) *Service {
	return NewServiceWithCacheAndEnvironmentAndClock(loader, mu, maxConcurrent, cache, environmentFingerprint, schedule.System())
}

// NewServiceWithCacheAndEnvironmentAndClock is the fully injectable
// composition constructor for the graph service.
func NewServiceWithCacheAndEnvironmentAndClock(loader PackagesLoader, mu *PathMutex, maxConcurrent int, cache ExtractionCache, environmentFingerprint string, clock schedule.Clock) *Service {
	service := NewServiceWithMaxConcurrentAndClock(loader, mu, maxConcurrent, clock)
	service.cache = cache
	service.environmentFingerprint = environmentFingerprint
	return service
}

// Extract validates the input, locks the absolute module path,
// invokes the loader, and normalizes the result. Errors are typed
// ExtractError so handlers can map them to Connect codes via
// ErrorToConnectCode.
func (s *Service) Extract(ctx context.Context, in ExtractInput) (Graph, []Warning, error) {
	graph, warnings, _, err := s.ExtractWithStats(ctx, in)
	return graph, warnings, err
}

// ExtractWithStats performs extraction and returns phase timings for
// diagnostics. The default profile remains full semantic extraction.
func (s *Service) ExtractWithStats(ctx context.Context, in ExtractInput) (Graph, []Warning, ExtractionStats, error) {
	var stats ExtractionStats
	if strings.TrimSpace(in.ModulePath) == "" {
		return Graph{}, nil, stats, ExtractError{
			Kind:    ExtractErrorInvalidInput,
			Message: "module_path is required",
		}
	}
	profile := in.Profile.normalized()
	patterns, err := normalizePackagePatterns(in.PackagePatterns)
	if err != nil {
		return Graph{}, nil, stats, err
	}

	abs, err := filepath.Abs(in.ModulePath)
	if err != nil {
		return Graph{}, nil, stats, ExtractError{
			Kind:    ExtractErrorPathUnreadable,
			Path:    in.ModulePath,
			Message: "resolve absolute path",
			Cause:   err,
		}
	}

	if err := preflightProject(abs, len(patterns) > 0); err != nil {
		return Graph{}, nil, stats, err
	}

	unlock := s.mu.Lock(abs)
	defer unlock()

	release, err := s.limiter.acquire(ctx)
	if err != nil {
		return Graph{}, nil, stats, ExtractError{
			Kind:    ExtractErrorInternal,
			Path:    abs,
			Message: "acquire extraction slot",
			Cause:   err,
		}
	}
	defer release()

	loadOptions := LoadOptions{
		IncludeVendor:          in.IncludeVendor,
		Profile:                profile,
		PackagePatterns:        patterns,
		EnvironmentFingerprint: s.environmentFingerprint,
	}
	var cacheKey string
	if s.cache != nil {
		fingerprintStart := s.clock.Now()
		fingerprint, fingerprintErr := moduleFingerprint(abs, loadOptions)
		stats.Fingerprint = schedule.Since(fingerprintStart)
		if fingerprintErr == nil {
			cacheKey = extractionCacheKey(abs, loadOptions, fingerprint)
			if graph, warnings, ok := s.cache.Get(cacheKey); ok {
				stats.CacheHit = true
				return graph, warnings, stats, nil
			}
		}
	}

	loadStart := s.clock.Now()
	pkgs, err := s.loader.Load(ctx, abs, loadOptions)
	stats.Load = schedule.Since(loadStart)
	if err != nil {
		return Graph{}, nil, stats, ExtractError{
			Kind:    ExtractErrorInternal,
			Path:    abs,
			Message: "packages loader",
			Cause:   err,
		}
	}

	if !in.IncludeVendor {
		pkgs = filterVendorPackages(pkgs)
	}

	normalizeStart := s.clock.Now()
	graph, warnings := Normalize(pkgs, abs)
	stats.Normalize = schedule.Since(normalizeStart)
	if s.cache != nil && cacheKey != "" {
		_ = s.cache.Put(cacheKey, graph, warnings)
	}
	return graph, warnings, stats, nil
}

func normalizePackagePatterns(patterns []string) ([]string, error) {
	if len(patterns) == 0 {
		return nil, nil
	}
	out := make([]string, 0, len(patterns))
	seen := make(map[string]struct{}, len(patterns))
	for _, pattern := range patterns {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			return nil, ExtractError{Kind: ExtractErrorInvalidInput, Message: "package_patterns contains an empty pattern"}
		}
		if filepath.IsAbs(pattern) || strings.HasPrefix(filepath.Clean(pattern), "..") {
			return nil, ExtractError{Kind: ExtractErrorInvalidInput, Message: "package_patterns must be module-relative Go patterns"}
		}
		if pattern != "." && !strings.HasPrefix(pattern, "./") {
			return nil, ExtractError{Kind: ExtractErrorInvalidInput, Message: "package_patterns must start with ./ or be ."}
		}
		if _, ok := seen[pattern]; ok {
			continue
		}
		seen[pattern] = struct{}{}
		out = append(out, pattern)
	}
	return out, nil
}

// filterVendorPackages drops packages whose loaded source directory sits
// inside a `vendor/` subtree of the module root. This implements
// REQ-P1-003: with IncludeVendor=false (the default) vendored packages
// must not appear in the extracted graph even when the host module
// vendored its dependencies.
//
// Filtering is directory-based, not import-path-based, because Go modules
// rewrite vendored imports back to their original paths
// (e.g. github.com/foo/bar) while keeping the source under
// moduleRoot/vendor/github.com/foo/bar/. PkgPath alone is therefore
// not sufficient.
func filterVendorPackages(pkgs []*packages.Package) []*packages.Package {
	vendorSeg := string(filepath.Separator) + "vendor" + string(filepath.Separator)
	out := pkgs[:0:0]
	for _, p := range pkgs {
		if p == nil {
			continue
		}
		dir := packageDir(p)
		if dir == "" {
			out = append(out, p)
			continue
		}
		if strings.Contains(dir, vendorSeg) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// packageDir returns the directory holding a package's first Go file,
// or "" if the package has no Go files. Used for vendor-filter source
// classification.
func packageDir(p *packages.Package) string {
	if len(p.GoFiles) > 0 {
		return filepath.Dir(p.GoFiles[0])
	}
	if len(p.CompiledGoFiles) > 0 {
		return filepath.Dir(p.CompiledGoFiles[0])
	}
	return ""
}

// preflightProject inspects the module path BEFORE the loader runs
// so we can return precise typed errors for the catastrophic cases
// (no go.mod, multiple go.mod, go.work, path unreadable). The loader
// itself would fail with a generic error.
func preflightProject(abs string, scoped bool) error {
	info, err := os.Stat(abs)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return ExtractError{Kind: ExtractErrorPathUnreadable, Path: abs, Message: "path does not exist", Cause: err}
		}
		return ExtractError{Kind: ExtractErrorPathUnreadable, Path: abs, Cause: err}
	}
	if !info.IsDir() {
		return ExtractError{Kind: ExtractErrorPathUnreadable, Path: abs, Message: "module_path is not a directory"}
	}

	if _, err := os.Stat(filepath.Join(abs, "go.work")); err == nil {
		return ExtractError{Kind: ExtractErrorWorkspaceUnsupported, Path: abs, Message: "go.work present"}
	}

	goMods, err := findGoMods(abs, scoped)
	if err != nil {
		return ExtractError{Kind: ExtractErrorPathUnreadable, Path: abs, Cause: err}
	}
	switch len(goMods) {
	case 0:
		return ExtractError{Kind: ExtractErrorNoGoMod, Path: abs, Message: "no go.mod under module_path"}
	case 1:
		// Common case.
	default:
		return ExtractError{Kind: ExtractErrorMultipleGoMod, Path: abs, Message: "multiple go.mod files found"}
	}
	return nil
}

// findGoMods returns the list of go.mod files under abs, descending
// into subdirectories but skipping vendor/, testdata/, and any path
// whose basename starts with "." (the loader skips those anyway).
func findGoMods(abs string, scoped bool) ([]string, error) {
	var found []string
	if scoped {
		if _, err := os.Stat(filepath.Join(abs, "go.mod")); err == nil {
			return []string{filepath.Join(abs, "go.mod")}, nil
		}
		return nil, nil
	}
	err := filepath.WalkDir(abs, func(p string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			base := d.Name()
			if base == "vendor" || base == "testdata" || (len(base) > 1 && base[0] == '.' && p != abs) {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == "go.mod" {
			found = append(found, p)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	return found, nil
}
