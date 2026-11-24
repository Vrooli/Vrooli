package modelcatalog

import (
	"strings"
	"testing"
)

func TestRuntimeDeclarationsHaveUniqueIDsAndExplicitArtifactPolicies(t *testing.T) {
	seen := make(map[string]struct{}, len(Runtimes))
	for _, runtime := range Runtimes {
		if strings.TrimSpace(runtime.ID) == "" {
			t.Fatal("runtime declaration has an empty ID")
		}
		if _, duplicate := seen[runtime.ID]; duplicate {
			t.Errorf("runtime ID %q is declared more than once", runtime.ID)
		}
		seen[runtime.ID] = struct{}{}
		switch runtime.ArtifactPolicy {
		case RuntimeArtifactRequired, RuntimeArtifactExternal:
		default:
			t.Errorf("runtime %q has missing or unknown artifact policy %q", runtime.ID, runtime.ArtifactPolicy)
		}
	}
}

func TestRuntimeArtifactValidationFollowsDeclaration(t *testing.T) {
	for _, runtime := range Runtimes {
		t.Run(runtime.ID, func(t *testing.T) {
			m := validModel("runtime-artifact-policy", runtime.Capabilities[0])
			m.Runtime = runtime.ID
			m.Artifact = nil
			got := hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidArtifact, m.ID)
			want := runtime.ArtifactPolicy == RuntimeArtifactRequired
			if got != want {
				t.Fatalf("artifact validation issue = %v, want %v for policy %q", got, want, runtime.ArtifactPolicy)
			}
		})
	}
}

func TestRuntimeRejectsUndeclaredCapability(t *testing.T) {
	m := validModel("tts-on-whisper", CapabilityTTS)
	m.Runtime = "whisper.cpp"
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueUnsupportedRuntimeCapability, m.ID) {
		t.Fatal("runtime accepted a capability with no declared adapter path")
	}
}

func TestValidationRejectsDuplicateIDs(t *testing.T) {
	first := validModel("duplicate", CapabilitySTT)
	second := validModel("duplicate", CapabilitySTT)
	issues := New(fixture(first, second)).ValidationIssues()
	if !hasIssue(issues, IssueDuplicateID, "duplicate") {
		t.Fatalf("expected duplicate_id, got %v", issues)
	}
}

func TestEngineDefaultMustNameItsEngineAndBeUnique(t *testing.T) {
	wrongEngine := validModel("wrong-engine-default", CapabilitySTT)
	wrongEngine.EngineIDs = []string{"whisper-local"}
	wrongEngine.EngineDefaults = []string{"sherpa-streaming"}
	if !hasIssue(New(fixture(wrongEngine)).ValidationIssues(), IssueInvalidEngineDefault, wrongEngine.ID) {
		t.Fatal("engine default not present in engine_ids was accepted")
	}

	first := validModel("engine-default-a", CapabilitySTT)
	first.EngineIDs = []string{"shared-engine"}
	first.EngineDefaults = []string{"shared-engine"}
	second := first
	second.ID = "engine-default-b"
	issues := New(fixture(first, second)).ValidationIssues()
	if !hasIssue(issues, IssueInvalidEngineDefault, second.ID) {
		t.Fatalf("duplicate capability/engine defaults were accepted: %v", issues)
	}
}

func TestValidationRequiresLicenseNotesForNonCommercial(t *testing.T) {
	m := validModel("nc", CapabilityTTS)
	m.Commercial = CommercialNo
	m.LicenseNotes = ""
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueMissingLicenseNotes, "nc") {
		t.Fatal("expected missing_license_notes for commercial_use=no")
	}

	unknown := validModel("unknown-licence", CapabilityTTS)
	unknown.Commercial = CommercialUnknown
	unknown.LicenseNotes = ""
	if !hasIssue(New(fixture(unknown)).ValidationIssues(), IssueMissingLicenseNotes, "unknown-licence") {
		t.Fatal("expected missing_license_notes for commercial_use=unknown")
	}
}

func TestValidationRejectsUnknownCommercialUse(t *testing.T) {
	m := validModel("unknown-commercial-use", CapabilityTTS)
	m.Commercial = CommercialUse("sometimes")

	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidCommercialUse, m.ID) {
		t.Fatal("unknown commercial-use declaration was accepted")
	}
}

func TestValidationRejectsNonAvailableDefaultWithoutOptIn(t *testing.T) {
	m := validModel("experimental-default", CapabilitySTT)
	m.Readiness = ReadinessExperimental
	m.Roles = []string{"default"}
	m.Notes = "experimental path"

	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidDefault, "experimental-default") {
		t.Fatal("expected invalid_default for an experimental default without allow_default")
	}

	m.AllowDefault = true
	if hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidDefault, "experimental-default") {
		t.Fatal("allow_default should make the default role valid")
	}
}

func TestDefaultPromotionRequiresAllOwnerReceiptsOrLegacyException(t *testing.T) {
	m := validModel("new-default", CapabilitySTT)
	m.Promotion = PromotionEvidence{}
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidPromotionEvidence, m.ID) {
		t.Fatal("default without qualification receipt or explicit legacy exception was accepted")
	}
	if m.DefaultEligible() {
		t.Fatal("unqualified model with default role is default-eligible")
	}
	m.Promotion = PromotionEvidence{
		CorpusQuality: EvidenceFacet{Status: "qualified", Ref: "experiment-quality:engine:model:strategy"},
		Latency:       EvidenceFacet{Status: "qualified", Ref: "latency:run-1"},
		Memory:        EvidenceFacet{Status: "qualified", Ref: "memory:run-1"},
		Platform:      EvidenceFacet{Status: "qualified", Ref: "platform:linux-amd64"},
		License:       EvidenceFacet{Status: "qualified", Ref: "license:source-revision"},
	}
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidPromotionEvidence, m.ID) {
		t.Fatal("self-declared refs were trusted without owner receipt resolution")
	}
	if m.Promotion.Qualified() || m.DefaultEligible() {
		t.Fatal("seed declarations alone qualified a promotion")
	}
	m.Promotion.Memory = EvidenceFacet{Status: "qualified"}
	if m.Promotion.Qualified() {
		t.Fatal("qualified facet without owner receipt reference counted as complete")
	}
}

func TestLegacyExceptionCannotAuthorizeANewDefault(t *testing.T) {
	m := validModel("fabricated-fifth-default", CapabilitySTT)
	m.Promotion = PromotionEvidence{LegacyException: "Existing operational default retained pending receipts."}

	if m.DefaultEligible() {
		t.Fatal("caller-declared legacy exception made a new model default-eligible")
	}
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidPromotionEvidence, m.ID) {
		t.Fatal("caller-declared legacy exception was not rejected by validation")
	}
}

func TestCurrentOperationalDefaultsRemainEligibleThroughAllowlist(t *testing.T) {
	c := Default()
	for _, capability := range DefaultCapabilities {
		m, err := c.DefaultFor(capability, "", Platform{})
		if err != nil {
			t.Fatalf("DefaultFor(%s): %v", capability, err)
		}
		if !legacyExceptionAllowed(m.ID, m.Promotion.LegacyException) {
			t.Errorf("current default %q is missing its allowlisted exception", m.ID)
		}
	}
}

func TestValidationRejectsEmptyProviderIDs(t *testing.T) {
	m := validModel("no-provider", CapabilityTTS)
	m.ProviderIDs = nil
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueMissingProviderID, "no-provider") {
		t.Fatal("expected missing_provider_id")
	}
}

func TestValidationRejectsUnknownProviderID(t *testing.T) {
	m := validModel("unknown-provider", CapabilityTTS)
	m.ProviderIDs = []string{"retired-provider"}
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueUnknownProviderID, "unknown-provider") {
		t.Fatal("expected unknown_provider_id")
	}
}

func TestValidationRejectsEngineOnNonSTT(t *testing.T) {
	m := validModel("tts-with-engine", CapabilityTTS)
	m.EngineIDs = []string{"whisper-local"}
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueUnexpectedEngineID, "tts-with-engine") {
		t.Fatal("expected unexpected_engine_id")
	}
}

func TestValidationRejectsEmptyLanguages(t *testing.T) {
	m := validModel("no-language", CapabilityTTS)
	m.Languages = nil
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidLanguage, "no-language") {
		t.Fatal("expected invalid_language")
	}
}

func TestValidationRejectsIncoherentHardware(t *testing.T) {
	m := validModel("both-cpu-and-gpu", CapabilityTTS)
	m.Hardware.CPUCapable = true
	m.Hardware.GPURequired = true
	m.Hardware.MinVRAMGB = 8
	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueInvalidHardware, "both-cpu-and-gpu") {
		t.Fatal("expected invalid_hardware")
	}
}

func TestValidationRejectsStalePolicy(t *testing.T) {
	seed := fixture(validModel("stt", CapabilitySTT))
	seed.SelectionPolicy.RoleOrder = []string{"default", "fast"}
	issues := New(seed).ValidationIssues()
	found := false
	for _, issue := range issues {
		if issue.Code == IssueInvalidPolicy {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected invalid_policy, got %v", issues)
	}
}

func TestDuplexPricingMustBeCompleteAndSourced(t *testing.T) {
	m := validModel("duplex", CapabilityDuplex)
	m.Runtime = "byok"
	m.ProviderIDs = []string{"openai-gpt-live"}
	m.Pricing = &Pricing{Currency: "USD", Unit: "minute", BillingIncrementSeconds: 1, SourceURL: "https://example.test/pricing"}
	issues := New(fixture(m)).ValidationIssues()
	if !hasIssue(issues, IssueInvalidPricing, "duplex") {
		t.Fatal("expected missing active and standby rates to be rejected")
	}
	active, standby := int64(50_000), int64(20_000)
	m.Pricing.ActiveMicrosPerMinute, m.Pricing.StandbyMicrosPerMinute = &active, &standby
	if issues := New(fixture(m)).ValidationIssues(); hasIssue(issues, IssueInvalidPricing, "duplex") {
		t.Fatalf("complete sourced pricing rejected: %v", issues)
	}
	negative := int64(-1)
	m.Pricing.ActiveMicrosPerMinute = &negative
	if issues := New(fixture(m)).ValidationIssues(); !hasIssue(issues, IssueInvalidPricing, "duplex") {
		t.Fatal("expected negative integer micro-USD rate to be rejected")
	}
}

func TestGPTLiveDeclarationStaysUnqualifiedAndRendersRates(t *testing.T) {
	m, ok := Default().ByID("gpt-live-1.1")
	if !ok {
		t.Fatal("GPT-Live 1.1 record missing")
	}
	if m.Readiness != ReadinessCandidate || m.HasRole("default") {
		t.Fatalf("GPT-Live readiness/roles = %s/%v, must remain unqualified", m.Readiness, m.Roles)
	}
	if m.Pricing == nil || m.Pricing.ActiveMicrosPerMinute == nil || *m.Pricing.ActiveMicrosPerMinute != 50_000 || m.Pricing.StandbyMicrosPerMinute == nil || *m.Pricing.StandbyMicrosPerMinute != 20_000 {
		t.Fatalf("published pricing incomplete: %+v", m.Pricing)
	}
	doc := Default().RenderMarkdown()
	for _, want := range []string{"gpt-live-1.1", "$0.05 active / $0.02 standby USD/min", "[source](https://developers.openai.com/api/docs/pricing)", "candidate"} {
		if !strings.Contains(doc, want) {
			t.Errorf("generated docs missing %q", want)
		}
	}
}
