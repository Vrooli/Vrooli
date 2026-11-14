package hostreqkit

import (
	"context"
	"errors"
	"slices"
	"strings"
	"testing"
)

func nativePolicyFixture() OpenRGBNativePolicy {
	return OpenRGBNativePolicy{Platform: "darwin", MachineID: strings.Repeat("a", 64), Account: "uid:501", ArtifactSHA256: strings.Repeat("b", 64), DetectorSHA256: openRGBQualifiedDetectorCatalogSHA256, Devices: []OpenRGBNativeDevice{{Detector: "ASUS Aura Motherboard", Kind: "hid", StableID: "IOService:/usb/port8/interface0", HardwareID: "0b05:19af", UsagePage: 0xff72, Usage: 0xa1}}}
}

func TestNativeHIDPolicyBindsOnlySourceQualifiedAccountAndSelectors(t *testing.T) {
	p := nativePolicyFixture()
	id, err := p.ID()
	if err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name   string
		change func(*OpenRGBNativePolicy)
	}{
		{"root", func(p *OpenRGBNativePolicy) { p.Account = "uid:0" }},
		{"system", func(p *OpenRGBNativePolicy) { p.Platform = "windows"; p.Account = "sid:S-1-5-18" }},
		{"virtual service", func(p *OpenRGBNativePolicy) { p.Platform = "windows"; p.Account = "sid:S-1-5-80-1-2-3-4-5" }},
		{"SMBus", func(p *OpenRGBNativePolicy) { p.Devices[0].Kind = "i2c" }},
		{"usage mismatch", func(p *OpenRGBNativePolicy) { p.Devices[0].Usage = 1 }},
		{"unknown controller", func(p *OpenRGBNativePolicy) { p.Devices[0].HardwareID = "0b05:ffff" }},
		{"unqualified catalog", func(p *OpenRGBNativePolicy) { p.DetectorSHA256 = strings.Repeat("c", 64) }},
		{"duplicate attachment", func(p *OpenRGBNativePolicy) { p.Devices = append(p.Devices, p.Devices[0]) }},
	} {
		t.Run(tt.name, func(t *testing.T) {
			copy := p
			copy.Devices = slices.Clone(p.Devices)
			tt.change(&copy)
			if err := ValidateOpenRGBNativeDetectorScope(copy); err == nil {
				t.Fatal("unsafe policy admitted")
			}
		})
	}
	p.Devices = append(p.Devices, OpenRGBNativeDevice{Detector: "ASUS Aura Motherboard", Kind: "hid", StableID: "IOService:/usb/port9/interface0", HardwareID: "0b05:18f3"})
	first, err := p.ID()
	if err != nil {
		t.Fatal(err)
	}
	slices.Reverse(p.Devices)
	second, err := p.ID()
	if err != nil || first != second || first == id {
		t.Fatal("identity must ignore ordering and bind every selector")
	}
}

func TestNativeHIDPolicyChecksScopeBeforeOpeningAnyTransport(t *testing.T) {
	p := nativePolicyFixture()
	identity := func() (string, string, error) { return p.MachineID, p.Account, nil }
	node := OpenRGBNativeNode{Selector: p.Devices[0], Path: "native-current-handle", Generation: "registry-birth-10"}
	for _, tt := range []struct {
		name  string
		nodes []OpenRGBNativeNode
	}{
		{"absent", nil},
		{"duplicate", []OpenRGBNativeNode{node, node}},
		{"unknown generation", []OpenRGBNativeNode{{Selector: node.Selector, Path: node.Path}}},
		{"extra callback candidate", []OpenRGBNativeNode{node, {Selector: OpenRGBNativeDevice{Detector: "ASUS Aura Motherboard", Kind: "hid", StableID: "port9", HardwareID: "0b05:18f3"}, Path: "other", Generation: "new"}}},
	} {
		t.Run(tt.name, func(t *testing.T) {
			opened := 0
			_, err := observeNativeOpenRGB(t.Context(), p, identity, func(context.Context) ([]OpenRGBNativeNode, error) { return tt.nodes, nil }, func(context.Context, OpenRGBNativeNode) error { opened++; return nil })
			if err == nil || opened != 0 {
				t.Fatalf("unsafe scope reached transport: %v opens=%d", err, opened)
			}
		})
	}
}

func TestNativeHIDPolicyGenerationAndAccessRemainSeparateFromStableIdentity(t *testing.T) {
	p := nativePolicyFixture()
	id, _ := p.ID()
	identity := func() (string, string, error) { return p.MachineID, p.Account, nil }
	node := OpenRGBNativeNode{Selector: p.Devices[0], Path: "same-path", Generation: "birth1"}
	discover := func(context.Context) ([]OpenRGBNativeNode, error) { return []OpenRGBNativeNode{node}, nil }
	calls := 0
	access := func(_ context.Context, n OpenRGBNativeNode) error {
		calls++
		if n != node {
			t.Fatal("different handle requested")
		}
		return nil
	}
	first, err := observeNativeOpenRGB(t.Context(), p, identity, discover, access)
	if err != nil || !first.CurrentScopeVerified || calls != 1 {
		t.Fatalf("%+v %v", first, err)
	}
	node.Generation = "birth2"
	second, err := observeNativeOpenRGB(t.Context(), p, identity, discover, access)
	if err != nil || second.DeviceGenerationSHA256 == first.DeviceGenerationSHA256 || second.PolicyID != id {
		t.Fatal("replug must change generation independently of persistent enrollment", err)
	}
	accessErr := errors.New("denied")
	if _, err = observeNativeOpenRGB(t.Context(), p, identity, discover, func(context.Context, OpenRGBNativeNode) error { return accessErr }); !errors.Is(err, accessErr) {
		t.Fatal("access denial lost", err)
	}
	_, err = observeNativeOpenRGB(t.Context(), p, identity, discover, func(context.Context, OpenRGBNativeNode) error { node.Generation = "birth3"; return nil })
	if err == nil {
		t.Fatal("replacement while opening admitted")
	}
}

func TestNativeHIDPolicyRejectsWrongAccountAndCancelledProofBeforeProbe(t *testing.T) {
	p := nativePolicyFixture()
	probed := false
	discover := func(context.Context) ([]OpenRGBNativeNode, error) { probed = true; return nil, nil }
	_, err := observeNativeOpenRGB(t.Context(), p, func() (string, string, error) { return p.MachineID, "uid:502", nil }, discover, nil)
	if err == nil || probed {
		t.Fatal("wrong account reached HID discovery")
	}
	ctx, cancel := context.WithCancel(t.Context())
	cancel()
	_, err = observeNativeOpenRGB(ctx, p, func() (string, string, error) { t.Fatal("cancelled proof read identity"); return "", "", nil }, discover, nil)
	if !errors.Is(err, context.Canceled) || probed {
		t.Fatal("cancelled proof reached discovery")
	}
}

func TestNativeHIDRetainedIdentitySurvivesSourceScopeEvolution(t *testing.T) {
	p := nativePolicyFixture()
	p.DetectorSHA256 = strings.Repeat("c", 64)
	p.Devices[0].HardwareID = "1234:5678"
	p.Devices[0].Detector = "Retired exact detector"
	if _, err := p.ID(); err != nil {
		t.Fatal("structural retained identity must remain readable for exact revocation", err)
	}
	_, err := observeNativeOpenRGB(t.Context(), p, func() (string, string, error) {
		t.Fatal("unqualified policy reached identity probe")
		return "", "", nil
	}, func(context.Context) ([]OpenRGBNativeNode, error) {
		t.Fatal("unqualified policy reached HID inventory")
		return nil, nil
	}, nil)
	if err == nil {
		t.Fatal("retained structural identity incorrectly qualified a launch")
	}
}

func TestNativeOperatorAccountShapesIncludeEntraAndExcludeServiceAuthority(t *testing.T) {
	for _, account := range []string{"sid:S-1-5-21-1-2-3-1001", "sid:S-1-12-1-4141952679-1282074057-627758481-2916039507"} {
		if err := ValidateNativeOpenRGBAccount("windows", account); err != nil {
			t.Fatalf("ordinary operator account refused: %s %v", account, err)
		}
	}
	for _, account := range []string{"sid:S-1-5-18", "sid:S-1-5-32-544", "sid:S-1-5-80-1-2-3-4-5", "sid:S-1-15-3-1024-1-2-3-4-5-6-7-8", "sid:S-1-12-1-1-2-3", "sid:S-1-12-1-01-2-3-4", "sid:S-1-12-1-4294967296-2-3-4"} {
		if err := ValidateNativeOpenRGBAccount("windows", account); err == nil {
			t.Fatalf("nonoperator or malformed account admitted: %s", account)
		}
	}
	if err := ValidateNativeOpenRGBAccount("linux", "sid:S-1-5-21-1-2-3-1001"); err == nil {
		t.Fatal("unknown native platform admitted")
	}
}

func TestNativeExternalTerminalEnrollsAllEnabledDetectorAttachments(t *testing.T) {
	p := nativePolicyFixture()
	terminal := OpenRGBNativeDevice{Detector: "ASUS ROG AURA Terminal", Kind: "hid", StableID: "IOService:/external/terminal", HardwareID: "0b05:1889", UsagePage: 0xff00, Usage: 1}
	p.Devices = []OpenRGBNativeDevice{terminal}
	if err := ValidateOpenRGBNativeDetectorScope(p); err != nil {
		t.Fatal(err)
	}
	terminalNode := OpenRGBNativeNode{Selector: terminal, Path: "terminal", Generation: "1"}
	motherboard := OpenRGBNativeNode{Selector: nativePolicyFixture().Devices[0], Path: "motherboard", Generation: "2"}
	identity := func() (string, string, error) { return p.MachineID, p.Account, nil }
	nodes := []OpenRGBNativeNode{terminalNode, motherboard}
	opens := 0
	observe := func() (OpenRGBNativeEvidence, error) {
		return observeNativeOpenRGB(t.Context(), p, identity, func(context.Context) ([]OpenRGBNativeNode, error) { return nodes, nil }, func(_ context.Context, n OpenRGBNativeNode) error {
			if n.Selector.Detector != terminal.Detector {
				t.Fatal("disabled detector opened")
			}
			opens++
			return nil
		})
	}
	evidence, err := observe()
	if err != nil || opens != 1 || len(evidence.SelectedPaths) != 1 {
		t.Fatalf("external-only detector scope failed: %+v %v opens=%d", evidence, err, opens)
	}
	extra := terminalNode
	extra.Selector.StableID = "IOService:/external/terminal2"
	extra.Path = "terminal2"
	nodes = append(nodes, extra)
	opens = 0
	if _, err = observe(); err == nil || opens != 0 {
		t.Fatal("unenrolled matching external attachment reached HID", err, opens)
	}
	for _, hardware := range []string{"0b05:1888", "1234:1889", "0b05:19af"} {
		copy := terminal
		copy.HardwareID = hardware
		if qualifiedNativeHID(copy) {
			t.Fatal("unregistered external identity qualified", hardware)
		}
	}
	terminal.Detector = "ASUS Aura Motherboard"
	if qualifiedNativeHID(terminal) {
		t.Fatal("Terminal mislabeled motherboard")
	}
}
