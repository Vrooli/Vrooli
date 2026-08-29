// Helpers for handlers that perform per-user installs (go install, npm

// install, file writes under $HOME) when the vrooli process itself may be
// running as root via sudo.
//
// # The problem
//
// When an operator runs `sudo vrooli setup`, the vrooli process executes
// as root with $HOME=/root (sudo's default). Any subprocess it spawns
// inherits that environment, so:
//
//   - `go install some/pkg` writes to /root/go/bin instead of
//     /home/$SUDO_USER/go/bin.
//   - `npm install --prefix ~/.cache/...` resolves the prefix against
//     /root/.cache, not the operator's cache.
//   - File writes via os.UserHomeDir / os.MkdirAll under $HOME end up
//     under /root.
//
// All of these break post-install discovery (the binary is somewhere the
// operator's PATH never visits) and leave behind root-owned files in
// directories the operator's normal-user processes need to write to.
//
// # The fix
//
// Distinguish between "this command needs root" (apt-get install, file
// writes to /usr/local/bin) and "this command should run as the invoking
// user" (go install, npm install, anything writing to $HOME). For the
// latter, drop privileges back to $SUDO_USER for the duration of the
// command.
//
// Handlers call RunAsInvokingUser instead of RunCommandFn for per-user
// installs. When the vrooli process is not root, this is a no-op
// (commands run as the current user). When the vrooli process is root
// with $SUDO_USER set, this wraps the command with `sudo -u $SUDO_USER -H
// -- ...` so it runs as the operator with the operator's $HOME.
//
// We use sudo (rather than syscall-level setuid) because:
//   - sudo handles secondary group membership and PAM session setup
//     correctly.
//   - -H sets HOME to the target user's home, which go and npm read.
//   - We're already root, so sudo doesn't prompt for a password.
//   - It's transparent to RunCommandFn (test seam) — the wrapper just
//     prepends `sudo` arguments.

package hostreqkit

import (
	"errors"
	"os"
	osuser "os/user"
	"strconv"
	"strings"
)

const (
	sudoUserRoot = "root"
)

// InvokingUser returns the username of the operator whose intent we should
// honor for per-user operations:
//
//   - When running as root (Geteuid()==0) AND $SUDO_USER is set, returns
//     $SUDO_USER. This is the typical `sudo vrooli setup` case — the
//     operator who escalated.
//   - Otherwise returns $USER (the current process's user).
//
// Returns empty string only in unusual contexts (daemonized processes
// with no $USER), which callers should treat as "no user-scoped operation
// possible".
func InvokingUser() string {
	if RunningAsRootFn() {
		if u := strings.TrimSpace(os.Getenv("SUDO_USER")); u != "" {
			return u
		}
	}
	return strings.TrimSpace(os.Getenv("USER"))
}

// InvokingUserIDs returns the numeric uid/gid of the operator who escalated via
// sudo, read from the authoritative $SUDO_UID/$SUDO_GID that sudo itself sets
// (no /etc/passwd parse needed). `ok` is true only when the process is
// root-via-sudo and both ids parse to a non-root uid — exactly the case where
// an in-process write into the operator's home would otherwise be left owned by
// root. When `ok` is false the caller must NOT chown: either no escalation
// happened (writes already land as the right user) or the target identity is
// unknown.
func InvokingUserIDs() (uid int, gid int, ok bool) {
	if !RunningAsRootFn() {
		return 0, 0, false
	}
	if strings.TrimSpace(os.Getenv("SUDO_USER")) == "" {
		return 0, 0, false
	}
	uidStr := strings.TrimSpace(os.Getenv("SUDO_UID"))
	gidStr := strings.TrimSpace(os.Getenv("SUDO_GID"))
	if uidStr == "" || gidStr == "" {
		return 0, 0, false
	}
	u, err := strconv.Atoi(uidStr)
	if err != nil || u == 0 {
		return 0, 0, false
	}
	g, err := strconv.Atoi(gidStr)
	if err != nil {
		return 0, 0, false
	}
	return u, g, true
}

// InvokingUserHomeDir returns the home directory of InvokingUser. Reads
// /etc/passwd directly because sudo sets HOME=/root by default — calling
// os.UserHomeDir from a sudo'd process returns /root, not the operator's
// home. /etc/passwd is consulted by both Linux and macOS for local
// accounts; AD-bound or NIS-only setups fall back to $HOME (which is
// correct for the invoking process even if not for cross-user lookups).
func InvokingUserHomeDir() (string, error) {
	user := InvokingUser()
	if user == "" {
		return os.UserHomeDir()
	}
	if home := lookupHomeFromPasswdFn(user); home != "" {
		return home, nil
	}
	if h := strings.TrimSpace(os.Getenv("HOME")); h != "" {
		return h, nil
	}
	return "", errors.New("hostreqkit: could not resolve invoking user's home directory")
}

// InvokingUserCommand returns a command invocation that targets the operator's
// user session. User-scoped systemd commands need both the operator's uid
// runtime directory and that user's D-Bus session bus; inheriting the
// elevated process environment commonly points at root's nonexistent bus.
// When the process is elevated, the command is also run as the invoking user.
func InvokingUserCommand(name string, args ...string) (string, []string) {
	user := InvokingUser()
	uid := 0
	if current, err := osuser.Current(); err == nil {
		uid, _ = strconv.Atoi(current.Uid)
	}
	if RunningAsRootFn() {
		if invokingUID, _, ok := InvokingUserIDs(); ok {
			uid = invokingUID
		} else {
			uid = 0
		}
	}

	commandArgs := append([]string(nil), args...)
	if name == "systemctl" && uid > 0 {
		busArgs := []string{
			"XDG_RUNTIME_DIR=/run/user/" + strconv.Itoa(uid),
			"DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/" + strconv.Itoa(uid) + "/bus",
			name,
		}
		commandArgs = append(busArgs, commandArgs...)
		name = "env"
	}
	if RunningAsRootFn() && user != "" && user != sudoUserRoot {
		return helpersSudo, append([]string{"-u", user, "-H", "--", name}, commandArgs...)
	}
	return name, commandArgs
}

// lookupHomeFromPasswdFn is a test seam over /etc/passwd. Production reads
// the real file via ReadFileFn (which is itself overridable for tests
// that exercise the full chain).
var lookupHomeFromPasswdFn = func(user string) string {
	data, err := ReadFileFn("/etc/passwd")
	if err != nil {
		return ""
	}
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Split(line, ":")
		// /etc/passwd format: name:passwd:uid:gid:gecos:home:shell
		if len(fields) >= 6 && fields[0] == user {
			return fields[5]
		}
	}
	return ""
}

// RunAsInvokingUser runs a command as the invoking user. When the vrooli
// process is running as root (typically because the operator invoked
// `sudo vrooli setup`), this drops privileges back to $SUDO_USER for the
// command's duration. Otherwise it just runs the command as-is.
//
// Use this for per-user installs (go install, npm install, etc.) where
// running as root would wreck file ownership in the user's home and
// inherit the wrong $HOME. Do NOT use this for commands that genuinely
// need root (apt-get install, writes to /usr/local/bin) — those should
// continue to use RunPrivilegedCommand.
//
// When wrapping, we prepend `sudo -u <user> -H --` so:
//   - -u: target user is $SUDO_USER (the operator).
//   - -H: HOME is set to the target user's home (otherwise go/npm
//     write to /root).
//   - --: end of sudo's option parsing, so the wrapped command's flags
//     are not interpreted by sudo.
//
// We're already root in this branch, so sudo does not prompt for a
// password; the elevation is silent and synchronous.
func RunAsInvokingUser(name string, args []string, opts EnsureOptions) error {
	user := InvokingUser()
	if !RunningAsRootFn() || user == "" || user == sudoUserRoot {
		return RunCommandFn(name, args, opts)
	}
	wrapped := append([]string{"-u", user, "-H", "--", name}, args...)
	return RunCommandFn(helpersSudo, wrapped, opts)
}

// RunAsInvokingUserWithInput is the secret-safe form of RunAsInvokingUser.
// The input is connected directly to the child process and is never placed in
// an argument, environment variable, or temporary file.
func RunAsInvokingUserWithInput(name string, args []string, input string, opts EnsureOptions) error {
	user := InvokingUser()
	if !RunningAsRootFn() || user == "" || user == sudoUserRoot {
		return RunCommandInputFn(name, args, input, opts)
	}
	wrapped := append([]string{"-u", user, "-H", "--", name}, args...)
	return RunCommandInputFn(helpersSudo, wrapped, input, opts)
}
