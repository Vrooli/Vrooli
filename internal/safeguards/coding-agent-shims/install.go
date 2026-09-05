package codingagentshims

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/artifactledger"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/repocontractmeta"
	"github.com/vrooli/vrooli/internal/tuning"
)

// launcherBinary is the multi-call binary every alias links to.
const launcherBinary = "vrooli-agent-launcher"

// runtimeHomeDirName mirrors the repo contract's runtime_home.dir_name.
//
// It is duplicated here rather than loaded, because this code runs on hosts
// that have no checkout -- the whole point of a shim is to work wherever the
// operator is. conformance_test.go asserts the two agree, so drift is a test
// failure rather than a silently wrong path.
const runtimeHomeDirName = repocontractmeta.ProjectConfigDir

// shimsDirName and binDirName mirror the contract's `shims` and `bin` entries.
const (
	shimsDirName = "shims"
	binDirName   = "bin"
)

var shimHomeDir = func() (string, error) {
	if hostreqkit.RunningAsRootFn() {
		return hostreqkit.InvokingUserHomeDir()
	}
	return os.UserHomeDir()
}

// ShimDir returns the directory the aliases are installed into.
func ShimDir() (string, error) {
	home, err := shimHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, runtimeHomeDirName, shimsDirName), nil
}

// LegacyShimDir returns the shared install root the aliases used to live in.
//
// They were moved out of it because a directory cannot be honestly declared
// twice: the shims are five regenerable links, and bin is a shared install root
// holding gigabytes of other components' build output. Declaring the shims'
// budget over bin put a 64MiB ceiling on that shared root, which is a licence to
// prune it that only a hard-coded guard was standing in the way of.
func LegacyShimDir() (string, error) {
	home, err := shimHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, runtimeHomeDirName, binDirName), nil
}

// LauncherPath returns the multi-call binary the aliases resolve to. It stays
// in the install root: it is a built artifact like every other CLI there.
func LauncherPath() (string, error) {
	home, err := shimHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, runtimeHomeDirName, binDirName, executableName(launcherBinary)), nil
}

// executableName appends the platform's executable suffix.
func executableName(name string) string {
	if hostreqspec.PlatformFromGOOS(runtime.GOOS) == hostreqspec.PlatformWindows {
		return name + ".exe"
	}
	return name
}

// AliasPaths returns every alias path to install, keyed by alias.
func AliasPaths() (map[string]string, error) {
	dir, err := ShimDir()
	if err != nil {
		return nil, err
	}
	aliases := make(map[string]string, len(cliutil.CodingAgentAliases()))
	for _, alias := range cliutil.CodingAgentAliases() {
		aliases[alias] = filepath.Join(dir, executableName(alias))
	}
	return aliases, nil
}

// MissingAliases returns the aliases that are absent or no longer point at
// launcher, in stable order.
//
// This is the whole of the "is it installed" question, and it is deliberately
// cheap: one Lstat per alias on Unix. The startup self-heal calls it on every
// vrooli invocation, so it must cost microseconds.
func MissingAliases(aliases map[string]string, launcher string) []string {
	missing := make([]string, 0, len(aliases))
	for _, alias := range cliutil.CodingAgentAliases() {
		if !aliasInstalled(aliases[alias], launcher) {
			missing = append(missing, alias)
		}
	}
	return missing
}

// aliasInstalled reports whether path already routes to launcher.
//
// On Unix an alias is a symlink and the link target answers it outright. On
// Windows it is a hard link, so identity is asked of the filesystem with
// os.SameFile -- which is also what makes a stale alias detectable after the
// launcher is rebuilt, because a rebuild replaces the file rather than writing
// through it.
//
// The byte comparison is the last resort for a host where hard links are
// unavailable and the alias had to be a plain copy. Without it, such a host
// would report every alias stale on every inspection and reinstall them
// forever. It is guarded by a size check so the common paths never read the
// binary at all: the previous implementation compared ~6.4MB on both sides for
// each of five aliases, 64MB of I/O to answer a question a stat answers.
func aliasInstalled(path, launcher string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if hostreqspec.PlatformFromGOOS(runtime.GOOS) != hostreqspec.PlatformWindows {
		if info.Mode()&os.ModeSymlink == 0 {
			return false
		}
		target, err := os.Readlink(path)
		if err != nil {
			return false
		}
		if !filepath.IsAbs(target) {
			target = filepath.Join(filepath.Dir(path), target)
		}
		return filepath.Clean(target) == filepath.Clean(launcher)
	}

	launcherInfo, err := os.Stat(launcher)
	if err != nil {
		return false
	}
	aliasInfo, err := os.Stat(path)
	if err != nil {
		return false
	}
	if os.SameFile(aliasInfo, launcherInfo) {
		return true
	}
	if aliasInfo.Size() != launcherInfo.Size() {
		return false
	}
	return sameFileContents(path, launcher)
}

func sameFileContents(left, right string) bool {
	leftData, err := os.ReadFile(left)
	if err != nil {
		return false
	}
	rightData, err := os.ReadFile(right)
	if err != nil {
		return false
	}
	return bytes.Equal(leftData, rightData)
}

// InstallAlias creates or repairs one alias.
//
// The replacement is atomic: the new link is created under a temporary name in
// the same directory and renamed over the target, because rename replaces in
// one step on POSIX and through MOVEFILE_REPLACE_EXISTING on Windows. The
// previous implementation removed the target and then created it, which left a
// window in which the alias did not exist at all -- and if anything went wrong
// in between, left it permanently absent. On Windows it also failed outright
// when the alias being replaced was running, because a file in use cannot be
// unlinked but can be renamed over.
func InstallAlias(path, launcher string) error {
	// EnsureOwnedDir, not MkdirAll: under `sudo vrooli setup` a plain MkdirAll
	// leaves a root-owned directory in the operator's home, and the per-user
	// self-heal that runs on every later vrooli invocation would then be unable
	// to write into it -- a safeguard that repairs itself only while elevated is
	// no better than one that never repairs itself.
	if _, err := config.EnsureOwnedDir(filepath.Dir(path)); err != nil {
		return fmt.Errorf("create shim directory: %w", err)
	}
	staging := path + ".new-" + processSuffix()
	// A staging file left by a previous crash must not block this attempt.
	if err := os.Remove(staging); err != nil && !errors.Is(err, fs.ErrNotExist) {
		return fmt.Errorf("clear staging path %s: %w", staging, err)
	}
	if err := materializeAlias(staging, launcher); err != nil {
		return err
	}
	if err := config.CommitOwnedPathAtomic(staging, path); err != nil {
		_ = os.Remove(staging)
		return fmt.Errorf("install shim %s: %w", path, err)
	}
	return nil
}

// materializeAlias creates the alias content at staging.
//
// Unix gets a symlink. Windows gets a hard link, falling back to a copy: a
// symlink there needs a privilege this safeguard deliberately does not ask for,
// while a hard link needs none on NTFS and costs no additional bytes. The
// fallback matters for filesystems that carry no hard links at all.
func materializeAlias(staging, launcher string) error {
	if hostreqspec.PlatformFromGOOS(runtime.GOOS) != hostreqspec.PlatformWindows {
		if err := os.Symlink(launcher, staging); err != nil {
			return fmt.Errorf("link shim %s: %w", staging, err)
		}
		return nil
	}
	if err := os.Link(launcher, staging); err == nil {
		return nil
	}
	data, err := os.ReadFile(launcher)
	if err != nil {
		return fmt.Errorf("read launcher: %w", err)
	}
	if err := os.WriteFile(staging, data, tuning.PermExecutable); err != nil {
		return fmt.Errorf("copy shim %s: %w", staging, err)
	}
	return nil
}

// processSuffix names staging files uniquely per process so two concurrent
// installers cannot collide on one temporary path.
func processSuffix() string {
	return fmt.Sprintf("%d", os.Getpid())
}

// EnsureInstalled re-asserts the alias set and reports which aliases it had to
// create. It is safe to call concurrently and on every process start.
//
// This is what makes the shims survive an install root that cannot be relied
// on. Installing once at setup time assumes nothing ever removes them; the
// alias set is five links reconstructible from a binary that is already on
// disk, so re-asserting is cheaper than depending on that assumption. A host
// with everything in place pays five Lstat calls.
func EnsureInstalled() ([]string, error) {
	launcher, err := LauncherPath()
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(launcher); err != nil {
		// Nothing to link to. Before the first build this is the normal state,
		// so it is not an error the caller should act on.
		return nil, nil
	}
	aliases, err := AliasPaths()
	if err != nil {
		return nil, err
	}
	missing := MissingAliases(aliases, launcher)
	installed := make([]string, 0, len(missing))
	for _, alias := range missing {
		if err := InstallAlias(aliases[alias], launcher); err != nil {
			return installed, err
		}
		installed = append(installed, alias)
	}
	return installed, nil
}

// legacyPredicate is the rule recorded on every legacy-alias receipt.
const legacyPredicate = "coding-agent alias superseded by the same alias in the dedicated shim directory"

// RemoveLegacyAliases deletes aliases left in the shared install root by the
// version of this safeguard that installed them there.
//
// They are removed rather than left in place because two aliases for one agent,
// in two directories both on PATH, is a state no operator can reason about: the
// live one depends on PATH order that another profile snippet may change. The
// removal goes through the ledger for the same reason every other removal in
// that directory does -- an unrecorded deletion there is exactly what made the
// original disappearance unattributable.
//
// Only links that actually resolve to the launcher are touched. An operator's
// own binary that happens to share the name is not this safeguard's to remove.
func RemoveLegacyAliases(ledger *artifactledger.Ledger) ([]string, error) {
	legacyDir, err := LegacyShimDir()
	if err != nil {
		return nil, err
	}
	launcher, err := LauncherPath()
	if err != nil {
		return nil, err
	}
	removed := make([]string, 0, len(cliutil.CodingAgentAliases()))
	for _, alias := range cliutil.CodingAgentAliases() {
		path := filepath.Join(legacyDir, executableName(alias))
		if !aliasInstalled(path, launcher) {
			continue
		}
		remove := func() error { return os.Remove(path) }
		if ledger != nil {
			err = ledger.Guard(artifactledger.Removal{
				Path:      path,
				Kind:      "coding-agent-shim",
				Component: "codingagentshims.RemoveLegacyAliases",
				Predicate: legacyPredicate,
			}, remove)
		} else {
			err = remove()
		}
		switch {
		case err == nil:
			removed = append(removed, alias)
		case errors.Is(err, fs.ErrNotExist):
			// Already gone is the outcome this wanted.
		default:
			return removed, fmt.Errorf("remove legacy shim %s: %w", path, err)
		}
	}
	return removed, nil
}

// shadowingNote warns when the shim directory is not ahead of the real agents
// on PATH. The shims are installed correctly in that case but never run, and an
// operator would otherwise see "installed" and wrong attribution at once.
func shadowingNote(shimDir string) []string {
	notes := make([]string, 0, 1)
	for _, alias := range cliutil.CodingAgentAliases() {
		resolved, err := cliutil.ResolveAgentBinaryExcluding(alias, "")
		if err != nil {
			continue
		}
		if filepath.Clean(filepath.Dir(resolved)) == filepath.Clean(shimDir) {
			continue
		}
		if !pathPrefersShimDir(shimDir, filepath.Dir(resolved)) {
			notes = append(notes, fmt.Sprintf(
				"%s resolves to %s before the shim directory; PATH must list %s first for attribution to apply",
				alias, resolved, shimDir))
		}
	}
	return notes
}

// pathPrefersShimDir reports whether shimDir appears before other on PATH.
func pathPrefersShimDir(shimDir, other string) bool {
	for _, entry := range filepath.SplitList(os.Getenv("PATH")) {
		switch filepath.Clean(entry) {
		case filepath.Clean(shimDir):
			return true
		case filepath.Clean(other):
			return false
		}
	}
	return false
}

// describeAliases renders an alias list for operator-facing notes.
func describeAliases(aliases []string) string { return strings.Join(aliases, ", ") }
