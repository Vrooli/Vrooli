package main

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"golang.org/x/mod/modfile"
	"golang.org/x/sync/errgroup"
)

const (
	apiCoreModule      = "github.com/vrooli/api-core"
	cliCoreModule      = "github.com/vrooli/cli-core"
	protoModule        = "github.com/vrooli/vrooli/packages/proto"
	repoContractModule = "github.com/vrooli/repo-contract-go"
)

func RunGoCliWorkspaceIndependence(ctx context.Context, repoRoot, scenarioName string) (result RuleResult) {
	start := time.Now()
	result = RuleResult{
		RuleID:    "GO_CLI_WORKSPACE_INDEPENDENCE",
		StartedAt: start,
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.Passed = !hasActionableFindings(result.Findings)
	}()

	glob := filepath.Join(repoRoot, "scenarios", "*", "cli", "go.mod")
	cleanedScenario := strings.TrimSpace(scenarioName)
	if cleanedScenario != "" {
		glob = filepath.Join(repoRoot, "scenarios", cleanedScenario, "cli", "go.mod")
	}

	goMods, err := filepath.Glob(glob)
	if err != nil {
		result.Findings = append(result.Findings, Finding{
			Level:   "error",
			Message: fmt.Sprintf("failed to list CLI modules: %v", err),
		})
		return result
	}

	sort.Strings(goMods)
	if len(goMods) > 0 {
		var mu sync.Mutex
		g, gCtx := errgroup.WithContext(ctx)
		g.SetLimit(5)
		for _, goMod := range goMods {
			goMod := goMod
			g.Go(func() (retErr error) {
				moduleDir := filepath.Dir(goMod)
				name := filepath.Base(filepath.Dir(moduleDir)) // scenario slug

				// Recover from unexpected panics so a single module can't crash
				// the entire rule run.
				defer func() {
					if r := recover(); r != nil {
						mu.Lock()
						result.Findings = append(result.Findings, Finding{
							Level:        "error",
							Message:      fmt.Sprintf("%s: internal error (panic): %v", name, r),
							ScenarioName: name,
							Evidence: []Evidence{
								{Type: "path", Ref: moduleDir},
							},
						})
						mu.Unlock()
					}
				}()

				buildCtx, cancel := context.WithTimeout(gCtx, 3*time.Minute)
				out, buildErr := runGoBuild(buildCtx, moduleDir)
				wasTimeout := buildCtx.Err() == context.DeadlineExceeded
				cancel()
				if buildErr != nil {
					mu.Lock()
					if wasTimeout {
						result.Findings = append(result.Findings, Finding{
							Level:        "error",
							Message:      fmt.Sprintf("%s: build timed out after 3 minutes", name),
							ScenarioName: name,
							Evidence: []Evidence{
								{Type: "path", Ref: moduleDir},
								{Type: "command", Ref: "GOWORK=off go build ./..."},
							},
						})
					} else {
						result.Findings = append(result.Findings, Finding{
							Level:        "error",
							Message:      fmt.Sprintf("%s: `GOWORK=off go build ./...` failed", name),
							ScenarioName: name,
							Evidence: []Evidence{
								{Type: "path", Ref: moduleDir},
								{Type: "command", Ref: "GOWORK=off go build ./..."},
								{Type: "note", Detail: out},
							},
						})
					}
					mu.Unlock()
				}
				return nil
			})
		}
		if err := g.Wait(); err != nil {
			result.Findings = append(result.Findings, Finding{
				Level:   "error",
				Message: fmt.Sprintf("unexpected error during parallel build checks: %v", err),
			})
		}
	}

	result.Findings = append(result.Findings, checkCliInternalImports(repoRoot, cleanedScenario)...)
	result.Findings = append(result.Findings, checkSharedModuleReplaceDirectives(repoRoot, cleanedScenario)...)
	result.Findings = append(result.Findings, checkReplaceDirectivePaths(repoRoot, cleanedScenario)...)
	return result
}

func runGoBuild(ctx context.Context, dir string) (string, error) {
	cmd := exec.CommandContext(ctx, "go", "build", "./...")
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GOWORK=off")
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	err := cmd.Run()
	out := strings.TrimSpace(buf.String())
	if ctx.Err() != nil {
		return out, ctx.Err()
	}
	return out, err
}

func checkCliInternalImports(repoRoot, scenarioName string) []Finding {
	findings := []Finding{}

	glob := filepath.Join(repoRoot, "scenarios", "*", "cli", "go.mod")
	if strings.TrimSpace(scenarioName) != "" {
		glob = filepath.Join(repoRoot, "scenarios", strings.TrimSpace(scenarioName), "cli", "go.mod")
	}
	cliGoMods, err := filepath.Glob(glob)
	if err != nil {
		return []Finding{{Level: "error", Message: fmt.Sprintf("failed to glob CLI go.mod files: %v", err)}}
	}
	sort.Strings(cliGoMods)

	for _, cliGoMod := range cliGoMods {
		scenarioDir := filepath.Dir(filepath.Dir(cliGoMod))
		scenSlug := filepath.Base(scenarioDir)
		apiGoMod := filepath.Join(scenarioDir, "api", "go.mod")
		if !fileExists(apiGoMod) {
			continue
		}

		apiModule := parseGoModModule(apiGoMod)
		if apiModule == "" {
			continue
		}

		cliDir := filepath.Dir(cliGoMod)
		if !cliImportsAPI(cliDir, apiModule) {
			continue
		}

		cliModule := parseGoModModule(cliGoMod)
		if cliModule == "" {
			cliModule = filepath.Base(scenarioDir) + "/cli"
		}

		cliModFile := parseGoModFile(cliGoMod)

		missing := []string{}
		if !goModFileHasReplace(cliModFile, apiModule) {
			missing = append(missing, fmt.Sprintf("replace %s => ../api", apiModule))
		}
		if !goModFileHasRequire(cliModFile, apiModule) {
			missing = append(missing, "require "+apiModule+" v0.0.0 (or similar)")
		}

		if len(missing) > 0 {
			findings = append(findings, Finding{
				Level:        "error",
				Message:      fmt.Sprintf("%s: CLI imports %s/* but is missing go.mod wiring", cliModule, apiModule),
				ScenarioName: scenSlug,
				Evidence: []Evidence{
					{Type: "file", Ref: cliGoMod},
					{Type: "note", Detail: "Add: " + strings.Join(missing, " | ")},
				},
			})
		}
	}

	return findings
}

func checkProtoReplaceForCliModules(repoRoot, scenarioName string) []Finding {
	return nil
}

// parseGoModModule extracts the module path from a go.mod file using golang.org/x/mod/modfile.
func parseGoModModule(goModPath string) string {
	f := parseGoModFile(goModPath)
	if f == nil || f.Module == nil {
		return ""
	}
	return f.Module.Mod.Path
}

// parseGoModFile parses a go.mod file into a structured representation.
func parseGoModFile(goModPath string) *modfile.File {
	b, err := os.ReadFile(goModPath)
	if err != nil {
		return nil
	}
	f, err := modfile.Parse(goModPath, b, nil)
	if err != nil {
		return nil
	}
	return f
}

// goModFileHasReplace checks if the parsed go.mod has a replace directive for the given module.
func goModFileHasReplace(f *modfile.File, module string) bool {
	if f == nil {
		return false
	}
	for _, rep := range f.Replace {
		if rep.Old.Path == module {
			return true
		}
	}
	return false
}

// goModFileHasRequire checks if the parsed go.mod has a require directive for the given module.
func goModFileHasRequire(f *modfile.File, module string) bool {
	if f == nil {
		return false
	}
	for _, req := range f.Require {
		if req.Mod.Path == module {
			return true
		}
	}
	return false
}

// checkReplaceDirectivePaths validates that local replace directives in scenario go.mod
// files point to directories that actually exist on disk.
func checkReplaceDirectivePaths(repoRoot, scenarioName string) []Finding {
	var findings []Finding

	goMods, err := listScenarioGoModFiles(repoRoot, scenarioName)
	if err != nil {
		return []Finding{{Level: "error", Message: fmt.Sprintf("failed to list scenario go.mod files: %v", err)}}
	}

	for _, goModPath := range goMods {
		modFile := parseGoModFile(goModPath)
		if modFile == nil {
			continue
		}

		scenSlug := scenarioNameFromGoModPath(goModPath)
		moduleDir := filepath.Dir(goModPath)

		for _, rep := range modFile.Replace {
			// Only check local (relative path) replacements.
			if rep.New.Path == "" || !isLocalReplacePath(rep.New.Path) {
				continue
			}

			targetDir := filepath.Join(moduleDir, rep.New.Path)
			if _, err := os.Stat(targetDir); os.IsNotExist(err) {
				findings = append(findings, Finding{
					Level:        "warn",
					Message:      fmt.Sprintf("replace directive for %s points to non-existent path: %s", rep.Old.Path, rep.New.Path),
					ScenarioName: scenSlug,
					Evidence: []Evidence{
						{Type: "file", Ref: goModPath},
						{Type: "path", Ref: targetDir},
						{Type: "note", Detail: "The target directory does not exist. Check that the relative path is correct and the dependency is present."},
					},
				})
			}
		}
	}

	return findings
}

// isLocalReplacePath returns true if the path is a relative filesystem path
// (used for local module replacements) rather than a module path.
func isLocalReplacePath(p string) bool {
	return strings.HasPrefix(p, ".") || strings.HasPrefix(p, "/")
}

func checkSharedModuleReplaceDirectives(repoRoot, scenarioName string) []Finding {
	goMods, err := listScenarioGoModFiles(repoRoot, scenarioName)
	if err != nil {
		return []Finding{{Level: "error", Message: fmt.Sprintf("failed to list scenario go.mod files: %v", err)}}
	}

	findings := []Finding{}
	for _, goModPath := range goMods {
		modFile := parseGoModFile(goModPath)
		if modFile == nil {
			continue
		}

		scenSlug := scenarioNameFromGoModPath(goModPath)
		if requiresAny(modFile, apiCoreModule, cliCoreModule) && !goModFileHasReplace(modFile, repoContractModule) {
			findings = append(findings, Finding{
				Level:        "error",
				Message:      "Module depends on api-core/cli-core but is missing a local replace directive for repo-contract-go (replace is not transitive)",
				ScenarioName: scenSlug,
				Evidence: []Evidence{
					{Type: "file", Ref: goModPath},
					{Type: "note", Detail: "Add: replace github.com/vrooli/repo-contract-go => <relative path to packages/repo-contract-go>"},
				},
			})
		}

		if goModFileHasRequire(modFile, protoModule) && !goModFileHasReplace(modFile, protoModule) {
			findings = append(findings, Finding{
				Level:        "error",
				Message:      "Module depends on packages/proto but is missing a local replace directive (replace is not transitive)",
				ScenarioName: scenSlug,
				Evidence: []Evidence{
					{Type: "file", Ref: goModPath},
					{Type: "note", Detail: "Add: replace github.com/vrooli/vrooli/packages/proto => <relative path to packages/proto>"},
				},
			})
		}
	}

	return findings
}

func requiresAny(f *modfile.File, modules ...string) bool {
	for _, module := range modules {
		if goModFileHasRequire(f, module) {
			return true
		}
	}
	return false
}

func listScenarioGoModFiles(repoRoot, scenarioName string) ([]string, error) {
	base := filepath.Join(repoRoot, "scenarios")
	if strings.TrimSpace(scenarioName) != "" {
		base = filepath.Join(base, strings.TrimSpace(scenarioName))
	}

	var goMods []string
	err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", "node_modules", "dist", "build", ".next", ".turbo", "tmp", "coverage":
				return filepath.SkipDir
			}
			return nil
		}
		if filepath.Base(path) == "go.mod" {
			goMods = append(goMods, path)
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Strings(goMods)
	return goMods, nil
}

func scenarioNameFromGoModPath(goModPath string) string {
	parts := strings.Split(filepath.ToSlash(goModPath), "/")
	for i := 0; i < len(parts)-1; i++ {
		if parts[i] == "scenarios" && i+1 < len(parts) {
			return parts[i+1]
		}
	}
	return ""
}
