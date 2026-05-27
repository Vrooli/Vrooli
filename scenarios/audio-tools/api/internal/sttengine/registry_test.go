package sttengine

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/require"

	"audio-tools/internal/ai/sttchain"
)

// TestEmbeddedManifestValid asserts the checked-in manifest loads + passes the
// cross-field invariants.
func TestEmbeddedManifestValid(t *testing.T) {
	r := Default()
	require.NotEmpty(t, r.Engines())
	_, _, ok := r.ActiveSpeakerMethod()
	require.True(t, ok, "active speaker method must resolve")
}

// TestSchemaJSONIsValidJSON guards against a malformed schema doc.
func TestSchemaJSONIsValidJSON(t *testing.T) {
	var v any
	require.NoError(t, json.Unmarshal(schemaJSON, &v))
}

// TestKnownStrategiesMatchSttchain keeps the manifest's allowed strategy ids in
// lockstep with the canonical sttchain.StrategyKind constants. This is the
// reason registry.go can stay free of an sttchain import (which would cycle via
// egress) without the two sets silently drifting.
func TestKnownStrategiesMatchSttchain(t *testing.T) {
	canonical := map[string]struct{}{
		string(sttchain.StrategyVADSegment):   {},
		string(sttchain.StrategyOverlapAgree): {},
		string(sttchain.StrategyPassthrough):  {},
		string(sttchain.StrategyBuffered):     {},
	}
	require.Equal(t, canonical, knownStrategies, "manifest strategy id set drifted from sttchain.StrategyKind constants")
}

// TestManifestStrategiesAreKnown asserts every strategy every engine lists is a
// real StrategyKind.
func TestManifestStrategiesAreKnown(t *testing.T) {
	r := Default()
	for _, e := range r.Engines() {
		for _, s := range e.Strategies {
			_, ok := knownStrategies[s]
			require.Truef(t, ok, "engine %q lists unknown strategy %q", e.ID, s)
		}
	}
}

// TestLocalResourceEnginesDeclaredInServiceJSON asserts every local_resource
// engine's resource appears in .vrooli/service.json dependencies.resources, so
// the manifest can never reference a resource the scenario does not declare.
func TestLocalResourceEnginesDeclaredInServiceJSON(t *testing.T) {
	declared := serviceResourceDeps(t)
	for _, e := range Default().Engines() {
		if e.Kind != KindLocalResource {
			continue
		}
		_, ok := declared[e.Resource]
		require.Truef(t, ok, "engine %q resource %q not declared in .vrooli/service.json dependencies.resources", e.ID, e.Resource)
	}
}

// TestEligibleStrategiesUnknownEngine returns nil so the selector falls back to
// provider traits.
func TestEligibleStrategiesUnknownEngine(t *testing.T) {
	require.Nil(t, Default().EligibleStrategies("does-not-exist"))
}

func TestValidateRejectsNativeStreamingNonPassthrough(t *testing.T) {
	m := Manifest{
		Engines: []Engine{{
			ID: "bad", DisplayName: "Bad", Kind: KindLocalResource, Resource: "x",
			Provides:   Provides{NativeStreaming: true},
			Strategies: []string{"vad_segment"},
		}},
		SpeakerIsolation: SpeakerIsolation{Active: "verification", Methods: map[string]SpeakerMethod{"verification": {}}},
	}
	require.Error(t, New(m).Validate())
}

func TestValidateRejectsUndeclaredActiveSpeakerMethod(t *testing.T) {
	m := Manifest{
		Engines: []Engine{{
			ID: "e", DisplayName: "E", Kind: KindLocalResource, Resource: "x",
			Strategies: []string{"vad_segment"},
		}},
		SpeakerIsolation: SpeakerIsolation{Active: "ghost", Methods: map[string]SpeakerMethod{"verification": {}}},
	}
	require.Error(t, New(m).Validate())
}

// serviceResourceDeps reads dependencies.resources from the scenario's
// .vrooli/service.json (three dirs up from this package).
func serviceResourceDeps(t *testing.T) map[string]any {
	t.Helper()
	path := filepath.Join("..", "..", "..", ".vrooli", "service.json")
	raw, err := os.ReadFile(path)
	require.NoError(t, err, "read service.json at %s", path)
	var doc struct {
		Dependencies struct {
			Resources map[string]any `json:"resources"`
		} `json:"dependencies"`
	}
	require.NoError(t, json.Unmarshal(raw, &doc))
	return doc.Dependencies.Resources
}
