package lifecycle

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
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
	root := filepath.Clean(item.SourcePath())
	info, err := os.Stat(root)
	if err != nil {
		return "", fmt.Errorf("stat scenario source: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("scenario source %s is not a directory", root)
	}

	hash := sha256.New()
	excludedPaths, err := buildIdentityExcludedPaths(item.Manifest.BuildIdentity.ExcludePaths)
	if err != nil {
		return "", err
	}
	runtimeDocs, err := buildIdentityRuntimeDocs(item.Manifest.BuildIdentity.RuntimeDocs, excludedPaths)
	if err != nil {
		return "", err
	}
	// service.json is authored lifecycle/build policy, even though other files
	// under .vrooli are generated runtime state and remain excluded.
	manifestPath := filepath.Join(root, ".vrooli", "service.json")
	if data, readErr := os.ReadFile(manifestPath); readErr == nil {
		_, _ = fmt.Fprintf(hash, "%s\x00%s\x00%d\x00", ".vrooli/service.json", "-rw-r--r--", len(data))
		_, _ = hash.Write(data)
	} else if !os.IsNotExist(readErr) {
		return "", fmt.Errorf("read scenario service manifest for build identity: %w", readErr)
	}
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
			if filepath.ToSlash(relative) == "docs" {
				return filepath.SkipDir
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
		relativePath := filepath.ToSlash(relative)
		if excludedPaths.contains(relativePath) || generatedIdentityMetadata(filepath.Base(relative)) {
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
	for _, relative := range runtimeDocs {
		fullPath := filepath.Join(root, filepath.FromSlash(relative))
		if _, err := os.Lstat(fullPath); err != nil {
			return "", fmt.Errorf("read declared runtime documentation %q: %w", relative, err)
		}
		err := filepath.WalkDir(fullPath, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if !entry.Type().IsRegular() {
				return fmt.Errorf("declared runtime documentation %q contains non-regular file %q", relative, path)
			}
			fileRelative, err := filepath.Rel(root, path)
			if err != nil {
				return err
			}
			fileRelative = filepath.ToSlash(fileRelative)
			if excludedPaths.contains(fileRelative) {
				return nil
			}
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			_, _ = fmt.Fprintf(hash, "%s\x00%s\x00%d\x00", fileRelative, entry.Type().String(), len(data))
			_, _ = hash.Write(data)
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("hash declared runtime documentation %q: %w", relative, err)
		}
	}
	return fmt.Sprintf("sha256:%x", hash.Sum(nil)), nil
}

type buildIdentityExclusions map[string]struct{}

func buildIdentityExcludedPaths(paths []string) (buildIdentityExclusions, error) {
	excluded := make(buildIdentityExclusions, len(paths))
	for _, candidate := range paths {
		clean := filepath.ToSlash(filepath.Clean(candidate))
		if candidate == "" || filepath.IsAbs(candidate) || clean == "." || clean == ".." || strings.HasPrefix(clean, "../") || clean != candidate || strings.ContainsAny(candidate, "*?[]\\") || clean == ".vrooli" || strings.HasPrefix(clean, ".vrooli/") {
			return nil, fmt.Errorf("invalid build identity exclusion path %q: expected an exact repo-relative path outside .vrooli", candidate)
		}
		excluded[clean] = struct{}{}
	}
	return excluded, nil
}

func (e buildIdentityExclusions) contains(relative string) bool {
	for candidate := range e {
		if relative == candidate || strings.HasPrefix(relative, candidate+"/") {
			return true
		}
	}
	return false
}

func buildIdentityRuntimeDocs(paths []string, excluded buildIdentityExclusions) ([]string, error) {
	docs := make([]string, 0, len(paths))
	seen := make(map[string]struct{}, len(paths))
	for _, candidate := range paths {
		clean := filepath.ToSlash(filepath.Clean(candidate))
		if candidate == "" || !strings.HasPrefix(clean, "docs/") || clean == "docs" || filepath.IsAbs(candidate) || clean == ".." || strings.HasPrefix(clean, "../") || clean != candidate || strings.ContainsAny(candidate, "*?[]\\") {
			return nil, fmt.Errorf("invalid runtime documentation path %q: expected an exact repo-relative path under docs/", candidate)
		}
		if _, ok := seen[clean]; ok {
			return nil, fmt.Errorf("duplicate runtime documentation path %q", clean)
		}
		seen[clean] = struct{}{}
		for excludedPath := range excluded {
			if clean == excludedPath || strings.HasPrefix(clean, excludedPath+"/") || strings.HasPrefix(excludedPath, clean+"/") {
				return nil, fmt.Errorf("runtime documentation path %q conflicts with excluded path %q", clean, excludedPath)
			}
		}
		docs = append(docs, clean)
	}
	sort.Strings(docs)
	for index := 1; index < len(docs); index++ {
		if strings.HasPrefix(docs[index], docs[index-1]+"/") {
			return nil, fmt.Errorf("runtime documentation paths %q and %q overlap", docs[index-1], docs[index])
		}
	}
	return docs, nil
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
	scenarioRoot := filepath.Clean(item.SourcePath())
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
