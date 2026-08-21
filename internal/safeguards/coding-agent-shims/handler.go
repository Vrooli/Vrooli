// Package codingagentshims installs PATH shims that route coding-agent
// invocations through vrooli-agent-launcher, so runs carry a Vrooli identity
// no matter how the agent was started.
//
// # Why a PATH shim rather than a shell function
//
// The predecessor of this safeguard was a set of bash functions sourced from
// the operator's ~/.bashrc. A shell function only exists inside an interactive
// bash that read that file, which left every other way of starting an agent
// silently unattributed: zsh (the macOS default), fish, PowerShell, scripts,
// make, CI, cron, systemd units, and any program that starts an agent with
// execve rather than through a shell. It also had no installer, so nothing
// could upgrade, verify, or remove it.
//
// A shim is a real executable. It works in every shell, in non-interactive
// contexts, and under execve, and it is the only form with a Windows story.
//
// # Why ~/.vrooli/bin
//
// That directory already exists, is already ahead of the agents' own install
// directories on the PATH Vrooli sets up, and needs no privilege. Installing
// here means the safeguard never touches a shell profile or a system path.
//
// # Why links to one binary
//
// vrooli-agent-launcher reads argv[0] to learn which agent it was invoked as
// (see cliutil.ShimAliasFromArgv0), so a single binary serves every agent.
// Adding an agent is a table entry, not another file to install and keep in
// sync. On Unix the aliases are symlinks; on Windows they are copies, because
// symlink creation there needs privilege this safeguard deliberately does not
// ask for.
//
// # Failure posture
//
// Attribution is observability, never a gate. If the shim is missing the
// operator simply gets an unattributed agent, which is why this safeguard is
// low risk and its absence is reported rather than escalated.
package codingagentshims

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

// launcherBinary is the multi-call binary every alias links to.
const launcherBinary = "vrooli-agent-launcher"

type handler struct {
	manifest hostreqkit.SafeguardManifest
}

// NewHandler is the constructor wired into the runtime registry under the
// handler name "coding_agent_shims" (see internal/runtime/registry.go).
func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

// ShimDir returns the directory the aliases are installed into.
func ShimDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home directory: %w", err)
	}
	return filepath.Join(home, ".vrooli", "bin"), nil
}

// executableName appends the platform's executable suffix.
func executableName(name string) string {
	if runtime.GOOS == "windows" {
		return name + ".exe"
	}
	return name
}

// aliasPaths returns the launcher path and every alias path to install.
func aliasPaths() (launcher string, aliases map[string]string, err error) {
	dir, err := ShimDir()
	if err != nil {
		return "", nil, err
	}
	launcher = filepath.Join(dir, executableName(launcherBinary))
	aliases = make(map[string]string, len(cliutil.CodingAgentAliases()))
	for _, alias := range cliutil.CodingAgentAliases() {
		aliases[alias] = filepath.Join(dir, executableName(alias))
	}
	return launcher, aliases, nil
}

// aliasInstalled reports whether path already routes to launcher. On Unix the
// link target must match; on Windows the copy must be byte-identical, which is
// also how a stale copy from an older build is detected.
func aliasInstalled(path, launcher string) bool {
	info, err := os.Lstat(path)
	if err != nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return sameFileContents(path, launcher)
	}
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

func sameFileContents(left, right string) bool {
	leftData, err := os.ReadFile(left)
	if err != nil {
		return false
	}
	rightData, err := os.ReadFile(right)
	if err != nil {
		return false
	}
	if len(leftData) != len(rightData) {
		return false
	}
	for i := range leftData {
		if leftData[i] != rightData[i] {
			return false
		}
	}
	return true
}

// installAlias creates or repairs one alias. Replacing is done by removing
// first so a stale link or an outdated copy is always superseded.
func installAlias(path, launcher string) error {
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stale shim %s: %w", path, err)
	}
	if runtime.GOOS == "windows" {
		data, err := os.ReadFile(launcher)
		if err != nil {
			return fmt.Errorf("read launcher: %w", err)
		}
		return os.WriteFile(path, data, 0o755)
	}
	return os.Symlink(launcher, path)
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	launcher, aliases, err := aliasPaths()
	if err != nil {
		return hostreqkit.InvalidConfigStatus(requirement, err.Error())
	}

	if _, err := os.Stat(launcher); err != nil {
		// Nothing to link to yet. This is the normal state before the first
		// build, so report it as pending work rather than a failure.
		status.Notes = append(status.Notes,
			"launcher not built yet at "+launcher+"; run `make install` before applying")
		return status
	}

	missing := make([]string, 0, len(aliases))
	for _, alias := range cliutil.CodingAgentAliases() {
		if !aliasInstalled(aliases[alias], launcher) {
			missing = append(missing, alias)
		}
	}
	if len(missing) > 0 {
		status.Notes = append(status.Notes,
			"agent shims missing or stale: "+strings.Join(missing, ", ")+"; those agents run unattributed")
		return status
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, fmt.Sprintf("%d agent shims installed in %s", len(aliases), filepath.Dir(launcher)))
	status.Notes = append(status.Notes, shadowingNote(filepath.Dir(launcher))...)
	return status
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

func (h handler) Apply(host hostreqkit.Host, status hostreqkit.ItemStatus, opts hostreqkit.EnsureOptions) (hostreqkit.ItemStatus, error) {
	switch status.SupportClass {
	case hostreqkit.SupportUnsupported:
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		return status, nil
	case hostreqkit.SupportNotApplicable:
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		return status, nil
	case hostreqkit.SupportManualOnly:
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status, nil
	}

	if status.Applied {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
		return status, nil
	}

	launcher, aliases, err := aliasPaths()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if _, err := os.Stat(launcher); err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "launcher missing at "+launcher+"; build it before applying")
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, fmt.Sprintf("dry-run: would install %d agent shims in %s", len(aliases), filepath.Dir(launcher)))
		return status, nil
	}

	installed := make([]string, 0, len(aliases))
	for _, alias := range cliutil.CodingAgentAliases() {
		path := aliases[alias]
		if aliasInstalled(path, launcher) {
			continue
		}
		if err := installAlias(path, launcher); err != nil {
			status.ExecutionState = hostreqkit.ExecutionFailed
			status.Notes = append(status.Notes, "install shim "+alias+" failed: "+err.Error())
			return status, nil
		}
		installed = append(installed, alias)
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionApplied
	status.Notes = append(status.Notes, "installed agent shims: "+strings.Join(installed, ", "))
	status.Notes = append(status.Notes, shadowingNote(filepath.Dir(launcher))...)
	return status, nil
}
