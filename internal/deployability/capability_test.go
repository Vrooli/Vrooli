package deployability

import (
	"strings"
	"testing"
)

func TestResolveCapabilityUsesDeclaredPeersAndMechanisms(t *testing.T) {
	implementations := []CapabilityImplementation{
		{Name: "linux-primary", Capability: "credential-storage", Role: "primary", Platforms: map[HostOS]PlatformDeclaration{
			HostOSLinux: {Status: string(StatusSupported)},
			HostOSMacOS: {Status: string(StatusUnsupported), Mechanism: "native-keychain adapter"},
		}},
		{Name: "mac-peer", Capability: "credential-storage", Role: "peer", Platforms: map[HostOS]PlatformDeclaration{
			HostOSMacOS: {Status: string(StatusSupported)},
		}},
	}

	mac := ResolveCapability(implementations, "credential-storage", HostOSMacOS)
	if mac.Status != CapabilityImplemented || mac.Implementer != "mac-peer" {
		t.Fatalf("expected macOS peer, got %+v", mac)
	}
	if mac.Qualification != QualificationQualified {
		t.Fatalf("expected a qualified macOS peer, got %+v", mac)
	}

	windows := ResolveCapability(implementations, "credential-storage", HostOSWindows)
	if windows.Status != CapabilityPeerless {
		t.Fatalf("expected peerless Windows capability, got %+v", windows)
	}

	unwired := ResolveCapability([]CapabilityImplementation{{
		Name: "linux-only", Capability: "credential-storage", Role: "primary",
		Platforms: map[HostOS]PlatformDeclaration{HostOSWindows: {Status: string(StatusUnsupported), Mechanism: "Windows Credential Manager adapter"}},
	}}, "credential-storage", HostOSWindows)
	if unwired.Status != CapabilityUnwired || unwired.Mechanism == "" {
		t.Fatalf("expected named unwired mechanism, got %+v", unwired)
	}
	if !strings.Contains(unwired.Reason, "no implementation is declared") {
		t.Fatalf("unwired reason must say no implementation is declared, got %q", unwired.Reason)
	}
	if strings.Contains(unwired.Reason, "wired for this host OS") {
		t.Fatalf("unwired reason must not claim a wiring gap it cannot know, got %q", unwired.Reason)
	}
}

func TestResolveCapabilityGivesEveryVocabularyMemberAnExplicitOutcome(t *testing.T) {
	cases := []struct {
		name          string
		declaration   PlatformDeclaration
		status        CapabilityResolutionStatus
		qualification Qualification
	}{
		{
			name:          "supported is implemented and qualified",
			declaration:   PlatformDeclaration{Status: string(StatusSupported)},
			status:        CapabilityImplemented,
			qualification: QualificationQualified,
		},
		{
			// This is the defect: system-monitor declares build-verified macOS
			// collectors with real darwin code behind them, and the ledger used
			// to call them unwired.
			name:          "build-verified is implemented, not unwired",
			declaration:   PlatformDeclaration{Status: string(StatusBuildVerified), Mechanism: "host_statistics"},
			status:        CapabilityImplemented,
			qualification: QualificationBuildVerified,
		},
		{
			name:          "experimental is implemented but unqualified",
			declaration:   PlatformDeclaration{Status: string(StatusExperimental), Mechanism: "systemd-analyze"},
			status:        CapabilityImplemented,
			qualification: QualificationUnqualified,
		},
		{
			name:          "unqualified is implemented but unqualified",
			declaration:   PlatformDeclaration{Status: string(StatusUnqualified), Mechanism: "host-integrity probe"},
			status:        CapabilityImplemented,
			qualification: QualificationUnqualified,
		},
		{
			name:          "partial is degraded",
			declaration:   PlatformDeclaration{Status: string(StatusPartial)},
			status:        CapabilityDegraded,
			qualification: QualificationDegraded,
		},
		{
			name:          "unsupported without a mechanism is ineligible",
			declaration:   PlatformDeclaration{Status: string(StatusUnsupported)},
			status:        CapabilityIneligible,
			qualification: QualificationIneligible,
		},
		{
			name:          "unsupported with a named mechanism is unwired",
			declaration:   PlatformDeclaration{Status: string(StatusUnsupported), Mechanism: "Windows Credential Manager adapter"},
			status:        CapabilityUnwired,
			qualification: QualificationUndeclared,
		},
	}
	for _, testCase := range cases {
		t.Run(testCase.name, func(t *testing.T) {
			resolution := ResolveCapability([]CapabilityImplementation{{
				Name: "collector", Capability: "host-metrics", Role: "primary",
				Platforms: map[HostOS]PlatformDeclaration{HostOSMacOS: testCase.declaration},
			}}, "host-metrics", HostOSMacOS)
			if resolution.Status != testCase.status {
				t.Fatalf("expected status %q, got %+v", testCase.status, resolution)
			}
			if resolution.Qualification != testCase.qualification {
				t.Fatalf("expected qualification %q, got %+v", testCase.qualification, resolution)
			}
			if resolution.Status == CapabilityUnwired {
				return
			}
			if resolution.Status.HasImplementation() != (testCase.qualification.AtLeast(QualificationDegraded)) {
				t.Fatalf("HasImplementation disagrees with the ladder for %+v", resolution)
			}
			if strings.TrimSpace(resolution.Reason) == "" {
				t.Fatalf("resolution carries no reason: %+v", resolution)
			}
		})
	}
}

func TestResolveCapabilityRejectsUnknownStatusTokens(t *testing.T) {
	resolution := ResolveCapability([]CapabilityImplementation{{
		Name: "collector", Capability: "host-metrics", Role: "primary",
		Platforms: map[HostOS]PlatformDeclaration{HostOSMacOS: {Status: "bundled", Mechanism: "electron-app"}},
	}}, "host-metrics", HostOSMacOS)
	if resolution.Status != CapabilityStatusInvalid {
		t.Fatalf("expected an invalid-status verdict, got %+v", resolution)
	}
	if resolution.Status.HasImplementation() {
		t.Fatalf("an invalid status must never claim an implementation: %+v", resolution)
	}
	if !strings.Contains(resolution.Reason, "bundled") {
		t.Fatalf("verdict must name the offending token, got %q", resolution.Reason)
	}
}

func TestResolveCapabilityPrefersTheBestProvenImplementer(t *testing.T) {
	resolution := ResolveCapability([]CapabilityImplementation{
		{Name: "unproven-primary", Capability: "host-metrics", Role: "primary", Platforms: map[HostOS]PlatformDeclaration{
			HostOSLinux: {Status: string(StatusExperimental)},
		}},
		{Name: "proven-peer", Capability: "host-metrics", Role: "peer", Platforms: map[HostOS]PlatformDeclaration{
			HostOSLinux: {Status: string(StatusSupported)},
		}},
	}, "host-metrics", HostOSLinux)
	if resolution.Implementer != "proven-peer" || resolution.Qualification != QualificationQualified {
		t.Fatalf("expected the qualified peer to win over an experimental primary, got %+v", resolution)
	}
}

func TestResolveCapabilityRequiresEveryControlAndReportsAbsentDeclarers(t *testing.T) {
	resolution := ResolveCapability([]CapabilityImplementation{
		{Name: "provider", Capability: "host-metrics", Role: "primary", Platforms: map[HostOS]PlatformDeclaration{HostOSWindows: {Status: string(StatusSupported)}}},
		{Name: "applied-control", Capability: "host-metrics", Role: "control", Platforms: map[HostOS]PlatformDeclaration{HostOSWindows: {Status: string(StatusSupported)}}},
		{Name: "missing-control", Capability: "host-metrics", Role: "control", Platforms: map[HostOS]PlatformDeclaration{HostOSLinux: {Status: string(StatusSupported)}, HostOSWindows: {Status: string(StatusUnsupported)}}},
	}, "host-metrics", HostOSWindows)
	if resolution.Status != CapabilityControlsIncomplete {
		t.Fatalf("expected incomplete controls, got %+v", resolution)
	}
	if len(resolution.Controls) != 1 || resolution.Controls[0] != "applied-control" {
		t.Fatalf("unexpected resolved controls: %+v", resolution.Controls)
	}
	if len(resolution.Absent) != 1 || resolution.Absent[0] != "missing-control" {
		t.Fatalf("unexpected absent declarers: %+v", resolution.Absent)
	}
	if len(resolution.AbsentControls) != 1 || resolution.AbsentControls[0] != "missing-control" || len(resolution.AbsentProviders) != 0 {
		t.Fatalf("unexpected role-separated absence: controls=%v providers=%v", resolution.AbsentControls, resolution.AbsentProviders)
	}
	if strings.Contains(resolution.Reason, "provider") && !strings.Contains(resolution.Reason, "provider \"provider\"") {
		t.Fatalf("required-control reason names an absent provider: %q", resolution.Reason)
	}
}

func TestResolveCapabilityReportsProviderAlternativesAbsentInWinnerBranch(t *testing.T) {
	resolution := ResolveCapability([]CapabilityImplementation{
		{Name: "windows-provider", Capability: "host-metrics", Role: "primary", Platforms: map[HostOS]PlatformDeclaration{HostOSWindows: {Status: string(StatusSupported)}}},
		{Name: "linux-provider", Capability: "host-metrics", Role: "peer", Platforms: map[HostOS]PlatformDeclaration{HostOSLinux: {Status: string(StatusSupported)}}},
	}, "host-metrics", HostOSWindows)
	if resolution.Status != CapabilityImplemented || len(resolution.Absent) != 1 || resolution.Absent[0] != "linux-provider" {
		t.Fatalf("expected winner plus absent provider alternative, got %+v", resolution)
	}
}
