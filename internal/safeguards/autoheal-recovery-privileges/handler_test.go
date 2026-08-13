package autohealrecoveryprivileges

import (
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func autohealTestRequirement() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "autoheal_recovery_privileges",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

func autohealTestHost(osName string) hostreqkit.Host { return hostreqkit.Host{OS: osName} }

func stubAutohealHandler(t *testing.T, present bool) *string {
	t.Helper()
	origRoot := hostreqkit.RunningAsRootFn
	origRead := hostreqkit.ReadFileFn
	origWrite := hostreqkit.WriteTempFileFn
	origRun := hostreqkit.RunCommandFn
	origFacts := hostreqkit.ElevationFactsFn
	origUser := os.Getenv("USER")
	origSudoUser := os.Getenv("SUDO_USER")
	os.Setenv("USER", "autoheal-test")
	os.Unsetenv("SUDO_USER")
	hostreqkit.RunningAsRootFn = func() bool { return true }
	hostreqkit.ElevationFactsFn = func() hostreqkit.ElevationFacts {
		return hostreqkit.ElevationFacts{Platform: "linux", Elevated: true, CanElevate: true, Mechanism: "root"}
	}
	want := buildSudoersContent("autoheal-test")
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == sudoersPath && present {
			return []byte(want), nil
		}
		return nil, os.ErrNotExist
	}
	var written string
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		written = content
		return "/tmp/autoheal-sudoers-test", nil
	}
	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error { return nil }
	t.Cleanup(func() {
		hostreqkit.RunningAsRootFn = origRoot
		hostreqkit.ReadFileFn = origRead
		hostreqkit.WriteTempFileFn = origWrite
		hostreqkit.RunCommandFn = origRun
		hostreqkit.ElevationFactsFn = origFacts
		os.Setenv("USER", origUser)
		if origSudoUser == "" {
			os.Unsetenv("SUDO_USER")
		} else {
			os.Setenv("SUDO_USER", origSudoUser)
		}
	})
	return &written
}

func TestApplyIsIdempotentAndUsesValidatedInstall(t *testing.T) {
	written := stubAutohealHandler(t, false)
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_recovery_privileges"})
	req := autohealTestRequirement()
	status := h.Inspect(autohealTestHost("linux"), req)
	if status.Required != true {
		t.Fatalf("root setup should promote missing grant to required: %+v", status)
	}
	first, err := h.Apply(autohealTestHost("linux"), status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("first Apply: %v", err)
	}
	if !first.Applied || first.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("first Apply status = %+v", first)
	}
	if !strings.Contains(*written, "NOPASSWD:") || strings.Contains(*written, "*") {
		t.Fatalf("grant is not a literal, wildcard-free sudoers policy: %q", *written)
	}
	firstBytes := *written

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == sudoersPath {
			return []byte(firstBytes), nil
		}
		return nil, os.ErrNotExist
	}
	second := h.Inspect(autohealTestHost("linux"), req)
	if !second.Applied || second.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("second Inspect status = %+v", second)
	}
	secondResult, err := h.Apply(autohealTestHost("linux"), second, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("second Apply: %v", err)
	}
	if secondResult.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Fatalf("second Apply status = %+v", secondResult)
	}
	if *written != firstBytes {
		t.Fatalf("idempotent apply changed grant bytes")
	}
}

func TestApplyVisudoFailureDoesNotInstall(t *testing.T) {
	stubAutohealHandler(t, false)
	var commands []string
	hostreqkit.RunCommandFn = func(name string, args []string, _ hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+strings.Join(args, " "))
		if name == "visudo" {
			return errors.New("invalid sudoers")
		}
		return nil
	}
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_recovery_privileges"})
	status := h.Inspect(autohealTestHost("linux"), autohealTestRequirement())
	result, err := h.Apply(autohealTestHost("linux"), status, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if result.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("status = %+v, want failed", result)
	}
	for _, command := range commands {
		if strings.Contains(command, " install ") {
			t.Fatalf("install ran after visudo failure: %v", commands)
		}
	}
	if !strings.Contains(strings.Join(result.Notes, " "), "not touched") {
		t.Fatalf("failure notes should say real drop-in was not touched: %v", result.Notes)
	}
}

func TestInspectNonRootRequiresManualSetup(t *testing.T) {
	stubAutohealHandler(t, false)
	hostreqkit.RunningAsRootFn = func() bool { return false }
	status := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_recovery_privileges"}).Inspect(autohealTestHost("linux"), autohealTestRequirement())
	if status.ExecutionState != hostreqkit.ExecutionManualActionRequired {
		t.Fatalf("ExecutionState = %q, want manual action required", status.ExecutionState)
	}
	if !strings.Contains(strings.Join(status.Notes, " "), "sudo vrooli setup") {
		t.Fatalf("notes should contain concrete setup command: %v", status.Notes)
	}
}

func TestInspectNonLinuxIsUnsupported(t *testing.T) {
	stubAutohealHandler(t, false)
	h := NewHandler(hostreqkit.SafeguardManifest{Name: "autoheal_recovery_privileges"})
	for _, osName := range []string{"darwin", "windows"} {
		status := h.Inspect(autohealTestHost(osName), autohealTestRequirement())
		if status.SupportClass != hostreqkit.SupportUnsupported || status.ExecutionState != hostreqkit.ExecutionUnsupported {
			t.Errorf("%s status = %+v, want unsupported", osName, status)
		}
	}
}
