package providercatalog_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/providercatalog"
)

func TestEmbeddedCatalogLoads(t *testing.T) {
	reg := providercatalog.Default()
	require.NotNil(t, reg)
	require.Equal(t, "1.0.0", reg.SchemaVersion())
	require.Len(t, reg.All(), 19)
	require.Len(t, reg.Active(), 15)
}

func TestAccessors(t *testing.T) {
	whisper, ok := providercatalog.ByID("whisper-stt")
	require.True(t, ok)
	require.Equal(t, "whisper", whisper.ResourceSlug)
	require.Equal(t, providercatalog.TierLocal, providercatalog.TierForProviderID("whisper-stt"))
	require.Equal(t, providercatalog.TierBYOK, providercatalog.TierForProviderID("openai-tts"))
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

func TestCapabilityForFeature(t *testing.T) {
	cases := map[string]providercatalog.Capability{
		"voice-input":                providercatalog.CapabilitySTT,
		"voice-streaming":            providercatalog.CapabilitySTT,
		"voice-streaming-realtime":   providercatalog.CapabilitySTT,
		"voice-speaker-verification": providercatalog.CapabilitySpeaker,
		"voice-enrollment":           providercatalog.CapabilitySpeaker,
		"voice-output":               providercatalog.CapabilityTTS,
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
	require.Len(t, owner, 3, "whisper-local, kyutai, and sherpa-streaming are the declared engines")
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
	_, ok := providercatalog.ProbeFor("openrouter")
	require.False(t, ok, "openrouter readiness is owned by its dedicated checker")
	openrouter, found := providercatalog.ByID("openrouter")
	require.True(t, found)
	require.NotEmpty(t, openrouter.ProbeAbsentReason)
	for _, providerID := range []string{"unknown", "whisper-stt", "browser-tts", ""} {
		_, ok := providercatalog.ProbeFor(providerID)
		require.Falsef(t, ok, "%q must not declare a probe", providerID)
	}
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

func TestPullModelOnlyForOllama(t *testing.T) {
	for _, p := range providercatalog.All() {
		for _, action := range p.Actions {
			if action != providercatalog.ActionPullModel {
				continue
			}
			require.Equalf(t, "ollama", p.ID, "pull_model is only valid on ollama")
		}
	}
	require.True(t, providercatalog.SupportsPullModel("ollama"))
	require.False(t, providercatalog.SupportsPullModel("speaker-verification"))
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
			name: "pull_model on a non-ollama provider",
			providers: func() []map[string]any {
				p := baseProvider()
				p["actions"] = []string{"start", "pull_model"}
				return []map[string]any{p}
			},
			want: "only ollama",
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

// baseBYOKProvider is a valid active remote provider fixture. It declares a
// probe so each case can add, replace, or remove that fact.
func baseBYOKProvider(id string) map[string]any {
	return map[string]any{
		"id":               id,
		"name":             "BYOK " + id,
		"description":      "BYOK cloud provider",
		"kind":             "byok_api",
		"tier":             "byok",
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
		"models":           []string{"whisper-medium"},
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
