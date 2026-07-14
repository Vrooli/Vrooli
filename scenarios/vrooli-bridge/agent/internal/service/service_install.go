package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"

	"vrooli-bridge/agent/internal/platform"
)

// commandRunner is the exec seam the OS managers drive their native tool through
// (systemctl / launchctl). Production runs os/exec; tests substitute a fake that
// records argv and returns canned output, so the exact command sequence — and
// the idempotent re-install path — is unit-testable without touching the host's
// service manager.
type commandRunner interface {
	run(ctx context.Context, argv ...string) (string, error)
}

// execRunner is the production commandRunner. It execs the argv directly (never
// through a shell) and returns combined stdout+stderr.
type execRunner struct{}

func (execRunner) run(ctx context.Context, argv ...string) (string, error) {
	if len(argv) == 0 {
		return "", errors.New("empty argv")
	}
	// #nosec G204 — argv is a fixed, code-constructed token list (a native
	// service-manager subcommand plus the resolved unit path/label); it is never
	// a shell string and never attacker-influenced.
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return string(out), fmt.Errorf("%s: %w: %s", strings.Join(argv, " "), err, strings.TrimSpace(string(out)))
	}
	return string(out), nil
}

// errRenderOnly is returned by the render-only managers' install operations. The
// operator installs the rendered unit with the platform's own tooling instead.
func errRenderOnly(goos string) error {
	return fmt.Errorf("service install is render-only on %s; render the unit with --print-service-unit and install it with the platform's native tooling", goos)
}

// ---------------------------------------------------------------------------
// systemd (Linux)
// ---------------------------------------------------------------------------

// systemdManager installs the agent as a systemd --user unit under
// ~/.config/systemd/user. A user unit (not a system unit) keeps install
// unprivileged — no root, no /etc — matching the dedicated non-privileged
// service principal (DECISIONS.md). The user manager runs the service whenever
// the owning user is logged in; on a headless host enable auto-start across
// reboots with `loginctl enable-linger <user>` (the Linux analogue of the macOS
// auto-login requirement, surfaced in the phase-8 runbook).
type systemdManager struct {
	unitDir func() (string, error)
	runner  commandRunner
}

func newSystemdManager() systemdManager {
	return systemdManager{unitDir: systemdUserUnitDir, runner: execRunner{}}
}

func (systemdManager) Kind() platform.ServiceManagerKind { return platform.ServiceManagerSystemd }

func (systemdManager) Render(d Definition) (string, error) { return SystemdUnit(d) }

func (systemdManager) unitName(d Definition) string { return d.Name + ".service" }

func (m systemdManager) unitPath(d Definition) (string, error) {
	dir, err := m.unitDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, m.unitName(d)), nil
}

func (m systemdManager) Install(ctx context.Context, d Definition) (InstallResult, error) {
	unit, err := SystemdUnit(d)
	if err != nil {
		return InstallResult{}, err
	}
	dir, err := m.unitDir()
	if err != nil {
		return InstallResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("create systemd user unit dir %q: %w", dir, err)
	}
	unitName := m.unitName(d)
	unitPath := filepath.Join(dir, unitName)
	// Rewrite the whole unit every time (never append) so the on-disk unit always
	// reflects the current Definition and the write itself is replay-safe.
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil {
		return InstallResult{}, fmt.Errorf("write systemd user unit %q: %w", unitPath, err)
	}
	if _, err := m.runner.run(ctx, "systemctl", "--user", "daemon-reload"); err != nil {
		return InstallResult{}, err
	}
	if _, err := m.runner.run(ctx, "systemctl", "--user", "enable", unitName); err != nil {
		return InstallResult{}, err
	}
	// restart, not start: on a re-install the service may already be running an
	// older unit, and only restart re-execs it with the freshly-written content.
	// On a first install restart simply starts it. This is the idempotent
	// converge point — running Install twice ends in the same live state.
	if _, err := m.runner.run(ctx, "systemctl", "--user", "restart", unitName); err != nil {
		return InstallResult{}, err
	}
	return InstallResult{
		Kind:     platform.ServiceManagerSystemd,
		UnitName: unitName,
		UnitPath: unitPath,
		Enabled:  true,
		Running:  true,
	}, nil
}

func (m systemdManager) Status(ctx context.Context, d Definition) (StatusResult, error) {
	unitPath, err := m.unitPath(d)
	if err != nil {
		return StatusResult{}, err
	}
	res := StatusResult{Kind: platform.ServiceManagerSystemd, UnitName: m.unitName(d), UnitPath: unitPath}
	if _, statErr := os.Stat(unitPath); statErr == nil {
		res.Installed = true
	}
	// `systemctl show` exits 0 even for an unknown unit (LoadState=not-found), so
	// its output — not the exit code — is the source of truth; ignore the error.
	out, _ := m.runner.run(ctx, "systemctl", "--user", "show", m.unitName(d),
		"--property=ActiveState,UnitFileState,MainPID,LoadState")
	props := parseSystemctlShow(out)
	res.Running = props["ActiveState"] == "active"
	res.Enabled = strings.HasPrefix(props["UnitFileState"], "enabled")
	res.PID = atoiSafe(props["MainPID"])
	res.Detail = fmt.Sprintf("ActiveState=%s UnitFileState=%s MainPID=%s",
		props["ActiveState"], props["UnitFileState"], props["MainPID"])
	return res, nil
}

func (m systemdManager) Uninstall(ctx context.Context, d Definition) (UninstallResult, error) {
	unitPath, err := m.unitPath(d)
	if err != nil {
		return UninstallResult{}, err
	}
	unitName := m.unitName(d)
	// Stop + disable first; ignore the error so uninstall is idempotent even when
	// the unit was never installed or is already stopped.
	_, _ = m.runner.run(ctx, "systemctl", "--user", "disable", "--now", unitName)
	res := UninstallResult{Kind: platform.ServiceManagerSystemd, UnitName: unitName, UnitPath: unitPath}
	if err := os.Remove(unitPath); err == nil {
		res.Removed = true
	} else if !os.IsNotExist(err) {
		return UninstallResult{}, fmt.Errorf("remove systemd user unit %q: %w", unitPath, err)
	}
	// Reload so the user manager forgets the removed unit; best-effort.
	_, _ = m.runner.run(ctx, "systemctl", "--user", "daemon-reload")
	return res, nil
}

func systemdUserUnitDir() (string, error) {
	base, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("resolve user config dir: %w", err)
	}
	return filepath.Join(base, "systemd", "user"), nil
}

// parseSystemctlShow parses `systemctl show`'s KEY=VALUE lines into a map.
func parseSystemctlShow(out string) map[string]string {
	props := map[string]string{}
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if k, v, ok := strings.Cut(line, "="); ok {
			props[k] = v
		}
	}
	return props
}

func atoiSafe(s string) int {
	n, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil {
		return 0
	}
	return n
}

// ---------------------------------------------------------------------------
// launchd (macOS)
// ---------------------------------------------------------------------------

// launchdManager installs the agent as a per-user launchd LaunchAgent under
// ~/Library/LaunchAgents and bootstraps it into the user's gui/<uid> domain.
//
// A LaunchAgent only runs while its user is logged in to a GUI session — so a
// headless Mac mini must have auto-login enabled for the agent to come up after
// reboot (the operator-assisted step in the phase-8 runbook; there is no headless
// equivalent of systemd linger for a gui-domain agent). The darwin path is
// covered by argv-level unit tests here and awaits the real mac run in phase 8;
// no mac evidence is fabricated.
type launchdManager struct {
	agentDir func() (string, error)
	uid      func() int
	runner   commandRunner
}

func newLaunchdManager() launchdManager {
	return launchdManager{agentDir: launchdAgentDir, uid: os.Getuid, runner: execRunner{}}
}

func (launchdManager) Kind() platform.ServiceManagerKind { return platform.ServiceManagerLaunchd }

func (launchdManager) Render(d Definition) (string, error) { return LaunchdPlist(d) }

func (m launchdManager) domainTarget() string { return fmt.Sprintf("gui/%d", m.uid()) }

func (m launchdManager) serviceTarget(d Definition) string {
	return m.domainTarget() + "/" + LaunchdLabel(d.Name)
}

func (m launchdManager) plistPath(d Definition) (string, error) {
	dir, err := m.agentDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, LaunchdLabel(d.Name)+".plist"), nil
}

func (m launchdManager) Install(ctx context.Context, d Definition) (InstallResult, error) {
	plist, err := LaunchdPlist(d)
	if err != nil {
		return InstallResult{}, err
	}
	dir, err := m.agentDir()
	if err != nil {
		return InstallResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return InstallResult{}, fmt.Errorf("create LaunchAgents dir %q: %w", dir, err)
	}
	plistPath := filepath.Join(dir, LaunchdLabel(d.Name)+".plist")
	if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil {
		return InstallResult{}, fmt.Errorf("write launchd plist %q: %w", plistPath, err)
	}
	// bootstrapping an already-loaded agent errors, so boot it out first and
	// ignore the "not loaded" failure on a fresh install. This is what makes a
	// re-install idempotent: it always converges on the freshly-written plist.
	_, _ = m.runner.run(ctx, "launchctl", "bootout", m.serviceTarget(d))
	if _, err := m.runner.run(ctx, "launchctl", "enable", m.serviceTarget(d)); err != nil {
		return InstallResult{}, err
	}
	if _, err := m.runner.run(ctx, "launchctl", "bootstrap", m.domainTarget(), plistPath); err != nil {
		return InstallResult{}, err
	}
	// kickstart -k restarts the service if it was already running so a re-install
	// always ends with the current plist's process live.
	if _, err := m.runner.run(ctx, "launchctl", "kickstart", "-k", m.serviceTarget(d)); err != nil {
		return InstallResult{}, err
	}
	return InstallResult{
		Kind:     platform.ServiceManagerLaunchd,
		UnitName: LaunchdLabel(d.Name),
		UnitPath: plistPath,
		Enabled:  true,
		Running:  true,
	}, nil
}

func (m launchdManager) Status(ctx context.Context, d Definition) (StatusResult, error) {
	plistPath, err := m.plistPath(d)
	if err != nil {
		return StatusResult{}, err
	}
	res := StatusResult{Kind: platform.ServiceManagerLaunchd, UnitName: LaunchdLabel(d.Name), UnitPath: plistPath}
	if _, statErr := os.Stat(plistPath); statErr == nil {
		res.Installed = true
		// The plist is present and Install always enables + bootstraps it, so a
		// present plist means it is set to run at load.
		res.Enabled = true
	}
	// `launchctl print` exits non-zero when the label is not bootstrapped; parse
	// the output rather than the exit code.
	out, _ := m.runner.run(ctx, "launchctl", "print", m.serviceTarget(d))
	res.Running = strings.Contains(out, "state = running")
	res.PID = parseLaunchctlPID(out)
	res.Detail = launchctlStateDetail(out)
	return res, nil
}

func (m launchdManager) Uninstall(ctx context.Context, d Definition) (UninstallResult, error) {
	plistPath, err := m.plistPath(d)
	if err != nil {
		return UninstallResult{}, err
	}
	// Ignore bootout failure: the agent may already be unloaded, and removing the
	// plist below is the durable part of the uninstall.
	_, _ = m.runner.run(ctx, "launchctl", "bootout", m.serviceTarget(d))
	res := UninstallResult{Kind: platform.ServiceManagerLaunchd, UnitName: LaunchdLabel(d.Name), UnitPath: plistPath}
	if err := os.Remove(plistPath); err == nil {
		res.Removed = true
	} else if !os.IsNotExist(err) {
		return UninstallResult{}, fmt.Errorf("remove launchd plist %q: %w", plistPath, err)
	}
	return res, nil
}

func launchdAgentDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve user home dir: %w", err)
	}
	return filepath.Join(home, "Library", "LaunchAgents"), nil
}

// parseLaunchctlPID extracts the `pid = N` line from `launchctl print` output.
func parseLaunchctlPID(out string) int {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if rest, ok := strings.CutPrefix(line, "pid = "); ok {
			return atoiSafe(rest)
		}
	}
	return 0
}

// launchctlStateDetail extracts the `state = …` line from `launchctl print`.
func launchctlStateDetail(out string) string {
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "state = ") {
			return line
		}
	}
	return ""
}

// ---------------------------------------------------------------------------
// Windows (render-only) + unsupported
// ---------------------------------------------------------------------------

// windowsManager renders the `sc.exe create` argv but does not install: Windows
// stays render-only this phase (the Kind()/renderer already model the argv shape
// so a later phase can add the real install without reshaping the abstraction).
type windowsManager struct{}

func (windowsManager) Kind() platform.ServiceManagerKind { return platform.ServiceManagerWindows }

func (windowsManager) Render(d Definition) (string, error) {
	args, err := WindowsServiceCreateArgs(d)
	if err != nil {
		return "", err
	}
	return "sc.exe " + strings.Join(args, " "), nil
}

func (windowsManager) Install(context.Context, Definition) (InstallResult, error) {
	return InstallResult{}, errRenderOnly("windows")
}

func (windowsManager) Status(context.Context, Definition) (StatusResult, error) {
	return StatusResult{}, errRenderOnly("windows")
}

func (windowsManager) Uninstall(context.Context, Definition) (UninstallResult, error) {
	return UninstallResult{}, errRenderOnly("windows")
}

// unsupportedManager is returned for a GOOS with no native service manager; the
// agent runs in the foreground instead.
type unsupportedManager struct{}

func (unsupportedManager) Kind() platform.ServiceManagerKind { return platform.ServiceManagerUnknown }

func (unsupportedManager) Render(Definition) (string, error) {
	return "", fmt.Errorf("no native service manager for GOOS %q; run the agent in the foreground", runtime.GOOS)
}

func (unsupportedManager) Install(context.Context, Definition) (InstallResult, error) {
	return InstallResult{}, errRenderOnly(runtime.GOOS)
}

func (unsupportedManager) Status(context.Context, Definition) (StatusResult, error) {
	return StatusResult{}, errRenderOnly(runtime.GOOS)
}

func (unsupportedManager) Uninstall(context.Context, Definition) (UninstallResult, error) {
	return UninstallResult{}, errRenderOnly(runtime.GOOS)
}
