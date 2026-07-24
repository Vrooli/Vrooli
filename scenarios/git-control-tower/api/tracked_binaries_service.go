package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Compiled executables are build output and must never be tracked: large,
// unmergeable, platform-specific, and stale the moment their source changes.
// They arrive the same way every time -- a build drops a binary beside its
// source and a broad `git add` sweeps it in -- and stay invisible because
// nothing in the normal review flow surfaces a file's type.
//
// Detection reads the index (git ls-files), not the working tree: an untracked
// binary sitting next to its source is a normal build artifact, not a defect.

// executableFormats maps a leading magic sequence to its format label.
var executableFormats = []struct {
	magic  []byte
	format string
}{
	{[]byte{0x7f, 'E', 'L', 'F'}, "elf"},
	{[]byte{0xfe, 0xed, 0xfa, 0xce}, "mach-o"},
	{[]byte{0xfe, 0xed, 0xfa, 0xcf}, "mach-o"},
	{[]byte{0xce, 0xfa, 0xed, 0xfe}, "mach-o"},
	{[]byte{0xcf, 0xfa, 0xed, 0xfe}, "mach-o"},
	{[]byte{0xca, 0xfe, 0xba, 0xbe}, "mach-o"},
	{[]byte{'M', 'Z'}, "pe"},
}

// binaryScanSkipExts are extensions that cannot be a stray build output. This is
// a speed filter over a whole repository index, not a correctness boundary.
var binaryScanSkipExts = map[string]struct{}{
	".go": {}, ".ts": {}, ".tsx": {}, ".js": {}, ".jsx": {}, ".mjs": {}, ".cjs": {},
	".json": {}, ".yaml": {}, ".yml": {}, ".toml": {}, ".md": {}, ".txt": {},
	".sh": {}, ".bash": {}, ".py": {}, ".sql": {}, ".proto": {}, ".html": {},
	".css": {}, ".scss": {}, ".svg": {}, ".png": {}, ".jpg": {}, ".jpeg": {},
	".gif": {}, ".webp": {}, ".ico": {}, ".mp3": {}, ".mp4": {}, ".wav": {},
	".glb": {}, ".gltf": {}, ".woff": {}, ".woff2": {}, ".ttf": {}, ".pdf": {},
	".lock": {}, ".sum": {}, ".mod": {}, ".jsonl": {}, ".csv": {}, ".xml": {},
}

// ownerRoots are the directory prefixes whose immediate child owns its own
// .gitignore, mirroring the per-resource convention already in the repo.
var ownerRoots = []string{"scenarios", "resources", "packages", "templates"}

const untrackHistoryWarning = "Untracking removes these from the working tree and future commits. " +
	"The bytes remain in git history, so repository size is unchanged until history is rewritten."

// AnalyzeTrackedBinaries reports every compiled executable in the index.
func AnalyzeTrackedBinaries(ctx context.Context, deps HealthDeps, git GitRunner) (*TrackedBinariesResponse, error) {
	tracked, err := git.ListTrackedFiles(ctx, deps.RepoDir)
	if err != nil {
		return nil, fmt.Errorf("list tracked files: %w", err)
	}

	resp := &TrackedBinariesResponse{Binaries: []TrackedBinary{}}
	for _, rel := range tracked {
		rel = strings.TrimSpace(rel)
		if rel == "" {
			continue
		}
		if _, skip := binaryScanSkipExts[strings.ToLower(filepath.Ext(rel))]; skip {
			continue
		}
		format, size, ok := classifyExecutable(deps.FS, filepath.Join(deps.RepoDir, filepath.FromSlash(rel)))
		if !ok {
			continue
		}
		ownerDir, pattern := ignoreTargetForBinary(rel)
		resp.Binaries = append(resp.Binaries, TrackedBinary{
			Path:           rel,
			Bytes:          size,
			Format:         format,
			OwnerDir:       ownerDir,
			IgnorePattern:  pattern,
			AlreadyIgnored: gitignoreContainsPattern(deps.FS, deps.RepoDir, ownerDir, pattern),
		})
		resp.TotalBytes += size
	}

	sort.Slice(resp.Binaries, func(i, j int) bool {
		if resp.Binaries[i].Bytes != resp.Binaries[j].Bytes {
			return resp.Binaries[i].Bytes > resp.Binaries[j].Bytes
		}
		return resp.Binaries[i].Path < resp.Binaries[j].Path
	})
	if len(resp.Binaries) > 0 {
		resp.HistoryWarning = untrackHistoryWarning
	}
	return resp, nil
}

// classifyExecutable reports the executable format and size of path.
//
// Symlinks are skipped: git stores a symlink's target path rather than the
// target's content, so following one would report a large binary that is not in
// history and point the remediation at the wrong object.
func classifyExecutable(fsio FileIO, path string) (format string, size int64, ok bool) {
	info, err := fsio.Lstat(path)
	if err != nil || info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
		return "", 0, false
	}
	data, err := fsio.ReadFile(path)
	if err != nil || len(data) < 2 {
		return "", 0, false
	}
	for _, candidate := range executableFormats {
		if len(data) >= len(candidate.magic) && bytes.Equal(data[:len(candidate.magic)], candidate.magic) {
			return candidate.format, info.Size(), true
		}
	}
	return "", 0, false
}

// ignoreTargetForBinary picks the .gitignore that should own the path and the
// pattern to add to it. Ownership goes to the scenario/resource/package
// directory so the ignore lives beside the thing that builds the binary, which
// is the convention every resource already follows. A repo-root binary has no
// owner directory and belongs in the root .gitignore.
func ignoreTargetForBinary(rel string) (ownerDir, pattern string) {
	parts := strings.Split(filepath.ToSlash(rel), "/")
	for _, root := range ownerRoots {
		if len(parts) >= 3 && parts[0] == root {
			owner := parts[0] + "/" + parts[1]
			return owner, "/" + strings.Join(parts[2:], "/")
		}
	}
	return "", "/" + filepath.ToSlash(rel)
}

func gitignorePathFor(repoDir, ownerDir string) string {
	if strings.TrimSpace(ownerDir) == "" {
		return filepath.Join(repoDir, ".gitignore")
	}
	return filepath.Join(repoDir, filepath.FromSlash(ownerDir), ".gitignore")
}

func gitignoreContainsPattern(fsio FileIO, repoDir, ownerDir, pattern string) bool {
	data, err := fsio.ReadFile(gitignorePathFor(repoDir, ownerDir))
	if err != nil {
		return false
	}
	want := strings.TrimSpace(pattern)
	trimmed := strings.TrimPrefix(want, "/")
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == want || line == trimmed {
			return true
		}
	}
	return false
}

// UntrackBinary removes one binary from the index and ensures its owning
// .gitignore covers it, so the next build does not re-stage it.
//
// Order matters: the ignore is written first. If the index removal then fails
// the tree is still safe (an ignored, tracked file is harmless); the reverse
// order would leave the file untracked and unignored, i.e. staged again by the
// next `git add -A`.
func UntrackBinary(ctx context.Context, deps HealthDeps, git GitRunner, req UntrackBinaryRequest) (*UntrackBinaryResponse, error) {
	path := strings.TrimSpace(req.Path)
	if path == "" {
		return &UntrackBinaryResponse{Error: "path is required"}, nil
	}
	if strings.Contains(path, "..") || filepath.IsAbs(path) {
		return &UntrackBinaryResponse{Error: "path must be repo-relative"}, nil
	}

	resp := &UntrackBinaryResponse{}
	pattern := strings.TrimSpace(req.IgnorePattern)
	if pattern == "" {
		_, pattern = ignoreTargetForBinary(path)
	}
	if !gitignoreContainsPattern(deps.FS, deps.RepoDir, req.OwnerDir, pattern) {
		ignorePath := gitignorePathFor(deps.RepoDir, req.OwnerDir)
		if err := appendGitignorePattern(deps.FS, ignorePath, pattern); err != nil {
			resp.Error = fmt.Sprintf("update %s: %v", ignorePath, err)
			return resp, nil
		}
		resp.IgnoreAddedTo = strings.TrimPrefix(strings.TrimPrefix(ignorePath, deps.RepoDir), string(filepath.Separator))
	}

	if err := git.RemoveFromIndex(ctx, deps.RepoDir, []string{path}); err != nil {
		resp.Error = err.Error()
		return resp, nil
	}
	resp.RemovedFromIndex = true
	resp.Success = true
	return resp, nil
}

func appendGitignorePattern(fsio FileIO, ignorePath, pattern string) error {
	existing, err := fsio.ReadFile(ignorePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		existing = nil
		if mkErr := fsio.MkdirAll(filepath.Dir(ignorePath), 0o755); mkErr != nil {
			return mkErr
		}
	}

	var b strings.Builder
	b.Write(existing)
	if len(existing) > 0 && !bytes.HasSuffix(existing, []byte("\n")) {
		b.WriteString("\n")
	}
	if len(existing) == 0 {
		b.WriteString("# Compiled build output. Never commit binaries.\n")
	}
	b.WriteString(pattern)
	b.WriteString("\n")
	return fsio.WriteFile(ignorePath, []byte(b.String()), 0o644)
}
