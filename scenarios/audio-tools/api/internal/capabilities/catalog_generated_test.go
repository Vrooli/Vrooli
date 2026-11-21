package capabilities

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/providercatalog"
)

// TestKnownMatchesActiveCatalog proves capabilities.Known is the generated
// active provider catalogue, not a hand-maintained second list.
func TestKnownMatchesActiveCatalog(t *testing.T) {
	active := providercatalog.Active()
	require.Len(t, Known, len(active))

	activeIDs := make([]string, 0, len(active))
	for _, provider := range active {
		activeIDs = append(activeIDs, provider.ID)
	}
	knownIDs := make([]string, 0, len(Known))
	for _, def := range Known {
		knownIDs = append(knownIDs, def.ID)
	}
	require.Equal(t, activeIDs, knownIDs)
}

func TestKnownExcludesPlannedProviders(t *testing.T) {
	for _, def := range Known {
		require.NotEqual(t, "rumble-stt", def.ID)
		require.NotEqual(t, "kale-tts", def.ID)
		require.NotEqual(t, "reverb-tts", def.ID)
		require.NotEqual(t, "verba-tts", def.ID)
	}
}

func TestEveryActiveFeatureMapsToCapability(t *testing.T) {
	for _, provider := range providercatalog.Active() {
		for _, feature := range provider.Features {
			capability, ok := CapabilityForFeature(feature)
			require.Truef(t, ok, "provider %q feature %q must map to a diagnostics capability", provider.ID, feature)
			require.NotEqualf(t, "CAPABILITY_UNSPECIFIED", capability.String(), "provider %q feature %q", provider.ID, feature)
		}
	}
}

func TestEveryLocalProviderHasResource(t *testing.T) {
	for _, provider := range providercatalog.Active() {
		if provider.Kind != providercatalog.KindLocalResource {
			continue
		}
		slug, ok := ResourceSlugForProviderID(provider.ID)
		require.Truef(t, ok, "local provider %q must resolve to a lifecycle slug", provider.ID)
		require.NotEmptyf(t, slug, "local provider %q has an empty lifecycle slug", provider.ID)
	}
}

func TestActionsOnlyForLocalResources(t *testing.T) {
	for _, provider := range providercatalog.All() {
		if len(provider.Actions) == 0 {
			continue
		}
		require.Equalf(t, providercatalog.KindLocalResource, provider.Kind, "%q advertises actions but is not a local resource", provider.ID)
		require.Truef(t, IsLocalProvider(provider.ID), "%q advertises actions but is not lifecycle-targetable", provider.ID)
	}
}

func TestPlannedProvidersAreNotProbed(t *testing.T) {
	known := make(map[string]struct{}, len(Known))
	for _, def := range Known {
		known[def.ID] = struct{}{}
	}
	for _, provider := range providercatalog.All() {
		if provider.Status == providercatalog.StatusActive {
			continue
		}
		_, present := known[provider.ID]
		require.Falsef(t, present, "%s provider %q must not be in capabilities.Known", provider.Status, provider.ID)
		require.Equalf(t, providercatalog.CheckerNone, provider.CheckerKind, "%s provider %q must not be probed", provider.Status, provider.ID)
	}
}

func TestPullModelOnlyForOllama(t *testing.T) {
	for _, provider := range providercatalog.All() {
		for _, action := range provider.Actions {
			if action == providercatalog.ActionPullModel {
				require.Equalf(t, "ollama", provider.ID, "pull_model must only be advertised by ollama")
			}
		}
	}
	require.True(t, SupportsPullModel("ollama"))
	for _, id := range []string{"whisper-stt", "speaker-verification", "openai-tts"} {
		require.Falsef(t, SupportsPullModel(id), "%q must not support pull_model", id)
	}
}

// TestGeneratedHelpersMatchProviderCatalog keeps the generated helper tables
// honest. It intentionally uses every generated helper so none can drift or be
// deleted without a compile/test failure.
func TestGeneratedHelpersMatchProviderCatalog(t *testing.T) {
	for _, feature := range providercatalog.KnownFeatures() {
		wantCatalog, wantOK := providercatalog.CapabilityForFeature(feature)
		wantProto, wantProtoOK := CapabilityForFeature(feature)
		gotProto, gotOK := generatedCapabilityForFeature(feature)
		require.Equalf(t, wantOK, wantProtoOK, "feature %q", feature)
		require.Equalf(t, wantProto, gotProto, "feature %q", feature)
		require.Equalf(t, gotOK, wantOK, "feature %q", feature)
		require.NotEmpty(t, wantCatalog)
	}

	for _, provider := range providercatalog.All() {
		require.Equalf(t, TierForProviderID(provider.ID), generatedTierForProviderID(provider.ID), "tier for %q", provider.ID)

		wantSlug, wantLocal := ResourceSlugForProviderID(provider.ID)
		gotSlug, gotLocal := generatedResourceSlugForProviderID(provider.ID)
		require.Equalf(t, wantSlug, gotSlug, "slug for %q", provider.ID)
		require.Equalf(t, wantLocal, gotLocal, "slug presence for %q", provider.ID)
		require.Equalf(t, IsLocalProvider(provider.ID), generatedIsLocalProvider(provider.ID), "local for %q", provider.ID)
		require.Equalf(t, SupportsPullModel(provider.ID), generatedSupportsPullModel(provider.ID), "pull model for %q", provider.ID)

		wantActions := make([]string, 0, len(provider.Actions))
		for _, action := range provider.Actions {
			wantActions = append(wantActions, string(action))
		}
		if provider.Status != providercatalog.StatusActive || provider.Kind != providercatalog.KindLocalResource {
			wantActions = nil
		}
		require.Equalf(t, wantActions, generatedActionsForProvider(provider.ID), "actions for %q", provider.ID)
	}
}

// TestBYOKCredentialCheckersHaveCatalogProbes keeps the composition root and
// the catalogue probe facts in lockstep: a provider the bootstrap wires for a
// credential probe must declare its probe shape as catalogue data, not as a
// switch in capabilities.
func TestBYOKCredentialCheckersHaveCatalogProbes(t *testing.T) {
	path := filepath.Join("..", "bootstrap", "bootstrap.go")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)
	source := string(raw)
	const marker = "[]struct{ id, provider, capability string }{"
	start := strings.Index(source, marker)
	require.GreaterOrEqual(t, start, 0, "bootstrap must declare the BYOK credential checker table")
	end := strings.Index(source[start:], "\n\t} {")
	require.GreaterOrEqual(t, end, 0, "bootstrap BYOK credential checker table is malformed")
	rows := regexp.MustCompile(`\{"[^"]+", "([^"]+)", "[^"]+"\}`).FindAllStringSubmatch(source[start:start+end], -1)
	require.NotEmpty(t, rows, "bootstrap BYOK credential checker table has no rows")
	for _, row := range rows {
		providerID := row[1]
		_, ok := providercatalog.ProbeFor(providerID)
		require.Truef(t, ok, "bootstrap credential-probes %q but the catalogue declares no probe", providerID)
	}
}

// TestEveryActiveLocalProviderHasBootstrapChecker proves the composition root
// still names a checker for every active local provider. The source scan is
// deliberate: bootstrap owns construction and this test refuses to let the
// catalogue grow a local provider with no health path.
func TestEveryActiveLocalProviderHasBootstrapChecker(t *testing.T) {
	path := filepath.Join("..", "bootstrap", "bootstrap.go")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "read %s", path)
	source := string(raw)
	for _, provider := range providercatalog.Active() {
		if !IsLocalProvider(provider.ID) {
			continue
		}
		require.NotEqualf(t, providercatalog.CheckerNone, provider.CheckerKind, "local provider %q has no checker kind", provider.ID)
		require.Truef(t, strings.Contains(source, `"`+provider.ID+`":`),
			"local provider %q has no checker row in %s", provider.ID, path)
	}
}

func TestLocalProviderIDsWrapper(t *testing.T) {
	ids := LocalProviderIDs()
	require.Equal(t, providercatalog.LocalProviderIDs(), ids)
	require.Contains(t, ids, "kyutai-stt", "local actions must surface kyutai-stt")
	require.Equal(t, ids, LocalProviderIDs(), "order must be deterministic")

	seen := make(map[string]struct{}, len(ids))
	for _, id := range ids {
		slug, ok := ResourceSlugForProviderID(id)
		require.Truef(t, ok, "%q must resolve to a lifecycle slug", id)
		_, dup := seen[slug]
		require.Falsef(t, dup, "resource slug %q appears twice in %v", slug, ids)
		seen[slug] = struct{}{}
	}
}
