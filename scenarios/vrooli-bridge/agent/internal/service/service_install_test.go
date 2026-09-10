package service

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"vrooli-bridge/agent/internal/platform"

	"github.com/stretchr/testify/require"
)

// fakeRunner records every argv it is asked to run and returns canned output for
// the first command whose joined argv contains a key substring (e.g. "show",
// "print"), so a manager's native-tool sequence is asserted without touching the
// host's systemd/launchd.
type fakeRunner struct {
	calls   [][]string
	outputs map[string]string // key substring (matched against the joined argv) → canned stdout
	errors  map[string]error  // key substring (matched against the joined argv) → canned error
}

func (r *fakeRunner) run(_ context.Context, argv ...string) (string, error) {
	r.calls = append(r.calls, append([]string(nil), argv...))
	joined := strings.Join(argv, " ")
	for key, out := range r.outputs {
		if strings.Contains(joined, key) {
			return out, nil
		}
	}
	for key, err := range r.errors {
		if strings.Contains(joined, key) {
			return "", err
		}
	}
	return "", nil
}

func (r *fakeRunner) argvStrings() []string {
	out := make([]string, len(r.calls))
	for i, c := range r.calls {
		out[i] = strings.Join(c, " ")
	}
	return out
}

// installDef names a real executable because Install runs the rendered
// unit through the native validator (systemd-analyze verify), which rejects
// an ExecStart that does not exist on the test host.
func installDef() Definition {
	return Definition{
		Name:        "vrooli-bridge-agent",
		Description: "Vrooli Bridge node agent",
		ExecPath:    "/bin/sh",
		Args:        []string{"--control-plane-url", "https://cp.example", "--node-id", "n1"},
		User:        "vrooli-agent",
	}
}

// [REQ:BRG-P0-007] systemd Install writes the rendered user unit to
// ~/.config/systemd/user and drives daemon-reload → enable → restart, so the
// service auto-starts and the freshly-written unit is what runs.
func TestSystemdInstall_WritesUnitAndConvergesState(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{}
	m := systemdManager{unitDir: func() (string, error) { return dir, nil }, runner: runner}

	res, err := m.Install(context.Background(), installDef())
	require.NoError(t, err)

	unitPath := filepath.Join(dir, "vrooli-bridge-agent.service")
	require.Equal(t, unitPath, res.UnitPath)
	require.Equal(t, "vrooli-bridge-agent.service", res.UnitName)
	require.True(t, res.Enabled)
	require.True(t, res.Running)
	require.Equal(t, platform.ServiceManagerSystemd, res.Kind)

	// The unit file on disk carries the rendered content.
	content, err := os.ReadFile(unitPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "ExecStart=\"/bin/sh\" --control-plane-url https://cp.example --node-id n1")
	require.Contains(t, string(content), "Restart=on-failure")

	// Exact argv sequence: reload picks up the write, enable sets auto-start,
	// restart re-execs on the fresh unit.
	require.Equal(t, []string{
		"systemctl --user daemon-reload",
		"systemctl --user enable vrooli-bridge-agent.service",
		"systemctl --user restart vrooli-bridge-agent.service",
	}, runner.argvStrings())
}

func TestSystemdSystemScopeUsesMachineUnitAndManager(t *testing.T) {
	d := installDef()
	d.Name = "vrooli-bridge-provisioner"
	d.User = "vrooli-provisioner"
	d.System = true
	m := systemdManager{unitDir: func() (string, error) { return t.TempDir(), nil }, runner: &fakeRunner{}}
	require.Equal(t, "/etc/systemd/system/vrooli-bridge-provisioner.service", func() string {
		path, err := m.unitPath(d)
		require.NoError(t, err)
		return path
	}())
	require.Equal(t, []string{"systemctl", "enable", "vrooli-bridge-provisioner.service"}, systemdArgs(d, "enable", "vrooli-bridge-provisioner.service"))
	unit, err := SystemdUnit(d)
	require.NoError(t, err)
	require.Contains(t, unit, "User=vrooli-provisioner")
}

func TestSystemScopeRequiresExplicitServiceUser(t *testing.T) {
	d := installDef()
	d.System = true
	d.User = ""

	_, err := SystemdUnit(d)
	require.EqualError(t, err, "system service user is required")

	_, err = WindowsServiceCreateArgs(d)
	require.EqualError(t, err, "system service user is required")
}

// [REQ:BRG-P0-007] Re-running systemd Install is idempotent: it rewrites the same
// unit and restarts again rather than erroring, converging on identical state.
func TestSystemdInstall_Idempotent(t *testing.T) {
	dir := t.TempDir()
	m := systemdManager{unitDir: func() (string, error) { return dir, nil }, runner: &fakeRunner{}}

	first, err := m.Install(context.Background(), installDef())
	require.NoError(t, err)
	unitPath := filepath.Join(dir, "vrooli-bridge-agent.service")
	firstContent, err := os.ReadFile(unitPath)
	require.NoError(t, err)

	// Second install with a fresh runner (a real re-run has no memory of the
	// first) must succeed and re-drive the same converge sequence.
	runner2 := &fakeRunner{}
	m2 := systemdManager{unitDir: func() (string, error) { return dir, nil }, runner: runner2}
	second, err := m2.Install(context.Background(), installDef())
	require.NoError(t, err)

	require.Equal(t, first, second)
	secondContent, err := os.ReadFile(unitPath)
	require.NoError(t, err)
	require.Equal(t, string(firstContent), string(secondContent), "re-install rewrites identical content")
	require.Contains(t, runner2.argvStrings(), "systemctl --user restart vrooli-bridge-agent.service")
}

// [REQ:BRG-P0-007] systemd Status parses `systemctl show` into installed /
// enabled / running / pid without mutating anything.
func TestSystemdStatus_ParsesShowOutput(t *testing.T) {
	dir := t.TempDir()
	unitPath := filepath.Join(dir, "vrooli-bridge-agent.service")
	require.NoError(t, os.WriteFile(unitPath, []byte("[Unit]\n"), 0o644))

	runner := &fakeRunner{outputs: map[string]string{
		"--user show": "ActiveState=active\nUnitFileState=enabled\nMainPID=4242\nLoadState=loaded\n",
	}}
	m := systemdManager{unitDir: func() (string, error) { return dir, nil }, runner: runner}

	res, err := m.Status(context.Background(), installDef())
	require.NoError(t, err)
	require.True(t, res.Installed)
	require.True(t, res.Enabled)
	require.True(t, res.Running)
	require.Equal(t, 4242, res.PID)
	// Status is read-only: the only command run is the query.
	require.Equal(t, []string{"systemctl --user show vrooli-bridge-agent.service --property=ActiveState,UnitFileState,MainPID,LoadState"}, runner.argvStrings())
}

// [REQ:BRG-P0-007] systemd Status of a never-installed unit reports not-installed
// / not-running (no unit file, `show` reports inactive).
func TestSystemdStatus_NotInstalled(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{outputs: map[string]string{
		"--user show": "ActiveState=inactive\nUnitFileState=\nMainPID=0\nLoadState=not-found\n",
	}}
	m := systemdManager{unitDir: func() (string, error) { return dir, nil }, runner: runner}

	res, err := m.Status(context.Background(), installDef())
	require.NoError(t, err)
	require.False(t, res.Installed)
	require.False(t, res.Running)
	require.False(t, res.Enabled)
	require.Zero(t, res.PID)
}

// [REQ:BRG-P0-007] systemd Uninstall stops+disables, removes the unit file, and
// reloads — fully reversing Install.
func TestSystemdUninstall_RemovesUnit(t *testing.T) {
	dir := t.TempDir()
	unitPath := filepath.Join(dir, "vrooli-bridge-agent.service")
	require.NoError(t, os.WriteFile(unitPath, []byte("[Unit]\n"), 0o644))

	runner := &fakeRunner{}
	m := systemdManager{unitDir: func() (string, error) { return dir, nil }, runner: runner}

	res, err := m.Uninstall(context.Background(), installDef())
	require.NoError(t, err)
	require.True(t, res.Removed)
	require.NoFileExists(t, unitPath)
	require.Equal(t, []string{
		"systemctl --user disable --now vrooli-bridge-agent.service",
		"systemctl --user daemon-reload",
	}, runner.argvStrings())
}

// [REQ:BRG-P0-007] Uninstall of a never-installed service is a clean no-op
// (Removed=false, no error) — idempotent teardown.
func TestSystemdUninstall_Idempotent(t *testing.T) {
	dir := t.TempDir()
	m := systemdManager{unitDir: func() (string, error) { return dir, nil }, runner: &fakeRunner{}}

	res, err := m.Uninstall(context.Background(), installDef())
	require.NoError(t, err)
	require.False(t, res.Removed)
}

// [REQ:BRG-P0-007] launchd Install writes the plist to the LaunchAgents dir and
// drives bootout → bootstrap → enable → kickstart against gui/<uid>/<label>. The
// darwin path is covered by argv-level assertions (no real mac run this phase).
func TestLaunchdInstall_WritesPlistAndBootstraps(t *testing.T) {
	dir := t.TempDir()
	runner := &fakeRunner{}
	m := launchdManager{
		agentDir: func() (string, error) { return dir, nil },
		uid:      func() int { return 501 },
		runner:   runner,
	}

	res, err := m.Install(context.Background(), installDef())
	require.NoError(t, err)

	plistPath := filepath.Join(dir, "com.vrooli.bridge.vrooli-bridge-agent.plist")
	require.Equal(t, plistPath, res.UnitPath)
	require.Equal(t, "com.vrooli.bridge.vrooli-bridge-agent", res.UnitName)
	require.True(t, res.Running)

	content, err := os.ReadFile(plistPath)
	require.NoError(t, err)
	require.Contains(t, string(content), "<string>com.vrooli.bridge.vrooli-bridge-agent</string>")
	require.Contains(t, string(content), "<key>KeepAlive</key>")

	require.Equal(t, []string{
		"launchctl bootout gui/501/com.vrooli.bridge.vrooli-bridge-agent",
		"launchctl bootstrap gui/501 " + plistPath,
		"launchctl enable gui/501/com.vrooli.bridge.vrooli-bridge-agent",
		"launchctl kickstart -k gui/501/com.vrooli.bridge.vrooli-bridge-agent",
	}, runner.argvStrings())
}

// [REQ:BRG-P0-007] launchd Install is idempotent: bootout-before-bootstrap means
// a re-install converges on the freshly-written plist rather than failing on an
// already-loaded label.
func TestLaunchdInstall_Idempotent(t *testing.T) {
	dir := t.TempDir()
	newMgr := func(r commandRunner) launchdManager {
		return launchdManager{
			agentDir: func() (string, error) { return dir, nil },
			uid:      func() int { return 501 },
			runner:   r,
		}
	}
	first, err := newMgr(&fakeRunner{}).Install(context.Background(), installDef())
	require.NoError(t, err)
	runner2 := &fakeRunner{}
	second, err := newMgr(runner2).Install(context.Background(), installDef())
	require.NoError(t, err)
	require.Equal(t, first, second)
	require.Equal(t, "launchctl bootout gui/501/com.vrooli.bridge.vrooli-bridge-agent", runner2.argvStrings()[0])
}

// A headless SSH session has no gui/<uid> bootstrap. The manager must use a
// root-owned LaunchDaemon, retain the unprivileged service user, and keep every
// privileged operation behind non-interactive sudo.
func TestLaunchdInstall_HeadlessUsesSystemDaemon(t *testing.T) {
	runner := &fakeRunner{}
	m := launchdManager{
		agentDir: func() (string, error) { return t.TempDir(), nil },
		uid:      func() int { return 501 },
		domain:   func() string { return "user/501" },
		runner:   runner,
	}

	res, err := m.Install(context.Background(), installDef())
	require.NoError(t, err)
	require.Equal(t, "/Library/LaunchDaemons/com.vrooli.bridge.vrooli-bridge-agent.plist", res.UnitPath)
	require.True(t, res.Enabled)
	require.True(t, res.Running)
	calls := runner.argvStrings()
	require.Len(t, calls, 7)
	require.True(t, strings.HasPrefix(calls[0], "sudo -n /usr/bin/install -o root -g wheel -m 0644 "))
	require.Contains(t, calls[0], " /Library/LaunchDaemons/com.vrooli.bridge.vrooli-bridge-agent.plist")
	require.Equal(t, "sudo -n launchctl bootout system/com.vrooli.bridge.vrooli-bridge-agent", calls[1])
	require.Equal(t, "sudo -n launchctl bootout gui/501/com.vrooli.bridge.vrooli-bridge-agent", calls[2])
	require.Equal(t, "sudo -n launchctl bootout user/501/com.vrooli.bridge.vrooli-bridge-agent", calls[3])
	require.Equal(t, "sudo -n launchctl bootstrap system /Library/LaunchDaemons/com.vrooli.bridge.vrooli-bridge-agent.plist", calls[4])
	require.Equal(t, "sudo -n launchctl enable system/com.vrooli.bridge.vrooli-bridge-agent", calls[5])
	require.Equal(t, "sudo -n launchctl kickstart -k system/com.vrooli.bridge.vrooli-bridge-agent", calls[6])
}

func TestLaunchdUserDomainsAreExactAndStable(t *testing.T) {
	require.Equal(t, []string{"gui/501", "user/501"}, launchdUserDomains(501))
	require.Empty(t, launchdUserDomains(-1))
}

// [REQ:BRG-P0-007] launchd Status parses `launchctl print` state = running / pid.
func TestLaunchdStatus_ParsesPrintOutput(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "com.vrooli.bridge.vrooli-bridge-agent.plist")
	require.NoError(t, os.WriteFile(plistPath, []byte("<plist/>"), 0o644))

	runner := &fakeRunner{outputs: map[string]string{
		"launchctl print": "com.vrooli.bridge.vrooli-bridge-agent = {\n\tstate = running\n\tpid = 7788\n}",
	}}
	m := launchdManager{
		agentDir: func() (string, error) { return dir, nil },
		uid:      func() int { return 501 },
		runner:   runner,
	}
	res, err := m.Status(context.Background(), installDef())
	require.NoError(t, err)
	require.True(t, res.Installed)
	require.True(t, res.Running)
	require.Equal(t, 7788, res.PID)
	require.Equal(t, []string{"launchctl print gui/501/com.vrooli.bridge.vrooli-bridge-agent"}, runner.argvStrings())
}

// [REQ:BRG-P0-007] launchd Uninstall boots the label out and removes the plist.
func TestLaunchdUninstall_RemovesPlist(t *testing.T) {
	dir := t.TempDir()
	plistPath := filepath.Join(dir, "com.vrooli.bridge.vrooli-bridge-agent.plist")
	require.NoError(t, os.WriteFile(plistPath, []byte("<plist/>"), 0o644))

	runner := &fakeRunner{}
	m := launchdManager{
		agentDir: func() (string, error) { return dir, nil },
		uid:      func() int { return 501 },
		runner:   runner,
	}
	res, err := m.Uninstall(context.Background(), installDef())
	require.NoError(t, err)
	require.True(t, res.Removed)
	require.NoFileExists(t, plistPath)
	require.Equal(t, []string{"launchctl bootout gui/501/com.vrooli.bridge.vrooli-bridge-agent"}, runner.argvStrings())
}

// [REQ:BRG-P0-007] The launchd plist XML-escapes argv values, so a control-plane
// URL carrying an ampersand produces a well-formed plist (the bug the render-only
// path had before this phase).
func TestLaunchdPlist_EscapesXML(t *testing.T) {
	d := installDef()
	d.Args = []string{"--control-plane-url", "https://cp.example/?a=1&b=2<x>"}
	plist, err := LaunchdPlist(d)
	require.NoError(t, err)
	require.Contains(t, plist, "https://cp.example/?a=1&amp;b=2&lt;x&gt;")
	require.NotContains(t, plist, "&b=2<x>")
}

// [REQ:BRG-P0-007] Windows drives the SCM through a typed, idempotent command
// sequence. The fake responses model the native `sc.exe` output; they do not
// substitute for a real Windows host run.
func TestWindowsManager_InstallStatusUninstall(t *testing.T) {
	d := installDef()
	runner := &scriptedRunner{responses: []scriptedResponse{
		{err: errFakeErr("1060: service does not exist")},
		{},
		{},
		{out: "STATE              : 4  RUNNING\n        PID                : 4242\n"},
		{out: "START_TYPE         : 2   AUTO_START\nBINARY_PATH_NAME   : /bin/sh --control-plane-url https://cp.example --node-id n1\nSERVICE_START_NAME : vrooli-agent\n"},
		{out: "STATE              : 4  RUNNING\n        PID                : 4242\n"},
		{out: "START_TYPE         : 2   AUTO_START\nBINARY_PATH_NAME   : /bin/sh --control-plane-url https://cp.example --node-id n1\nSERVICE_START_NAME : vrooli-agent\n"},
		{out: "STATE              : 1  STOPPED\n"},
		{err: errFakeErr("1062: service is not active")},
		{},
	}}
	m := windowsManager{runner: runner}

	installed, err := m.Install(context.Background(), d)
	require.NoError(t, err)
	require.True(t, installed.Enabled)
	require.True(t, installed.Running)
	require.Equal(t, platform.ServiceManagerWindows, installed.Kind)
	require.Equal(t, []string{
		"sc.exe queryex vrooli-bridge-agent",
		"sc.exe create vrooli-bridge-agent binPath= /bin/sh --control-plane-url https://cp.example --node-id n1 start= auto DisplayName= Vrooli Bridge node agent obj= vrooli-agent",
		"sc.exe start vrooli-bridge-agent",
		"sc.exe queryex vrooli-bridge-agent",
		"sc.exe qc vrooli-bridge-agent",
	}, runner.argvStrings()[:5])

	status, err := m.Status(context.Background(), d)
	require.NoError(t, err)
	require.True(t, status.Installed)
	require.True(t, status.Configured)
	require.True(t, status.Running)
	require.True(t, status.Enabled)
	require.Equal(t, 4242, status.PID)

	removed, err := m.Uninstall(context.Background(), d)
	require.NoError(t, err)
	require.True(t, removed.Removed)
	require.Equal(t, []string{
		"sc.exe queryex vrooli-bridge-agent",
		"sc.exe stop vrooli-bridge-agent",
		"sc.exe delete vrooli-bridge-agent",
	}, runner.argvStrings()[7:])
}

func TestWindowsManagerStatus_RejectsTamperedServiceConfiguration(t *testing.T) {
	d := installDef()
	runner := &scriptedRunner{responses: []scriptedResponse{
		{out: "STATE              : 4  RUNNING\n        PID                : 4242\n"},
		{out: "START_TYPE         : 2   AUTO_START\nBINARY_PATH_NAME   : C:\\Users\\Public\\rogue.exe\nSERVICE_START_NAME : Administrator\n"},
	}}
	m := windowsManager{runner: runner}

	status, err := m.Status(context.Background(), d)
	require.NoError(t, err)
	require.True(t, status.Installed)
	require.False(t, status.Configured)
	require.True(t, status.Running)
	require.Contains(t, status.Detail, "CONFIGURATION_MISMATCH")
}

func TestWindowsManager_MissingServiceIsReadOnlyNoop(t *testing.T) {
	runner := &scriptedRunner{responses: []scriptedResponse{{err: errFakeErr("1060: does not exist")}, {err: errFakeErr("1060: does not exist")}}}
	m := windowsManager{runner: runner}
	res, err := m.Status(context.Background(), installDef())
	require.NoError(t, err)
	require.False(t, res.Installed)
	require.False(t, res.Running)
	removed, err := m.Uninstall(context.Background(), installDef())
	require.NoError(t, err)
	require.False(t, removed.Removed)
}

// [REQ:BRG-P0-007] A systemd Install whose enable step fails surfaces the error
// (fails safe) rather than reporting a bogus success.
func TestSystemdInstall_PropagatesToolFailure(t *testing.T) {
	dir := t.TempDir()
	failing := &failOnRunner{targetContains: "enable"}
	m := systemdManager{unitDir: func() (string, error) { return dir, nil }, runner: failing}
	_, err := m.Install(context.Background(), installDef())
	require.Error(t, err)
}

var errFake = errFakeErr("boom")

type errFakeErr string

func (e errFakeErr) Error() string { return string(e) }

type scriptedResponse struct {
	out string
	err error
}

type scriptedRunner struct {
	calls     [][]string
	responses []scriptedResponse
}

func (r *scriptedRunner) run(_ context.Context, argv ...string) (string, error) {
	r.calls = append(r.calls, append([]string(nil), argv...))
	if len(r.responses) == 0 {
		return "", nil
	}
	response := r.responses[0]
	r.responses = r.responses[1:]
	return response.out, response.err
}

func (r *scriptedRunner) argvStrings() []string {
	out := make([]string, len(r.calls))
	for i, call := range r.calls {
		out[i] = strings.Join(call, " ")
	}
	return out
}

// failOnRunner fails the first call whose joined argv contains a substring, so a
// test can force a specific step to fail while others succeed.
type failOnRunner struct {
	targetContains string
	calls          [][]string
}

func (r *failOnRunner) run(_ context.Context, argv ...string) (string, error) {
	r.calls = append(r.calls, argv)
	if strings.Contains(strings.Join(argv, " "), r.targetContains) {
		return "", errFake
	}
	return "", nil
}
