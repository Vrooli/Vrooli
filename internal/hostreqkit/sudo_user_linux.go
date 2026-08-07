//go:build linux

package hostreqkit

import (
	"os"
	"strconv"
	"strings"
)

// RunAsInvokingUserWithSession restores the invoking user's Linux session bus
// when a root-via-sudo control-plane process invokes a per-user command.
func RunAsInvokingUserWithSession(name string, args []string, opts EnsureOptions) error {
	runtimeDir := invokingSessionRuntimeDir()
	if runtimeDir == "" {
		return RunAsInvokingUser(name, args, opts)
	}
	envArgs := []string{
		"XDG_RUNTIME_DIR=" + runtimeDir,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtimeDir + "/bus",
		name,
	}
	envArgs = append(envArgs, args...)
	return RunAsInvokingUser("env", envArgs, opts)
}

func invokingSessionRuntimeDir() string {
	runtimeDir := strings.TrimSpace(os.Getenv("XDG_RUNTIME_DIR"))
	if uid, _, ok := InvokingUserIDs(); ok {
		runtimeDir = "/run/user/" + strconv.Itoa(uid)
	} else if !RunningAsRootFn() {
		if uid := currentUserIDFn(); uid != "" {
			runtimeDir = "/run/user/" + uid
		}
	}
	return runtimeDir
}

func RunAsInvokingUserWithSessionOutput(name string, args []string, _ EnsureOptions) ([]byte, error) {
	runtimeDir := invokingSessionRuntimeDir()
	if runtimeDir == "" {
		return runAsInvokingUserOutput(name, args)
	}
	envArgs := []string{
		"XDG_RUNTIME_DIR=" + runtimeDir,
		"DBUS_SESSION_BUS_ADDRESS=unix:path=" + runtimeDir + "/bus",
		name,
	}
	envArgs = append(envArgs, args...)
	return runAsInvokingUserOutput("env", envArgs)
}
