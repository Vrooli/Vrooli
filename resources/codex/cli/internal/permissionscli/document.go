package permissionscli

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"regexp"
	"strconv"
	"time"

	"github.com/vrooli/agentharness"
	"github.com/vrooli/vrooli/internal/cliout"
	"github.com/vrooli/vrooli/resources/codex/cli/internal/permissions"
)

var codexPermissionPosture = agentharness.EnforcementPosture{Permissions: "hook_unverified", Caveats: []string{"Native execution configuration and hook presence do not prove effective enforcement. Verify runtime version, configuration layers and a fresh-session canary; staged profiles keep networking disabled until activation."}}

func (h *Handlers) Plan(args []string) error {
	fs, scope := h.flagSet("permissions plan")
	path := fs.String("document", "", "Path to desired permission document, or - for stdin")
	executable := fs.String("codex-executable", "", "Absolute native executable whose version should be qualified (default PATH codex)")
	activate := fs.Bool("activate", false, "Preview selecting the native profile and approval settings")
	asJSON := fs.Bool("json", false, "Emit JSON")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("--document is required")
	}
	result, _, _, err := h.planDocument(*path, *scope, *activate, *executable)
	if err != nil {
		return err
	}
	return h.writePlan(result, *asJSON)
}

func (h *Handlers) Reconcile(args []string) error {
	fs, scope := h.flagSet("permissions reconcile")
	path := fs.String("document", "", "Path to desired permission document, or - for stdin")
	executable := fs.String("codex-executable", "", "Absolute native executable whose version should be qualified (default PATH codex)")
	activate := fs.Bool("activate", false, "Select the native profile and approval settings")
	digest := fs.String("expected-digest", "", "Preview digest required for execution changes")
	asJSON := fs.Bool("json", false, "Emit JSON")
	authorized := fs.Bool("i-was-explicitly-authorized", false, "Override agent gate only when a human explicitly authorized this call")
	if err := fs.Parse(args); err != nil {
		return err
	}
	if *path == "" {
		return errors.New("--document is required")
	}
	result, desired, a, err := h.planDocument(*path, *scope, *activate, *executable)
	if err != nil {
		return err
	}
	if desired.Execution != nil && *digest == "" {
		return errors.New("execution reconciliation requires --expected-digest from permissions plan with the same --activate selection")
	}
	if *digest != "" && *digest != result.PreviewDigest {
		return errors.New("preview digest differs; native files or requested settings changed; plan again")
	}
	if err = h.gate("permissions reconcile", true, *authorized); err != nil {
		return err
	}
	if err = a.Reconcile(desired, h.CLIVersion, result.StateDrift); err != nil {
		return err
	}
	result.RecoveryBackup = a.LastBackup

	return h.writePlan(result, *asJSON)
}

func (h *Handlers) planDocument(path, scope string, activate bool, executable string) (agentharness.PermissionPlanResult, permissions.Policy, *permissions.Adapter, error) {
	fail := func(err error) (agentharness.PermissionPlanResult, permissions.Policy, *permissions.Adapter, error) {
		return agentharness.PermissionPlanResult{}, permissions.Policy{}, nil, err
	}
	a, err := h.adapter(scope)
	if err != nil {
		return fail(err)
	}
	d, data, err := agentharness.LoadPermissionDocumentForRunner(path, h.Stdin, "codex")
	if err != nil {
		return fail(err)
	}
	if d.Scope != "" && d.Scope != string(a.Scope) {
		return fail(fmt.Errorf("document scope %q does not match --scope %q", d.Scope, a.Scope))
	}
	d.Scope = string(a.Scope)
	if activate && d.Execution == nil {
		return fail(errors.New("--activate requires execution intent"))
	}
	var runtimeVersion string
	var versionCommand []string
	if d.Execution != nil {
		runtimeVersion, versionCommand, err = h.requirePermissionProfiles(executable, *d.Execution.Network.Enabled)
		if err != nil {
			return fail(err)
		}
	} else if executable != "" {
		return fail(errors.New("--codex-executable requires execution intent"))
	}
	live, err := a.Load()
	if err != nil {
		return fail(err)
	}
	allow, ask, deny := agentharness.PermissionPatterns(d)
	desired := permissions.Policy{BashAllow: allow, BashAsk: ask, BashDeny: deny, Execution: d.Execution, ExecutionActive: activate, SnapshotDigest: live.SnapshotDigest, NativeFingerprint: live.NativeFingerprint}
	if d.Execution != nil {
		if err = a.CheckExecutionProjection(desired); err != nil {
			return fail(err)
		}
		desired.NativeFingerprint = permissions.DesiredExecutionFingerprint(d.Execution, activate)
	}
	desired.HookFingerprint = a.DesiredHookFingerprint(desired)
	paths := []string{a.SettingsPath, a.StatePath()}
	if a.HookPath != "" {
		paths = append(paths, a.HookPath)
	}
	result := agentharness.PlanPermissionProjection("codex", d, data, agentharness.PermissionProjection{Allow: live.BashAllow, Ask: live.BashAsk, Deny: live.BashDeny}, paths, codexPermissionPosture)
	if live.NativeFingerprint != desired.NativeFingerprint {
		result.Changes = append(result.Changes, "replace managed native execution profile or activation settings")
	}
	if live.HookFingerprint != desired.HookFingerprint {
		result.Changes = append(result.Changes, "reconcile managed PreToolUse hook")
	}
	result.LiveFingerprint = permissions.Fingerprint(live)
	result.DesiredFingerprint = permissions.Fingerprint(desired)
	stateSnapshot, err := agentharness.ReadPermissionFile(a.StatePath())
	if err != nil {
		return fail(err)
	}
	desired.StateSnapshotDigest = agentharness.PermissionFilesDigest(stateSnapshot)
	state, err := a.LoadState()
	if err != nil {
		return fail(err)
	}
	if state != nil && state.SchemaVersion > agentharness.PermissionStateSchemaVersion {
		return fail(fmt.Errorf("future state schema %d is unsupported; use the owning resource version", state.SchemaVersion))
	}
	result.StateDrift = state == nil || state.SchemaVersion != agentharness.PermissionStateSchemaVersion || state.Fingerprint != result.DesiredFingerprint
	if result.StateDrift {
		result.Changes = append(result.Changes, "publish managed permission state")
	}
	result.Drift = result.LiveFingerprint != result.DesiredFingerprint || result.StateDrift
	if d.Execution != nil {
		result.Execution = &agentharness.ExecutionPlan{Desired: d.Execution, NativeNetworkEnabled: activate && *d.Execution.Network.Enabled, Capability: agentharness.PermissionExecutionCapability("codex"), Profile: permissions.ExecutionProfile, Activate: activate, RuntimeEvidence: "unverified", RuntimeVersion: runtimeVersion, VersionCommand: versionCommand}
	}
	// Bind the preview to the source document, exact native snapshots and activation.
	payload, _ := json.Marshal(struct {
		Document, Files, State, Desired, Version, Executable string
		Activate                                             bool
	}{result.DesiredDigest, live.SnapshotDigest, desired.StateSnapshotDigest, result.DesiredFingerprint, runtimeVersion, executable, activate})
	sum := sha256.Sum256(payload)
	result.PreviewDigest = hex.EncodeToString(sum[:])
	return result, desired, a, nil
}

var codexVersion = regexp.MustCompile(`(?:^|\s)(\d+)\.(\d+)\.(\d+)(?:\s|$|[-+])`)

func (h *Handlers) requirePermissionProfiles(executable string, networkEnabled bool) (string, []string, error) {
	command := append([]string(nil), h.VersionCommand...)
	if executable != "" {
		if !filepath.IsAbs(executable) {
			return "", nil, errors.New("--codex-executable must be absolute")
		}
		command = []string{executable, "--version"}
	}
	if h.VersionRunner == nil {
		return "", nil, errors.New("installed Codex version cannot be verified")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	output, err := h.VersionRunner(ctx, command)
	if err != nil {
		return "", nil, fmt.Errorf("verify installed Codex version: %w", err)
	}
	match := codexVersion.FindStringSubmatch(output)
	if match == nil {
		return "", nil, fmt.Errorf("unrecognized Codex version %q", output)
	}
	major, _ := strconv.Atoi(match[1])
	minor, _ := strconv.Atoi(match[2])
	if major == 0 && minor < 138 {
		return "", nil, fmt.Errorf("native permission profiles require Codex >= 0.138.0; installed %s", output)
	}
	patch, _ := strconv.Atoi(match[3])
	if networkEnabled && major == 0 && (minor < 156 || (minor == 156 && patch < 1)) {
		return "", nil, fmt.Errorf("filtered networking requires a qualified Codex runtime >= 0.156.1; installed %s; select a qualified installation with --codex-executable and rerun the native canary", output)
	}
	return output, command, nil
}

func (h *Handlers) writePlan(result agentharness.PermissionPlanResult, asJSON bool) error {
	if asJSON {
		data, err := cliout.MarshalIndent(result)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintln(h.Stdout, string(data))
		return err
	}
	fmt.Fprintf(h.Stdout, "runner=%s scope=%s drift=%t changes=%d enforcement=%s\npreview_digest=%s\n", result.Runner, result.Scope, result.Drift, len(result.Changes), result.Enforcement.Permissions, result.PreviewDigest)
	for _, change := range result.Changes {
		fmt.Fprintln(h.Stdout, change)
	}
	if result.RecoveryBackup != "" {
		fmt.Fprintf(h.Stdout, "recovery_backup=%s\n", result.RecoveryBackup)
	}
	return nil
}
