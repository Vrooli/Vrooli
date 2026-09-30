package targetmodel

import (
	"encoding/json"
	"os"
	"reflect"
	"strings"
	"testing"
	"time"

	commonv1 "github.com/vrooli/vrooli/packages/proto/gen/go/common/v1"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/proto"
)

func desktopDescriptor() SurfaceDescriptor {
	now := time.Date(2026, 9, 5, 0, 0, 0, 0, time.UTC)
	return SurfaceDescriptor{
		Ref:  SurfaceRef{Target: TargetRef{OwnerScenario: "vrooli-bridge", ResourceID: "machine-1", HostNodeID: "node-1"}, OwnerScenario: "device-control", SurfaceID: "desktop-1"},
		Kind: SurfaceDesktop, DisplayLabel: "Office PC", DesktopSessionID: "login-2", DisplayIDs: []string{"monitor-1"},
		ProtocolVersions: []string{"vrooli.desktop.v1"},
		Capabilities:     []CapabilityFact{{Capability: "desktop.observe", State: CapabilityReady, EvidenceID: "probe-1", ObservedAt: now, ExpiresAt: now.Add(time.Minute)}},
	}
}

func TestSurfaceDescriptorRoundTripPreservesOwnerTopologyAndSession(t *testing.T) {
	original := desktopDescriptor()
	data, err := json.Marshal(original)
	if err != nil {
		t.Fatal(err)
	}
	decoded, err := DecodeSurfaceDescriptor(data)
	if err != nil || !reflect.DeepEqual(original, decoded) {
		t.Fatalf("round trip: %+v, %v", decoded, err)
	}
	// Two owners can use the same resource ID without naming the same target.
	other := original.Ref.Target
	other.OwnerScenario = "device-control"
	if other == decoded.Ref.Target {
		t.Fatal("owner namespaces collapsed")
	}
	lease := SessionRef{Surface: decoded.Ref, SessionID: "lease-1", DesktopSessionID: decoded.DesktopSessionID}
	if err := lease.Validate(); err != nil {
		t.Fatal(err)
	}
}

func TestSurfaceDescriptorRejectsUntrustedAuthority(t *testing.T) {
	data, _ := json.Marshal(desktopDescriptor())
	for name, input := range map[string][]byte{
		"endpoint":     []byte(strings.Replace(string(data), `"display_label":`, `"endpoint":"https://attacker.example","display_label":`, 1)),
		"token":        []byte(strings.Replace(string(data), `"surface_id":`, `"token":"private-secret","surface_id":`, 1)),
		"url identity": []byte(strings.Replace(string(data), `"machine-1"`, `"https://attacker.example"`, 1)),
		"trailing":     append(append([]byte{}, data...), []byte(` {}`)...),
		"oversized":    []byte(strings.Repeat(" ", 64*1024+1)),
	} {
		t.Run(name, func(t *testing.T) {
			if _, err := DecodeSurfaceDescriptor(input); err == nil {
				t.Fatal("accepted untrusted authority or malformed payload")
			} else if strings.Contains(err.Error(), "private-secret") {
				t.Fatal("decoder error leaked credentials")
			}
		})
	}
}

func TestSurfaceDescriptorRejectsAmbiguousOrUnprovenFacts(t *testing.T) {
	for name, mutate := range map[string]func(*SurfaceDescriptor){
		"missing owner":         func(d *SurfaceDescriptor) { d.Ref.Target.OwnerScenario = "" },
		"missing target":        func(d *SurfaceDescriptor) { d.Ref.Target.ResourceID = "" },
		"missing surface":       func(d *SurfaceDescriptor) { d.Ref.SurfaceID = "" },
		"missing desktop login": func(d *SurfaceDescriptor) { d.DesktopSessionID = "" },
		"unknown kind":          func(d *SurfaceDescriptor) { d.Kind = "native-ipc" },
		"duplicate fact":        func(d *SurfaceDescriptor) { d.Capabilities = append(d.Capabilities, d.Capabilities[0]) },
		"duplicate display":     func(d *SurfaceDescriptor) { d.DisplayIDs = []string{"monitor-1", "monitor-1"} },
		"no evidence":           func(d *SurfaceDescriptor) { d.Capabilities[0].EvidenceID = "" },
		"unknown state":         func(d *SurfaceDescriptor) { d.Capabilities[0].State = "maybe" },
		"denied without reason": func(d *SurfaceDescriptor) { d.Capabilities[0].State = CapabilityDenied },
		"no expiry":             func(d *SurfaceDescriptor) { d.Capabilities[0].ExpiresAt = time.Time{} },
		"no protocol":           func(d *SurfaceDescriptor) { d.ProtocolVersions = nil },
	} {
		t.Run(name, func(t *testing.T) {
			d := desktopDescriptor()
			mutate(&d)
			if err := d.Validate(); err == nil {
				t.Fatal("accepted invalid descriptor")
			}
		})
	}
}

func TestCapabilityFreshnessAndLegacyConversion(t *testing.T) {
	fact := desktopDescriptor().Capabilities[0]
	for _, now := range []time.Time{fact.ObservedAt.Add(-time.Nanosecond), fact.ExpiresAt, fact.ExpiresAt.Add(time.Second)} {
		if fact.EffectiveState(now) != CapabilityUnknown {
			t.Fatal("stale or future observation authorized readiness")
		}
	}
	for _, state := range []CapabilityState{CapabilityReady, CapabilityMissing, CapabilityUnsupported, CapabilityUnknown, CapabilityDenied} {
		fact.State, fact.ReasonCode = state, "probe_result"
		if got := fact.EffectiveState(fact.ObservedAt); got != state {
			t.Fatalf("lost explicit %q fact: %q", state, got)
		}
		if state == CapabilityDenied || state == CapabilityUnsupported {
			if state.LegacyReadiness() != ReadinessUnknown {
				t.Fatal("legacy projection reinterpreted denial or unsupported as absence")
			}
		}
	}
}

func TestProtocolNegotiationSeparatesTerminalAndDesktop(t *testing.T) {
	if _, err := NegotiateProtocol([]string{"vrooli.desktop.v1"}, []string{"vrooli.terminal.v1"}); err == nil {
		t.Fatal("terminal protocol accepted for desktop")
	}
	got, err := NegotiateProtocol([]string{"vrooli.desktop.v2", "vrooli.desktop.v1"}, []string{"vrooli.desktop.v1", "vrooli.desktop.v2"})
	if err != nil || got != "vrooli.desktop.v2" {
		t.Fatalf("consumer preference lost: %q %v", got, err)
	}
	if _, err := NegotiateProtocol([]string{"vrooli.desktop.v1"}, []string{"vrooli.desktop.v1", "https://attacker.example"}); err == nil {
		t.Fatal("accepted malformed offer after compatible version")
	}
}

func TestSurfaceProtoRoundTripAndMalformedReferences(t *testing.T) {
	original := desktopDescriptor()
	wire, err := original.Proto()
	if err != nil {
		t.Fatal(err)
	}
	data, err := (protojson.MarshalOptions{UseProtoNames: true}).Marshal(wire)
	if err != nil {
		t.Fatal(err)
	}
	var decoded commonv1.SurfaceDescriptor
	fixture, err := os.ReadFile("testdata/surface.json")
	if err != nil {
		t.Fatal(err)
	}
	if err := protojson.Unmarshal(fixture, &decoded); err != nil || !proto.Equal(wire, &decoded) {
		t.Fatalf("cross-language fixture diverged: %v", err)
	}
	if err := protojson.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	result, err := SurfaceDescriptorFromProto(&decoded)
	if err != nil || !reflect.DeepEqual(original, result) {
		t.Fatalf("wire round trip changed descriptor: %+v %v", result, err)
	}
	for name, mutate := range map[string]func(*commonv1.SurfaceDescriptor){
		"nil target":        func(d *commonv1.SurfaceDescriptor) { d.Ref.Target = nil },
		"nil fact":          func(d *commonv1.SurfaceDescriptor) { d.Capabilities[0] = nil },
		"nil timestamp":     func(d *commonv1.SurfaceDescriptor) { d.Capabilities[0].ObservedAt = nil },
		"unknown kind":      func(d *commonv1.SurfaceDescriptor) { d.Kind = 999 },
		"unknown state":     func(d *commonv1.SurfaceDescriptor) { d.Capabilities[0].State = 999 },
		"invalid timestamp": func(d *commonv1.SurfaceDescriptor) { d.Capabilities[0].ObservedAt.Nanos = -1 },
	} {
		t.Run(name, func(t *testing.T) {
			invalid := proto.Clone(wire).(*commonv1.SurfaceDescriptor)
			mutate(invalid)
			if _, err := SurfaceDescriptorFromProto(invalid); err == nil {
				t.Fatal("accepted invalid wire contract")
			}
		})
	}
}
