package hostreqkit

import (
	"context"
	"errors"
	"slices"
	"testing"
)

// Both detector names enable every matching registered interface, including
// non-PU motherboard and Terminal entries whose usage pair is unrestricted.
func TestAdversarialBNativeMixedEnrollmentIsCompleteBeforeAnyOpen(t *testing.T) {
	p := nativePolicyFixture()
	p.Devices = append(p.Devices,
		OpenRGBNativeDevice{Detector: "ASUS Aura Motherboard", Kind: "hid", StableID: "board2", HardwareID: "0b05:1bed", UsagePage: 1, Usage: 2},
		OpenRGBNativeDevice{Detector: "ASUS ROG AURA Terminal", Kind: "hid", StableID: "terminal", HardwareID: "0b05:1889", UsagePage: 3, Usage: 4})
	nodes := make([]OpenRGBNativeNode, len(p.Devices))
	for i, d := range p.Devices {
		nodes[i] = OpenRGBNativeNode{Selector: d, Path: d.StableID, Generation: "birth"}
	}
	identity := func() (string, string, error) { return p.MachineID, p.Account, nil }
	for omitted := -1; omitted < len(nodes); omitted++ {
		selected := p
		selected.Devices = slices.Clone(p.Devices)
		if omitted >= 0 {
			selected.Devices = slices.Delete(selected.Devices, omitted, omitted+1)
			// Preserve both enabled detector groups even when omitting Terminal.
			if omitted == 2 {
				selected.Devices = append(selected.Devices, OpenRGBNativeDevice{Detector: "ASUS ROG AURA Terminal", Kind: "hid", StableID: "absent-terminal", HardwareID: "0b05:1889"})
			}
		}
		opens := 0
		e, err := observeNativeOpenRGB(t.Context(), selected, identity, func(context.Context) ([]OpenRGBNativeNode, error) {
			return slices.Clone(nodes), nil
		}, func(context.Context, OpenRGBNativeNode) error { opens++; return nil })
		if omitted >= 0 {
			if err == nil || opens != 0 || e.CurrentScopeVerified {
				t.Fatalf("partial mixed selection reached access: omitted=%d opens=%d evidence=%+v err=%v", omitted, opens, e, err)
			}
		} else if err != nil || opens != len(nodes) || !e.CurrentScopeVerified {
			t.Fatalf("complete mixed selection rejected: %+v opens=%d err=%v", e, opens, err)
		}
	}
}

func TestAdversarialBNativeMetadataNeverCertifiesTokenOrDeviceAccess(t *testing.T) {
	p := nativePolicyFixture()
	identity := func() (string, string, error) { return p.MachineID, p.Account, nil }
	nodes := []OpenRGBNativeNode{{Selector: p.Devices[0], Path: "board", Generation: "birth"}}
	discover := func(context.Context) ([]OpenRGBNativeNode, error) { return nodes, nil }
	metadata, err := observeNativeOpenRGB(t.Context(), p, identity, discover, nil)
	if err != nil || metadata.CurrentScopeVerified || metadata.PolicyID == "" {
		t.Fatalf("metadata evidence falsely claims opened access: %+v %v", metadata, err)
	}
	denied := errors.New("OS denied the actual operator HID open")
	e, err := observeNativeOpenRGB(t.Context(), p, identity, discover, func(context.Context, OpenRGBNativeNode) error { return denied })
	if !errors.Is(err, denied) || e.CurrentScopeVerified || e.PolicyID != "" {
		t.Fatalf("valid account metadata hid an access denial: %+v %v", e, err)
	}
	// A new matching interface after an otherwise successful open invalidates
	// the whole proof; previously enrolled interface access is insufficient.
	e, err = observeNativeOpenRGB(t.Context(), p, identity, discover, func(context.Context, OpenRGBNativeNode) error {
		extra := nodes[0]
		extra.Selector.StableID, extra.Path = "new-board", "new-path"
		nodes = append(nodes, extra)
		return nil
	})
	if err == nil || e.CurrentScopeVerified || e.PolicyID != "" {
		t.Fatalf("new callback candidate admitted during proof: %+v %v", e, err)
	}
}
