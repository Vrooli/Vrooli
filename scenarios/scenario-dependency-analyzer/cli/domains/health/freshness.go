package health

import (
	"bytes"
	"context"
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
	"scenario-dependency-analyzer/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

type freshnessReport struct {
	Clean       bool                     `json:"clean"`
	Root        string                   `json:"root"`
	Mode        string                   `json:"mode"`
	Touched     []string                 `json:"touched,omitempty"`
	Surfaces    []freshnessSurfaceReport `json:"surfaces"`
	Summary     freshnessSummary         `json:"summary"`
	NextActions []freshnessAction        `json:"next_actions,omitempty"`
	ElapsedMs   int64                    `json:"elapsed_ms"`
}

type freshnessSummary struct {
	Checked int `json:"checked"`
	Clean   int `json:"clean"`
	Stale   int `json:"stale"`
	Errors  int `json:"errors"`
	Skipped int `json:"skipped"`
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
}

type freshnessAction struct {
	Code       string `json:"code"`
	Message    string `json:"message"`
	Command    string `json:"command"`
	Fixability string `json:"fixability,omitempty"`
}

type goSurface struct {
	scenario string
	surface  string
	goMod    string
	module   string
	requires map[string]struct{}
}

type goModule struct {
	dir      string
	module   string
	requires map[string]struct{}
}

func runFreshness(args []string) error {
	fs := support.NewFlagSet("freshness")
	var repoRoot string
	var concurrency int
	var touched, all, apply, build, jsonOutput bool
	fs.StringVar(&repoRoot, "repo-root", "", "Repository root (defaults to current workspace)")
	fs.IntVar(&concurrency, "concurrency", 8, "Maximum package surfaces to check concurrently")
	fs.BoolVar(&touched, "touched", false, "Only check surfaces impacted by changed in-repo modules")
	fs.BoolVar(&all, "all", false, "Check every discovered Go scenario surface")
	fs.BoolVar(&apply, "apply", false, "Run go mod tidy on stale surfaces")
	fs.BoolVar(&build, "build", false, "Run go build ./... after tidy checks")
	fs.BoolVar(&jsonOutput, "json", false, "Output raw JSON")
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
	report, err := checkFreshness(context.Background(), root, freshnessRequest{touched: touched, apply: apply, build: build, concurrency: concurrency})
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
	apply       bool
	build       bool
	concurrency int
}

func checkFreshness(ctx context.Context, root string, req freshnessRequest) (freshnessReport, error) {
	start := time.Now()
	surfaces, err := discoverGoSurfaces(root)
	if err != nil {
		return freshnessReport{}, err
	}
	modules, err := discoverGoModules(root)
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
		impacted = impactedSurfaces(root, surfaces, modules, touchedPaths)
		touchedPaths = reportTouchedPaths(impacted)
		surfaces = filterImpactedSurfaces(surfaces, impacted)
	}

	report := freshnessReport{
		Clean:     true,
		Root:      root,
		Mode:      "all",
		Touched:   touchedPaths,
		Surfaces:  make([]freshnessSurfaceReport, 0, len(surfaces)),
		ElapsedMs: time.Since(start).Milliseconds(),
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
	scenariosDir := filepath.Join(root, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []goSurface
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		scenarioDir := filepath.Join(scenariosDir, entry.Name())
		err := filepath.WalkDir(scenarioDir, func(path string, d os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if d.IsDir() {
				switch d.Name() {
				case ".git", "node_modules", "vendor", "data", "dist", "build", ".cache":
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() != "go.mod" {
				return nil
			}
			view, err := parseGoModFile(path)
			if err != nil {
				return nil
			}
			surfaceRoot := filepath.Dir(path)
			rel, _ := filepath.Rel(scenarioDir, surfaceRoot)
			surfaceID := filepath.ToSlash(rel)
			if surfaceID == "." {
				surfaceID = entry.Name()
			}
			out = append(out, goSurface{
				scenario: entry.Name(),
				surface:  surfaceID,
				goMod:    path,
				module:   view.module,
				requires: view.requires,
			})
			return nil
		})
		if err != nil {
			return nil, err
		}
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
			case ".git", "node_modules", "vendor", "data", "dist", "build", ".cache":
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() != "go.mod" {
			return nil
		}
		view, err := parseGoModFile(path)
		if err != nil || view.module == "" {
			return nil
		}
		out[view.module] = goModule{
			dir:      filepath.Dir(path),
			module:   view.module,
			requires: view.requires,
		}
		return nil
	})
	return out, err
}

type goModView struct {
	module   string
	requires map[string]struct{}
}

func parseGoModFile(path string) (goModView, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return goModView{}, err
	}
	view := goModView{requires: map[string]struct{}{}}
	inRequireBlock := false
	for _, raw := range strings.Split(string(data), "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if rest, ok := strings.CutPrefix(line, "module "); ok {
			view.module = strings.Fields(rest)[0]
			continue
		}
		if inRequireBlock {
			if line == ")" {
				inRequireBlock = false
				continue
			}
			if fields := strings.Fields(line); len(fields) >= 2 {
				view.requires[fields[0]] = struct{}{}
			}
			continue
		}
		if strings.HasPrefix(line, "require (") {
			inRequireBlock = true
			continue
		}
		if rest, ok := strings.CutPrefix(line, "require "); ok {
			if fields := strings.Fields(rest); len(fields) >= 2 {
				view.requires[fields[0]] = struct{}{}
			}
		}
	}
	return view, nil
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

var runGoCommand = runGo

func runGo(ctx context.Context, dir string, args ...string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(ctx, 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = dir
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"GOWORK=off", "GOFLAGS="})
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

func changedPaths(root string) ([]string, error) {
	var combined []string
	for _, args := range [][]string{{"diff", "--name-only", "HEAD"}, {"diff", "--cached", "--name-only"}} {
		cmd := exec.Command("git", args...)
		cmd.Dir = root
		out, err := cmd.Output()
		if err != nil && args[0] == "diff" && args[len(args)-1] == "HEAD" {
			cmd = exec.Command("git", "status", "--porcelain=v1", "--untracked-files=no")
			cmd.Dir = root
			out, err = cmd.Output()
		}
		if err != nil {
			return nil, err
		}
		combined = append(combined, strings.Split(string(out), "\n")...)
	}
	seen := map[string]struct{}{}
	for _, line := range combined {
		path := strings.TrimSpace(line)
		if len(path) > 3 && path[1] == ' ' {
			path = strings.TrimSpace(path[3:])
		}
		if path != "" {
			seen[filepath.ToSlash(path)] = struct{}{}
		}
	}
	out := make([]string, 0, len(seen))
	for path := range seen {
		out = append(out, path)
	}
	sort.Strings(out)
	return out, nil
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
