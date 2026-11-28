package deployability

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/vrooli/envkit-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/shell"
	"github.com/vrooli/vrooli/internal/tuning"
)

// ConformanceTarget binds an authored platform claim to the Go module that
// implements it. The manifest path is retained in every result so a compiler
// failure points at the claim an operator must repair.
type ConformanceTarget struct {
	ManifestPath string `json:"manifest_path"`
	OS           HostOS `json:"os"`
	Architecture string `json:"architecture"`
	CodeRoot     string `json:"code_root"`
}

type ConformanceFinding struct {
	ManifestPath string `json:"manifest_path"`
	OS           HostOS `json:"os"`
	Architecture string `json:"architecture"`
	CodeRoot     string `json:"code_root"`
	Rule         string `json:"rule,omitempty"`
	Message      string `json:"message"`
}

type ConformanceReport struct {
	Targets  []ConformanceTarget  `json:"targets"`
	Findings []ConformanceFinding `json:"findings"`
	Warnings []ConformanceFinding `json:"warnings,omitempty"`
}

type conformanceWarning struct{ message string }

func (w conformanceWarning) Error() string { return w.message }

// DiscoverConformanceTargets finds every platform claim whose implementation
// is Go-backed. It intentionally discovers modules rather than maintaining a
// hand-written list of packages.
func DiscoverConformanceTargets(ctx context.Context, root string) ([]ConformanceTarget, error) {
	if ctx == nil {
		return nil, errors.New("conformance context is nil")
	}
	if err := ctx.Err(); err != nil {
		return nil, err
	}
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
		if target.Kind == repocontract.TargetKindDocs || target.Kind == repocontract.TargetKindTeam {
			continue
		}
		targetRoot := filepath.Join(root, filepath.FromSlash(target.Root))
		if err := filepath.WalkDir(targetRoot, func(path string, entry os.DirEntry, walkErr error) error {
			if err := ctx.Err(); err != nil {
				return err
			}
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() && (entry.Name() == "node_modules" || entry.Name() == "vendor" || (path != targetRoot && strings.EqualFold(entry.Name(), "docs"))) {
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
			for _, targetPlatform := range conformancePlatforms {
				targets = append(targets, ConformanceTarget{
					ManifestPath: conformanceManifest(root, target, module),
					OS:           HostOS(targetPlatform.hostOS),
					Architecture: targetPlatform.architecture,
					CodeRoot:     relativePath(root, module),
				})
			}
			return nil
		}); err != nil {
			return nil, err
		}
	}
	sort.Slice(targets, func(i, j int) bool { return targetKey(targets[i]) < targetKey(targets[j]) })
	return targets, nil
}

var conformancePlatforms = [...]struct{ hostOS, architecture string }{
	{hostOS: string(hostreqspec.PlatformLinux), architecture: "amd64"},
	{hostOS: string(hostreqspec.PlatformLinux), architecture: "arm64"},
	{hostOS: string(hostreqspec.PlatformMacOS), architecture: "amd64"},
	{hostOS: string(hostreqspec.PlatformMacOS), architecture: "arm64"},
	{hostOS: string(hostreqspec.PlatformWindows), architecture: "amd64"},
	{hostOS: string(hostreqspec.PlatformWindows), architecture: "arm64"},
}

// CheckRepository cross-compiles each discovered module/OS pair once and
// fans failures back out to every manifest making that claim.
func CheckRepository(ctx context.Context, root string, progress io.Writer) (ConformanceReport, error) {
	if ctx == nil {
		return ConformanceReport{}, errors.New("conformance context is nil")
	}
	targets, err := DiscoverConformanceTargets(ctx, root)
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
	workers := tuning.BuildWidth()
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
				results <- result{target: target, err: crossCompile(ctx, filepath.Join(root, target.CodeRoot), target.OS, target.Architecture)}
			}
		}()
	}
	go func() {
		for _, target := range targets {
			select {
			case jobs <- target:
			case <-ctx.Done():
				close(jobs)
				return
			}
		}
		close(jobs)
	}()
	go func() {
		wg.Wait()
		close(results)
	}()
	modules := make(map[string]struct{})
	for _, target := range targets {
		modules[target.CodeRoot] = struct{}{}
	}
	if progress != nil {
		_, _ = fmt.Fprintf(progress, "conformance: checking %d modules across %d target cells\n", len(modules), len(targets))
	}
	startedAt := time.Now()
	var tick <-chan time.Time
	if progress != nil {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		tick = ticker.C
	}
	progressStep := max(1, (len(targets)+9)/10)
	completed := 0
loop:
	for {
		select {
		case item, ok := <-results:
			if !ok {
				break loop
			}
			completed++
			if progress != nil && (completed%progressStep == 0 || completed == len(targets)) {
				_, _ = fmt.Fprintf(progress, "conformance: checked %d/%d target cells\n", completed, len(targets))
			}
			if item.err != nil {
				finding := ConformanceFinding{ManifestPath: item.target.ManifestPath, OS: item.target.OS, Architecture: item.target.Architecture, CodeRoot: item.target.CodeRoot, Message: item.err.Error()}
				var warning conformanceWarning
				if errors.As(item.err, &warning) {
					report.Warnings = append(report.Warnings, finding)
				} else {
					report.Findings = append(report.Findings, finding)
				}
			}
		case <-tick:
			_, _ = fmt.Fprintf(progress, "conformance: still running after %s; checked %d/%d target cells\n", time.Since(startedAt).Round(time.Second), completed, len(targets))
		}
	}
	if err := ctx.Err(); err != nil {
		return report, err
	}
	sort.Slice(report.Findings, func(i, j int) bool {
		return report.Findings[i].ManifestPath+string(report.Findings[i].OS) < report.Findings[j].ManifestPath+string(report.Findings[j].OS)
	})
	sort.Slice(report.Warnings, func(i, j int) bool {
		return report.Warnings[i].ManifestPath+string(report.Warnings[i].OS) < report.Warnings[j].ManifestPath+string(report.Warnings[j].OS)
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

func crossCompile(ctx context.Context, module string, hostOS HostOS, architecture string) error {
	arch := architecture
	goos := string(hostOS)
	if hostOS == HostOSMacOS {
		goos = string(hostreqspec.PlatformDarwin)
	}
	cmd := shell.NewCommandContext(ctx, "go", "vet", "./...")
	cmd.Dir = module
	cmd.Env = envkit.Toolchain(
		envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"GOWORK=off", "GOOS=" + goos, "GOARCH=" + arch}),
		envkit.ToolchainOptions{Width: 1},
	)
	out, err := cmd.CombinedOutput()
	if err != nil {
		message := fmt.Sprintf("GOOS=%s GOARCH=%s: %v: %s", goos, arch, err, strings.TrimSpace(string(out)))
		if isVetStyleDiagnostic(string(out)) {
			return conformanceWarning{message: message}
		}
		return fmt.Errorf("%s", message)
	}
	return nil
}

func isVetStyleDiagnostic(output string) bool {
	trimmed := strings.TrimSpace(output)
	if trimmed == "" || strings.Contains(trimmed, "vet:") {
		return false
	}
	for _, marker := range []string{
		"undefined:",
		"unknown field",
		"cannot use ",
		"syntax error",
		"build constraints exclude all Go files",
		"no required module provides package",
		"import cycle",
		"redeclared in this block",
	} {
		if strings.Contains(trimmed, marker) {
			return false
		}
	}
	// Vet diagnostics that do not contain a load/compile marker are warnings.
	// This includes analyzer findings, and toolchain housekeeping such as a
	// module whose go.mod would be rewritten by the selected target. The caller
	// still records the complete message; only a package that could not load is
	// a conformance failure.
	return true
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
	return t.ManifestPath + "\x00" + string(t.OS) + "\x00" + t.Architecture + "\x00" + t.CodeRoot
}
