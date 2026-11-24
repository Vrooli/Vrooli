package modelcatalog

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEmbeddedSeedIsValid(t *testing.T) {
	c := Default()
	issues := c.ValidationIssues()
	if len(issues) != 0 {
		for _, issue := range issues {
			t.Errorf("validation issue: %s", issue)
		}
	}
	if got := len(c.All()); got < 20 {
		t.Fatalf("embedded seed has %d records, want a complete registry", got)
	}
}

func TestEveryCapabilityHasModel(t *testing.T) {
	c := Default()
	for _, capability := range Capabilities {
		models := c.ByCapability(capability)
		if len(models) == 0 {
			t.Errorf("capability %q has no model", capability)
			continue
		}
		for _, m := range models {
			if len(m.ProviderIDs) == 0 {
				t.Errorf("%s (%s) has no provider ids", m.ID, capability)
			}
		}
	}
}

func TestByEngineModelIDResolvesDeclaredRuntimeAlias(t *testing.T) {
	model, ok := Default().ByEngineModelID("sherpa-streaming", "sherpa-onnx/streaming-zipformer-en-2023-06-26")
	if !ok {
		t.Fatal("runtime model id did not resolve")
	}
	if model.ID != "sherpa-zipformer-en-2023-06-26" {
		t.Fatalf("canonical id = %q", model.ID)
	}
	if model.RuntimeModelID != "sherpa-onnx/streaming-zipformer-en-2023-06-26" {
		t.Fatalf("runtime model id = %q", model.RuntimeModelID)
	}
}

func TestByEngineModelIDRejectsAmbiguousRuntimeAlias(t *testing.T) {
	catalog := &Catalog{byEngine: map[string][]Model{"engine": {
		{ID: "model-a", EngineIDs: []string{"engine"}, RuntimeModelID: "runtime-model"},
		{ID: "model-b", EngineIDs: []string{"engine"}, RuntimeModelID: "runtime-model"},
	}}}
	if _, ok := catalog.ByEngineModelID("engine", "runtime-model"); ok {
		t.Fatal("ambiguous runtime model id resolved")
	}
}

func TestDefaultMapsToKnownProvider(t *testing.T) {
	c := Default()
	known := stringSet(c.seed.KnownProviderIDs)
	for _, capability := range DefaultCapabilities {
		m, err := c.DefaultFor(capability, "", Platform{})
		if err != nil {
			t.Errorf("DefaultFor(%s): %v", capability, err)
			continue
		}
		if len(m.ProviderIDs) == 0 {
			t.Errorf("default %s has no provider ids", m.ID)
		}
		for _, id := range m.ProviderIDs {
			if _, ok := known[id]; !ok {
				t.Errorf("default %s names unknown provider id %q", m.ID, id)
			}
		}
	}
}

func TestEngineIDsFollowStreamSelection(t *testing.T) {
	c := Default()
	// Local streaming/batch engines participate in StreamConfig.EngineID
	// selection and must name their engine.
	for _, id := range []string{
		"whisper-medium", "whisper-large-v3", "whisper-large-v3-turbo",
	} {
		m, ok := c.ByID(id)
		if !ok {
			t.Fatalf("%s record missing", id)
		}
		if !contains(m.EngineIDs, "whisper-local") {
			t.Errorf("%s engine ids = %v, want whisper-local", id, m.EngineIDs)
		}
	}
	// Chain-routed remote STT providers do not declare local engine ids.
	for _, id := range []string{"openai-whisper-1", "deepgram-nova-3"} {
		m, ok := c.ByID(id)
		if !ok {
			t.Fatalf("%s record missing", id)
		}
		if len(m.EngineIDs) != 0 {
			t.Errorf("%s engine ids = %v, want none", id, m.EngineIDs)
		}
	}
	rumble, ok := c.ByID("rumble-2-flash")
	if !ok || len(rumble.EngineIDs) != 0 {
		t.Errorf("Rumble engine ids = %v, want none (unary model routing is separate from stream engine selection)", rumble.EngineIDs)
	}
	for _, m := range c.ByCapability(CapabilityTTS) {
		if len(m.EngineIDs) != 0 {
			t.Errorf("non-STT model %s declares engine ids %v", m.ID, m.EngineIDs)
		}
	}
	sherpa, ok := c.ByID("sherpa-zipformer-en-2023-06-26")
	if !ok {
		t.Fatal("sherpa model record missing")
	}
	if !contains(sherpa.ProviderIDs, "sherpa-stt") {
		t.Errorf("sherpa provider ids = %v, want sherpa-stt", sherpa.ProviderIDs)
	}
	if !contains(sherpa.EngineIDs, "sherpa-streaming") {
		t.Errorf("sherpa engine ids = %v, want sherpa-streaming", sherpa.EngineIDs)
	}
}

func TestRumbleCandidateCannotBecomeDefaultWithoutQualification(t *testing.T) {
	c := Default()
	rumble, ok := c.ByID("rumble-2-flash")
	if !ok {
		t.Fatal("rumble-2-flash record missing")
	}
	if rumble.Readiness != ReadinessCandidate {
		t.Fatalf("Rumble readiness = %q, want candidate until its open runtime/quality gates pass", rumble.Readiness)
	}
	if rumble.DefaultEligible() || rumble.HasRole("default") {
		t.Fatalf("Rumble is eligible/promoted before its runtime exists: readiness=%q roles=%v", rumble.Readiness, rumble.Roles)
	}
	if rumble.Artifact != nil {
		t.Fatalf("Rumble declares local artifact %q/%q without a managed resource", rumble.Artifact.Resource, rumble.Artifact.Name)
	}
	if rumble.BatchAdapterID != "vllm-openai-chat-audio" {
		t.Fatalf("Rumble batch adapter = %q, want declared vLLM audio-chat protocol", rumble.BatchAdapterID)
	}
	if len(rumble.EngineIDs) != 0 {
		t.Fatalf("Rumble must remain outside stream engine selection, got ids %v", rumble.EngineIDs)
	}
	got, err := c.DefaultFor(CapabilitySTT, "en", Platform{OS: "linux", Arch: "amd64", Accelerated: true, FreeVRAMGB: 24, RAMGB: 64})
	if err != nil {
		t.Fatalf("existing STT default should remain resolvable: %v", err)
	}
	if got.ID == rumble.ID {
		t.Fatal("Rumble became the STT default without runtime qualification")
	}
}

func TestSherpa20MZipformerIsRegistryOnlyCandidate(t *testing.T) {
	c := Default()
	model, ok := c.ByID("sherpa-zipformer-en-20m-2023-02-17")
	if !ok {
		t.Fatal("Sherpa 20M Zipformer candidate missing")
	}
	if model.Runtime != "sherpa-onnx" || model.RuntimeModelID != "sherpa-onnx/streaming-zipformer-en-20M-2023-02-17" {
		t.Fatalf("Sherpa 20M runtime identity = %q/%q", model.Runtime, model.RuntimeModelID)
	}
	if model.Readiness != ReadinessCandidate || model.DefaultEligible() || model.HasRole("default") {
		t.Fatalf("Sherpa 20M candidate was promoted: readiness=%q roles=%v", model.Readiness, model.Roles)
	}
	if model.Artifact != nil {
		t.Fatalf("registry-only candidate unexpectedly claims acquired artifact %q/%q", model.Artifact.Resource, model.Artifact.Name)
	}
	if len(model.EngineIDs) != 0 {
		t.Fatalf("candidate unexpectedly enters engine selection before artifact/runtime support: %v", model.EngineIDs)
	}
	if model.SizeMBApprox != 44 || model.License != "Apache-2.0" || model.Commercial != CommercialYes {
		t.Fatalf("unexpected candidate footprint/license: size=%f license=%q commercial=%q", model.SizeMBApprox, model.License, model.Commercial)
	}
	got, err := c.DefaultFor(CapabilitySTT, "en", Platform{OS: "linux", Arch: "amd64", RAMGB: 32})
	if err != nil {
		t.Fatalf("existing STT default should remain resolvable: %v", err)
	}
	if got.ID != "whisper-small" {
		t.Fatalf("STT default = %q; adding an unevaluated Sherpa candidate changed it", got.ID)
	}
}

func TestBatchAdapterDeclarationsValidateProtocolRequirements(t *testing.T) {
	model := validModel("vllm-stt", CapabilitySTT)
	model.Runtime = "vllm"
	model.RuntimeModelID = "served-model"
	model.BatchAdapterID = "vllm-openai-chat-audio"
	if issues := New(fixture(model)).ValidationIssues(); hasIssueCode(issues, IssueInvalidBatchAdapter) {
		t.Fatalf("valid declared batch protocol has adapter validation issues: %v", issues)
	}

	model.BatchAdapterID = "unimplemented-protocol"
	issues := New(fixture(model)).ValidationIssues()
	if !hasIssueCode(issues, IssueInvalidBatchAdapter) {
		t.Fatalf("unknown batch protocol was accepted: %v", issues)
	}

	model.BatchAdapterID = "vllm-openai-chat-audio"
	model.RuntimeModelID = ""
	issues = New(fixture(model)).ValidationIssues()
	if !hasIssueCode(issues, IssueInvalidBatchAdapter) {
		t.Fatalf("protocol without runtime_model_id was accepted: %v", issues)
	}
}

func hasIssueCode(issues []Issue, code string) bool {
	for _, issue := range issues {
		if issue.Code == code {
			return true
		}
	}
	return false
}

func TestDefaultsArePromotedAndEligible(t *testing.T) {
	c := Default()
	for _, capability := range DefaultCapabilities {
		m, err := c.DefaultFor(capability, "", Platform{})
		if err != nil {
			t.Errorf("DefaultFor(%s): %v", capability, err)
			continue
		}
		if !m.HasRole("default") {
			t.Errorf("default for %s is %s, which does not carry the default role", capability, m.ID)
		}
		if !m.DefaultEligible() {
			t.Errorf("default for %s is %s, which is not default-eligible", capability, m.ID)
		}
	}
}

func TestCurrentDefaultsDeclareEvidenceLimitations(t *testing.T) {
	c := Default()
	for _, capability := range DefaultCapabilities {
		m, err := c.DefaultFor(capability, "", Platform{})
		if err != nil {
			t.Fatalf("DefaultFor(%s): %v", capability, err)
		}
		if m.Promotion.Qualified() {
			t.Errorf("%s is marked qualified without a linked promotion receipt", m.ID)
		}
		if strings.TrimSpace(m.Promotion.LegacyException) == "" {
			t.Errorf("operational default %s lacks explicit qualification limitation", m.ID)
		}
	}
}

func TestLegacyPromotionExceptionsAreLimitedToCurrentDefaults(t *testing.T) {
	c := Default()
	allowed := map[string]bool{
		"whisper-small":                  false,
		"sherpa-zipformer-en-2023-06-26": false,
		"kokoro-82m-v1.0":                false,
		"qwen3:1.7b":                     false,
		"sherpa-onnx-speaker-embedding":  false,
	}
	for _, model := range c.All() {
		if strings.TrimSpace(model.Promotion.LegacyException) == "" {
			continue
		}
		if _, ok := allowed[model.ID]; !ok || (!model.HasRole("default") && len(model.EngineDefaults) == 0) {
			t.Errorf("legacy promotion exception is not authorized for model %q", model.ID)
			continue
		}
		allowed[model.ID] = true
	}
	for modelID, seen := range allowed {
		if !seen {
			t.Errorf("current operational default %q is missing its explicit legacy exception", modelID)
		}
	}
}

func TestCommercialDefaultNeverNonCommercial(t *testing.T) {
	c := Default()
	for _, capability := range DefaultCapabilities {
		m, err := c.CommercialDefaultFor(capability, "", Platform{})
		if err != nil {
			continue // no commercial candidate is a valid, typed outcome
		}
		if m.Commercial != CommercialYes {
			t.Errorf("CommercialDefaultFor(%s) = %s with commercial_use=%s", capability, m.ID, m.Commercial)
		}
		if !m.DefaultEligible() {
			t.Errorf("CommercialDefaultFor(%s) returned default-ineligible %s", capability, m.ID)
		}
		if !m.HasRole("default") {
			t.Errorf("CommercialDefaultFor(%s) returned %s without the default role", capability, m.ID)
		}
	}
	for _, id := range []string{"reverb-tts", "verba-tts-indic-zero"} {
		m, ok := c.ByID(id)
		if !ok {
			t.Fatalf("%s record missing", id)
		}
		if m.Commercial == CommercialYes {
			t.Errorf("%s is marked commercial; this test expects a non-commercial/watch record", id)
		}
	}
	ttsDefault, err := c.CommercialDefaultFor(CapabilityTTS, "", Platform{})
	if err != nil {
		t.Fatalf("CommercialDefaultFor(tts): %v", err)
	}
	if ttsDefault.ID == "reverb-tts" || ttsDefault.ID == "verba-tts-indic-zero" {
		t.Fatalf("CommercialDefaultFor(tts) = %s, want a commercial record", ttsDefault.ID)
	}
}

func TestCommercialDefaultFailsClosedOnUnknown(t *testing.T) {
	m := validModel("unknown-tts", CapabilityTTS)
	m.Commercial = CommercialUnknown
	m.LicenseNotes = "licence not published"
	m.Readiness = ReadinessCandidate
	m.AllowDefault = true
	m.Roles = []string{"default"}

	c := New(fixture(m))
	if _, err := c.CommercialDefaultFor(CapabilityTTS, "", Platform{}); !errors.Is(err, ErrNoCommercialModel) {
		t.Fatalf("CommercialDefaultFor error = %v, want ErrNoCommercialModel", err)
	}
}

func TestSelectionIsDeterministic(t *testing.T) {
	c := Default()
	platform := Platform{OS: "linux", Arch: "amd64", Accelerated: true, FreeVRAMGB: 24, RAMGB: 32}
	want, err := c.DefaultFor(CapabilitySTT, "en", platform)
	if err != nil {
		t.Fatalf("DefaultFor: %v", err)
	}
	for i := 0; i < 100; i++ {
		got, err := c.DefaultFor(CapabilitySTT, "en", platform)
		if err != nil {
			t.Fatalf("iteration %d: %v", i, err)
		}
		if got.ID != want.ID {
			t.Fatalf("iteration %d selected %s, want %s", i, got.ID, want.ID)
		}
	}

	candidates := c.CandidatesFor(CapabilitySTT, "en", platform)
	again := c.CandidatesFor(CapabilitySTT, "en", platform)
	if len(candidates) != len(again) {
		t.Fatalf("candidate count changed between calls")
	}
	for i := range candidates {
		if candidates[i].ID != again[i].ID {
			t.Fatalf("candidate order changed at %d: %s vs %s", i, candidates[i].ID, again[i].ID)
		}
	}
}

func TestAvailableNonDefaultDoesNotBeatPromotedDefault(t *testing.T) {
	// An available record without the default role is an alternative, not a
	// default. The promoted candidate must win even though it is less ready.
	availableFast := validModel("available-fast", CapabilitySTT)
	availableFast.Roles = []string{"fast"}

	candidateDefault := validModel("candidate-default", CapabilitySTT)
	candidateDefault.Readiness = ReadinessCandidate
	candidateDefault.AllowDefault = true
	candidateDefault.Roles = []string{"default"}

	c := New(fixture(availableFast, candidateDefault))
	got, err := c.DefaultFor(CapabilitySTT, "", Platform{})
	if err != nil {
		t.Fatalf("DefaultFor: %v", err)
	}
	if got.ID != "candidate-default" {
		t.Fatalf("DefaultFor = %s, want candidate-default", got.ID)
	}
}

func TestMissingDefaultRoleReturnsError(t *testing.T) {
	onlyFast := validModel("only-fast", CapabilitySTT)
	onlyFast.Roles = []string{"fast"}

	c := New(fixture(onlyFast))
	if _, err := c.DefaultFor(CapabilitySTT, "", Platform{}); !errors.Is(err, ErrNoEligibleModel) {
		t.Fatalf("DefaultFor error = %v, want ErrNoEligibleModel", err)
	}
	if _, err := c.CommercialDefaultFor(CapabilitySTT, "", Platform{}); !errors.Is(err, ErrNoCommercialModel) {
		t.Fatalf("CommercialDefaultFor error = %v, want ErrNoCommercialModel", err)
	}
}

func TestDefaultForLanguage(t *testing.T) {
	c := Default()
	got, err := c.DefaultFor(CapabilityTTS, "ja", Platform{})
	if err != nil {
		t.Fatalf("DefaultFor(tts, ja): %v", err)
	}
	if !got.Speaks("ja") {
		t.Fatalf("DefaultFor(tts, ja) = %s which does not speak ja", got.ID)
	}
	if _, err := c.DefaultFor(CapabilityTTS, "xx", Platform{}); !errors.Is(err, ErrNoEligibleModel) {
		t.Fatalf("DefaultFor(tts, xx) error = %v, want ErrNoEligibleModel", err)
	}
	// Primary-subtag matching: en-US matches en.
	if _, err := c.DefaultFor(CapabilityTTS, "en-US", Platform{}); err != nil {
		t.Fatalf("DefaultFor(tts, en-US): %v", err)
	}
}

func TestEngineSelection(t *testing.T) {
	c := Default()
	sherpa := c.ByEngine("sherpa-streaming")
	if len(sherpa) != 2 {
		t.Fatalf("ByEngine(sherpa-streaming) = %v, want the Kroko default and the 2023 fallback", modelIDs(sherpa))
	}
	ids := map[string]bool{}
	for _, m := range sherpa {
		ids[m.ID] = true
	}
	for _, want := range []string{"sherpa-zipformer-en-kroko-2025-08-06", "sherpa-zipformer-en-2023-06-26"} {
		if !ids[want] {
			t.Fatalf("ByEngine(sherpa-streaming) = %v, missing %q", modelIDs(sherpa), want)
		}
	}
	got, err := c.DefaultForEngine(CapabilitySTT, "whisper-local", "en", Platform{})
	if err != nil {
		t.Fatalf("DefaultForEngine(whisper-local): %v", err)
	}
	if got.ID != "whisper-small" || !got.HasRole("default") {
		t.Fatalf("DefaultForEngine(whisper-local) = %s, want the resource-conscious default whisper-small", got.ID)
	}
	// Turbo remains the explicit accuracy tier.
	turbo, ok := c.ByID("whisper-large-v3-turbo")
	if !ok || !turbo.HasRole("quality") || turbo.HasRole("default") {
		t.Fatalf("whisper-large-v3-turbo must be a quality tier, not a default: %+v", turbo.Roles)
	}
	// Kyutai records exist, but no Kyutai model declares the default role, so
	// an engine-scoped default must fail rather than silently promote one.
	if _, err := c.DefaultForEngine(CapabilitySTT, "kyutai", "en", Platform{Accelerated: true, FreeVRAMGB: 24, RAMGB: 32}); !errors.Is(err, ErrNoEligibleModel) {
		t.Fatalf("DefaultForEngine(kyutai) error = %v, want ErrNoEligibleModel", err)
	}
	if _, err := c.DefaultForEngine(CapabilitySTT, "does-not-exist", "en", Platform{}); !errors.Is(err, ErrNoEligibleModel) {
		t.Fatalf("DefaultForEngine(unknown) error = %v, want ErrNoEligibleModel", err)
	}
}

func TestEngineScopedDefaultResolvesSelectedSherpaModel(t *testing.T) {
	c := Default()
	got, err := c.DefaultForEngine(CapabilitySTT, "sherpa-streaming", "en", Platform{OS: "linux", Arch: "amd64"})
	if err != nil {
		t.Fatalf("DefaultForEngine(sherpa-streaming): %v", err)
	}
	if got.ID != "sherpa-zipformer-en-2023-06-26" || !got.HasEngineDefault("sherpa-streaming") {
		t.Fatalf("Sherpa engine default = %q, want declared Zipformer model", got.ID)
	}
	commercial, err := c.CommercialDefaultForEngineVerified(CapabilitySTT, "sherpa-streaming", "en", Platform{OS: "linux", Arch: "amd64"}, nil)
	if err != nil || commercial.ID != got.ID {
		t.Fatalf("commercial Sherpa default = %q, %v; want %q", commercial.ID, err, got.ID)
	}
	if _, err := c.DefaultForEngine(CapabilitySTT, "sherpa-streaming", "es", Platform{}); !errors.Is(err, ErrNoEligibleModel) {
		t.Fatalf("Sherpa Spanish default error = %v, want no eligible model", err)
	}
	generic, err := c.DefaultFor(CapabilitySTT, "", Platform{})
	if err != nil || generic.ID != "whisper-small" {
		t.Fatalf("generic STT default = %q, %v; want multilingual Whisper Small", generic.ID, err)
	}
}

func TestEngineScopedDefaultWinsAndFailsClosedBeforeGenericDefault(t *testing.T) {
	generic := validModel("generic-engine-default", CapabilitySTT)
	generic.EngineIDs = []string{"shared-engine"}
	scoped := validModel("scoped-engine-default", CapabilitySTT)
	scoped.EngineIDs = []string{"shared-engine"}
	scoped.EngineDefaults = []string{"shared-engine"}
	scoped.Roles = []string{"fast"}
	c := New(Seed{Models: []Model{generic, scoped}})
	got, err := c.DefaultForEngine(CapabilitySTT, "shared-engine", "en", Platform{})
	if err != nil || got.ID != scoped.ID {
		t.Fatalf("engine default = %q, %v; want scoped model %q", got.ID, err, scoped.ID)
	}

	scoped.Readiness = ReadinessCandidate
	c = New(Seed{Models: []Model{generic, scoped}})
	if _, err := c.DefaultForEngine(CapabilitySTT, "shared-engine", "en", Platform{}); !errors.Is(err, ErrNoEligibleModel) {
		t.Fatalf("unqualified scoped default fell back to generic default: %v", err)
	}

	scoped = validModel("scoped-engine-default", CapabilitySTT)
	scoped.EngineIDs = []string{"shared-engine"}
	scoped.EngineDefaults = []string{"shared-engine"}
	scoped.Roles = []string{"fast"}
	scoped.Commercial = CommercialUnknown
	c = New(Seed{Models: []Model{generic, scoped}})
	if _, err := c.CommercialDefaultForEngineVerified(CapabilitySTT, "shared-engine", "en", Platform{}, nil); !errors.Is(err, ErrNoCommercialModel) {
		t.Fatalf("commercial engine default fell back to a different model: %v", err)
	}
}

func TestCandidatesForKeepsNonCommercialAlternatives(t *testing.T) {
	c := Default()
	candidates := c.CandidatesFor(CapabilityTTS, "en", Platform{})
	var sawReverb, sawVerba bool
	for _, m := range candidates {
		switch m.ID {
		case "reverb-tts":
			sawReverb = true
		case "verba-tts-indic-zero":
			sawVerba = true
		}
	}
	if !sawReverb {
		t.Error("CandidatesFor(tts, en) omitted reverb-tts; candidates are an inventory, not a commercial filter")
	}
	if !sawVerba {
		t.Error("CandidatesFor(tts, en) omitted verba-tts-indic-zero")
	}
}

func TestDocsMatchRenderer(t *testing.T) {
	want := Default().RenderMarkdown()
	path := filepath.Join("..", "..", "..", "docs", "reference", "models.md")
	raw, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v (regenerate with: go run ./internal/modelcatalog/cmd/render-models)", path, err)
	}
	if string(raw) != want {
		t.Fatalf("%s is stale; regenerate with: go run ./internal/modelcatalog/cmd/render-models", path)
	}
}

func TestRemoteSTTModelMustNotDeclareEngine(t *testing.T) {
	m := validModel("remote-stt", CapabilitySTT)
	m.Runtime = "byok"
	m.EngineIDs = []string{"remote-engine"}

	if !hasIssue(New(fixture(m)).ValidationIssues(), IssueUnexpectedEngineID, "remote-stt") {
		t.Fatal("expected an unexpected_engine_id issue for a chain-routed STT model")
	}
}

func fixture(models ...Model) Seed {
	return Seed{
		SchemaVersion: SupportedSchemaVersion,
		Generated:     "test",
		KnownProviderIDs: []string{
			"whisper-stt", "speaker-verification",
		},
		SelectionPolicy: SelectionPolicy{
			ReadinessRank:    []string{"available", "candidate", "experimental", "watch"},
			RoleOrder:        append([]string(nil), Roles...),
			LanguageFallback: "none",
		},
		Models: models,
	}
}

func validModel(id string, capability Capability) Model {
	promotion := PromotionEvidence{
		CorpusQuality: EvidenceFacet{Status: "qualified", Ref: "test:corpus"},
		Latency:       EvidenceFacet{Status: "qualified", Ref: "test:latency"},
		Memory:        EvidenceFacet{Status: "qualified", Ref: "test:memory"},
		Platform:      EvidenceFacet{Status: "qualified", Ref: "test:platform"},
		License:       EvidenceFacet{Status: "qualified", Ref: "test:license"},
		verified:      true,
	}
	m := Model{
		ID:           id,
		Name:         id,
		ProviderIDs:  []string{"whisper-stt"},
		Capability:   capability,
		Runtime:      "whisper.cpp",
		Languages:    []string{"en"},
		SizeMBApprox: 100,
		Params:       "1M",
		License:      "MIT",
		Commercial:   CommercialYes,
		Readiness:    ReadinessAvailable,
		Roles:        []string{"default"},
		Promotion:    promotion,
		Hardware: Hardware{
			CPUCapable: true,
			OSArch:     []string{"linux/amd64"},
		},
		Notes: "fixture",
	}
	switch capability {
	case CapabilitySTT:
		m.EngineIDs = []string{"whisper-local"}
	case CapabilityTTS:
		m.Runtime = "kokoro"
	case CapabilitySummarize:
		m.Runtime = "ollama"
	case CapabilitySpeaker:
		m.Runtime = "sherpa-onnx"
	case CapabilityDuplex:
		m.Runtime = "byok"
		m.ProviderIDs = []string{"openai-gpt-live"}
	}
	return m
}

func modelIDs(models []Model) []string {
	out := make([]string, len(models))
	for i, m := range models {
		out[i] = m.ID
	}
	return out
}

func hasIssue(issues []Issue, code, modelID string) bool {
	for _, issue := range issues {
		if issue.Code == code && issue.ModelID == modelID {
			return true
		}
	}
	return false
}
