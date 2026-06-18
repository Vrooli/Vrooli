package service_test

import (
	"strings"
	"testing"

	"vrooli-bridge/agent/internal/platform"
	"vrooli-bridge/agent/internal/service"

	"github.com/stretchr/testify/require"
)

func sampleDef() service.Definition {
	return service.Definition{
		Name:        "vrooli-bridge-agent",
		Description: "Vrooli Bridge node agent",
		ExecPath:    "/opt/vrooli/bin/vrooli-bridge-agent",
		Args:        []string{"--control-plane-url", "https://cp.example", "--node-id", "n1"},
		WorkingDir:  "/var/lib/vrooli-bridge-agent",
		User:        "vrooli-agent",
	}
}

// [REQ:BRG-P0-007] The service-install adapter selects systemd/launchd/Windows
// by service-manager kind — one codebase, no scattered GOOS checks.
func TestManagerForKind_SelectsByKind(t *testing.T) {
	require.Equal(t, platform.ServiceManagerSystemd, service.ManagerForKind(platform.ServiceManagerSystemd).Kind())
	require.Equal(t, platform.ServiceManagerLaunchd, service.ManagerForKind(platform.ServiceManagerLaunchd).Kind())
	require.Equal(t, platform.ServiceManagerWindows, service.ManagerForKind(platform.ServiceManagerWindows).Kind())
	require.Equal(t, platform.ServiceManagerUnknown, service.ManagerForKind(platform.ServiceManagerUnknown).Kind())
}

// [REQ:BRG-P0-007] NewManager picks the manager matching the host GOOS (the
// pure NativeServiceManager mapping), so the installed service is always the
// platform-native one.
func TestNewManager_MatchesHostGOOS(t *testing.T) {
	require.Equal(t, platform.NativeServiceManager(), service.NewManager().Kind())
}

// [REQ:BRG-P0-007] The systemd unit carries the configured argv, runs as the
// dedicated user, restarts on failure, and contains NO hardcoded path other
// than the ones supplied on the Definition (no Linux-only assumptions baked in).
func TestSystemdUnit_RendersConfiguredPaths(t *testing.T) {
	unit, err := service.SystemdUnit(sampleDef())
	require.NoError(t, err)
	require.Contains(t, unit, "ExecStart=/opt/vrooli/bin/vrooli-bridge-agent --control-plane-url https://cp.example --node-id n1")
	require.Contains(t, unit, "User=vrooli-agent")
	require.Contains(t, unit, "WorkingDirectory=/var/lib/vrooli-bridge-agent")
	require.Contains(t, unit, "Restart=on-failure")
	require.Contains(t, unit, "WantedBy=multi-user.target")
	assertNoHardcodedHostPaths(t, unit, sampleDef())
}

// [REQ:BRG-P0-007] The launchd plist carries ProgramArguments as discrete
// elements (no shell), keeps the service alive, and runs as the configured user.
func TestLaunchdPlist_RendersProgramArguments(t *testing.T) {
	plist, err := service.LaunchdPlist(sampleDef())
	require.NoError(t, err)
	require.Contains(t, plist, "<string>com.vrooli.bridge.vrooli-bridge-agent</string>")
	require.Contains(t, plist, "<string>/opt/vrooli/bin/vrooli-bridge-agent</string>")
	require.Contains(t, plist, "<string>--control-plane-url</string>")
	require.Contains(t, plist, "<key>KeepAlive</key>")
	require.Contains(t, plist, "<string>vrooli-agent</string>")
	assertNoHardcodedHostPaths(t, plist, sampleDef())
}

// [REQ:BRG-P0-007] The Windows registration is a typed `sc.exe create` argv
// (never a shell string) carrying the configured binPath and auto start.
func TestWindowsServiceCreateArgs_TypedArgv(t *testing.T) {
	args, err := service.WindowsServiceCreateArgs(sampleDef())
	require.NoError(t, err)
	require.Equal(t, "create", args[0])
	require.Equal(t, "vrooli-bridge-agent", args[1])
	require.Contains(t, args, "binPath=")
	require.Contains(t, args, "start=")
	require.Contains(t, args, "auto")
	require.Contains(t, args, "obj=")
	require.Contains(t, args, "vrooli-agent")
}

// [REQ:BRG-P0-007] The privileged provisioning helper installs under its OWN
// principal — the same renderer, a different User — so the two trust tiers are
// distinct OS principals at install time.
func TestSystemdUnit_PrivilegedHelperSeparatePrincipal(t *testing.T) {
	agent, err := service.SystemdUnit(sampleDef())
	require.NoError(t, err)

	helper := sampleDef()
	helper.Name = "vrooli-bridge-provisioner"
	helper.User = "vrooli-provisioner"
	helperUnit, err := service.SystemdUnit(helper)
	require.NoError(t, err)

	require.Contains(t, agent, "User=vrooli-agent")
	require.Contains(t, helperUnit, "User=vrooli-provisioner")
	require.NotEqual(t, agent, helperUnit, "the privileged helper is a distinct unit with a distinct principal")
}

// [REQ:BRG-P0-007] A missing required field is a clean error, not a malformed
// unit.
func TestRenderers_RejectMissingFields(t *testing.T) {
	_, err := service.SystemdUnit(service.Definition{Name: "x"})
	require.Error(t, err, "missing exec path")
	_, err = service.LaunchdPlist(service.Definition{ExecPath: "/x"})
	require.Error(t, err, "missing name")
	_, err = service.WindowsServiceCreateArgs(service.Definition{})
	require.Error(t, err)
}

// assertNoHardcodedHostPaths fails if a rendered unit contains an absolute POSIX
// path that did not come from the Definition — guarding against Linux-only
// assumptions creeping into the renderers.
func assertNoHardcodedHostPaths(t *testing.T, unit string, d service.Definition) {
	t.Helper()
	allowed := append([]string{d.ExecPath, d.WorkingDir}, d.Args...)
	for _, line := range strings.Split(unit, "\n") {
		for _, tok := range strings.Fields(line) {
			tok = strings.Trim(tok, `"<>`)
			tok = strings.TrimPrefix(tok, "ExecStart=")
			tok = strings.TrimPrefix(tok, "WorkingDirectory=")
			// A real filesystem path has at least two segments (e.g. /opt/x);
			// single-segment tokens like the stripped XML tag </array> → "/array"
			// are not paths.
			if !strings.HasPrefix(tok, "/") || strings.Count(tok, "/") < 2 {
				continue
			}
			require.True(t, containsToken(allowed, tok),
				"rendered unit contains hardcoded path %q not supplied on the Definition", tok)
		}
	}
}

func containsToken(set []string, tok string) bool {
	for _, s := range set {
		if s == tok {
			return true
		}
	}
	return false
}
