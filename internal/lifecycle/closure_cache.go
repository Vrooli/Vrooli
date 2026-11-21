package lifecycle

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/tuning"
)

const closureCacheVersion = 3

// closureCache is the durable dependency-closure record for one Go component.
// It lives beside the component's build output, not in a shared global cache,
// so two scenarios cannot invalidate or consume one another's inputs.
type closureCache struct {
	Version      int      `json:"version"`
	Key          string   `json:"key"`
	Inputs       []string `json:"inputs"`
	Files        []string `json:"files"`
	EmbedWatches []string `json:"embed_watches,omitempty"`
	Toolchain    string   `json:"toolchain"`
}

type sourceFingerprint struct {
	digest string
	err    error
}

type closureFingerprinter struct {
	readFile func(string) ([]byte, error)
	files    map[string]sourceFingerprint
}

func newClosureFingerprinter(deps hostProbeDeps) *closureFingerprinter {
	readFile := deps.readFile
	if readFile == nil {
		readFile = os.ReadFile
	}
	return &closureFingerprinter{readFile: readFile, files: make(map[string]sourceFingerprint)}
}

func (f *closureFingerprinter) digest(path string) (string, error) {
	path = filepath.Clean(path)
	if cached, ok := f.files[path]; ok {
		return cached.digest, cached.err
	}
	data, err := f.readFile(path)
	result := sourceFingerprint{err: err}
	if err == nil {
		result.digest = fmt.Sprintf("%x", sha256.Sum256(data))
	}
	f.files[path] = result
	return result.digest, result.err
}

func closureCachePath(dir string) string {
	return filepath.Join(dir, ".vrooli-closure-go_module.json")
}

func closureCacheKey(dir, repoRoot, toolchain string, inputs []string, deps hostProbeDeps) string {
	h := sha256.New()
	fingerprints := newClosureFingerprinter(deps)
	sourceDigest, sourceTreeComplete := closureSourceDigest(dir, fingerprints, deps)
	fmt.Fprintf(h, "component:%s\nrepository:%s\nsource:%s\n", filepath.Clean(dir), filepath.Clean(repoRoot), sourceDigest)
	addFile := func(label, path string) {
		digest, err := fingerprints.digest(path)
		if err != nil {
			fmt.Fprintf(h, "%s:error:%v\n", label, err)
			return
		}
		fmt.Fprintf(h, "%s:%s\n", label, digest)
	}
	addFile("go.mod", filepath.Join(dir, "go.mod"))
	addFile("go.sum", filepath.Join(dir, "go.sum"))
	root := filepath.Clean(repoRoot)
	paths := append([]string(nil), inputs...)
	sort.Strings(paths)
	for _, input := range paths {
		clean := filepath.Clean(filepath.FromSlash(input))
		if filepath.IsAbs(clean) || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
			continue
		}
		path := filepath.Join(root, clean)
		if !pathUnderRoot(root, path) {
			continue
		}
		info, err := os.Stat(path)
		if err != nil {
			fmt.Fprintf(h, "input:%s:error:%v\n", clean, err)
			continue
		}
		if info.IsDir() {
			if sourceTreeComplete && pathUnderRoot(filepath.Clean(dir), path) {
				// The component digest covers compiler sources under its module root,
				// including this package directory. A second walk adds no freshness.
				fmt.Fprintf(h, "module-package:%s\n", clean)
				continue
			}
			fmt.Fprintf(h, "package:%s:%s\n", clean, goPackageSourceDigest(path, fingerprints, deps))
			continue
		}
		addFile("input:"+clean, path)
	}
	fmt.Fprintf(h, "toolchain:%s\n", strings.TrimSpace(toolchain))
	return fmt.Sprintf("%x", h.Sum(nil))
}

// goPackageSourceDigest fingerprints build-source files in the resolved package
// directory. Walking descendants of a local replace (especially the repo
// root) makes an unrelated scenario edit invalidate every consumer's closure.
func goPackageSourceDigest(root string, fingerprints *closureFingerprinter, deps hostProbeDeps) string {
	walkDir := deps.walkDir
	if walkDir == nil {
		walkDir = filepath.WalkDir
	}
	records := []string{}
	err := walkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			records = append(records, fmt.Sprintf("walk-error:%s:%v", filepath.ToSlash(path), walkErr))
			return nil
		}
		if entry.IsDir() {
			if path != root {
				return fs.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !isGoBuildSource(name) || strings.HasSuffix(name, "_test.go") {
			return nil
		}
		digest, err := fingerprints.digest(path)
		if err != nil {
			records = append(records, fmt.Sprintf("read-error:%s:%v", filepath.ToSlash(path), err))
			return nil
		}
		records = append(records, fmt.Sprintf("%s:%s", name, digest))
		return nil
	})
	if err != nil {
		records = append(records, fmt.Sprintf("walk-error:%s:%v", filepath.ToSlash(root), err))
	}
	sort.Strings(records)
	h := sha256.New()
	for _, record := range records {
		fmt.Fprintln(h, record)
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// closureSourceDigest fingerprints compiler source under one component module.
// Besides Go files that can change the import closure, non-test compiler inputs
// must invalidate freshness when added, removed, or changed. Module files alone
// are insufficient: moving an import between internal packages leaves
// go.mod/go.sum unchanged. The digest is content-based, independent of mtimes.
func closureSourceDigest(root string, fingerprints *closureFingerprinter, deps hostProbeDeps) (string, bool) {
	walkDir := deps.walkDir
	if walkDir == nil {
		walkDir = filepath.WalkDir
	}

	records := []string{}
	complete := true
	err := walkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			complete = false
			records = append(records, fmt.Sprintf("walk-error:%s:%v", filepath.ToSlash(path), walkErr))
			return nil
		}
		if entry.IsDir() {
			if entry.Name() == ".git" {
				return fs.SkipDir
			}
			return nil
		}
		name := entry.Name()
		if !isGoBuildSource(name) {
			return nil
		}
		digest, err := fingerprints.digest(path)
		if err != nil {
			records = append(records, fmt.Sprintf("read-error:%s:%v", filepath.ToSlash(path), err))
			return nil
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			records = append(records, fmt.Sprintf("rel-error:%s:%v", filepath.ToSlash(path), err))
			return nil
		}
		records = append(records, fmt.Sprintf("%s:%s", filepath.ToSlash(relative), digest))
		return nil
	})
	if err != nil {
		complete = false
		records = append(records, fmt.Sprintf("walk-error:%s:%v", filepath.ToSlash(root), err))
	}
	sort.Strings(records)
	h := sha256.New()
	for _, record := range records {
		fmt.Fprintln(h, record)
	}
	return fmt.Sprintf("%x", h.Sum(nil)), complete
}

func readClosureCache(path, toolchain string, read func(string) ([]byte, error)) (closureCache, bool) {
	if read == nil {
		read = os.ReadFile
	}
	data, err := read(path)
	if err != nil {
		return closureCache{}, false
	}
	var cached closureCache
	if json.Unmarshal(data, &cached) != nil || cached.Version != closureCacheVersion || cached.Toolchain != toolchain || len(cached.Inputs) == 0 || len(cached.Files) == 0 {
		return closureCache{}, false
	}
	cached.Inputs = append([]string(nil), cached.Inputs...)
	cached.Files = append([]string(nil), cached.Files...)
	return cached, true
}

func writeClosureCache(path string, cached closureCache) error {
	data, err := json.MarshalIndent(cached, "", "  ")
	if err != nil {
		return err
	}
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".vrooli-closure-go_module-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(tuning.PermFile); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

// Embed patterns can acquire new files without changing a Go source. Watch the
// matched trees for cache refresh; only go-list's actual EmbedFiles enter the
// build manifest (so hidden or otherwise excluded assets are not build inputs).
func resolvedClosureCacheKey(dir, root, toolchain string, cached closureCache, deps hostProbeDeps) string {
	h := sha256.New()
	fmt.Fprintln(h, closureCacheKey(dir, root, toolchain, cached.Inputs, deps))
	for _, pattern := range cached.EmbedWatches {
		matches, err := filepath.Glob(pattern)
		fmt.Fprintf(h, "pattern:%s:error:%v\n", pattern, err)
		for _, match := range matches {
			err := filepath.WalkDir(match, func(path string, entry fs.DirEntry, walkErr error) error {
				if walkErr != nil {
					return walkErr
				}
				if entry.IsDir() {
					return nil
				}
				data, err := os.ReadFile(path)
				if err != nil {
					return err
				}
				fmt.Fprintf(h, "%s:%x\n", path, sha256.Sum256(data))
				return nil
			})
			fmt.Fprintf(h, "walk:%v\n", err)
		}
	}
	return fmt.Sprintf("%x", h.Sum(nil))
}

// New C/assembly/swig files can change go-list's compiled file set too.
func isGoBuildSource(name string) bool {
	return buildinfo.IsGoBuildSource(name)
}
