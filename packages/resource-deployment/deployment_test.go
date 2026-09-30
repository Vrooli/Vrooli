package resourcedeployment

import "testing"

func TestTargetNormalizesPlatformsAndEnforcesArchitecture(t *testing.T) {
	d := Deployment{Profiles: map[string]Profile{"desktop": {
		Linux:   &Target{Support: "supported", Architectures: []string{"amd64"}},
		MacOS:   &Target{Support: "conditional", Architectures: []string{"arm64"}},
		Windows: &Target{Support: "unsupported", Reason: "not available"},
	}}}
	if target, ok := d.Target("desktop", "mac", "arm64"); !ok || target.Support != "conditional" {
		t.Fatalf("mac/arm64 target = %#v, %t", target, ok)
	}
	if _, ok := d.Target("desktop", "linux", "arm64"); ok {
		t.Fatal("linux arm64 must not resolve when only amd64 is declared")
	}
	if target, ok := d.Target("desktop", "win", "arm64"); !ok || target.Support != "unsupported" {
		t.Fatalf("unsupported target = %#v, %t", target, ok)
	}
}

func TestParsePlatformAndArtifactPathSafety(t *testing.T) {
	platform, err := ParsePlatform("mac-arm64")
	if err != nil || platform.OS != "macos" || platform.Arch != "arm64" {
		t.Fatalf("ParsePlatform(mac-arm64) = %#v, %v", platform, err)
	}
	if _, err := ParsePlatform("linux"); err == nil {
		t.Fatal("expected target without architecture to be rejected")
	}
	if _, err := ArtifactName("../escape_${os}_${arch}", "linux", "amd64"); err == nil {
		t.Fatal("expected traversal artifact template to be rejected")
	}
	files, err := ArtifactFiles("resource-demo_linux_amd64")
	if err != nil || len(files) != 3 || files[2] != "resource-demo_linux_amd64.build.json" {
		t.Fatalf("ArtifactFiles = %v, %v", files, err)
	}
}

func TestProviderPolicyFailsClosedForSharedAndExternalModes(t *testing.T) {
	policy := ProviderPolicy{
		DefaultMode:                ProviderManagedPrivate,
		AllowedModes:               []ProviderMode{ProviderManagedPrivate, ProviderManagedShared, ProviderAttachOnly, ProviderRemoteVrooli},
		SharedReuseRequiresConsent: true,
		ExternalManagement:         "forbidden",
		ExternalAccessCapabilities: []AccessCapability{AccessReadOnly},
	}
	if got, err := policy.ResolveProvider(ProviderRequest{}); err != nil || got != ProviderManagedPrivate {
		t.Fatalf("default provider = %q, %v", got, err)
	}
	if _, err := policy.ResolveProvider(ProviderRequest{Mode: ProviderManagedShared}); err == nil {
		t.Fatal("shared provider must require explicit consent")
	}
	if got, err := policy.ResolveProvider(ProviderRequest{Mode: ProviderManagedShared, SharedConsented: true}); err != nil || got != ProviderManagedShared {
		t.Fatalf("consented shared provider = %q, %v", got, err)
	}
	if got, err := policy.ResolveProvider(ProviderRequest{Mode: ProviderAttachOnly}); err != nil || got != ProviderAttachOnly {
		t.Fatalf("attach-only provider = %q, %v", got, err)
	}
	if _, err := policy.ResolveProvider(ProviderRequest{Mode: "unverified-local"}); err == nil {
		t.Fatal("unverified local provider mode must be rejected")
	}
}

func TestProviderPolicyUsesTargetDefaultsWithoutOverridingExplicitChoice(t *testing.T) {
	policy := ProviderPolicy{
		DefaultMode: ProviderManagedDiscovered,
		TargetDefaults: map[ProviderTarget]ProviderMode{
			ProviderTargetControlPlane:  ProviderManagedShared,
			ProviderTargetDesktopBundle: ProviderManagedPrivate,
		},
		AllowedModes:               []ProviderMode{ProviderManagedPrivate, ProviderManagedShared, ProviderManagedDiscovered},
		SharedReuseRequiresConsent: true,
		ExternalManagement:         "forbidden",
	}

	if got, err := policy.ResolveProvider(ProviderRequest{Target: ProviderTargetControlPlane}); err != nil || got != ProviderManagedShared {
		t.Fatalf("control-plane default = %q, %v", got, err)
	}
	if got, err := policy.ResolveProvider(ProviderRequest{Target: ProviderTargetDesktopBundle}); err != nil || got != ProviderManagedPrivate {
		t.Fatalf("desktop default = %q, %v", got, err)
	}
	if _, err := policy.ResolveProvider(ProviderRequest{Target: ProviderTargetDesktopBundle, Mode: ProviderManagedShared}); err == nil {
		t.Fatal("desktop shared provider must require explicit consent")
	}
	if got, err := policy.ResolveProvider(ProviderRequest{Target: ProviderTargetDesktopBundle, Mode: ProviderManagedShared, SharedConsented: true}); err != nil || got != ProviderManagedShared {
		t.Fatalf("consented desktop shared provider = %q, %v", got, err)
	}
	if got, err := policy.ResolveProvider(ProviderRequest{Target: ProviderTargetDesktopBundle, Mode: ProviderManagedDiscovered}); err != nil || got != ProviderManagedDiscovered {
		t.Fatalf("explicit provider must override desktop default = %q, %v", got, err)
	}
	if _, err := policy.ResolveProvider(ProviderRequest{Target: ProviderTarget("unknown-target")}); err == nil {
		t.Fatal("unknown target without an explicit provider was accepted")
	}
}

func TestProviderPolicyRequiresTargetDefaultsForManagedServices(t *testing.T) {
	policy := ProviderPolicy{
		DefaultMode:  ProviderManagedPrivate,
		AllowedModes: []ProviderMode{ProviderManagedPrivate, ProviderManagedShared},
	}
	if err := policy.ValidateManagedServiceTargets(); err == nil {
		t.Fatal("managed service accepted a static default")
	}

	policy.DefaultMode = ""
	policy.TargetDefaults = map[ProviderTarget]ProviderMode{
		ProviderTargetControlPlane:  ProviderManagedShared,
		ProviderTargetDesktopBundle: ProviderManagedPrivate,
	}
	if err := policy.ValidateManagedServiceTargets(); err != nil {
		t.Fatalf("managed service target defaults: %v", err)
	}
}

func TestProviderPolicyRequiresDeclaredExternalAccess(t *testing.T) {
	policy := ProviderPolicy{
		DefaultMode:        ProviderAttachOnly,
		AllowedModes:       []ProviderMode{ProviderAttachOnly},
		ExternalManagement: "forbidden",
	}
	if _, err := policy.ResolveProvider(ProviderRequest{}); err == nil {
		t.Fatal("attach-only provider without access capabilities was accepted")
	}
	policy.ExternalAccessCapabilities = []AccessCapability{AccessReadOnly}
	if _, err := policy.ResolveProvider(ProviderRequest{}); err != nil {
		t.Fatalf("attach-only provider with explicit capability: %v", err)
	}
	if !policy.AllowsExternalAccess(AccessReadOnly) || policy.AllowsExternalAccess(AccessReadWrite) {
		t.Fatal("external access capability policy did not prevent escalation")
	}
}

func TestProviderPolicyRejectsAmbiguousProviderAndAccessCombinations(t *testing.T) {
	for _, policy := range []ProviderPolicy{
		{
			DefaultMode:                ProviderManagedPrivate,
			AllowedModes:               []ProviderMode{ProviderManagedPrivate},
			ExternalAccessCapabilities: []AccessCapability{AccessReadOnly},
		},
		{
			DefaultMode:  ProviderManagedPrivate,
			AllowedModes: []ProviderMode{ProviderManagedPrivate, ProviderAttachOnly},
		},
		{
			DefaultMode:                ProviderAttachOnly,
			AllowedModes:               []ProviderMode{ProviderAttachOnly},
			ExternalAccessCapabilities: []AccessCapability{AccessCapability("administrative")},
		},
		{
			DefaultMode:                ProviderAttachOnly,
			AllowedModes:               []ProviderMode{ProviderAttachOnly},
			ExternalAccessCapabilities: []AccessCapability{AccessReadOnly, AccessReadOnly},
		},
	} {
		if err := policy.Validate(); err == nil {
			t.Fatalf("ambiguous policy %#v was accepted", policy)
		}
	}
}

func TestManagedServiceAttachHealthPathRejectsAmbiguousTargets(t *testing.T) {
	for _, path := range []string{"/v1/sys/health", "/health/check"} {
		if err := (ManagedService{AttachHealthPath: path}).ValidateAttachHealthPath(); err != nil {
			t.Fatalf("valid attach path %q: %v", path, err)
		}
	}
	for _, path := range []string{"", "health", "https://vault.example/health", "/health?token=x", "/health#fragment"} {
		if err := (ManagedService{AttachHealthPath: path}).ValidateAttachHealthPath(); err == nil {
			t.Fatalf("invalid attach path %q was accepted", path)
		}
	}
}

func TestServiceShutdownValidatesPlatformNeutralSignals(t *testing.T) {
	for _, signal := range []string{ServiceShutdownTerminate, ServiceShutdownInterrupt} {
		if err := (ServiceShutdown{Signal: signal}).Validate(); err != nil {
			t.Fatalf("valid shutdown signal %q: %v", signal, err)
		}
	}
	for _, signal := range []string{"", "sigkill", "SIGTERM"} {
		if err := (ServiceShutdown{Signal: signal}).Validate(); err == nil {
			t.Fatalf("invalid shutdown signal %q was accepted", signal)
		}
	}
}
