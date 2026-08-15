package deployability

import "testing"

func TestResolveCapabilityUsesDeclaredPeersAndMechanisms(t *testing.T) {
	implementations := []CapabilityImplementation{
		{Name: "linux-primary", Capability: "credential-storage", Role: "primary", Platforms: map[HostOS]PlatformDeclaration{
			HostOSLinux: {Status: platformSupported},
			HostOSMacOS: {Status: platformUnsupported, Mechanism: "native-keychain adapter"},
		}},
		{Name: "mac-peer", Capability: "credential-storage", Role: "peer", Platforms: map[HostOS]PlatformDeclaration{
			HostOSMacOS: {Status: platformSupported},
		}},
	}

	mac := ResolveCapability(implementations, "credential-storage", HostOSMacOS)
	if mac.Status != CapabilityImplemented || mac.Implementer != "mac-peer" {
		t.Fatalf("expected macOS peer, got %+v", mac)
	}

	windows := ResolveCapability(implementations, "credential-storage", HostOSWindows)
	if windows.Status != CapabilityPeerless {
		t.Fatalf("expected peerless Windows capability, got %+v", windows)
	}

	unwired := ResolveCapability([]CapabilityImplementation{{
		Name: "linux-only", Capability: "credential-storage", Role: "primary",
		Platforms: map[HostOS]PlatformDeclaration{HostOSWindows: {Status: platformUnsupported, Mechanism: "Windows Credential Manager adapter"}},
	}}, "credential-storage", HostOSWindows)
	if unwired.Status != CapabilityUnwired || unwired.Mechanism == "" {
		t.Fatalf("expected named unwired mechanism, got %+v", unwired)
	}
}
