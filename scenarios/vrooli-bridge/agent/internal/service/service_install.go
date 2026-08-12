package service

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"os/user"
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
	dir, err := m.unitDirFor(d)
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, m.unitName(d)), nil
}

func (m systemdManager) unitDirFor(d Definition) (string, error) {
	if d.System {
		return "/etc/systemd/system", nil
	}
	return m.unitDir()
}

func systemdArgs(d Definition, command ...string) []string {
	if d.System {
		return append([]string{"systemctl"}, command...)
	}
	return append([]string{"systemctl", "--user"}, command...)
}

func (m systemdManager) Install(ctx context.Context, d Definition) (InstallResult, error) {
	unit, err := SystemdUnit(d)
	if err != nil {
		return InstallResult{}, err
	}
	dir, err := m.unitDirFor(d)
	if err != nil {
		return InstallResult{}, err
	}
	if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- native service managers require a traversable unit directory.
		return InstallResult{}, fmt.Errorf("create systemd user unit dir %q: %w", dir, err)
	}
	unitName := m.unitName(d)
	unitPath := filepath.Join(dir, unitName)
	// Rewrite the whole unit every time (never append) so the on-disk unit always
	// reflects the current Definition and the write itself is replay-safe.
	if err := os.WriteFile(unitPath, []byte(unit), 0o644); err != nil { // #nosec G306 -- unit files contain no secrets and must be readable by the supervisor.
		return InstallResult{}, fmt.Errorf("write systemd user unit %q: %w", unitPath, err)
	}
	if _, err := m.runner.run(ctx, systemdArgs(d, "daemon-reload")...); err != nil {
		return InstallResult{}, err
	}
	if _, err := m.runner.run(ctx, systemdArgs(d, "enable", unitName)...); err != nil {
		return InstallResult{}, err
	}
	// restart, not start: on a re-install the service may already be running an
	// older unit, and only restart re-execs it with the freshly-written content.
	// On a first install restart simply starts it. This is the idempotent
	// converge point — running Install twice ends in the same live state.
	if _, err := m.runner.run(ctx, systemdArgs(d, "restart", unitName)...); err != nil {
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
	out, _ := m.runner.run(ctx, append(systemdArgs(d, "show", m.unitName(d)), "--property=ActiveState,UnitFileState,MainPID,LoadState")...)
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
	_, _ = m.runner.run(ctx, systemdArgs(d, "disable", "--now", unitName)...)
	res := UninstallResult{Kind: platform.ServiceManagerSystemd, UnitName: unitName, UnitPath: unitPath}
	if err := os.Remove(unitPath); err == nil {
		res.Removed = true
	} else if !os.IsNotExist(err) {
		return UninstallResult{}, fmt.Errorf("remove systemd user unit %q: %w", unitPath, err)
	}
	// Reload so the user manager forgets the removed unit; best-effort.
	_, _ = m.runner.run(ctx, systemdArgs(d, "daemon-reload")...)
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
	domain   func() string
	runner   commandRunner
}

func newLaunchdManager() launchdManager {
	return launchdManager{agentDir: launchdAgentDir, uid: os.Getuid, domain: func() string { return resolveLaunchdDomain(os.Getuid()) }, runner: execRunner{}}
}

func (launchdManager) Kind() platform.ServiceManagerKind { return platform.ServiceManagerLaunchd }

func (launchdManager) Render(d Definition) (string, error) { return LaunchdPlist(d) }

func (m launchdManager) domainTarget() string {
	if m.domain != nil {
		return m.domain()
	}
	return fmt.Sprintf("gui/%d", m.uid())
}

func (m launchdManager) plistPath(d Definition) (string, error) {
	if d.System {
		return launchdSystemPath(d), nil
	}
	dir, err := m.agentDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, LaunchdLabel(d.Name)+".plist"), nil
}

const launchdSystemDir = "/Library/LaunchDaemons"

func launchdSystemPath(d Definition) string {
	return filepath.Join(launchdSystemDir, LaunchdLabel(d.Name)+".plist")
}

func runSudo(ctx context.Context, runner commandRunner, argv ...string) (string, error) {
	return runner.run(ctx, append([]string{"sudo", "-n"}, argv...)...)
}

func currentLaunchdUser() (string, error) {
	u, err := user.Current()
	if err != nil {
		return "", fmt.Errorf("resolve launchd service user: %w", err)
	}
	if strings.TrimSpace(u.Username) == "" {
		return "", errors.New("resolve launchd service user: empty username")
	}
	return u.Username, nil
}

func (m launchdManager) Install(ctx context.Context, d Definition) (InstallResult, error) {
	plist, err := LaunchdPlist(d)
	if err != nil {
		return InstallResult{}, err
	}
	domain := m.domainTarget()
	systemScope := d.System || strings.HasPrefix(domain, "user/")
	plistPath := ""
	stagedPath := ""
	if systemScope {
		// SSH-only macOS sessions have no gui/<uid> bootstrap. A LaunchAgent
		// cannot be loaded into the background user domain, so use a
		// LaunchDaemon and explicitly retain the unprivileged agent identity.
		if strings.TrimSpace(d.User) == "" {
			d.User, err = currentLaunchdUser()
			if err != nil {
				return InstallResult{}, err
			}
		}
		plist, err = LaunchdPlist(d)
		if err != nil {
			return InstallResult{}, err
		}
		stagedFile, createErr := os.CreateTemp("", "vrooli-bridge-agent-*.plist")
		if createErr != nil {
			return InstallResult{}, fmt.Errorf("stage launchd daemon plist: %w", createErr)
		}
		stagedPath = stagedFile.Name()
		if closeErr := stagedFile.Close(); closeErr != nil {
			_ = os.Remove(stagedPath)
			return InstallResult{}, fmt.Errorf("close staged launchd daemon plist: %w", closeErr)
		}
		if err := os.WriteFile(stagedPath, []byte(plist), 0o644); err != nil { // #nosec G306 -- staged plist contains no secrets and is immediately installed as a root-owned unit.
			_ = os.Remove(stagedPath)
			return InstallResult{}, fmt.Errorf("write staged launchd daemon plist: %w", err)
		}
		defer os.Remove(stagedPath)
		plistPath = launchdSystemPath(d)
		if _, err := runSudo(ctx, m.runner, "/usr/bin/install", "-o", "root", "-g", "wheel", "-m", "0644", stagedPath, plistPath); err != nil {
			return InstallResult{}, fmt.Errorf("install launchd daemon plist %q: %w", plistPath, err)
		}
	} else {
		dir, dirErr := m.agentDir()
		if dirErr != nil {
			return InstallResult{}, dirErr
		}
		if err := os.MkdirAll(dir, 0o755); err != nil { // #nosec G301 -- LaunchAgents must be searchable by launchd.
			return InstallResult{}, fmt.Errorf("create LaunchAgents dir %q: %w", dir, err)
		}
		plistPath = filepath.Join(dir, LaunchdLabel(d.Name)+".plist")
		if err := os.WriteFile(plistPath, []byte(plist), 0o644); err != nil { // #nosec G306 -- plist contains no secrets and launchd reads it as the supervisor.
			return InstallResult{}, fmt.Errorf("write launchd plist %q: %w", plistPath, err)
		}
	}
	serviceDomain := domain
	if systemScope {
		serviceDomain = "system"
	}
	target := serviceDomain + "/" + LaunchdLabel(d.Name)
	run := m.runner.run
	if systemScope {
		run = func(ctx context.Context, argv ...string) (string, error) {
			return runSudo(ctx, m.runner, argv...)
		}
	}
	// Bootstrapping an already-loaded agent errors, so boot it out first and
	// ignore the "not loaded" failure on a fresh install. This is what makes a
	// re-install idempotent: it always converges on the freshly-written plist.
	_, _ = run(ctx, "launchctl", "bootout", target)
	if _, err := run(ctx, "launchctl", "bootstrap", serviceDomain, plistPath); err != nil {
		return InstallResult{}, err
	}
	// A launchd service must be loaded before its per-service enable state can
	// be changed. Enabling before bootstrap is rejected by modern macOS with
	// exit 125 ("Domain does not support specified action").
	if _, err := run(ctx, "launchctl", "enable", target); err != nil {
		return InstallResult{}, err
	}
	// kickstart -k restarts the service if it was already running so a re-install
	// always ends with the current plist's process live.
	if _, err := run(ctx, "launchctl", "kickstart", "-k", target); err != nil {
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
	domain := m.domainTarget()
	systemScope := d.System || strings.HasPrefix(domain, "user/")
	plistPath, err := m.plistPath(d)
	if err != nil {
		return StatusResult{}, err
	}
	if systemScope {
		plistPath = launchdSystemPath(d)
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
	serviceDomain := domain
	if systemScope {
		serviceDomain = "system"
	}
	target := serviceDomain + "/" + LaunchdLabel(d.Name)
	var out string
	if systemScope {
		out, _ = runSudo(ctx, m.runner, "launchctl", "print", target)
	} else {
		out, _ = m.runner.run(ctx, "launchctl", "print", target)
	}
	res.Running = strings.Contains(out, "state = running")
	res.PID = parseLaunchctlPID(out)
	res.Detail = launchctlStateDetail(out)
	return res, nil
}

func (m launchdManager) Uninstall(ctx context.Context, d Definition) (UninstallResult, error) {
	domain := m.domainTarget()
	systemScope := d.System || strings.HasPrefix(domain, "user/")
	plistPath, err := m.plistPath(d)
	if err != nil {
		return UninstallResult{}, err
	}
	if systemScope {
		plistPath = launchdSystemPath(d)
	}
	// Ignore bootout failure: the agent may already be unloaded, and removing the
	// plist below is the durable part of the uninstall.
	serviceDomain := domain
	if systemScope {
		serviceDomain = "system"
	}
	target := serviceDomain + "/" + LaunchdLabel(d.Name)
	if systemScope {
		_, _ = runSudo(ctx, m.runner, "launchctl", "bootout", target)
	} else {
		_, _ = m.runner.run(ctx, "launchctl", "bootout", target)
	}
	res := UninstallResult{Kind: platform.ServiceManagerLaunchd, UnitName: LaunchdLabel(d.Name), UnitPath: plistPath}
	if systemScope {
		if _, rmErr := runSudo(ctx, m.runner, "/bin/rm", "-f", plistPath); rmErr != nil {
			return UninstallResult{}, fmt.Errorf("remove launchd daemon plist %q: %w", plistPath, rmErr)
		}
		res.Removed = true
		return res, nil
	}
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

// resolveLaunchdDomain selects the per-user launchd namespace that exists in
// the current session. A logged-in desktop user has a gui/<uid> bootstrap;
// SSH-only/headless sessions on modern macOS expose user/<uid> instead. Using
// gui/<uid> unconditionally makes bootstrap fail with exit 125 on a headless
// Mac mini even though the user launchd domain is available.
func resolveLaunchdDomain(uid int) string {
	gui := fmt.Sprintf("gui/%d", uid)
	if exec.Command("launchctl", "print", gui).Run() == nil { // #nosec G204 -- launchctl is fixed and gui is a locally-derived numeric domain.
		return gui
	}
	return fmt.Sprintf("user/%d", uid)
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
