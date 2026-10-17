package treedigest

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	pathpkg "path"
	"path/filepath"
	"sort"
	"strings"
	"sync"
)

const InputManifestSchemaVersion = 1

// InputSelection declares one file glob. Required selections fail closed when
// they resolve to no eligible files; optional selections contribute only when
// present.
type InputSelection struct {
	Glob     string `json:"glob"`
	Required bool   `json:"required,omitempty"`
}

// RootSpec names one independently resolved content root. Root paths are
// location metadata and never participate in equality; Name is the stable
// composition key used by consumers to declare shared dependencies.
type RootSpec struct {
	Name       string           `json:"name"`
	Path       string           `json:"path"`
	Selections []InputSelection `json:"selections"`
	// Files is an already-enumerated owner input set, mutually exclusive with
	// Selections. These literal, required paths are not re-filtered through Git
	// ignores or interpreted as globs (e.g. a UI route named [id].tsx).
	Files []string `json:"files,omitempty"`
}

// ManifestRequest describes the complete declared input closure. Attribution
// is recorded for diagnosis but deliberately excluded from Identity.
type ManifestRequest struct {
	Primary       RootSpec            `json:"primary"`
	Dependencies  []RootSpec          `json:"dependencies,omitempty"`
	Configuration map[string]string   `json:"configuration,omitempty"`
	Toolchain     map[string]string   `json:"toolchain,omitempty"`
	Attribution   ManifestAttribution `json:"attribution,omitempty"`
}

type ManifestAttribution struct {
	Commit string `json:"commit,omitempty"`
	Branch string `json:"branch,omitempty"`
	Dirty  bool   `json:"dirty,omitempty"`
}

type ManifestFile struct {
	Path   string `json:"path"`
	Digest string `json:"digest"`
	Size   int64  `json:"size"`
}

type ManifestRoot struct {
	Name     string         `json:"name"`
	Identity string         `json:"identity"`
	Files    []ManifestFile `json:"files"`
}

// InputManifest is a frozen, portable description of every byte and scalar
// that can affect one validation result. Identity excludes repository and
// filesystem attribution such as commit, branch, dirty/index state, mtime, and
// absolute checkout location.
type InputManifest struct {
	SchemaVersion int                 `json:"schema_version"`
	Identity      string              `json:"identity"`
	Roots         []ManifestRoot      `json:"roots"`
	Configuration map[string]string   `json:"configuration,omitempty"`
	Toolchain     map[string]string   `json:"toolchain,omitempty"`
	Attribution   ManifestAttribution `json:"attribution,omitempty"`
}

type manifestDeps struct {
	ctx      context.Context
	run      Runner
	readFile func(string) ([]byte, error)
	lstat    func(string) (os.FileInfo, error)
	digest   func([]byte) string
}

// DigestCache is a bounded, concurrency-safe optimization over bytes already
// read from disk. Its exact-byte key is deliberately independent of mtime,
// inode, and path metadata; a cache hit can never suppress an authoritative
// file read or stable-read check.
type DigestCache struct {
	mu         sync.Mutex
	entries    map[string]string
	bytes      int
	limitBytes int
	hits       uint64
	misses     uint64
}

type DigestCacheStats struct {
	Entries int
	Bytes   int
	Hits    uint64
	Misses  uint64
}

func NewDigestCache(limitBytes int) *DigestCache {
	if limitBytes < 0 {
		limitBytes = 0
	}
	return &DigestCache{entries: map[string]string{}, limitBytes: limitBytes}
}

func (c *DigestCache) Digest(data []byte) string {
	if c == nil || c.limitBytes == 0 || len(data) > c.limitBytes {
		return digestBytes(data)
	}
	key := string(data)
	c.mu.Lock()
	defer c.mu.Unlock()
	if digest, ok := c.entries[key]; ok {
		c.hits++
		return digest
	}
	c.misses++
	if c.bytes+len(data) > c.limitBytes {
		clear(c.entries)
		c.bytes = 0
	}
	digest := digestBytes(data)
	c.entries[key] = digest
	c.bytes += len(data)
	return digest
}

func (c *DigestCache) Stats() DigestCacheStats {
	if c == nil {
		return DigestCacheStats{}
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	return DigestCacheStats{Entries: len(c.entries), Bytes: c.bytes, Hits: c.hits, Misses: c.misses}
}

// ManifestBuilder reuses safe exact-byte digests across captures. It does not
// cache directory enumeration or manifests, because doing so could hide a
// newly created, removed, or renamed relevant input.
type ManifestBuilder struct {
	cache *DigestCache
}

func NewManifestBuilder(cacheBytes int) *ManifestBuilder {
	return &ManifestBuilder{cache: NewDigestCache(cacheBytes)}
}

func (b *ManifestBuilder) Build(req ManifestRequest) (InputManifest, error) {
	if b == nil {
		return BuildInputManifest(req)
	}
	return buildInputManifest(req, manifestDeps{run: defaultRunner, readFile: os.ReadFile, lstat: os.Lstat, digest: b.cache.Digest})
}

func (b *ManifestBuilder) BuildContext(ctx context.Context, req ManifestRequest) (InputManifest, error) {
	if b == nil {
		return BuildInputManifestContext(ctx, req)
	}
	return buildInputManifest(req, manifestDeps{ctx: ctx, run: contextRunner(ctx), readFile: os.ReadFile, lstat: os.Lstat, digest: b.cache.Digest})
}

func (b *ManifestBuilder) CacheStats() DigestCacheStats {
	if b == nil {
		return DigestCacheStats{}
	}
	return b.cache.Stats()
}

// BuildInputManifest resolves and freezes the request against current bytes.
func BuildInputManifest(req ManifestRequest) (InputManifest, error) {
	return BuildInputManifestWithRunner(req, defaultRunner)
}

// BuildInputManifestContext is the cancellation-aware production entry point.
// Cancellation bounds Git enumeration and is checked between every selected
// file read without changing the manifest identity.
func BuildInputManifestContext(ctx context.Context, req ManifestRequest) (InputManifest, error) {
	return buildInputManifest(req, manifestDeps{ctx: ctx, run: contextRunner(ctx), readFile: os.ReadFile, lstat: os.Lstat, digest: digestBytes})
}

func contextRunner(ctx context.Context) Runner {
	return func(dir, name string, args ...string) ([]byte, error) {
		cmd := exec.CommandContext(ctx, name, args...)
		cmd.Dir = dir
		var stdout, stderr bytes.Buffer
		cmd.Stdout = &stdout
		cmd.Stderr = &stderr
		if err := cmd.Run(); err != nil {
			return nil, fmt.Errorf("%s %s: %w (%s)", name, strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
		}
		return stdout.Bytes(), nil
	}
}

// BuildInputManifestWithRunner exposes only Git enumeration for hermetic
// tests. File reads remain strict: a selected file that disappears, becomes a
// symlink, cannot be read, or changes during capture rejects the manifest.
func BuildInputManifestWithRunner(req ManifestRequest, run Runner) (InputManifest, error) {
	return buildInputManifest(req, manifestDeps{run: run, readFile: os.ReadFile, lstat: os.Lstat, digest: digestBytes})
}

func buildInputManifest(req ManifestRequest, deps manifestDeps) (InputManifest, error) {
	if deps.ctx == nil {
		deps.ctx = context.Background()
	}
	if err := deps.ctx.Err(); err != nil {
		return InputManifest{}, err
	}
	if strings.TrimSpace(req.Primary.Name) == "" {
		req.Primary.Name = "primary"
	}
	specs := append([]RootSpec{req.Primary}, req.Dependencies...)
	seenRoots := make(map[string]struct{}, len(specs))
	roots := make([]ManifestRoot, 0, len(specs))
	for _, spec := range specs {
		if err := deps.ctx.Err(); err != nil {
			return InputManifest{}, err
		}
		name := strings.TrimSpace(spec.Name)
		if name == "" {
			return InputManifest{}, fmt.Errorf("input root name is required")
		}
		if _, duplicate := seenRoots[name]; duplicate {
			return InputManifest{}, fmt.Errorf("duplicate input root name %q", name)
		}
		seenRoots[name] = struct{}{}
		root, err := buildManifestRoot(spec, deps)
		if err != nil {
			return InputManifest{}, fmt.Errorf("resolve input root %q: %w", name, err)
		}
		roots = append(roots, root)
	}
	sort.Slice(roots, func(i, j int) bool { return roots[i].Name < roots[j].Name })
	manifest := InputManifest{
		SchemaVersion: InputManifestSchemaVersion,
		Roots:         roots,
		Configuration: cloneStrings(req.Configuration),
		Toolchain:     cloneStrings(req.Toolchain),
		Attribution:   req.Attribution,
	}
	manifest.Identity = composeManifestIdentity(manifest)
	return manifest, nil
}

// Validate rejects malformed, reordered, or identity-inconsistent manifests at
// storage and wire boundaries. Unknown JSON fields remain forward-compatible;
// a new semantic shape requires a new schema version.
func (m InputManifest) Validate() error {
	if m.SchemaVersion != InputManifestSchemaVersion {
		return fmt.Errorf("unsupported input manifest schema version %d", m.SchemaVersion)
	}
	if len(m.Roots) == 0 {
		return fmt.Errorf("input manifest requires at least one root")
	}
	seenRoots := make(map[string]struct{}, len(m.Roots))
	for i, root := range m.Roots {
		if strings.TrimSpace(root.Name) == "" {
			return fmt.Errorf("input manifest root name is required")
		}
		if _, duplicate := seenRoots[root.Name]; duplicate {
			return fmt.Errorf("duplicate input manifest root %q", root.Name)
		}
		seenRoots[root.Name] = struct{}{}
		if i > 0 && m.Roots[i-1].Name >= root.Name {
			return fmt.Errorf("input manifest roots are not canonically ordered")
		}
		seenPaths := make(map[string]struct{}, len(root.Files))
		for j, file := range root.Files {
			if file.Path != normalizeManifestPath(file.Path) || file.Path == "" || strings.Contains(file.Path, "../") || portableAbsolutePath(file.Path) {
				return fmt.Errorf("unsafe input manifest path %q", file.Path)
			}
			if _, duplicate := seenPaths[strings.ToLower(file.Path)]; duplicate {
				return fmt.Errorf("duplicate or case-colliding input manifest path %q", file.Path)
			}
			seenPaths[strings.ToLower(file.Path)] = struct{}{}
			if j > 0 && root.Files[j-1].Path >= file.Path {
				return fmt.Errorf("input manifest files are not canonically ordered")
			}
			if !validSHA256Identity(file.Digest) || file.Size < 0 {
				return fmt.Errorf("invalid input manifest file metadata for %q", file.Path)
			}
		}
		if root.Identity != composeRootIdentity(root.Files) {
			return fmt.Errorf("input manifest root identity mismatch for %q", root.Name)
		}
	}
	if m.Identity != composeManifestIdentity(m) {
		return fmt.Errorf("input manifest identity mismatch")
	}
	return nil
}

func portableAbsolutePath(value string) bool {
	if filepath.IsAbs(value) || strings.HasPrefix(value, "/") {
		return true
	}
	return len(value) >= 2 && ((value[0] >= 'a' && value[0] <= 'z') || (value[0] >= 'A' && value[0] <= 'Z')) && value[1] == ':'
}

func buildManifestRoot(spec RootSpec, deps manifestDeps) (ManifestRoot, error) {
	path := strings.TrimSpace(spec.Path)
	info, err := os.Stat(path)
	if err != nil || !info.IsDir() {
		return ManifestRoot{}, fmt.Errorf("root path %q is not a readable directory", path)
	}
	if len(spec.Files) > 0 && len(spec.Selections) > 0 {
		return ManifestRoot{}, fmt.Errorf("input files and selections are mutually exclusive")
	}
	if len(spec.Selections) == 0 && len(spec.Files) == 0 {
		return ManifestRoot{}, fmt.Errorf("at least one input selection is required")
	}
	files := spec.Files
	if len(spec.Files) > 0 {
		for _, file := range spec.Files {
			rel := normalizeManifestPath(file)
			if rel == "" || rel == "." || rel == ".." || strings.HasPrefix(rel, "../") || portableAbsolutePath(rel) {
				return ManifestRoot{}, fmt.Errorf("invalid literal input file %q", file)
			}
			for parent := pathpkg.Dir(rel); parent != "."; parent = pathpkg.Dir(parent) {
				info, err := deps.lstat(filepath.Join(path, filepath.FromSlash(parent)))
				if err != nil || !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
					return ManifestRoot{}, fmt.Errorf("literal input %q has an unavailable or symlink parent %q", file, parent)
				}
			}
			spec.Selections = append(spec.Selections, InputSelection{Glob: rel, Required: true})
		}
	} else {
		files, err = listManifestFiles(path, deps.run)
		if err != nil {
			return ManifestRoot{}, err
		}
	}
	matched := make([]bool, len(spec.Selections))
	exact := make(map[string][]int)
	globs := make(map[int]string)
	for i, selection := range spec.Selections {
		glob := normalizeManifestPath(selection.Glob)
		if glob == "" || (len(spec.Files) == 0 && strings.HasPrefix(glob, "!")) {
			return ManifestRoot{}, fmt.Errorf("invalid input selection %q", selection.Glob)
		}
		if len(spec.Files) == 0 && strings.ContainsAny(glob, "*?[") {
			globs[i] = glob
		} else {
			exact[glob] = append(exact[glob], i)
		}
	}
	selected := make(map[string]struct{})
	portable := make(map[string]string)
	for _, candidate := range files {
		if err := deps.ctx.Err(); err != nil {
			return ManifestRoot{}, err
		}
		rel := normalizeManifestPath(candidate)
		indices := append([]int(nil), exact[rel]...)
		for i, glob := range globs {
			if scopedGlobMatch(glob, rel) {
				indices = append(indices, i)
			}
		}
		if len(indices) == 0 {
			continue
		}
		// `git ls-files --cached` includes paths deleted from the working tree.
		// Content identity describes the bytes that exist now, so an already
		// absent index entry is excluded just like any other non-selected path.
		// Once a path is observed here, readStableManifestFile still fails if it
		// disappears or changes during capture.
		if _, err := deps.lstat(filepath.Join(path, filepath.FromSlash(rel))); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return ManifestRoot{}, fmt.Errorf("stat candidate input %q: %w", rel, err)
		}
		for _, i := range indices {
			matched[i] = true
			folded := strings.ToLower(rel)
			if prior, collision := portable[folded]; collision && prior != rel {
				return ManifestRoot{}, fmt.Errorf("portable path collision between %q and %q", prior, rel)
			}
			portable[folded] = rel
			selected[rel] = struct{}{}
		}
	}
	for i, selection := range spec.Selections {
		if selection.Required && !matched[i] {
			return ManifestRoot{}, fmt.Errorf("required input selection %q matched no eligible files", selection.Glob)
		}
	}
	paths := make([]string, 0, len(selected))
	for path := range selected {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	root := ManifestRoot{Name: strings.TrimSpace(spec.Name), Files: make([]ManifestFile, 0, len(paths))}
	for _, rel := range paths {
		if err := deps.ctx.Err(); err != nil {
			return ManifestRoot{}, err
		}
		file, err := readStableManifestFile(path, rel, deps)
		if err != nil {
			return ManifestRoot{}, err
		}
		root.Files = append(root.Files, file)
	}
	root.Identity = composeRootIdentity(root.Files)
	return root, nil
}

func readStableManifestFile(root, rel string, deps manifestDeps) (ManifestFile, error) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	before, err := deps.lstat(path)
	if err != nil {
		return ManifestFile{}, fmt.Errorf("stat selected input %q: %w", rel, err)
	}
	if before.Mode()&os.ModeSymlink != 0 {
		return ManifestFile{}, fmt.Errorf("selected input %q is a symlink", rel)
	}
	if !before.Mode().IsRegular() {
		return ManifestFile{}, fmt.Errorf("selected input %q is not a regular file", rel)
	}
	data, err := deps.readFile(path)
	if err != nil {
		return ManifestFile{}, fmt.Errorf("read selected input %q: %w", rel, err)
	}
	after, err := deps.lstat(path)
	if err != nil {
		return ManifestFile{}, fmt.Errorf("re-stat selected input %q: %w", rel, err)
	}
	if !os.SameFile(before, after) || before.Size() != after.Size() || !before.ModTime().Equal(after.ModTime()) {
		return ManifestFile{}, fmt.Errorf("selected input %q changed while its manifest was captured", rel)
	}
	digest := deps.digest
	if digest == nil {
		digest = digestBytes
	}
	return ManifestFile{Path: rel, Digest: digest(data), Size: int64(len(data))}, nil
}

func digestBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func composeRootIdentity(files []ManifestFile) string {
	h := sha256.New()
	for _, file := range files {
		_, _ = fmt.Fprintf(h, "%s\x00%s\x00%d\n", file.Path, file.Digest, file.Size)
	}
	return "ri:v1:" + hex.EncodeToString(h.Sum(nil))
}

func composeManifestIdentity(manifest InputManifest) string {
	h := sha256.New()
	_, _ = fmt.Fprintf(h, "input-manifest\x00%d\n", manifest.SchemaVersion)
	for _, root := range manifest.Roots {
		_, _ = fmt.Fprintf(h, "root\x00%s\x00%s\n", root.Name, root.Identity)
	}
	writeSortedScalars(h, "configuration", manifest.Configuration)
	writeSortedScalars(h, "toolchain", manifest.Toolchain)
	return "ci:v1:" + hex.EncodeToString(h.Sum(nil))
}

type stringWriter interface{ Write([]byte) (int, error) }

func writeSortedScalars(w stringWriter, kind string, values map[string]string) {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		_, _ = fmt.Fprintf(w, "%s\x00%s\x00%s\n", kind, key, values[key])
	}
}

func normalizeManifestPath(path string) string {
	value := strings.ReplaceAll(strings.TrimSpace(path), "\\", "/")
	for strings.HasPrefix(value, "./") {
		value = strings.TrimPrefix(value, "./")
	}
	if value == "" {
		return ""
	}
	return pathpkg.Clean(value)
}

func validSHA256Identity(identity string) bool {
	encoded := strings.TrimPrefix(identity, "sha256:")
	if len(encoded) != sha256.Size*2 || encoded == identity {
		return false
	}
	_, err := hex.DecodeString(encoded)
	return err == nil
}

func listManifestFiles(root string, run Runner) ([]string, error) {
	if out, err := run(root, "git", "ls-files", "--cached", "--others", "--exclude-standard"); err == nil {
		seen := map[string]struct{}{}
		var files []string
		for _, line := range strings.Split(string(out), "\n") {
			rel := normalizeManifestPath(line)
			if rel == "" || isExcluded(rel) {
				continue
			}
			if _, duplicate := seen[rel]; duplicate {
				continue
			}
			seen[rel] = struct{}{}
			files = append(files, rel)
		}
		return files, nil
	}
	var files []string
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		rel = normalizeManifestPath(rel)
		if entry.IsDir() {
			if rel != "." && isExcluded(rel) {
				return filepath.SkipDir
			}
			return nil
		}
		if !isExcluded(rel) {
			files = append(files, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("enumerate manifest files: %w", err)
	}
	return files, nil
}

func cloneStrings(in map[string]string) map[string]string {
	if len(in) == 0 {
		return nil
	}
	out := make(map[string]string, len(in))
	for key, value := range in {
		out[key] = value
	}
	return out
}
