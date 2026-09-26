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
// # Why ~/.vrooli/shims and not ~/.vrooli/bin
//
// The aliases originally went into the shared install root, which needed no new
// directory and was already on PATH. That convenience cost more than it saved.
// A storage declaration describes one directory, and the safeguard's
// declaration therefore described bin: five regenerable links, declared as a
// 64MiB cache, over a shared root holding gigabytes of other components' build
// output. Nothing about that is true of bin, and the untruth was load-bearing --
// a declared budget is what the retention enforcer prunes to, and only a
// hard-coded guard stood between that declaration and the install root.
//
// Owning a directory outright makes the declaration honest: everything in
// ~/.vrooli/shims really is regenerable from the launcher binary, really does
// fit in the declared budget, and really can be wiped without collateral. It
// also decouples the aliases from a directory whose contents other installers
// churn.
//
// # Why links to one binary
//
// vrooli-agent-launcher reads argv[0] to learn which agent it was invoked as
// (see cliutil.ShimAliasFromArgv0), so a single binary serves every agent.
// Adding an agent is a table entry, not another file to install and keep in
// sync. On Unix the aliases are symlinks; on Windows they are hard links, which
// need no privilege on NTFS while symlinks there do, falling back to copies on
// filesystems that carry no hard links.
//
// # Failure posture
//
// Attribution is observability, never a gate. If the shim is missing the
// operator simply gets an unattributed agent, which is why this safeguard is
// low risk and its absence is reported rather than escalated. That posture is
// also why the alias set is re-asserted on every control-plane start (see
// EnsureInstalled) instead of only at setup time: a safeguard whose absence is
// never escalated needs to repair itself, or it degrades silently and stays
// degraded.
package codingagentshims

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/vrooli/internal/artifactledger"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

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

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported

	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}

	launcher, aliases, err := resolvePaths()
	if err != nil {
		return hostreqkit.InvalidConfigStatus(requirement, err.Error())
	}

	if !launcherBuilt(launcher) {
		// Nothing to link to yet. This is the normal state before the first
		// build, so it is reported as pending work rather than a failure.
		status.Notes = append(status.Notes,
			"launcher not built yet at "+launcher+"; run `make install` before applying")
		return status
	}

	shimDir, err := ShimDir()
	if err != nil {
		return hostreqkit.InvalidConfigStatus(requirement, err.Error())
	}

	if missing := MissingAliases(aliases, launcher); len(missing) > 0 {
		status.Notes = append(status.Notes,
			"agent shims missing or stale: "+describeAliases(missing)+"; those agents run unattributed")
		return status
	}

	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, fmt.Sprintf("%d agent shims installed in %s", len(aliases), shimDir))
	status.Notes = append(status.Notes, shadowingNote(shimDir)...)
	return status
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

	launcher, aliases, err := resolvePaths()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if !launcherBuilt(launcher) {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "launcher missing at "+launcher+"; build it before applying")
		return status, nil
	}
	shimDir, err := ShimDir()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}

	// Applying is not skipped when Inspect said the aliases were present: apply
	// also retires aliases the previous layout left in the install root, and
	// that migration has to run once on a host whose new-location shims are
	// already correct.
	missing := MissingAliases(aliases, launcher)
	legacy, legacyErr := legacyAliasesPresent()
	if legacyErr != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, legacyErr.Error())
		return status, nil
	}

	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, dryRunNote(missing, legacy, shimDir))
		return status, nil
	}

	installed, err := EnsureInstalled()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "install agent shims failed: "+err.Error())
		return status, nil
	}

	removed, err := RemoveLegacyAliases(removalLedger())
	if err != nil {
		// The new aliases are in place, so attribution already works. Retiring
		// the old ones is cleanup; reporting it beats failing an applied item.
		status.Notes = append(status.Notes, "retiring superseded shims failed: "+err.Error())
	}

	status.Applied = true
	if len(installed) == 0 && len(removed) == 0 {
		status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	} else {
		status.ExecutionState = hostreqkit.ExecutionApplied
	}
	if len(installed) > 0 {
		status.Notes = append(status.Notes, "installed agent shims: "+describeAliases(installed))
	}
	if len(removed) > 0 {
		legacyDir, _ := LegacyShimDir()
		status.Notes = append(status.Notes,
			"retired superseded shims in "+legacyDir+": "+describeAliases(removed))
	}
	status.Notes = append(status.Notes, shadowingNote(shimDir)...)
	return status, nil
}

// resolvePaths returns the launcher and the alias set in one step, so Inspect
// and Apply cannot disagree about where either lives.
func resolvePaths() (string, map[string]string, error) {
	launcher, err := LauncherPath()
	if err != nil {
		return "", nil, fmt.Errorf("resolve coding-agent launcher path: %w", err)
	}
	aliases, err := AliasPaths()
	if err != nil {
		return "", nil, fmt.Errorf("resolve coding-agent alias paths: %w", err)
	}
	return launcher, aliases, nil
}

func dryRunNote(missing, legacy []string, shimDir string) string {
	note := fmt.Sprintf("dry-run: would install %d agent shims in %s", len(missing), shimDir)
	if len(missing) == 0 {
		note = "dry-run: agent shims already present in " + shimDir
	}
	if len(legacy) > 0 {
		note += "; would retire superseded shims: " + describeAliases(legacy)
	}
	return note
}

// legacyAliasesPresent lists aliases still installed in the shared install root.
func legacyAliasesPresent() ([]string, error) {
	legacyDir, err := LegacyShimDir()
	if err != nil {
		return nil, err
	}
	launcher, err := LauncherPath()
	if err != nil {
		return nil, err
	}
	present := make([]string, 0, len(cliutil.CodingAgentAliases()))
	for _, alias := range cliutil.CodingAgentAliases() {
		if aliasInstalled(filepath.Join(legacyDir, executableName(alias)), launcher) {
			present = append(present, alias)
		}
	}
	return present, nil
}

// launcherBuilt reports whether the multi-call binary exists to link to.
func launcherBuilt(launcher string) bool {
	_, err := os.Stat(launcher)
	return err == nil
}

// removalLedger resolves the receipt ledger legacy-alias removal writes to.
//
// A nil ledger means the removal proceeds unrecorded. That is the wrong
// direction in general, and it is the right one here: this is a best-effort
// migration of an alias the operator no longer reaches, and refusing to tidy it
// because the state directory is unavailable would leave two aliases for one
// agent on PATH -- the more confusing outcome of the two.
func removalLedger() *artifactledger.Ledger {
	home, err := shimHomeDir()
	if err != nil {
		return nil
	}
	ledger, err := artifactledger.New(home)
	if err != nil {
		return nil
	}
	return ledger
}
