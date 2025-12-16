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
	"time"
)

func RunGoCliWorkspaceIndependence(ctx context.Context, repoRoot string) RuleResult {
	start := time.Now()
	result := RuleResult{
		RuleID:    "GO_CLI_WORKSPACE_INDEPENDENCE",
		StartedAt: start,
	}
	defer func() {
		result.FinishedAt = time.Now()
		result.Passed = len(result.Findings) == 0
	}()

	goMods, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", "cli", "go.mod"))
	if err != nil {
		result.Findings = append(result.Findings, Finding{
			Level:   "error",
			Message: fmt.Sprintf("failed to list CLI modules: %v", err),
		})
		return result
	}

	sort.Strings(goMods)
	if len(goMods) == 0 {
		result.Findings = append(result.Findings, Finding{
			Level:   "warn",
			Message: "no Go-based scenario CLIs found under scenarios/*/cli/go.mod",
		})
		return result
	}

	for _, goMod := range goMods {
		moduleDir := filepath.Dir(goMod)
		name := filepath.Base(filepath.Dir(moduleDir)) // scenario slug
		buildCtx, cancel := context.WithTimeout(ctx, 3*time.Minute)
		out, buildErr := runGoBuild(buildCtx, moduleDir)
		cancel()
		if buildErr != nil {
			result.Findings = append(result.Findings, Finding{
				Level:   "error",
				Message: fmt.Sprintf("%s: `GOWORK=off go build ./...` failed", name),
				Evidence: []Evidence{
					{Type: "path", Ref: moduleDir},
					{Type: "command", Ref: "GOWORK=off go build ./..."},
					{Type: "note", Detail: out},
				},
			})
		}
	}

	result.Findings = append(result.Findings, checkCliInternalImports(repoRoot)...)
	result.Findings = append(result.Findings, checkProtoReplaceForCliModules(repoRoot)...)
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

func checkCliInternalImports(repoRoot string) []Finding {
	findings := []Finding{}

	cliGoMods, _ := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", "cli", "go.mod"))
	sort.Strings(cliGoMods)

	for _, cliGoMod := range cliGoMods {
		scenarioDir := filepath.Dir(filepath.Dir(cliGoMod))
		apiGoMod := filepath.Join(scenarioDir, "api", "go.mod")
		if !fileExists(apiGoMod) {
			continue
		}

		apiModule := parseGoModuleLine(apiGoMod)
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

		cliModule := parseGoModuleLine(cliGoMod)
		if cliModule == "" {
			cliModule = filepath.Base(scenarioDir) + "/cli"
		}

		raw, _ := os.ReadFile(cliGoMod)
		text := string(raw)
		expectReplace := fmt.Sprintf("replace %s => ../api", apiModule)
		expectRequire := fmt.Sprintf("\t%s ", apiModule)

		missing := []string{}
		if !strings.Contains(text, expectReplace) {
			missing = append(missing, expectReplace)
		}
		if !strings.Contains(text, expectRequire) && !strings.Contains(text, "\n"+apiModule+" ") {
			missing = append(missing, "require "+apiModule+" v0.0.0 (or similar)")
		}

		if len(missing) > 0 {
			findings = append(findings, Finding{
				Level:   "error",
				Message: fmt.Sprintf("%s: CLI imports %s/internal/* but is missing go.mod wiring", cliModule, apiModule),
				Evidence: []Evidence{
					{Type: "file", Ref: cliGoMod},
					{Type: "note", Detail: "Add: " + strings.Join(missing, " | ")},
				},
			})
		}
	}

	return findings
}

func checkProtoReplaceForCliModules(repoRoot string) []Finding {
	findings := []Finding{}

	cliGoMods, _ := filepath.Glob(filepath.Join(repoRoot, "scenarios", "*", "cli", "go.mod"))
	sort.Strings(cliGoMods)

	for _, cliGoMod := range cliGoMods {
		raw, err := os.ReadFile(cliGoMod)
		if err != nil {
			continue
		}
		text := string(raw)
		if !strings.Contains(text, "github.com/vrooli/vrooli/packages/proto") {
			continue
		}
		if strings.Contains(text, "replace github.com/vrooli/vrooli/packages/proto =>") {
			continue
		}

		findings = append(findings, Finding{
			Level:   "error",
			Message: "CLI depends on packages/proto but is missing a local replace directive (replace is not transitive)",
			Evidence: []Evidence{
				{Type: "file", Ref: cliGoMod},
				{Type: "note", Detail: "Add: replace github.com/vrooli/vrooli/packages/proto => ../../../packages/proto (path adjusted per module depth)"},
			},
		})
	}

	return findings
}

func parseGoModuleLine(goModPath string) string {
	b, err := os.ReadFile(goModPath)
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "module "))
		}
	}
	return ""
}

