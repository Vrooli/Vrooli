package deployability

import "testing"

func testDependency(name string, requirements *ResourceRequirements) DependencyDeclaration {
	return DependencyDeclaration{
		Kind: "resource", Name: name, Required: true,
		Bundling: BundlingVendorable, Present: true, Artifact: true,
		PlatformSupport: map[HostOS]PlatformDeclaration{
			HostOSLinux: {Status: "supported"}, HostOSMacOS: {Status: "supported"}, HostOSWindows: {Status: "supported"},
		},
		Requirements: requirements,
	}
}

func TestResolveUsesDeclaredRequirementsRatherThanInstanceNames(t *testing.T) {
	requirements := &ResourceRequirements{Class: "custom", Weight: 1, RAMMB: 512, CPUCores: 1}
	result := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{testDependency("not-a-known-resource", requirements)}},
		Tier:   TierMobile, OS: HostOSLinux,
	})
	if result.Verdict != VerdictEligible {
		t.Fatalf("verdict = %s, want eligible: %+v", result.Verdict, result.Reasons)
	}
}

func TestResolveReturnsUnknownWhenRequirementsAreMissing(t *testing.T) {
	result := Resolve(ResolutionInput{Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{testDependency("custom", nil)}}, Tier: TierLocal, OS: HostOSLinux})
	if result.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %s, want unknown", result.Verdict)
	}
}

func TestResolveReturnsIneligibleForDeclaredMobileFootprint(t *testing.T) {
	result := Resolve(ResolutionInput{Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{testDependency("custom", &ResourceRequirements{Class: "custom", Weight: 3, RAMMB: 4096, CPUCores: 2})}}, Tier: TierMobile, OS: HostOSLinux})
	if result.Verdict != VerdictIneligible {
		t.Fatalf("verdict = %s, want ineligible", result.Verdict)
	}
}

func TestResolveUsesDeclaredDesktopHostRequirements(t *testing.T) {
	dependency := testDependency("config-driven", &ResourceRequirements{Class: "service", Weight: 1})
	dependency.HostRequirements = map[HostOS][]string{HostOSLinux: {"docker"}}

	result := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
		Tier:   TierDesktop,
		OS:     HostOSLinux,
	})
	if result.Verdict != VerdictIneligible {
		t.Fatalf("verdict = %s, want ineligible: %+v", result.Verdict, result.Reasons)
	}
	foundRequirement := false
	for _, reason := range result.Reasons {
		if reason.Requirement == "docker" {
			foundRequirement = true
			break
		}
	}
	if !foundRequirement {
		t.Fatalf("reasons = %#v, want a docker host requirement", result.Reasons)
	}

	local := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
		Tier:   TierLocal,
		OS:     HostOSLinux,
	})
	if local.Verdict != VerdictEligible {
		t.Fatalf("local verdict = %s, want eligible: %+v", local.Verdict, local.Reasons)
	}
}

func TestResolveReturnsUnknownForUnrecognizedVocabulary(t *testing.T) {
	result := Resolve(ResolutionInput{Target: TargetDeclaration{Name: "sample"}, Tier: DeliveryTier("portable"), OS: HostOSLinux})
	if result.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %s, want unknown", result.Verdict)
	}
}

func TestResolvePropagatesTransitiveUnknown(t *testing.T) {
	child := TargetDeclaration{Name: "child", Dependencies: []DependencyDeclaration{testDependency("missing", nil)}}
	parent := testDependency("child", &ResourceRequirements{Class: "service", Weight: 1})
	parent.Kind = "scenario"
	parent.Children = []TargetDeclaration{child}
	result := Resolve(ResolutionInput{Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{parent}}, Tier: TierLocal, OS: HostOSLinux})
	if result.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %s, want unknown", result.Verdict)
	}
}

func TestFindInstanceLiteralsDetectsManifestNames(t *testing.T) {
	hits, err := FindInstanceLiterals("sample.go", []byte(`package sample
const dependency = "fleet-resource"
`), []string{"fleet-resource"})
	if err != nil {
		t.Fatalf("FindInstanceLiterals: %v", err)
	}
	if len(hits) != 1 || hits[0].Value != "fleet-resource" {
		t.Fatalf("hits = %+v, want one exact manifest name", hits)
	}
}

func TestValidateObservedPlatformRejectsContradictoryObservation(t *testing.T) {
	input := ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{testDependency("custom", nil)}},
		Tier:   TierLocal,
	}
	result := ValidateObservedPlatform(input, PlatformObservation{OS: HostOSWindows, Reason: "target probe failed"})
	if result.Verdict != VerdictIneligible {
		t.Fatalf("verdict = %s, want ineligible", result.Verdict)
	}
	if len(result.Reasons) == 0 || result.Reasons[len(result.Reasons)-1].Code != "observed_platform_unavailable" {
		t.Fatalf("reasons = %#v, want observed_platform_unavailable", result.Reasons)
	}
}
