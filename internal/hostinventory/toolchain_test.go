package hostinventory

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/vrooli/vrooli/internal/shell/shelltest"
)

func newToolchainCollector(goos string, commands CommandRunner) Collector {
	return Collector{GOOS: goos, GOARCH: "amd64", Commands: commands}.withDefaults()
}

func newToolchainSnapshot(goos string) *Snapshot {
	return &Snapshot{
		OS:              goos,
		RuntimeTools:    map[string]Tool{},
		ProbeStatuses:   map[string]string{},
		FieldProvenance: map[string]Provenance{},
	}
}

func TestAppleToolchainIsNotProbedOnNonDarwinHosts(t *testing.T) {
	// An Apple toolchain is categorically absent off darwin. Recording an empty
	// probe would present a terminal fact as a missing measurement.
	commands := &shelltest.Fake{Paths: map[string]string{}}
	snap := newToolchainSnapshot("linux")
	newToolchainCollector("linux", commands).collectAppleToolchain(context.Background(), snap, time.Now())

	if _, probed := snap.ProbeStatuses[ProbeAppleToolchain]; probed {
		t.Fatalf("apple toolchain must not be probed on linux, got %v", snap.ProbeStatuses)
	}
	if len(commands.Calls()) != 0 {
		t.Fatalf("expected no commands on linux, got %v", commands.Calls())
	}
}

func TestAppleToolchainReportsVersionAndSimulatorRuntimes(t *testing.T) {
	commands := &shelltest.Fake{
		Paths: map[string]string{ToolXcodebuild: "/usr/bin/xcodebuild", ToolXcrun: "/usr/bin/xcrun"},
		OutputStrings: map[string]string{
			"xcodebuild -version": "Xcode 26.1\nBuild version 17B55\n",
			"xcrun simctl list runtimes": strings.Join([]string{
				"== Runtimes ==",
				"iOS 26.1 (26.1 - 23B74) - com.apple.CoreSimulator.SimRuntime.iOS-26-1",
				"watchOS 26.0 (26.0 - 23R5) - com.apple.CoreSimulator.SimRuntime.watchOS-26-0",
			}, "\n"),
		},
	}
	snap := newToolchainSnapshot("darwin")
	newToolchainCollector("darwin", commands).collectAppleToolchain(context.Background(), snap, time.Now())

	if status := snap.ProbeStatuses[ProbeAppleToolchain]; status != "ok" {
		t.Fatalf("expected ok probe status, got %q", status)
	}
	xcodebuild := snap.RuntimeTools[ToolXcodebuild]
	if !xcodebuild.Present || xcodebuild.Version != "26.1" {
		t.Fatalf("expected xcodebuild 26.1 present, got %+v", xcodebuild)
	}
	simctl := snap.RuntimeTools[ToolSimctl]
	if !simctl.Present || !strings.Contains(simctl.Version, "iOS 26.1") {
		t.Fatalf("expected simctl to report iOS 26.1, got %+v", simctl)
	}
}

func TestAppleToolchainReportsFailureWhenXcodebuildIsUnusable(t *testing.T) {
	// A present-but-unusable xcodebuild (unaccepted license, command-line tools
	// only) must not read as a working toolchain.
	commands := &shelltest.Fake{
		Paths:  map[string]string{ToolXcodebuild: "/usr/bin/xcodebuild", ToolXcrun: "/usr/bin/xcrun"},
		Errors: map[string]error{"xcodebuild -version": errors.New("requires Xcode, but active developer directory is a CommandLineTools instance")},
	}
	snap := newToolchainSnapshot("darwin")
	newToolchainCollector("darwin", commands).collectAppleToolchain(context.Background(), snap, time.Now())

	if status := snap.ProbeStatuses[ProbeAppleToolchain]; status != "failed" {
		t.Fatalf("expected failed probe status, got %q", status)
	}
	if xcodebuild := snap.RuntimeTools[ToolXcodebuild]; xcodebuild.Present {
		t.Fatalf("an unusable xcodebuild must report absent, got %+v", xcodebuild)
	}
	if version := snap.RuntimeTools[ToolXcodebuild].Version; version != "" {
		t.Fatalf("an unusable xcodebuild must report no version, got %q", version)
	}
}

func TestAppleToolchainReportsMissingSimulatorRuntimesWithoutFailing(t *testing.T) {
	// On Intel hosts the universal runtime variant must be fetched explicitly,
	// so "no runtimes installed" is a real observation, not a probe error.
	commands := &shelltest.Fake{
		Paths: map[string]string{ToolXcodebuild: "/usr/bin/xcodebuild", ToolXcrun: "/usr/bin/xcrun"},
		OutputStrings: map[string]string{
			"xcodebuild -version":        "Xcode 26.1\n",
			"xcrun simctl list runtimes": "== Runtimes ==\n",
		},
	}
	snap := newToolchainSnapshot("darwin")
	newToolchainCollector("darwin", commands).collectAppleToolchain(context.Background(), snap, time.Now())

	if status := snap.ProbeStatuses[ProbeAppleToolchain]; status != "ok" {
		t.Fatalf("expected ok probe status, got %q", status)
	}
	if simctl := snap.RuntimeTools[ToolSimctl]; !simctl.Present || simctl.Version != "" {
		t.Fatalf("expected present simctl with no runtimes, got %+v", simctl)
	}
}

func TestAndroidToolchainReportsPartialAndAbsentStates(t *testing.T) {
	tests := []struct {
		name     string
		paths    map[string]string
		expected string
	}{
		{name: "absent", paths: map[string]string{}, expected: "tool_not_present"},
		{name: "partial", paths: map[string]string{ToolADB: "/opt/adb"}, expected: "partial"},
		{name: "complete", paths: map[string]string{
			ToolADB: "/opt/adb", ToolEmulator: "/opt/emulator",
			ToolSDKManager: "/opt/sdkmanager", ToolAVDManager: "/opt/avdmanager",
		}, expected: "ok"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			commands := &shelltest.Fake{Paths: tt.paths}
			snap := newToolchainSnapshot("linux")
			newToolchainCollector("linux", commands).collectAndroidToolchain(context.Background(), snap, time.Now())

			if status := snap.ProbeStatuses[ProbeAndroidToolchain]; status != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, status)
			}
			// Every probed tool must appear, so a consumer can distinguish
			// "probed and missing" from "never probed".
			for _, tool := range []string{ToolADB, ToolEmulator, ToolSDKManager, ToolAVDManager} {
				if _, recorded := snap.RuntimeTools[tool]; !recorded {
					t.Fatalf("tool %q was not recorded", tool)
				}
			}
		})
	}
}

func TestKVMIsOnlyReportedOnLinux(t *testing.T) {
	// macOS and Windows accelerate through their own hypervisors, so an absent
	// /dev/kvm there is not a finding and must not be recorded as one.
	snap := newToolchainSnapshot("darwin")
	newToolchainCollector("darwin", &shelltest.Fake{Paths: map[string]string{}}).
		collectAndroidToolchain(context.Background(), snap, time.Now())

	if _, recorded := snap.RuntimeTools[ToolKVM]; recorded {
		t.Fatalf("kvm must not be recorded on darwin, got %+v", snap.RuntimeTools[ToolKVM])
	}
}

func TestParseXcodeVersion(t *testing.T) {
	tests := []struct {
		name, input, expected string
	}{
		{name: "standard", input: "Xcode 26.1\nBuild version 17B55", expected: "26.1"},
		{name: "major only", input: "Xcode 27\nBuild version 18A1", expected: "27"},
		{name: "patch", input: "Xcode 26.0.1\n", expected: "26.0.1"},
		{name: "command line tools", input: "xcode-select: error: tool 'xcodebuild' requires Xcode", expected: ""},
		{name: "empty", input: "", expected: ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ParseXcodeVersion(tt.input); got != tt.expected {
				t.Fatalf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func TestParseSimulatorRuntimesIsSortedAndDeduplicated(t *testing.T) {
	input := strings.Join([]string{
		"iOS 26.1 (26.1 - 23B74) - com.apple.CoreSimulator.SimRuntime.iOS-26-1",
		"iOS 18.0 (18.0 - 22A1) - com.apple.CoreSimulator.SimRuntime.iOS-18-0",
		"iOS 26.1 duplicate - com.apple.CoreSimulator.SimRuntime.iOS-26-1",
		"tvOS 26.0 (26.0 - 23J1) - com.apple.CoreSimulator.SimRuntime.tvOS-26-0",
	}, "\n")

	got := ParseSimulatorRuntimes(input)
	want := []string{"iOS 18.0", "iOS 26.1", "tvOS 26.0"}
	if len(got) != len(want) {
		t.Fatalf("expected %v, got %v", want, got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("expected %v, got %v", want, got)
		}
	}
}

func TestParseSimulatorRuntimesReturnsEmptyForNoRuntimes(t *testing.T) {
	if got := ParseSimulatorRuntimes("== Runtimes ==\n"); len(got) != 0 {
		t.Fatalf("expected no runtimes, got %v", got)
	}
}
