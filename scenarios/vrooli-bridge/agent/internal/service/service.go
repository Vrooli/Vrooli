// Package service installs the node-agent as the platform-native background
// service (OT-P0-007): systemd on Linux, launchd on macOS, the Windows Service
// Control Manager on Windows. A single codebase renders all three unit shapes
// from one typed Definition — there are no Linux-only assumptions and no
// hardcoded POSIX paths; every path comes from the Definition the caller
// resolves via package platform. The unit renderers are pure functions so the
// exact generated content is unit-testable without touching the host.
//
// The Definition's User field carries the OS principal the service runs as,
// which is how the two trust tiers (DECISIONS.md) become concrete at install
// time: the agent + non-privileged runner install under the dedicated
// unprivileged service user, while the privileged provisioning helper installs
// as its own separate principal. Same renderer, different Definition.
package service

import (
	"context"
	"encoding/xml"
	"fmt"
	"strings"

	"vrooli-bridge/agent/internal/platform"
)

// Definition is the platform-neutral description of a background service to
// install. The renderers below project it onto each OS's native unit format.
type Definition struct {
	// Name is the service identifier (systemd unit name, launchd label suffix,
	// Windows service name). Required; must be a safe identifier.
	Name string

	// Description is the human-readable service description.
	Description string

	// ExecPath is the absolute path to the agent binary to run. Required.
	ExecPath string

	// Args are the agent's command-line arguments (e.g. --control-plane-url, the
	// node id). Rendered as the service's argv after ExecPath.
	Args []string

	// WorkingDir is the directory the service runs in (the node's Vrooli
	// checkout for the provisioning helper; empty uses the platform default).
	WorkingDir string

	// User is the OS principal the service runs as — the dedicated unprivileged
	// service user for the agent/runner, or the separate privileged principal
	// for the provisioning helper. Empty installs under the installing user.
	User string

	// StandardOutPath and StandardErrorPath are optional native-supervisor log
	// destinations. They are especially important for headless LaunchDaemons,
	// where stdout/stderr otherwise point at /dev/null.
	StandardOutPath   string
	StandardErrorPath string

	// System requests the machine-wide native service namespace. It is used for
	// the privileged helper, whose User field must be honored by the supervisor.
	System bool

	// RestartSeconds is the auto-restart backoff. 0 uses a sane default.
	RestartSeconds int
}

// defaultRestartSeconds is applied when RestartSeconds <= 0.
const defaultRestartSeconds = 5

// validate checks the minimal invariants every renderer relies on.
func (d Definition) validate() error {
	if strings.TrimSpace(d.Name) == "" {
		return fmt.Errorf("service name is required")
	}
	if strings.TrimSpace(d.ExecPath) == "" {
		return fmt.Errorf("service exec path is required")
	}
	return nil
}

func (d Definition) restart() int {
	if d.RestartSeconds <= 0 {
		return defaultRestartSeconds
	}
	return d.RestartSeconds
}

// execLine renders the ExecPath + Args as a single space-joined command line,
// quoting any token that contains whitespace.
func (d Definition) execLine() string {
	parts := make([]string, 0, len(d.Args)+1)
	parts = append(parts, quoteIfNeeded(d.ExecPath))
	for _, a := range d.Args {
		parts = append(parts, quoteIfNeeded(a))
	}
	return strings.Join(parts, " ")
}

func quoteIfNeeded(s string) string {
	if strings.ContainsAny(s, " \t") {
		return `"` + s + `"`
	}
	return s
}

// SystemdUnit renders the [Unit]/[Service]/[Install] file for Linux. It restarts
// on failure, runs as Definition.User when set, and uses only the configured
// ExecPath/WorkingDir (no hardcoded paths).
func SystemdUnit(d Definition) (string, error) {
	if err := d.validate(); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString("[Unit]\n")
	fmt.Fprintf(&b, "Description=%s\n", fallback(d.Description, d.Name))
	b.WriteString("After=network-online.target\n")
	b.WriteString("Wants=network-online.target\n\n")
	b.WriteString("[Service]\n")
	b.WriteString("Type=simple\n")
	fmt.Fprintf(&b, "ExecStart=%s\n", d.execLine())
	if d.User != "" {
		fmt.Fprintf(&b, "User=%s\n", d.User)
	}
	if d.WorkingDir != "" {
		fmt.Fprintf(&b, "WorkingDirectory=%s\n", d.WorkingDir)
	}
	b.WriteString("Restart=on-failure\n")
	fmt.Fprintf(&b, "RestartSec=%d\n\n", d.restart())
	b.WriteString("[Install]\n")
	b.WriteString("WantedBy=multi-user.target\n")
	return b.String(), nil
}

// LaunchdPlist renders the macOS launchd property list. It keeps the service
// alive and runs the configured argv; ProgramArguments carries ExecPath + Args
// as discrete elements (no shell).
func LaunchdPlist(d Definition) (string, error) {
	if err := d.validate(); err != nil {
		return "", err
	}
	var b strings.Builder
	b.WriteString(`<?xml version="1.0" encoding="UTF-8"?>` + "\n")
	b.WriteString(`<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">` + "\n")
	b.WriteString(`<plist version="1.0">` + "\n<dict>\n")
	fmt.Fprintf(&b, "  <key>Label</key>\n  <string>%s</string>\n", plistValue(LaunchdLabel(d.Name)))
	b.WriteString("  <key>ProgramArguments</key>\n  <array>\n")
	for _, arg := range append([]string{d.ExecPath}, d.Args...) {
		fmt.Fprintf(&b, "    <string>%s</string>\n", plistValue(arg))
	}
	b.WriteString("  </array>\n")
	if d.WorkingDir != "" {
		fmt.Fprintf(&b, "  <key>WorkingDirectory</key>\n  <string>%s</string>\n", plistValue(d.WorkingDir))
	}
	if d.User != "" {
		fmt.Fprintf(&b, "  <key>UserName</key>\n  <string>%s</string>\n", plistValue(d.User))
	}
	if d.StandardOutPath != "" {
		fmt.Fprintf(&b, "  <key>StandardOutPath</key>\n  <string>%s</string>\n", plistValue(d.StandardOutPath))
	}
	if d.StandardErrorPath != "" {
		fmt.Fprintf(&b, "  <key>StandardErrorPath</key>\n  <string>%s</string>\n", plistValue(d.StandardErrorPath))
	}
	b.WriteString("  <key>RunAtLoad</key>\n  <true/>\n")
	b.WriteString("  <key>KeepAlive</key>\n  <true/>\n")
	b.WriteString("</dict>\n</plist>\n")
	return b.String(), nil
}

// LaunchdLabel is the reverse-DNS launchd label for a service name.
func LaunchdLabel(name string) string {
	return "com.vrooli.bridge." + name
}

// plistValue XML-escapes a string for safe interpolation into a plist <string>
// element. Argv tokens and paths can contain &, <, > (a control-plane URL query
// string, a WorkingDir with a shell-special segment); without escaping those
// would produce a malformed plist launchd rejects at load. Mirrors the
// runtime-supervisor installer's escaping (internal/runtimesupervisor).
func plistValue(value string) string {
	var b strings.Builder
	_ = xml.EscapeText(&b, []byte(value))
	return b.String()
}

// WindowsServiceCreateArgs renders the `sc.exe create` argv that registers the
// service with the Windows Service Control Manager. It is returned as a typed
// argv (never a shell string); binPath carries the quoted ExecPath + Args, and
// obj= carries Definition.User when set.
func WindowsServiceCreateArgs(d Definition) ([]string, error) {
	if err := d.validate(); err != nil {
		return nil, err
	}
	args := []string{
		"create", d.Name,
		"binPath=", d.execLine(),
		"start=", "auto",
		"DisplayName=", fallback(d.Description, d.Name),
	}
	if d.User != "" {
		args = append(args, "obj=", d.User)
	}
	return args, nil
}

func fallback(primary, secondary string) string {
	if strings.TrimSpace(primary) != "" {
		return primary
	}
	return secondary
}

// Manager installs/uninstalls/queries the node-agent service for the host OS.
// NewManager returns the OS-appropriate implementation; the concrete managers
// (in service_install.go) shell to the native tool (systemctl/launchctl/sc) and
// use the pure renderers above for their unit content.
//
// Install/Status/Uninstall are the real capability the bootstrap installer
// drives (OT-P0-007). They are idempotent by construction: Install rewrites the
// unit and converges the service state (reload → enable → restart) so re-running
// it is a no-op-shaped convergence, never an error; Uninstall tolerates an
// already-absent unit; Status is read-only. Windows stays render-only — its
// Install/Status/Uninstall return a "render-only" error and the operator installs
// the rendered `sc.exe` argv with the platform's own tooling.
type Manager interface {
	// Kind reports which native service mechanism this manager drives.
	Kind() platform.ServiceManagerKind

	// Render returns the native unit content (systemd unit / launchd plist) or,
	// for Windows, the joined `sc.exe` argv — so the install artifact is
	// inspectable before anything touches the host.
	Render(d Definition) (string, error)

	// Install writes the rendered unit to the OS's unit location, then enables +
	// starts the service so it auto-starts and survives process death via the
	// native supervisor. It is idempotent: re-running rewrites the unit and
	// restarts rather than erroring.
	Install(ctx context.Context, d Definition) (InstallResult, error)

	// Status reports whether the service is installed, enabled, and running,
	// without mutating anything.
	Status(ctx context.Context, d Definition) (StatusResult, error)

	// Uninstall stops + disables the service and removes its unit file, fully
	// reversing Install. It tolerates a partially- or never-installed service.
	Uninstall(ctx context.Context, d Definition) (UninstallResult, error)
}

// InstallResult reports where the unit landed and the converged service state
// after Install.
type InstallResult struct {
	Kind     platform.ServiceManagerKind `json:"kind"`
	UnitName string                      `json:"unit_name"` // systemd unit name / launchd label
	UnitPath string                      `json:"unit_path"` // absolute path of the written unit file
	Enabled  bool                        `json:"enabled"`   // set to auto-start
	Running  bool                        `json:"running"`   // started
}

// StatusResult is the read-only view of an installed service.
type StatusResult struct {
	Kind      platform.ServiceManagerKind `json:"kind"`
	UnitName  string                      `json:"unit_name"`
	UnitPath  string                      `json:"unit_path"`
	Installed bool                        `json:"installed"` // unit file present
	Enabled   bool                        `json:"enabled"`   // set to auto-start
	Running   bool                        `json:"running"`   // currently active
	PID       int                         `json:"pid"`       // main process pid, 0 if not running/unknown
	Detail    string                      `json:"detail"`    // native state summary for humans
}

// UninstallResult reports what Uninstall reversed.
type UninstallResult struct {
	Kind     platform.ServiceManagerKind `json:"kind"`
	UnitName string                      `json:"unit_name"`
	UnitPath string                      `json:"unit_path"`
	Removed  bool                        `json:"removed"` // the unit file was present and was removed
}

// NewManager returns the Manager for the OS the agent is running on, mirroring
// platform.NativeServiceManager so install logic never scatters GOOS checks.
func NewManager() Manager {
	return ManagerForKind(platform.NativeServiceManager())
}

// ManagerForKind returns the Manager for an explicit kind (tests / cross-OS
// rendering) without depending on the host's GOOS. Each manager is constructed
// with production seams (real filesystem + native tool exec); tests construct
// the concrete managers directly with fakes.
func ManagerForKind(kind platform.ServiceManagerKind) Manager {
	switch kind {
	case platform.ServiceManagerSystemd:
		return newSystemdManager()
	case platform.ServiceManagerLaunchd:
		return newLaunchdManager()
	case platform.ServiceManagerWindows:
		return windowsManager{}
	default:
		return unsupportedManager{}
	}
}
