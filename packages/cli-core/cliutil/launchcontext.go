package cliutil

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	repocontract "github.com/vrooli/repo-contract-go"
)

const (
	// LaunchSourceRootEnv is the canonical source-tree hint inherited by
	// coding-agent children. It is intentionally distinct from the directory
	// the agent edits: an operator may launch from a subdirectory or a sandbox.
	LaunchSourceRootEnv    = "VROOLI_SOURCE_ROOT"
	LaunchRepoRootEnv      = "VROOLI_ROOT"
	LaunchRuntimeBinaryEnv = "VROOLI_BIN"
	LaunchProjectRootEnv   = "PROJECT_ROOT"
	LaunchSandboxMergedEnv = "VROOLI_SANDBOX_MERGED"
	LaunchSourcePointerEnv = "VROOLI_SOURCE_ROOT_POINTER"
)

// LaunchContext is the validated process context shared by coding-agent
// launchers and other native interactive tools.
//
// WorkingDir is the tree the child edits. ProjectRoot is the logical project
// root associated with that tree, which may be a non-Vrooli application in a
// bundled environment. SourceRoot is an actual Vrooli checkout, when one is
// available for locating Vrooli-owned capabilities.
type LaunchContext struct {
	WorkingDir  string
	ProjectRoot string
	SourceRoot  string
	ToolDirs    []string
	Source      string
}

// LaunchContextRequest supplies the caller-owned pieces of launch context.
// Environment is a complete child environment when non-nil; nil means the
// current process environment.
type LaunchContextRequest struct {
	WorkingDir  string
	Environment []string
}

// ResolveLaunchContext resolves and validates the context for a native child.
// An explicit working directory is authoritative and must exist. When the
// inherited cwd is unusable, a configured repository pointer can still recover
// the Vrooli source tree. If neither is available, returning an error is safer
// than silently placing a coding agent in a temporary directory.
func ResolveLaunchContext(request LaunchContextRequest) (LaunchContext, error) {
	environment := request.Environment
	if environment == nil {
		environment = os.Environ()
	}

	if explicit := strings.TrimSpace(request.WorkingDir); explicit != "" {
		workingDir, err := usableDirectory(explicit)
		if err != nil {
			return LaunchContext{}, fmt.Errorf("resolve requested working directory %q: %w", explicit, err)
		}
		return contextForWorkingDir(workingDir, environment, "explicit"), nil
	}

	if merged := environmentValue(environment, LaunchSandboxMergedEnv); merged != "" {
		if workingDir, err := usableDirectory(merged); err == nil {
			return contextForWorkingDir(workingDir, environment, "sandbox"), nil
		}
	}

	if cwd, err := currentWorkingDirectory(); err == nil {
		if workingDir, dirErr := usableDirectory(cwd); dirErr == nil {
			return contextForWorkingDir(workingDir, environment, "current-directory"), nil
		}
	}

	if projectRoot := configuredProjectRoot(environment); projectRoot != "" {
		return contextForWorkingDir(projectRoot, environment, "configured-project-root"), nil
	}

	if root := configuredSourceRoot(environment); root != "" {
		return contextForWorkingDir(root, environment, "configured-source-root"), nil
	}

	if pointer := sourceRootPointer(environment); pointer != "" {
		if root := validRepo(pointer); root != "" {
			return contextForWorkingDir(root, environment, "source-root-pointer"), nil
		}
	}

	return LaunchContext{}, errors.New("unable to resolve a usable working directory; set --cwd, PROJECT_ROOT, or VROOLI_SOURCE_ROOT")
}

// PrepareLaunchEnvironment applies the resolved context to a child
// environment. Existing explicit values win. PATH entries are prepended only
// when they are real directories, and all joining uses the host's path-list
// separator.
func PrepareLaunchEnvironment(environment []string, context LaunchContext) []string {
	if environment == nil {
		environment = os.Environ()
	}
	prepared := append([]string(nil), environment...)
	if context.SourceRoot != "" {
		if environmentValue(prepared, LaunchSourceRootEnv) == "" || validRepo(environmentValue(prepared, LaunchSourceRootEnv)) == "" {
			prepared = withEnvironmentValue(prepared, LaunchSourceRootEnv, context.SourceRoot)
		}
		if environmentValue(prepared, LaunchRepoRootEnv) == "" || validRepo(environmentValue(prepared, LaunchRepoRootEnv)) == "" {
			prepared = withEnvironmentValue(prepared, LaunchRepoRootEnv, context.SourceRoot)
		}
	}
	return prependPathEntries(prepared, context.ToolDirs)
}

func contextForWorkingDir(workingDir string, environment []string, source string) LaunchContext {
	context := LaunchContext{WorkingDir: workingDir, Source: source}
	context.ProjectRoot = validRepo(workingDir)
	if context.ProjectRoot == "" {
		context.ProjectRoot = workingDir
	}
	context.SourceRoot = configuredSourceRoot(environment)
	if context.SourceRoot == "" {
		context.SourceRoot = validRepo(workingDir)
	}
	if context.SourceRoot == "" {
		context.SourceRoot = sourceRootPointer(environment)
	}
	context.ToolDirs = runtimeToolDirs(context.SourceRoot, environment)
	return context
}

var currentWorkingDirectory = os.Getwd

func usableDirectory(path string) (string, error) {
	path = strings.TrimSpace(path)
	if path == "" {
		return "", errors.New("path is empty")
	}
	absolute, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("make absolute: %w", err)
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return "", err
	}
	if !info.IsDir() {
		return "", errors.New("path is not a directory")
	}
	return filepath.Clean(absolute), nil
}

func configuredSourceRoot(environment []string) string {
	for _, key := range []string{LaunchSourceRootEnv, LaunchRepoRootEnv} {
		if root := validRepo(environmentValue(environment, key)); root != "" {
			return root
		}
	}
	return ""
}

func configuredProjectRoot(environment []string) string {
	if projectRoot, err := usableDirectory(environmentValue(environment, LaunchProjectRootEnv)); err == nil {
		return projectRoot
	}
	return configuredSourceRoot(environment)
}

func validRepo(path string) string {
	path = strings.TrimSpace(path)
	if path == "" {
		return ""
	}
	root, err := repocontract.FindRepoRootFromPath(path)
	if err != nil {
		return ""
	}
	return filepath.Clean(root)
}

func sourceRootPointer(environment []string) string {
	pointers := []string{strings.TrimSpace(environmentValue(environment, LaunchSourcePointerEnv))}
	if home, err := os.UserHomeDir(); err == nil {
		pointers = append(pointers, repocontract.SourceRootPointerPath(home))
	}
	for _, pointer := range pointers {
		if pointer == "" {
			continue
		}
		contents, err := os.ReadFile(pointer)
		if err != nil {
			continue
		}
		if root := validRepo(strings.TrimSpace(string(contents))); root != "" {
			return root
		}
	}
	return ""
}

func runtimeToolDirs(sourceRoot string, environment []string) []string {
	var dirs []string
	if binary := strings.TrimSpace(environmentValue(environment, LaunchRuntimeBinaryEnv)); binary != "" {
		if info, err := os.Stat(binary); err == nil {
			if info.IsDir() {
				dirs = append(dirs, binary)
			} else {
				dirs = append(dirs, filepath.Dir(binary))
			}
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		home = ""
	}
	if sourceRoot != "" && home != "" {
		if contract, loadErr := repocontract.LoadDefault(sourceRoot); loadErr == nil {
			for _, key := range []string{repocontract.HomeKeyBin, repocontract.HomeKeyShims} {
				if entry, entryErr := contract.RuntimeHomeEntry(home, key); entryErr == nil {
					dirs = append(dirs, entry.AbsPath)
				}
			}
		}
	}
	return existingUniqueDirectories(dirs)
}

func existingUniqueDirectories(dirs []string) []string {
	seen := make(map[string]struct{}, len(dirs))
	result := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dir = strings.TrimSpace(dir)
		if dir == "" {
			continue
		}
		info, err := os.Stat(dir)
		if err != nil || !info.IsDir() {
			continue
		}
		dir = filepath.Clean(dir)
		if _, ok := seen[dir]; ok {
			continue
		}
		seen[dir] = struct{}{}
		result = append(result, dir)
	}
	return result
}

func prependPathEntries(environment, dirs []string) []string {
	if len(dirs) == 0 {
		return environment
	}
	path := environmentValue(environment, "PATH")
	entries := filepath.SplitList(path)
	seen := make(map[string]struct{}, len(entries)+len(dirs))
	for _, entry := range entries {
		seen[pathIdentity(entry)] = struct{}{}
	}
	prefix := make([]string, 0, len(dirs))
	for _, dir := range dirs {
		dir = filepath.Clean(strings.TrimSpace(dir))
		if dir == "" {
			continue
		}
		identity := pathIdentity(dir)
		if _, ok := seen[identity]; ok {
			continue
		}
		seen[identity] = struct{}{}
		prefix = append(prefix, dir)
	}
	if len(prefix) == 0 {
		return environment
	}
	return withEnvironmentValue(environment, "PATH", strings.Join(append(prefix, entries...), string(os.PathListSeparator)))
}

func pathIdentity(path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if runtime.GOOS == "windows" {
		return strings.ToLower(path)
	}
	return path
}

func environmentKeyEqual(left, right string) bool {
	if runtime.GOOS == "windows" {
		return strings.EqualFold(left, right)
	}
	return left == right
}
