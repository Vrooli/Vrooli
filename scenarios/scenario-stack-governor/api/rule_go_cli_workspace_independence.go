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

func RunGoCliWorkspaceIndependence(ctx context.Context, repoRoot, scenarioName string) (result RuleResult) {
	start := time.Now()
	result = RuleResult{
		RuleID:    "GO_CLI_WORKSPACE_INDEPENDENCE",
		StartedAt: start,
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.Passed = len(result.Findings) == 0
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
	if len(goMods) == 0 {
		// No Go CLIs found. For per-scenario runs this is expected (most
		// scenarios don't have a Go CLI), so we return no findings and
		// let the rule pass. For full-repo runs it simply means none exist.
		return result
	}

	var mu sync.Mutex
	g, gCtx := errgroup.WithContext(ctx)
	g.SetLimit(5)
	for _, goMod := range goMods {
		goMod := goMod
		g.Go(func() error {
			moduleDir := filepath.Dir(goMod)
			name := filepath.Base(filepath.Dir(moduleDir)) // scenario slug
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
	_ = g.Wait()

	result.Findings = append(result.Findings, checkCliInternalImports(repoRoot, cleanedScenario)...)
	result.Findings = append(result.Findings, checkProtoReplaceForCliModules(repoRoot, cleanedScenario)...)
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
		usesInternal := false
		_ = filepath.WalkDir(cliDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil || d.IsDir() {
				return nil
			}
			if !strings.HasSuffix(path, ".go") {
				return nil
			}
			b, err := os.ReadFile(path)
			if err != nil {
				return nil
			}
			if bytes.Contains(b, []byte(`"`+apiModule+`/internal/`)) {
				usesInternal = true
			}
			return nil
		})
		if !usesInternal {
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
				Message:      fmt.Sprintf("%s: CLI imports %s/internal/* but is missing go.mod wiring", cliModule, apiModule),
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

	const protoModule = "github.com/vrooli/vrooli/packages/proto"

	for _, cliGoMod := range cliGoMods {
		modFile := parseGoModFile(cliGoMod)
		if modFile == nil {
			continue
		}

		// Check if proto is referenced (in require or anywhere in the parsed module).
		hasProto := false
		for _, req := range modFile.Require {
			if req.Mod.Path == protoModule {
				hasProto = true
				break
			}
		}
		if !hasProto {
			// Also check raw text for proto references (could be in replace without require).
			raw, err := os.ReadFile(cliGoMod)
			if err != nil {
				continue
			}
			if !strings.Contains(string(raw), protoModule) {
				continue
			}
			hasProto = true
		}

		if goModFileHasReplace(modFile, protoModule) {
			continue
		}

		scenSlug := filepath.Base(filepath.Dir(filepath.Dir(cliGoMod)))
		findings = append(findings, Finding{
			Level:        "error",
			Message:      "CLI depends on packages/proto but is missing a local replace directive (replace is not transitive)",
			ScenarioName: scenSlug,
			Evidence: []Evidence{
				{Type: "file", Ref: cliGoMod},
				{Type: "note", Detail: "Add: replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto (path adjusted per module depth)"},
			},
		})
	}

	return findings
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

// checkReplaceDirectivePaths validates that local replace directives in CLI go.mod
// files point to directories that actually exist on disk.
func checkReplaceDirectivePaths(repoRoot, scenarioName string) []Finding {
	var findings []Finding

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
		modFile := parseGoModFile(cliGoMod)
		if modFile == nil {
			continue
		}

		scenSlug := filepath.Base(filepath.Dir(filepath.Dir(cliGoMod)))
		cliDir := filepath.Dir(cliGoMod)

		for _, rep := range modFile.Replace {
			// Only check local (relative path) replacements.
			if rep.New.Path == "" || !isLocalReplacePath(rep.New.Path) {
				continue
			}

			targetDir := filepath.Join(cliDir, rep.New.Path)
			if _, err := os.Stat(targetDir); os.IsNotExist(err) {
				findings = append(findings, Finding{
					Level:        "warn",
					Message:      fmt.Sprintf("replace directive for %s points to non-existent path: %s", rep.Old.Path, rep.New.Path),
					ScenarioName: scenSlug,
					Evidence: []Evidence{
						{Type: "file", Ref: cliGoMod},
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
