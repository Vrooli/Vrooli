package deployability

import (
	"strings"
	"testing"
)

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

func TestResolveUsesArchitectureSpecificPlatformSupport(t *testing.T) {
	dependency := testDependency("architecture-specific", &ResourceRequirements{Class: "service", Weight: 1})
	dependency.PlatformSupportByTarget = map[string]PlatformDeclaration{
		"linux-amd64": {Status: "supported"},
		"linux-arm64": {Status: "unsupported"},
	}

	amd64 := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
		Tier:   TierDesktop,
		OS:     HostOSLinux,
		Arch:   "amd64",
	})
	if amd64.Verdict != VerdictEligible {
		t.Fatalf("amd64 verdict = %s, want eligible: %#v", amd64.Verdict, amd64.Reasons)
	}

	arm64 := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
		Tier:   TierDesktop,
		OS:     HostOSLinux,
		Arch:   "arm64",
	})
	if arm64.Verdict != VerdictIneligible {
		t.Fatalf("arm64 verdict = %s, want ineligible: %#v", arm64.Verdict, arm64.Reasons)
	}
}

func TestResolveReturnsIneligibleForDeclaredMobileFootprint(t *testing.T) {
	result := Resolve(ResolutionInput{Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{testDependency("custom", &ResourceRequirements{Class: "custom", Weight: 3, RAMMB: 4096, CPUCores: 2})}}, Tier: TierMobile, OS: HostOSLinux})
	if result.Verdict != VerdictIneligible {
		t.Fatalf("verdict = %s, want ineligible", result.Verdict)
	}
}

func TestResolveEvaluatesMinimumCUDAComputeFromFacts(t *testing.T) {
	dependency := testDependency("gpu-service", &ResourceRequirements{
		Class:          "inference",
		Weight:         1,
		GPURequirement: &GPURequirement{MinCUDACompute: "8.9"},
	})
	eligible := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
		Tier:   TierLocal, OS: HostOSLinux, Arch: "amd64",
		Facts: map[string]string{"gpu.cuda_compute": "9.0"},
	})
	if eligible.Verdict != VerdictEligible {
		t.Fatalf("eligible verdict = %s, want eligible: %+v", eligible.Verdict, eligible.Reasons)
	}
	ineligible := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
		Tier:   TierLocal, OS: HostOSLinux, Arch: "amd64",
		Facts: map[string]string{"gpu.cuda_compute": "8.0"},
	})
	if ineligible.Verdict != VerdictIneligible {
		t.Fatalf("ineligible verdict = %s, want ineligible: %+v", ineligible.Verdict, ineligible.Reasons)
	}
}

func TestResolveGPURequirementIsUnknownWhenFactIsAbsent(t *testing.T) {
	dependency := testDependency("gpu-service", &ResourceRequirements{
		Class:          "inference",
		Weight:         1,
		GPURequirement: &GPURequirement{MinCUDACompute: "8.9"},
	})
	result := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
		Tier:   TierLocal, OS: HostOSLinux, Arch: "amd64",
	})
	if result.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %s, want unknown: %+v", result.Verdict, result.Reasons)
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

func TestResolveGivesEveryPlatformStatusAnExplicitVerdict(t *testing.T) {
	cases := []struct {
		status  PlatformStatus
		verdict Verdict
		code    string
	}{
		{status: StatusSupported, verdict: VerdictEligible},
		{status: StatusBuildVerified, verdict: VerdictDegraded, code: "platform_build_verified"},
		{status: StatusExperimental, verdict: VerdictDegraded, code: "platform_experimental"},
		{status: StatusUnqualified, verdict: VerdictDegraded, code: "platform_unqualified"},
		{status: StatusPartial, verdict: VerdictDegraded, code: "platform_partial"},
		{status: StatusUnsupported, verdict: VerdictIneligible, code: "platform_unsupported"},
	}
	for _, testCase := range cases {
		t.Run(string(testCase.status), func(t *testing.T) {
			dependency := testDependency("subject", &ResourceRequirements{Class: "service", Weight: 1})
			dependency.PlatformSupport = map[HostOS]PlatformDeclaration{HostOSLinux: {Status: string(testCase.status)}}
			result := Resolve(ResolutionInput{
				Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
				Tier:   TierLocal, OS: HostOSLinux,
			})
			if result.Verdict != testCase.verdict {
				t.Fatalf("verdict = %s, want %s: %+v", result.Verdict, testCase.verdict, result.Reasons)
			}
			if testCase.code == "" {
				if len(result.Reasons) != 0 {
					t.Fatalf("a fully supported platform should need no reasons, got %+v", result.Reasons)
				}
				return
			}
			if !hasReasonCode(result.Reasons, testCase.code) {
				t.Fatalf("expected reason code %q, got %+v", testCase.code, result.Reasons)
			}
		})
	}
}

func TestResolveRejectsAPlatformStatusOutsideTheVocabulary(t *testing.T) {
	dependency := testDependency("subject", &ResourceRequirements{Class: "service", Weight: 1})
	dependency.PlatformSupport = map[HostOS]PlatformDeclaration{HostOSLinux: {Status: "available"}}
	result := Resolve(ResolutionInput{
		Target: TargetDeclaration{Name: "sample", Dependencies: []DependencyDeclaration{dependency}},
		Tier:   TierLocal, OS: HostOSLinux,
	})
	if result.Verdict != VerdictUnknown {
		t.Fatalf("verdict = %s, want unknown", result.Verdict)
	}
	if !hasReasonCode(result.Reasons, "platform_status_unknown") {
		t.Fatalf("expected platform_status_unknown, got %+v", result.Reasons)
	}
	for _, reason := range result.Reasons {
		if reason.Code == "platform_status_unknown" && !strings.Contains(reason.Message, "available") {
			t.Fatalf("rejection must name the offending token, got %q", reason.Message)
		}
	}
}

func hasReasonCode(reasons []Reason, code string) bool {
	for _, reason := range reasons {
		if reason.Code == code {
			return true
		}
	}
	return false
}
