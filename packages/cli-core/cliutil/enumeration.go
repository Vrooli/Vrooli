package cliutil

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
)

// EnumerationSource identifies how build inputs were discovered.
type EnumerationSource string

const (
	EnumerationGit  EnumerationSource = "git"
	EnumerationWalk EnumerationSource = "walk"
)

// EnumerateDeps contains the small host seam needed by EnumerateInputs.
// Command is injectable so callers can test git failure without touching a
// repository or changing the process environment.
type EnumerateDeps struct {
	Command          func(string, ...string) ([]byte, error)
	ExecutableSuffix string
}

// EnumerateInputs returns repo-relative regular files beneath roots. A work
// tree uses Git's ignore rules for enumeration, while the verdict remains the
// freshness manifest's content/stat comparison. Outside a work tree, or when
// Git is unavailable, it falls back to a conservative filesystem walk.
func EnumerateInputs(repoRoot string, roots []string, deps EnumerateDeps) ([]string, EnumerationSource, error) {
	root := filepath.Clean(repoRoot)
	if deps.Command == nil {
		deps.Command = func(name string, args ...string) ([]byte, error) {
			commandArgs := append([]string{"-C", root}, args...)
			return exec.Command(name, commandArgs...).Output()
		}
	}
	if deps.ExecutableSuffix == "" {
		if runtime.GOOS == "windows" {
			deps.ExecutableSuffix = ".exe"
		}
	}
	args := []string{"ls-files", "--cached", "--others", "--exclude-standard", "-z", "--"}
	for _, r := range roots {
		if strings.TrimSpace(r) != "" {
			args = append(args, filepath.ToSlash(filepath.Clean(r)))
		}
	}
	if out, err := deps.Command("git", args...); err == nil {
		files := splitNUL(out)
		for i := range files {
			files[i] = filepath.ToSlash(filepath.Clean(files[i]))
		}
		sort.Strings(files)
		return uniqueStrings(files), EnumerationGit, nil
	}

	seen := map[string]struct{}{}
	var files []string
	for _, inputRoot := range roots {
		start := filepath.Join(root, filepath.FromSlash(inputRoot))
		err := filepath.WalkDir(start, func(path string, entry fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if entry.IsDir() {
				if buildOutputDirectory(entry.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if !entry.Type().IsRegular() || buildOutputFile(entry.Name(), deps.ExecutableSuffix) {
				return nil
			}
			rel, relErr := filepath.Rel(root, path)
			if relErr != nil {
				return relErr
			}
			rel = filepath.ToSlash(rel)
			if _, ok := seen[rel]; !ok {
				seen[rel] = struct{}{}
				files = append(files, rel)
			}
			return nil
		})
		if err != nil && !os.IsNotExist(err) {
			return nil, EnumerationWalk, err
		}
	}
	sort.Strings(files)
	return files, EnumerationWalk, nil
}

func splitNUL(data []byte) []string {
	parts := strings.Split(string(data), "\x00")
	if len(parts) > 0 && parts[len(parts)-1] == "" {
		parts = parts[:len(parts)-1]
	}
	return parts
}

func uniqueStrings(values []string) []string {
	out := values[:0]
	for _, value := range values {
		if len(out) == 0 || out[len(out)-1] != value {
			out = append(out, value)
		}
	}
	return out
}

func buildOutputDirectory(name string) bool {
	switch name {
	case ".git", ".vrooli", "node_modules", "dist", "build", "coverage", "data", "tmp":
		return true
	default:
		return false
	}
}

func buildOutputFile(name, executableSuffix string) bool {
	if name == "coverage.out" || strings.HasSuffix(name, ".freshness.json") || strings.HasSuffix(name, ".log") || strings.HasSuffix(name, ".test") {
		return true
	}
	return executableSuffix != "" && strings.HasSuffix(strings.ToLower(name), strings.ToLower(executableSuffix))
}
