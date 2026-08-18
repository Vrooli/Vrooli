package validationmatrix

import (
	"github.com/vrooli/api-core/targetmodel"
	deliveryramp "github.com/vrooli/vrooli/packages/delivery-ramp-go"
)

// Platform capability names a delivery ramp understands for Apple targets.
// The generic vocabulary lives in targetmodel; these extend it for iOS without
// widening the shared model for a single ramp's benefit.
const (
	CapabilityXcode        = "xcodebuild"
	CapabilitySimctl       = "simctl"
	CapabilityIOSSimulator = "ios-simulator"
)

// Probed runtime-tool names. These mirror the control plane's host-inventory
// output, which the spine consumes as JSON over dispatch. They are duplicated
// deliberately rather than imported: the coupling is a wire contract with the
// probe payload, and importing the control-plane package here would create a
// module edge from a shared package into the root module's internal tree.
const (
	toolXcodebuild = "xcodebuild"
	toolSimctl     = "simctl"
	toolADB        = "adb"
	toolEmulator   = "emulator"
	toolKVM        = "kvm"
)

// platformClass is the coarse platform a probed node can serve. A node may
// satisfy more than one; the classifier reports every class it can prove.
type platformClass struct {
	Platform     string
	DeviceKind   string
	Capabilities []string
	Reason       string
	Missing      string
	NextAction   string
}

// classifyHost derives platform capability from probed host facts.
//
// This replaces reading node.Capabilities, which holds allowlisted dispatch
// verbs ("host inventory*", "setup*") rather than platform names. Those two
// vocabularies never intersect, so capability derived from the registry field
// was empty for every node regardless of what the node could actually do.
func classifyHost(facts HostFacts) []platformClass {
	classes := make([]platformClass, 0, 2)
	if apple, ok := classifyApple(facts); ok {
		classes = append(classes, apple)
	}
	if android, ok := classifyAndroid(facts); ok {
		classes = append(classes, android)
	}
	return classes
}

// classifyApple reports an iOS class only when the node can actually build and
// boot a simulator. A darwin host without a usable toolchain is not an iOS
// target; saying otherwise would let a release gate select a node that cannot
// produce Apple evidence.
func classifyApple(facts HostFacts) (platformClass, bool) {
	if facts.OS != "darwin" {
		return platformClass{}, false
	}
	class := platformClass{
		Platform:   "ios",
		DeviceKind: "emulator",
		Reason:     targetmodel.ReasonBridgeAuthorizedIOS,
		NextAction: "install Xcode and an iOS simulator runtime on the node, then probe again",
	}
	xcode, xcodeProbed := facts.Tool(toolXcodebuild)
	if !xcodeProbed {
		class.Missing = "host toolchain probe"
		class.NextAction = "update the node's control plane so it reports Apple toolchain facts, then probe again"
		return class, true
	}
	if !xcode.Present {
		class.Missing = CapabilityXcode
		return class, true
	}
	class.Capabilities = append(class.Capabilities, CapabilityXcode)

	simctl, simctlProbed := facts.Tool(toolSimctl)
	switch {
	case !simctlProbed || !simctl.Present:
		class.Missing = CapabilitySimctl
		return class, true
	case simctl.Version == "":
		// simctl works but no runtime is installed. On Intel hosts the
		// universal runtime variant must be fetched explicitly, so this is a
		// real and actionable state rather than a broken toolchain.
		class.Capabilities = append(class.Capabilities, CapabilitySimctl)
		class.Missing = CapabilityIOSSimulator
		class.NextAction = "install an iOS simulator runtime (on Intel hosts fetch the universal architecture variant), then probe again"
		return class, true
	}
	class.Capabilities = append(class.Capabilities, CapabilitySimctl, CapabilityIOSSimulator)
	return class, true
}

// classifyAndroid reports an Android class when the node has the SDK tooling to
// drive a device or emulator. Unlike Apple this is not OS-terminal, so any
// probed host may qualify.
func classifyAndroid(facts HostFacts) (platformClass, bool) {
	adb, adbProbed := facts.Tool(toolADB)
	if !adbProbed {
		return platformClass{}, false
	}
	class := platformClass{
		Platform:   "android",
		DeviceKind: "physical",
		Reason:     targetmodel.ReasonBridgeAuthorizedAndroid,
		NextAction: "install the Android SDK platform-tools on the node, then probe again",
	}
	if !adb.Present {
		class.Missing = string(deliveryramp.CapabilityAndroidSDK)
		return class, true
	}
	class.Capabilities = append(class.Capabilities, deliveryramp.CapabilityDeviceControl, deliveryramp.CapabilityAndroidSDK)

	// An emulator without hardware acceleration renders too slowly to produce
	// usable video evidence, so the emulator class requires both.
	if facts.HasTool(toolEmulator) && (facts.OS != "linux" || facts.HasTool(toolKVM)) {
		class.DeviceKind = "emulator"
		class.Capabilities = append(class.Capabilities, deliveryramp.CapabilityAndroidEmulator)
	}
	return class, true
}

// capabilityClassFor selects the class matching a requested platform, so one
// probed node can serve an iOS inventory and an Android inventory differently.
func capabilityClassFor(classes []platformClass, platform string) (platformClass, bool) {
	for _, class := range classes {
		if class.Platform == platform {
			return class, true
		}
	}
	return platformClass{}, false
}
