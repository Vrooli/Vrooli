package workspacesandboxuserns

import (
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

func restoreHostreq(t *testing.T) func() {
	t.Helper()
	origLookPath := hostreqkit.LookPathFn
	origReadFile := hostreqkit.ReadFileFn
	origCombinedOutput := hostreqkit.CombinedOutputFn
	origRunCommand := hostreqkit.RunCommandFn
	origWriteTempFile := hostreqkit.WriteTempFileFn
	origRunningAsRoot := hostreqkit.RunningAsRootFn
	return func() {
		hostreqkit.LookPathFn = origLookPath
		hostreqkit.ReadFileFn = origReadFile
		hostreqkit.CombinedOutputFn = origCombinedOutput
		hostreqkit.RunCommandFn = origRunCommand
		hostreqkit.WriteTempFileFn = origWriteTempFile
		hostreqkit.RunningAsRootFn = origRunningAsRoot
	}
}

var newTestHandler = workspaceSandboxTestHandler

func workspaceSandboxTestHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{
		Name:    "workspace_sandbox_userns",
		Handler: "workspace_sandbox_userns",
	})
}

var linuxReq = workspaceSandboxLinuxReq

func workspaceSandboxLinuxReq() hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name:     "workspace_sandbox_userns",
		Kind:     hostreqspec.KindSafeguard,
		Required: true,
	}
}

var linuxHost = workspaceSandboxLinuxHost

func workspaceSandboxLinuxHost() hostreqkit.Host {
	return hostreqkit.Host{OS: "linux"}
}

func TestInspectNonLinuxNotApplicable(t *testing.T) {
	status := newTestHandler().Inspect(hostreqkit.Host{OS: "darwin"}, linuxReq())
	if status.SupportClass != hostreqkit.SupportNotApplicable {
		t.Fatalf("SupportClass = %q", status.SupportClass)
	}
	if status.ExecutionState != hostreqkit.ExecutionNotApplicable {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectPlainUnshareSatisfied(t *testing.T) {
	restore := restoreHostreq(t)
	defer restore()

	hostreqkit.RunningAsRootFn = func() bool { return false }
	hostreqkit.LookPathFn = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "unshare" {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected command %s", name)
	}

	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("Applied = false, notes: %v", status.Notes)
	}
}

func TestInspectRootPlainUnshareDoesNotSatisfyUnprivilegedRequirement(t *testing.T) {
	restore := restoreHostreq(t)
	defer restore()

	hostreqkit.RunningAsRootFn = func() bool { return true }
	hostreqkit.LookPathFn = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	hostreqkit.ReadFileFn = func(string) ([]byte, error) { return nil, os.ErrNotExist }
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "unshare" {
			return nil, nil
		}
		return nil, fmt.Errorf("profile missing")
	}

	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("root-only plain unshare marked safeguard applied")
	}
	if status.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestInspectAppArmorProfileSatisfied(t *testing.T) {
	restore := restoreHostreq(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "aa-exec" {
			return nil, nil
		}
		return nil, fmt.Errorf("denied")
	}

	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if !status.Applied {
		t.Fatalf("Applied = false, notes: %v", status.Notes)
	}
}

func TestInspectPendingWhenNoLaunchPathWorks(t *testing.T) {
	restore := restoreHostreq(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		switch path {
		case profilePath:
			return []byte("stale"), nil
		default:
			return []byte("1\n"), nil
		}
	}
	hostreqkit.CombinedOutputFn = func(string, ...string) ([]byte, error) {
		return nil, fmt.Errorf("denied")
	}

	status := newTestHandler().Inspect(linuxHost(), linuxReq())
	if status.Applied {
		t.Fatal("Applied = true")
	}
	if status.ExecutionState != hostreqkit.ExecutionPending {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}

func TestApplyInstallsLoadsAndValidatesProfile(t *testing.T) {
	restore := restoreHostreq(t)
	defer restore()

	var commands []string
	hostreqkit.RunningAsRootFn = func() bool { return true }
	hostreqkit.LookPathFn = func(name string) (string, error) { return "/usr/bin/" + name, nil }
	tempFile := filepath.Join(t.TempDir(), "profile")
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		if content != profileContent {
			t.Fatalf("unexpected profile content")
		}
		if err := os.WriteFile(tempFile, []byte(content), 0o644); err != nil {
			return "", err
		}
		return tempFile, nil
	}
	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		commands = append(commands, name+" "+fmt.Sprint(args))
		return nil
	}
	hostreqkit.CombinedOutputFn = func(name string, args ...string) ([]byte, error) {
		if name == "aa-exec" {
			return nil, nil
		}
		return nil, fmt.Errorf("unexpected command %s", name)
	}

	status, err := newTestHandler().Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q, notes: %v", status.ExecutionState, status.Notes)
	}
	want := []string{
		"mkdir [-p /etc/apparmor.d]",
		"install [-m 644 " + tempFile + " /etc/apparmor.d/vrooli-workspace-sandbox]",
		"apparmor_parser [-r /etc/apparmor.d/vrooli-workspace-sandbox]",
	}
	if !reflect.DeepEqual(commands, want) {
		t.Fatalf("commands = %#v, want %#v", commands, want)
	}
}

func TestApplyFailsWithoutAaExec(t *testing.T) {
	restore := restoreHostreq(t)
	defer restore()

	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "aa-exec" {
			return "", os.ErrNotExist
		}
		return "/usr/bin/" + name, nil
	}
	status, err := newTestHandler().Apply(linuxHost(), hostreqkit.ItemStatus{
		SupportClass: hostreqkit.SupportSupported,
	}, hostreqkit.EnsureOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if status.ExecutionState != hostreqkit.ExecutionFailed {
		t.Fatalf("ExecutionState = %q", status.ExecutionState)
	}
}
