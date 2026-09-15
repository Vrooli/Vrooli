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

	platformgo "github.com/vrooli/platform-go"

	"vrooli-bridge/agent/internal/platform"
)

// commandRunner is the exec seam the OS managers drive their native tool through
// (systemctl / launchctl / sc.exe). Production runs os/exec; tests substitute a fake that
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

// errRenderOnly is returned only for unsupported host platforms. Supported
// platforms have a native install manager; the operator can still inspect its
// rendered artifact before installation.
func errRenderOnly(goos string) error {
	return fmt.Errorf("no native service manager is implemented for GOOS %q; run the agent in the foreground", goos)
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
	artifact, err := d.Artifact("linux")
	if err != nil {
		return InstallResult{}, err
	}
	// Ask systemd whether it would load the render before the unit
	// directory is touched: a rejected render must never replace a working
	// unit. An unavailable validator is not a rejection.
	if verdict := platformgo.ValidateArtifact(artifact, systemdScope(d)); verdict.Rejected() {
		return InstallResult{}, fmt.Errorf("systemd rejected the rendered %s unit: %s", d.Name, verdict.Output)
	}
	unit := artifact.Primary().Content
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
		res.Configured = true
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

// launchdUserDomains returns the two per-user namespaces that can contain a
// stale Bridge LaunchAgent. An SSH-only install converges on a system
// LaunchDaemon, but a previous GUI-session install may still be loaded under
// gui/<uid>, and older headless installs may be under user/<uid>. Evicting only
// this exact managed label prevents two agents with different node identities
// from running concurrently after a re-enrollment.
func launchdUserDomains(uid int) []string {
	if uid < 0 {
		return nil
	}
	return []string{fmt.Sprintf("gui/%d", uid), fmt.Sprintf("user/%d", uid)}
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
	plist, err := validatedLaunchdPlist(d)
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
		// The definition becomes system-scope so the plist carries UserName.
		d.System = true
		if strings.TrimSpace(d.User) == "" {
			d.User, err = currentLaunchdUser()
			if err != nil {
				return InstallResult{}, err
			}
		}
		plist, err = validatedLaunchdPlist(d)
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
	if systemScope {
		// A system daemon is the canonical service for SSH/headless installs.
		// Remove any same-label per-user service as well; otherwise launchd can
		// keep an older node process alive and race the freshly installed daemon.
		// These are exact fixed Bridge labels, and failures are intentionally
		// ignored because either domain may not exist on this host.
		for _, userDomain := range launchdUserDomains(m.uid()) {
			_, _ = runSudo(ctx, m.runner, "launchctl", "bootout", userDomain+"/"+LaunchdLabel(d.Name))
		}
	}
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
		res.Configured = true
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
// Windows Service Control Manager
// ---------------------------------------------------------------------------

// windowsManager drives the Windows Service Control Manager through sc.exe.
// The command runner is injected for tests so command construction and state
// parsing are covered on non-Windows hosts without pretending to have native
// Windows evidence. The installed agent is responsible for entering the SCM
// service dispatcher when it is launched by the service manager.
type windowsManager struct{ runner commandRunner }

func newWindowsManager() windowsManager { return windowsManager{runner: execRunner{}} }

func (windowsManager) Kind() platform.ServiceManagerKind { return platform.ServiceManagerWindows }

func (windowsManager) Render(d Definition) (string, error) {
	args, err := WindowsServiceCreateArgs(d)
	if err != nil {
		return "", err
	}
	return "sc.exe " + strings.Join(args, " "), nil
}

func (m windowsManager) Install(ctx context.Context, d Definition) (InstallResult, error) {
	if err := d.validate(); err != nil {
		return InstallResult{}, err
	}
	if strings.TrimSpace(d.User) == "" {
		return InstallResult{}, errors.New("windows service user is required")
	}
	if m.runner == nil {
		return InstallResult{}, errors.New("windows service command runner is required")
	}
	_, installed, err := m.query(ctx, d)
	if err != nil {
		return InstallResult{}, err
	}
	if installed {
		// `sc config` is the convergent path for a re-install: it updates the
		// executable, arguments, account, and auto-start policy without deleting
		// a service that may currently own an active node identity.
		if _, err := m.runner.run(ctx, "sc.exe", "config", d.Name,
			"binPath=", d.execLine(),
			"start=", "auto",
			"DisplayName=", fallback(d.Description, d.Name),
			"obj=", d.User); err != nil {
			return InstallResult{}, fmt.Errorf("configure Windows service %q: %w", d.Name, err)
		}
	} else {
		args, err := WindowsServiceCreateArgs(d)
		if err != nil {
			return InstallResult{}, err
		}
		if _, err := m.runner.run(ctx, append([]string{"sc.exe"}, args...)...); err != nil {
			return InstallResult{}, fmt.Errorf("create Windows service %q: %w", d.Name, err)
		}
	}
	if out, err := m.runner.run(ctx, "sc.exe", "start", d.Name); err != nil && !windowsAlreadyRunning(out, err) {
		return InstallResult{}, fmt.Errorf("start Windows service %q: %w", d.Name, err)
	}
	status, err := m.Status(ctx, d)
	if err != nil {
		return InstallResult{}, err
	}
	if !status.Configured {
		return InstallResult{}, fmt.Errorf("Windows service %q does not match the requested executable or principal: %s", d.Name, status.Detail)
	}
	if !status.Running {
		return InstallResult{}, fmt.Errorf("Windows service %q did not reach running state: %s", d.Name, status.Detail)
	}
	return InstallResult{
		Kind:     platform.ServiceManagerWindows,
		UnitName: d.Name,
		Enabled:  status.Enabled,
		Running:  true,
	}, nil
}

func (m windowsManager) Status(ctx context.Context, d Definition) (StatusResult, error) {
	if err := d.validate(); err != nil {
		return StatusResult{}, err
	}
	if m.runner == nil {
		return StatusResult{}, errors.New("windows service command runner is required")
	}
	query, installed, err := m.query(ctx, d)
	if err != nil {
		return StatusResult{}, err
	}
	res := StatusResult{Kind: platform.ServiceManagerWindows, UnitName: d.Name}
	if !installed {
		res.Detail = "SERVICE_STATUS=not-installed"
		return res, nil
	}
	qc, err := m.runner.run(ctx, "sc.exe", "qc", d.Name)
	if err != nil {
		return StatusResult{}, fmt.Errorf("query Windows service configuration %q: %w", d.Name, err)
	}
	running, state := parseSCState(query)
	res.Installed = true
	res.Configured = windowsServiceConfigurationMatches(qc, d)
	res.Running = running
	res.Enabled = parseSCAutoStart(qc)
	res.PID = parseSCPID(query)
	res.Detail = fmt.Sprintf("STATE=%s PID=%d START_TYPE=%s CONFIGURED=%t", state, res.PID, parseSCStartType(qc), res.Configured)
	if !res.Configured {
		res.Detail += " CONFIGURATION_MISMATCH"
	}
	return res, nil
}

func (m windowsManager) Uninstall(ctx context.Context, d Definition) (UninstallResult, error) {
	if err := d.validate(); err != nil {
		return UninstallResult{}, err
	}
	if m.runner == nil {
		return UninstallResult{}, errors.New("windows service command runner is required")
	}
	_, installed, err := m.query(ctx, d)
	if err != nil {
		return UninstallResult{}, err
	}
	res := UninstallResult{Kind: platform.ServiceManagerWindows, UnitName: d.Name}
	if !installed {
		return res, nil
	}
	if out, err := m.runner.run(ctx, "sc.exe", "stop", d.Name); err != nil && !windowsNotActive(out, err) {
		return UninstallResult{}, fmt.Errorf("stop Windows service %q: %w", d.Name, err)
	}
	if out, err := m.runner.run(ctx, "sc.exe", "delete", d.Name); err != nil && !windowsServiceMissing(out, err) {
		return UninstallResult{}, fmt.Errorf("delete Windows service %q: %w", d.Name, err)
	}
	res.Removed = true
	return res, nil
}

func (m windowsManager) query(ctx context.Context, d Definition) (string, bool, error) {
	out, err := m.runner.run(ctx, "sc.exe", "queryex", d.Name)
	if err != nil {
		if windowsServiceMissing(out, err) {
			return "", false, nil
		}
		return "", false, fmt.Errorf("query Windows service %q: %w", d.Name, err)
	}
	return out, true, nil
}

func windowsServiceMissing(output string, err error) bool {
	text := strings.ToLower(output + " " + errorText(err))
	return strings.Contains(text, "1060") || strings.Contains(text, "does not exist") || strings.Contains(text, "not exist")
}

func windowsAlreadyRunning(output string, err error) bool {
	text := strings.ToLower(output + " " + errorText(err))
	return strings.Contains(text, "1056") || strings.Contains(text, "already running")
}

func windowsNotActive(output string, err error) bool {
	text := strings.ToLower(output + " " + errorText(err))
	return strings.Contains(text, "1062") || strings.Contains(text, "not started") || strings.Contains(text, "not active")
}

func errorText(err error) string {
	if err == nil {
		return ""
	}
	return err.Error()
}

func parseSCState(output string) (bool, string) {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "STATE") {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			continue
		}
		if len(fields) == 1 {
			return fields[0] == "4", fields[0]
		}
		return fields[0] == "4", strings.Join(fields, " ")
	}
	return false, "UNKNOWN"
}

func parseSCPID(output string) int { return parseSCIntField(output, "PID") }

func parseSCStartType(output string) string {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), "START_TYPE") {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if ok {
			return strings.TrimSpace(value)
		}
	}
	return "UNKNOWN"
}

func parseSCAutoStart(output string) bool {
	return strings.HasPrefix(strings.TrimSpace(parseSCStartType(output)), "2")
}

func windowsServiceConfigurationMatches(output string, d Definition) bool {
	binaryPath := strings.TrimSpace(parseSCStringField(output, "BINARY_PATH_NAME"))
	serviceUser := strings.TrimSpace(parseSCStringField(output, "SERVICE_START_NAME"))
	return normalizeSCValue(binaryPath) == normalizeSCValue(d.execLine()) &&
		normalizeSCValue(serviceUser) == normalizeSCValue(d.User)
}

func parseSCStringField(output, field string) string {
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, field) {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if ok {
			return strings.TrimSpace(value)
		}
	}
	return ""
}

func normalizeSCValue(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func parseSCIntField(output, field string) int {
	for _, line := range strings.Split(output, "\n") {
		if !strings.HasPrefix(strings.TrimSpace(line), field) {
			continue
		}
		_, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		fields := strings.Fields(value)
		if len(fields) == 0 {
			continue
		}
		n, err := strconv.Atoi(fields[0])
		if err == nil {
			return n
		}
	}
	return 0
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

// systemdScope maps the Definition's namespace onto the validator's.
func systemdScope(d Definition) platformgo.Scope {
	if d.System {
		return platformgo.ScopeSystem
	}
	return platformgo.ScopeUser
}

// validatedLaunchdPlist renders the plist and runs it through plutil where
// available; a rejected plist is never written.
func validatedLaunchdPlist(d Definition) (string, error) {
	artifact, err := d.Artifact("darwin")
	if err != nil {
		return "", err
	}
	if verdict := platformgo.ValidateArtifact(artifact, systemdScope(d)); verdict.Rejected() {
		return "", fmt.Errorf("plutil rejected the rendered %s plist: %s", d.Name, verdict.Output)
	}
	return artifact.Primary().Content, nil
}
