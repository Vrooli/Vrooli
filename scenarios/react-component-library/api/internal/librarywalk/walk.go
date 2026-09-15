// Package librarywalk owns filesystem traversal policy for the component
// library. Callers get one consistent exclusion policy, including quarantine.
package librarywalk

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

type Scope struct{ Assets map[string]struct{} }

// Set is the immutable input set prepared for a gate run. Files contains
// authored source/config paths and Versions contains live version roots.
// Keeping this set beside traversal policy prevents each gate from inventing
// its own corpus walk.
type Set struct {
	Files    []string
	Versions []VersionDir
}

type VersionDir struct {
	AssetID string
	Version string
	Path    string
}

type Reads uint8

const (
	ReadsAsset Reads = iota
	ReadsClosure
	ReadsCorpus
)

type Options struct {
	Skip           map[string]bool
	IncludeRetired bool
}

func FullCorpus() Scope { return Scope{} }

// Glob is the single pattern-matching seam for derived file readers. Callers
// use it instead of opening a second filesystem-discovery policy in a gate.
func Glob(pattern string) ([]string, error) { return filepath.Glob(pattern) }

// Kinds returns live asset-kind directories, including support. Generated and
// quarantined directories are excluded by the same traversal policy.
func Kinds(root string) ([]string, error) {
	entries, err := os.ReadDir(root)
	if err != nil {
		return nil, err
	}
	kinds := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() && entry.Name() != ".retired" && entry.Name() != "node_modules" && entry.Name() != "dist" {
			kinds = append(kinds, entry.Name())
		}
	}
	sort.Strings(kinds)
	return kinds, nil
}

// Sources returns authored library source files, applying the same exclusions
// and optional asset filter as Walk. Asset filters accept either the directory
// name or its lowercase dotted catalog form (for example Button or
// controls.button).
func Sources(root string, scope Scope) ([]string, error) {
	return SourcesContext(context.Background(), root, scope)
}

// SourcesContext is the cancellable source-file projection used by jobs.
func SourcesContext(ctx context.Context, root string, scope Scope) ([]string, error) {
	var paths []string
	err := WalkContext(ctx, root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() || (filepath.Ext(path) != ".ts" && filepath.Ext(path) != ".tsx" && filepath.Ext(path) != ".css") {
			return nil
		}
		if len(scope.Assets) > 0 {
			asset := filepath.Base(filepath.Dir(path))
			if filepath.Base(filepath.Dir(filepath.Dir(path))) == "versions" {
				asset = filepath.Base(filepath.Dir(filepath.Dir(filepath.Dir(path))))
			}
			matched := false
			for candidate := range scope.Assets {
				if candidate == asset || candidate == strings.ToLower(asset) || strings.HasSuffix(candidate, "."+strings.ToLower(asset)) {
					matched = true
					break
				}
			}
			if !matched {
				return nil
			}
		}
		paths = append(paths, path)
		return nil
	})
	return paths, err
}

// BuildSet prepares the shared input set for a repository or library root.
// When passed a repository root it resolves the scenario library automatically.
func BuildSet(ctx context.Context, root string, scope Scope) (Set, error) {
	return Files(ctx, root, scope, ReadsAsset)
}

// Files resolves the input set for a declared read level.
func Files(ctx context.Context, root string, scope Scope, reads Reads) (Set, error) {
	libraryRoot := root
	if filepath.Base(filepath.Clean(root)) != "library" {
		if _, err := os.Stat(filepath.Join(root, "components")); err == nil {
			libraryRoot = root
		} else {
			libraryRoot = filepath.Join(root, "scenarios", "react-component-library", "library")
		}
	}
	fileScope := scope
	if reads == ReadsCorpus {
		fileScope = FullCorpus()
	} else if reads == ReadsClosure {
		fileScope = Scope{Assets: expandAssetScope(ctx, libraryRoot, scope)}
	}
	files, err := SourcesContext(ctx, libraryRoot, fileScope)
	if err != nil {
		return Set{}, err
	}
	set := Set{Files: append([]string(nil), files...)}
	err = WalkContext(ctx, libraryRoot, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if !entry.IsDir() || entry.Name() == "versions" || filepath.Base(filepath.Dir(path)) != "versions" {
			return nil
		}
		if reads != ReadsCorpus && len(fileScope.Assets) > 0 {
			asset := filepath.Base(filepath.Dir(filepath.Dir(path)))
			matched := false
			for candidate := range fileScope.Assets {
				if candidate == asset || candidate == strings.ToLower(asset) || strings.HasSuffix(candidate, "."+strings.ToLower(asset)) {
					matched = true
					break
				}
			}
			if !matched {
				return filepath.SkipDir
			}
		}
		set.Versions = append(set.Versions, VersionDir{
			AssetID: filepath.Base(filepath.Dir(filepath.Dir(path))),
			Version: filepath.Base(path),
			Path:    path,
		})
		return filepath.SkipDir
	})
	return set, err
}

func expandAssetScope(ctx context.Context, root string, scope Scope) map[string]struct{} {
	allowed := make(map[string]struct{}, len(scope.Assets))
	for asset := range scope.Assets {
		allowed[scopeAssetName(asset)] = struct{}{}
	}
	if len(allowed) == 0 {
		return allowed
	}
	dependencies := map[string][]string{}
	_ = WalkContext(ctx, root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() || entry.Name() == "versions" || filepath.Base(filepath.Dir(path)) != "versions" {
			return nil
		}
		asset := scopeAssetName(filepath.Base(filepath.Dir(filepath.Dir(path))))
		data, readErr := os.ReadFile(filepath.Join(path, "dependencies.json"))
		if readErr == nil {
			var lock struct {
				Dependencies []struct {
					LibraryID string `json:"libraryId"`
				} `json:"dependencies"`
			}
			if json.Unmarshal(data, &lock) == nil {
				for _, dependency := range lock.Dependencies {
					if name := scopeAssetName(dependency.LibraryID); name != "" {
						dependencies[asset] = append(dependencies[asset], name)
					}
				}
			}
		}
		return filepath.SkipDir
	})
	queue := make([]string, 0, len(allowed))
	for asset := range allowed {
		queue = append(queue, asset)
	}
	for len(queue) > 0 {
		asset := queue[0]
		queue = queue[1:]
		for _, dependency := range dependencies[asset] {
			if _, ok := allowed[dependency]; ok {
				continue
			}
			allowed[dependency] = struct{}{}
			queue = append(queue, dependency)
		}
	}
	return allowed
}

func scopeAssetName(value string) string {
	value = strings.TrimSpace(value)
	if index := strings.LastIndex(value, ":"); index >= 0 {
		value = value[index+1:]
	}
	if index := strings.LastIndex(value, "."); index >= 0 {
		value = value[index+1:]
	}
	return strings.ToLower(value)
}

// Walk traverses a library-related tree while excluding generated, dependency,
// and retired trees. The callback is intentionally compatible with WalkDir so
// existing gate logic remains small while traversal policy stays centralized.
func Walk(root string, fn func(string, os.DirEntry, error) error) error {
	return WalkContext(context.Background(), root, fn)
}

// WalkContext is Walk with cancellation checked before each callback and
// before descending into a directory. Heavy catalog jobs use this seam so a
// disconnected RPC does not continue traversing the repository.
func WalkContext(ctx context.Context, root string, fn func(string, os.DirEntry, error) error) error {
	return WalkTreeWithOptions(ctx, root, Options{}, fn)
}

// WalkTree is the shared cancellable tree traversal for API readers using the
// standard exclusion policy.
func WalkTree(ctx context.Context, root string, fn func(string, os.DirEntry, error) error) error {
	return WalkTreeWithOptions(ctx, root, Options{}, fn)
}

// WalkTreeWithOptions is the shared cancellable tree traversal for API readers. Callers
// may add explicit skips for non-library trees, but cannot accidentally lose
// the default dependency, build, or retired-tree exclusions.
func WalkTreeWithOptions(ctx context.Context, root string, options Options, fn func(string, os.DirEntry, error) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return fn(path, entry, err)
		}
		if entry.IsDir() {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return ctxErr
			}
			switch entry.Name() {
			case ".retired":
				if options.IncludeRetired {
					break
				}
				if path != root {
					return filepath.SkipDir
				}
			case "node_modules", "dist":
				if path != root {
					return filepath.SkipDir
				}
			}
			if options.Skip != nil && options.Skip[entry.Name()] && path != root {
				return filepath.SkipDir
			}
		}
		return fn(path, entry, nil)
	})
}

// WalkInfo adapts the legacy filepath.Walk callback shape to the centralized
// traversal policy. It exists for readers that need os.FileInfo metadata.
func WalkInfo(ctx context.Context, root string, fn func(string, os.FileInfo, error) error) error {
	if ctx == nil {
		ctx = context.Background()
	}
	return WalkTree(ctx, root, func(path string, entry os.DirEntry, err error) error {
		if ctxErr := ctx.Err(); ctxErr != nil {
			return ctxErr
		}
		if err != nil {
			return fn(path, nil, err)
		}
		if entry.IsDir() {
			switch entry.Name() {
			case ".retired", "node_modules", "dist":
				if path != root {
					return filepath.SkipDir
				}
			}
		}
		info, infoErr := entry.Info()
		if infoErr != nil {
			return fn(path, nil, infoErr)
		}
		return fn(path, info, nil)
	})
}
