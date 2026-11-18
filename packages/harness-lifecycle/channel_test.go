package agentinstall

import (
	"path/filepath"
	"testing"

	"github.com/vrooli/binaryfetch"
)

func TestNPMInstallPrefixForChannel(t *testing.T) {
	home := t.TempDir()
	defaultPrefix, err := NPMInstallPrefixForChannel(home, "opencode", "", "1.19.6")
	if err != nil {
		t.Fatal(err)
	}
	v2Prefix, err := NPMInstallPrefixForChannel(home, "opencode", "v2", "2.3.2")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(filepath.Dir(defaultPrefix)) != "cli" {
		t.Fatalf("default prefix should stay directly under cli: %s", defaultPrefix)
	}
	if filepath.Base(filepath.Dir(v2Prefix)) != "v2" {
		t.Fatalf("v2 prefix should be scoped under a channel root: %s", v2Prefix)
	}
	if filepath.Dir(filepath.Dir(v2Prefix)) != filepath.Dir(defaultPrefix) {
		t.Fatalf("channel roots should share the same cli parent: %s vs %s", v2Prefix, defaultPrefix)
	}
	if _, err := NPMInstallPrefixForChannel(home, "opencode", "../escape", "2.3.2"); err == nil {
		t.Fatal("channel traversal must be refused")
	}
}

func TestLoadHarnessChannelSelectsDeclaredTarget(t *testing.T) {
	acquisition := &binaryfetch.Acquisition{
		Kind: "npm",
		Targets: []binaryfetch.AcquisitionTarget{
			{When: map[string]string{"os": "linux", "arch": "amd64"}, Package: "opencode-ai", Version: "1.19.6", SHA256: "sha", BinPath: "node_modules/opencode-linux-x64-baseline/bin/opencode"},
			{When: map[string]string{"os": "linux", "arch": "amd64"}, Channel: "v2", Package: "@opencode/cli", Version: "2.3.2", SHA256: "sha", BinPath: "node_modules/@opencode/cli-linux-x64-baseline/bin/opencode"},
		},
	}
	deps := harnessLoadDeps{
		loadManifest: func(string) (harnessManifest, error) {
			return harnessManifest{Name: "opencode", Acquisition: acquisition}, nil
		},
		homeDir: func() (string, error) { return "/home/test", nil },
		goos:    "linux",
		goarch:  "amd64",
	}
	defaultHarness, err := loadHarnessWithChannel(t.TempDir(), "opencode", "", deps)
	if err != nil {
		t.Fatal(err)
	}
	if defaultHarness.Package != "opencode-ai" || defaultHarness.Channel != "" || defaultHarness.ReviewedVersion != "1.19.6" {
		t.Fatalf("default harness = %+v", defaultHarness)
	}
	v2Harness, err := loadHarnessWithChannel(t.TempDir(), "opencode", "v2", deps)
	if err != nil {
		t.Fatal(err)
	}
	if v2Harness.Package != "@opencode/cli" || v2Harness.Channel != "v2" || v2Harness.ReviewedVersion != "2.3.2" {
		t.Fatalf("v2 harness = %+v", v2Harness)
	}
	if v2Harness.Executable != "node_modules/@opencode/cli-linux-x64-baseline/bin/opencode" {
		t.Fatalf("v2 executable = %q", v2Harness.Executable)
	}
	if _, err := loadHarnessWithChannel(t.TempDir(), "opencode", "v3", deps); err == nil {
		t.Fatal("an undeclared channel must be refused")
	}
}

func TestMajorVersionAndChannelForMajor(t *testing.T) {
	if major, ok := MajorVersion("opencode v2.3.2"); !ok || major != 2 {
		t.Fatalf("MajorVersion = (%d, %v)", major, ok)
	}
	if _, ok := MajorVersion("not a version"); ok {
		t.Fatal("invalid version must not parse")
	}
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	channel, err := ChannelForMajor(root, "opencode", 2)
	if err != nil || channel != "v2" {
		t.Fatalf("ChannelForMajor(2) = (%q, %v)", channel, err)
	}
	if _, err := ChannelForMajor(root, "opencode", 9); err == nil {
		t.Fatal("an undeclared major must be refused")
	}
}

func TestInspectRecognizesManagedLinkFromAnotherChannel(t *testing.T) {
	f := newHarnessFixture(t, "opencode", "opencode")
	f.harness.Channel = "v2"
	f.harness.ReviewedVersion = "2.0.0"
	f.harness.Acquisition = &binaryfetch.Acquisition{
		Kind: "npm",
		Targets: []binaryfetch.AcquisitionTarget{
			{When: map[string]string{"os": "linux", "arch": "amd64"}, Package: "@fixture/opencode", Version: "1.0.0", SHA256: "sha"},
			{When: map[string]string{"os": "linux", "arch": "amd64"}, Channel: "v2", Package: "@fixture/opencode", Version: "2.0.0", SHA256: "sha"},
		},
	}
	f.latest = "2.0.0"
	f.link("1.5.0") // installed by the default channel
	report := f.inspect()
	wantState(t, report, StateUpdateReady, ActionUpdate)
	if report.ManagedVersion != "1.5.0" || report.Installed != "1.5.0" || report.Latest != "2.0.0" {
		t.Fatalf("cross-channel report = %+v", report)
	}
}
