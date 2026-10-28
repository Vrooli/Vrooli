package health

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/envkit-go"
	"golang.org/x/mod/modfile"
	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

type freshnessReport struct {
	Clean       bool                       `json:"clean"`
	Root        string                     `json:"root"`
	Mode        string                     `json:"mode"`
	Touched     []string                   `json:"touched,omitempty"`
	Surfaces    []freshnessSurfaceReport   `json:"surfaces"`
	Exclusions  []freshnessExclusionReport `json:"exclusions,omitempty"`
	Summary     freshnessSummary           `json:"summary"`
	NextActions []freshnessAction          `json:"next_actions,omitempty"`
	ElapsedMs   int64                      `json:"elapsed_ms"`
}

type freshnessSummary struct {
	Checked       int `json:"checked"`
	Clean         int `json:"clean"`
	Stale         int `json:"stale"`
	Errors        int `json:"errors"`
	NeedsDownload int `json:"needs_download"`
	Skipped       int `json:"skipped"`
}

type freshnessSurfaceReport struct {
	Scenario        string   `json:"scenario"`
	Surface         string   `json:"surface"`
	GoModPath       string   `json:"go_mod_path"`
	Status          string   `json:"status"`
	DiffPaths       []string `json:"diff_paths,omitempty"`
	ImpactedBy      []string `json:"impacted_by,omitempty"`
	ImpactedByCount int      `json:"impacted_by_count,omitempty"`
	Error           string   `json:"error,omitempty"`
	CacheHit        bool     `json:"cache_hit,omitempty"`
}

type freshnessAction struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Command    string `json:"command"`
	Fixability string `json:"fixability,omitempty"`
}

type freshnessExclusionReport struct {
	GoModPath string `json:"go_mod_path"`
	Reason    string `json:"reason"`
}

type goSurface struct {
	scenario                   string
	surface                    string
	goMod                      string
	module                     string
	requires                   map[string]struct{}
	replaces                   map[string]string
	moduleFingerprint          string
	moduleFingerprintReady     bool
	dependencyFingerprint      string
	dependencyFingerprintReady bool
	cacheKey                   string
	cacheKeyReady              bool
}

type goModule struct {
	ambiguous bool
	dir       string
	module    string
	requires  map[string]struct{}
	replaces  map[string]string
}

type moduleExclusion struct {
	Path   string
	Reason string
}

var freshnessExclusions = []moduleExclusion{
	{Path: "scenarios/browser-automation-studio/bas/seeds/go.mod", Reason: "synthetic BAS seed fixture"},
	{Path: "scenarios/go-code-graph/bas/fixtures/go-cycles/go.mod", Reason: "synthetic BAS fixture"},
	{Path: "scenarios/go-code-graph/bas/fixtures/go-usage-facts/go.mod", Reason: "synthetic BAS fixture"},
	{Path: "scenarios/go-code-graph/bas/fixtures/go-tests/go.mod", Reason: "synthetic BAS fixture"},
	{Path: "scenarios/go-code-graph/bas/fixtures/go-mislocated/go.mod", Reason: "synthetic BAS fixture"},
}

func runFreshness(args []string) error {
	fs := support.NewFlagSet("freshness")
	var repoRoot string
	var concurrency int
	var touched, all, apply, build, jsonOutput bool
	var module string
	var timeout time.Duration
	var noCache bool
	fs.StringVar(&repoRoot, "repo-root", "", "Repository root (defaults to current workspace)")
	fs.IntVar(&concurrency, "concurrency", 8, "Maximum package surfaces to check concurrently")
	fs.BoolVar(&touched, "touched", false, "Only check surfaces impacted by changed in-repo modules")
	fs.BoolVar(&all, "all", false, "Check every discovered Go scenario surface")
	fs.BoolVar(&apply, "apply", false, "Run go mod tidy on stale surfaces")
	fs.BoolVar(&build, "build", false, "Run go build ./... after tidy checks")
	fs.StringVar(&module, "module", "", "Only check one Go module by repository-relative path or module path")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.DurationVar(&timeout, "timeout", 5*time.Minute, "Overall freshness deadline")
	fs.BoolVar(&noCache, "no-cache", false, "Do not read or write the freshness result cache")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	moduleSpecified := false
	fs.Visit(func(f *flag.Flag) {
		if f.Name == "module" {
			moduleSpecified = true
		}
	})
	if moduleSpecified && strings.TrimSpace(module) == "" {
		return fmt.Errorf("--module requires a nonempty module path")
	}
	if len(fs.Args()) != 0 || (touched && all) || (moduleSpecified && (touched || all)) {
		return fmt.Errorf("usage: %s freshness [--touched|--all|--module <path>] [--apply] [--build] [--concurrency <n>] [--repo-root <path>] [--json]", support.AppName)
	}
	if !touched && !all && module == "" {
		touched = true
	}
	root, err := resolveRepoRoot(repoRoot)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	report, err := checkFreshness(ctx, root, freshnessRequest{touched: touched, all: all, module: module, apply: apply, build: build, noCache: noCache, concurrency: concurrency})
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintReportJSON(report)
	}
	return printFreshnessReport(report)
}

type freshnessRequest struct {
	touched     bool
	all         bool
	module      string
	apply       bool
	build       bool
	noCache     bool
	concurrency int
}

func checkFreshness(ctx context.Context, root string, req freshnessRequest) (freshnessReport, error) {
	start := time.Now()
	modules, err := discoverGoModules(root)
	if err != nil {
		return freshnessReport{}, err
	}
	// An existing repository path explicitly names the owner even when a
	// retained fixture declares the same module ID. ID-only ambiguity refuses.
	if req.module != "" {
		candidate := filepath.Join(root, filepath.FromSlash(strings.TrimSpace(req.module)))
		if filepath.Base(candidate) != "go.mod" {
			candidate = filepath.Join(candidate, "go.mod")
		}
		if resolved, resolveErr := filepath.EvalSymlinks(candidate); resolveErr == nil {
			canonicalRoot, err := filepath.EvalSymlinks(root)
			if err != nil {
				return freshnessReport{}, err
			}
			rel, err := filepath.Rel(canonicalRoot, resolved)
			if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) || filepath.IsAbs(rel) {
				return freshnessReport{}, fmt.Errorf("module path escapes repository root")
			}
			view, err := parseGoModFile(resolved)
			if err != nil {
				return freshnessReport{}, err
			}
			if !excludedGoMod(candidate, root) {
				modules[view.module] = goModule{dir: filepath.Dir(candidate), module: view.module, requires: view.requires, replaces: view.replaces}
			}
		}
	}

	surfaces, err := discoverGoSurfacesFromModules(root, modules)
	if err != nil {
		return freshnessReport{}, err
	}
	touchedPaths := []string(nil)
	impacted := map[string][]string{}
	if req.module != "" {
		surfaces = filterGoSurfacesByModule(root, surfaces, req.module)
		if len(surfaces) == 0 {
			return freshnessReport{}, fmt.Errorf("no Go module matches selector %q", req.module)
		}
	} else if req.touched {
		touchedPaths, err = changedPaths(root)
		if err != nil {
			return freshnessReport{}, err
		}
		impacted = impactedSurfaces(root, surfaces, modules, touchedPaths)
		touchedPaths = reportTouchedPaths(impacted)
		surfaces = filterImpactedSurfaces(surfaces, impacted)
	}
	for _, surface := range surfaces {
		if modules[surface.module].ambiguous {
			return freshnessReport{}, fmt.Errorf("Go module %q has multiple source directories; cannot establish a unique owner", surface.module)
		}
	}
	cacheEnabled := !req.noCache && !req.apply && !req.build
	if cacheEnabled {
		attachFreshnessCacheFingerprints(surfaces, modules)
		for idx := range surfaces {
			surfaces[idx].cacheKey, surfaces[idx].cacheKeyReady = freshnessCacheKey(surfaces[idx])
		}
	}

	report := freshnessReport{
		Clean:      true,
		Root:       root,
		Mode:       "all",
		Touched:    touchedPaths,
		Surfaces:   make([]freshnessSurfaceReport, 0, len(surfaces)),
		ElapsedMs:  time.Since(start).Milliseconds(),
		Exclusions: discoverGoExclusions(root, modules),
	}
	if req.touched {
		report.Mode = "touched"
	} else if req.module != "" {
		report.Mode = "module"
	}
	report.Surfaces = checkGoSurfaces(ctx, root, surfaces, impacted, req)
	if cacheEnabled {
		// Recompute only identities that may be stored, sharing module scans
		// across the pending results.
		var pending []goSurface
		var pendingIndexes []int
		for idx, surface := range surfaces {
			item := report.Surfaces[idx]
			if !surface.cacheKeyReady || item.CacheHit || item.Status == "needs_download" {
				continue
			}
			pending = append(pending, surface)
			pendingIndexes = append(pendingIndexes, idx)
		}
		attachFreshnessCacheFingerprints(pending, modules)
		for pendingIdx, surface := range pending {
			currentKey, currentReady := freshnessCacheKey(surface)
			if !currentReady || currentKey != surface.cacheKey {
				continue
			}
			storeFreshnessCache(root, surface.cacheKey, report.Surfaces[pendingIndexes[pendingIdx]])
		}
	}
	for _, item := range report.Surfaces {
		switch item.Status {
		case "clean":
			report.Summary.Clean++
		case "stale":
			report.Summary.Stale++
			report.Clean = false
		case "error":
			report.Summary.Errors++
			report.Clean = false
		case "needs_download":
			report.Summary.NeedsDownload++
			report.Clean = false
		default:
			report.Summary.Skipped++
			report.Clean = false
		}
	}
	report.Summary.Checked = len(report.Surfaces)
	if report.Summary.Stale > 0 {
		command := fmt.Sprintf("%s freshness --%s --apply", support.AppName, report.Mode)
		if report.Mode == "module" {
			command = fmt.Sprintf("%s freshness --module %s --apply", support.AppName, shellQuoteCommandArg(req.module))
		}
		report.NextActions = append(report.NextActions, freshnessAction{
			Code:       "apply_go_tidy",
			Message:    "Run SDA-owned package freshness repair for impacted Go surfaces.",
			Command:    command,
			Fixability: "automatic",
		})
	}
	if reportHasMissingLocalReplaceError(report) {
		report.NextActions = append(report.NextActions, freshnessAction{
			Code:       "preview_missing_local_replaces",
			Message:    "Preview SDA-owned local replace reconciliation for errored Go surfaces.",
			Command:    fmt.Sprintf("%s deps reconcile --all --json", support.AppName),
			Fixability: "guided",
		})
	}
	report.ElapsedMs = time.Since(start).Milliseconds()
	return report, nil
}

func shellQuoteCommandArg(value string) string {
	if value != "" && strings.IndexFunc(value, func(r rune) bool {
		return !((r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || strings.ContainsRune("_./-", r))
	}) < 0 {
		return value
	}
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func filterGoSurfacesByModule(root string, surfaces []goSurface, selector string) []goSurface {
	selector = strings.TrimSpace(selector)
	if selector == "" {
		return nil
	}
	selector = filepath.ToSlash(filepath.Clean(selector))
	out := make([]goSurface, 0, 1)
	for _, surface := range surfaces {
		relGoMod := filepath.ToSlash(mustRel(root, surface.goMod))
		relDir := filepath.ToSlash(mustRel(root, filepath.Dir(surface.goMod)))
		if selector == surface.module || selector == relGoMod || selector == relDir {
			out = append(out, surface)
		}
	}
	return out
}

func discoverGoExclusions(root string, modules map[string]goModule) []freshnessExclusionReport {
	var out []freshnessExclusionReport
	seen := map[string]struct{}{}
	for _, module := range modules {
		path := filepath.Join(module.dir, "go.mod")
		rel, err := filepath.Rel(root, path)
		if err != nil {
			continue
		}
		rel = filepath.ToSlash(rel)
		reason := ""
		for _, exclusion := range freshnessExclusions {
			if rel == exclusion.Path {
				reason = exclusion.Reason
				break
			}
		}
		if reason == "" && strings.HasPrefix(rel, "templates/") {
			reason = "template module is a generation source, not a fleet surface"
		}
		if reason != "" {
			out = append(out, freshnessExclusionReport{GoModPath: rel, Reason: reason})
			seen[rel] = struct{}{}
		}
	}
	for _, exclusion := range freshnessExclusions {
		if _, ok := seen[exclusion.Path]; ok {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, exclusion.Path)); err == nil {
			out = append(out, freshnessExclusionReport{GoModPath: exclusion.Path, Reason: exclusion.Reason})
		}
	}
	templatesRoot := filepath.Join(root, "templates")
	_ = filepath.WalkDir(templatesRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "go.mod" {
			rel, _ := filepath.Rel(root, path)
			rel = filepath.ToSlash(rel)
			if _, ok := seen[rel]; !ok {
				out = append(out, freshnessExclusionReport{GoModPath: rel, Reason: "template module is a generation source, not a fleet surface"})
				seen[rel] = struct{}{}
			}
		}
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].GoModPath < out[j].GoModPath })
	return out
}

func checkGoSurfaces(ctx context.Context, root string, surfaces []goSurface, impacted map[string][]string, req freshnessRequest) []freshnessSurfaceReport {
	if len(surfaces) == 0 {
		return nil
	}
	concurrency := req.concurrency
	if concurrency <= 0 {
		concurrency = 8
	}
	if concurrency > len(surfaces) {
		concurrency = len(surfaces)
	}
	out := make([]freshnessSurfaceReport, len(surfaces))
	jobs := make(chan int)
	var wg sync.WaitGroup
	wg.Add(concurrency)
	for worker := 0; worker < concurrency; worker++ {
		go func() {
			defer wg.Done()
			for idx := range jobs {
				surface := surfaces[idx]
				out[idx] = checkGoFreshness(ctx, root, surface, impacted[surface.goMod], req)
			}
		}()
	}
	for idx := range surfaces {
		jobs <- idx
	}
	close(jobs)
	wg.Wait()
	return out
}

func discoverGoSurfacesFromModules(root string, modules map[string]goModule) ([]goSurface, error) {
	var out []goSurface
	for _, module := range modules {
		goModPath := filepath.Join(module.dir, "go.mod")
		if excludedGoMod(goModPath, root) {
			continue
		}
		rel, _ := filepath.Rel(root, module.dir)
		parts := strings.Split(filepath.ToSlash(rel), "/")
		scenario, surface := "root", filepath.ToSlash(rel)
		if len(parts) >= 2 && parts[0] == "scenarios" {
			scenario = parts[1]
			surface = strings.Join(parts[2:], "/")
			if surface == "" {
				surface = scenario
			}
		}
		out = append(out, goSurface{scenario: scenario, surface: surface, goMod: goModPath, module: module.module, requires: module.requires, replaces: module.replaces})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].goMod < out[j].goMod })
	return out, nil
}

func discoverGoModules(root string) (map[string]goModule, error) {
	out := map[string]goModule{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "vendor", "data", "dist", "build", ".cache", "phase-cache":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		view, err := parseGoModFile(path)
		if err != nil || view.module == "" {
			if excludedGoMod(path, root) {
				out["__excluded:"+path] = goModule{dir: filepath.Dir(path)}
			}
			return nil
		}
		// Retained fixture/evidence copies must not shadow the repository's
		// own module, which owns resource CLIs and the control plane.
		if existing, ok := out[view.module]; ok && existing.dir == root {
			return nil
		}
		existing, duplicate := out[view.module]
		out[view.module] = goModule{
			ambiguous: filepath.Dir(path) != root && duplicate && existing.dir != filepath.Dir(path),
			dir:       filepath.Dir(path),
			module:    view.module,
			requires:  view.requires,
			replaces:  view.replaces,
		}
		return nil
	})
	return out, err
}

type goModView struct {
	module   string
	requires map[string]struct{}
	replaces map[string]string
}

func parseGoModFile(path string) (goModView, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return goModView{}, err
	}
	file, err := modfile.Parse(path, data, nil)
	if err != nil {
		return goModView{}, fmt.Errorf("parse %s: %w", path, err)
	}
	view := goModView{requires: map[string]struct{}{}, replaces: map[string]string{}}
	if file.Module != nil {
		view.module = file.Module.Mod.Path
	}
	if view.module == "" {
		return goModView{}, fmt.Errorf("%s: module path is empty", path)
	}
	for _, req := range file.Require {
		view.requires[req.Mod.Path] = struct{}{}
	}
	for _, replacement := range file.Replace {
		view.replaces[replacement.Old.Path] = replacement.New.Path
	}
	return view, nil
}

func excludedGoMod(path, root string) bool {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return false
	}
	rel = filepath.ToSlash(rel)
	for _, exclusion := range freshnessExclusions {
		if rel == exclusion.Path || strings.HasPrefix(rel, "templates/") {
			return true
		}
	}
	return false
}

func impactedSurfaces(root string, surfaces []goSurface, modules map[string]goModule, touched []string) map[string][]string {
	moduleByDir := map[string]string{}
	for _, module := range modules {
		if module.module != "" {
			moduleByDir[filepath.ToSlash(mustRel(root, module.dir))] = module.module
		}
	}
	changedModules := map[string][]string{}
	rootModuleChanged := false
	for _, path := range touched {
		if path == "go.mod" || path == "go.sum" {
			rootModuleChanged = true
			continue
		}
		for dir, module := range moduleByDir {
			if path == dir || strings.HasPrefix(path, dir+"/") {
				changedModules[module] = append(changedModules[module], path)
			}
		}
	}
	out := map[string][]string{}
	for _, surface := range surfaces {
		if rootModuleChanged {
			out[surface.goMod] = append(out[surface.goMod], "go.mod", "go.sum")
			continue
		}
		surfaceDir := filepath.ToSlash(mustRel(root, filepath.Dir(surface.goMod)))
		for _, path := range touched {
			if path == surfaceDir+"/go.mod" || path == surfaceDir+"/go.sum" {
				out[surface.goMod] = append(out[surface.goMod], path)
			}
		}
		required := transitiveRequirements(surface.module, modules)
		for module, paths := range changedModules {
			if _, ok := required[module]; ok && module != surface.module {
				out[surface.goMod] = append(out[surface.goMod], paths...)
			}
		}
	}
	for key, paths := range out {
		sort.Strings(paths)
		out[key] = compactStrings(paths)
	}
	return out
}

func transitiveRequirements(module string, modules map[string]goModule) map[string]struct{} {
	out := map[string]struct{}{}
	var walk func(string)
	walk = func(path string) {
		mod, ok := modules[path]
		if !ok {
			return
		}
		for required := range mod.requires {
			if _, seen := out[required]; seen {
				continue
			}
			out[required] = struct{}{}
			walk(required)
		}
	}
	walk(module)
	return out
}

func filterImpactedSurfaces(surfaces []goSurface, impacted map[string][]string) []goSurface {
	if len(impacted) == 0 {
		return nil
	}
	out := surfaces[:0:0]
	for _, surface := range surfaces {
		if len(impacted[surface.goMod]) > 0 {
			out = append(out, surface)
		}
	}
	return out
}

func checkGoFreshness(ctx context.Context, root string, surface goSurface, impactedBy []string, req freshnessRequest) freshnessSurfaceReport {
	item := freshnessSurfaceReport{
		Scenario: surface.scenario, Surface: surface.surface,
		GoModPath: filepath.ToSlash(mustRel(root, surface.goMod)), Status: "clean",
		ImpactedBy: limitStrings(compactStrings(impactedBy), maxImpactedByPaths), ImpactedByCount: len(compactStrings(impactedBy)),
	}
	if !req.noCache && !req.apply && !req.build {
		if surface.cacheKeyReady {
			if cached, ok := loadFreshnessCache(root, surface.cacheKey); ok {
				cached.Scenario, cached.Surface, cached.GoModPath = item.Scenario, item.Surface, item.GoModPath
				cached.ImpactedBy, cached.ImpactedByCount = item.ImpactedBy, item.ImpactedByCount
				cached.CacheHit = true
				return cached
			}
		}
	}
	return checkGoFreshnessUncached(ctx, root, surface, impactedBy, req)
}

func checkGoFreshnessUncached(ctx context.Context, root string, surface goSurface, impactedBy []string, req freshnessRequest) freshnessSurfaceReport {
	impactedBy = compactStrings(impactedBy)
	item := freshnessSurfaceReport{
		Scenario:        surface.scenario,
		Surface:         surface.surface,
		GoModPath:       filepath.ToSlash(mustRel(root, surface.goMod)),
		Status:          "clean",
		ImpactedBy:      limitStrings(impactedBy, maxImpactedByPaths),
		ImpactedByCount: len(impactedBy),
	}
	surfaceRoot := filepath.Dir(surface.goMod)
	diff, clean, err := runGoCommand(ctx, surfaceRoot, "mod", "tidy", "-diff")
	if err != nil {
		if needsModuleDownload(err) && !knownInRepoDownload(root, err) {
			// Offline evaluation is authoritative for defects. A single ambient
			// proxy retry distinguishes a cold host cache from a bad surface.
			retryDiff, retryClean, retryErr := runGoNetwork(ctx, surfaceRoot, "mod", "tidy", "-diff")
			item.Status = "needs_download"
			if retryErr != nil {
				item.Error = retryErr.Error()
			} else if !retryClean {
				item.DiffPaths = tidyDiffPaths(root, surface.goMod, retryDiff)
			}
			return item
		}
		item.Status = "error"
		item.Error = err.Error()
		return item
	}
	if !clean {
		item.Status = "stale"
		item.DiffPaths = tidyDiffPaths(root, surface.goMod, diff)
		if req.apply {
			if _, _, err := runGoCommand(ctx, surfaceRoot, "mod", "tidy"); err != nil {
				item.Status = "error"
				item.Error = "go mod tidy --apply failed: " + err.Error()
				return item
			}
			diff, clean, err = runGoCommand(ctx, surfaceRoot, "mod", "tidy", "-diff")
			if err != nil {
				item.Status = "error"
				item.Error = "go mod tidy post-apply verification failed: " + err.Error()
				return item
			}
			if !clean {
				// `go mod tidy` succeeded and the surface is still stale, so
				// another tidy cannot fix it either (see forceGoSumRewrite).
				// Force one full rewrite, then verify again.
				if err := forceGoSumRewrite(ctx, surfaceRoot); err != nil {
					item.Status = "error"
					item.Error = "go.sum forced rewrite failed: " + err.Error()
					return item
				}
				diff, clean, err = runGoCommand(ctx, surfaceRoot, "mod", "tidy", "-diff")
				if err != nil {
					item.Status = "error"
					item.Error = "go mod tidy post-rewrite verification failed: " + err.Error()
					return item
				}
			}
			if clean {
				item.Status = "clean"
				item.DiffPaths = nil
			} else {
				item.DiffPaths = tidyDiffPaths(root, surface.goMod, diff)
			}
		}
		return item
	}
	if req.build {
		_, _, err := runGoCommand(ctx, surfaceRoot, "build", "./...")
		if err != nil {
			item.Status = "error"
			item.Error = "go build ./... failed: " + err.Error()
		}
	}
	return item
}

const (
	freshnessCacheSchema = 3
	freshnessCacheLimit  = 2000
)

var freshnessCacheMu sync.Mutex

func freshnessCacheDir(root string) string {
	return filepath.Join(root, "scenarios", "scenario-dependency-analyzer", "data", "freshness-cache")
}

func freshnessCacheKey(surface goSurface) (string, bool) {
	if !surface.moduleFingerprintReady || !surface.dependencyFingerprintReady {
		return "", false
	}
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("schema=%d\nmodule=%s\n", freshnessCacheSchema, surface.module)))
	h.Write([]byte("surface\n" + surface.moduleFingerprint + "\n"))
	h.Write([]byte("local-dependencies\n" + surface.dependencyFingerprint + "\n"))
	return fmt.Sprintf("%x", h.Sum(nil)), true
}

// attachFreshnessCacheFingerprints binds each selected surface's cache
// identity to its own module and the discovered in-repository modules that its
// Go requirement closure can load. Module content is fingerprinted once per
// pass even when several selected surfaces share modules.
func attachFreshnessCacheFingerprints(surfaces []goSurface, modules map[string]goModule) {
	requirements := make([][]string, len(surfaces))
	needed := map[string]goModule{}
	for idx, surface := range surfaces {
		if module, ok := modules[surface.module]; ok {
			needed[surface.module] = module
		}
		for modulePath := range transitiveRequirements(surface.module, modules) {
			module, ok := modules[modulePath]
			if !ok {
				continue
			}
			requirements[idx] = append(requirements[idx], modulePath)
			needed[modulePath] = module
		}
		sort.Strings(requirements[idx])
	}

	fingerprints := make(map[string]string, len(needed))
	valid := make(map[string]bool, len(needed))
	for modulePath, module := range needed {
		fingerprints[modulePath], valid[modulePath] = freshnessModuleFingerprint(module.dir)
	}
	for idx := range surfaces {
		h := sha256.New()
		ready := true
		ownFingerprint, ownReady := fingerprints[surfaces[idx].module]
		surfaces[idx].moduleFingerprint = ownFingerprint
		surfaces[idx].moduleFingerprintReady = ownReady && ownFingerprint != ""
		for _, modulePath := range requirements[idx] {
			if !valid[modulePath] {
				ready = false
				break
			}
			h.Write([]byte(modulePath + "\n" + fingerprints[modulePath] + "\n"))
		}
		surfaces[idx].dependencyFingerprint = fmt.Sprintf("%x", h.Sum(nil))
		surfaces[idx].dependencyFingerprintReady = ready && surfaces[idx].moduleFingerprintReady
	}
}

func freshnessModuleFingerprint(moduleDir string) (string, bool) {
	h := sha256.New()
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(moduleDir, name))
		if err != nil {
			if name == "go.sum" && os.IsNotExist(err) {
				continue
			}
			return "", false
		}
		h.Write([]byte(name + "\n"))
		h.Write(data)
	}
	var files []string
	if err := filepath.WalkDir(moduleDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != moduleDir {
				if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
					return filepath.SkipDir
				} else if !os.IsNotExist(err) {
					return err
				}
			}
			switch d.Name() {
			case ".git", "vendor", "node_modules", "dist", "build", ".cache":
				return filepath.SkipDir
			}
			return nil
		}
		if strings.HasSuffix(d.Name(), ".go") {
			files = append(files, path)
		}
		return nil
	}); err != nil {
		return "", false
	}
	sort.Strings(files)
	for _, path := range files {
		data, err := os.ReadFile(path)
		if err != nil {
			return "", false
		}
		h.Write([]byte(filepath.ToSlash(path) + "\n"))
		h.Write(data)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), true
}

func loadFreshnessCache(root, key string) (freshnessSurfaceReport, bool) {
	if key == "" {
		return freshnessSurfaceReport{}, false
	}
	data, err := os.ReadFile(filepath.Join(freshnessCacheDir(root), key+".json"))
	if err != nil {
		return freshnessSurfaceReport{}, false
	}
	var item freshnessSurfaceReport
	if json.Unmarshal(data, &item) != nil || item.Status == "" || item.Status == "needs_download" {
		return freshnessSurfaceReport{}, false
	}
	return item, true
}

func storeFreshnessCache(root, key string, item freshnessSurfaceReport) {
	freshnessCacheMu.Lock()
	defer freshnessCacheMu.Unlock()
	if key == "" {
		return
	}
	dir := freshnessCacheDir(root)
	if os.MkdirAll(dir, 0o755) != nil {
		return
	}
	data, err := json.Marshal(item)
	if err != nil {
		return
	}
	tmp, err := os.CreateTemp(dir, ".freshness-*")
	if err != nil {
		return
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err = tmp.Write(data); err == nil {
		err = tmp.Close()
	} else {
		_ = tmp.Close()
	}
	if err == nil {
		_ = os.Rename(tmpName, filepath.Join(dir, key+".json"))
	}
	entries, _ := os.ReadDir(dir)
	if len(entries) > freshnessCacheLimit {
		for _, entry := range entries[:len(entries)-freshnessCacheLimit] {
			_ = os.Remove(filepath.Join(dir, entry.Name()))
		}
	}
}

// forceGoSumRewrite repairs a go.sum that `go mod tidy` cannot converge.
//
// `go mod tidy` rewrites go.sum only when the set of hashes changes. A go.sum
// holding exactly the right hashes in the wrong order therefore survives every
// tidy untouched — tidy exits 0, writes nothing, and `tidy -diff` keeps
// reporting a diff against its canonical sorted form. Without this, `--apply`
// reports that surface stale forever while advertising an automatic fix.
//
// Removing go.sum forces tidy to regenerate it from go.mod, which writes it
// sorted. The regenerated file must carry the same hash set as the original: a
// set that changed would mean a real dependency change, which freshness repair
// must never make on its own. If it differs — or anything else fails — the
// original file is restored and the caller reports an error.
func forceGoSumRewrite(ctx context.Context, surfaceRoot string) error {
	goSum := filepath.Join(surfaceRoot, "go.sum")
	info, err := os.Stat(goSum)
	if err != nil {
		return fmt.Errorf("stat go.sum: %w", err)
	}
	original, err := os.ReadFile(goSum)
	if err != nil {
		return fmt.Errorf("read go.sum: %w", err)
	}
	restore := func() {
		_ = os.WriteFile(goSum, original, info.Mode().Perm())
	}
	if err := os.Remove(goSum); err != nil {
		return fmt.Errorf("remove go.sum: %w", err)
	}
	if _, _, err := runGoCommand(ctx, surfaceRoot, "mod", "tidy"); err != nil {
		restore()
		return fmt.Errorf("go mod tidy after go.sum removal: %w", err)
	}
	rewritten, err := os.ReadFile(goSum)
	if err != nil {
		restore()
		return fmt.Errorf("read regenerated go.sum: %w", err)
	}
	if !sameGoSumEntries(original, rewritten) {
		restore()
		return fmt.Errorf("regenerated go.sum changed the hash set; restored the original")
	}
	return nil
}

// sameGoSumEntries reports whether two go.sum files carry the same hash lines,
// ignoring order and blank lines. Order is exactly what this repair changes.
func sameGoSumEntries(a, b []byte) bool {
	left, right := sortedGoSumLines(a), sortedGoSumLines(b)
	if len(left) != len(right) {
		return false
	}
	for i := range left {
		if left[i] != right[i] {
			return false
		}
	}
	return true
}

func sortedGoSumLines(data []byte) []string {
	fields := strings.Split(string(data), "\n")
	out := make([]string, 0, len(fields))
	for _, line := range fields {
		if line = strings.TrimSpace(line); line != "" {
			out = append(out, line)
		}
	}
	sort.Strings(out)
	return out
}

const maxImpactedByPaths = 20

func reportTouchedPaths(impacted map[string][]string) []string {
	var touched []string
	for _, paths := range impacted {
		touched = append(touched, paths...)
	}
	return compactStrings(touched)
}

func limitStrings(values []string, limit int) []string {
	if len(values) <= limit {
		return values
	}
	out := append([]string(nil), values[:limit]...)
	out = append(out, fmt.Sprintf("... %d more", len(values)-limit))
	return out
}

var (
	runGoCommand = runGo
	runGoNetwork = runGoNetworkCommand
)

func runGo(ctx context.Context, dir string, args ...string) (string, bool, error) {
	return runGoEnv(ctx, dir, true, args...)
}

// freshnessGoEnv is the environment of every Go probe before the toolchain
// floor: the workspace is switched off and offline probes pin GOPROXY=off.
// The inherited GOFLAGS is kept, never cleared; the spawn site composes the
// build width through envkit.Toolchain.
func freshnessGoEnv(parent envkit.Env, offline bool) envkit.Env {
	overlay := envkit.Env{"GOWORK=off"}
	if offline {
		overlay = append(overlay, "GOPROXY=off")
	}
	return envkit.WithOverlay(parent, envkit.SameScenario, overlay)
}

func runGoNetworkCommand(ctx context.Context, dir string, args ...string) (string, bool, error) {
	return runGoEnv(ctx, dir, false, args...)
}

func runGoEnv(ctx context.Context, dir string, offline bool, args ...string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = envkit.Toolchain(freshnessGoEnv(envkit.Env(os.Environ()), offline), envkit.ToolchainOptions{})
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return stdout.String(), true, nil
	}
	if len(args) >= 3 && args[0] == "mod" && args[1] == "tidy" && args[2] == "-diff" {
		if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && stdout.Len() > 0 {
			return stdout.String(), false, nil
		}
	}
	return stdout.String(), false, fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
}

func needsModuleDownload(err error) bool {
	return err != nil && strings.Contains(err.Error(), "module lookup disabled by GOPROXY=off")
}

func knownInRepoDownload(root string, err error) bool {
	message := err.Error()
	// In-repo modules are a surface defect when their replace is absent. They
	// must never enter the ambient retry tier, or the same local defect becomes
	// a network request for every consumer.
	_, statErr := os.Stat(filepath.Join(root, "packages", "envkit-go", "go.mod"))
	return strings.Contains(message, "github.com/vrooli/envkit-go") && statErr == nil
}

func changedPaths(root string) ([]string, error) {
	cmd := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all")
	cmd.Dir = root
	raw, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	for _, line := range strings.Split(string(raw), "\n") {
		if len(line) < 4 {
			continue
		}
		path := strings.TrimSpace(line[3:])
		if arrow := strings.Index(path, " -> "); arrow >= 0 {
			path = path[arrow+4:]
		}
		if path != "" {
			seen[filepath.ToSlash(path)] = struct{}{}
		}
	}
	paths := make([]string, 0, len(seen))
	for path := range seen {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths, nil
}

func tidyDiffPaths(root, goModPath, diff string) []string {
	candidates := []string{filepath.ToSlash(mustRel(root, goModPath)), filepath.ToSlash(mustRel(root, filepath.Join(filepath.Dir(goModPath), "go.sum")))}
	var out []string
	for _, candidate := range candidates {
		if strings.Contains(diff, filepath.Base(candidate)) {
			out = append(out, candidate)
		}
	}
	if len(out) == 0 {
		out = []string{candidates[0]}
	}
	return out
}

func printFreshnessReport(report freshnessReport) error {
	status := "clean"
	if !report.Clean {
		status = "needs attention"
	}
	lines := []string{
		fmt.Sprintf("Status: %s", status),
		fmt.Sprintf("Mode: %s", report.Mode),
		fmt.Sprintf("Surfaces checked: %d", report.Summary.Checked),
		fmt.Sprintf("Stale: %d", report.Summary.Stale),
		fmt.Sprintf("Errors: %d", report.Summary.Errors),
		fmt.Sprintf("Needs download: %d", report.Summary.NeedsDownload),
	}
	var results []string
	for _, surface := range report.Surfaces {
		if surface.Status == "clean" {
			continue
		}
		results = append(results, fmt.Sprintf("%s/%s: %s %s", surface.Scenario, surface.Surface, surface.Status, strings.Join(surface.DiffPaths, ", ")))
	}
	hints := freshnessRetrievalHints(report)
	return support.PrintList(false, cliapp.ListReport{
		Summary:        lines,
		ResultsHeading: "Impacted Surfaces",
		Results:        results,
		RetrievalHints: hints,
	}, nil)
}

func freshnessRetrievalHints(report freshnessReport) []string {
	var hints []string
	for _, action := range report.NextActions {
		if strings.TrimSpace(action.Command) != "" {
			hints = append(hints, action.Command)
		}
	}
	hints = append(hints, fmt.Sprintf("%s freshness --%s --json", support.AppName, report.Mode))
	return hints
}

func reportHasMissingLocalReplaceError(report freshnessReport) bool {
	for _, surface := range report.Surfaces {
		if surface.Status == "error" && isMissingLocalReplaceError(surface.Error) {
			return true
		}
	}
	return false
}

func isMissingLocalReplaceError(message string) bool {
	message = strings.ToLower(message)
	return strings.Contains(message, "github.com/vrooli/") &&
		strings.Contains(message, "go.mod at revision v0.0.0") &&
		strings.Contains(message, "repository not found")
}

func resolveRepoRoot(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return filepath.Abs(explicit)
	}
	cmd := exec.Command("git", "rev-parse", "--show-toplevel")
	out, err := cmd.Output()
	if err != nil {
		wd, wdErr := os.Getwd()
		if wdErr != nil {
			return "", err
		}
		return wd, nil
	}
	return filepath.Clean(strings.TrimSpace(string(out))), nil
}

func compactStrings(values []string) []string {
	if len(values) == 0 {
		return nil
	}
	seen := map[string]struct{}{}
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	sort.Strings(out)
	return out
}

func mustRel(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return rel
}

func _ensureFreshnessReportJSONShape(report freshnessReport) ([]byte, error) {
	return json.Marshal(report)
}
