package validationmatrix

import "testing"

func TestClassifyHostPreservesMobileClassesAndAddsDesktop(t *testing.T) {
	facts := HostFacts{
		OS:              "darwin",
		RuntimeTools:    map[string]HostTool{toolXcodebuild: {Present: true}, toolSimctl: {Present: true, Version: "iOS 18.0"}, toolNode: {Present: true}, toolElectron: {Present: true}},
		SessionType:     "aqua",
		DisplayAttached: true,
	}
	classes := classifyHost(facts)
	if len(classes) != 2 {
		t.Fatalf("got %d classes, want iOS and desktop: %#v", len(classes), classes)
	}
	if classes[0].Platform != "desktop" || classes[1].Platform != "ios" {
		t.Fatalf("unexpected class order/content: %#v", classes)
	}

	android := classifyHost(HostFacts{OS: "linux", RuntimeTools: map[string]HostTool{toolADB: {Present: true}, toolEmulator: {Present: true}, toolKVM: {Present: true}}})
	if len(android) != 1 || android[0].Platform != "android" || android[0].DeviceKind != "emulator" {
		t.Fatalf("Android classification changed: %#v", android)
	}
}

func TestClassifyDesktopReportsMissingCapability(t *testing.T) {
	class, ok := classifyDesktop(HostFacts{OS: "darwin", RuntimeTools: map[string]HostTool{toolNode: {Present: true}}, SessionType: "aqua", DisplayAttached: true})
	if !ok || class.Platform != "desktop" || class.Missing != toolElectron || len(class.Capabilities) != 0 {
		t.Fatalf("unexpected missing Electron result: %#v, %v", class, ok)
	}
}
