package hostreqspec

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestDeclarationMinVersionRoundTrips(t *testing.T) {
	raw := []byte(`{"name":"node","required":true,"min_version":"20.0.0","reason":"Builds the TypeScript sidecar."}`)
	var declaration Declaration
	if err := json.Unmarshal(raw, &declaration); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if declaration.MinVersion != "20.0.0" {
		t.Fatalf("MinVersion = %q, want 20.0.0", declaration.MinVersion)
	}
	encoded, err := json.Marshal(declaration)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(encoded), `"min_version":"20.0.0"`) {
		t.Fatalf("round-trip output = %s", encoded)
	}
}

func TestNormalizePlatform(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "legacy darwin", input: "darwin", want: "macos"},
		{name: "trim and lowercase", input: "  DaRwIn  ", want: "macos"},
		{name: "linux unchanged", input: "linux", want: "linux"},
		{name: "windows unchanged", input: "windows", want: "windows"},
		{name: "unknown unchanged", input: "freebsd", want: "freebsd"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := NormalizePlatform(test.input); got != test.want {
				t.Fatalf("NormalizePlatform(%q) = %q, want %q", test.input, got, test.want)
			}
		})
	}
}

func TestCapabilityRequirementPlatformFacts(t *testing.T) {
	wayland := true
	tests := []struct {
		name        string
		requirement CapabilityRequirement
		facts       CapabilityFacts
		wantOK      bool
		wantReason  string
	}{
		{name: "init system", requirement: CapabilityRequirement{InitSystem: "systemd"}, facts: CapabilityFacts{InitSystem: "launchd"}, wantReason: `requires init system "systemd"`},
		{name: "session type", requirement: CapabilityRequirement{SessionType: "x11"}, facts: CapabilityFacts{SessionType: "wayland"}, wantReason: `requires session type "x11"`},
		{name: "display manager", requirement: CapabilityRequirement{DisplayManager: "gdm"}, facts: CapabilityFacts{DisplayManager: "sddm"}, wantReason: `requires display manager "gdm"`},
		{name: "wayland", requirement: CapabilityRequirement{WaylandAttainable: &wayland}, facts: CapabilityFacts{WaylandAttainable: false}, wantReason: "requires Wayland attainable=true"},
		{name: "all satisfied", requirement: CapabilityRequirement{InitSystem: "systemd", SessionType: "x11", DisplayManager: "gdm", WaylandAttainable: &wayland}, facts: CapabilityFacts{InitSystem: "systemd", SessionType: "x11", DisplayManager: "gdm", WaylandAttainable: true}, wantOK: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			ok, reason := test.requirement.Evaluate(test.facts)
			if ok != test.wantOK {
				t.Fatalf("Evaluate() = %t, want %t (%s)", ok, test.wantOK, reason)
			}
			if test.wantReason != "" && !strings.Contains(reason, test.wantReason) {
				t.Fatalf("reason = %q, want substring %q", reason, test.wantReason)
			}
		})
	}
}
