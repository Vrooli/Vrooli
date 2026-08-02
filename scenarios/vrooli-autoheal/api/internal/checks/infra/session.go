// Package infra provides infrastructure health checks
// [REQ:INFRA-DISPLAY-001] [REQ:INFRA-RDP-001] [REQ:TEST-SEAM-001]
package infra

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/vrooli/vrooli/scenarios/vrooli-autoheal/api/internal/checks"
)

// sessionBusEnv returns the environment assignments needed to reach the
// graphical session user's D-Bus session bus, in `KEY=value` form suitable for
// prefixing a command with env(1).
//
// This is required rather than optional. The autoheal API process inherits an
// ambient XDG_RUNTIME_DIR that points at whichever runtime directory launched
// it, which on observed hosts is /run/user/0 while the graphical session lives
// under /run/user/1000. Tools that read session-scoped state through libsecret
// silently report partial results when pointed at the wrong bus, so callers
// must set these explicitly instead of trusting the ambient environment.
//
// Returns nil when no active graphical session user can be resolved.
func sessionBusEnv(ctx context.Context, exec checks.CommandExecutor) []string {
	user := seat0SessionUser(ctx, exec)
	if user == "" {
		return nil
	}

	output, err := exec.Output(ctx, "id", "-u", user)
	if err != nil {
		return nil
	}
	uid := strings.TrimSpace(string(output))
	if uid == "" {
		return nil
	}

	runtimeDir := fmt.Sprintf("/run/user/%s", uid)
	return []string{
		fmt.Sprintf("XDG_RUNTIME_DIR=%s", runtimeDir),
		fmt.Sprintf("DBUS_SESSION_BUS_ADDRESS=unix:path=%s/bus", runtimeDir),
	}
}

// keyringProbeTimeout bounds the secret-service probe.
const keyringProbeTimeout = 10 * time.Second

// loginKeyringCollectionPresent reports whether the GNOME login keyring is
// unlocked and registered on the session bus.
//
// This is the root-cause predicate for a whole class of failures: anything
// stored in the login keyring is unreadable while this is false. GNOME Remote
// Desktop keeps its RDP credentials there, which is why an autologin host can
// run a healthy daemon that denies every client.
//
// The probe reads the Secret Service `Collections` property and looks for the
// login collection. Do not reach for `gdbus introspect` on the collection path
// instead: it returns an empty node rather than an error when the object does
// not exist, so it cannot distinguish absent from present-but-empty.
//
// Returns false when the probe cannot run at all, because an unverifiable
// keyring must not be reported as healthy.
func loginKeyringCollectionPresent(ctx context.Context, exec checks.CommandExecutor) bool {
	ctx, cancel := context.WithTimeout(ctx, keyringProbeTimeout)
	defer cancel()

	env := sessionBusEnv(ctx, exec)
	if len(env) == 0 {
		return false
	}

	args := append(append([]string{}, env...),
		"gdbus", "call", "--session",
		"--dest", "org.freedesktop.secrets",
		"--object-path", "/org/freedesktop/secrets",
		"--method", "org.freedesktop.DBus.Properties.Get",
		"org.freedesktop.Secret.Service", "Collections")

	output, err := exec.CombinedOutput(ctx, "env", args...)
	if err != nil {
		return false
	}

	return strings.Contains(string(output), "/org/freedesktop/secrets/collection/login")
}

// autoLoginUser returns the user GDM is configured to log in automatically, or
// an empty string when autologin is not configured.
//
// Autologin matters beyond convenience: the gdm-autologin PAM stack
// authenticates through pam_permit with no password, so pam_gnome_keyring has
// nothing to unlock the login keyring with. See loginKeyringCollectionPresent.
func autoLoginUser() string {
	// Try both gdm3 (Debian/Ubuntu) and gdm (RHEL/Fedora) config paths
	configPaths := []string{
		"/etc/gdm3/custom.conf",
		"/etc/gdm/custom.conf",
	}

	for _, configPath := range configPaths {
		if user := autoLoginUserFromFile(configPath); user != "" {
			return user
		}
	}

	return ""
}

// autoLoginUserFromFile parses one GDM custom.conf.
func autoLoginUserFromFile(configPath string) string {
	file, err := os.Open(configPath)
	if err != nil {
		return ""
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	inDaemonSection := false
	autoLoginEnabled := false
	user := ""

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Track which section we're in
		if strings.HasPrefix(line, "[") {
			inDaemonSection = strings.HasPrefix(line, "[daemon]")
			continue
		}

		if !inDaemonSection {
			continue
		}

		if strings.HasPrefix(line, "AutomaticLoginEnable") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				value := strings.TrimSpace(strings.ToLower(parts[1]))
				autoLoginEnabled = value == "true" || value == "1" || value == "yes"
			}
		}
		if strings.HasPrefix(line, "AutomaticLogin=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				user = strings.TrimSpace(parts[1])
			}
		}
	}

	if autoLoginEnabled && user != "" {
		return user
	}
	return ""
}

// seat0SessionUser resolves the user that owns the active graphical session on
// seat0 (the main display). It returns an empty string when no active session
// exists or the session cannot be resolved.
//
// This is shared by the display and RDP checks so both agree on who owns the
// graphical session.
func seat0SessionUser(ctx context.Context, exec checks.CommandExecutor) string {
	output, err := exec.Output(ctx, "loginctl", "show-seat", "seat0", "-p", "ActiveSession", "--value")
	if err != nil {
		return ""
	}

	sessionID := strings.TrimSpace(string(output))
	if sessionID == "" {
		return ""
	}

	output, err = exec.Output(ctx, "loginctl", "show-session", sessionID, "-p", "Name", "--value")
	if err != nil {
		return ""
	}

	return strings.TrimSpace(string(output))
}

// gnomeShellRunningFor reports whether a gnome-shell process exists. When user
// is non-empty the probe is scoped to that user; otherwise any gnome-shell
// process counts.
func gnomeShellRunningFor(ctx context.Context, exec checks.CommandExecutor, user string) bool {
	var (
		output []byte
		err    error
	)

	if user != "" {
		output, err = exec.Output(ctx, "pgrep", "-u", user, "gnome-shell")
	} else {
		output, err = exec.Output(ctx, "pgrep", "gnome-shell")
	}

	if err != nil {
		return false
	}
	return len(strings.TrimSpace(string(output))) > 0
}

// graphicalSessionAvailable reports whether a usable graphical session exists,
// together with the user it belongs to.
//
// When preferredUser is non-empty (for example a configured auto-login user)
// the probe is scoped to that user. Otherwise the owner of the active seat0
// session is resolved first, and a bare gnome-shell probe is the fallback for
// hosts where seat resolution is unavailable.
//
// Callers must not treat a false result as a statement about RDP. This reports
// the graphical-session dependency layer only.
func graphicalSessionAvailable(ctx context.Context, exec checks.CommandExecutor, preferredUser string) (string, bool) {
	if preferredUser != "" {
		return preferredUser, gnomeShellRunningFor(ctx, exec, preferredUser)
	}

	if user := seat0SessionUser(ctx, exec); user != "" {
		if gnomeShellRunningFor(ctx, exec, user) {
			return user, true
		}
		return user, false
	}

	return "", gnomeShellRunningFor(ctx, exec, "")
}
