package modelcatalog

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommercialDefaultExplainsHostMemoryMismatch(t *testing.T) {
	_, err := Default().CommercialDefaultFor(CapabilitySummarize, "en", Platform{
		OS: "linux", Arch: "amd64", RAMGB: 4, CPULimit: 2,
	})
	if err == nil {
		t.Fatal("expected the 4 GB host to reject the 6 GB Qwen3 1.7B minimum")
	}
	if errors.Is(err, ErrNoCommercialModel) {
		t.Fatalf("host capacity failure was misreported as a licensing gap: %v", err)
	}
	if !errors.Is(err, ErrNoEligibleModel) {
		t.Fatalf("error = %v, want ErrNoEligibleModel", err)
	}
	if !strings.Contains(err.Error(), "qwen3:1.7b") || !strings.Contains(err.Error(), "requires at least 6 GB RAM; host reports 4 GB") {
		t.Fatalf("error does not explain the selected model's host constraint: %v", err)
	}
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
		} else if model.ID < lastID {
			t.Fatalf("ids not ascending within %s at %s", model.Capability, model.ID)
		}
		lastID = model.ID
	}
}

func TestDefaultsByCapabilityResolvesOneDefaultPerCapability(t *testing.T) {
	c := Default()
	defaults := c.DefaultsByCapability(Platform{})
	if len(defaults) != len(DefaultCapabilities) {
		t.Fatalf("defaults for %d capabilities, want %d", len(defaults), len(DefaultCapabilities))
	}
	for _, capability := range DefaultCapabilities {
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
	for _, capability := range DefaultCapabilities {
		if first[capability].ID != second[capability].ID {
			t.Fatalf("capability %s default changed between calls: %s vs %s",
				capability, first[capability].ID, second[capability].ID)
		}
	}
}

// TestSummarizeRecordsMirrorResourceOllamaPolicy keeps the registry honest
// about what the summarize path actually runs. resource-ollama owns the role
// policy; this test reads it and fails when the concrete models behind
// summarize.audio_tools and summarize.default (or a declared fallback) have no record. The scenario can
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
	if primary.ID != "qwen3.5:4b" || primary.HasRole("default") || !primary.HasRole("quality") {
		t.Fatalf("generic summarize.default model = %q roles %v; want 4B quality alternative, not Audio Tools default", primary.ID, primary.Roles)
	}

	// The model registry describes the Audio Tools product route. Keep its
	// compact model default-eligible through the controlled caveat, while the
	// shared generic role above remains on the 4B quality alternative.
	audioRole, ok := policy.Roles["summarize.audio_tools"]
	if !ok {
		t.Fatal("resource-ollama policy has no summarize.audio_tools role")
	}
	compact, ok := c.ByID(audioRole.Model)
	if !ok {
		t.Fatalf("summarize.audio_tools resolves to %q, which the registry does not declare", audioRole.Model)
	}
	if compact.ID != "qwen3:1.7b" || !compact.HasRole("fast") || !compact.HasRole("default") {
		t.Fatalf("Audio Tools compact model = %q roles %v; want qwen3:1.7b default and fast", compact.ID, compact.Roles)
	}
	if audioRole.Model == defaultRole.Model {
		t.Fatalf("Audio Tools compact role unexpectedly shares generic summarize.default model %q", audioRole.Model)
	}
	if compact.Promotion.Qualified() || !compact.DefaultEligible() || !legacyExceptionAllowed(compact.ID, compact.Promotion.LegacyException) {
		t.Fatalf("Audio Tools compact default must be eligible only through its explicit, unqualified evidence exception: %+v", compact.Promotion)
	}
	resolved, err := c.DefaultFor(CapabilitySummarize, "", Platform{})
	if err != nil || resolved.ID != compact.ID {
		t.Fatalf("registry summarize default = %v, %v; want Audio Tools role %q", resolved, err, compact.ID)
	}
	commercial, err := c.CommercialDefaultFor(CapabilitySummarize, "", Platform{})
	if err != nil || commercial.ID != compact.ID {
		t.Fatalf("commercial registry summarize default = %v, %v; want Audio Tools role %q", commercial, err, compact.ID)
	}
	for _, tag := range audioRole.Fallbacks {
		_, ok := c.ByID(tag)
		if !ok {
			t.Errorf("summarize.audio_tools fallback %q has no registry record", tag)
		}
	}

	for _, tag := range defaultRole.Fallbacks {
		_, ok := c.ByID(tag)
		if !ok {
			t.Errorf("summarize.default fallback %q has no registry record", tag)
			continue
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
	Platforms      map[string]string `json:"platforms"`
	ManagedService struct {
		DataArtifacts []struct {
			Name            string `json:"name"`
			Path            string `json:"path"`
			AcquisitionMode string `json:"acquisition_mode"`
		} `json:"data_artifacts"`
		Environment map[string]string `json:"environment"`
	} `json:"managed_service"`
}

// TestKaleModelsPlatformMatchesManagedResource prevents the model registry
// from advertising Kale models on hosts where their declared resource cannot
// run. Tiny/Base remain candidates without runtime artifacts, so their host
// envelope is deliberately bounded to the only supported provider platform.
func TestKaleModelsPlatformMatchesManagedResource(t *testing.T) {
	c := Default()
	manifest := loadResourceManifest(t, "kale-tts")
	models := c.ByProvider("kale-tts")
	if len(models) == 0 {
		t.Fatal("Kale provider has no model records")
	}
	for _, m := range models {
		if len(m.Hardware.OSArch) == 0 {
			t.Errorf("Kale model %q has no bounded host platform", m.ID)
			continue
		}
		for _, host := range m.Hardware.OSArch {
			osName, _, ok := strings.Cut(host, "/")
			if !ok {
				t.Errorf("Kale model %q has malformed hardware platform %q", m.ID, host)
				continue
			}
			if manifest.Platforms[osName] == "unsupported" {
				t.Errorf("Kale model %q claims %q but resource marks %s unsupported", m.ID, host, osName)
			}
		}
	}
	// The resource manifest currently supports Linux as an OS, but its composed
	// Python runtime and pinned artifact are Linux/amd64 only. Keep Linux/arm64
	// in this negative matrix so an OS-only check cannot accidentally advertise
	// Kale on an unsupported architecture.
	for _, platform := range []Platform{
		{OS: "darwin", Arch: "arm64"},
		{OS: "windows", Arch: "amd64"},
		{OS: "linux", Arch: "arm64"},
	} {
		for _, candidate := range c.CandidatesFor(CapabilityTTS, "en", platform) {
			for _, providerID := range candidate.ProviderIDs {
				if providerID == "kale-tts" {
					t.Errorf("Kale model %q remains selectable on unsupported platform %s/%s", candidate.ID, platform.OS, platform.Arch)
				}
			}
		}
	}
}

func TestKaleNanoLicenseUpdateDoesNotPromoteUnqualifiedModel(t *testing.T) {
	model, ok := Default().ByID("kale-tts-1-nano")
	if !ok {
		t.Fatal("Kale Nano model record missing")
	}
	if model.Commercial != CommercialYes || model.License != "Apache-2.0" {
		t.Fatalf("Kale Nano current license declaration = %q / %q, want Apache-2.0 / commercial allowed", model.License, model.Commercial)
	}
	if model.Readiness != ReadinessCandidate || model.HasRole("default") || model.AllowDefault || model.Promotion.Qualified() {
		t.Fatalf("license metadata incorrectly promoted Kale Nano: readiness=%q roles=%v allow_default=%v evidence_qualified=%v", model.Readiness, model.Roles, model.AllowDefault, model.Promotion.Qualified())
	}
	if !strings.Contains(model.Notes, "Conditional product auto route") || !strings.Contains(model.Notes, "complete installed model bundle is about 20 MB") {
		t.Fatalf("Kale Nano notes must describe its conditional route and installed bundle size: %q", model.Notes)
	}
	kokoro, ok := Default().ByID("kokoro-82m-v1.0")
	if !ok || !kokoro.HasRole("default") || !strings.Contains(kokoro.Notes, "Qualified catalog default") {
		t.Fatalf("Kokoro must remain the clearly declared catalog default, got %#v", kokoro)
	}
	if !strings.Contains(kokoro.Notes, "Product auto mode tries the conditional Kale Nano route first when healthy, with Kokoro as the availability fallback") {
		t.Fatalf("Kokoro notes must describe the conditional product route accurately: %q", kokoro.Notes)
	}
}

func TestLocalTTSRuntimeProtocolNamesAreDeclared(t *testing.T) {
	c := Default()
	for providerID, want := range map[string]string{
		"kokoro-tts": "kokoro",
		"kale-tts":   "kale-tts-1-nano",
	} {
		models := c.ByProvider(providerID)
		found := false
		for _, model := range models {
			if model.Capability == CapabilityTTS && model.RuntimeModelID == want {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("provider %q has no TTS model declaring runtime_model_id %q", providerID, want)
		}
	}
}

func TestQwenVoiceCloneCandidateUsesDocumentedVRAMEstimate(t *testing.T) {
	model, ok := Default().ByID("qwen3-tts-12hz-0.6b-base")
	if !ok {
		t.Fatal("Qwen voice-cloning candidate missing from model registry")
	}
	if model.Hardware.MinVRAMGB != 5 {
		t.Fatalf("Qwen candidate VRAM estimate = %v GiB, want the explicitly documented tuned-profile estimate", model.Hardware.MinVRAMGB)
	}
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
		if runtimeLookup()[m.Runtime].ArtifactPolicy != RuntimeArtifactRequired {
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

// TestDeclaredCandidateArtifactsResolve checks that candidate records can
// carry concrete acquisition provenance without being promoted to available.
func TestDeclaredCandidateArtifactsResolve(t *testing.T) {
	c := Default()
	checked := 0
	for _, m := range c.All() {
		if m.Artifact == nil {
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
			t.Errorf("model %q names artifact %q, which resources/%s does not declare", m.ID, m.Artifact.Name, m.Artifact.Resource)
		}
		checked++
	}
	if checked == 0 {
		t.Fatal("no model artifacts were checked")
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
