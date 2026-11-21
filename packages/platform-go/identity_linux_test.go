//go:build linux

package platform

import (
	"reflect"
	"testing"

	"os/user"
)

func TestIdentityCommandForRealRootLoginRunsDirectly(t *testing.T) {
	command, args, err := identityCommandForUser(0, func(string) string { return "" }, nil, "vrooli", []string{"credentials", "doctor"})
	if err != nil {
		t.Fatalf("identityCommandForUser: %v", err)
	}
	if command != "vrooli" || !reflect.DeepEqual(args, []string{"credentials", "doctor"}) {
		t.Fatalf("command = %q %q, want direct root command", command, args)
	}
}

func TestIdentityCommandForSudoInvokedRootPreservesOperatorBoundary(t *testing.T) {
	env := map[string]string{"SUDO_USER": "operator", "SUDO_UID": "1007"}
	command, args, err := identityCommandForUser(0, func(key string) string { return env[key] }, nil, "/opt/vrooli", []string{"scenario", "start", "onboarding"})
	if err != nil {
		t.Fatalf("identityCommandForUser: %v", err)
	}
	want := []string{"-u", "operator", "-H", "--", "env", "XDG_RUNTIME_DIR=/run/user/1007", "DBUS_SESSION_BUS_ADDRESS=unix:path=/run/user/1007/bus", "/opt/vrooli", "scenario", "start", "onboarding"}
	if command != "sudo" || !reflect.DeepEqual(args, want) {
		t.Fatalf("command = %q %q, want sudo with structured invoking-user arguments %q", command, args, want)
	}
}

func TestIdentityCommandForSudoInvokedRootResolvesMissingUID(t *testing.T) {
	command, args, err := identityCommandForUser(0, func(key string) string {
		if key == "SUDO_USER" {
			return "operator"
		}
		return ""
	}, func(name string) (*user.User, error) {
		if name != "operator" {
			t.Fatalf("lookup name = %q, want operator", name)
		}
		return &user.User{Username: name, Uid: "1008"}, nil
	}, "env", []string{"KEY=value"})
	if err != nil {
		t.Fatalf("identityCommandForUser: %v", err)
	}
	if command != "sudo" || args[0] != "-u" || args[1] != "operator" || args[5] != "XDG_RUNTIME_DIR=/run/user/1008" {
		t.Fatalf("command = %q %q, want resolved operator UID", command, args)
	}
}
