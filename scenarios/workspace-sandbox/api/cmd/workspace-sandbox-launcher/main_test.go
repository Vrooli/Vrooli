package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"workspace-sandbox/internal/driverid"
	"workspace-sandbox/internal/driverpref"
)

func testDeps(t *testing.T) deps {
	t.Helper()
	return deps{
		goos:           "linux",
		geteuid:        func() int { return 1000 },
		getenv:         func(string) string { return "" },
		probePlain:     func() error { return errors.New("plain denied") },
		probeAppArmor:  func() error { return nil },
		defaultBaseDir: func() string { return t.TempDir() },
		execProcess: func(string, []string, []string) error {
			return nil
		},
		runAndWait: func(string, []string, []string) error {
			return nil
		},
		writeDiagnostic: func(string, ...any) {},
	}
}

func TestChooseLaunchModeLinuxDefaultUsesAppArmorWhenAvailable(t *testing.T) {
	mode, err := chooseLaunchMode("", testDeps(t))
	if err != nil {
		t.Fatal(err)
	}
	if mode != modeAppArmor {
		t.Fatalf("mode = %q, want %q", mode, modeAppArmor)
	}
}

func TestChooseLaunchModeLinuxDefaultUsesPlainUnshareFallback(t *testing.T) {
	d := testDeps(t)
	d.probeAppArmor = func() error { return errors.New("profile missing") }
	d.probePlain = func() error { return nil }
	mode, err := chooseLaunchMode("", d)
	if err != nil {
		t.Fatal(err)
	}
	if mode != modeUnshare {
		t.Fatalf("mode = %q, want %q", mode, modeUnshare)
	}
}

func TestChooseLaunchModeLinuxDefaultFailsWithoutUsernsPath(t *testing.T) {
	d := testDeps(t)
	d.probeAppArmor = func() error { return errors.New("profile missing") }
	d.probePlain = func() error { return errors.New("plain denied") }
	_, err := chooseLaunchMode("", d)
	if err == nil {
		t.Fatal("chooseLaunchMode succeeded")
	}
	if !strings.Contains(err.Error(), "workspace_sandbox_userns") {
		t.Fatalf("error = %v", err)
	}
}

func TestChooseLaunchModeDirectDriversDoNotProbeUserns(t *testing.T) {
	d := testDeps(t)
	probed := false
	d.probeAppArmor = func() error {
		probed = true
		return nil
	}
	mode, err := chooseLaunchMode(driverid.Copy, d)
	if err != nil {
		t.Fatal(err)
	}
	if mode != modeDirect {
		t.Fatalf("mode = %q, want %q", mode, modeDirect)
	}
	if probed {
		t.Fatal("direct driver unexpectedly probed userns")
	}
}

func TestChooseLaunchModeOverlayfsRootRequiresRoot(t *testing.T) {
	d := testDeps(t)
	_, err := chooseLaunchMode(driverid.OverlayfsRoot, d)
	if err == nil {
		t.Fatal("chooseLaunchMode succeeded")
	}
	d.geteuid = func() int { return 0 }
	mode, err := chooseLaunchMode(driverid.OverlayfsRoot, d)
	if err != nil {
		t.Fatal(err)
	}
	if mode != modeDirect {
		t.Fatalf("mode = %q, want %q", mode, modeDirect)
	}
}

func TestChooseLaunchModeNonLinuxIsDirect(t *testing.T) {
	d := testDeps(t)
	d.goos = "darwin"
	mode, err := chooseLaunchMode(driverid.OverlayfsUserNS, d)
	if err != nil {
		t.Fatal(err)
	}
	if mode != modeDirect {
		t.Fatalf("mode = %q, want %q", mode, modeDirect)
	}
}

func TestRunReadsPreferenceAndExecsExpectedCommand(t *testing.T) {
	dir := t.TempDir()
	if err := driverpref.Save(dir, driverid.Copy); err != nil {
		t.Fatal(err)
	}
	var gotArgv0 string
	var gotArgv []string
	d := testDeps(t)
	d.defaultBaseDir = func() string { return dir }
	d.execProcess = func(argv0 string, argv []string, env []string) error {
		gotArgv0 = argv0
		gotArgv = argv
		return nil
	}
	if err := run([]string{"/tmp/api"}, d); err != nil {
		t.Fatal(err)
	}
	if gotArgv0 != "/tmp/api" {
		t.Fatalf("argv0 = %q, want /tmp/api", gotArgv0)
	}
	if len(gotArgv) != 1 || gotArgv[0] != "/tmp/api" {
		t.Fatalf("argv = %#v", gotArgv)
	}
}

func TestRunRejectsMalformedPreference(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, driverpref.FileName), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	d := testDeps(t)
	d.defaultBaseDir = func() string { return dir }
	if err := run(nil, d); err == nil {
		t.Fatal("run succeeded with malformed preference")
	}
}
