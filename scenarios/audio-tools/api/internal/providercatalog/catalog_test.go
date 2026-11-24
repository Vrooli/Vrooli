package providercatalog_test

import (
	"encoding/json"
	"os"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/providercatalog"
)

func TestEmbeddedCatalogLoads(t *testing.T) {
	reg := providercatalog.Default()
	require.NotNil(t, reg)
	require.Equal(t, "1.0.0", reg.SchemaVersion())
	require.NotEmpty(t, reg.All())
	require.Len(t, reg.Active(), 15)
}

func TestProviderSeedRejectsDuplicatedModelFacts(t *testing.T) {
	provider := baseProvider()
	provider["models"] = []string{"whisper-medium"}
	raw, err := json.Marshal(map[string]any{"schema_version": "1.0.0", "providers": []any{provider}})
	require.NoError(t, err)
	_, err = providercatalog.LoadBytes(raw)
	require.ErrorContains(t, err, "unknown field \"models\"")
}

func TestProviderHealthDeclarationValidation(t *testing.T) {
	readSeed := func() map[string]any {
		raw, err := os.ReadFile("catalog.seed.json")
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, json.Unmarshal(raw, &doc))
		return doc
	}
	providerIn := func(doc map[string]any, id string) map[string]any {
		for _, value := range doc["providers"].([]any) {
			provider := value.(map[string]any)
			if provider["id"] == id {
				return provider
			}
		}
		t.Fatalf("%s missing from fixture", id)
		return nil
	}
	check := func(doc map[string]any, want string) {
		raw, err := json.Marshal(doc)
		require.NoError(t, err)
		_, err = providercatalog.LoadBytes(raw)
		require.ErrorContains(t, err, want)
	}
	doc := readSeed()
	providerIn(doc, "whisper-stt")["health_profile"] = "unknown-profile"
	check(doc, "unknown health_profile")
	doc = readSeed()
	providerIn(doc, "whisper-stt")["health_profile"] = ""
	check(doc, "aggregate membership without health_profile")
	doc = readSeed()
	providerIn(doc, "kale-tts")["health_profile"] = string(providercatalog.HealthProfileHTTP)
	providerIn(doc, "kale-tts")["health_probe"] = map[string]any{
		"base_url_field": "SherpaURL", "readiness_path": "/ready", "liveness_path": "/ready",
	}
	providerIn(doc, "kale-tts")["liveness_aggregate"] = true
	check(doc, "non-active provider")
	doc = readSeed()
	providerIn(doc, "sherpa-stt")["health_probe"].(map[string]any)["readiness_path"] = "//attacker.invalid/health"
	check(doc, "clean absolute path")
	doc = readSeed()
	providerIn(doc, "sherpa-stt")["health_probe"].(map[string]any)["base_url_field"] = "OPENAI_API_KEY"
	check(doc, "unsupported base_url_field")
	doc = readSeed()
	delete(providerIn(doc, "sherpa-stt"), "health_probe")
	check(doc, "requires health_probe")
}

func TestTTSRuntimeAutoPriorityMustBeUnique(t *testing.T) {
	raw, err := os.ReadFile("catalog.seed.json")
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	providers := doc["providers"].([]any)
	var kale, kokoro map[string]any
	for _, value := range providers {
		provider := value.(map[string]any)
		switch provider["id"] {
		case "kale-tts":
			kale = provider
		case "kokoro-tts":
			kokoro = provider
		}
	}
	require.NotNil(t, kale)
	require.NotNil(t, kokoro)
	kale["tts_runtime"].(map[string]any)["auto_priority"] = kokoro["tts_runtime"].(map[string]any)["auto_priority"]
	duplicate, err := json.Marshal(doc)
	require.NoError(t, err)
	_, err = providercatalog.LoadBytes(duplicate)
	require.ErrorContains(t, err, "TTS auto_priority")
}

func TestTTSRuntimeRejectsUnknownSpeedMode(t *testing.T) {
	raw, err := os.ReadFile("catalog.seed.json")
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	providers := doc["providers"].([]any)
	var kale map[string]any
	for _, value := range providers {
		provider := value.(map[string]any)
		if provider["id"] == "kale-tts" {
			kale = provider
			break
		}
	}
	require.NotNil(t, kale)
	kale["tts_runtime"].(map[string]any)["speed_mode"] = "guess"
	invalid, err := json.Marshal(doc)
	require.NoError(t, err)
	_, err = providercatalog.LoadBytes(invalid)
	require.ErrorContains(t, err, `unsupported TTS speed_mode "guess"`)
	kale["tts_runtime"].(map[string]any)["speed_mode"] = ""
	missing, err := json.Marshal(doc)
	require.NoError(t, err)
	_, err = providercatalog.LoadBytes(missing)
	require.ErrorContains(t, err, "speed_mode is required")
}

func TestAccessors(t *testing.T) {
	whisper, ok := providercatalog.ByID("whisper-stt")
	require.True(t, ok)
	require.Equal(t, "whisper", whisper.ResourceSlug)
	require.Equal(t, providercatalog.TierLocal, providercatalog.TierForProviderID("whisper-stt"))
	require.Equal(t, providercatalog.TierBYOK, providercatalog.TierForProviderID("openai-tts"))
	duplex, ok := providercatalog.ByID("openai-gpt-live")
	require.True(t, ok)
	require.Equal(t, providercatalog.KindFullDuplex, duplex.Kind)
	require.Equal(t, providercatalog.StatusPlanned, duplex.Status)
	require.Contains(t, duplex.Capabilities, providercatalog.CapabilityDuplex)
	require.Equal(t, providercatalog.TierBYOK, duplex.Tier)
	require.Equal(t, providercatalog.TierBrowser, providercatalog.TierForProviderID("browser-stt"))
	require.Equal(t, providercatalog.TierNone, providercatalog.TierForProviderID("audio-tools"))
	require.Equal(t, providercatalog.TierNone, providercatalog.TierForProviderID("unknown"))

	slug, ok := providercatalog.ResourceSlugForProviderID("speaker-verification")
	require.True(t, ok)
	require.Equal(t, "sherpa-onnx", slug)
	require.True(t, providercatalog.IsLocalProvider("sherpa-stt"))
	require.False(t, providercatalog.IsLocalProvider("openai-tts"))
	require.False(t, providercatalog.IsLocalProvider("rumble-stt"), "planned providers are not lifecycle targets")

	require.True(t, providercatalog.SupportsPullModel("ollama"))
	require.False(t, providercatalog.SupportsPullModel("whisper-stt"))
	require.Equal(t, []providercatalog.Action{
		providercatalog.ActionStart, providercatalog.ActionStop, providercatalog.ActionRestart, providercatalog.ActionViewLogs,
	}, providercatalog.ActionsForProvider("whisper-stt"))
	require.Contains(t, providercatalog.ActionsForProvider("ollama"), providercatalog.ActionPullModel)
	require.Nil(t, providercatalog.ActionsForProvider("openai-tts"))

	actions := providercatalog.ActionsForProvider("ollama")
	actions[0] = providercatalog.ActionStop
	require.Equal(t, providercatalog.ActionStart, providercatalog.ActionsForProvider("ollama")[0], "actions must be a copy")
}

func TestFullDuplexConsentRequirementIsDeclaredAndCopied(t *testing.T) {
	requirement, ok := providercatalog.ConsentRequirementFor("openai-gpt-live")
	require.True(t, ok)
	require.Equal(t, providercatalog.ConsentRequirement{
		Source: "audio-tools-openai-gpt-live", RevalidateActiveSessions: true,
	}, requirement)
	provider, ok := providercatalog.ByID("openai-gpt-live")
	require.True(t, ok)
	require.NotNil(t, provider.ConsentRequirement)
	*provider.ConsentRequirement = providercatalog.ConsentRequirement{Source: "mutated"}
	copy, ok := providercatalog.ByID("openai-gpt-live")
	require.True(t, ok)
	require.Equal(t, providercatalog.ConsentSource("audio-tools-openai-gpt-live"), copy.ConsentRequirement.Source)
}

func TestFullDuplexConsentRequirementIsMandatoryAndValid(t *testing.T) {
	readSeed := func() map[string]any {
		raw, err := os.ReadFile("catalog.seed.json")
		require.NoError(t, err)
		var doc map[string]any
		require.NoError(t, json.Unmarshal(raw, &doc))
		return doc
	}
	providerIn := func(doc map[string]any) map[string]any {
		for _, value := range doc["providers"].([]any) {
			provider := value.(map[string]any)
			if provider["id"] == "openai-gpt-live" {
				return provider
			}
		}
		t.Fatal("openai-gpt-live missing from fixture")
		return nil
	}
	check := func(doc map[string]any, want string) {
		raw, err := json.Marshal(doc)
		require.NoError(t, err)
		_, err = providercatalog.LoadBytes(raw)
		require.ErrorContains(t, err, want)
	}
	doc := readSeed()
	delete(providerIn(doc), "consent_requirement")
	check(doc, "must declare consent_requirement")
	doc = readSeed()
	providerIn(doc)["consent_requirement"].(map[string]any)["source"] = "  "
	check(doc, "blank or invalid consent source")
	doc = readSeed()
	providerIn(doc)["consent_requirement"].(map[string]any)["revalidate_active_sessions"] = false
	check(doc, "must revalidate consent for active sessions")
}

func TestCapabilityForFeature(t *testing.T) {
	cases := map[string]providercatalog.Capability{
		"voice-input":                providercatalog.CapabilitySTT,
		"voice-streaming":            providercatalog.CapabilitySTT,
		"voice-streaming-realtime":   providercatalog.CapabilitySTT,
		"voice-speaker-verification": providercatalog.CapabilitySpeaker,
		"voice-enrollment":           providercatalog.CapabilitySpeaker,
		"voice-output":               providercatalog.CapabilityTTS,
		"voice-cloning":              providercatalog.CapabilityTTS,
		"tts-summarization":          providercatalog.CapabilitySummarize,
		"tts-cache":                  providercatalog.CapabilityTTS,
		"tts-paragraph-split":        providercatalog.CapabilityTTS,
		"audio-provider-routing":     providercatalog.CapabilitySTT,
		"ai-command-generation":      providercatalog.CapabilitySummarize,
		"transcode":                  providercatalog.CapabilityTranscode,
	}
	for feature, want := range cases {
		got, ok := providercatalog.CapabilityForFeature(feature)
		require.True(t, ok, feature)
		require.Equal(t, want, got, feature)
	}
	_, ok := providercatalog.CapabilityForFeature("not-a-feature")
	require.False(t, ok)
}

func TestByCapabilityReturnsOnlyActiveProviders(t *testing.T) {
	tts := providercatalog.ByCapability(providercatalog.CapabilityTTS)
	ids := make([]string, 0, len(tts))
	for _, p := range tts {
		ids = append(ids, p.ID)
	}
	require.Equal(t, []string{"audio-tools", "kokoro-tts", "openai-tts", "elevenlabs", "browser-tts"}, ids)

	stt := providercatalog.ByCapability(providercatalog.CapabilitySTT)
	for _, p := range stt {
		require.NotEqual(t, "rumble-stt", p.ID, "planned entries are not capability providers")
	}
}

func TestActiveExcludesPlannedWatchAndTestOnlyEngines(t *testing.T) {
	active := make(map[string]struct{})
	for _, p := range providercatalog.Active() {
		active[p.ID] = struct{}{}
	}
	for _, id := range []string{"rumble-stt", "kale-tts", "reverb-tts", "verba-tts", "virtual-replay"} {
		_, found := active[id]
		require.False(t, found, "%q must not be active", id)
	}
	_, ok := providercatalog.ProviderIDForEngine("virtual-replay")
	require.False(t, ok, "test-only engines must not map to a production provider")
}

func TestEngineProviderMappingIsBidirectional(t *testing.T) {
	for _, p := range providercatalog.All() {
		for _, engineID := range p.EngineIDs {
			providerID, ok := providercatalog.ProviderIDForEngine(engineID)
			require.Truef(t, ok, "engine %q has no provider mapping", engineID)
			require.Equalf(t, p.ID, providerID, "engine %q mapping", engineID)
			require.Contains(t, providercatalog.EngineIDsForProvider(p.ID), engineID)
		}
	}
	for _, id := range []string{"whisper-stt", "kyutai-stt", "sherpa-stt"} {
		require.NotEmpty(t, providercatalog.EngineIDsForProvider(id), "%q must declare an STT engine", id)
	}
}

func TestEngineIDsAreUniqueAcrossProviders(t *testing.T) {
	owner := make(map[string]string)
	for _, provider := range providercatalog.All() {
		for _, engineID := range provider.EngineIDs {
			require.NotContainsf(t, owner, engineID, "engine %q is declared by both %q and %q", engineID, owner[engineID], provider.ID)
			owner[engineID] = provider.ID
		}
	}
	require.Len(t, owner, 3, "whisper-local, kyutai, and sherpa-streaming are the streaming engines")
}

func TestRuntimeAliasLookups(t *testing.T) {
	cases := map[string]string{
		"kokoro-local": "kokoro-tts",
		"ollama-local": "ollama",
	}
	for raw, want := range cases {
		got, ok := providercatalog.ProviderIDForRuntimeAlias(raw)
		require.Truef(t, ok, "%q must resolve to a provider", raw)
		require.Equalf(t, want, got, "%q runtime alias owner", raw)
		require.Containsf(t, providercatalog.RuntimeAliasesForProvider(want), raw, "%q must be declared by %q", raw, want)
	}
	for _, unknown := range []string{"", "deepgram", "whisper-local", "no-such-runtime"} {
		_, ok := providercatalog.ProviderIDForRuntimeAlias(unknown)
		require.Falsef(t, ok, "%q is not a declared runtime alias", unknown)
	}
}

// TestRuntimeAliasesAreUniqueAndOwnedByOneProvider is the alias half of the
// identity gate: one alias names exactly one provider, and an alias may not
// shadow a provider ID or an engine ID.
func TestRuntimeAliasesAreUniqueAndOwnedByOneProvider(t *testing.T) {
	providerIDs := make(map[string]struct{})
	engineIDs := make(map[string]string)
	for _, provider := range providercatalog.All() {
		providerIDs[provider.ID] = struct{}{}
		for _, engineID := range provider.EngineIDs {
			engineIDs[engineID] = provider.ID
		}
	}
	owner := make(map[string]string)
	for _, provider := range providercatalog.All() {
		for _, alias := range provider.RuntimeAliases {
			require.NotEmptyf(t, alias, "provider %q declares a blank runtime_alias", provider.ID)
			require.NotEqualf(t, provider.ID, alias, "provider %q declares its own id as a runtime_alias", provider.ID)
			_, shadowsProvider := providerIDs[alias]
			require.Falsef(t, shadowsProvider, "runtime_alias %q shadows a provider id", alias)
			_, shadowsEngine := engineIDs[alias]
			require.Falsef(t, shadowsEngine, "runtime_alias %q shadows an engine id", alias)
			previous, dup := owner[alias]
			require.Falsef(t, dup, "runtime_alias %q is declared by both %q and %q", alias, previous, provider.ID)
			owner[alias] = provider.ID
		}
	}
	require.Len(t, owner, 2, "kokoro-local and ollama-local are the declared runtime aliases")
}

// TestRemoteProvidersDeclareProbeOrAbsence is the readiness-fact gate: a
// provider that health may ask for a credential probe states either the probe
// shape or why it has none, and non-remote providers state neither.
func TestRemoteProvidersDeclareProbeOrAbsence(t *testing.T) {
	for _, provider := range providercatalog.All() {
		remote := provider.Tier == providercatalog.TierBYOK || provider.Tier == providercatalog.TierVrooli
		probe, hasProbe := providercatalog.ProbeFor(provider.ID)
		if !remote {
			require.Falsef(t, hasProbe, "non-remote provider %q must not declare a probe", provider.ID)
			require.Emptyf(t, provider.ProbeAbsentReason, "non-remote provider %q must not declare probe_absent_reason", provider.ID)
			continue
		}
		if hasProbe {
			require.NotEmptyf(t, probe.URL, "provider %q probe needs a url", provider.ID)
			require.NotEmptyf(t, probe.Header, "provider %q probe needs a header", provider.ID)
			require.Truef(t, strings.HasPrefix(probe.URL, "https://"), "provider %q probe must be https", provider.ID)
			require.Emptyf(t, provider.ProbeAbsentReason, "provider %q declares both a probe and an absence", provider.ID)
			continue
		}
		if provider.Status == providercatalog.StatusActive {
			require.NotEmptyf(t, provider.ProbeAbsentReason, "active remote provider %q must declare a probe or a probe_absent_reason", provider.ID)
		}
	}
}

func TestProbeFactsForBYOKProviders(t *testing.T) {
	cases := map[string]providercatalog.Probe{
		"openai-whisper": {
			URL: "https://api.openai.com/v1/models", Header: "Authorization", ValuePrefix: "Bearer ",
		},
		"openai-tts": {
			URL: "https://api.openai.com/v1/models", Header: "Authorization", ValuePrefix: "Bearer ",
		},
		"deepgram": {
			URL: "https://api.deepgram.com/v1/projects", Header: "Authorization", ValuePrefix: "Token ",
		},
		"elevenlabs": {
			URL: "https://api.elevenlabs.io/v1/user", Header: "xi-api-key",
		},
	}
	for providerID, want := range cases {
		got, ok := providercatalog.ProbeFor(providerID)
		require.Truef(t, ok, "%q must declare a probe", providerID)
		require.Equalf(t, want, got, "%q probe facts", providerID)
	}
	for _, providerID := range []string{"unknown", "whisper-stt", "browser-tts", "openrouter", ""} {
		_, ok := providercatalog.ProbeFor(providerID)
		require.Falsef(t, ok, "%q must not declare a probe", providerID)
	}
	openrouter, ok := providercatalog.ByID("openrouter")
	require.True(t, ok)
	require.Contains(t, openrouter.ProbeAbsentReason, "per-request BYOK key")
}

func TestLocalProviderIDsDeterministicAndResourceUnique(t *testing.T) {
	ids := providercatalog.LocalProviderIDs()
	require.Equal(t, ids, providercatalog.LocalProviderIDs(), "order must be deterministic")
	require.Equal(t, []string{"whisper-stt", "kyutai-stt", "kokoro-tts", "speaker-verification", "ollama"}, ids)

	seenSlugs := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		slug, ok := providercatalog.ResourceSlugForProviderID(id)
		require.Truef(t, ok, "%q must resolve to a resource slug", id)
		_, dup := seenSlugs[slug]
		require.Falsef(t, dup, "resource slug %q appears more than once in %v", slug, ids)
		seenSlugs[slug] = struct{}{}
		require.NotEmpty(t, providercatalog.ActionsForProvider(id), "%q must advertise lifecycle actions", id)
	}

	// Every active local resource is represented exactly once.
	resourceSlugs := make(map[string]struct{})
	for _, p := range providercatalog.Active() {
		if p.Kind != providercatalog.KindLocalResource {
			continue
		}
		resourceSlugs[p.ResourceSlug] = struct{}{}
	}
	require.Len(t, seenSlugs, len(resourceSlugs), "every active local resource must be represented exactly once")
	for slug := range resourceSlugs {
		require.Contains(t, seenSlugs, slug)
		// The primary row is the first active local provider for the resource.
		var first string
		for _, p := range providercatalog.All() {
			if p.Status == providercatalog.StatusActive && p.Kind == providercatalog.KindLocalResource && p.ResourceSlug == slug {
				first = p.ID
				break
			}
		}
		require.Containsf(t, ids, first, "resource %q must keep its first/primary provider row", slug)
	}
}

func TestEveryActiveLocalSTTProviderDeclaresAnEngine(t *testing.T) {
	for _, p := range providercatalog.Active() {
		if p.Kind != providercatalog.KindLocalResource || !hasCapability(p, providercatalog.CapabilitySTT) {
			continue
		}
		require.NotEmptyf(t, p.EngineIDs, "%q must declare at least one engine_id", p.ID)
	}
}

func TestEveryLocalProviderHasResource(t *testing.T) {
	for _, p := range providercatalog.Active() {
		if p.Kind != providercatalog.KindLocalResource {
			continue
		}
		require.NotEmptyf(t, p.ResourceSlug, "%q must declare a resource_slug", p.ID)
		slug, ok := providercatalog.ResourceSlugForProviderID(p.ID)
		require.Truef(t, ok, "%q must resolve through lifecycle", p.ID)
		require.Equal(t, p.ResourceSlug, slug)
	}
}

func TestActionsOnlyForLocalResources(t *testing.T) {
	for _, p := range providercatalog.All() {
		if len(p.Actions) == 0 {
			continue
		}
		require.Equalf(t, providercatalog.KindLocalResource, p.Kind, "%q advertises actions but is not a local resource", p.ID)
		require.Truef(t, providercatalog.IsLocalProvider(p.ID), "%q advertises actions but is not lifecycle-targetable", p.ID)
	}
}

func TestPullModelActionIsProviderDeclared(t *testing.T) {
	for _, p := range providercatalog.All() {
		for _, action := range p.Actions {
			if action != providercatalog.ActionPullModel {
				continue
			}
		}
	}
	require.True(t, providercatalog.SupportsPullModel("ollama"))
	require.False(t, providercatalog.SupportsPullModel("speaker-verification"))

	// The action belongs to the provider declaration, not to a provider ID.
	raw, err := os.ReadFile("catalog.seed.json")
	require.NoError(t, err)
	var doc map[string]any
	require.NoError(t, json.Unmarshal(raw, &doc))
	for _, value := range doc["providers"].([]any) {
		provider := value.(map[string]any)
		if provider["id"] == "ollama" {
			provider["id"] = "local-model-service"
			break
		}
	}
	declared, err := json.Marshal(doc)
	require.NoError(t, err)
	registry, err := providercatalog.LoadBytes(declared)
	require.NoError(t, err)
	require.True(t, registry.SupportsPullModel("local-model-service"))
}

func TestEveryActiveFeatureMapsToCapability(t *testing.T) {
	for _, p := range providercatalog.Active() {
		for _, feature := range p.Features {
			capability, ok := providercatalog.CapabilityForFeature(feature)
			require.Truef(t, ok, "%q feature %q must map to a capability", p.ID, feature)
			require.Truef(t, knownCapability(capability), "%q feature %q mapped to invalid capability %q", p.ID, feature, capability)
		}
	}
}

func TestPlannedProvidersAreNotProbed(t *testing.T) {
	for _, p := range providercatalog.All() {
		if p.Status == providercatalog.StatusActive {
			continue
		}
		require.Equalf(t, providercatalog.CheckerNone, p.CheckerKind, "%q must not declare a checker", p.ID)
		require.Emptyf(t, p.Actions, "%q must not advertise lifecycle actions", p.ID)
	}
}

func TestValidationRules(t *testing.T) {
	tests := []struct {
		name      string
		providers func() []map[string]any
		want      string
	}{
		{
			name: "duplicate provider id",
			providers: func() []map[string]any {
				return []map[string]any{baseProvider(), baseProvider()}
			},
			want: "duplicate provider id",
		},
		{
			name: "invalid kind",
			providers: func() []map[string]any {
				p := baseProvider()
				p["kind"] = "mystery"
				return []map[string]any{p}
			},
			want: "invalid kind",
		},
		{
			name: "unsupported TTS endpoint field",
			providers: func() []map[string]any {
				p := baseProvider()
				p["id"], p["name"], p["description"] = "sample-tts", "Sample TTS", "Local TTS provider"
				p["capabilities"], p["features"] = []string{"tts"}, []string{"voice-output"}
				delete(p, "engine_ids")
				p["adapter_id"] = "local:sample"
				p["tts_runtime"] = map[string]any{"backend_id": "sample", "url_field": "ARBITRARY_ENV", "health_path": "/health", "model_role": "default", "cache_backend": "sample"}
				return []map[string]any{p}
			},
			want: "unsupported tts_runtime url_field",
		},
		{
			name: "tier does not match kind",
			providers: func() []map[string]any {
				p := baseProvider()
				p["tier"] = "byok"
				return []map[string]any{p}
			},
			want: "requires tier",
		},
		{
			name: "active provider without features",
			providers: func() []map[string]any {
				p := baseProvider()
				p["features"] = []string{}
				return []map[string]any{p}
			},
			want: "declares no features",
		},
		{
			name: "active local provider without resource slug",
			providers: func() []map[string]any {
				p := baseProvider()
				delete(p, "resource_slug")
				p["dependency_slug"] = "whisper"
				return []map[string]any{p}
			},
			want: "has no resource_slug",
		},
		{
			name: "unknown feature",
			providers: func() []map[string]any {
				p := baseProvider()
				p["features"] = []string{"invented-feature"}
				return []map[string]any{p}
			},
			want: "unknown feature",
		},
		{
			name: "actions on a non-local provider",
			providers: func() []map[string]any {
				p := baseProvider()
				p["kind"] = "scenario"
				p["tier"] = "none"
				p["dependency_kind"] = "scenario"
				delete(p, "resource_slug")
				delete(p, "engine_ids")
				return []map[string]any{p}
			},
			want: "must not advertise lifecycle actions",
		},
		{
			name: "planned provider with a checker",
			providers: func() []map[string]any {
				p := baseProvider()
				p["status"] = "planned"
				return []map[string]any{p}
			},
			want: "must not be probed",
		},
		{
			name: "planned provider with actions",
			providers: func() []map[string]any {
				p := baseProvider()
				p["status"] = "planned"
				p["checker_kind"] = "none"
				return []map[string]any{p}
			},
			want: "must not advertise lifecycle actions",
		},
		{
			name: "active local STT provider without engine ids",
			providers: func() []map[string]any {
				p := baseProvider()
				delete(p, "engine_ids")
				return []map[string]any{p}
			},
			want: "declares no engine_ids",
		},
		{
			name: "duplicate engine id",
			providers: func() []map[string]any {
				p := baseProvider()
				p["engine_ids"] = []string{"whisper-local", "whisper-local"}
				return []map[string]any{p}
			},
			want: "declared by both",
		},
		{
			name: "degraded platform without reason",
			providers: func() []map[string]any {
				p := baseProvider()
				p["platform"] = map[string]any{"support": "degraded"}
				return []map[string]any{p}
			},
			want: "requires a reason",
		},
		{
			name: "incomplete presentation",
			providers: func() []map[string]any {
				p := baseProvider()
				delete(p, "presentation")
				return []map[string]any{p}
			},
			want: "incomplete presentation",
		},
		{
			name: "invalid checker kind",
			providers: func() []map[string]any {
				p := baseProvider()
				p["checker_kind"] = "magic"
				return []map[string]any{p}
			},
			want: "invalid checker_kind",
		},
		{
			name: "invalid capability",
			providers: func() []map[string]any {
				p := baseProvider()
				p["capabilities"] = []string{"video"}
				return []map[string]any{p}
			},
			want: "invalid capability",
		},
		{
			name: "blank runtime alias",
			providers: func() []map[string]any {
				p := baseProvider()
				p["runtime_aliases"] = []string{""}
				return []map[string]any{p}
			},
			want: "blank runtime_alias",
		},
		{
			name: "voice aliases without adapter",
			providers: func() []map[string]any {
				p := baseProvider()
				p["voice_aliases"] = map[string]string{"voice.feminine.warm": "bella"}
				return []map[string]any{p}
			},
			want: "voice_aliases without adapter_id",
		},
		{
			name: "invalid voice alias declaration",
			providers: func() []map[string]any {
				p := baseBYOKProvider("invalid-tts")
				p["capabilities"] = []string{"tts"}
				p["adapter_id"] = "test:tts"
				p["voice_aliases"] = map[string]string{"feminine.warm": " "}
				return []map[string]any{p}
			},
			want: "invalid voice_alias",
		},
		{
			name: "runtime alias equal to the provider id",
			providers: func() []map[string]any {
				p := baseProvider()
				p["runtime_aliases"] = []string{"whisper-stt"}
				return []map[string]any{p}
			},
			want: "own id as a runtime_alias",
		},
		{
			name: "runtime alias shadowing another provider id",
			providers: func() []map[string]any {
				other := baseBYOKProvider("acme")
				other["runtime_aliases"] = []string{"whisper-stt"}
				return []map[string]any{baseProvider(), other}
			},
			want: "collides with provider id",
		},
		{
			name: "runtime alias shadowing an engine id",
			providers: func() []map[string]any {
				p := baseProvider()
				p["runtime_aliases"] = []string{"whisper-local"}
				return []map[string]any{p}
			},
			want: "collides with an engine_id",
		},
		{
			name: "duplicate runtime alias",
			providers: func() []map[string]any {
				first := baseBYOKProvider("acme")
				first["runtime_aliases"] = []string{"acme-local"}
				second := baseBYOKProvider("globex")
				second["runtime_aliases"] = []string{"acme-local"}
				return []map[string]any{first, second}
			},
			want: "declared by both",
		},
		{
			name: "probe on a local provider",
			providers: func() []map[string]any {
				p := baseProvider()
				p["probe"] = map[string]any{"url": "https://example.test/v1/models", "header": "Authorization"}
				return []map[string]any{p}
			},
			want: "must not declare a probe",
		},
		{
			name: "probe with a non-https url",
			providers: func() []map[string]any {
				p := baseBYOKProvider("acme")
				p["probe"] = map[string]any{"url": "http://example.test/v1/models", "header": "Authorization"}
				return []map[string]any{p}
			},
			want: "must be https",
		},
		{
			name: "probe without a header",
			providers: func() []map[string]any {
				p := baseBYOKProvider("acme")
				p["probe"] = map[string]any{"url": "https://example.test/v1/models"}
				return []map[string]any{p}
			},
			want: "probe has no header",
		},
		{
			name: "probed BYOK provider has multiple capabilities",
			providers: func() []map[string]any {
				p := baseBYOKProvider("acme")
				p["capabilities"] = []string{"stt", "tts"}
				p["features"] = []string{"voice-input", "voice-output"}
				return []map[string]any{p}
			},
			want: "must declare exactly one capability for provider-level health",
		},
		{
			name: "probe and documented absence",
			providers: func() []map[string]any {
				p := baseBYOKProvider("acme")
				p["probe"] = map[string]any{"url": "https://example.test/v1/models", "header": "Authorization"}
				p["probe_absent_reason"] = "unused"
				return []map[string]any{p}
			},
			want: "both a probe and a probe_absent_reason",
		},
		{
			name: "documented absence on a local provider",
			providers: func() []map[string]any {
				p := baseProvider()
				p["probe_absent_reason"] = "local providers are probed by resource health"
				return []map[string]any{p}
			},
			want: "must not declare a probe_absent_reason",
		},
		{
			name: "active remote provider without probe or absence",
			providers: func() []map[string]any {
				p := baseBYOKProvider("acme")
				delete(p, "probe")
				return []map[string]any{p}
			},
			want: "must declare a probe or a probe_absent_reason",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			raw, err := json.Marshal(map[string]any{
				"schema_version": "1.0.0",
				"providers":      tc.providers(),
			})
			require.NoError(t, err)
			_, err = providercatalog.LoadBytes(raw)
			require.ErrorContains(t, err, tc.want)
		})
	}
}

func TestValidationKeepsFirstIssueOrder(t *testing.T) {
	provider := baseProvider()
	provider["id"] = ""
	provider["name"] = ""
	raw, err := json.Marshal(map[string]any{"schema_version": "1.0.0", "providers": []any{provider}})
	require.NoError(t, err)
	_, err = providercatalog.LoadBytes(raw)
	require.EqualError(t, err, "providercatalog: provider with empty id at index 0")
}

func TestActiveBYOKProviderRequiresProtocolDeclaration(t *testing.T) {
	provider := baseBYOKProvider("sample-cloud")
	delete(provider, "byok_adapter_id")
	raw, err := json.Marshal(map[string]any{"schema_version": "1.0.0", "providers": []any{provider}})
	require.NoError(t, err)
	_, err = providercatalog.LoadBytes(raw)
	require.ErrorContains(t, err, "must declare byok_adapter_id")
}

// baseBYOKProvider is a valid active remote provider fixture. It declares a
// probe so each case can add, replace, or remove that fact.
func baseBYOKProvider(id string) map[string]any {
	return map[string]any{
		"id":               id,
		"name":             "BYOK " + id,
		"description":      "BYOK cloud provider",
		"kind":             "byok_api",
		"tier":             "byok",
		"byok_adapter_id":  "tts.generic-v1",
		"status":           "active",
		"capabilities":     []string{"tts"},
		"features":         []string{"voice-output"},
		"dependency_kind":  "scenario",
		"dependency_slug":  "audio-tools",
		"platform":         map[string]any{"support": "supported"},
		"checker_kind":     "http_json",
		"probe":            map[string]any{"url": "https://example.test/v1/models", "header": "Authorization", "value_prefix": "Bearer "},
		"operator_command": "audio-tools settings providers",
		"criticality":      "optional",
		"def_action_kind":  "owner_guidance",
		"presentation": map[string]any{
			"label_key":       "providers." + id + ".label",
			"description_key": "providers." + id + ".description",
		},
	}
}

func baseProvider() map[string]any {
	return map[string]any{
		"id":               "whisper-stt",
		"name":             "Whisper STT",
		"description":      "Local batch speech-to-text provider",
		"kind":             "local_resource",
		"tier":             "local",
		"status":           "active",
		"capabilities":     []string{"stt"},
		"features":         []string{"voice-input"},
		"engine_ids":       []string{"whisper-local"},
		"resource_slug":    "whisper",
		"dependency_kind":  "resource",
		"platform":         map[string]any{"support": "supported"},
		"actions":          []string{"start", "stop", "restart", "view_logs"},
		"checker_kind":     "http_json",
		"operator_command": "vrooli resource start whisper",
		"criticality":      "optional",
		"def_action_kind":  "operator_command",
		"presentation": map[string]any{
			"label_key":       "providers.whisper-stt.label",
			"description_key": "providers.whisper-stt.description",
		},
	}
}

func hasCapability(p providercatalog.Provider, capability providercatalog.Capability) bool {
	for _, declared := range p.Capabilities {
		if declared == capability {
			return true
		}
	}
	return false
}

func knownCapability(capability providercatalog.Capability) bool {
	switch capability {
	case providercatalog.CapabilitySTT,
		providercatalog.CapabilityTTS,
		providercatalog.CapabilitySummarize,
		providercatalog.CapabilityTranscode,
		providercatalog.CapabilitySpeaker:
		return true
	default:
		return false
	}
}
