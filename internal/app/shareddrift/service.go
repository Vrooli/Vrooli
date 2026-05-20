package shareddrift

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

const defaultConcurrency = 8

// sharedPackagePaths lists the in-repo shared package directories whose state
// affects dependent scenarios. The list is derived dynamically from each
// scenario's replace directives, but this is the canonical "trigger" list used
// for OnlyTouched filtering.
var sharedTriggerPathPrefixes = []string{
	"packages/api-core/",
	"packages/proto/",
	"packages/repo-contract-go/",
}

// rootGoModPaths trigger every dependent scenario because they govern the
// top-level module that scenarios replace with `=> ../../..`.
var rootGoModPaths = []string{"go.mod", "go.sum"}

type Service struct {
	Root string
}

type scenarioInfo struct {
	Path     string
	APIDir   string
	Replaces []replaceTarget
}

type replaceTarget struct {
	Module string
	Local  string
}

func (s Service) Check(req CheckRequest) (Report, error) {
	start := time.Now()
	root := strings.TrimSpace(s.Root)
	if root == "" {
		return Report{}, fmt.Errorf("repo root is required")
	}
	root = filepath.Clean(root)
	report := Report{
		Root:            root,
		OnlyTouchedUsed: req.OnlyTouched,
		BuildChecked:    req.Build,
		FixApplied:      req.Fix,
	}

	scenarios, err := discoverScenarios(root)
	if err != nil {
		return report, fmt.Errorf("discover scenarios: %w", err)
	}

	var touched []string
	if req.OnlyTouched {
		touched, err = touchedTriggers(root)
		if err != nil {
			return report, fmt.Errorf("read staged changes: %w", err)
		}
		report.TouchedPackages = touched
		scenarios = filterByTouched(scenarios, touched, root)
	}

	concurrency := req.Concurrency
	if concurrency <= 0 {
		concurrency = defaultConcurrency
	}

	results := make([]ScenarioReport, len(scenarios))
	sem := make(chan struct{}, concurrency)
	var wg sync.WaitGroup
	for i, sc := range scenarios {
		wg.Add(1)
		sem <- struct{}{}
		go func(idx int, info scenarioInfo) {
			defer wg.Done()
			defer func() { <-sem }()
			results[idx] = checkScenario(root, info, req)
		}(i, sc)
	}
	wg.Wait()

	report.Scenarios = results
	clean := true
	for _, r := range results {
		if r.Status != StatusClean && r.Status != StatusSkipped {
			clean = false
			break
		}
	}
	report.Clean = clean
	report.ElapsedMs = time.Since(start).Milliseconds()
	return report, nil
}

func discoverScenarios(root string) ([]scenarioInfo, error) {
	scenariosDir := filepath.Join(root, "scenarios")
	entries, err := os.ReadDir(scenariosDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var out []scenarioInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		apiDir := filepath.Join(scenariosDir, entry.Name(), "api")
		modPath := filepath.Join(apiDir, "go.mod")
		data, err := os.ReadFile(modPath)
		if err != nil {
			continue
		}
		replaces := parseLocalReplaces(string(data))
		if len(replaces) == 0 {
			continue
		}
		out = append(out, scenarioInfo{
			Path:     filepath.ToSlash(filepath.Join("scenarios", entry.Name())),
			APIDir:   apiDir,
			Replaces: replaces,
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out, nil
}

// parseLocalReplaces extracts replace directives whose target is a relative
// filesystem path (i.e. an in-repo replace). Handles single-line and block forms.
var replaceLineRe = regexp.MustCompile(`^([^\s]+)(?:\s+v\S+)?\s+=>\s+(\S+)`)

func parseLocalReplaces(content string) []replaceTarget {
	var out []replaceTarget
	inBlock := false
	for _, raw := range strings.Split(content, "\n") {
		line := strings.TrimSpace(raw)
		if line == "" || strings.HasPrefix(line, "//") {
			continue
		}
		if inBlock {
			if line == ")" {
				inBlock = false
				continue
			}
			if m := replaceLineRe.FindStringSubmatch(line); m != nil && isRelativePath(m[2]) {
				out = append(out, replaceTarget{Module: m[1], Local: m[2]})
			}
			continue
		}
		if strings.HasPrefix(line, "replace (") {
			inBlock = true
			continue
		}
		if strings.HasPrefix(line, "replace ") {
			rest := strings.TrimPrefix(line, "replace ")
			if m := replaceLineRe.FindStringSubmatch(rest); m != nil && isRelativePath(m[2]) {
				out = append(out, replaceTarget{Module: m[1], Local: m[2]})
			}
		}
	}
	return out
}

func isRelativePath(p string) bool {
	return strings.HasPrefix(p, ".") || strings.HasPrefix(p, "/")
}

// touchedTriggers returns the set of repo-relative path prefixes that have
// pending changes (staged or unstaged) and are relevant to drift checking.
func touchedTriggers(root string) ([]string, error) {
	cmd := exec.Command("git", "diff", "--name-only", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		// Fall back to working tree changes only if HEAD does not exist.
		cmd = exec.Command("git", "status", "--porcelain=v1", "--untracked-files=no")
		cmd.Dir = root
		out2, err2 := cmd.Output()
		if err2 != nil {
			return nil, err
		}
		out = out2
	}
	cachedCmd := exec.Command("git", "diff", "--cached", "--name-only")
	cachedCmd.Dir = root
	cachedOut, _ := cachedCmd.Output()

	seen := map[string]bool{}
	for _, line := range append(strings.Split(string(out), "\n"), strings.Split(string(cachedOut), "\n")...) {
		path := strings.TrimSpace(line)
		if len(path) > 3 && (path[0] == 'M' || path[0] == 'A' || path[0] == 'D' || path[0] == '?') && path[1] == ' ' {
			path = strings.TrimSpace(path[3:])
		}
		if path == "" {
			continue
		}
		seen[filepath.ToSlash(path)] = true
	}
	var paths []string
	for p := range seen {
		paths = append(paths, p)
	}
	sort.Strings(paths)
	return paths, nil
}

func filterByTouched(scenarios []scenarioInfo, touched []string, root string) []scenarioInfo {
	if len(touched) == 0 {
		return nil
	}
	rootTouched := false
	touchedPkgPrefixes := map[string]bool{}
	for _, p := range touched {
		for _, rp := range rootGoModPaths {
			if p == rp {
				rootTouched = true
			}
		}
		for _, prefix := range sharedTriggerPathPrefixes {
			if strings.HasPrefix(p, prefix) {
				touchedPkgPrefixes[strings.TrimSuffix(prefix, "/")] = true
			}
		}
	}
	if rootTouched {
		return scenarios
	}
	if len(touchedPkgPrefixes) == 0 {
		return nil
	}
	var out []scenarioInfo
	for _, sc := range scenarios {
		if scenarioTouches(sc, touchedPkgPrefixes, root) {
			out = append(out, sc)
		}
	}
	return out
}

func scenarioTouches(sc scenarioInfo, touchedPkgPrefixes map[string]bool, root string) bool {
	for _, r := range sc.Replaces {
		resolved := filepath.Clean(filepath.Join(sc.APIDir, r.Local))
		rel, err := filepath.Rel(root, resolved)
		if err != nil {
			continue
		}
		relSlash := filepath.ToSlash(rel)
		if touchedPkgPrefixes[relSlash] {
			return true
		}
	}
	return false
}

func checkScenario(root string, info scenarioInfo, req CheckRequest) ScenarioReport {
	rep := ScenarioReport{
		Path:   info.Path,
		APIDir: filepath.ToSlash(mustRel(root, info.APIDir)),
	}
	for _, r := range info.Replaces {
		rep.Replaces = append(rep.Replaces, r.Module)
	}

	modPath := filepath.Join(info.APIDir, "go.mod")
	sumPath := filepath.Join(info.APIDir, "go.sum")

	if _, err := os.ReadFile(modPath); err != nil {
		rep.Status = StatusError
		rep.Error = fmt.Sprintf("read go.mod: %v", err)
		return rep
	}

	diffOut, clean, err := runGoModTidyDiff(info.APIDir)
	if err != nil {
		rep.Status = StatusError
		rep.Error = fmt.Sprintf("go mod tidy -diff failed: %v", err)
		return rep
	}

	if !clean {
		rep.Status = StatusStaleModules
		var diffs []string
		if strings.Contains(diffOut, "go.mod") {
			diffs = append(diffs, filepath.ToSlash(mustRel(root, modPath)))
		}
		if strings.Contains(diffOut, "go.sum") {
			diffs = append(diffs, filepath.ToSlash(mustRel(root, sumPath)))
		}
		if len(diffs) == 0 {
			diffs = []string{filepath.ToSlash(mustRel(root, modPath))}
		}
		rep.DiffPaths = diffs
		if req.Fix {
			if err := runGoModTidy(info.APIDir); err != nil {
				rep.Error = fmt.Sprintf("go mod tidy --fix failed: %v", err)
			}
		}
		return rep
	}

	if req.Build {
		if errMsg := runGoBuild(info.APIDir); errMsg != "" {
			rep.Status = StatusStaleBuild
			rep.BuildError = errMsg
			return rep
		}
	}

	rep.Status = StatusClean
	return rep
}

// runGoModTidyDiff invokes `go mod tidy -diff` which is read-only. Exit code
// 0 means clean; non-zero with diff output on stdout means stale.
func runGoModTidyDiff(apiDir string) (string, bool, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy", "-diff")
	cmd.Dir = apiDir
	cmd.Env = append(os.Environ(), "GOFLAGS=")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	err := cmd.Run()
	if err == nil {
		return "", true, nil
	}
	if exitErr, ok := err.(*exec.ExitError); ok && exitErr.ExitCode() == 1 && stdout.Len() > 0 {
		return stdout.String(), false, nil
	}
	return "", false, fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
}

func runGoModTidy(apiDir string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "mod", "tidy")
	cmd.Dir = apiDir
	cmd.Env = append(os.Environ(), "GOFLAGS=")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("%v: %s", err, strings.TrimSpace(stderr.String()))
	}
	return nil
}

func runGoBuild(apiDir string) string {
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = apiDir
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return strings.TrimSpace(stderr.String())
	}
	return ""
}

func mustRel(root, p string) string {
	rel, err := filepath.Rel(root, p)
	if err != nil {
		return p
	}
	return rel
}
