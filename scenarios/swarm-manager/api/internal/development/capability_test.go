package development

import (
	"errors"
	"testing"
)

func TestSelectLaneUsesOnlyQualifiedOwnerCapabilities(t *testing.T) {
	cases := []struct {
		name       string
		preference string
		wantLane   string
		wantErr    bool
	}{
		{name: "auto falls back from unqualified native", preference: "auto", wantLane: LaneFallback},
		{name: "fallback required", preference: "fallback-required", wantLane: LaneFallback},
		{name: "native required refuses without parity", preference: "native-required", wantErr: true},
	}
	capabilities := []LaneCapability{{Lane: LaneNative, Revision: "native/preview", Binding: true}, QualifiedFallbackCapability()}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			selection, err := SelectLane(tc.preference, CapabilityRequirements{Binding: true, Continuation: true, Metering: true, Cancellation: true, Containment: true}, capabilities)
			if tc.wantErr {
				if !errors.Is(err, ErrDenied) {
					t.Fatalf("error = %v, want ErrDenied", err)
				}
				return
			}
			if err != nil || selection.Lane != tc.wantLane || selection.Revision == "" || selection.Reason == "" {
				t.Fatalf("selection=%+v err=%v", selection, err)
			}
		})
	}
}

func TestSelectLaneRejectsUnknownPreference(t *testing.T) {
	if _, err := SelectLane("provider-name", CapabilityRequirements{}, []LaneCapability{QualifiedFallbackCapability()}); !errors.Is(err, ErrInvalid) {
		t.Fatalf("unknown preference error = %v", err)
	}
}
