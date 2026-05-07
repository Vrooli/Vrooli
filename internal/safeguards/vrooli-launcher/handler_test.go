package vroolilauncher

import (
	"errors"
	"io/fs"
	"strings"
	"testing"

	"github.com/vrooli/vrooli/internal/hostreqkit"
	"github.com/vrooli/vrooli/internal/hostreqspec"
)

type capturedCommand struct {
	Name string
	Args []string
}

// stubAll wires test-friendly stubs over the package-level seams the handler
// touches. Returns the captured-commands slice plus a restore callback.
func stubAll(t *testing.T) (*[]capturedCommand, func()) {
	t.Helper()
	origRun := hostreqkit.RunCommandFn
	origLook := hostreqkit.LookPathFn
	origRead := hostreqkit.ReadFileFn
	origWriteTemp := hostreqkit.WriteTempFileFn

	captured := []capturedCommand{}

	hostreqkit.RunCommandFn = func(name string, args []string, opts hostreqkit.EnsureOptions) error {
		captured = append(captured, capturedCommand{Name: name, Args: append([]string(nil), args...)})
		return nil
	}
	hostreqkit.LookPathFn = func(name string) (string, error) {
		if name == "sudo" {
			return "/usr/bin/sudo", nil
		}
		return "/usr/bin/" + name, nil
	}
	// Default: shim is missing. Individual tests override.
	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return nil, fs.ErrNotExist
	}
	hostreqkit.WriteTempFileFn = func(content string) (string, error) {
		// Tests don't need a real file; the handler only passes the path
		// to `install`, which is captured in RunCommandFn above.
		return "/tmp/vrooli-launcher-stub", nil
	}

	return &captured, func() {
		hostreqkit.RunCommandFn = origRun
		hostreqkit.LookPathFn = origLook
		hostreqkit.ReadFileFn = origRead
		hostreqkit.WriteTempFileFn = origWriteTemp
	}
}

func newHandler() hostreqkit.Handler {
	return NewHandler(hostreqkit.SafeguardManifest{Name: "vrooli_launcher", Handler: "vrooli_launcher"})
}

func req(manual bool) hostreqspec.ResolvedRequirement {
	return hostreqspec.ResolvedRequirement{
		Name: "vrooli_launcher", Kind: hostreqspec.KindSafeguard, Required: false, Manual: manual,
	}
}

func linuxHost() hostreqkit.Host  { return hostreqkit.Host{OS: "linux"} }
func darwinHost() hostreqkit.Host { return hostreqkit.Host{OS: "darwin"} }
func winHost() hostreqkit.Host    { return hostreqkit.Host{OS: "windows"} }

func TestInspectMissingShimIsPending(t *testing.T) {
	_, restore := stubAll(t)
	defer restore()

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.SupportClass != hostreqkit.SupportSupported {
		t.Errorf("SupportClass = %q", st.SupportClass)
	}
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q, want pending", st.ExecutionState)
	}
	if st.Applied {
		t.Errorf("Applied should be false when shim is missing")
	}
}

func TestInspectMatchingShimIsAlreadyPresent(t *testing.T) {
	_, restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		if path == LauncherPath {
			return []byte(shimContent), nil
		}
		return nil, fs.ErrNotExist
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionAlreadyPresent {
		t.Errorf("ExecutionState = %q, want already_present", st.ExecutionState)
	}
	if !st.Applied {
		t.Errorf("Applied should be true")
	}
}

func TestInspectStaleShimIsPending(t *testing.T) {
	// An older shim with different content should be re-applied, not
	// reported as already present. Catches the case where shimContent
	// evolves between Vrooli versions.
	_, restore := stubAll(t)
	defer restore()

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return []byte("#!/bin/sh\nexec /old/vrooli \"$@\"\n"), nil
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	if st.ExecutionState != hostreqkit.ExecutionPending {
		t.Errorf("ExecutionState = %q, want pending (stale shim)", st.ExecutionState)
	}
}

func TestInspectDarwinIsSupported(t *testing.T) {
	// macOS uses the same shim — local accounts read from /etc/passwd, sudo's
	// secure_path includes /usr/local/bin. Keep parity with Linux.
	_, restore := stubAll(t)
	defer restore()

	st := newHandler().Inspect(darwinHost(), req(false))
	if st.SupportClass != hostreqkit.SupportSupported {
		t.Errorf("SupportClass = %q on darwin, want supported", st.SupportClass)
	}
}

func TestInspectWindowsUnsupported(t *testing.T) {
	_, restore := stubAll(t)
	defer restore()

	st := newHandler().Inspect(winHost(), req(false))
	if st.SupportClass != hostreqkit.SupportUnsupported {
		t.Errorf("SupportClass = %q on windows, want unsupported", st.SupportClass)
	}
	if st.ExecutionState != hostreqkit.ExecutionUnsupported {
		t.Errorf("ExecutionState = %q on windows", st.ExecutionState)
	}
}

func TestApplyWritesShimUnderSudo(t *testing.T) {
	cmds, restore := stubAll(t)
	defer restore()

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "ask"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionApplied {
		t.Fatalf("ExecutionState = %q, want applied", out.ExecutionState)
	}
	if !out.Applied {
		t.Errorf("Applied should be true after successful Apply")
	}

	// The Apply path should call `install -m 0755 <tmp> /usr/local/bin/vrooli`,
	// wrapped with sudo via WithSudo. Confirm by inspecting the captured
	// command + args.
	var found bool
	for _, c := range *cmds {
		joined := c.Name + " " + strings.Join(c.Args, " ")
		if strings.Contains(joined, "install -m 0755") && strings.Contains(joined, LauncherPath) {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected `install -m 0755 ... %s` to be invoked, captured: %+v", LauncherPath, *cmds)
	}
}

func TestApplyDryRunSkipsWrite(t *testing.T) {
	cmds, restore := stubAll(t)
	defer restore()

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{DryRun: true})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionWouldApply {
		t.Fatalf("ExecutionState = %q, want would_apply", out.ExecutionState)
	}
	for _, c := range *cmds {
		if strings.Contains(c.Name+" "+strings.Join(c.Args, " "), "install") {
			t.Errorf("dry-run should not invoke install: %+v", c)
		}
	}
}

func TestApplySudoSkipReportsFailure(t *testing.T) {
	// SudoMode=skip with sudo on PATH returns the typed sentinel; the
	// handler folds it into ExecutionFailed with the sentinel string in
	// notes, which the runtime's apply loop then promotes to
	// BlockingNeedsSudo for the renderer.
	_, restore := stubAll(t)
	defer restore()

	hostreqkit.RunCommandFn = func(string, []string, hostreqkit.EnsureOptions) error {
		t.Fatal("RunCommandFn should not run when WithSudo refuses")
		return errors.New("unreachable")
	}

	st := newHandler().Inspect(linuxHost(), req(false))
	out, err := newHandler().Apply(linuxHost(), st, hostreqkit.EnsureOptions{SudoMode: "skip"})
	if err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if out.ExecutionState != hostreqkit.ExecutionFailed {
		t.Errorf("ExecutionState = %q, want failed", out.ExecutionState)
	}
	if !strings.Contains(strings.Join(out.Notes, " | "), "sudo skipped") {
		t.Errorf("notes should mention sudo skipped: %v", out.Notes)
	}
}

func TestInspectSelfPromotesToRequiredWhenRoot(t *testing.T) {
	// When the vrooli process is running as root (typically `sudo vrooli
	// setup`), the launcher handler promotes the requirement to required
	// so the runtime applies it without --include-optional. This is the
	// bootstrap path: a single `sudo vrooli setup` becomes one-stop and
	// installs the shim as a side effect.
	_, restore := stubAll(t)
	defer restore()

	origRoot := hostreqkit.RunningAsRootFn
	defer func() { hostreqkit.RunningAsRootFn = origRoot }()
	hostreqkit.RunningAsRootFn = func() bool { return true }

	// Manifest declares Required=false (matches service.json).
	requirement := hostreqspec.ResolvedRequirement{
		Name: "vrooli_launcher", Kind: hostreqspec.KindSafeguard, Required: false,
	}
	st := newHandler().Inspect(linuxHost(), requirement)
	if !st.Required {
		t.Fatalf("Required = false; expected handler to self-promote when running as root")
	}
}

func TestInspectStaysOptionalWhenNotRoot(t *testing.T) {
	_, restore := stubAll(t)
	defer restore()

	origRoot := hostreqkit.RunningAsRootFn
	defer func() { hostreqkit.RunningAsRootFn = origRoot }()
	hostreqkit.RunningAsRootFn = func() bool { return false }

	requirement := hostreqspec.ResolvedRequirement{
		Name: "vrooli_launcher", Kind: hostreqspec.KindSafeguard, Required: false,
	}
	st := newHandler().Inspect(linuxHost(), requirement)
	if st.Required {
		t.Fatalf("Required = true; expected to stay optional when not running as root")
	}
}

func TestInspectSelfPromotionDoesNotOverrideAlreadyPresent(t *testing.T) {
	// If the shim is already present, no promotion happens — there's
	// nothing to apply, and the report would be misleading otherwise.
	_, restore := stubAll(t)
	defer restore()

	origRoot := hostreqkit.RunningAsRootFn
	defer func() { hostreqkit.RunningAsRootFn = origRoot }()
	hostreqkit.RunningAsRootFn = func() bool { return true }

	hostreqkit.ReadFileFn = func(path string) ([]byte, error) {
		return []byte(shimContent), nil
	}

	requirement := hostreqspec.ResolvedRequirement{
		Name: "vrooli_launcher", Kind: hostreqspec.KindSafeguard, Required: false,
	}
	st := newHandler().Inspect(linuxHost(), requirement)
	if st.Required {
		t.Fatalf("Required = true on AlreadyPresent path; promotion should only fire when shim is missing")
	}
}

func TestNameAndKind(t *testing.T) {
	h := newHandler()
	if h.Name() != "vrooli_launcher" {
		t.Errorf("Name = %q", h.Name())
	}
	if h.Kind() != hostreqspec.KindSafeguard {
		t.Errorf("Kind = %q", h.Kind())
	}
}
