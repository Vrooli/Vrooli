package main

import (
	"context"
	"strings"
	"testing"
)

func findCheck(result PreflightResult, name string) PreflightCheck {
	for _, check := range result.Checks {
		if check.Name == name {
			return check
		}
	}
	return PreflightCheck{}
}

// The unit's PATH lacked /usr/local/go/bin on this host, so the floor's `go
// mod download` could not run and every attempt burned a breaker slot. The
// preflight must say "toolchain" and "go" before any heal is tried.
func TestPreflightNamesToolchainWhenGoMissing(t *testing.T) {
	isolatedHome(t)
	config := testConfig(t)
	config.VrooliCmdPath = fakeVrooli(t, contractBody(""))
	t.Setenv("PATH", t.TempDir())
	empty := t.TempDir()
	restore := toolchainPathEntries
	toolchainPathEntries = func(string, string) []string { return []string{empty} }
	t.Cleanup(func() { toolchainPathEntries = restore })

	result := Preflight(context.Background(), config)
	if result.OK {
		t.Fatal("preflight passed without a go toolchain")
	}
	toolchain := findCheck(result, "toolchain")
	if toolchain.Status != CheckFailed || !strings.Contains(toolchain.Reason, "go not found") || !strings.Contains(toolchain.Reason, empty) {
		t.Fatalf("toolchain check = %+v; want failed, naming go and every searched location", toolchain)
	}
	for _, name := range []string{"cli-resolves", "cli-answers", "cli-contract", "state-writable", "root-resolves"} {
		if check := findCheck(result, name); check.Status != CheckOK {
			t.Errorf("%s = %+v, want ok", name, check)
		}
	}
	if failed := result.Failed(); len(failed) != 1 || !strings.HasPrefix(failed[0], "toolchain:") {
		t.Fatalf("Failed() = %v", failed)
	}
	if class := result.FailureClass(); class != "preflight" {
		t.Fatalf("a missing toolchain is the preflight's own failure, got class %q", class)
	}

	config.ManageAPILifecycle = false
	if check := findCheck(Preflight(context.Background(), config), "toolchain"); check.Status != CheckSkipped {
		t.Fatalf("toolchain must be skipped when the recovery floor is off, got %+v", check)
	}
}

func TestPreflightFailsWhenTheCLIIsOutOfContract(t *testing.T) {
	isolatedHome(t)
	config := testConfig(t)
	config.VrooliCmdPath = fakeVrooli(t, usageBody)
	result := Preflight(context.Background(), config)
	if result.OK {
		t.Fatal("a CLI that rejects its argv passed preflight")
	}
	for _, name := range []string{"cli-answers", "cli-contract"} {
		if check := findCheck(result, name); check.Status != CheckFailed || check.Class != "usage" || !strings.Contains(check.Reason, "usage") {
			t.Errorf("%s = %+v, want failed with the usage class", name, check)
		}
	}
	if result.FailureClass() != "usage" {
		t.Fatalf("FailureClass() = %q, want usage", result.FailureClass())
	}
}

func TestPreflightSkipsInvocationsWhenTheBinaryIsMissing(t *testing.T) {
	isolatedHome(t)
	config := testConfig(t)
	config.VrooliCmdPath = ""
	result := Preflight(context.Background(), config)
	if findCheck(result, "cli-resolves").Status != CheckFailed {
		t.Fatal("cli-resolves must fail without a binary")
	}
	for _, name := range []string{"cli-answers", "cli-contract"} {
		if check := findCheck(result, name); check.Status != CheckSkipped {
			t.Errorf("%s = %+v, want skipped", name, check)
		}
	}
}
