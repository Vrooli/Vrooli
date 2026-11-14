package agentinstall

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/vrooli/cli-core/cliutil"
	"github.com/vrooli/cli-core/upstreamcheck"
	governancev1 "github.com/vrooli/vrooli/packages/proto/gen/go/scenario-dependency-analyzer/v1/dependency_governance"
	"google.golang.org/protobuf/encoding/protojson"
)

// Harness states and actions. The values are the Web Console contract in
// capabilities.proto (AgentUpdateResponse.state and primary_action).
const (
	StateMissing     = "missing"
	StateFound       = "found"
	StateShadowed    = "shadowed"
	StateUnlinked    = "unlinked"
	StateNeedsSetup  = "needs_setup"
	StateUpdateReady = "update_ready"
	StateUpToDate    = "up_to_date"
	StateCheckFailed = "check_failed"

	ActionInstall         = "install"
	ActionUseManagedCopy  = "use_managed_copy"
	ActionLinkManagedCopy = "link_managed_copy"
	ActionUpdate          = "update"
	ActionRetry           = "retry"
	ActionRollback        = "rollback"

	InstallKindManaged           = "managed"
	InstallKindUpstreamInstaller = "upstream_installer"
	InstallKindStandalone        = "standalone"
	InstallKindRootOwned         = "root_owned"
	InstallKindUnknown           = "unknown"

	// NeedsSetupCommand is the one elevation boundary. The
	// harness_file_ownership safeguard is what makes this command repair the
	// state that reports it.
	NeedsSetupCommand = "sudo vrooli setup --include-optional"

	// sessionImpact is what the code guarantees: the link swap leaves running
	// processes on their inode, version prefixes are not removed, and every
	// launch resolves the binary again. A resumed session is a new process.
	sessionImpact = "Running sessions keep the version they started with. New and resumed sessions use the new version."

	dependencyAnalyzerCommand = "scenario-dependency-analyzer"
)

// HarnessCheck is one update precondition with a stable ID.
type HarnessCheck struct {
	ID      string `json:"id"`
	Passed  bool   `json:"passed"`
	Message string `json:"message"`
}

// HarnessSource names where the latest version came from.
type HarnessSource struct {
	Kind string `json:"kind"`
	ID   string `json:"id"`
}

// HarnessReport is the read-only observation of one harness on this host.
type HarnessReport struct {
	Name            string `json:"name"`
	Binary          string `json:"binary"`
	State           string `json:"state"`
	PrimaryAction   string `json:"primary_action,omitempty"`
	OperatorCommand string `json:"operator_command,omitempty"`
	Reason          string `json:"reason"`
	InstallKind     string `json:"install_kind"`
	// InstallState is independent from update eligibility. A PATH-discovered
	// executable is found; only a verified receipt-owned prefix is managed.
	InstallState string `json:"install_state"`
	SignInState  string `json:"sign_in_state"`
	UpdateState  string `json:"update_state"`
	// Installed is what the copy that runs reports. ManagedVersion is the
	// package the managed link selects; the two differ while a copy shadows it.
	Installed         string               `json:"installed,omitempty"`
	ManagedVersion    string               `json:"managed_version,omitempty"`
	ManagedPath       string               `json:"managed_path,omitempty"`
	Pinned            string               `json:"pinned,omitempty"`
	Latest            string               `json:"latest,omitempty"`
	Status            upstreamcheck.Status `json:"status"`
	CommandPath       string               `json:"command_path,omitempty"`
	ExecutablePath    string               `json:"executable_path,omitempty"`
	OS                string               `json:"os"`
	Arch              string               `json:"arch"`
	CheckedAt         time.Time            `json:"checked_at"`
	Source            HarnessSource        `json:"source"`
	RollbackVersion   string               `json:"rollback_version,omitempty"`
	NewerThanReviewed bool                 `json:"newer_than_reviewed"`
	Checks            []HarnessCheck       `json:"checks"`
}

// gateResult is the Dependency Analyzer's answer for one exact version.
type gateResult struct {
	Reachable bool
	Blocked   bool
	Verdict   string
	Message   string
}

// harnessDeps carries the two seams with no local fixture: the network and the
// dependency gateway. Everything else observes the real filesystem and PATH.
type harnessDeps struct {
	desktop         bool
	installRelease  func(context.Context, Harness, string, harnessDeps) error
	platform        platformDeps
	latest          func(ctx context.Context, h Harness) (HarnessSource, string, error)
	gate            func(ctx context.Context, h Harness, version string, platform platformDeps) gateResult
	readApproval    func(ctx context.Context, h Harness, platform platformDeps) (approvalRecord, error)
	writeApproval   func(ctx context.Context, h Harness, record approvalRecord, platform platformDeps) error
	install         func(ctx context.Context, spec Spec, packageRef string) error
	installWithDeps func(ctx context.Context, spec Spec, packageRef string, platform platformDeps) error
	// lookPath selects like launch does: skipping the Vrooli shim.
	lookPath func(string) (string, error)
	version  func(ctx context.Context, path string) (string, error)
	rootOwn  func(os.FileInfo) bool
	now      func() time.Time
	record   ArtifactRecorder
	// actor names who is acting, in the approval record this writes.
	actor string
}

func defaultHarnessDeps() harnessDeps {
	return harnessDeps{
		platform: defaultPlatformDeps(),
		version:  InstalledVersion,
		gate: func(ctx context.Context, h Harness, version string, platform platformDeps) gateResult {
			return dependencyGateWithPlatform(ctx, h, version, platform)
		},
		readApproval: func(ctx context.Context, h Harness, platform platformDeps) (approvalRecord, error) {
			return readApprovalRecord(ctx, h, platform)
		},
		writeApproval: func(ctx context.Context, h Harness, record approvalRecord, platform platformDeps) error {
			return writeApprovalRecord(ctx, h, record, platform)
		},
		install:         installNPM,
		installWithDeps: installNPMWithDeps,
		lookPath:        cliutil.ResolveAgentBinary,
		rootOwn:         ownedByRoot,
		now:             time.Now,
	}
}

// latestHarnessRelease reports what an update would actually install. For an
// npm-acquired harness that is the package registry, even where upstream also
// publishes a release channel: a channel version not yet on npm cannot install.
func latestHarnessRelease(ctx context.Context, h Harness) (HarnessSource, string, error) {
	return latestHarnessReleaseWithDeps(ctx, h, defaultPlatformDeps())
}

func latestHarnessReleaseWithDeps(ctx context.Context, h Harness, platform platformDeps) (HarnessSource, string, error) {
	if h.ManifestBase != "" {
		platformID := releasePlatformWith(platform)
		release, err := fetchReleaseManifestWithDeps(ctx, h.ManifestBase, platformID, platform)
		return HarnessSource{Kind: "release-manifest", ID: platformID}, release.Version, err
	}
	if h.Package == "" {
		return HarnessSource{}, "", fmt.Errorf("this resource declares no update source")
	}
	handlers := upstreamcheck.Default(upstreamcheck.Config{SourceKind: upstreamcheck.SourceNPM, SourceID: h.Package})
	latest, err := handlers.LatestFetcher(ctx, upstreamcheck.SourceNPM, h.Package)
	return HarnessSource{Kind: string(upstreamcheck.SourceNPM), ID: h.Package}, upstreamcheck.ExtractSemver(latest), err
}

// dependencyGate asks the gateway about one exact version with a dry run, the
// same request the install makes. A blocked verdict exits non-zero and still
// prints its receipt, so the receipt decides.
func dependencyGate(ctx context.Context, h Harness, version string) gateResult {
	return dependencyGateWithPlatform(ctx, h, version, defaultPlatformDeps())
}

func dependencyGateWithPlatform(ctx context.Context, h Harness, version string, platform platformDeps) gateResult {
	output, _ := platform.withDefaults().commandOutput(ctx, dependencyAnalyzerCommand, "deps", "install", "npm/"+h.Package+"@"+version, "--scenario", h.Resource, "--surface", "host-tool", "--exact", "--json")
	var receipt governancev1.InstallDependencyResponse
	if err := (protojson.UnmarshalOptions{DiscardUnknown: true}).Unmarshal(output, &receipt); err != nil || receipt.Verdict == "" {
		return gateResult{Message: "The Dependency Analyzer did not answer."}
	}
	return gateResult{Reachable: true, Blocked: receipt.Blocked, Verdict: receipt.Verdict, Message: receipt.Message}
}

// InspectHarness observes one harness without changing anything.
func InspectHarness(ctx context.Context, root, resource string) (HarnessReport, error) {
	harness, err := LoadHarness(root, resource)
	if err != nil {
		return HarnessReport{}, err
	}
	return inspectHarness(ctx, harness, defaultHarnessDeps()), nil
}

func inspectHarness(ctx context.Context, h Harness, deps harnessDeps) HarnessReport {
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
	report := HarnessReport{
		Name: h.Resource, Binary: h.Binary, Pinned: h.ReviewedVersion, Status: upstreamcheck.StatusUnknown, InstallKind: InstallKindUnknown,
		InstallState: "broken", SignInState: "unknown", UpdateState: "unknown",
		OS: runtime.GOOS, Arch: runtime.GOARCH, CheckedAt: deps.now().UTC(), Checks: []HarnessCheck{},
	}
	fail := func(reason string) HarnessReport {
		report.State, report.PrimaryAction, report.Reason = StateCheckFailed, ActionRetry, reason
		return report
	}
	link, err := h.readLink()
	if err != nil {
		return fail(fmt.Sprintf("Could not read %s: %v", h.linkPath(), err))
	}
	if link.Kind == linkManaged {
		report.ManagedVersion = link.Version
		report.ManagedPath = h.linkPath()
	}
	selected, startup := h.selectedCopy(deps)
	report.CommandPath = selected
	if selected != "" {
		report.InstallState = "found"
		report.SignInState = inspectSignIn(ctx, selected, h.SignInProbe)
		report.ExecutablePath = selected
		if resolved, err := filepath.EvalSymlinks(selected); err == nil {
			report.ExecutablePath = resolved
		}
		if raw, err := deps.version(ctx, selected); err == nil {
			report.Installed = upstreamcheck.ExtractSemver(raw)
		}
	}
	report.NewerThanReviewed = report.Pinned != "" && report.Installed != "" && upstreamcheck.Compare(report.Pinned, report.Installed) == upstreamcheck.StatusBehind

	// The comparison is reported in every state so Manage can show it, but it
	// only becomes an update offer in the last branch below.
	source, latest, latestErr := deps.latest(ctx, h)
	report.Source, report.Latest = source, latest
	if latestErr == nil && report.Installed != "" {
		report.Status = upstreamcheck.Compare(report.Installed, latest)
		switch report.Status {
		case upstreamcheck.StatusBehind:
			report.UpdateState = "available"
		case upstreamcheck.StatusUpToDate, upstreamcheck.StatusAhead:
			report.UpdateState = "current"
		}
	}
	addCheck := func(id string, passed bool, message string) {
		report.Checks = append(report.Checks, HarnessCheck{ID: id, Passed: passed, Message: message})
	}

	if selected == "" {
		installs, err := h.managedInstalls()
		if err != nil {
			return fail("Could not verify the installed copies: " + err.Error())
		}
		if len(installs) > 0 {
			report.State, report.PrimaryAction, report.InstallKind = StateUnlinked, ActionLinkManagedCopy, InstallKindManaged
			report.InstallState, report.ManagedVersion = "managed", installs[0].Version
			report.Reason = fmt.Sprintf("Vrooli's verified %s is installed, but its launcher is not linked.", installs[0].Version)
			addCheck("managed_link", false, "Link the installed copy into your user-owned bin directory.")
			return report
		}
		report.State, report.PrimaryAction, report.InstallKind = StateMissing, ActionInstall, InstallKindUnknown
		report.InstallState = "absent"
		report.Reason = fmt.Sprintf("%s is not installed on this machine.", h.Binary)
		addCheck("selected_copy", false, "No "+h.Binary+" is on PATH.")
		return report
	}

	if blocker := h.rootOwnedBlocker(selected, deps); blocker != "" && !safeguardCovers(blocker, h.Home) {
		// Setup only hands back files under the operator's home, so offering its
		// command here would be a promise it does not keep.
		report.InstallKind = InstallKindRootOwned
		addCheck("ownership", false, blocker+" is owned by root and outside your home.")
		return fail(fmt.Sprintf("%s belongs to root and is outside your home, so setup will not change it. Fix its owner or unset the setting that points there.", blocker))
	} else if blocker != "" {
		report.State, report.OperatorCommand, report.InstallKind = StateNeedsSetup, NeedsSetupCommand, InstallKindRootOwned
		report.Reason = fmt.Sprintf("%s belongs to root, so Web Console cannot change it. Setup hands it back to you.", blocker)
		addCheck("ownership", false, blocker+" is owned by root or not writable.")
		return report
	}
	addCheck("ownership", true, "The files this update replaces are yours.")

	selectedIsLink := sameFile(selected, h.linkPath())
	if !selectedIsLink {
		if !withinDir(selected, h.Home) && pathOwnedByRoot(selected, deps) {
			report.State, report.InstallKind, report.InstallState = StateFound, InstallKindRootOwned, "found"
			if report.UpdateState == "available" {
				report.UpdateState = "blocked(system_copy)"
			}
			report.Reason = fmt.Sprintf("A system copy at %s runs first. Put %s ahead of %s on PATH or remove the system package.", selected, h.BinDir, filepath.Dir(selected))
			addCheck("selected_copy", false, selected+" runs first and belongs to the system.")
			return report
		}
		report.State, report.InstallKind, report.InstallState = StateFound, InstallKindUpstreamInstaller, "found"
		if report.UpdateState == "available" {
			report.UpdateState = "blocked(found_copy)"
		}
		report.Reason = fmt.Sprintf("An older copy at %s runs first", selected)
		if report.Installed != "" {
			report.Reason = fmt.Sprintf("A copy of %s at %s runs first", report.Installed, selected)
		}
		if report.ManagedVersion != "" {
			report.Reason += fmt.Sprintf(", while Vrooli's managed %s remains available", report.ManagedVersion)
		}
		report.Reason += "."
		if startup {
			report.Reason += " Your shell startup puts it on PATH."
		}
		addCheck("selected_copy", false, selected+" runs instead of "+h.linkPath()+".")
		return report
	}
	addCheck("selected_copy", true, h.linkPath()+" is what runs.")

	if link.Kind != linkManaged {
		report.State, report.InstallKind, report.InstallState = StateFound, InstallKindStandalone, "found"
		report.UpdateState = "blocked(found_copy)"
		report.Reason = fmt.Sprintf("A standalone copy at %s was found. It remains unchanged and is not managed by Vrooli.", h.linkPath())
		addCheck("managed_link", false, h.linkPath()+" is not a link to Vrooli's copy.")
		return report
	}
	report.InstallKind = InstallKindManaged
	report.InstallState = "managed"
	addCheck("managed_link", true, h.linkPath()+" links to "+link.Version+".")

	if installs, err := h.managedInstalls(); err == nil {
		report.RollbackVersion = previousInstall(installs, link.Version).Version
	}
	if report.Installed == "" {
		return fail(fmt.Sprintf("%s did not report a version.", selected))
	}
	if report.Installed != link.Version {
		if h.DelegatePath != "" {
			report.State = StateFound
			report.UpdateState = "blocked(found_copy)"
			report.Reason = fmt.Sprintf("Vrooli's %s launcher delegates to the found copy at %s, which reports %s. The found copy remains unchanged.", link.Version, h.DelegatePath, report.Installed)
			addCheck("selected_copy", false, "The package launcher runs "+h.DelegatePath+".")
			return report
		}
		return fail(fmt.Sprintf("Vrooli's %s package reports version %s.", link.Version, report.Installed))
	}
	if latestErr != nil {
		addCheck("upstream", false, latestErr.Error())
		return fail("Could not read the latest version.")
	}
	addCheck("upstream", true, fmt.Sprintf("%s publishes %s.", source.ID, latest))
	switch report.Status {
	case upstreamcheck.StatusUpToDate, upstreamcheck.StatusAhead:
		report.State, report.Reason = StateUpToDate, fmt.Sprintf("%s is current.", report.Installed)
		return report
	case upstreamcheck.StatusBehind:
	default:
		return fail(fmt.Sprintf("Cannot compare %s with %q.", report.Installed, latest))
	}
	if reason := h.updateBlocker(ctx, latest, addCheck, &report, deps); reason != "" {
		return fail(reason)
	}
	report.State, report.PrimaryAction = StateUpdateReady, ActionUpdate
	report.Reason = fmt.Sprintf("Version %s is ready to install. %s", latest, sessionImpact)
	return report
}

// updateBlocker checks what can still fail an otherwise eligible update. It
// fills the report's command when the operator has something to run.
func (h Harness) updateBlocker(ctx context.Context, version string, addCheck func(string, bool, string), report *HarnessReport, deps harnessDeps) string {
	if h.ManifestBase != "" {
		addCheck("updater", true, "This agent updates from its verified release manifest.")
		return ""
	}
	if h.Package == "" {
		addCheck("updater", false, "This resource declares no updater.")
		return "This agent has no updater."
	}
	if deps.desktop {
		if h.NativeArtifact == nil {
			return "This platform has no verified native artifact route."
		}
		addCheck("updater", true, "Installs the upstream native artifact after verifying its published integrity.")
		return ""
	}
	addCheck("updater", true, "Installs through the Dependency Analyzer.")
	if reason := h.nodeBlocker(ctx, addCheck, deps.platform); reason != "" {
		return reason
	}
	gate := deps.gate(ctx, h, version, deps.platform)
	switch {
	case !gate.Reachable:
		addCheck("dependency_gate", false, gate.Message)
		report.OperatorCommand = "vrooli scenario start scenario-dependency-analyzer"
		return "Installs need the Dependency Analyzer, which is not answering."
	case !gate.Blocked:
		addCheck("dependency_gate", true, fmt.Sprintf("Version %s is approved.", version))
	case gate.Verdict == "out_of_range":
		// The approval is exact, so every release starts out of range. Updating
		// records the next exact version through the same gateway.
		addCheck("dependency_gate", true, fmt.Sprintf("Updating records %s as the approved version.", version))
	default:
		addCheck("dependency_gate", false, gate.Message)
		return fmt.Sprintf("The dependency policy refuses this package (%s).", gate.Verdict)
	}
	return ""
}

// nodeBlocker checks the runtime npm and a node-launched package both need.
func (h Harness) nodeBlocker(ctx context.Context, addCheck func(string, bool, string), platform platformDeps) string {
	raw, err := platform.withDefaults().commandOutput(ctx, "node", "--version")
	version := upstreamcheck.ExtractSemver(string(raw))
	if err != nil || version == "" {
		addCheck("node", false, "node is not on PATH.")
		return "Installing this package needs Node.js, which is not on PATH."
	}
	if minimum := h.requiredNodeMajor(); minimum > 0 {
		major, _ := strconv.Atoi(strings.SplitN(version, ".", 2)[0])
		if major < minimum {
			addCheck("node", false, fmt.Sprintf("Node.js %s is older than the %d this package needs to run.", version, minimum))
			return fmt.Sprintf("This package needs Node.js %d or newer to run. This machine has %s.", minimum, version)
		}
	}
	addCheck("node", true, "Node.js "+version+".")
	return ""
}

// requiredNodeMajor reads engines.node from the selected package, the only copy
// on disk; the next release could raise it. A package whose entry point is a
// native binary is not launched by node, and npm only warns about engines, so
// there the field cannot fail an update.
func (h Harness) requiredNodeMajor() int {
	link, err := h.readLink()
	if err != nil || link.Kind != linkManaged || h.Package == "" {
		return 0
	}
	if !launchedByNode(filepath.Join(link.Prefix, filepath.FromSlash(h.Executable))) {
		return 0
	}
	data, err := os.ReadFile(filepath.Join(link.Prefix, "node_modules", filepath.FromSlash(h.Package), "package.json"))
	if err != nil {
		return 0
	}
	return nodeMajorFromPackage(data)
}

func launchedByNode(path string) bool {
	file, err := os.Open(path)
	if err != nil {
		return false
	}
	defer file.Close()
	header := make([]byte, 128)
	count, _ := file.Read(header)
	line, _, _ := strings.Cut(string(header[:count]), "\n")
	return strings.HasPrefix(line, "#!") && strings.Contains(line, "node")
}

// selectedCopy reports the copy a launch runs. A shell-started session reads
// startup files the API's own PATH did not, so a dir an upstream installer wrote
// into them counts as first even when this process's PATH orders it later.
func (h Harness) selectedCopy(deps harnessDeps) (path string, viaStartup bool) {
	selected, err := deps.lookPath(h.Binary)
	if err != nil {
		selected = ""
	}
	for _, candidate := range h.InstallerPaths {
		if _, err := os.Stat(candidate); err != nil {
			continue
		}
		if selected != "" && sameFile(candidate, selected) {
			return selected, shellStartupMentions(h.Home, filepath.Dir(candidate))
		}
		if shellStartupMentions(h.Home, filepath.Dir(candidate)) {
			return candidate, true
		}
	}
	return selected, false
}

// shellStartupMentions reads, and never edits, the operator's startup files.
func shellStartupMentions(home, dir string) bool {
	relative, err := filepath.Rel(home, dir)
	if err != nil {
		return false
	}
	spellings := []string{dir, "$HOME/" + relative, "${HOME}/" + relative, "~/" + relative}
	for _, name := range []string{".bashrc", ".bash_profile", ".profile", ".zshrc", ".zprofile", ".zshenv", ".config/fish/config.fish"} {
		data, err := os.ReadFile(filepath.Join(home, name))
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(strings.TrimSpace(line), "#") {
				continue
			}
			for _, spelling := range spellings {
				if strings.Contains(line, spelling) {
					return true
				}
			}
		}
	}
	return false
}

// rootOwnedBlocker returns the first path an action would have to change but
// cannot. BlockingSystemInstall only inspects the directory's mode bits, which
// misses a root-owned file inside a user-writable directory.
func (h Harness) rootOwnedBlocker(selected string, deps harnessDeps) string {
	candidates := []string{h.linkPath()}
	if withinDir(selected, h.Home) {
		candidates = append(candidates, selected)
	}
	if h.DelegatePath != "" {
		candidates = append(candidates, h.DelegatePath)
	}
	if root, err := h.versionsRoot(); err == nil {
		candidates = append(candidates, root)
	}
	for _, candidate := range candidates {
		if pathOwnedByRoot(candidate, deps) {
			return candidate
		}
	}
	if withinDir(selected, h.Home) && !sameFile(selected, h.linkPath()) && !writableByCaller(filepath.Dir(selected)) {
		return filepath.Dir(selected)
	}
	if _, err := os.Stat(h.BinDir); err == nil && !writableByCaller(h.BinDir) {
		return h.BinDir
	}
	return ""
}

func pathOwnedByRoot(path string, deps harnessDeps) bool {
	info, err := os.Lstat(path)
	return err == nil && deps.rootOwn(info)
}

func sameFile(a, b string) bool {
	left, err := os.Stat(a)
	if err != nil {
		return false
	}
	right, err := os.Stat(b)
	return err == nil && os.SameFile(left, right)
}

func withinDir(path, dir string) bool {
	relative, err := filepath.Rel(dir, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

// previousInstall is the newest complete version older than current.
func previousInstall(installs []managedInstall, current string) managedInstall {
	for _, install := range installs {
		if upstreamcheck.Compare(install.Version, current) == upstreamcheck.StatusBehind {
			return install
		}
	}
	return managedInstall{}
}
