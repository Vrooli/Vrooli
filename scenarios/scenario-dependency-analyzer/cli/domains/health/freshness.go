package health

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/json"
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
	scenario string
	surface  string
	goMod    string
	module   string
	requires map[string]struct{}
	replaces map[string]string
}

type goModule struct {
	dir      string
	module   string
	requires map[string]struct{}
	replaces map[string]string
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
	var timeout time.Duration
	var noCache bool
	fs.StringVar(&repoRoot, "repo-root", "", "Repository root (defaults to current workspace)")
	fs.IntVar(&concurrency, "concurrency", 8, "Maximum package surfaces to check concurrently")
	fs.BoolVar(&touched, "touched", false, "Only check surfaces impacted by changed in-repo modules")
	fs.BoolVar(&all, "all", false, "Check every discovered Go scenario surface")
	fs.BoolVar(&apply, "apply", false, "Run go mod tidy on stale surfaces")
	fs.BoolVar(&build, "build", false, "Run go build ./... after tidy checks")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
	fs.DurationVar(&timeout, "timeout", 5*time.Minute, "Overall freshness deadline")
	fs.BoolVar(&noCache, "no-cache", false, "Do not read or write the freshness result cache")
	if err := support.ParseFlags(fs, args); err != nil {
		return err
	}
	if len(fs.Args()) != 0 || (touched && all) {
		return fmt.Errorf("usage: %s freshness [--touched|--all] [--apply] [--build] [--concurrency <n>] [--repo-root <path>] [--json]", support.AppName)
	}
	if !touched && !all {
		touched = true
	}
	root, err := resolveRepoRoot(repoRoot)
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	report, err := checkFreshness(ctx, root, freshnessRequest{touched: touched, apply: apply, build: build, noCache: noCache, concurrency: concurrency})
	if err != nil {
		return err
	}
	if jsonOutput {
		return support.PrintReportJSON(report)
	}
	return printFreshnessReport(report)
}

type freshnessRequest struct {
	touched       bool
	apply         bool
	build         bool
	noCache       bool
	fastRootCache bool
	treeDigest    string
	concurrency   int
}

func checkFreshness(ctx context.Context, root string, req freshnessRequest) (freshnessReport, error) {
	start := time.Now()
	modules, err := discoverGoModules(root)
	if err != nil {
		return freshnessReport{}, err
	}
	surfaces, err := discoverGoSurfacesFromModules(root, modules)
	if err != nil {
		return freshnessReport{}, err
	}
	touchedPaths := []string(nil)
	impacted := map[string][]string{}
	if req.touched {
		touchedPaths, err = changedPaths(root)
		if err != nil {
			return freshnessReport{}, err
		}
		req.fastRootCache = containsRootModuleMetadata(touchedPaths)
		impacted = impactedSurfaces(root, surfaces, modules, touchedPaths)
		touchedPaths = reportTouchedPaths(impacted)
		surfaces = filterImpactedSurfaces(surfaces, impacted)
		if !req.noCache && !req.apply && !req.build {
			req.treeDigest = digestStrings(touchedPaths)
		}
	}
	if !req.noCache && !req.apply && !req.build && req.treeDigest == "" {
		req.treeDigest = freshnessTreeDigest(root)
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
	}
	report.Surfaces = checkGoSurfaces(ctx, root, surfaces, impacted, req)
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
		default:
			report.Summary.Skipped++
		}
	}
	report.Summary.Checked = len(report.Surfaces)
	if report.Summary.Stale > 0 {
		report.NextActions = append(report.NextActions, freshnessAction{
			Code:       "apply_go_tidy",
			Message:    "Run SDA-owned package freshness repair for impacted Go surfaces.",
			Command:    fmt.Sprintf("%s freshness --%s --apply", support.AppName, report.Mode),
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

func discoverGoSurfaces(root string) ([]goSurface, error) {
	modules, err := discoverGoModules(root)
	if err != nil {
		return nil, err
	}
	return discoverGoSurfacesFromModules(root, modules)
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
		out[view.module] = goModule{
			dir:      filepath.Dir(path),
			module:   view.module,
			requires: view.requires,
			replaces: view.replaces,
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
		if req.fastRootCache {
			if cached, ok := loadFreshnessCacheIndex(root, surface, req.treeDigest); ok {
				cached.Scenario, cached.Surface, cached.GoModPath = item.Scenario, item.Surface, item.GoModPath
				cached.ImpactedBy, cached.ImpactedByCount, cached.CacheHit = item.ImpactedBy, item.ImpactedByCount, true
				return cached
			}
		}
		if cached, ok := loadFreshnessCache(root, surface); ok {
			cached.Scenario, cached.Surface, cached.GoModPath = item.Scenario, item.Surface, item.GoModPath
			cached.ImpactedBy, cached.ImpactedByCount = item.ImpactedBy, item.ImpactedByCount
			cached.CacheHit = true
			storeFreshnessCache(root, surface, cached, req.treeDigest)
			return cached
		}
	}
	result := checkGoFreshnessUncached(ctx, root, surface, impactedBy, req)
	if !req.noCache && !req.apply && !req.build && result.Status != "needs_download" {
		storeFreshnessCache(root, surface, result, req.treeDigest)
	}
	return result
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
	freshnessCacheSchema = 2
	freshnessCacheLimit  = 2000
)

var freshnessCacheMu sync.Mutex

func containsRootModuleMetadata(paths []string) bool {
	for _, path := range paths {
		if path == "go.mod" || path == "go.sum" {
			return true
		}
	}
	return false
}

func freshnessTreeDigest(root string) string {
	h := sha256.New()
	status := exec.Command("git", "status", "--porcelain=v1", "--untracked-files=all")
	status.Dir = root
	if data, err := status.Output(); err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasSuffix(line, ".go") || strings.HasSuffix(line, ".mod") || strings.HasSuffix(line, ".sum") {
				h.Write([]byte(line + "\n"))
			}
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

func freshnessCacheDir(root string) string {
	return filepath.Join(root, "scenarios", "scenario-dependency-analyzer", "data", "freshness-cache")
}

func freshnessCacheKey(surface goSurface) (string, bool) {
	h := sha256.New()
	h.Write([]byte(fmt.Sprintf("schema=%d\nmodule=%s\n", freshnessCacheSchema, surface.module)))
	for _, name := range []string{"go.mod", "go.sum"} {
		data, err := os.ReadFile(filepath.Join(filepath.Dir(surface.goMod), name))
		if err == nil {
			h.Write([]byte(name + "\n"))
			h.Write(data)
		}
	}
	var files []string
	_ = filepath.WalkDir(filepath.Dir(surface.goMod), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if d.IsDir() {
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
	})
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

func loadFreshnessCache(root string, surface goSurface) (freshnessSurfaceReport, bool) {
	key, ok := freshnessCacheKey(surface)
	if !ok {
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

type freshnessCacheIndex struct {
	TreeDigest string            `json:"tree_digest"`
	Keys       map[string]string `json:"keys"`
}

func loadFreshnessCacheIndex(root string, surface goSurface, treeDigest string) (freshnessSurfaceReport, bool) {
	data, err := os.ReadFile(filepath.Join(freshnessCacheDir(root), "index.json"))
	if err != nil {
		return freshnessSurfaceReport{}, false
	}
	var index freshnessCacheIndex
	if json.Unmarshal(data, &index) != nil {
		return freshnessSurfaceReport{}, false
	}
	if index.TreeDigest != treeDigest {
		return freshnessSurfaceReport{}, false
	}
	key, ok := index.Keys[surface.goMod]
	if !ok {
		return freshnessSurfaceReport{}, false
	}
	data, err = os.ReadFile(filepath.Join(freshnessCacheDir(root), key+".json"))
	if err != nil {
		return freshnessSurfaceReport{}, false
	}
	var item freshnessSurfaceReport
	if json.Unmarshal(data, &item) != nil || item.Status == "" || item.Status == "needs_download" {
		return freshnessSurfaceReport{}, false
	}
	return item, true
}

func storeFreshnessCache(root string, surface goSurface, item freshnessSurfaceReport, treeDigest string) {
	freshnessCacheMu.Lock()
	defer freshnessCacheMu.Unlock()
	key, ok := freshnessCacheKey(surface)
	if !ok {
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
		indexPath := filepath.Join(dir, "index.json")
		index := freshnessCacheIndex{TreeDigest: treeDigest, Keys: map[string]string{}}
		if existing, readErr := os.ReadFile(indexPath); readErr == nil {
			_ = json.Unmarshal(existing, &index)
		}
		if index.Keys == nil {
			index.Keys = map[string]string{}
		}
		index.TreeDigest = treeDigest
		index.Keys[surface.goMod] = key
		if encoded, marshalErr := json.Marshal(index); marshalErr == nil {
			_ = os.WriteFile(indexPath, encoded, 0o644)
		}
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

func runGoNetworkCommand(ctx context.Context, dir string, args ...string) (string, bool, error) {
	return runGoEnv(ctx, dir, false, args...)
}

func runGoEnv(ctx context.Context, dir string, offline bool, args ...string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	overlay := envkit.Env{"GOWORK=off", "GOFLAGS="}
	if offline {
		overlay = append(overlay, "GOPROXY=off")
	}
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, overlay)
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

func digestStrings(values []string) string {
	h := sha256.New()
	for _, value := range values {
		h.Write([]byte(value))
		h.Write([]byte{'\n'})
	}
	return fmt.Sprintf("%x", h.Sum(nil))
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
