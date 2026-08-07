//go:build darwin

package hostreqkit

import (
	"strings"
	"testing"
)

func TestRunAsInvokingUserWithSessionUsesLaunchctlAsuser(t *testing.T) {
	origRoot := RunningAsRootFn
	origRun := RunCommandFn
	origUID := currentUserIDFn
	defer func() {
		RunningAsRootFn = origRoot
		RunCommandFn = origRun
		currentUserIDFn = origUID
	}()
	RunningAsRootFn = func() bool { return false }
	currentUserIDFn = func() string { return "501" }

	var name string
	var args []string
	RunCommandFn = func(gotName string, gotArgs []string, _ EnsureOptions) error {
		name = gotName
		args = append([]string(nil), gotArgs...)
		return nil
	}

	if err := RunAsInvokingUserWithSession("security", []string{"find-generic-password"}, EnsureOptions{}); err != nil {
		t.Fatal(err)
	}
	if name != "launchctl" || strings.Join(args, " ") != "asuser 501 security find-generic-password" {
		t.Fatalf("command = %q %v", name, args)
	}
}
