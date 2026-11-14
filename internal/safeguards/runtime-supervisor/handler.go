package runtimesupervisorsafeguard

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"strings"

	platformgo "github.com/vrooli/platform-go"
	repocontract "github.com/vrooli/repo-contract-go"
	"github.com/vrooli/vrooli/internal/buildinfo"
	"github.com/vrooli/vrooli/internal/config"
	"github.com/vrooli/vrooli/internal/coreactivation"
	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
	"github.com/vrooli/vrooli/internal/runtimesupervisor"
)

// Seams stubbed by tests. Every host interaction goes through one of these so
// a unit test never touches the real service manager.
var (
	userHomeFn      = hostreqkit.InvokingUserHomeDir
	resolveRootFn   = repocontract.ResolveRepoRoot
	commandOutputFn = func(name string, args ...string) ([]byte, error) { return hostreqkit.CombinedOutputFn(name, args...) }
	runCommandFn    = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		return hostreqkit.RunCommandFn(name, args, opts)
	}
	installFileFn         = hostreqkit.InstallUserFile
	restoreDefinitionFn   = func(path, content string) error { return config.WriteOwnedFileDurable(path, []byte(content), 0644) }
	validateFn            = platformgo.ValidateArtifact
	processImageMatchesFn = processImageMatches
	executableFn          = func(home, requested string) (string, bool, error) {
		return runtimesupervisor.ExecutablePath(home, requested)
	}
	acquireInstallLockFn  = buildinfo.AcquireBinaryInstallLockContext
	guardRootActivationFn = func(ctx context.Context, home string) error {
		store, err := coreactivation.NewStore(home)
		if err != nil {
			return err
		}
		return store.GuardOrdinaryMutation(ctx)
	}
	probeRootBootstrapFn = func(ctx context.Context, root, executable string) error {
		if err := coreactivation.ValidateRootExecutableRole(executable); err != nil {
			return err
		}
		identity, err := coreactivation.ProbeRootIdentity(ctx, executable)
		if err != nil {
			return err
		}
		expected, err := nativeDefinitionHomeDigest(root)
		if err != nil {
			return err
		}

		if identity.GetRuntimeHomeContractSha256() != expected {
			return fmt.Errorf("installed runtime-home authority differs from the native definition authority")
		}

		return nil
	}
)

// Compiled native authority survives source loss. Invalid compiled metadata is
// an error, never permission to change authority through source fallback.
func nativeDefinitionHomeDigest(root string) (string, error) {
	if buildinfo.RuntimeHomeSnapshotBase64 != "" {
		digest, err := buildinfo.RuntimeHomeSnapshotDigest()
		if err != nil {
			return "", fmt.Errorf("compiled runtime-home authority is invalid: %w", err)
		}
		return digest, nil
	}
	c, err := repocontract.LoadDefault(root)
	if err != nil {
		return "", err
	}
	data, err := c.RuntimeHomeSnapshot()
	if err != nil {
		return "", err
	}
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:]), nil
}

type handler struct{ manifest hostreqkit.SafeguardManifest }

// processImageMatches compares the live Linux image's file identity with the
// installed binary. procfs retains the original inode across atomic updates.
// A deleted image or a different inode needs adoption through a unit restart.
func processImageMatches(pid int, executable string) (bool, error) {
	if pid <= 0 {
		return false, fmt.Errorf("invalid process id")
	}
	running, err := os.Stat(filepath.Join("/proc", strconv.Itoa(pid), "exe"))
	if err != nil {
		return false, fmt.Errorf("read running executable: %w", err)
	}
	installed, err := os.Stat(executable)
	if err != nil {
		return false, fmt.Errorf("read installed executable: %w", err)
	}
	return os.SameFile(running, installed), nil
}

// NewHandler is the registry constructor.
func NewHandler(manifest hostreqkit.SafeguardManifest) hostreqkit.Handler {
	return handler{manifest: manifest}
}

func (h handler) Name() string           { return h.manifest.Name }
func (h handler) Kind() hostreqspec.Kind { return hostreqspec.KindSafeguard }

// Rendered is the supervisor unit for one host: the artifact, where it is
// installed, and its native identity.
type Rendered struct {
	Artifact   platformgo.RenderedArtifact
	Path       string
	UnitName   string
	Executable string
	Canonical  bool
	LogPath    string
	Target     string
}

// Render builds the supervisor's unit for a host. requestedExecutable is the
// binary to name when no installed CLI exists yet; empty means the running
// one.
func Render(osName, home, root, requestedExecutable string) (Rendered, error) {
	target, err := platformgo.NormalizeTarget(renderTarget(osName))
	if err != nil {
		return Rendered{}, err
	}
	logPath, err := runtimesupervisor.LogPath(home)
	if err != nil {
		return Rendered{}, err
	}
	executable, canonical, err := executableFn(home, requestedExecutable)
	if err != nil {
		return Rendered{}, err
	}
	definition, err := platformgo.RuntimeSupervisorDefinition(target, platformgo.RuntimeSupervisorOptions{Home: home, Executable: executable, SourceRoot: root, LogPath: logPath})
	if err != nil {
		return Rendered{}, err
	}
	artifact, err := platformgo.RenderDefinition(definition, target)
	if err != nil {
		return Rendered{}, err
	}
	unit, _ := platformgo.CoreUnitByID(platformgo.CoreUnitRuntimeSupervisor)
	return Rendered{
		Artifact:   artifact,
		Path:       platformgo.RuntimeSupervisorUnitPath(target, home),
		UnitName:   unit.NativeName(target),
		Executable: executable,
		Canonical:  canonical,
		LogPath:    logPath,
		Target:     target,
	}, nil
}

func renderTarget(osName string) string {
	if isMacOS(osName) {
		return "darwin"
	}
	return osName
}

func isMacOS(osName string) bool {
	return osName == string(hostreqspec.PlatformDarwin) || osName == "macos"
}

func supported(host hostreqkit.Host) bool {
	return isMacOS(host.OS) || (hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux && host.SupportsSystemd)
}

func (h handler) Inspect(host hostreqkit.Host, requirement hostreqspec.ResolvedRequirement) hostreqkit.ItemStatus {
	status := hostreqkit.BaseStatus(requirement)
	status.SupportClass = hostreqkit.SupportSupported
	if requirement.Manual {
		status.SupportClass = hostreqkit.SupportManualOnly
		status.ExecutionState = hostreqkit.ExecutionManualActionRequired
		return status
	}
	if !supported(host) {
		status.SupportClass = hostreqkit.SupportUnsupported
		status.ExecutionState = hostreqkit.ExecutionUnsupported
		status.Notes = append(status.Notes, "the runtime supervisor has no setup-converged native unit on "+host.OS+"; Windows installs through the Service Control Manager path in packages/platform-go/service_windows.go")
		return status
	}
	if isMacOS(host.OS) && !guiLaunchdAvailable() {
		status.SupportClass = hostreqkit.SupportNotApplicable
		status.ExecutionState = hostreqkit.ExecutionNotApplicable
		status.Notes = append(status.Notes, "the invoking user's GUI launchd domain is unavailable; this user LaunchAgent is not applicable to the current SSH/headless session")
		return status
	}
	home, root, err := homeAndRoot()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status
	}
	rendered, err := Render(host.OS, home, root, "")
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "render runtime supervisor unit: "+err.Error())
		return status
	}
	verdict := validateFn(rendered.Artifact, platformgo.ScopeUser)
	recordVerdict(&status, verdict)
	status.Evidence["executable"] = rendered.Executable
	status.Evidence["executable_is_canonical"] = rendered.Canonical
	pending := false
	if !hostreqkit.FileContentMatches(rendered.Path, rendered.Artifact.Primary().Content) {
		pending = true
		status.Notes = append(status.Notes, driftNote(rendered))
	}
	state := unitState(host.OS, rendered.UnitName)
	if state.probeErr != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, "native scheduler bus unavailable: "+state.probeErr.Error(), recoveryNote())
		return status
	}
	if len(state.properties) > 0 {
		status.Evidence["unit_state"] = state.properties
	}
	if !state.enabled || !state.active {
		pending = true
		status.Notes = append(status.Notes, fmt.Sprintf("runtime supervisor unit pending: enabled=%t active=%t", state.enabled, state.active))
	}
	if state.active && hostreqspec.PlatformFromGOOS(host.OS) == hostreqspec.PlatformLinux {
		pid, current, imageErr := inspectUnitProcessImage(state, rendered.Executable)
		status.Evidence["process_image_current"] = current && imageErr == nil
		status.Evidence["process_image_pid"] = pid
		if imageErr != nil {
			pending = true
			status.Notes = append(status.Notes, "runtime supervisor process image could not be verified: "+imageErr.Error())
		} else if !current {
			pending = true
			status.Notes = append(status.Notes, "runtime supervisor process older than installed executable; refresh will restart the unit")
		}
	}
	if result := state.properties["Result"]; result != "" && result != "success" {
		status.Notes = append(status.Notes, "runtime supervisor unit last result is "+result+" after "+state.properties["NRestarts"]+" restarts; see "+rendered.LogPath)
	}
	if verdict.Rejected() {
		pending = true
		status.Notes = append(status.Notes, "the native validator rejects the rendered runtime supervisor unit: "+verdict.Output)
	}
	if pending {
		return status
	}
	status.Applied = true
	status.ExecutionState = hostreqkit.ExecutionAlreadyPresent
	status.Notes = append(status.Notes, "runtime supervisor unit matches its definition, is validated, enabled and active")
	return status
}

// driftNote names the drift instead of saying "stale": the readiness check
// and an operator both need to see which ExecStart the unit on disk carries.
func driftNote(rendered Rendered) string {
	note := "runtime supervisor unit is missing or stale: " + rendered.Path
	data, err := os.ReadFile(rendered.Path)
	if err != nil {
		return note
	}
	onDisk := execLine(string(data))
	want := execLine(rendered.Artifact.Primary().Content)
	if onDisk != "" && onDisk != want {
		return note + " (on-disk " + onDisk + "; rendered " + want + ")"
	}
	return note
}

func execLine(content string) string {
	for _, line := range strings.Split(content, "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "ExecStart=") || strings.Contains(trimmed, "<key>ProgramArguments</key>") {
			return trimmed
		}
	}
	return ""
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
	home, root, err := homeAndRoot()
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error())
		return status, nil
	}
	if opts.DryRun {
		status.ExecutionState = hostreqkit.ExecutionWouldApply
		status.Notes = append(status.Notes, "dry-run: would render, validate, install and enable the runtime supervisor unit")
		return status, nil
	}
	result, err := Converge(context.Background(), ConvergeOptions{OS: host.OS, Home: home, Root: root, Ensure: opts, Restart: status.Evidence["process_image_current"] == false})
	recordVerdict(&status, result.Verdict)
	if err != nil {
		status.ExecutionState = hostreqkit.ExecutionFailed
		status.Notes = append(status.Notes, err.Error(), recoveryNote())
		return status, nil
	}
	if result.Restarted {
		status.Notes = append(status.Notes, "restarted the runtime supervisor unit onto its new definition")
	}
	verified := h.Inspect(host, hostreqspec.ResolvedRequirement{Name: status.Name, Kind: status.Kind, Required: status.Required, Config: status.Config})
	verified.Notes = append(status.Notes, verified.Notes...)
	if !verified.Applied {
		verified.ExecutionState = hostreqkit.ExecutionFailed
		verified.Notes = append(verified.Notes, "unit converged but verification did not prove it active", recoveryNote())
		return verified, nil
	}
	verified.ExecutionState = hostreqkit.ExecutionApplied
	return verified, nil
}

// ConvergeOptions are the inputs to Converge.
type ConvergeOptions struct {
	// Set all three only for reviewed retained-definition restoration. These
	// checks run under the same canonical executable-install lock as writes.
	ExpectedExecutableSHA        string
	ExpectedCurrentDefinitionSHA string
	RetainedDefinitionSHA        string
	OS                           string
	Home                         string
	Root                         string
	Restart                      bool
	// Executable is the binary to name when no installed CLI exists yet.
	Executable string
	Ensure     hostreqkit.EnsureOptions
}

// ConvergeResult reports what Converge did.
type ConvergeResult struct {
	Rendered  Rendered
	Verdict   platformgo.Verdict
	Changed   bool
	Restarted bool
	Active    bool
	// ProcessImageVerified is established only by the Linux live-image probe.
	ProcessImageVerified bool
}

// Converge is the one install path for the supervisor unit: render, validate
// (a rejected render never replaces a working unit), install as the invoking
// user, reload and enable. It restarts for changed unit content or when an
// explicit reinstall requests a fresh executable. The setup safeguard remains
// idempotent when the unit definition is already current.
func Converge(ctx context.Context, options ConvergeOptions) (ConvergeResult, error) {
	if options.Ensure.DryRun && (options.ExpectedExecutableSHA != "" || options.ExpectedCurrentDefinitionSHA != "" || options.RetainedDefinitionSHA != "") {
		return ConvergeResult{}, fmt.Errorf("guarded retained-definition restoration does not execute dry-run requests")
	}
	if err := ctx.Err(); err != nil {
		return ConvergeResult{}, err
	}
	rendered, err := Render(options.OS, options.Home, options.Root, options.Executable)
	if err != nil {
		return ConvergeResult{}, fmt.Errorf("render runtime supervisor unit: %w", err)
	}
	result := ConvergeResult{Rendered: rendered}
	release, err := acquireInstallLockFn(ctx, rendered.Executable)
	if err != nil {
		return result, fmt.Errorf("lock root supervisor convergence: %w", err)
	}
	defer release()
	if err := guardRootActivationFn(ctx, options.Home); err != nil {
		return result, err
	}
	guarded := options.ExpectedExecutableSHA != "" || options.ExpectedCurrentDefinitionSHA != "" || options.RetainedDefinitionSHA != ""
	if guarded {
		for _, value := range []string{options.ExpectedExecutableSHA, options.ExpectedCurrentDefinitionSHA, options.RetainedDefinitionSHA} {
			decoded, err := hex.DecodeString(value)
			if err != nil || len(decoded) != 32 || value != strings.ToLower(value) {
				return result, fmt.Errorf("exact restoration identities required")
			}
		}
		image, err := coreactivation.ImageDigest(ctx, rendered.Executable)
		if err != nil || image != options.ExpectedExecutableSHA {
			return result, fmt.Errorf("restored executable differs from reviewed prior")
		}
		content := rendered.Artifact.Primary().Content
		sum := sha256.Sum256([]byte(content))
		if hex.EncodeToString(sum[:]) != options.RetainedDefinitionSHA {
			return result, fmt.Errorf("rendered definition differs from retained prior")
		}
		current, err := boundedDefinitionDigest(rendered.Path)
		if err != nil || current != options.ExpectedCurrentDefinitionSHA && current != options.RetainedDefinitionSHA {
			return result, fmt.Errorf("current supervisor definition is foreign; refused overwrite")
		}
	}
	if err := probeRootBootstrapFn(ctx, options.Root, rendered.Executable); err != nil {
		return result, fmt.Errorf("installed root CLI native bootstrap is unqualified; rebuild through the governed installer before changing its unit: %w", err)
	}
	result.Verdict = validateFn(rendered.Artifact, platformgo.ScopeUser)
	if result.Verdict.Rejected() {
		return result, fmt.Errorf("the native validator rejected the rendered %s; nothing was installed: %s", rendered.UnitName, result.Verdict.Output)
	}
	content := rendered.Artifact.Primary().Content
	result.Changed = !hostreqkit.FileContentMatches(rendered.Path, content)
	if rendered.LogPath != "" {
		// systemd creates the log file but not its directory; a missing
		// directory fails the unit at start with a message that does not
		// mention logging at all.
		if err := runCommandFn("mkdir", []string{"-p", filepath.Dir(rendered.LogPath)}, options.Ensure); err != nil {
			return result, fmt.Errorf("create supervisor log directory: %w", err)
		}
	}
	var installErr error
	if guarded {
		installErr = restoreDefinitionFn(rendered.Path, content)
	} else {
		installErr = installFileFn(rendered.Path, content, options.Ensure)
	}
	if err := installErr; err != nil {
		return result, fmt.Errorf("install %s: %w", rendered.Path, err)
	}
	if err := enableUnit(rendered, options.Ensure); err != nil {
		return result, fmt.Errorf("enable %s: %w", rendered.UnitName, err)
	}
	if result.Changed || options.Restart {
		if err := restartUnit(rendered, options.Ensure); err != nil {
			return result, fmt.Errorf("restart %s onto its new definition: %w", rendered.UnitName, err)
		}
		result.Restarted = true
	}
	state := unitState(options.OS, rendered.UnitName)
	if state.probeErr != nil {
		return result, fmt.Errorf("probe %s after install: %w", rendered.UnitName, state.probeErr)
	}
	result.Active = state.active
	if !state.active {
		return result, fmt.Errorf("installed %s but it is not active (Result=%s); see %s", rendered.UnitName, state.properties["Result"], rendered.LogPath)
	}
	if hostreqspec.PlatformFromGOOS(options.OS) == hostreqspec.PlatformLinux {
		_, current, err := inspectUnitProcessImage(state, rendered.Executable)
		if err != nil {
			return result, fmt.Errorf("verify %s running image after install: %w", rendered.UnitName, err)
		}
		if !current {
			return result, fmt.Errorf("installed %s is active but has not adopted the intended executable", rendered.UnitName)
		}
		result.ProcessImageVerified = true
	}
	return result, nil
}

func inspectUnitProcessImage(state unitProbe, executable string) (int, bool, error) {
	pid, err := strconv.Atoi(state.properties["MainPID"])
	if err != nil || pid <= 0 {
		return pid, false, fmt.Errorf("active unit has no valid MainPID")
	}
	current, err := processImageMatchesFn(pid, executable)
	return pid, current, err
}

func asUserArgs(name string, args ...string) (string, []string) {
	return hostreqkit.InvokingUserCommand(name, args...)
}

func enableUnit(rendered Rendered, opts hostreqkit.EnsureOptions) error {
	if rendered.Target == "darwin" {
		cmd, args := asUserArgs("launchctl", "bootstrap", "gui/"+currentUID(), rendered.Path)
		if err := runCommandFn(cmd, args, opts); err != nil {
			// bootstrap of an already-loaded agent fails; kickstart below
			// covers the reload.
			cmd, args = asUserArgs("launchctl", "kickstart", "gui/"+currentUID()+"/"+rendered.UnitName)
			return runCommandFn(cmd, args, opts)
		}
		return nil
	}
	for _, args := range [][]string{{"--user", "daemon-reload"}, {"--user", "enable", "--now", rendered.UnitName}} {
		cmd, wrapped := asUserArgs("systemctl", args...)
		if err := runCommandFn(cmd, wrapped, opts); err != nil {
			return err
		}
	}
	return nil
}

func restartUnit(rendered Rendered, opts hostreqkit.EnsureOptions) error {
	if rendered.Target == "darwin" {
		cmd, args := asUserArgs("launchctl", "kickstart", "-k", "gui/"+currentUID()+"/"+rendered.UnitName)
		return runCommandFn(cmd, args, opts)
	}
	// reset-failed first: a unit that exhausted StartLimitBurst refuses a
	// plain restart until its failure counter is cleared.
	for _, args := range [][]string{{"--user", "reset-failed", rendered.UnitName}, {"--user", "restart", rendered.UnitName}} {
		cmd, wrapped := asUserArgs("systemctl", args...)
		if err := runCommandFn(cmd, wrapped, opts); err != nil {
			return err
		}
	}
	return nil
}

type unitProbe struct {
	enabled    bool
	active     bool
	properties map[string]string
	probeErr   error
}

// unitState reads the properties that distinguish an active unit from one
// that is active but crash-looping. It is evidence; the readiness check
// decides what "zero restarts since boot" means.
func unitState(osName, name string) unitProbe {
	if isMacOS(osName) {
		cmd, args := asUserArgs("launchctl", "print", "gui/"+currentUID()+"/"+name)
		out, err := commandOutputFn(cmd, args...)
		loaded := err == nil
		return unitProbe{enabled: loaded, active: loaded && strings.Contains(string(out), "state = running"), properties: map[string]string{"launchctl": strings.TrimSpace(string(out))}}
	}
	cmd, args := asUserArgs("systemctl", "--user", "show", name, "-p", "ActiveState", "-p", "UnitFileState", "-p", "NRestarts", "-p", "Result", "-p", "MainPID")
	out, err := commandOutputFn(cmd, args...)
	if err != nil {
		return unitProbe{probeErr: fmt.Errorf("systemctl user bus: %s", strings.TrimSpace(string(out)))}
	}
	properties := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if key, value, ok := strings.Cut(line, "="); ok {
			properties[strings.TrimSpace(key)] = strings.TrimSpace(value)
		}
	}
	return unitProbe{
		enabled:    properties["UnitFileState"] == "enabled",
		active:     properties["ActiveState"] == "active",
		properties: properties,
	}
}

func homeAndRoot() (string, string, error) {
	home, err := userHomeFn()
	if err != nil || strings.TrimSpace(home) == "" {
		return "", "", fmt.Errorf("could not determine the invoking user's home directory")
	}
	root, err := resolveRootFn()
	if err != nil {
		return "", "", fmt.Errorf("could not determine Vrooli root: %w", err)
	}
	return home, root, nil
}

func guiLaunchdAvailable() bool {
	_, err := commandOutputFn("launchctl", "print", "gui/"+currentUID())
	return err == nil
}

func currentUID() string {
	if current, err := user.Current(); err == nil && current.Uid != "" {
		return current.Uid
	}
	return "0"
}

func recordVerdict(status *hostreqkit.ItemStatus, verdict platformgo.Verdict) {
	if status.Evidence == nil {
		status.Evidence = map[string]any{}
	}
	if verdict.State == "" {
		return
	}
	status.Evidence["validator_verdict"] = verdict
	if verdict.State == platformgo.VerdictUnavailable {
		status.Notes = append(status.Notes, "native validator unavailable; rendered unit is unproven: "+verdict.Output)
	}
}

func recoveryNote() string {
	return "recovery: rerun `vrooli setup` (or `vrooli runtime supervisor install --user`) and read " + runtimesupervisor.LogFileName + " under the Vrooli logs directory"
}

func boundedDefinitionDigest(path string) (string, error) {
	before, err := os.Lstat(path)
	if err != nil || !before.Mode().IsRegular() || before.Size() > 1<<20 {
		return "", fmt.Errorf("supervisor definition is not a bounded regular file")
	}
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()
	after, err := f.Stat()
	if err != nil || !os.SameFile(before, after) {
		return "", fmt.Errorf("supervisor definition changed while opening")
	}
	h := sha256.New()
	n, err := io.Copy(h, io.LimitReader(f, 1<<20+1))
	if err != nil || n > 1<<20 {
		return "", fmt.Errorf("supervisor definition exceeded bound")
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}
