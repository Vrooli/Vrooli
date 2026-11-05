package agentinstall

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/binaryfetch"
	"github.com/vrooli/cli-core/upstreamcheck"
)

// HarnessActionResult is the outcome of one action and the observation after it.
type HarnessActionResult struct {
	CleanupPlan *CleanupPlan `json:"cleanup_plan,omitempty"`
	Action      string       `json:"action"`
	Success     bool         `json:"success"`
	Message     string       `json:"message"`
	// PreservedPath is where a replaced file now sits, so it can be restored.
	PreservedPath string        `json:"preserved_path,omitempty"`
	Report        HarnessReport `json:"report"`
}

// ArtifactRecorder adds a path to the install ledger so uninstall can find it.
// The control plane passes the ledger in: this package is compiled into every
// resource CLI, whose module does not carry the ledger's dependencies.
type ArtifactRecorder func(home, resource, path string, directory bool) error

func (d harnessDeps) recordArtifact(home, resource, path string, directory bool) error {
	if d.record == nil {
		return nil
	}
	return d.record(home, resource, path, directory)
}

// RunHarnessAction performs update, use_managed_copy, link_managed_copy or
// rollback, serialized per harness across processes.
//
// actor names who is acting, for the records this writes. It is attribution:
// operator rule 9 keeps agent identity from being a reason to refuse this.
func RunHarnessAction(ctx context.Context, root, resource, action, actor string, record ArtifactRecorder) (HarnessActionResult, error) {
	harness, err := LoadHarness(root, resource)
	if err != nil {
		return HarnessActionResult{}, err
	}
	deps := defaultHarnessDeps()
	deps.record, deps.actor = record, actor
	return runHarnessAction(ctx, harness, action, deps), nil
}

// Uninstall removes only version prefixes with this resource's installation
// receipt and the launcher symlink when it points into one of those prefixes.
// A PATH-discovered or standalone upstream copy is deliberately left alone.
func Uninstall(ctx context.Context, root, resource string) error {
	h, err := LoadHarness(root, resource)
	if err != nil {
		return err
	}
	return uninstallHarness(ctx, h)
}

func uninstallHarness(ctx context.Context, h Harness) error {
	deps := defaultHarnessDeps()
	unlock, err := deps.platform.withDefaults().acquireLock(ctx, "resource-update-"+h.Resource)
	if err != nil {
		return err
	}
	defer unlock()
	return uninstallHarnessLocked(ctx, h)
}

func uninstallHarnessLocked(ctx context.Context, h Harness) error {
	installs, err := h.managedInstalls()
	if err != nil {
		return err
	}
	link, err := h.readLink()
	if err != nil {
		return err
	}
	return removeManagedInstalls(ctx, h, installs, link, true)
}

func removeManagedInstalls(ctx context.Context, h Harness, installs []managedInstall, link linkState, removeLauncher bool) error {
	// Anchor deletion at Home. Root operations reject paths that escape through
	// symlinks, and the relative paths below are checked again by the kernel-backed
	// os.Root implementation at the point of removal.
	home, err := filepath.Abs(h.Home)
	if err != nil {
		return err
	}
	homeRoot, err := os.OpenRoot(home)
	if err != nil {
		return err
	}
	defer homeRoot.Close()
	versions, err := h.versionsRoot()
	if err != nil {
		return err
	}
	versionsRelative, err := filepath.Rel(home, versions)
	if err != nil || filepath.IsAbs(versionsRelative) || versionsRelative == ".." || strings.HasPrefix(versionsRelative, ".."+string(filepath.Separator)) {
		return fmt.Errorf("refuse unsafe managed versions root: %s", versions)
	}
	versionsRoot, err := openDirectoryRootNoSymlinks(homeRoot, versionsRelative)
	if err != nil {
		return fmt.Errorf("refuse unsafe managed versions root %s: %w", versions, err)
	}
	defer versionsRoot.Close()
	owned := make(map[string]bool, len(installs))
	relatives := make(map[string]string, len(installs))
	for _, install := range installs {
		if !withinDir(install.Prefix, h.Home) {
			return fmt.Errorf("refuse to remove managed prefix outside home: %s", install.Prefix)
		}
		relative, err := filepath.Rel(versions, install.Prefix)
		if err != nil || relative == "." || filepath.Dir(relative) != "." || filepath.IsAbs(relative) {
			return fmt.Errorf("refuse unsafe managed prefix: %s", install.Prefix)
		}
		owned[filepath.Clean(install.Prefix)] = true
		relatives[filepath.Clean(install.Prefix)] = filepath.Base(relative)
	}
	for _, install := range installs {
		if err := versionsRoot.RemoveAll(relatives[filepath.Clean(install.Prefix)]); err != nil {
			return fmt.Errorf("remove managed %s %s: %w", h.Resource, install.Version, err)
		}
	}
	if removeLauncher && link.Kind == linkManaged && owned[filepath.Clean(link.Prefix)] {
		linkRelative, err := filepath.Rel(home, h.linkPath())
		if err != nil || filepath.IsAbs(linkRelative) || linkRelative == ".." || strings.HasPrefix(linkRelative, ".."+string(filepath.Separator)) {
			return fmt.Errorf("refuse unsafe managed launcher path: %s", h.linkPath())
		}
		if err := homeRoot.Remove(linkRelative); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}

// openDirectoryRootNoSymlinks pins each directory component beneath base.
// Lstat rejects symlinks, and SameFile ensures OpenRoot opened the directory
// that was inspected even if an ancestor changed between those operations.
func openDirectoryRootNoSymlinks(base *os.Root, relative string) (*os.Root, error) {
	if filepath.IsAbs(relative) || relative == "." || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return nil, fmt.Errorf("unsafe relative directory %q", relative)
	}
	current := base
	var owned *os.Root
	for _, component := range strings.Split(filepath.Clean(relative), string(filepath.Separator)) {
		if component == "" || component == "." || component == ".." {
			if owned != nil {
				_ = owned.Close()
			}
			return nil, fmt.Errorf("unsafe directory component %q", component)
		}
		before, err := current.Lstat(component)
		if err != nil {
			if owned != nil {
				_ = owned.Close()
			}
			return nil, err
		}
		if before.Mode()&os.ModeSymlink != 0 || !before.IsDir() {
			if owned != nil {
				_ = owned.Close()
			}
			return nil, fmt.Errorf("directory component %q is not a real directory", component)
		}
		next, err := current.OpenRoot(component)
		if err != nil {
			if owned != nil {
				_ = owned.Close()
			}
			return nil, err
		}
		after, err := next.Stat(".")
		if err != nil || !os.SameFile(before, after) {
			_ = next.Close()
			if owned != nil {
				_ = owned.Close()
			}
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("directory component %q changed while opening", component)
		}
		if owned != nil {
			_ = owned.Close()
		}
		owned, current = next, next
	}
	return owned, nil
}

func runHarnessAction(ctx context.Context, h Harness, action string, deps harnessDeps) HarnessActionResult {
	deps.platform = deps.platform.withDefaults()
	if deps.latest == nil {
		deps.latest = func(ctx context.Context, h Harness) (HarnessSource, string, error) {
			return latestHarnessReleaseWithDeps(ctx, h, deps.platform)
		}
	}
	if deps.version == nil {
		deps.version = func(ctx context.Context, path string) (string, error) {
			return installedVersion(ctx, path, deps.platform)
		}
	}
	result := HarnessActionResult{Action: action}
	release, err := deps.platform.acquireLock(ctx, "resource-update-"+h.Resource)
	if err != nil {
		result.Message = "Another action on this agent is still running: " + err.Error()
		result.Report = inspectHarness(ctx, h, deps)
		return result
	}
	defer release()

	before := inspectHarness(ctx, h, deps)
	switch action {
	case ActionUpdate:
		result.Message, err = h.runUpdate(ctx, before, deps)
	case ActionUseManagedCopy:
		result.Message, result.PreservedPath, err = h.useManagedCopy(ctx, before, deps)
	case ActionLinkManagedCopy:
		result.Message, result.PreservedPath, err = h.linkManagedCopy(ctx, before, deps)
	case ActionRollback:
		result.Message, err = h.rollback(ctx, before, deps)
	default:
		err = fmt.Errorf("unknown action %q", action)
	}
	if err != nil {
		result.Message = err.Error()
	} else {
		result.Success = true
	}
	result.Report = inspectHarness(ctx, h, deps)
	return result
}

// harnessChange undoes or settles the side effects an action makes beyond the
// file it replaced: the approval record and the launcher's delegate.
type harnessChange struct {
	undo   []func(context.Context) error
	settle []func()
}

func (c *harnessChange) onFailure(undo func(context.Context) error) { c.undo = append(c.undo, undo) }
func (c *harnessChange) onSuccess(settle func())                    { c.settle = append(c.settle, settle) }

// revert undoes in reverse order, on its own clock: a timeout is often what
// failed, and the restoration has to run anyway.
func (c *harnessChange) revert(ctx context.Context) error {
	restoring, cancel := context.WithTimeout(context.WithoutCancel(ctx), time.Minute)
	defer cancel()
	var failures []error
	for i := len(c.undo) - 1; i >= 0; i-- {
		if err := c.undo[i](restoring); err != nil {
			failures = append(failures, err)
		}
	}
	return errors.Join(failures...)
}

func (c *harnessChange) commit() {
	for _, settle := range c.settle {
		settle()
	}
}

func refuseFromState(action string, report HarnessReport) error {
	return fmt.Errorf("%s does not apply while this agent is %s: %s", action, report.State, report.Reason)
}

func (h Harness) runUpdate(ctx context.Context, before HarnessReport, deps harnessDeps) (string, error) {
	switch before.State {
	case StateUpToDate:
		return fmt.Sprintf("%s is already current.", before.Installed), nil
	case StateUpdateReady:
	default:
		return "", refuseFromState("Updating", before)
	}
	previous, err := os.Readlink(h.linkPath())
	if err != nil {
		return "", err
	}
	change := &harnessChange{}
	if err := h.installRelease(ctx, before.Latest, deps, change); err != nil {
		// A selected build that then fails verification must not stay selected.
		if current, readErr := os.Readlink(h.linkPath()); readErr == nil && current != previous {
			if restoreErr := publishLinkWithDeps(previous, h.linkPath(), deps.platform); restoreErr != nil {
				err = fmt.Errorf("%w; restoring %s also failed: %v", err, previous, restoreErr)
			}
		}
		if revertErr := change.revert(ctx); revertErr != nil {
			err = fmt.Errorf("%w; undoing it also failed: %v", err, revertErr)
		}
		return "", err
	}
	change.commit()
	return fmt.Sprintf("Updated %s to %s. %s", before.Installed, before.Latest, sessionImpact), nil
}

// installRelease installs and selects one exact release through its owner. The
// caller reverts change on any later failure, so an approval never outlives the
// install it was recorded for.
func (h Harness) installRelease(ctx context.Context, version string, deps harnessDeps, change *harnessChange) error {
	if deps.installRelease != nil {
		return deps.installRelease(ctx, h, version, deps)
	}
	if h.ManifestBase != "" {
		return h.installFromManifest(ctx, deps)
	}
	gate := deps.gate(ctx, h, version, deps.platform)
	if !gate.Reachable {
		return fmt.Errorf("the Dependency Analyzer is not answering")
	}
	var previous approvalRecord
	if gate.Blocked {
		if gate.Verdict != "out_of_range" {
			return fmt.Errorf("the dependency policy refuses %s@%s: %s", h.Package, version, gate.Message)
		}
		var err error
		if previous, err = h.moveApproval(ctx, version, deps, change); err != nil {
			return fmt.Errorf("record approval for %s@%s: %w", h.Package, version, err)
		}
	}
	var installErr error
	if deps.installWithDeps != nil {
		installErr = deps.installWithDeps(ctx, h.spec(), h.Package+"@"+version, deps.platform)
	} else if deps.install != nil {
		installErr = deps.install(ctx, h.spec(), h.Package+"@"+version)
	} else {
		installErr = installNPMWithDeps(ctx, h.spec(), h.Package+"@"+version, deps.platform)
	}
	if installErr != nil {
		return installErr
	}
	if previous != nil {
		if err := h.savePreviousApproval(version, previous); err != nil {
			return fmt.Errorf("keep the previous approval for rollback: %w", err)
		}
	}
	if err := h.refreshDelegate(ctx, version, deps, change); err != nil {
		return err
	}
	return verifySelectedUpdate(ctx, h.Binary, h.linkPath(), version, updateDeps{installed: deps.version, lookPath: deps.lookPath, stat: os.Stat})
}

// approvalRecord is a record as the gateway's explain verb prints it, kept as
// JSON so fields this package does not interpret survive unchanged.
type approvalRecord map[string]any

// previousApprovalFile sits beside the receipt, outside the digested node_modules.
const previousApprovalFile = "previous-approval.json"

// moveApproval points the exact approval at one new version and records who did
// it. It leaves lastReviewed alone and clears approvedDate, because an update is
// not a dependency review and the record must not claim one; the date is in the
// rationale. Notes and scope stay.
func (h Harness) moveApproval(ctx context.Context, version string, deps harnessDeps, change *harnessChange) (approvalRecord, error) {
	previous, err := deps.readApproval(ctx, h, deps.platform)
	if err != nil {
		return nil, err
	}
	next := make(approvalRecord, len(previous))
	for key, value := range previous {
		next[key] = value
	}
	actor := deps.actor
	if actor == "" {
		actor = "an unnamed caller"
	}
	next["versionRange"], next["rangePolicy"], next["approvedBy"] = version, "exact", actor
	next["rationale"] = fmt.Sprintf("harness update to %s by %s on %s. Recorded by vrooli resource update, moved from %v. No dependency review took place.", version, actor, deps.now().UTC().Format("2006-01-02"), previous["versionRange"])
	delete(next, "approvedDate")
	if err := deps.writeApproval(ctx, h, next, deps.platform); err != nil {
		return nil, err
	}
	change.onFailure(func(ctx context.Context) error {
		if err := deps.writeApproval(ctx, h, previous, deps.platform); err != nil {
			return fmt.Errorf("restore the %s approval: %w", h.Package, err)
		}
		return nil
	})
	return previous, nil
}

func (h Harness) savePreviousApproval(version string, previous approvalRecord) error {
	prefix, err := NPMInstallPrefix(h.Home, h.Resource, version)
	if err != nil {
		return err
	}
	data, err := json.Marshal(previous)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(prefix, previousApprovalFile), data, 0o600)
}

// restorePreviousApproval undoes the record an update moved, and only if the
// record still names the version being left: anything else is a later decision
// this must not overwrite. It returns a sentence for the operator.
func (h Harness) restorePreviousApproval(ctx context.Context, leaving string, deps harnessDeps) string {
	prefix, err := NPMInstallPrefix(h.Home, h.Resource, leaving)
	if err != nil {
		return ""
	}
	path := filepath.Join(prefix, previousApprovalFile)
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var previous approvalRecord
	if err := json.Unmarshal(data, &previous); err != nil {
		return fmt.Sprintf(" The dependency approval was not restored: %s is unreadable.", path)
	}
	current, err := deps.readApproval(ctx, h, deps.platform)
	if err != nil {
		return fmt.Sprintf(" The dependency approval still names %s: %v.", leaving, err)
	}
	if current["versionRange"] != leaving {
		return fmt.Sprintf(" The dependency approval was left at %v, which a later decision set.", current["versionRange"])
	}
	if err := deps.writeApproval(ctx, h, previous, deps.platform); err != nil {
		return fmt.Sprintf(" The dependency approval still names %s: %v.", leaving, err)
	}
	_ = os.Remove(path)
	return fmt.Sprintf(" The dependency approval is back at %v.", previous["versionRange"])
}

// readApprovalRecord and writeApprovalRecord use the gateway's own verbs.
func readApprovalRecord(ctx context.Context, h Harness, platform platformDeps) (approvalRecord, error) {
	output, err := platform.withDefaults().commandOutput(ctx, dependencyAnalyzerCommand, "deps", "approved", "explain", "npm/"+h.Package, "--json")
	if err != nil {
		return nil, fmt.Errorf("read the approval record: %w", err)
	}
	var explained struct {
		Record approvalRecord `json:"record"`
		Found  bool           `json:"found"`
	}
	if err := json.Unmarshal(output, &explained); err != nil {
		return nil, err
	}
	if !explained.Found || explained.Record == nil {
		return nil, fmt.Errorf("no approval record exists for %s", h.Package)
	}
	return explained.Record, nil
}

func writeApprovalRecord(ctx context.Context, _ Harness, record approvalRecord, platform platformDeps) error {
	data, err := json.Marshal(record)
	if err != nil {
		return err
	}
	file, err := os.CreateTemp("", "harness-approval-*.json")
	if err != nil {
		return err
	}
	defer os.Remove(file.Name())
	if _, err := file.Write(data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if output, err := platform.withDefaults().commandCombined(ctx, dependencyAnalyzerCommand, "deps", "approved", "upsert", "--file", file.Name(), "--apply", "--json"); err != nil {
		return fmt.Errorf("%w: %s", err, output)
	}
	return nil
}

// useManagedCopy repairs managed selection without renaming a found copy or
// editing shell startup; an independent PATH entry stays under owner control.
func (h Harness) useManagedCopy(ctx context.Context, before HarnessReport, deps harnessDeps) (message, preserved string, err error) {
	switch {
	case before.State == StateUpToDate || before.State == StateUpdateReady:
		return "Vrooli's copy already runs first.", "", nil
	case before.State == StateFound:
		return "This found copy was left unchanged. Put Vrooli's managed bin directory before it on PATH if you want Vrooli's copy to run.", "", nil
	case before.State != StateShadowed || before.PrimaryAction != ActionUseManagedCopy:
		return "", "", refuseFromState("Using Vrooli's copy", before)
	}
	change := &harnessChange{}
	if sameFile(before.CommandPath, h.linkPath()) {
		if err := h.refreshDelegate(ctx, before.ManagedVersion, deps, change); err != nil {
			return "", "", err
		}
		change.commit()
		return fmt.Sprintf("Vrooli's %s now runs.", before.ManagedVersion), "", nil
	}
	// A discovered upstream copy is user-owned. Selection must never rename it.
	return "", "", fmt.Errorf("the found copy at %s is left unchanged; put %s first on PATH to select Vrooli's copy", before.CommandPath, h.BinDir)
}

// linkManagedCopy replaces a standalone file at the link path. The newest
// managed version is linked when it is not older than what runs; otherwise the
// latest release installs, because linking must not silently downgrade.
func (h Harness) linkManagedCopy(ctx context.Context, before HarnessReport, deps harnessDeps) (message, preserved string, err error) {
	switch {
	case before.State == StateUpToDate || before.State == StateUpdateReady:
		return "Vrooli's copy is already linked.", "", nil
	case before.State != StateUnlinked:
		return "", "", refuseFromState("Linking Vrooli's copy", before)
	}
	link, err := h.readLink()
	if err != nil {
		return "", "", err
	}
	installs, err := h.managedInstalls()
	if err != nil {
		return "", "", err
	}
	var candidate managedInstall
	if len(installs) > 0 && (before.Installed == "" || upstreamcheck.Compare(before.Installed, installs[0].Version) != upstreamcheck.StatusAhead) {
		candidate = installs[0]
	}
	if candidate.Version == "" && before.Latest == "" {
		return "", "", fmt.Errorf("Vrooli has no copy to link and the latest version is unknown")
	}

	// A foreign symlink stays until the new link replaces it atomically.
	path := h.linkPath()
	if link.Kind == linkStandalone {
		if preserved, err = moveAside(path, deps); err != nil {
			return "", "", err
		}
	}
	change := &harnessChange{}
	restore := func(cause error) (string, string, error) {
		if revertErr := change.revert(ctx); revertErr != nil {
			cause = fmt.Errorf("%w; undoing it also failed: %v", cause, revertErr)
		}
		var restoreErr error
		if link.Kind == linkStandalone {
			// Rename alone replaces whatever is there: the command never goes missing.
			restoreErr = os.Rename(preserved, path)
		} else {
			restoreErr = publishLinkWithDeps(link.Target, path, deps.platform)
		}
		if restoreErr != nil {
			return "", preserved, fmt.Errorf("%w; restoring %s also failed: %v", cause, path, restoreErr)
		}
		return "", "", cause
	}

	version := candidate.Version
	if version != "" {
		if err := h.selectInstall(ctx, candidate, deps); err != nil {
			return restore(err)
		}
		if err := h.refreshDelegate(ctx, version, deps, change); err != nil {
			return restore(err)
		}
	} else {
		version = before.Latest
		if err := h.installRelease(ctx, version, deps, change); err != nil {
			return restore(err)
		}
	}
	raw, err := deps.version(ctx, path)
	if err != nil {
		return restore(fmt.Errorf("%s does not run: %w", path, err))
	}
	if got := upstreamcheck.ExtractSemver(raw); got != version {
		return restore(fmt.Errorf("%s reports %q, expected %s", path, raw, version))
	}
	change.commit()
	message = fmt.Sprintf("%s now links to Vrooli's %s.", path, version)
	if preserved != "" {
		message += " The previous file is at " + preserved + "."
	}
	return message, preserved, nil
}

// rollback relinks to the previous version directory. It reinstalls nothing, so
// it needs neither the network nor the Dependency Analyzer to get the operator
// back to a working version; update's refusal to downgrade stays intact because
// this is a separate, explicit path. The approval the update moved goes back too,
// but a gateway outage only leaves a sentence: it must not block the way back.
func (h Harness) rollback(ctx context.Context, before HarnessReport, deps harnessDeps) (string, error) {
	if before.RollbackVersion == "" {
		return "", fmt.Errorf("no previous version is installed")
	}
	if before.State == StateNeedsSetup || before.InstallKind != InstallKindManaged {
		return "", refuseFromState("Rolling back", before)
	}
	installs, err := h.managedInstalls()
	if err != nil {
		return "", err
	}
	var target managedInstall
	for _, install := range installs {
		if install.Version == before.RollbackVersion {
			target = install
		}
	}
	if target.Version == "" {
		return "", fmt.Errorf("version %s is no longer installed", before.RollbackVersion)
	}
	if err := h.verifyInstall(ctx, target); err != nil {
		return "", fmt.Errorf("version %s failed verification: %w", target.Version, err)
	}
	current, err := os.Readlink(h.linkPath())
	if err != nil {
		return "", err
	}
	change := &harnessChange{}
	undo := func(cause error) (string, error) {
		if restoreErr := publishLinkWithDeps(current, h.linkPath(), deps.platform); restoreErr != nil {
			cause = fmt.Errorf("%w; relinking %s also failed: %v", cause, current, restoreErr)
		}
		if revertErr := change.revert(ctx); revertErr != nil {
			cause = fmt.Errorf("%w; undoing it also failed: %v", cause, revertErr)
		}
		return "", cause
	}
	if err := h.selectInstall(ctx, target, deps); err != nil {
		return "", err
	}
	if err := h.refreshDelegate(ctx, target.Version, deps, change); err != nil {
		return undo(err)
	}
	raw, err := deps.version(ctx, h.linkPath())
	if err != nil {
		return undo(err)
	}
	if got := upstreamcheck.ExtractSemver(raw); got != target.Version {
		return undo(fmt.Errorf("%s reports %q, expected %s", h.linkPath(), raw, target.Version))
	}
	change.commit()
	return fmt.Sprintf("Rolled back to %s. %s%s", target.Version, sessionImpact, h.restorePreviousApproval(ctx, before.ManagedVersion, deps)), nil
}

// verifyInstall re-checks a prefix against its receipt before it is selected again.
func (h Harness) verifyInstall(ctx context.Context, install managedInstall) error {
	record, err := readInstallationRecord(install.Prefix)
	if err != nil {
		return err
	}
	if record.FileSHA256 != "" {
		digest, err := binaryfetch.FileDigestContext(ctx, install.Executable)
		if err != nil {
			return err
		}
		if digest != record.FileSHA256 {
			return fmt.Errorf("the file no longer matches its receipt")
		}
		return nil
	}
	return binaryfetch.VerifyArtifactContext(ctx, filepath.Join(install.Prefix, "node_modules"), "dir", record.TreeSHA256)
}

// selectInstall publishes the managed link under the lock the installer holds.
func (h Harness) selectInstall(ctx context.Context, install managedInstall, deps harnessDeps) error {
	deps.platform = deps.platform.withDefaults()
	release, err := deps.platform.acquireLock(ctx, "resource-agent-"+h.Resource)
	if err != nil {
		return err
	}
	defer release()
	if err := deps.recordArtifact(h.Home, h.Resource, h.linkPath(), false); err != nil {
		return err
	}
	return linkNPMBinaryWithDeps(install.Executable, h.Binary, h.BinDir, deps.platform)
}

// refreshDelegate makes the package's launcher run the selected version. A
// delegate that already reports the version is left alone. Otherwise the stale
// entry point is retired and the launcher's own bootstrap rebuilds it from the
// selected package; older versioned files stay where they are. change puts the
// old delegate back if the action later fails, because a restored link's launcher
// would otherwise run the newer delegate, and drops a moved-aside copy on success.
func (h Harness) refreshDelegate(ctx context.Context, version string, deps harnessDeps, change *harnessChange) error {
	if h.DelegatePath == "" {
		return nil
	}
	// A native managed executable does not read the npm launcher's delegate.
	// Preserve an independently installed copy when the selection already works.
	if raw, err := deps.version(ctx, h.linkPath()); err == nil && upstreamcheck.ExtractSemver(raw) == version {
		return nil
	}
	info, err := os.Lstat(h.DelegatePath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if raw, err := deps.version(ctx, h.DelegatePath); err == nil && upstreamcheck.ExtractSemver(raw) == version {
		return nil
	}
	var restore func() error
	settle := func() {}
	if info.Mode()&os.ModeSymlink != 0 {
		previous, err := os.Readlink(h.DelegatePath)
		if err != nil {
			return err
		}
		if err := os.Remove(h.DelegatePath); err != nil {
			return err
		}
		restore = func() error { return publishLinkWithDeps(previous, h.DelegatePath, deps.platform) }
	} else {
		preserved, err := moveAside(h.DelegatePath, deps)
		if err != nil {
			return err
		}
		restore = func() error { return os.Rename(preserved, h.DelegatePath) }
		settle = func() { _ = os.Remove(preserved) }
	}
	raw, err := deps.version(ctx, h.linkPath())
	if err == nil && upstreamcheck.ExtractSemver(raw) == version {
		change.onFailure(func(context.Context) error { return restore() })
		change.onSuccess(settle)
		return nil
	}
	if restoreErr := restore(); restoreErr != nil {
		return fmt.Errorf("the launcher did not rebuild %s, and restoring it failed: %v", h.DelegatePath, restoreErr)
	}
	return fmt.Errorf("the package launcher did not rebuild %s for %s (reported %q)", h.DelegatePath, version, raw)
}

// moveAside renames a file beside itself so an operator can restore it with one
// mv. It refuses a root-owned file: that belongs to setup, not to this process.
func moveAside(path string, deps harnessDeps) (string, error) {
	if pathOwnedByRoot(path, deps) {
		return "", fmt.Errorf("%s belongs to root; run %s", path, NeedsSetupCommand)
	}
	stamp := deps.now().UTC().Format("20060102T150405Z")
	preserved := path + ".vrooli-replaced-" + stamp
	for suffix := 1; ; suffix++ {
		if _, err := os.Lstat(preserved); os.IsNotExist(err) {
			break
		} else if err != nil {
			return "", err
		}
		preserved = path + ".vrooli-replaced-" + stamp + "-" + strconv.Itoa(suffix)
	}
	if err := os.Rename(path, preserved); err != nil {
		return "", fmt.Errorf("keep %s: %w", path, err)
	}
	return preserved, nil
}

var nodeEngineMajor = regexp.MustCompile(`\d+`)

// nodeMajorFromPackage reads the lower bound of engines.node. A range it cannot
// read as a simple minimum yields zero, leaving the verdict to npm.
func nodeMajorFromPackage(data []byte) int {
	var manifest struct {
		Engines struct {
			Node string `json:"node"`
		} `json:"engines"`
	}
	if json.Unmarshal(data, &manifest) != nil {
		return 0
	}
	match := nodeEngineMajor.FindString(manifest.Engines.Node)
	if match == "" {
		return 0
	}
	major, _ := strconv.Atoi(match)
	return major
}
