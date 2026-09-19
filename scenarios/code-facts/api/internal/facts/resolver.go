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
	if (resolvedKind == factsv1.TargetKind_TARGET_KIND_PATH || resolvedKind == factsv1.TargetKind_TARGET_KIND_MODULE) && scenario != "" {
		resolvedKind = factsv1.TargetKind_TARGET_KIND_SCENARIO
	}
	roots := targetRoots(target, root, repoRoot)
	return &factsv1.TargetContext{
		Requested:     target,
		ResolvedKind:  resolvedKind,
		RootPath:      root,
		RootPaths:     roots,
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
	case factsv1.TargetKind_TARGET_KIND_PROJECT, factsv1.TargetKind_TARGET_KIND_REPO, factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE, factsv1.TargetKind_TARGET_KIND_PACKAGE:
		repoOverride := strings.TrimSpace(target.GetRepoRoot())
		// Preserve the existing generic-project API, where callers supplied the
		// project root in path. New CLI prefixes use repo_root explicitly.
		if repoOverride == "" && target.GetKind() == factsv1.TargetKind_TARGET_KIND_PROJECT && strings.TrimSpace(target.GetPath()) != "" {
			repoOverride = target.GetPath()
		}
		repoRoot, err = resolveRepoRoot(repoOverride)
		if err != nil {
			return "", "", err
		}
		switch target.GetKind() {
		case factsv1.TargetKind_TARGET_KIND_PACKAGE:
			name := strings.TrimSpace(target.GetPackageName())
			if name == "" {
				return "", "", fmt.Errorf("target.package_name is required for package targets")
			}
			if filepath.Base(name) != name || name == "." || name == ".." {
				return "", "", fmt.Errorf("target.package_name %q must name one package directory", name)
			}
			root = filepath.Join(repoRoot, "packages", name)
		case factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE:
			root = repoRoot
		default:
			root = repoRoot
		}
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

func targetRoots(target *factsv1.CodeTarget, root, repoRoot string) []string {
	if repoRoot == "" {
		return nil
	}
	switch target.GetKind() {
	case factsv1.TargetKind_TARGET_KIND_CONTROL_PLANE:
		return existingRoots(repoRoot, filepath.Join(repoRoot, "cmd", "vrooli"), filepath.Join(repoRoot, "internal"))
	case factsv1.TargetKind_TARGET_KIND_PROJECT:
		// PROJECT is the governed Vrooli tree. Keep generated artifacts and
		// unrelated repository metadata out of the default fleet scan while
		// retaining stable roots for scenario and shared-substrate attribution.
		return existingRoots(repoRoot, filepath.Join(repoRoot, "scenarios"), filepath.Join(repoRoot, "packages"), filepath.Join(repoRoot, "cmd", "vrooli"), filepath.Join(repoRoot, "internal"))
	case factsv1.TargetKind_TARGET_KIND_PACKAGE, factsv1.TargetKind_TARGET_KIND_SCENARIO:
		return []string{root}
	default:
		return nil
	}
}

func existingRoots(repoRoot string, candidates ...string) []string {
	seen := make(map[string]struct{}, len(candidates))
	roots := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		candidate = filepath.Clean(candidate)
		if _, ok := seen[candidate]; ok {
			continue
		}
		info, err := os.Stat(candidate)
		if err == nil && info.IsDir() {
			seen[candidate] = struct{}{}
			roots = append(roots, candidate)
		}
	}
	return roots
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
