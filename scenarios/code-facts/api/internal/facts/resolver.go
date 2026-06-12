package facts

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
	factsv1 "github.com/vrooli/vrooli/packages/proto/gen/go/code-facts/v1/facts"
)

const resolverAnalyzer = "code-facts.target-resolver"

func resolveTarget(target *factsv1.CodeTarget) (*factsv1.TargetContext, error) {
	root, repoRoot, err := targetRoot(target)
	if err != nil {
		return nil, err
	}

	scenario := strings.TrimSpace(target.GetScenario())
	if scenario == "" {
		scenario = scenarioFromPath(root, repoRoot)
	}

	resolvedKind := target.GetKind()
	if scenario != "" {
		resolvedKind = factsv1.TargetKind_TARGET_KIND_SCENARIO
	}
	return &factsv1.TargetContext{
		Requested:     target,
		ResolvedKind:  resolvedKind,
		RootPath:      root,
		Scenario:      scenario,
		ScenarioAware: scenario != "",
	}, nil
}

func targetRoot(target *factsv1.CodeTarget) (root string, repoRoot string, err error) {
	switch target.GetKind() {
	case factsv1.TargetKind_TARGET_KIND_SCENARIO:
		repoRoot, err = resolveRepoRoot(target.GetRepoRoot())
		if err != nil {
			return "", "", err
		}
		root = filepath.Join(repoRoot, "scenarios", target.GetScenario())
	default:
		root = target.GetPath()
		if !filepath.IsAbs(root) {
			root, err = filepath.Abs(root)
			if err != nil {
				return "", "", fmt.Errorf("resolve target path: %w", err)
			}
		}
		if repoRoot, err = resolveRepoRoot(target.GetRepoRoot()); err != nil {
			repoRoot, _ = repocontract.FindRepoRootFromPath(root)
		}
	}
	root = filepath.Clean(root)
	if _, err := os.Stat(root); err != nil {
		return "", "", fmt.Errorf("target root %q: %w", root, err)
	}
	return root, repoRoot, nil
}

func resolveRepoRoot(override string) (string, error) {
	if strings.TrimSpace(override) != "" {
		root := override
		if !filepath.IsAbs(root) {
			abs, err := filepath.Abs(root)
			if err != nil {
				return "", err
			}
			root = abs
		}
		return filepath.Clean(root), nil
	}
	return repocontract.ResolveRepoRoot()
}

func scenarioFromPath(path, repoRoot string) string {
	path = filepath.Clean(path)
	if repoRoot != "" {
		scenariosDir := filepath.Join(filepath.Clean(repoRoot), "scenarios")
		if rel, err := filepath.Rel(scenariosDir, path); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			parts := strings.Split(filepath.ToSlash(rel), "/")
			if len(parts) > 0 && hasServiceManifest(filepath.Join(scenariosDir, parts[0])) {
				return parts[0]
			}
		}
	}

	dir := path
	if info, err := os.Stat(dir); err == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}
	for {
		if filepath.Base(filepath.Dir(dir)) == "scenarios" && hasServiceManifest(dir) {
			return filepath.Base(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

func hasServiceManifest(root string) bool {
	info, err := os.Stat(filepath.Join(root, ".vrooli", "service.json"))
	return err == nil && !info.IsDir()
}

func evidence(status factsv1.EvidenceStatus, message string, files ...string) *factsv1.Evidence {
	ev := &factsv1.Evidence{
		Status:     status,
		Confidence: 1,
		Analyzer:   resolverAnalyzer,
		Message:    message,
	}
	if len(files) > 0 && files[0] != "" {
		ev.Range = &factsv1.SourceRange{File: files[0]}
	}
	return ev
}
