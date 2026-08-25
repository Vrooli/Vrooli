package preview

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/vrooli/envkit-go"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliutil"
)

var (
	previewStoreSurfaceUnsafe   = regexp.MustCompile(`[^a-zA-Z0-9]+`)
	runPreviewDependencyInstall = func(ctx context.Context, repoRoot, packageName, versionRange, surface string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, "scenario-dependency-analyzer", "deps", "install", "npm/"+packageName+"@"+versionRange,
			"--scenario", "react-component-library", "--surface", "tools/"+surface, "--apply", "--json")
		cmd.Dir = repoRoot
		cmd.Env = envkit.WithOverlay(envkit.Env(os.Environ()), envkit.SameScenario, envkit.Env{"VROOLI_ROOT=" + repoRoot})
		return cmd.CombinedOutput()
	}
)

type previewStoreDeclaration struct {
	Name  string
	Range string
}

func (h *handlers) populateStore(ctx cliapp.RunContext) error {
	repoRoot := cliutil.ResolveRepoRoot()
	componentID := strings.TrimSpace(ctx.Positional("component-id"))
	version := strings.TrimSpace(ctx.Flag("version"))
	source, err := findPreviewAssetSource(repoRoot, componentID, version)
	if err != nil {
		return err
	}
	declarations, err := readPreviewAssetDeclarations(source)
	if err != nil {
		return err
	}
	if len(declarations) == 0 {
		return ctx.RenderList(cliapp.ListReport{
			Summary: []string{fmt.Sprintf("No external runtime dependencies declared by %s@%s.", componentID, version)},
		})
	}

	var results []string
	for _, declaration := range declarations {
		if declaration.Name == "react" || declaration.Name == "react-dom" || strings.HasPrefix(declaration.Name, "react/") {
			results = append(results, fmt.Sprintf("Kept %s@%s vendored in the React runtime.", declaration.Name, declaration.Range))
			continue
		}
		surface := previewStoreSurfaceName(declaration.Name, declaration.Range)
		if err := ensurePreviewStoreSurface(repoRoot, surface, declaration); err != nil {
			return err
		}
		output, installErr := runPreviewDependencyInstall(context.Background(), repoRoot, declaration.Name, declaration.Range, surface)
		if installErr != nil {
			return fmt.Errorf("populate %s@%s through scenario-dependency-analyzer: %w\n%s", declaration.Name, declaration.Range, installErr, strings.TrimSpace(string(output)))
		}
		results = append(results, fmt.Sprintf("Installed %s@%s in tools/%s through Scenario Dependency Analyzer.", declaration.Name, declaration.Range, surface))
	}
	sort.Strings(results)
	return ctx.RenderList(cliapp.ListReport{
		Summary:        []string{fmt.Sprintf("Populated governed preview runtime store for %s@%s.", componentID, version)},
		ResultsHeading: "Store changes",
		Results:        results,
		RetrievalHints: []string{"Preview dependencies resolve from scenarios/react-component-library/tools/preview-runtime-*; React remains vendored."},
	})
}

func previewStoreSurfaceName(name, versionRange string) string {
	key := previewStoreSurfaceUnsafe.ReplaceAllString(strings.TrimSpace(name+"-"+versionRange), "-")
	key = strings.Trim(key, "-")
	if key == "" {
		key = "dependency"
	}
	return "preview-runtime-" + strings.ToLower(key)
}

func ensurePreviewStoreSurface(repoRoot, surface string, declaration previewStoreDeclaration) error {
	dir := filepath.Join(repoRoot, "scenarios", "react-component-library", "tools", surface)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create governed preview surface %s: %w", surface, err)
	}
	manifest := map[string]any{
		"name":    "@vrooli/preview-" + surface,
		"private": true,
		"dependencies": map[string]string{
			declaration.Name: declaration.Range,
		},
	}
	raw, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("encode preview surface manifest: %w", err)
	}
	raw = append(raw, '\n')
	if err := os.WriteFile(filepath.Join(dir, "package.json"), raw, 0o644); err != nil {
		return fmt.Errorf("write governed preview surface manifest: %w", err)
	}
	lockPath := filepath.Join(dir, "pnpm-lock.yaml")
	if _, err := os.Stat(lockPath); os.IsNotExist(err) {
		if err := os.WriteFile(lockPath, []byte("lockfileVersion: '9.0'\n"), 0o644); err != nil {
			return fmt.Errorf("write governed preview surface lockfile: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check governed preview surface lockfile: %w", err)
	}
	configPath := filepath.Join(dir, ".npmrc")
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		if err := os.WriteFile(configPath, []byte("store-dir=~/.pnpm-store\n"), 0o644); err != nil {
			return fmt.Errorf("write governed preview package-manager config: %w", err)
		}
	} else if err != nil {
		return fmt.Errorf("check governed preview package-manager config: %w", err)
	}
	return nil
}

func findPreviewAssetSource(repoRoot, componentID, version string) (string, error) {
	componentID = strings.TrimSpace(componentID)
	if index := strings.LastIndex(componentID, ":"); index >= 0 {
		componentID = componentID[index+1:]
	}
	componentID = filepath.Base(filepath.Clean(componentID))
	version = strings.TrimSpace(version)
	if componentID == "." || componentID == "" || version == "" || strings.Contains(componentID, "..") || strings.Contains(version, "/") {
		return "", fmt.Errorf("preview store populate requires a safe component id and version")
	}
	for _, kind := range []string{"components", "hooks"} {
		pattern := filepath.Join(repoRoot, "scenarios", "react-component-library", "library", kind, componentID, "versions", version, "*.tsx")
		matches, err := filepath.Glob(pattern)
		if err != nil {
			return "", fmt.Errorf("find preview asset source: %w", err)
		}
		if len(matches) > 0 {
			sort.Strings(matches)
			return matches[0], nil
		}
	}
	return "", fmt.Errorf("preview asset %s@%s was not found", componentID, version)
}

func readPreviewAssetDeclarations(source string) ([]previewStoreDeclaration, error) {
	raw, err := os.ReadFile(source)
	if err != nil {
		return nil, fmt.Errorf("read preview asset %s: %w", source, err)
	}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "*") || !strings.Contains(line, "@deps") {
			continue
		}
		value := strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(strings.TrimPrefix(line, "*")), "@deps"))
		var ranges map[string]string
		if err := json.Unmarshal([]byte(value), &ranges); err != nil {
			return nil, fmt.Errorf("parse @deps in %s: %w", source, err)
		}
		out := make([]previewStoreDeclaration, 0, len(ranges))
		for name, versionRange := range ranges {
			if strings.TrimSpace(name) == "" || strings.TrimSpace(versionRange) == "" {
				continue
			}
			out = append(out, previewStoreDeclaration{Name: name, Range: versionRange})
		}
		sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
		return out, nil
	}
	return nil, nil
}
