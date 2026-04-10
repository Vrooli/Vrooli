package buildinfo

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"syscall"
	"time"
)

const (
	// SourceRootEnvVar points to the repository root used for stale checks.
	SourceRootEnvVar = "VROOLI_SOURCE_ROOT"
	// SourceRootFallbackEnvVar falls back to the active Vrooli root when set.
	SourceRootFallbackEnvVar = "VROOLI_ROOT"
	// FingerprintPathsEnvVar overrides which relative paths participate in the fingerprint.
	FingerprintPathsEnvVar = "VROOLI_FINGERPRINT_PATHS"
	// BuildTargetEnvVar overrides the go build target used by RebuildAndReexec.
	BuildTargetEnvVar = "VROOLI_BUILD_TARGET"
	// RebuildLoopEnvVar records the fingerprint that triggered the last rebuild.
	RebuildLoopEnvVar = "VROOLI_REBUILD_FINGERPRINT"
)

var (
	// Fingerprint is populated at build time via -ldflags.
	Fingerprint = "unknown"
	// GitCommit is populated at build time via -ldflags.
	GitCommit = "unknown"
	// BuildTime is populated at build time via -ldflags.
	BuildTime = "unknown"
)

var (
	nowFunc         = func() time.Time { return time.Now().UTC() }
	commandOutputFn = func(dir, name string, args ...string) ([]byte, error) {
		cmd := exec.Command(name, args...)
		cmd.Dir = dir
		return cmd.Output()
	}
	goBuildFn = func(dir string, args []string) error {
		cmd := exec.Command("go", args...)
		cmd.Dir = dir
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Stdin = os.Stdin
		return cmd.Run()
	}
	execFn = func(argv0 string, argv []string, envv []string) error {
		return syscall.Exec(argv0, argv, envv)
	}
)

var skippedDirs = map[string]struct{}{
	".git":         {},
	".vrooli":      {},
	"build":        {},
	"coverage":     {},
	"dist":         {},
	"node_modules": {},
	"tmp":          {},
}

// ComputeSourceFingerprint returns a deterministic fingerprint of all Go files
// reachable beneath the provided root directory.
func ComputeSourceFingerprint(rootDir string) (string, error) {
	return ComputeSourceFingerprintForPaths(rootDir)
}

// ComputeSourceFingerprintForPaths returns a deterministic fingerprint of all Go
// files beneath the provided relative paths. When no paths are provided, the
// entire root is scanned.
func ComputeSourceFingerprintForPaths(rootDir string, relPaths ...string) (string, error) {
	rootDir = filepath.Clean(rootDir)
	if rootDir == "" {
		return "", errors.New("root directory is required")
	}

	targets := normalizeTargets(relPaths)
	if len(targets) == 0 {
		targets = []string{"."}
	}

	entries := make([]fingerprintEntry, 0)
	for _, target := range targets {
		base := filepath.Join(rootDir, filepath.FromSlash(target))
		info, err := os.Stat(base)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return "", fmt.Errorf("stat %s: %w", target, err)
		}

		if info.IsDir() {
			if err := collectFingerprintEntries(rootDir, base, &entries); err != nil {
				return "", err
			}
			continue
		}

		if !strings.HasSuffix(info.Name(), ".go") {
			continue
		}
		entry, err := fingerprintFile(rootDir, base)
		if err != nil {
			return "", err
		}
		entries = append(entries, entry)
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rel < entries[j].rel
	})

	hasher := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(hasher, "%s|%d|%x\n", entry.rel, entry.size, entry.hash)
	}

	return fmt.Sprintf("%x", hasher.Sum(nil)), nil
}

// ResolveSourceRoot returns the repository root used for source fingerprinting.
func ResolveSourceRoot() (string, error) {
	for _, key := range []string{SourceRootEnvVar, SourceRootFallbackEnvVar} {
		if value := strings.TrimSpace(os.Getenv(key)); value != "" {
			return filepath.Clean(value), nil
		}
	}

	if cwd, err := os.Getwd(); err == nil {
		if root, ok := findModuleRoot(cwd); ok {
			return root, nil
		}
	}

	executable, err := os.Executable()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	if root, ok := findModuleRoot(filepath.Dir(executable)); ok {
		return root, nil
	}

	return "", errors.New("unable to resolve source root")
}

// CurrentFingerprint computes the fingerprint for the current binary's source set.
func CurrentFingerprint() (string, error) {
	root, err := ResolveSourceRoot()
	if err != nil {
		return "", err
	}

	targets, err := fingerprintTargets()
	if err != nil {
		return "", err
	}
	return ComputeSourceFingerprintForPaths(root, targets...)
}

// IsStale returns true when the embedded fingerprint differs from current sources.
func IsStale() bool {
	if strings.TrimSpace(Fingerprint) == "" || Fingerprint == "unknown" {
		return false
	}

	current, err := CurrentFingerprint()
	if err != nil {
		return false
	}

	return current != Fingerprint
}

// RebuildAndReexec rebuilds the current binary from source and re-execs it.
func RebuildAndReexec(argv []string) error {
	root, err := ResolveSourceRoot()
	if err != nil {
		return err
	}

	executable, err := os.Executable()
	if err != nil {
		return fmt.Errorf("resolve executable: %w", err)
	}

	buildTarget, err := buildTargetForExecutable(executable)
	if err != nil {
		return err
	}

	currentFingerprint, err := CurrentFingerprint()
	if err != nil {
		return fmt.Errorf("compute current fingerprint: %w", err)
	}
	if previousFingerprint := strings.TrimSpace(os.Getenv(RebuildLoopEnvVar)); previousFingerprint == currentFingerprint {
		return fmt.Errorf("rebuild loop detected for fingerprint %s", currentFingerprint)
	}

	gitCommit := strings.TrimSpace(GitCommit)
	if gitCommit == "" || gitCommit == "unknown" {
		if output, cmdErr := commandOutputFn(root, "git", "rev-parse", "HEAD"); cmdErr == nil {
			gitCommit = strings.TrimSpace(string(output))
		}
	}
	if gitCommit == "" {
		gitCommit = "unknown"
	}

	buildTime := nowFunc().Format(time.RFC3339)
	ldflags := fmt.Sprintf(
		"-s -w -X %s=%s -X %s=%s -X %s=%s",
		"github.com/vrooli/vrooli/internal/buildinfo.Fingerprint", currentFingerprint,
		"github.com/vrooli/vrooli/internal/buildinfo.GitCommit", gitCommit,
		"github.com/vrooli/vrooli/internal/buildinfo.BuildTime", buildTime,
	)

	buildArgs := []string{"build", "-trimpath", "-ldflags", ldflags, "-o", executable, buildTarget}
	if err := goBuildFn(root, buildArgs); err != nil {
		return fmt.Errorf("rebuild %s: %w", buildTarget, err)
	}

	execArgs := append([]string{executable}, argv...)
	return execFn(executable, execArgs, setEnvValue(os.Environ(), RebuildLoopEnvVar, currentFingerprint))
}

type fingerprintEntry struct {
	rel  string
	size int64
	hash [32]byte
}

func collectFingerprintEntries(rootDir, start string, entries *[]fingerprintEntry) error {
	return filepath.WalkDir(start, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil {
			return relErr
		}
		rel = filepath.ToSlash(rel)
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			if shouldSkipDir(rel) {
				return filepath.SkipDir
			}
			return nil
		}

		if !strings.HasSuffix(rel, ".go") {
			return nil
		}

		entry, fileErr := fingerprintFile(rootDir, path)
		if fileErr != nil {
			return fileErr
		}
		*entries = append(*entries, entry)
		return nil
	})
}

func fingerprintFile(rootDir, path string) (fingerprintEntry, error) {
	info, err := os.Stat(path)
	if err != nil {
		return fingerprintEntry{}, err
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return fingerprintEntry{}, fmt.Errorf("read %s: %w", path, err)
	}

	rel, err := filepath.Rel(rootDir, path)
	if err != nil {
		return fingerprintEntry{}, err
	}

	return fingerprintEntry{
		rel:  filepath.ToSlash(rel),
		size: info.Size(),
		hash: sha256.Sum256(content),
	}, nil
}

func normalizeTargets(paths []string) []string {
	if len(paths) == 0 {
		return nil
	}

	set := make(map[string]struct{}, len(paths))
	targets := make([]string, 0, len(paths))
	for _, candidate := range paths {
		candidate = filepath.ToSlash(strings.TrimSpace(candidate))
		candidate = strings.TrimPrefix(candidate, "./")
		if candidate == "" {
			continue
		}
		if _, exists := set[candidate]; exists {
			continue
		}
		set[candidate] = struct{}{}
		targets = append(targets, candidate)
	}
	sort.Strings(targets)
	return targets
}

func shouldSkipDir(rel string) bool {
	rel = filepath.ToSlash(rel)
	parts := strings.Split(rel, "/")
	for _, part := range parts {
		if _, skip := skippedDirs[part]; skip {
			return true
		}
	}
	return false
}

func findModuleRoot(start string) (string, bool) {
	dir := filepath.Clean(start)
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, true
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", false
		}
		dir = parent
	}
}

func fingerprintTargets() ([]string, error) {
	if value := strings.TrimSpace(os.Getenv(FingerprintPathsEnvVar)); value != "" {
		parts := strings.Split(value, ",")
		targets := normalizeTargets(parts)
		if len(targets) == 0 {
			return nil, errors.New("no fingerprint paths configured")
		}
		return targets, nil
	}

	executable, err := os.Executable()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}

	switch filepath.Base(executable) {
	case "vrooli-api":
		return []string{"cmd/vrooli-api", "internal"}, nil
	case "vrooli":
		return []string{"cmd/vrooli", "internal"}, nil
	default:
		return nil, nil
	}
}

func buildTargetForExecutable(executable string) (string, error) {
	if override := strings.TrimSpace(os.Getenv(BuildTargetEnvVar)); override != "" {
		return override, nil
	}

	name := filepath.Base(executable)
	if name == "" {
		return "", errors.New("unable to infer build target from executable name")
	}

	return "./cmd/" + name, nil
}

func setEnvValue(env []string, key, value string) []string {
	prefix := key + "="
	for i, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			updated := append([]string(nil), env...)
			updated[i] = prefix + value
			return updated
		}
	}
	return append(append([]string(nil), env...), prefix+value)
}
