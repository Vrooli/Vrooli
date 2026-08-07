//go:build darwin

package hostreqkit

import (
	"strconv"
	"strings"
)

func invokingSessionUID() string {
	if uid, _, ok := InvokingUserIDs(); ok {
		return strconv.Itoa(uid)
	}
	if !RunningAsRootFn() {
		return strings.TrimSpace(currentUserIDFn())
	}
	return ""
}

// RunAsInvokingUserWithSession enters the invoking user's launchd session.
// launchctl asuser must run as the target user, so the common privilege-drop
// wrapper remains outside the launchctl command.
func RunAsInvokingUserWithSession(name string, args []string, opts EnsureOptions) error {
	uid := invokingSessionUID()
	if uid == "" {
		return RunAsInvokingUser(name, args, opts)
	}
	launchArgs := []string{"asuser", uid, name}
	launchArgs = append(launchArgs, args...)
	return RunAsInvokingUser("launchctl", launchArgs, opts)
}

func RunAsInvokingUserWithSessionOutput(name string, args []string, _ EnsureOptions) ([]byte, error) {
	uid := invokingSessionUID()
	if uid == "" {
		return runAsInvokingUserOutput(name, args)
	}
	launchArgs := []string{"asuser", uid, name}
	launchArgs = append(launchArgs, args...)
	return runAsInvokingUserOutput("launchctl", launchArgs)
}
