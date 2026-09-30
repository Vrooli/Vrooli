package cliutil

import (
	"crypto/sha256"
	"fmt"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/vrooli/cli-core/buildinfo"
)

// freshnessManifestVersion is the on-disk schema version of FreshnessManifest.
// Bump only on incompatible layout changes; a version mismatch is treated as
// "no usable manifest" (stale-once, self-healing) by EvaluateFreshness callers.
const freshnessManifestVersion = 1

// FileManifestEntry records one input file's fingerprint at the time the
// artifact was built. mtime is stored (as Unix nanoseconds) so the stat-cache
// can skip re-hashing files whose (size, mtime) are unchanged.
type FileManifestEntry struct {
	Rel     string `json:"rel"`
	Size    int64  `json:"size"`
	MTimeNS int64  `json:"mtime_ns"`
	Hash    string `json:"sha256"`
}

// DirectoryManifestEntry records the directories that contain recorded input
// files. Their metadata lets a warm check detect additions and removals without
// walking every input directory; a changed directory falls back to the full
// enumerator so the file-set semantics remain unchanged.
type DirectoryManifestEntry struct {
	Rel      string   `json:"rel"`
	Size     int64    `json:"size"`
	MTimeNS  int64    `json:"mtime_ns"`
	Children []string `json:"children,omitempty"`
}

// FreshnessManifest is the recorded freshness stamp written next to a build
// artifact at setup success. Staleness becomes a manifest comparison (stat-cache
// + selective re-hash) rather than a full mtime tree walk. The aggregate Digest
// keys both file content AND non-file build inputs (toolchain version, NODE_ENV)
// so a byte-identical source tree built under a different toolchain/env still
// reads as stale — closing the false-negative hole that pure content hashing
// would open.
type FreshnessManifest struct {
	Version     int    `json:"version"`
	CheckType   string `json:"check_type"`
	WrittenAtNS int64  `json:"written_at_ns"`
	// Digest is the aggregate fingerprint over file content + KeyInputs. It is a
	// debug/identity field surfaced in the stamped manifest (and `--json` reads),
	// NOT the freshness decision input: EvaluateFreshness compares KeyInputs and
	// per-file entries directly (so it can name the precise offender), never the
	// aggregate. Retained as a single human-legible "what was this build" handle.
	Digest      string                   `json:"digest"`
	Inputs      []string                 `json:"inputs,omitempty"`
	KeyInputs   map[string]string        `json:"key_inputs,omitempty"`
	Files       []FileManifestEntry      `json:"files"`
	Directories []DirectoryManifestEntry `json:"directories,omitempty"`
}

// FreshnessVerdict is the result of comparing the live input set against a
// recorded manifest.
type FreshnessVerdict struct {
	Stale  bool
	Reason string // human-facing "what changed" (e.g. "content changed")
	File   string // offending input (rel path) or key-input name, when applicable
}

func staleVerdict(reason, file string) FreshnessVerdict {
	return FreshnessVerdict{Stale: true, Reason: reason, File: file}
}

// ComputeFreshnessManifest builds a manifest from a freshness spec plus the
// non-file keyed inputs (toolchain version, NODE_ENV, …). writtenAtNS stamps the
// write time used by the racy-index re-hash rule; callers pass a real clock
// value (time.Now().UnixNano()).
func ComputeFreshnessManifest(spec FreshnessSpec, checkType string, keyInputs map[string]string, writtenAtNS int64) (FreshnessManifest, error) {
	entries, err := collectFreshnessEntries(spec)
	if err != nil {
		return FreshnessManifest{}, err
	}
	return FreshnessManifest{
		Version:     freshnessManifestVersion,
		CheckType:   strings.TrimSpace(checkType),
		WrittenAtNS: writtenAtNS,
		Digest:      aggregateManifestDigest(entries, keyInputs),
		Inputs:      append([]string(nil), spec.Inputs...),
		KeyInputs:   copyStringMap(keyInputs),
		Files:       entries,
		Directories: freshnessInputDirectories(spec, entries),
	}, nil
}

// EvaluateFreshness compares the current input set against the recorded manifest
// using a stat-cache: a file whose (size, mtime) match the manifest AND whose
// mtime is strictly older than the manifest's own write time is trusted without
// re-hashing. Any file whose mtime changed, or whose mtime is at/after the
// manifest write time (Git's racy-index rule, defending coarse-granularity
// filesystems), is re-hashed and compared. A key-input change, an added or
// removed input, or a content change all yield a stale verdict naming the cause.
//
// A manifest with the wrong version or a mismatched check type is reported stale
// so callers bootstrap a fresh stamp (self-healing).
func EvaluateFreshness(spec FreshnessSpec, manifest FreshnessManifest, keyInputs map[string]string) (FreshnessVerdict, error) {
	if manifest.Version != freshnessManifestVersion {
		return staleVerdict("manifest version mismatch", ""), nil
	}
	if v := compareKeyInputs(manifest.KeyInputs, keyInputs); v.Stale {
		return v, nil
	}

	current, err := statFreshnessInputsForManifest(spec, manifest)
	if err != nil {
		return FreshnessVerdict{}, err
	}

	// On a case-insensitive volume the OS may report a different casing for the
	// same file across builds, so fold comparison keys to avoid spurious
	// new/removed-input verdicts. The reported offender keeps its live casing.
	foldKey := func(rel string) string { return rel }
	if spec.CaseInsensitive {
		foldKey = strings.ToLower
	}

	recorded := make(map[string]FileManifestEntry, len(manifest.Files))
	for _, e := range manifest.Files {
		recorded[foldKey(e.Rel)] = e
	}

	// Stable iteration order so the named offender is deterministic.
	rels := make([]string, 0, len(current))
	for rel := range current {
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	for _, rel := range rels {
		st := current[rel]
		rec, ok := recorded[foldKey(rel)]
		if !ok {
			return staleVerdict("new input file", rel), nil
		}
		if st.size != rec.Size {
			return staleVerdict("size changed", rel), nil
		}
		// Racy-index rule: trust the recorded hash only when the mtime is
		// unchanged AND strictly older than the manifest write time. Otherwise
		// re-hash to confirm.
		if st.mtimeNS != rec.MTimeNS || st.mtimeNS >= manifest.WrittenAtNS {
			hash, err := hashFileContent(st.abs)
			if err != nil {
				return FreshnessVerdict{}, err
			}
			if hash != rec.Hash {
				return staleVerdict("content changed", rel), nil
			}
		}
	}

	currentKeys := make(map[string]struct{}, len(current))
	for rel := range current {
		currentKeys[foldKey(rel)] = struct{}{}
	}
	for rel := range recorded {
		if _, ok := currentKeys[rel]; ok {
			continue
		}
		// A manifest written before build outputs were excluded still lists them.
		// Treating that as a removed input would declare every pre-existing
		// manifest stale at once, so ignore entries the current rules exclude.
		if isBuildOutput(rel) {
			continue
		}
		return staleVerdict("input file removed", rel), nil
	}

	return FreshnessVerdict{Stale: false}, nil
}

func compareKeyInputs(recorded, current map[string]string) FreshnessVerdict {
	// Union of keys so both an added and a removed key are detected.
	seen := make(map[string]struct{}, len(recorded)+len(current))
	keys := make([]string, 0, len(recorded)+len(current))
	for k := range recorded {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}
	for k := range current {
		if _, ok := seen[k]; !ok {
			seen[k] = struct{}{}
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	for _, k := range keys {
		if recorded[k] != current[k] {
			// Reason is the generic cause and File carries the offending key,
			// mirroring the file-offender verdicts ("content changed" + path) so
			// callers that render "(reason: file)" don't double-print the key.
			return staleVerdict("build input changed", k)
		}
	}
	return FreshnessVerdict{}
}

type statEntry struct {
	size    int64
	mtimeNS int64
	abs     string
}

// statFreshnessInputs enumerates the spec's input files and records (size,
// mtime) without hashing. It applies the identical skip-dir / skip-file /
// compiled-binary exclusions as the manifest collector so the live set and the
// recorded set are directly comparable. The compiled-binary check peeks only the
// first bytes, keeping the steady-state sweep stat-(plus-tiny-read)-only.
func statFreshnessInputs(spec FreshnessSpec) (map[string]statEntry, error) {
	out := map[string]statEntry{}
	visit := func(root, abs string, info fs.FileInfo) {
		rel, err := filepath.Rel(root, abs)
		if err != nil {
			return
		}
		out[filepath.ToSlash(rel)] = statEntry{size: info.Size(), mtimeNS: info.ModTime().UnixNano(), abs: abs}
	}
	return out, walkFreshnessInputs(spec, func(root, abs string, info fs.FileInfo, isCompiled func() bool) error {
		if isCompiled() {
			return nil
		}
		visit(root, abs, info)
		return nil
	})
}

// statFreshnessInputsForManifest is the warm path for the manifest engine. A
// manifest from the current schema records the parent directory metadata for
// every input file. When those directories are unchanged, a direct stat of
// the recorded files is enough: no file can have been added or removed beneath
// an unchanged directory. Any directory change re-enters the canonical walk,
// preserving the complete input-set semantics of EvaluateFreshness.
func statFreshnessInputsForManifest(spec FreshnessSpec, manifest FreshnessManifest) (map[string]statEntry, error) {
	if len(manifest.Directories) == 0 {
		return statFreshnessInputs(spec)
	}
	contextRoot := filepath.Clean(strings.TrimSpace(spec.ContextRoot))
	if contextRoot == "" {
		contextRoot = filepath.Clean(strings.TrimSpace(spec.SourceRoot))
	}
	for _, directory := range manifest.Directories {
		abs := filepath.Join(contextRoot, filepath.FromSlash(directory.Rel))
		info, err := os.Stat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				return statFreshnessInputs(spec)
			}
			return nil, err
		}
		if !info.IsDir() {
			return statFreshnessInputs(spec)
		}
		if directory.Rel == "." {
			children, childrenErr := freshnessDirectoryChildren(abs, spec)
			if childrenErr != nil || !slices.Equal(children, directory.Children) {
				return statFreshnessInputs(spec)
			}
		} else if info.Size() != directory.Size || info.ModTime().UnixNano() != directory.MTimeNS {
			return statFreshnessInputs(spec)
		}
	}

	current := make(map[string]statEntry, len(manifest.Files))
	for _, file := range manifest.Files {
		abs := filepath.Join(contextRoot, filepath.FromSlash(file.Rel))
		info, err := os.Lstat(abs)
		if err != nil {
			if os.IsNotExist(err) {
				return statFreshnessInputs(spec)
			}
			return nil, err
		}
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return statFreshnessInputs(spec)
		}
		current[file.Rel] = statEntry{size: info.Size(), mtimeNS: info.ModTime().UnixNano(), abs: abs}
	}
	return current, nil
}

func freshnessInputDirectories(spec FreshnessSpec, entries []FileManifestEntry) []DirectoryManifestEntry {
	contextRoot := filepath.Clean(strings.TrimSpace(spec.ContextRoot))
	if contextRoot == "" {
		contextRoot = filepath.Clean(strings.TrimSpace(spec.SourceRoot))
	}
	seen := make(map[string]struct{})
	directories := make([]DirectoryManifestEntry, 0, len(entries))
	for _, entry := range entries {
		rel := filepath.ToSlash(filepath.Dir(entry.Rel))
		if rel == "." {
			// The source root itself is still relevant for detecting a new
			// top-level input file.
		}
		if _, ok := seen[rel]; ok {
			continue
		}
		abs := filepath.Join(contextRoot, filepath.FromSlash(rel))
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			continue
		}
		seen[rel] = struct{}{}
		entry := DirectoryManifestEntry{Rel: rel, Size: info.Size(), MTimeNS: info.ModTime().UnixNano()}
		if rel == "." {
			entry.Children, _ = freshnessDirectoryChildren(abs, spec)
		}
		directories = append(directories, entry)
	}
	sort.Slice(directories, func(i, j int) bool { return directories[i].Rel < directories[j].Rel })
	return directories
}

func freshnessDirectoryChildren(dir string, spec FreshnessSpec) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	children := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := entry.Name()
		if name == FreshnessManifestSuffix || strings.HasSuffix(name, FreshnessManifestSuffix) || isBuildOutput(name) || hasSkippedSuffix(name, spec.SkipSuffixes) {
			continue
		}
		if entry.IsDir() {
			name += "/"
		}
		children = append(children, name)
	}
	sort.Strings(children)
	return children, nil
}

func collectFreshnessEntries(spec FreshnessSpec) ([]FileManifestEntry, error) {
	var entries []FileManifestEntry
	err := walkFreshnessInputs(spec, func(root, abs string, info fs.FileInfo, _ func() bool) error {
		content, readErr := os.ReadFile(abs)
		if readErr != nil {
			return readErr
		}
		if buildinfo.IsCompiledBinary(content) {
			return nil
		}
		rel, relErr := filepath.Rel(root, abs)
		if relErr != nil {
			return relErr
		}
		sum := sha256.Sum256(content)
		entries = append(entries, FileManifestEntry{
			Rel:     filepath.ToSlash(rel),
			Size:    info.Size(),
			MTimeNS: info.ModTime().UnixNano(),
			Hash:    fmt.Sprintf("%x", sum),
		})
		return nil
	})
	if err != nil {
		return nil, err
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].Rel < entries[j].Rel })
	return entries, nil
}

// walkFreshnessInputs is the single traversal shared by the manifest collector
// (which hashes) and the stat-cache enumerator (which does not). The visit
// callback receives the traversal root used for rel-path computation, the
// absolute file path, its FileInfo, and a lazy isCompiled() predicate (reads
// only the leading magic bytes) so the caller decides whether to exclude
// compiled artifacts without an unconditional full read.
func walkFreshnessInputs(spec FreshnessSpec, visit func(root, abs string, info fs.FileInfo, isCompiled func() bool) error) error {
	sourceRoot := filepath.Clean(strings.TrimSpace(spec.SourceRoot))
	if sourceRoot == "" {
		return fmt.Errorf("freshness source root must not be empty")
	}
	skip := append([]string(nil), spec.SkipFiles...)
	skipSuffixes := append([]string(nil), spec.SkipSuffixes...)
	inputs := trimNonEmpty(spec.Inputs)
	if len(inputs) == 0 {
		files, _, err := EnumerateInputs(sourceRoot, []string{"."}, EnumerateDeps{})
		if err != nil {
			return err
		}
		for _, rel := range files {
			match := filepath.Join(sourceRoot, filepath.FromSlash(rel))
			info, err := os.Stat(match)
			if err != nil {
				return err
			}
			if buildinfoSkipFile(rel, skip) || hasSkippedSuffix(rel, skipSuffixes) || isBuildOutput(rel) {
				continue
			}
			if err := visit(sourceRoot, match, info, magicPeek(match)); err != nil {
				return err
			}
		}
		return nil
	}
	contextRoot := filepath.Clean(strings.TrimSpace(spec.ContextRoot))
	if contextRoot == "" {
		contextRoot = sourceRoot
	}
	for _, input := range inputs {
		matches, err := expandDeclaredInputPaths(contextRoot, input)
		if err != nil {
			return err
		}
		for _, match := range matches {
			linkInfo, linkErr := os.Lstat(match)
			if linkErr != nil {
				if os.IsNotExist(linkErr) {
					continue
				}
				return linkErr
			}
			if linkInfo.Mode()&os.ModeSymlink != 0 {
				// Symlinks are references, not source files. In particular, a
				// broken fixture link must not make an otherwise valid artifact
				// impossible to stamp.
				continue
			}
			info, err := os.Stat(match)
			if err != nil {
				if os.IsNotExist(err) {
					continue
				}
				return err
			}
			if info.IsDir() {
				if err := walkTreeInputs(contextRoot, match, skip, skipSuffixes, visit); err != nil {
					return err
				}
				continue
			}
			if !info.Mode().IsRegular() {
				// Sockets, devices, FIFOs, and other runtime artifacts are not
				// build inputs and may not be readable as file content.
				continue
			}
			rel, err := filepath.Rel(contextRoot, match)
			if err != nil {
				return err
			}
			relSlash := filepath.ToSlash(rel)
			if buildinfoSkipFile(relSlash, skip) || hasSkippedSuffix(relSlash, skipSuffixes) || isBuildOutput(relSlash) {
				continue
			}
			if err := visit(contextRoot, match, info, magicPeek(match)); err != nil {
				return err
			}
		}
	}
	return nil
}

func walkTreeInputs(root, start string, skip, skipSuffixes []string, visit func(root, abs string, info fs.FileInfo, isCompiled func() bool) error) error {
	return filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}
		if d.Type()&os.ModeSymlink != 0 {
			return nil
		}
		if buildinfoSkipDir(rel) && d.IsDir() {
			return filepath.SkipDir
		}
		if d.IsDir() || buildinfoSkipFile(rel, skip) || hasSkippedSuffix(rel, skipSuffixes) || isBuildOutput(rel) {
			return nil
		}
		info, infoErr := d.Info()
		if infoErr != nil {
			return infoErr
		}
		if !info.Mode().IsRegular() {
			return nil
		}
		return visit(root, path, info, magicPeek(path))
	})
}

// BuildOutputSkipNames and BuildOutputSkipSuffixes name files that a build or
// test run *produces*, never files it consumes. They are applied unconditionally
// inside the freshness walk so every caller — the manifest writer, the stat-cache
// evaluator, and the preflight staleness checker — shares one definition and
// cannot drift apart.
//
// This exists because these outputs were being recorded as build inputs: a
// `go test -coverprofile` run rewrote coverage.out, which made the binary look
// stale, which (under any auto-rebuild policy) would rebuild it, which the next
// test run would invalidate again. Compiled artifacts are already excluded by
// the magic-number check; these are the text outputs it cannot see.
var (
	BuildOutputSkipNames = []string{
		"coverage.out", "cov.out", "cover.out",
		".vrooli-closure-go_module.json",
		// jscpd writes its duplication report into the api tree, so a scan run
		// would otherwise stale the binary that never consumed it.
		"jscpd-report.json",
	}

	BuildOutputSkipSuffixes = []string{".log", ".test", ".tmp"}
)

// isBuildOutput reports whether rel names a build/test product rather than a
// build input.
func isBuildOutput(rel string) bool {
	base := path.Base(rel)
	for _, name := range BuildOutputSkipNames {
		if base == name {
			return true
		}
	}
	for _, suffix := range BuildOutputSkipSuffixes {
		if strings.HasSuffix(base, suffix) {
			return true
		}
	}
	return false
}

// BuildOutputSkipSuffixesFor returns output suffixes for the target platform.
// Windows executables are excluded only when the caller's executable probe
// says .exe is meaningful; a source file named *.exe on Unix is an input.
func BuildOutputSkipSuffixesFor(executableSuffix string) []string {
	suffixes := append([]string(nil), BuildOutputSkipSuffixes...)
	if strings.TrimSpace(executableSuffix) != "" {
		suffixes = append(suffixes, executableSuffix)
	}
	return suffixes
}

func hasSkippedSuffix(rel string, suffixes []string) bool {
	for _, s := range suffixes {
		s = strings.TrimSpace(s)
		if s != "" && strings.HasSuffix(rel, s) {
			return true
		}
	}
	return false
}

// magicPeek returns a predicate that reports whether the file at path begins
// with a compiled-executable magic number, reading only the leading bytes.
func magicPeek(path string) func() bool {
	return func() bool {
		f, err := os.Open(path)
		if err != nil {
			return false
		}
		defer f.Close()
		var buf [4]byte
		n, _ := f.Read(buf[:])
		return buildinfo.IsCompiledBinary(buf[:n])
	}
}

func hashFileContent(path string) (string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(content)
	return fmt.Sprintf("%x", sum), nil
}

func aggregateManifestDigest(entries []FileManifestEntry, keyInputs map[string]string) string {
	hasher := sha256.New()
	for _, e := range entries {
		fmt.Fprintf(hasher, "%s|%d|%s\n", e.Rel, e.Size, e.Hash)
	}
	for _, k := range sortedKeys(keyInputs) {
		fmt.Fprintf(hasher, "@%s=%s\n", k, keyInputs[k])
	}
	return fmt.Sprintf("%x", hasher.Sum(nil))
}

func sortedKeys(m map[string]string) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func copyStringMap(m map[string]string) map[string]string {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
