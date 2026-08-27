package hostinventory

import (
	"context"
	"fmt"
	"os"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/vrooli/vrooli/internal/hostreqspec"
)

const (
	toolchainParameterA = 2
	toolchainParameterB = 3
	toolchainParameterC = 4
)

// Mobile delivery ramps need to know which build and device toolchains a host
// actually has before they can claim a target. Detection of host state belongs
// to the control plane, so the facts are collected here and projected through
// the normal inventory snapshot rather than being declared by a node, asserted
// by a scenario, or inferred from the operating-system name.
//
// Every probe here is capability-shaped: it reports what was observed and, when
// a tool is absent, leaves the entry present-but-false so a consumer can tell
// "probed and missing" apart from "never probed".

// Toolchain probe status keys published in Snapshot.ProbeStatuses.
const (
	ProbeAppleToolchain   = "apple_toolchain"
	ProbeAndroidToolchain = "android_toolchain"
)

// Runtime tool names published in Snapshot.RuntimeTools. These are stable
// identifiers consumers may match on; they are deliberately the executable
// names so a reader can reproduce the probe by hand.
const (
	ToolXcodebuild = "xcodebuild"
	ToolXcrun      = "xcrun"
	ToolSimctl     = "simctl"
	ToolADB        = "adb"
	ToolEmulator   = "emulator"
	ToolSDKManager = "sdkmanager"
	ToolAVDManager = "avdmanager"
	ToolKVM        = "kvm"
)

var xcodeVersionPattern = regexp.MustCompile(`Xcode\s+([0-9]+(?:\.[0-9]+)*)`)

// simulatorRuntimePattern matches the runtime identifiers simctl reports, e.g.
// "com.apple.CoreSimulator.SimRuntime.iOS-26-1".
var simulatorRuntimePattern = regexp.MustCompile(`SimRuntime\.(iOS|watchOS|tvOS|xrOS)-([0-9]+(?:-[0-9]+)*)`)

// collectAppleToolchain records the Apple build toolchain on darwin hosts.
//
// It never runs on a non-darwin host: an Apple toolchain is categorically
// absent elsewhere, and recording an empty probe there would misrepresent a
// terminal fact as a missing measurement.
func (c Collector) collectAppleToolchain(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	if hostreqspec.PlatformFromGOOS(snap.OS) != hostreqspec.PlatformMacOS {
		return
	}

	xcodebuildPath, xcodebuildErr := c.Commands.LookPath(ToolXcodebuild)
	snap.RuntimeTools[ToolXcodebuild] = Tool{Present: xcodebuildErr == nil, Path: xcodebuildPath}
	xcrunPath, xcrunErr := c.Commands.LookPath(ToolXcrun)
	snap.RuntimeTools[ToolXcrun] = Tool{Present: xcrunErr == nil, Path: xcrunPath}

	if xcodebuildErr != nil {
		snap.ProbeStatuses[ProbeAppleToolchain] = "tool_not_present"
		return
	}

	if out, err := c.Commands.Run(ctx, ToolXcodebuild, "-version"); err != nil {
		// A present-but-unusable xcodebuild is the common "license not
		// accepted" or "command line tools only" state. Report it rather than
		// letting a version-less tool read as a working toolchain.
		snap.RuntimeTools[ToolXcodebuild] = Tool{Present: false, Path: xcodebuildPath}
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("xcodebuild -version: %v", err))
		snap.ProbeStatuses[ProbeAppleToolchain] = "failed"
		return
	} else if version := ParseXcodeVersion(string(out)); version != "" {
		snap.RuntimeTools[ToolXcodebuild] = Tool{Present: true, Path: xcodebuildPath, Version: version}
		snap.FieldProvenance["runtime_tools.xcodebuild.version"] = Provenance{
			SourceKind: SourceKindCommand,
			Source:     ToolXcodebuild,
			ObservedAt: observedAt,
			Confidence: "high",
			Command:    "xcodebuild -version",
		}
	}

	c.collectSimulatorRuntimes(ctx, snap, observedAt, xcrunErr == nil)
	snap.ProbeStatuses[ProbeAppleToolchain] = "ok"
}

// collectSimulatorRuntimes records installed simulator runtimes. On Intel hosts
// the universal architecture variant must be fetched explicitly, so an empty
// runtime list is a real and actionable observation rather than an error.
func (c Collector) collectSimulatorRuntimes(ctx context.Context, snap *Snapshot, observedAt time.Time, xcrunPresent bool) {
	if !xcrunPresent {
		snap.RuntimeTools[ToolSimctl] = Tool{Present: false}
		return
	}
	out, err := c.Commands.Run(ctx, ToolXcrun, "simctl", "list", "runtimes")
	if err != nil {
		snap.RuntimeTools[ToolSimctl] = Tool{Present: false}
		snap.Warnings = append(snap.Warnings, fmt.Sprintf("xcrun simctl list runtimes: %v", err))
		return
	}
	runtimes := ParseSimulatorRuntimes(string(out))
	snap.RuntimeTools[ToolSimctl] = Tool{Present: true, Version: strings.Join(runtimes, ",")}
	snap.FieldProvenance["runtime_tools.simctl"] = Provenance{
		SourceKind: SourceKindCommand,
		Source:     ToolXcrun,
		ObservedAt: observedAt,
		Confidence: "high",
		Command:    "xcrun simctl list runtimes",
	}
}

// collectAndroidToolchain records the Android SDK tooling and hardware
// acceleration state. Unlike the Apple toolchain this is not OS-terminal —
// Android tooling runs on linux, darwin, and windows alike — so the probe runs
// everywhere and reports absence rather than skipping.
func (c Collector) collectAndroidToolchain(ctx context.Context, snap *Snapshot, observedAt time.Time) {
	found := 0
	for _, tool := range []string{ToolADB, ToolEmulator, ToolSDKManager, ToolAVDManager} {
		path, err := c.Commands.LookPath(tool)
		snap.RuntimeTools[tool] = Tool{Present: err == nil, Path: path}
		if err == nil {
			found++
		}
	}

	// Hardware acceleration is what separates a usable emulator from one that
	// renders too slowly to produce video evidence, so it is probed as a
	// first-class fact rather than assumed from the presence of the emulator.
	c.collectKVM(snap, observedAt)

	switch {
	case found == 0:
		snap.ProbeStatuses[ProbeAndroidToolchain] = "tool_not_present"
	case found < toolchainParameterC:
		snap.ProbeStatuses[ProbeAndroidToolchain] = "partial"
	default:
		snap.ProbeStatuses[ProbeAndroidToolchain] = "ok"
	}
	_ = ctx
}

// collectKVM reports whether this host can hardware-accelerate an Android
// emulator. Presence of the device node is not sufficient — the invoking user
// must be able to open it read-write — so the probe records effective access.
func (c Collector) collectKVM(snap *Snapshot, observedAt time.Time) {
	if snap.OS != "linux" {
		// macOS and Windows accelerate through their own hypervisors; absence
		// of /dev/kvm there is not a finding.
		return
	}
	const devKVM = "/dev/kvm"
	writable := false
	if handle, err := os.OpenFile(devKVM, os.O_RDWR, 0); err == nil {
		writable = true
		_ = handle.Close()
	}
	snap.RuntimeTools[ToolKVM] = Tool{Present: writable, Path: pathIfPresent(devKVM, writable)}
	snap.FieldProvenance["runtime_tools.kvm"] = Provenance{
		SourceKind: SourceKindFile,
		Source:     devKVM,
		ObservedAt: observedAt,
		Confidence: "high",
		File:       devKVM,
	}
}

func pathIfPresent(path string, present bool) string {
	if present {
		return path
	}
	return ""
}

// ParseXcodeVersion extracts the marketing version from `xcodebuild -version`.
// It returns an empty string when the output carries no recognisable version so
// the caller reports an unknown version rather than inventing one.
func ParseXcodeVersion(output string) string {
	match := xcodeVersionPattern.FindStringSubmatch(output)
	if len(match) < toolchainParameterA {
		return ""
	}
	return match[1]
}

// ParseSimulatorRuntimes extracts installed simulator runtime identifiers from
// `xcrun simctl list runtimes`, normalising "iOS-26-1" to "iOS 26.1". Results
// are sorted and de-duplicated so the fact is stable across probes.
func ParseSimulatorRuntimes(output string) []string {
	matches := simulatorRuntimePattern.FindAllStringSubmatch(output, -1)
	seen := make(map[string]struct{}, len(matches))
	runtimes := make([]string, 0, len(matches))
	for _, match := range matches {
		if len(match) < toolchainParameterB {
			continue
		}
		runtime := match[1] + " " + strings.ReplaceAll(match[2], "-", ".")
		if _, duplicate := seen[runtime]; duplicate {
			continue
		}
		seen[runtime] = struct{}{}
		runtimes = append(runtimes, runtime)
	}
	sort.Strings(runtimes)
	return runtimes
}
