package routing

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/require"

	registryv1 "github.com/vrooli/vrooli/packages/proto/gen/go/search-hub/v1/registry"
)

// --- parsing ---------------------------------------------------------------

func TestParseClassifierResponse_GatewayEnvelopeWithThink(t *testing.T) {
	// The shape resource-ollama gateway generate --json actually returns for a
	// qwen3 model with /no_think: an outer {"response":…} wrapping an (empty)
	// <think> block + the JSON object.
	raw := []byte(`{"response":"<think>\n\n</think>\n\n{\"types\":[\"command\",\"doc\"],\"confidence\":0.8,\"reason\":\"restart is a CLI op\"}","eval_count":20}`)
	res, err := parseClassifierResponse(raw)
	require.NoError(t, err)
	require.Equal(t, []string{"command", "doc"}, res.Types)
	require.InDelta(t, 0.8, res.Confidence, 1e-9)
	require.Equal(t, "restart is a CLI op", res.Rationale)
}

func TestParseClassifierResponse_PlainJSON(t *testing.T) {
	res, err := parseClassifierResponse([]byte(`{"types":["record"],"confidence":0.42}`))
	require.NoError(t, err)
	require.Equal(t, []string{"record"}, res.Types)
	require.InDelta(t, 0.42, res.Confidence, 1e-9)
	require.False(t, res.WebShaped, "web_shaped absent defaults to false")
}

func TestParseClassifierResponse_WebShaped(t *testing.T) {
	res, err := parseClassifierResponse([]byte(`{"types":["web"],"confidence":0.8,"web_shaped":true}`))
	require.NoError(t, err)
	require.True(t, res.WebShaped, "web_shaped:true is decoded (OT-P2-002)")
	require.InDelta(t, 0.8, res.Confidence, 1e-9)
}

func TestParseClassifierResponse_ConfidenceClamped(t *testing.T) {
	res, err := parseClassifierResponse([]byte(`{"response":"{\"types\":[\"x\"],\"confidence\":7}"}`))
	require.NoError(t, err)
	require.Equal(t, 1.0, res.Confidence, "confidence above 1 is clamped")
}

func TestParseClassifierResponse_NoJSON(t *testing.T) {
	_, err := parseClassifierResponse([]byte(`{"response":"I cannot help with that."}`))
	require.Error(t, err)
}

func TestParseClassifierResponse_SalvagesMalformedTypesArray(t *testing.T) {
	// The real qwen3:1.7b failure mode: confidence/reason merged into the types
	// array, so strict JSON rejects it. Salvage must still recover the decision.
	raw := []byte(`{"response":"<think>\n\n</think>\n\n{\"types\":[\"command\",\"confidence\":0.45,\"reason\":\"restart is a CLI op\"]}","eval_count":40}`)
	res, err := parseClassifierResponse(raw)
	require.NoError(t, err)
	require.Contains(t, res.Types, "command", "the real type token is recovered")
	require.InDelta(t, 0.45, res.Confidence, 1e-9, "confidence recovered from the malformed body")
	require.Equal(t, "restart is a CLI op", res.Rationale)
	// 'confidence'/'reason' are over-extracted as tokens but widenPolicy drops
	// them against the live registry — assert they don't crowd out the real type.
	chosen, _ := widenPolicy(res, []string{"command", "component", "record"})
	require.Equal(t, []string{"command"}, chosen)
}

// --- widen policy ----------------------------------------------------------

func TestWidenPolicy_ConfidentNarrows(t *testing.T) {
	chosen, widened := widenPolicy(
		ClassifyResult{Types: []string{"command"}, Confidence: 0.9},
		[]string{"command", "component", "record"},
	)
	require.Equal(t, []string{"command"}, chosen)
	require.False(t, widened)
}

func TestWidenPolicy_LowConfidenceWidens(t *testing.T) {
	chosen, widened := widenPolicy(
		ClassifyResult{Types: []string{"command"}, Confidence: 0.3},
		[]string{"command", "component", "record"},
	)
	require.True(t, widened, "low confidence over-fetches across every type")
	require.Equal(t, []string{"command", "component", "record"}, chosen)
}

func TestWidenPolicy_NoUsableMatchWidens(t *testing.T) {
	// Classifier named a type no provider serves ⇒ nothing intersects ⇒ widen.
	chosen, widened := widenPolicy(
		ClassifyResult{Types: []string{"doc"}, Confidence: 0.95},
		[]string{"command", "record"},
	)
	require.True(t, widened)
	require.Equal(t, []string{"command", "record"}, chosen)
}

func TestWidenPolicy_DropsUnknownButKeepsKnown(t *testing.T) {
	chosen, widened := widenPolicy(
		ClassifyResult{Types: []string{"doc", "command"}, Confidence: 0.8},
		[]string{"command", "record"},
	)
	require.False(t, widened)
	require.Equal(t, []string{"command"}, chosen, "unknown 'doc' dropped, known 'command' kept")
}

// --- profile derivation ----------------------------------------------------

func TestBuildProfiles_OnePerTypeJoinsDescriptions(t *testing.T) {
	active := []*registryv1.ProviderDescriptor{
		descWithEndpoint("a.x", "g", "command", "first"),
		descWithEndpoint("b.y", "g", "command", "second"),
		descWithEndpoint("c.z", "g", "record", "third"),
		gapDescriptor("d.gap", "scenario"), // no endpoint ⇒ excluded
	}
	profiles := buildProfiles(active)
	require.Len(t, profiles, 2)
	require.Equal(t, "command", profiles[0].Type)
	require.Equal(t, "first second", profiles[0].Description, "shared-type descriptions join")
	require.Equal(t, "record", profiles[1].Type)
}

func TestAvailableTypes_DistinctSorted(t *testing.T) {
	active := []*registryv1.ProviderDescriptor{
		descWithEndpoint("a", "g", "record", "d"),
		descWithEndpoint("b", "g", "command", "d"),
		descWithEndpoint("c", "g", "command", "d"),
	}
	require.Equal(t, []string{"command", "record"}, availableTypes(active))
}

// --- OllamaClassifier with a seamed runner ---------------------------------

func TestOllamaClassifier_Classify_UsesRunnerOutput(t *testing.T) {
	var gotModel, gotPrompt string
	c := &OllamaClassifier{
		model:     "qwen3:1.7b",
		maxTokens: classifierMaxTokens,
		generate: func(_ context.Context, model, prompt string, _ int) ([]byte, error) {
			gotModel, gotPrompt = model, prompt
			return []byte(`{"response":"{\"types\":[\"command\"],\"confidence\":0.9}"}`), nil
		},
	}
	res, err := c.Classify(context.Background(), "restart a scenario",
		[]ProviderProfile{{Type: "command", Description: "CLI commands"}})
	require.NoError(t, err)
	require.Equal(t, []string{"command"}, res.Types)
	require.Equal(t, "qwen3:1.7b", gotModel)
	require.Contains(t, gotPrompt, "CLI commands", "the provider description must reach the prompt")
	require.Contains(t, gotPrompt, "restart a scenario")
}

func TestOllamaClassifier_Classify_RunnerErrorPropagates(t *testing.T) {
	c := &OllamaClassifier{
		generate: func(_ context.Context, _, _ string, _ int) ([]byte, error) {
			return nil, errors.New("daemon down")
		},
	}
	_, err := c.Classify(context.Background(), "q", []ProviderProfile{{Type: "command", Description: "d"}})
	require.ErrorContains(t, err, "daemon down")
}

func TestOllamaClassifier_Classify_EmptyInputs(t *testing.T) {
	c := &OllamaClassifier{generate: func(context.Context, string, string, int) ([]byte, error) { return nil, nil }}
	_, err := c.Classify(context.Background(), "  ", []ProviderProfile{{Type: "x"}})
	require.Error(t, err)
	_, err = c.Classify(context.Background(), "q", nil)
	require.Error(t, err)
}

// --- helpers ---------------------------------------------------------------

func descWithEndpoint(id, group, typ, desc string) *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId: id, ProviderGroup: group, Type: typ, Description: desc,
		State: registryv1.ProviderState_PROVIDER_STATE_ACTIVE,
		Endpoint: &registryv1.Endpoint{Kind: &registryv1.Endpoint_HttpJson{
			HttpJson: &registryv1.HttpJsonEndpoint{ScenarioId: group, Path: "/x", BodyTemplate: `{"q":"{{query}}"}`},
		}},
	}
}

func gapDescriptor(id, intendedHome string) *registryv1.ProviderDescriptor {
	return &registryv1.ProviderDescriptor{
		ProviderId: id, Type: "scenario", IntendedHome: intendedHome,
		State: registryv1.ProviderState_PROVIDER_STATE_CAPABILITY_GAP,
	}
}
