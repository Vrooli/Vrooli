package lifecycle

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/scenario"
)

// builderFreshnessContext derives freshness artifacts from one registry row.
// The row supplies the file inputs, closure resolver, key resolver, and output
// contract. Builder-specific freshness verdict functions must not be added.
func builderFreshnessContext(ctx context.Context, appRoot, repoRoot, scenarioName, componentName string, component scenario.Component, spec BuilderSpec, deps hostProbeDeps) ([]artifactFreshness, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	root := repoRoot
	if strings.TrimSpace(root) == "" {
		root = appRoot
	}
	buildDir := resolveCheckPath(appRoot, component.Build.Dir)
	if strings.TrimSpace(component.Build.Dir) == "" {
		return nil, fmt.Errorf("component %q build.dir is required", componentName)
	}
	targets, err := componentBuildTargets(componentName, appRoot, scenarioName, component, spec, runtimeGOOS())
	if err != nil {
		return nil, err
	}

	inputs, err := builderFreshnessInputs(ctx, root, buildDir, spec, component, deps)
	if err != nil {
		return nil, err
	}
	keys := builderFreshnessKeys(spec, deps)
	if spec.FollowsWorkspaceFileDeps {
		for key, value := range sharedPackageFreshnessKeyInputs(repoRoot, buildDir, &deps) {
			keys[key] = value
		}
	}

	artifacts := make([]artifactFreshness, 0, len(targets))
	for _, target := range targets {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		targetName, err := filepath.Rel(appRoot, target.Output)
		if err != nil {
			return nil, err
		}
		targetName = filepath.ToSlash(targetName)
		skipSuffixes := append([]string(nil), spec.SkipSuffixes...)
		skipSuffixes = append(skipSuffixes, cliutil.FreshnessManifestSuffix)
		skipFiles := []string(nil)
		if spec.ClosureResolver == "go_list" {
			skipFiles = []string{filepath.Base(target.Output)}
		}
		artifacts = append(artifacts, artifactFreshness{
			Target:       targetName,
			ArtifactPath: target.Output,
			ManifestPath: cliutil.FreshnessManifestPath(target.Output),
			CheckType:    component.Build.Kind,
			KeyInputs:    cloneStringMap(keys),
			Spec: cliutil.FreshnessSpec{
				SourceRoot:      buildDir,
				ContextRoot:     root,
				Inputs:          inputs,
				SkipFiles:       skipFiles,
				SkipSuffixes:    skipSuffixes,
				CaseInsensitive: deps.volumeCaseInsensitive(target.Output),
			},
		})
	}
	return artifacts, nil
}

func builderFreshnessInputs(ctx context.Context, root, buildDir string, spec BuilderSpec, component scenario.Component, deps hostProbeDeps) ([]string, error) {
	if spec.ClosureResolver == "go_list" {
		if inputs, ok := goListFreshnessInputsContext(ctx, buildDir, root, deps); ok {
			return inputs, nil
		}
		return binariesFreshnessInputsContext(ctx, buildDir, root, deps)
	}
	inputs := make([]string, 0, len(spec.Inputs))
	for _, pattern := range spec.Inputs {
		pattern = strings.TrimSpace(pattern)
		if pattern == "" {
			continue
		}
		pattern = strings.ReplaceAll(pattern, "{src}", filepath.ToSlash(filepath.Join(buildDir, "src")))
		pattern = strings.ReplaceAll(pattern, "{dir}", filepath.ToSlash(buildDir))
		if !filepath.IsAbs(pattern) {
			pattern = filepath.ToSlash(filepath.Join(buildDir, filepath.FromSlash(pattern)))
		}
		inputs = append(inputs, relUnder(root, pattern))
	}
	if len(inputs) == 0 {
		inputs = []string{relUnder(root, buildDir)}
	}
	if spec.FollowsWorkspaceFileDeps {
		sourceDir := filepath.Join(buildDir, "src")
		if dependencySpecs, err := fileDependenciesWithDeps(filepath.Join(buildDir, "package.json"), deps); err == nil {
			for _, dependency := range dependencySpecs {
				resolved := resolveCheckPath(buildDir, strings.TrimPrefix(dependency.Spec, "file:"))
				dependencyInputs, dependencyErr := uiFileDependencyFreshnessInputsContext(ctx, root, sourceDir, dependency.Name, resolved, deps)
				if dependencyErr != nil {
					return nil, dependencyErr
				}
				inputs = append(inputs, dependencyInputs...)
			}
		}
	}
	return inputs, nil
}

func builderFreshnessKeys(spec BuilderSpec, deps hostProbeDeps) map[string]string {
	switch spec.KeyResolver {
	case "go_env":
		return goBuildKeyInputs(deps, strings.Join(spec.Build, " "))
	case "node":
		return uiBuildKeyInputs(deps)
	case "python":
		return pythonBuildKeyInputs(deps)
	default:
		return map[string]string{}
	}
}
