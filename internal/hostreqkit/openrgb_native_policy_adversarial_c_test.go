package hostreqkit

import (
	"context"
	"slices"
	"testing"
)

func TestAdversarialCNativeDisabledDetectorChurnDoesNotExpandAccess(t *testing.T) {
	p := nativePolicyFixture()
	p.Devices = []OpenRGBNativeDevice{{Detector: "ASUS ROG AURA Terminal", Kind: "hid", StableID: "terminal", HardwareID: "0b05:1889", UsagePage: 0xffff, Usage: 0xffff}}
	terminal := OpenRGBNativeNode{Selector: p.Devices[0], Path: "terminal-path", Generation: "terminal-birth"}
	board := OpenRGBNativeNode{Selector: nativePolicyFixture().Devices[0], Path: "board-path", Generation: "board-birth"}
	calls, opens := 0, 0
	e, err := observeNativeOpenRGB(t.Context(), p, func() (string, string, error) { return p.MachineID, p.Account, nil }, func(context.Context) ([]OpenRGBNativeNode, error) {
		calls++
		if calls == 1 {
			return []OpenRGBNativeNode{board, terminal}, nil
		}
		// A disabled detector's replacement is irrelevant to Terminal's proof.
		board.Generation, board.Path = "replacement-birth", "replacement-path"
		return []OpenRGBNativeNode{terminal, board}, nil
	}, func(_ context.Context, n OpenRGBNativeNode) error {
		opens++
		if n != terminal {
			t.Fatalf("opened disabled detector: %+v", n)
		}
		return nil
	})
	if err != nil || opens != 1 || !e.CurrentScopeVerified || !slices.Equal(e.SelectedPaths, []string{terminal.Path}) {
		t.Fatalf("unrelated detector changed enrolled access: %+v opens=%d err=%v", e, opens, err)
	}
}

func TestAdversarialCNativePostOpenIdentityChangeCannotProduceWitness(t *testing.T) {
	for _, changeMachine := range []bool{false, true} {
		p := nativePolicyFixture()
		node := OpenRGBNativeNode{Selector: p.Devices[0], Path: "selected", Generation: "birth"}
		observations, opens := 0, 0
		e, err := observeNativeOpenRGB(t.Context(), p, func() (string, string, error) {
			observations++
			if observations == 1 {
				return p.MachineID, p.Account, nil
			}
			if changeMachine {
				return "ffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffffff", p.Account, nil
			}
			return p.MachineID, "uid:502", nil
		}, func(context.Context) ([]OpenRGBNativeNode, error) { return []OpenRGBNativeNode{node}, nil }, func(context.Context, OpenRGBNativeNode) error { opens++; return nil })
		if err == nil || opens != 1 || e.PolicyID != "" || e.CurrentScopeVerified || e.DeviceGenerationSHA256 != "" {
			t.Fatalf("identity changed after access but witness survived: %+v err=%v", e, err)
		}
	}
}

func TestAdversarialCNativeGenerationDigestIgnoresInventoryOrder(t *testing.T) {
	p := nativePolicyFixture()
	d := p.Devices[0]
	d.StableID = "second-board"
	p.Devices = append(p.Devices, d)
	nodes := []OpenRGBNativeNode{{Selector: p.Devices[0], Path: "first", Generation: "birth1"}, {Selector: d, Path: "second", Generation: "birth2"}}
	observe := func() OpenRGBNativeEvidence {
		calls := 0
		e, err := observeNativeOpenRGB(t.Context(), p, func() (string, string, error) { return p.MachineID, p.Account, nil }, func(context.Context) ([]OpenRGBNativeNode, error) {
			calls++
			copy := slices.Clone(nodes)
			if calls%2 == 0 {
				slices.Reverse(copy)
			}
			return copy, nil
		}, func(context.Context, OpenRGBNativeNode) error { return nil })
		if err != nil {
			t.Fatal(err)
		}
		return e
	}
	first := observe()
	slices.Reverse(nodes)
	second := observe()
	if first.PolicyID != second.PolicyID || first.DeviceGenerationSHA256 != second.DeviceGenerationSHA256 {
		t.Fatal("OS enumeration order changed enrollment or generation")
	}
	nodes[0].Generation = "replacement"
	third := observe()
	if third.PolicyID != first.PolicyID || third.DeviceGenerationSHA256 == first.DeviceGenerationSHA256 {
		t.Fatal("replacement did not change generation independently of enrollment")
	}
}
