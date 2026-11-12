package lifecycle

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// This validator closes the owner-declared production package graph. It is not
// a JavaScript analyser or sandbox: undeclared imports, absolute paths, dynamic
// loading and native-library dependencies remain the runtime owner's contract.
// The caller must pin Node, disable global search paths and reject NODE_PATH and
// NODE_OPTIONS overrides. Ancestor node_modules are checked at verification
// time because --no-global-search-paths does not disable them. This is not a
// sandbox: later mutation of those ancestors remains outside this check.
// No package manager, Node process or package code runs.
func validateRetainedNodeRuntime(root, cwd, script string) error {
	root, err := filepath.Abs(root)
	if err != nil {
		return err
	}
	root, err = filepath.EvalSymlinks(root)
	if err != nil {
		return fmt.Errorf("retained Node root: %w", err)
	}
	v := retainedNodeValidator{root: root, visited: map[string]bool{}}
	if _, err = v.realWithin(cwd); err != nil {
		return fmt.Errorf("retained Node working directory: %w", err)
	}
	script, err = v.realWithin(script)
	if err != nil {
		return fmt.Errorf("retained Node entry: %w", err)
	}
	switch filepath.Ext(script) {
	case ".js", ".mjs", ".cjs":
	default:
		return fmt.Errorf("retained Node entry requires a .js, .mjs or .cjs file")
	}
	info, err := os.Stat(script)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() {
		return fmt.Errorf("retained Node entry is not a regular file")
	}
	// Node interprets an entry relative to its nearest package scope, not cwd.
	for dir := filepath.Dir(script); ; dir = filepath.Dir(dir) {
		path := filepath.Join(dir, "package.json")
		_, err := os.Lstat(path)
		if err == nil {
			return v.visit(dir, true)
		}
		if !os.IsNotExist(err) {
			return err
		}
		if dir == root || filepath.Base(dir) == "node_modules" {
			break
		}
	}
	return fmt.Errorf("retained Node entry has no package.json inside retained runtime")
}

const retainedNodeMaxPackages = 4096
const retainedNodeMaxEdges = 32768
const retainedNodeManifestLimit = 1024 * 1024

var retainedNodePackageName = regexp.MustCompile(`^(?:@[a-zA-Z0-9_~][a-zA-Z0-9._~-]*/)?[a-zA-Z0-9_~][a-zA-Z0-9._~-]*$`)
var errRetainedNodePackageAbsent = errors.New("package absent from retained runtime")

type retainedNodePackage struct {
	Main                 string            `json:"main"`
	Dependencies         map[string]string `json:"dependencies"`
	OptionalDependencies map[string]string `json:"optionalDependencies"`
	PeerDependencies     map[string]string `json:"peerDependencies"`
	PeerDependenciesMeta map[string]struct {
		Optional bool `json:"optional"`
	} `json:"peerDependenciesMeta"`
}
type retainedNodeValidator struct {
	root    string
	visited map[string]bool
	edges   int
}

func (v *retainedNodeValidator) within(path string) bool {
	rel, err := filepath.Rel(v.root, path)
	return err == nil && !filepath.IsAbs(rel) && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}
func (v *retainedNodeValidator) realWithin(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	if !v.within(path) {
		return "", fmt.Errorf("path escapes retained runtime: %s", path)
	}
	real, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if !v.within(real) {
		return "", fmt.Errorf("resolved path escapes retained runtime: %s", path)
	}
	return real, nil
}
func (v *retainedNodeValidator) readPackage(dir string) (retainedNodePackage, error) {
	var p retainedNodePackage
	path, err := v.realWithin(filepath.Join(dir, "package.json"))
	if err != nil {
		return p, fmt.Errorf("retained Node package manifest: %w", err)
	}
	pre, err := os.Stat(path)
	if err != nil {
		return p, err
	}
	if !pre.Mode().IsRegular() || pre.Size() > retainedNodeManifestLimit {
		return p, fmt.Errorf("unsupported retained Node package manifest: %s", path)
	}
	f, err := os.Open(path)
	if err != nil {
		return p, err
	}
	defer f.Close()
	st, err := f.Stat()
	if err != nil {
		return p, err
	}
	if !st.Mode().IsRegular() || st.Size() > retainedNodeManifestLimit {
		return p, fmt.Errorf("unsupported retained Node package manifest: %s", path)
	}
	data, err := io.ReadAll(io.LimitReader(f, retainedNodeManifestLimit+1))
	if err != nil {
		return p, err
	}
	if len(data) > retainedNodeManifestLimit {
		return p, fmt.Errorf("retained Node package manifest exceeds size limit")
	}
	if err = json.Unmarshal(data, &p); err != nil {
		return p, fmt.Errorf("invalid retained Node package %s: %w", dir, err)
	}
	if strings.TrimSpace(string(data)) == "null" {
		return p, fmt.Errorf("retained Node package manifest must be an object")
	}
	return p, nil
}
func (v *retainedNodeValidator) resolve(from, name string) (string, error) {
	if !retainedNodePackageName.MatchString(name) {
		return "", fmt.Errorf("unsupported retained Node package name %q", name)
	}
	// Node's normal module search continues beyond the retained root even with
	// --no-global-search-paths. A package directory with no loadable entry may
	// still fall through. Conservatively reject every external candidate even
	// when an internal directory exists; do not pretend to implement Node's full
	// main/index/exports resolution algorithm.
	for dir := filepath.Dir(v.root); dir != v.root; dir = filepath.Dir(dir) {
		if filepath.Base(dir) != "node_modules" {
			candidate := filepath.Join(dir, "node_modules", filepath.FromSlash(name))
			present, err := retainedNodeCandidatePresent(candidate)
			if err != nil {
				return "", fmt.Errorf("external retained Node ancestor candidate: %w", err)
			}
			if present {
				return "", fmt.Errorf("declared Node package %q has an external ancestor candidate outside retained runtime: %s", name, candidate)
			}
		}
		if parent := filepath.Dir(dir); parent == dir {
			break
		}
	}
	for dir := from; ; dir = filepath.Dir(dir) {
		// Node does not search node_modules/node_modules at a node_modules ancestor.
		if filepath.Base(dir) != "node_modules" {
			candidate := filepath.Join(dir, "node_modules", filepath.FromSlash(name))
			present, err := retainedNodeCandidatePresent(candidate)
			if err != nil {
				return "", err
			}
			if present {
				real, err := v.realWithin(candidate)
				if err != nil {
					return "", err
				}
				st, err := os.Stat(real)
				if err != nil {
					return "", err
				}
				if !st.IsDir() {
					return "", fmt.Errorf("retained Node package %q is not a directory", name)
				}
				return real, nil
			}
		}
		if dir == v.root {
			break
		}
	}

	return "", fmt.Errorf("%w: %s (from %s); external/global fallback is unsupported", errRetainedNodePackageAbsent, name, from)
}

// The supported closure consists of package directories with manifests. CJS
// also tries loose .js/.json/.node files before a same-named directory. Reject
// those candidates rather than validating a directory Node would not load, or
// treating a resolvable optional module as absent.
func retainedNodeCandidatePresent(candidate string) (bool, error) {
	_, candidateErr := os.Lstat(candidate)
	if candidateErr != nil && !os.IsNotExist(candidateErr) {
		return false, candidateErr
	}
	for _, extension := range []string{".js", ".json", ".node"} {
		path := candidate + extension
		if _, err := os.Lstat(path); err == nil {
			return false, fmt.Errorf("unsupported retained Node loose-file candidate: %s", path)
		} else if !os.IsNotExist(err) {
			return false, err
		}
	}
	return candidateErr == nil, nil
}

// Check the common CJS entry candidates without claiming full JS resolution.
// A missing literal main can load an extension or index candidate instead, so
// each candidate and its existing path prefixes must remain in the closure.
func (v *retainedNodeValidator) validatePackageEntryPaths(dir, main string) error {
	candidates := []string{}
	if main != "" {
		if filepath.IsAbs(main) || strings.ContainsAny(main, "\\\x00") {
			return fmt.Errorf("unsupported retained Node package main %q", main)
		}
		target := filepath.Join(dir, filepath.FromSlash(main))
		candidates = append(candidates, target)
		info, statErr := os.Stat(target)
		for _, extension := range []string{".js", ".json", ".node"} {
			candidates = append(candidates, target+extension)
			if statErr != nil || info.IsDir() {
				candidates = append(candidates, filepath.Join(target, "index"+extension))
			}
		}
	}
	// Invalid/missing main also allows the package's own index fallback.
	for _, extension := range []string{".js", ".json", ".node"} {
		candidates = append(candidates, filepath.Join(dir, "index"+extension))
	}
	for _, candidate := range candidates {
		if !v.within(candidate) {
			return fmt.Errorf("retained Node package main/index escapes retained runtime: %s", candidate)
		}
		for path := candidate; ; path = filepath.Dir(path) {
			if _, err := os.Lstat(path); err == nil {
				if _, err = v.realWithin(path); err != nil {
					return fmt.Errorf("retained Node package main/index: %w", err)
				}
				break
			} else if !os.IsNotExist(err) {
				return fmt.Errorf("retained Node package main/index: %w", err)
			}
			if path == v.root {
				break
			}
		}
	}
	return nil
}

func (v *retainedNodeValidator) visit(dir string, entry bool) error {
	real, err := v.realWithin(dir)
	if err != nil {
		return err
	}
	if v.visited[real] {
		return nil
	}
	if len(v.visited) >= retainedNodeMaxPackages {
		return fmt.Errorf("retained Node package graph exceeds package limit")
	}
	p, err := v.readPackage(real)
	if err != nil {
		return err
	}
	if entry && p.Dependencies == nil {
		return fmt.Errorf("retained Node entry package requires an explicit dependencies object (empty is allowed)")
	}
	if err := v.validatePackageEntryPaths(real, p.Main); err != nil {
		return err
	}
	v.visited[real] = true
	// npm's optionalDependencies override dependencies. A peer is required
	// unless its metadata explicitly marks it optional. Optional peer metadata
	// cannot weaken an independent dependency requirement; present optional
	// packages still undergo the same closure and path validation as required ones.
	required := map[string]bool{}
	for name := range p.Dependencies {
		required[name] = true
	}
	for name := range p.OptionalDependencies {
		required[name] = false
	}
	for name := range p.PeerDependencies {
		required[name] = required[name] || !p.PeerDependenciesMeta[name].Optional
	}
	names := make([]string, 0, len(required))
	for name := range required {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		v.edges++
		if v.edges > retainedNodeMaxEdges {
			return fmt.Errorf("retained Node package graph exceeds dependency limit")
		}
		target, err := v.resolve(real, name)
		if err != nil {
			if !required[name] && errors.Is(err, errRetainedNodePackageAbsent) {
				continue
			}
			return err
		}
		if err = v.visit(target, false); err != nil {
			return fmt.Errorf("retained Node dependency %s: %w", name, err)
		}
	}
	return nil
}
