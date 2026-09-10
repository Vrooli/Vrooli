package lifecycle

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/vrooli/vrooli/internal/scenario"
)

const buildIdentityEnv = "VROOLI_BUILD_IDENTITY"

var buildIdentityIgnoredDirectories = map[string]struct{}{
	".git": {}, ".vrooli": {}, ".cache": {}, "coverage": {}, "data": {},
	"dist": {}, "logs": {}, "node_modules": {}, "tmp": {}, ".vite": {},
}

// scenarioBuildIdentity hashes authored scenario inputs in a stable order.
// Generated outputs and runtime state are excluded so a health identity tracks
// the source that a managed build was asked to serve.
func scenarioBuildIdentity(item scenario.Scenario) (string, error) {
	root := filepath.Clean(item.Path)
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("stat scenario source: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("scenario source %s is not a directory", root)
	}

	hash := sha256.New()
	generatedOutputs, err := scenarioGeneratedOutputs(item)
	if err != nil {
		return "", err
	}
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path != root && entry.IsDir() {
			if _, ignored := buildIdentityIgnoredDirectories[entry.Name()]; ignored {
				return filepath.SkipDir
			}
			relative, relativeErr := filepath.Rel(root, path)
			if relativeErr != nil {
				return relativeErr
			}
			if generatedOutputs.contains(filepath.ToSlash(relative)) {
				return filepath.SkipDir
			}
			return nil
		}
		if path == root || !entry.Type().IsRegular() {
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		if generatedOutputs.contains(filepath.ToSlash(relative)) {
			return nil
		}
		if generatedIdentityMetadata(filepath.Base(relative)) {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		mode := entry.Type().String()
		_, _ = fmt.Fprintf(hash, "%s\x00%s\x00%d\x00", filepath.ToSlash(relative), mode, len(data))
		_, _ = hash.Write(data)
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("hash scenario source %s: %w", root, err)
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

func generatedIdentityMetadata(name string) bool {
	return strings.HasPrefix(name, ".vrooli-") || strings.HasSuffix(name, ".freshness.json") || name == "coverage.out"
}

type generatedOutputSet map[string]struct{}

func (s generatedOutputSet) contains(path string) bool {
	path = filepath.ToSlash(filepath.Clean(path))
	for output := range s {
		if path == output || strings.HasPrefix(path, output+"/") {
			return true
		}
	}
	return false
}

// scenarioGeneratedOutputs derives generated artifacts from the same
// component build contract used by lifecycle startup. Runtime identity is
// computed before a build and checked after it, so declared outputs must not
// turn a successful build into an apparently stale runtime.
func scenarioGeneratedOutputs(item scenario.Scenario) (generatedOutputSet, error) {
	outputs := generatedOutputSet{}
	scenarioRoot := filepath.Clean(item.Path)
	scenarioName := strings.TrimSpace(item.Slug)
	if scenarioName == "" {
		scenarioName = filepath.Base(scenarioRoot)
	}
	for name, component := range item.Manifest.Components {
		spec, ok := builderRegistry[strings.TrimSpace(component.Build.Kind)]
		if !ok || spec.Reserved {
			continue
		}
		targets, err := componentBuildTargets(name, scenarioRoot, scenarioName, component, spec, "")
		if err != nil {
			return nil, fmt.Errorf("derive generated output for component %q: %w", name, err)
		}
		for _, target := range targets {
			relative, err := filepath.Rel(scenarioRoot, target.Output)
			if err != nil || filepath.IsAbs(relative) || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
				continue
			}
			outputs[filepath.ToSlash(filepath.Clean(relative))] = struct{}{}
		}
	}
	return outputs, nil
}

func buildIdentityMatches(expected, served string) bool {
	expected = strings.TrimSpace(expected)
	return expected == "" || expected == strings.TrimSpace(served)
}
