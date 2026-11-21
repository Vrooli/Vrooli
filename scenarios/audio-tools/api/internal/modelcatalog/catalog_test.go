package modelcatalog

import (
	"encoding/json"
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

func TestDefaultMapsToKnownProvider(t *testing.T) {
	c := Default()
	known := stringSet(c.seed.KnownProviderIDs)
	for _, capability := range Capabilities {
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
	// Chain-routed STT providers and batch adapters with no engine do not
	// declare an engine id: no sttengine provider owns one.
	for _, id := range []string{"openai-whisper-1", "deepgram-nova-3", "rumble-2-flash"} {
		m, ok := c.ByID(id)
		if !ok {
			t.Fatalf("%s record missing", id)
		}
		if len(m.EngineIDs) != 0 {
			t.Errorf("%s engine ids = %v, want none", id, m.EngineIDs)
		}
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

func TestDefaultsArePromotedAndEligible(t *testing.T) {
	c := Default()
	for _, capability := range Capabilities {
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

func TestCommercialDefaultNeverNonCommercial(t *testing.T) {
	c := Default()
	for _, capability := range Capabilities {
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
		m.ProviderIDs = []string{"speaker-verification"}
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

func TestSortedForDisplayGroupsByCanonicalCapabilityThenID(t *testing.T) {
	c := Default()
	got := c.SortedForDisplay()
	if len(got) != len(c.All()) {
		t.Fatalf("SortedForDisplay returned %d records, want %d", len(got), len(c.All()))
	}
	lastOrder := -1
	lastID := ""
	for i, model := range got {
		order, ok := CapabilityOrder(model.Capability)
		if !ok {
			t.Fatalf("record %d (%s) has unknown capability %q", i, model.ID, model.Capability)
		}
		if order < lastOrder {
			t.Fatalf("capability order regressed at %s", model.ID)
		}
		if order > lastOrder {
			lastOrder = order
			lastID = ""
		} else if model.ID < lastID {
			t.Fatalf("ids not ascending within %s at %s", model.Capability, model.ID)
		}
		lastID = model.ID
	}
}

func TestDefaultsByCapabilityResolvesOneDefaultPerCapability(t *testing.T) {
	c := Default()
	defaults := c.DefaultsByCapability(Platform{})
	if len(defaults) != len(Capabilities) {
		t.Fatalf("defaults for %d capabilities, want %d", len(defaults), len(Capabilities))
	}
	for _, capability := range Capabilities {
		model, ok := defaults[capability]
		if !ok {
			t.Fatalf("capability %s has no default", capability)
		}
		if model.Capability != capability {
			t.Fatalf("default %s has capability %s, want %s", model.ID, model.Capability, capability)
		}
		if !model.HasRole("default") {
			t.Fatalf("default %s does not declare the default role", model.ID)
		}
	}
}

func TestDefaultsByCapabilityIsDeterministic(t *testing.T) {
	c := Default()
	first := c.DefaultsByCapability(Platform{})
	second := c.DefaultsByCapability(Platform{})
	for _, capability := range Capabilities {
		if first[capability].ID != second[capability].ID {
			t.Fatalf("capability %s default changed between calls: %s vs %s",
				capability, first[capability].ID, second[capability].ID)
		}
	}
}

// TestSummarizeRecordsMirrorResourceOllamaPolicy keeps the registry honest
// about what the summarize path actually runs. resource-ollama owns the role
// policy; this test reads it and fails when the concrete model behind
// summarize.default (or a declared fallback) has no record. The scenario can
// be checked out without the resources tree, so a missing policy is a skip,
// not a failure.
func TestSummarizeRecordsMirrorResourceOllamaPolicy(t *testing.T) {
	path := filepath.Join("..", "..", "..", "..", "..", "resources", "ollama", "model-policy.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("resource-ollama policy not present at %s", path)
		}
		t.Fatalf("read resource-ollama policy: %v", err)
	}
	var policy struct {
		Roles map[string]struct {
			Model     string   `json:"model"`
			Fallbacks []string `json:"fallbacks"`
		} `json:"roles"`
	}
	if err := json.Unmarshal(raw, &policy); err != nil {
		t.Fatalf("parse resource-ollama policy: %v", err)
	}

	c := Default()
	defaultRole, ok := policy.Roles["summarize.default"]
	if !ok {
		t.Fatal("resource-ollama policy has no summarize.default role")
	}
	primary, ok := c.ByID(defaultRole.Model)
	if !ok {
		t.Fatalf("summarize.default resolves to %q, which the registry does not declare", defaultRole.Model)
	}
	if primary.Readiness != ReadinessAvailable {
		t.Fatalf("summarize.default model %q readiness=%s, want available", primary.ID, primary.Readiness)
	}
	if !primary.HasRole("default") {
		t.Fatalf("summarize.default model %q does not carry the default role", primary.ID)
	}
	resolved, err := c.DefaultFor(CapabilitySummarize, "", Platform{})
	if err != nil {
		t.Fatalf("DefaultFor(summarize): %v", err)
	}
	if resolved.ID != primary.ID {
		t.Fatalf("registry default %q does not match resource-ollama summarize.default %q", resolved.ID, primary.ID)
	}

	for _, tag := range defaultRole.Fallbacks {
		if _, ok := c.ByID(tag); !ok {
			t.Errorf("summarize.default fallback %q has no registry record", tag)
		}
	}
	if chat, ok := policy.Roles["chat.small"]; ok {
		if _, ok := c.ByID(chat.Model); !ok {
			t.Errorf("chat.small model %q has no registry record", chat.Model)
		}
		for _, tag := range chat.Fallbacks {
			if _, ok := c.ByID(tag); !ok {
				t.Errorf("chat.small fallback %q has no registry record", tag)
			}
		}
	}
}

// resourceManifest is the subset of resources/<slug>/resource.json this
// package's conformance tests read. The manifest owns acquisition facts; the
// registry must not copy them.
type resourceManifest struct {
	ManagedService struct {
		DataArtifacts []struct {
			Name            string `json:"name"`
			Path            string `json:"path"`
			AcquisitionMode string `json:"acquisition_mode"`
		} `json:"data_artifacts"`
		Environment map[string]string `json:"environment"`
	} `json:"managed_service"`
}

func loadResourceManifest(t *testing.T, slug string) resourceManifest {
	t.Helper()
	path := filepath.Join("..", "..", "..", "..", "..", "resources", slug, "resource.json")
	raw, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			t.Skipf("resource manifest not present at %s (scenario-only checkout)", path)
		}
		t.Fatalf("read %s: %v", path, err)
	}
	var manifest resourceManifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		t.Fatalf("parse %s: %v", path, err)
	}
	if len(manifest.ManagedService.DataArtifacts) == 0 {
		t.Fatalf("%s declares no data artifacts; a local model cannot be available", path)
	}
	return manifest
}

// TestAvailableLocalModelsNameShippedResourceArtifacts closes the
// "registry says available, resource does not ship it" class. Every available
// local model must name a data artifact that its resource manifest declares.
// The same gate runs against whisper, kyutai, sherpa-onnx, and kokoro records.
func TestAvailableLocalModelsNameShippedResourceArtifacts(t *testing.T) {
	c := Default()
	checked := 0
	for _, m := range c.All() {
		if m.Readiness != ReadinessAvailable {
			continue
		}
		if _, local := artifactProvidingRuntimes[m.Runtime]; !local {
			continue
		}
		if m.Artifact == nil {
			t.Errorf("model %q is available with local runtime %q but names no artifact", m.ID, m.Runtime)
			continue
		}
		manifest := loadResourceManifest(t, m.Artifact.Resource)
		found := false
		for _, artifact := range manifest.ManagedService.DataArtifacts {
			if artifact.Name == m.Artifact.Name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("model %q names artifact %q, which resources/%s does not declare",
				m.ID, m.Artifact.Name, m.Artifact.Resource)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no available local models were checked; the artifact gate is not exercising the registry")
	}
}

// TestWhisperRuntimeArtifactMatchesRegistryDefault keeps the model the
// whisper resource actually starts in lockstep with the registry's declared
// default for whisper-local. This is the runtime-truth seam: a seed edit that
// promotes one model while the resource pins another fails the build, and so
// does making the pinned artifact on-demand.
func TestWhisperRuntimeArtifactMatchesRegistryDefault(t *testing.T) {
	manifest := loadResourceManifest(t, "whisper")
	modelPath := manifest.ManagedService.Environment["WHISPER_MODEL_PATH"]
	if modelPath == "" {
		t.Fatal("resources/whisper does not pin WHISPER_MODEL_PATH")
	}
	const dataPrefix = "${RESOURCE_DATA_DIR}/"
	if !strings.HasPrefix(modelPath, dataPrefix) {
		t.Fatalf("WHISPER_MODEL_PATH=%q does not resolve under RESOURCE_DATA_DIR", modelPath)
	}
	wantPath := strings.TrimPrefix(modelPath, dataPrefix)

	var runtimeArtifact string
	for _, artifact := range manifest.ManagedService.DataArtifacts {
		if artifact.Path != wantPath {
			continue
		}
		runtimeArtifact = artifact.Name
		if artifact.AcquisitionMode == "on-demand" {
			t.Errorf("runtime default artifact %q is on-demand; a fresh install would not provide it", artifact.Name)
		}
	}
	if runtimeArtifact == "" {
		t.Fatalf("WHISPER_MODEL_PATH=%q does not match any declared data artifact path", modelPath)
	}

	resolved, err := Default().DefaultForEngine(CapabilitySTT, "whisper-local", "", Platform{})
	if err != nil {
		t.Fatalf("DefaultForEngine(stt, whisper-local): %v", err)
	}
	if resolved.Artifact == nil {
		t.Fatalf("registry default %q names no artifact", resolved.ID)
	}
	if resolved.Artifact.Name != runtimeArtifact {
		t.Fatalf("registry default %q ships artifact %q, but resources/whisper pins %q",
			resolved.ID, resolved.Artifact.Name, runtimeArtifact)
	}
}
