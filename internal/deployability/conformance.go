package deployability

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"

	"github.com/vrooli/envkit-go"
	repocontract "github.com/vrooli/repo-contract-go"
)

// ConformanceTarget binds an authored platform claim to the Go module that
// implements it. The manifest path is retained in every result so a compiler
// failure points at the claim an operator must repair.
type ConformanceTarget struct {
	ManifestPath string `json:"manifest_path"`
	OS           HostOS `json:"os"`
	CodeRoot     string `json:"code_root"`
}

type ConformanceFinding struct {
	ManifestPath string `json:"manifest_path"`
	OS           HostOS `json:"os"`
	CodeRoot     string `json:"code_root"`
	Message      string `json:"message"`
}

type ConformanceReport struct {
	Targets  []ConformanceTarget  `json:"targets"`
	Findings []ConformanceFinding `json:"findings"`
}

// DiscoverConformanceTargets finds every platform claim whose implementation
// is Go-backed. It intentionally discovers modules rather than maintaining a
// hand-written list of packages.
func DiscoverConformanceTargets(root string) ([]ConformanceTarget, error) {
	contract, err := repocontract.LoadDefault(root)
	if err != nil {
		return nil, err
	}
	concrete, err := contract.EnumerateTargets(root)
	if err != nil {
		return nil, err
	}
	sort.SliceStable(concrete, func(i, j int) bool { return len(concrete[i].Root) > len(concrete[j].Root) })
	var targets []ConformanceTarget
	seenModules := map[string]struct{}{}
	for _, target := range concrete {
		targetRoot := filepath.Join(root, filepath.FromSlash(target.Root))
		if err := filepath.WalkDir(targetRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && (entry.Name() == "node_modules" || entry.Name() == "vendor") {
				return filepath.SkipDir
			}
			if target.Kind == repocontract.TargetKindProject && path != targetRoot && entry.IsDir() {
				return filepath.SkipDir
			}
			if entry.IsDir() || entry.Name() != "go.mod" || isFixtureModule(path) {
				return nil
			}
			module := filepath.Dir(path)
			key := filepath.Clean(module)
			if _, ok := seenModules[key]; ok {
				return nil
			}
			seenModules[key] = struct{}{}
			for _, hostOS := range []HostOS{HostOSMacOS, HostOSWindows} {
				targets = append(targets, ConformanceTarget{
					ManifestPath: conformanceManifest(root, target, module),
					OS:           hostOS,
					CodeRoot:     relativePath(root, module),
				})
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targetKey(targets[i]) < targetKey(targets[j]) })
	return dedupeTargets(targets), nil
}

// CheckRepository cross-compiles each discovered module/OS pair once and
// fans failures back out to every manifest making that claim.
func CheckRepository(ctx context.Context, root string) (ConformanceReport, error) {
	targets, err := DiscoverConformanceTargets(root)
	if err != nil {
		return ConformanceReport{}, err
	}
	report := ConformanceReport{Targets: targets}
	type result struct {
		target ConformanceTarget
		err    error
	}
	jobs := make(chan ConformanceTarget)
	results := make(chan result, len(targets))
	workers := 8
	if len(targets) < workers {
		workers = len(targets)
	}
	if workers == 0 {
		return report, nil
	}
	var wg sync.WaitGroup
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for target := range jobs {
				results <- result{target: target, err: crossCompile(ctx, filepath.Join(root, target.CodeRoot), target.OS)}
			}
		}()
	}
	go func() {
		for _, target := range targets {
			jobs <- target
		}
		close(jobs)
		wg.Wait()
		close(results)
	}()
	for item := range results {
		if item.err != nil {
			report.Findings = append(report.Findings, ConformanceFinding{ManifestPath: item.target.ManifestPath, OS: item.target.OS, CodeRoot: item.target.CodeRoot, Message: item.err.Error()})
		}
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		return report.Findings[i].ManifestPath+string(report.Findings[i].OS) < report.Findings[j].ManifestPath+string(report.Findings[j].OS)
	})
	return report, nil
}

func conformanceManifest(root string, target repocontract.Target, module string) string {
	for _, marker := range []string{".vrooli/service.json", "resource.json", "tool.json", "safeguard.json", "manifest.json"} {
		candidate := filepath.Join(root, filepath.FromSlash(target.Root), filepath.FromSlash(marker))
		if _, err := os.Stat(candidate); err == nil {
			return relativePath(root, candidate)
		}
	}
	return relativePath(root, filepath.Join(module, "go.mod"))
}

func isFixtureModule(path string) bool {
	for dir := filepath.Dir(path); dir != filepath.Dir(dir); dir = filepath.Dir(dir) {
		if filepath.Base(dir) == "fixtures" {
			return true
		}
	}
	return false
}

func crossCompile(ctx context.Context, module string, hostOS HostOS) error {
	arch := "amd64"
	goos := string(hostOS)
	if hostOS == HostOSMacOS {
		goos = "darwin"
		arch = "arm64"
	}
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = module
	cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"GOWORK=off", "GOOS=" + goos, "GOARCH=" + arch})
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("GOOS=%s GOARCH=%s: %w: %s", goos, arch, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func decodeJSON(path string, value any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(data, value); err != nil {
		return fmt.Errorf("parse %s: %w", path, err)
	}
	return nil
}
func relativePath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return filepath.ToSlash(rel)
}
func targetKey(t ConformanceTarget) string {
	return t.ManifestPath + "\x00" + string(t.OS) + "\x00" + t.CodeRoot
}
func dedupeTargets(in []ConformanceTarget) []ConformanceTarget {
	out := in[:0]
	seen := map[string]bool{}
	for _, target := range in {
		key := targetKey(target)
		if !seen[key] {
			seen[key] = true
			out = append(out, target)
		}
	}
	return out
}

func nearestGoModule(root, start string) (string, error) {
	path := start
	for {
		if _, err := os.Stat(filepath.Join(path, "go.mod")); err == nil {
			return path, nil
		}
		if path == root || filepath.Dir(path) == path {
			break
		}
		path = filepath.Dir(path)
	}
	return "", fmt.Errorf("no go.mod found from %s", start)
}

func conformanceHostOS(value string) (HostOS, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case string(HostOSLinux):
		return HostOSLinux, true
	case string(HostOSMacOS), "darwin":
		return HostOSMacOS, true
	case string(HostOSWindows):
		return HostOSWindows, true
	default:
		return "", false
	}
}
