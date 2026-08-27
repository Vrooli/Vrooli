package buildinfo

import (
	"crypto/sha256"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"

	platform "github.com/vrooli/platform-go"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/scenarioruntime"
	"github.com/vrooli/vrooli/internal/shell"
)

const (
	// SourceRootEnvVar points to the repository root used for stale checks.
	SourceRootEnvVar = "VROOLI_SOURCE_ROOT"
	// SourceRootFallbackEnvVar falls back to the active Vrooli root when set.
	SourceRootFallbackEnvVar = "VROOLI_ROOT"
	// SourceRootPointerFile is written by the authenticated prebuilt installer so
	// an installed CLI can find its matching source tree outside a checkout.
	SourceRootPointerFile = ".vrooli/source-root"
	// FingerprintPathsEnvVar overrides which relative paths participate in the fingerprint.
	FingerprintPathsEnvVar = "VROOLI_FINGERPRINT_PATHS"
	// BuildTargetEnvVar overrides the go build target used by RebuildAndReexec.
	BuildTargetEnvVar = "VROOLI_BUILD_TARGET"
	// RebuildLoopEnvVar records the fingerprint that triggered the last rebuild.
	RebuildLoopEnvVar = "VROOLI_REBUILD_FINGERPRINT"
	// FingerprintDebugEnvVar enables a per-file fingerprint dump to debugWriter
	// (default os.Stderr) for diagnosing stale-rebuild loops. When set to a
	// truthy value, every ComputeSourceFingerprintReport call emits one line per
	// matched file: "<rel> <size> <sha256-hex>", sorted by relative path.
	FingerprintDebugEnvVar = "VROOLI_FINGERPRINT_DEBUG"
	rebuildTempPrefix      = "vrooli-rebuild-"
	rebuildOrphanHorizon   = time.Hour
)

var (
	// Fingerprint is populated at build time via -ldflags.
	Fingerprint = scenarioruntime.HealthStatusUnknown
	// GitCommit is populated at build time via -ldflags.
	GitCommit = scenarioruntime.HealthStatusUnknown
	// BuildTime is populated at build time via -ldflags.
	BuildTime = scenarioruntime.HealthStatusUnknown
)

var (
	nowFunc          = func() time.Time { return time.Now().UTC() }
	executablePathFn = os.Executable
	homeDirFn        = config.HomeDir
	commandOutputFn  = func(dir, name string, args ...string) ([]byte, error) {
		return shell.Output(shell.Spec{
			Name: name,
			Args: args,
			Dir:  dir,
		})
	}
	goBuildFn = func(dir string, args []string) error {
		return shell.Run(shell.Spec{
			Name:   "go",
			Args:   args,
			Dir:    dir,
			Stdout: os.Stdout,
			Stderr: os.Stderr,
			Stdin:  os.Stdin,
		})
	}
	execFn = platform.ReplaceProcess
	// openFileFn opens lock files; injectable for test fault injection.
	openFileFn = func(name string, flag int, perm os.FileMode) (*os.File, error) {
		return os.OpenFile(name, flag, perm)
	}
	// renameFn commits a prepared file through the canonical owned-write seam;
	// it remains injectable so tests can assert the atomic-install invariant.
	renameFn = replaceManagedFile
	// debugWriter is the destination for VROOLI_FINGERPRINT_DEBUG dumps. Tests
	// override it; production code leaves it pointing at os.Stderr.
	debugWriter io.Writer = os.Stderr
)

var skippedDirs = map[string]struct{}{
	".git":                            {},
	repocontractmeta.ProjectConfigDir: {},
	"build":                           {},
	"coverage":                        {},
	"dist":                            {},
	"node_modules":                    {},
	"tmp":                             {},
}

// FingerprintOptions controls how source fingerprint requests validate their
// target set. The zero value keeps fingerprint checks permissive by default.
type FingerprintOptions struct {
	RequireExistingTargets bool
	RequireGoFiles         bool
}

// FingerprintReport describes the source set that participated in a fingerprint
// calculation. It is primarily intended for build tooling and diagnostics.
type FingerprintReport struct {
	Root           string
	Targets        []string
	MissingTargets []string
	MatchedFiles   int
	Fingerprint    string
}

// StaleCheck describes the outcome of comparing the embedded build fingerprint
// to the current source tree.
type StaleCheck struct {
	Root                string
	Targets             []string
	CurrentFingerprint  string
	EmbeddedFingerprint string
	Stale               bool
}

// MissingTargetsError reports targets that were requested but do not exist
// beneath the repository root.
type MissingTargetsError struct {
	Targets []string
}

func (e MissingTargetsError) Error() string {
	return fmt.Sprintf("missing fingerprint targets: %s", strings.Join(e.Targets, ", "))
}

// NoGoFilesMatchedError reports that the requested targets exist but do not
// include any Go files in the fingerprint set.
type NoGoFilesMatchedError struct {
	Root    string
	Targets []string
}

func (e NoGoFilesMatchedError) Error() string {
	return fmt.Sprintf("no Go files matched beneath root %q for targets %s", e.Root, strings.Join(e.Targets, ", "))
}

type TargetPathErrorReason string

const (
	TargetPathMustBeRelative TargetPathErrorReason = "must_be_relative"
	TargetPathEscapesRoot    TargetPathErrorReason = "escapes_root"
)

// TargetPathError reports invalid target path requests before the filesystem is
// accessed.
type TargetPathError struct {
	Target string
	Root   string
	Reason TargetPathErrorReason
}

func (e TargetPathError) Error() string {
	switch e.Reason {
	case TargetPathMustBeRelative:
		return fmt.Sprintf("target %q must be relative to the repository root", e.Target)
	case TargetPathEscapesRoot:
		return fmt.Sprintf("target %q escapes repository root %s", e.Target, e.Root)
	default:
		return fmt.Sprintf("target %q is invalid", e.Target)
	}
}

// ComputeSourceFingerprint returns a deterministic fingerprint of all Go files
// reachable beneath the provided root directory.
func ComputeSourceFingerprint(rootDir string) (string, error) {
	return ComputeSourceFingerprintForPaths(rootDir)
}

// ComputeSourceFingerprintReport returns a fingerprint plus details about the
// source set that participated in the calculation.
func ComputeSourceFingerprintReport(rootDir string, options FingerprintOptions, relPaths ...string) (FingerprintReport, error) {
	rootDir = strings.TrimSpace(rootDir)
	if rootDir == "" {
		return FingerprintReport{}, errors.New("root directory is required")
	}
	absoluteRoot, err := filepath.Abs(filepath.Clean(rootDir))
	if err != nil {
		return FingerprintReport{}, fmt.Errorf("resolve root directory: %w", err)
	}

	targets := normalizeTargets(relPaths)
	if len(targets) == 0 {
		targets = []string{"."}
	}

	report := FingerprintReport{
		Root:    absoluteRoot,
		Targets: append([]string(nil), targets...),
	}
	entries := make([]fingerprintEntry, 0)
	for _, target := range targets {
		base, err := resolveTargetPath(absoluteRoot, target)
		if err != nil {
			return FingerprintReport{}, err
		}
		info, err := os.Stat(base)
		if err != nil {
			if os.IsNotExist(err) {
				report.MissingTargets = append(report.MissingTargets, target)
				continue
			}
			return FingerprintReport{}, fmt.Errorf("stat %s: %w", target, err)
		}

		if info.IsDir() {
			if err := collectFingerprintEntries(absoluteRoot, base, &entries); err != nil {
				return FingerprintReport{}, err
			}
			continue
		}

		if !strings.HasSuffix(info.Name(), ".go") {
			continue
		}
		entry, err := fingerprintFile(absoluteRoot, base)
		if err != nil {
			return FingerprintReport{}, err
		}
		entries = append(entries, entry)
	}

	if options.RequireExistingTargets && len(report.MissingTargets) > 0 {
		return FingerprintReport{}, MissingTargetsError{Targets: append([]string(nil), report.MissingTargets...)}
	}

	sort.Slice(entries, func(i, j int) bool {
		return entries[i].rel < entries[j].rel
	})

	report.MatchedFiles = len(entries)
	hasher := sha256.New()
	for _, entry := range entries {
		fmt.Fprintf(hasher, "%s|%d|%x\n", entry.rel, entry.size, entry.hash)
	}
	report.Fingerprint = fmt.Sprintf("%x", hasher.Sum(nil))

	if isFingerprintDebugEnabled() {
		dumpFingerprintEntries(debugWriter, report.Root, report.Fingerprint, entries)
	}

	if options.RequireGoFiles && report.MatchedFiles == 0 {
		return FingerprintReport{}, NoGoFilesMatchedError{
			Root:    report.Root,
			Targets: append([]string(nil), report.Targets...),
		}
	}

	return report, nil
}

// ComputeSourceFingerprintForPaths returns a deterministic fingerprint of all Go
// files beneath the provided relative paths. When no paths are provided, the
// entire root is scanned.
func ComputeSourceFingerprintForPaths(rootDir string, relPaths ...string) (string, error) {
	report, err := ComputeSourceFingerprintReport(rootDir, FingerprintOptions{}, relPaths...)
	if err != nil {
		return "", err
	}
	return report.Fingerprint, nil
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

	executable, err := executablePathFn()
	if err != nil {
		return "", fmt.Errorf("resolve executable: %w", err)
	}
	if root, ok := findModuleRoot(filepath.Dir(executable)); ok {
		return root, nil
	}

	if home, homeErr := homeDirFn(); homeErr == nil {
		pointer := filepath.Join(home, filepath.FromSlash(SourceRootPointerFile))
		if contents, readErr := os.ReadFile(pointer); readErr == nil {
			candidate := filepath.Clean(strings.TrimSpace(string(contents)))
			if root, ok := findModuleRoot(candidate); ok && root == candidate {
				return root, nil
			}
		}

		// Local development installs historically did not write the source
		// pointer. Make the bootstrap path forgiving, while requiring strong
		// repository identity so a random Go module in the home directory is
		// never selected as Vrooli's source root.
		if root, ok := findVrooliSourceRootFromHome(home); ok {
			return root, nil
		}
	}

	return "", errors.New("unable to resolve source root")
}

const (
	vrooliModulePath       = "github.com/vrooli/vrooli"
	maxHomeDiscoveryDepth  = 4
	sourceRootMainFilePath = "cmd/vrooli/main.go"
)

// findVrooliSourceRootFromHome finds an existing checkout below the invoking
// user's home directory. This is intentionally bounded and identity-checked:
// it provides a useful bootstrap fallback without searching arbitrary mounts
// or accepting an unrelated Go repository.
func findVrooliSourceRootFromHome(home string) (string, bool) {
	home = filepath.Clean(strings.TrimSpace(home))
	if home == "" || home == "." {
		return "", false
	}

	var found string
	err := filepath.WalkDir(home, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if entry == nil {
			return nil
		}

		rel, relErr := filepath.Rel(home, path)
		if relErr != nil {
			return nil
		}
		depth := 0
		if rel != "." {
			depth = strings.Count(filepath.ToSlash(rel), "/") + 1
		}
		if depth > maxHomeDiscoveryDepth {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if rel != "." && entry.IsDir() {
			name := entry.Name()
			if strings.HasPrefix(name, ".") || name == "node_modules" || name == "go" {
				return filepath.SkipDir
			}
		}
		if entry.IsDir() && isVrooliSourceRoot(path) {
			found = filepath.Clean(path)
			return errSourceRootFound
		}
		return nil
	})
	if err != nil && !errors.Is(err, errSourceRootFound) {
		return "", false
	}
	return found, found != ""
}

var errSourceRootFound = errors.New("source root found")

func isVrooliSourceRoot(root string) bool {
	data, err := os.ReadFile(filepath.Join(root, "go.mod"))
	if err != nil {
		return false
	}
	moduleLine := "module " + vrooliModulePath
	moduleFound := false
	for _, line := range strings.Split(string(data), "\n") {
		if strings.TrimSpace(line) == moduleLine {
			moduleFound = true
			break
		}
	}
	if !moduleFound {
		return false
	}
	info, err := os.Stat(filepath.Join(root, filepath.FromSlash(sourceRootMainFilePath)))
	return err == nil && !info.IsDir()
}

// CurrentFingerprint computes the fingerprint for the current binary's source set.
func CurrentFingerprint() (string, error) {
	report, err := CurrentFingerprintReport()
	if err != nil {
		return "", err
	}
	return report.Fingerprint, nil
}

// CurrentFingerprintReport computes the current binary's source fingerprint and
// returns metadata about the source set used for the calculation.
func CurrentFingerprintReport() (FingerprintReport, error) {
	root, err := ResolveSourceRoot()
	if err != nil {
		return FingerprintReport{}, err
	}

	targets, err := fingerprintTargets()
	if err != nil {
		return FingerprintReport{}, err
	}
	return ComputeSourceFingerprintReport(root, FingerprintOptions{
		RequireExistingTargets: true,
		RequireGoFiles:         true,
	}, targets...)
}

// FingerprintTargetsForExecutable returns the canonical source targets used to
// decide whether a project-level binary is fresh. Build and distribution tools
// must use this function rather than maintaining a second target list, otherwise
// a transferred sidecar can never be authoritative on the destination host.
func FingerprintTargetsForExecutable(executableName string) []string {
	executableName = strings.TrimSuffix(filepath.Base(strings.TrimSpace(executableName)), ".exe")
	switch executableName {
	case "vrooli-api":
		return []string{"cmd/vrooli-api", "internal"}
	case "vrooli":
		return []string{"cmd/vrooli", "internal"}
	default:
		return nil
	}
}

// IsStale returns true when the embedded fingerprint differs from current sources.
func IsStale() bool {
	status, err := CheckStaleness()
	if err != nil {
		return false
	}
	return status.Stale
}

// CheckStaleness compares the embedded build fingerprint to the current source
// fingerprint and returns the comparison result plus diagnostic metadata.
//
// A `<executable>.fp` sidecar, if present and at least as new as the
// executable, can short-circuit the embedded compare — this lets a freshly
// rebuilt binary be recognized as fresh by sibling processes whose embedded
// symbol still reflects the old (in-memory) build. The embedded fingerprint
// remains authoritative when no usable sidecar is present.
func CheckStaleness() (StaleCheck, error) {
	status := StaleCheck{
		EmbeddedFingerprint: strings.TrimSpace(Fingerprint),
	}

	report, err := CurrentFingerprintReport()
	if err != nil {
		return StaleCheck{}, err
	}
	status.Root = report.Root
	status.Targets = append([]string(nil), report.Targets...)
	status.CurrentFingerprint = report.Fingerprint

	if sidecarMatches(status.CurrentFingerprint) {
		status.Stale = false
		return status, nil
	}

	status.Stale = status.EmbeddedFingerprint == "" ||
		status.EmbeddedFingerprint == scenarioruntime.HealthStatusUnknown ||
		status.CurrentFingerprint != status.EmbeddedFingerprint
	return status, nil
}

// sidecarMatches returns true when a `<executable>.fp` sidecar exists, equals
// currentFingerprint, and was written no earlier than the executable's mtime
// (so a developer's manual `go build` doesn't get falsely treated as fresh).
func sidecarMatches(currentFingerprint string) bool {
	executable, err := executablePathFn()
	if err != nil || executable == "" {
		return false
	}
	return SidecarMatches(executable, currentFingerprint)
}

// SidecarMatches reports whether executable.fp contains the supplied source
// fingerprint and is not older than the executable. Taking the executable path
// explicitly lets build/transfer tooling validate an artifact before executing
// it, while CheckStaleness continues to use the current process executable.
func SidecarMatches(executable, currentFingerprint string) bool {
	sidecar := executable + ".fp"
	sidecarInfo, err := os.Stat(sidecar)
	if err != nil {
		return false
	}
	binInfo, err := os.Stat(executable)
	if err == nil && sidecarInfo.ModTime().Before(binInfo.ModTime()) {
		return false
	}
	contents, err := os.ReadFile(sidecar)
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(contents)) == currentFingerprint
}

// WriteSidecarFingerprint writes <executable>.fp atomically via a temp file and
// replacement. Its mtime is clamped to at least the binary mtime so copied or
// reproducibly timestamped binaries remain fresh even under clock skew.
func WriteSidecarFingerprint(executable, fingerprint string) error {
	sidecar := executable + ".fp"
	tmp := fmt.Sprintf("%s.tmp.%d", sidecar, os.Getpid())
	defer os.Remove(tmp)
	if err := os.WriteFile(tmp, []byte(fingerprint+"\n"), tuning.PermFile); err != nil {
		return err
	}
	if err := renameFn(tmp, sidecar); err != nil {
		_ = os.Remove(tmp)
		return err
	}
	mtime := nowFunc()
	if info, err := os.Stat(executable); err == nil && info.ModTime().After(mtime) {
		mtime = info.ModTime()
	}
	if err := os.Chtimes(sidecar, mtime, mtime); err != nil {
		return err
	}
	return nil
}

func replaceManagedFile(source, destination string) error {
	info, err := os.Stat(source)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(source)
	if err != nil {
		return err
	}
	return config.WriteOwnedFileAtomic(destination, data, info.Mode().Perm())
}

// RebuildAndReexec rebuilds the current binary from source and re-execs it.
//
// Concurrency: a host-wide flock at <executable>.lock serializes concurrent
// rebuilders so two sibling processes (e.g. autoheal subprocesses, or any two
// `vrooli` invocations on a stale tree) cannot race on `go build -o
// <executable>`. The install is atomic via a temp file and replacement, so an
// in-flight exec on the executable always sees a complete binary.
func RebuildAndReexec(argv []string) error {
	root, err := ResolveSourceRoot()
	if err != nil {
		return err
	}

	executable, err := executablePathFn()
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

	release, err := acquireRebuildLock(executable)
	if err != nil {
		return fmt.Errorf("acquire rebuild lock: %w", err)
	}
	defer release()

	buildDir, err := prepareRebuildDirectory(executable)
	if err != nil {
		return err
	}

	// After acquiring the lock, re-check whether a sibling rebuilder already
	// landed a fresh binary at the same fingerprint. The sidecar is the shared
	// on-disk signal; this process's embedded fingerprint may still be stale.
	// If either signal proves freshness, skip the rebuild and exec straight into
	// the installed binary.
	if strings.TrimSpace(Fingerprint) == currentFingerprint || SidecarMatches(executable, currentFingerprint) {
		execArgs := append([]string{executable}, argv...)
		return execFn(executable, execArgs, setEnvValue(os.Environ(), RebuildLoopEnvVar, currentFingerprint))
	}

	gitCommit := strings.TrimSpace(GitCommit)
	if gitCommit == "" || gitCommit == scenarioruntime.HealthStatusUnknown {
		if output, cmdErr := commandOutputFn(root, "git", "rev-parse", "HEAD"); cmdErr == nil {
			gitCommit = strings.TrimSpace(string(output))
		}
	}
	if gitCommit == "" {
		gitCommit = scenarioruntime.HealthStatusUnknown
	}

	buildTime := nowFunc().Format(time.RFC3339)
	ldflags := fmt.Sprintf(
		"-s -w -X %s=%s -X %s=%s -X %s=%s",
		"github.com/vrooli/vrooli/internal/buildinfo.Fingerprint", currentFingerprint,
		"github.com/vrooli/vrooli/internal/buildinfo.GitCommit", gitCommit,
		"github.com/vrooli/vrooli/internal/buildinfo.BuildTime", buildTime,
	)

	tempFile, err := os.CreateTemp(buildDir, rebuildTempPrefix)
	if err != nil {
		return fmt.Errorf("create rebuild output: %w", err)
	}
	tempPath := tempFile.Name()
	if err := tempFile.Close(); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("close rebuild output: %w", err)
	}
	buildArgs := []string{"build", "-trimpath", "-ldflags", ldflags, "-o", tempPath, buildTarget}
	if err := goBuildFn(root, buildArgs); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("rebuild %s: %w", buildTarget, err)
	}
	_ = PreserveRootBinaryFallback(executable)
	if err := config.InstallExecutableAtomic(tempPath, executable); err != nil {
		_ = os.Remove(tempPath)
		return fmt.Errorf("install rebuilt binary %s: %w", executable, err)
	}
	if err := os.Remove(tempPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove rebuild output %s: %w", tempPath, err)
	}
	// Sidecar is a strict optimization for sibling processes whose embedded
	// symbol still reflects the pre-rebuild fingerprint. Failures are
	// non-fatal: the embedded-fingerprint compare remains authoritative.
	_ = WriteSidecarFingerprint(executable, currentFingerprint)

	execArgs := append([]string{executable}, argv...)
	return execFn(executable, execArgs, setEnvValue(os.Environ(), RebuildLoopEnvVar, currentFingerprint))
}

func prepareRebuildDirectory(executable string) (string, error) {
	home, err := homeDirFn()
	if err != nil {
		return "", fmt.Errorf("resolve rebuild home: %w", err)
	}
	buildDir, err := repocontract.RuntimeHomeEntryPath(home, repocontract.HomeKeyBuild)
	if err != nil {
		return "", fmt.Errorf("resolve rebuild directory: %w", err)
	}
	if _, err := config.EnsureOwnedDir(buildDir); err != nil {
		return "", fmt.Errorf("ensure rebuild directory: %w", err)
	}
	if err := sweepRebuildOrphans(buildDir, executable, nowFunc()); err != nil {
		return "", fmt.Errorf("sweep rebuild orphans: %w", err)
	}
	return buildDir, nil
}

func sweepRebuildOrphans(buildDir, executable string, now time.Time) error {
	entries, err := os.ReadDir(buildDir)
	if err != nil {
		return err
	}
	for _, entry := range entries {
		if !strings.HasPrefix(entry.Name(), rebuildTempPrefix) {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() || now.Sub(info.ModTime()) < rebuildOrphanHorizon {
			continue
		}
		if err := os.Remove(filepath.Join(buildDir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}

	executableDir := filepath.Dir(executable)
	legacyPrefix := filepath.Base(executable) + ".tmp."
	legacyEntries, err := os.ReadDir(executableDir)
	if err != nil {
		return err
	}
	for _, entry := range legacyEntries {
		suffix, ok := strings.CutPrefix(entry.Name(), legacyPrefix)
		if !ok || suffix == "" || strings.Trim(suffix, "0123456789") != "" {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		if !info.Mode().IsRegular() {
			continue
		}
		if err := os.Remove(filepath.Join(executableDir, entry.Name())); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

func isFingerprintDebugEnabled() bool {
	v := strings.TrimSpace(os.Getenv(FingerprintDebugEnvVar))
	if v == "" {
		return false
	}
	switch strings.ToLower(v) {
	case "0", "false", "no", "off":
		return false
	}
	return true
}

func dumpFingerprintEntries(w io.Writer, root, fingerprint string, entries []fingerprintEntry) {
	if w == nil {
		return
	}
	fmt.Fprintf(w, "# fingerprint=%s root=%s files=%d\n", fingerprint, root, len(entries))
	for _, e := range entries {
		fmt.Fprintf(w, "%s %d %x\n", e.rel, e.size, e.hash)
	}
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

func resolveTargetPath(rootDir, target string) (string, error) {
	if filepath.IsAbs(target) {
		return "", TargetPathError{Target: target, Reason: TargetPathMustBeRelative}
	}

	base := filepath.Join(rootDir, filepath.FromSlash(target))
	absoluteBase, err := filepath.Abs(base)
	if err != nil {
		return "", fmt.Errorf("resolve target %s: %w", target, err)
	}

	relToRoot, err := filepath.Rel(rootDir, absoluteBase)
	if err != nil {
		return "", fmt.Errorf("resolve target %s: %w", target, err)
	}
	relToRoot = filepath.ToSlash(relToRoot)
	if relToRoot == ".." || strings.HasPrefix(relToRoot, "../") {
		return "", TargetPathError{Target: target, Root: rootDir, Reason: TargetPathEscapesRoot}
	}

	return absoluteBase, nil
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

	executable, err := executablePathFn()
	if err != nil {
		return nil, fmt.Errorf("resolve executable: %w", err)
	}

	return FingerprintTargetsForExecutable(executable), nil
}

func buildTargetForExecutable(executable string) (string, error) {
	if override := strings.TrimSpace(os.Getenv(BuildTargetEnvVar)); override != "" {
		return override, nil
	}

	executable = strings.TrimSpace(executable)
	if executable == "" {
		return "", errors.New("unable to infer build target from executable path")
	}

	name := filepath.Base(executable)
	if name == "" || name == "." {
		return "", errors.New("unable to infer build target from executable path")
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
