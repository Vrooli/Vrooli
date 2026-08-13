package routing

import (
	"context"
	"errors"
	"fmt"
	"strings"
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

func TestParseClassifierResponse_ExactProviderIDs(t *testing.T) {
	res, err := parseClassifierResponse([]byte(`{"provider_ids":["source-ledger.scope.team:monetization"],"confidence":0.9}`))
	require.NoError(t, err)
	require.Equal(t, []string{"source-ledger.scope.team:monetization"}, res.ProviderIDs)
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
	// The real qwen3:1.7b failure mode: confidence/reason merged into the provider_ids
	// array, so strict JSON rejects it. Salvage must still recover the decision.
	raw := []byte(`{"response":"<think>\n\n</think>\n\n{\"types\":[\"command\",\"confidence\":0.45,\"reason\":\"restart is a CLI op\"]}","eval_count":40}`)
	res, err := parseClassifierResponse(raw)
	require.NoError(t, err)
	require.Contains(t, res.Types, "command", "the real type token is recovered")
	require.InDelta(t, 0.45, res.Confidence, 1e-9, "confidence recovered from the malformed body")
	require.Equal(t, "restart is a CLI op", res.Rationale)
	// 'confidence'/'reason' are over-extracted as tokens but widenPolicy drops
	// them against the live registry — assert they don't crowd out the real id.
	profiles := buildProfiles([]*registryv1.ProviderDescriptor{
		descWithEndpoint("command", "g", "command", "command"),
		descWithEndpoint("component", "g", "component", "component"),
		descWithEndpoint("record", "g", "record", "record"),
	})
	chosen, _, _ := widenPolicy(res, profiles, 6)
	require.Equal(t, []string{"command"}, chosen)
}

// --- widen policy ----------------------------------------------------------

func TestWidenPolicy_ConfidentNarrows(t *testing.T) {
	profiles := buildProfiles([]*registryv1.ProviderDescriptor{
		descWithEndpoint("command", "g", "command", "command"),
		descWithEndpoint("component", "g", "component", "component"),
		descWithEndpoint("record", "g", "record", "record"),
	})
	chosen, widened, bound := widenPolicy(
		ClassifyResult{Types: []string{"command"}, Confidence: 0.9},
		profiles, 6,
	)
	require.Equal(t, []string{"command"}, chosen)
	require.False(t, widened)
	require.False(t, bound)
}

func TestWidenPolicy_LowConfidenceWidens(t *testing.T) {
	profiles := buildProfiles([]*registryv1.ProviderDescriptor{
		descWithEndpoint("command", "g", "command", "command"),
		descWithEndpoint("component", "g", "component", "component"),
		descWithEndpoint("record", "g", "record", "record"),
	})
	chosen, widened, bound := widenPolicy(
		ClassifyResult{Types: []string{"command"}, Confidence: 0.3},
		profiles, 6,
	)
	require.True(t, widened, "low confidence over-fetches across every type")
	require.Equal(t, []string{"command"}, chosen)
	require.False(t, bound)
}

func TestWidenPolicy_NoUsableMatchWidens(t *testing.T) {
	// Classifier named a type no provider serves ⇒ nothing intersects ⇒ widen.
	profiles := buildProfiles([]*registryv1.ProviderDescriptor{
		descWithEndpoint("command", "g", "command", "command"),
		descWithEndpoint("record", "g", "record", "record"),
	})
	chosen, widened, _ := widenPolicy(
		ClassifyResult{Types: []string{"doc"}, Confidence: 0.95},
		profiles, 6,
	)
	require.True(t, widened)
	require.Equal(t, []string{"command", "record"}, chosen)
}

func TestWidenPolicy_DropsUnknownButKeepsKnown(t *testing.T) {
	profiles := buildProfiles([]*registryv1.ProviderDescriptor{
		descWithEndpoint("command", "g", "command", "command"),
		descWithEndpoint("record", "g", "record", "record"),
	})
	chosen, widened, _ := widenPolicy(
		ClassifyResult{Types: []string{"doc", "command"}, Confidence: 0.8},
		profiles, 6,
	)
	require.False(t, widened)
	require.Equal(t, []string{"command"}, chosen, "unknown 'doc' dropped, known 'command' kept")
}

func TestWidenPolicyBoundsSiblingLeaves(t *testing.T) {
	profiles := buildProfiles([]*registryv1.ProviderDescriptor{
		descWithEndpoint("command-a", "g", "command", "a"),
		descWithEndpoint("command-b", "g", "command", "b"),
		descWithEndpoint("command-c", "g", "command", "c"),
		descWithEndpoint("record", "g", "record", "record"),
	})
	chosen, widened, bound := widenPolicy(ClassifyResult{ProviderIDs: []string{"command-a"}, Confidence: 0.1}, profiles, 2)
	require.Equal(t, []string{"command-a", "command-b"}, chosen)
	require.True(t, widened)
	require.True(t, bound)
}

func TestWidenPolicyBoundsProviderGrowth(t *testing.T) {
	// Automatic routing must remain bounded as a provider group grows. This is
	// deliberately much larger than the production bound so the assertion
	// protects the cost contract rather than merely exercising a small fixture.
	const providerCount = 1000
	profiles := make([]ProviderProfile, 0, providerCount)
	for i := 0; i < providerCount; i++ {
		id := fmt.Sprintf("command-%04d", i)
		profiles = append(profiles, ProviderProfile{
			ProviderID:  id,
			Type:        "command",
			Group:       "cli-health",
			Description: id,
		})
	}

	chosen, widened, bound := widenPolicy(
		ClassifyResult{ProviderIDs: []string{"command-0000"}, Confidence: 0.1},
		profiles,
		defaultMaxFanoutWidth,
	)
	require.Len(t, chosen, defaultMaxFanoutWidth)
	require.True(t, widened)
	require.True(t, bound)
	require.Equal(t, "command-0000", chosen[0])
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
	require.Len(t, profiles, 3)
	require.Equal(t, "a.x", profiles[0].ProviderID)
	require.Equal(t, "first", profiles[0].Description, "leaf descriptions remain unmodified")
	require.Equal(t, "b.y", profiles[1].ProviderID)
	require.Equal(t, "second", profiles[1].Description)
	require.Equal(t, "c.z", profiles[2].ProviderID)
}

func TestAvailableTypes_DistinctSorted(t *testing.T) {
	active := []*registryv1.ProviderDescriptor{
		descWithEndpoint("a", "g", "record", "d"),
		descWithEndpoint("b", "g", "command", "d"),
		descWithEndpoint("c", "g", "command", "d"),
	}
	profiles := buildProfiles(active)
	require.Equal(t, []string{"a", "b", "c"}, availableProviderIDs(profiles))
}

// --- OllamaClassifier with a seamed runner ---------------------------------

func TestOllamaClassifier_Classify_UsesRunnerOutput(t *testing.T) {
	var gotRole, gotPrompt string
	c := &OllamaClassifier{
		role:      "classify.routing",
		maxTokens: classifierMaxTokens,
		generate: func(_ context.Context, role, prompt string, _ int) ([]byte, error) {
			gotRole, gotPrompt = role, prompt
			return []byte(`{"response":"{\"provider_ids\":[\"cli-health.commands\"],\"confidence\":0.9}"}`), nil
		},
	}
	res, err := c.Classify(context.Background(), "restart a scenario",
		[]ProviderProfile{{ProviderID: "cli-health.commands", Type: "command", Group: "cli-health", Description: "CLI commands"}})
	require.NoError(t, err)
	require.Equal(t, []string{"cli-health.commands"}, res.ProviderIDs)
	require.Equal(t, "classify.routing", gotRole)
	require.Contains(t, gotPrompt, "CLI commands", "the provider description must reach the prompt")
	require.Contains(t, gotPrompt, "cli-health.commands", "the exact provider id must reach the prompt")
	require.Contains(t, gotPrompt, "restart a scenario")
}

func TestOllamaClassifier_ClassifyAddsDescriptionBackedExternalRecall(t *testing.T) {
	c := &OllamaClassifier{
		generate: func(context.Context, string, string, int) ([]byte, error) {
			return []byte(`{"provider_ids":["knowledge-observatory.docs"],"confidence":0.9}`), nil
		},
	}
	profiles := []ProviderProfile{
		{ProviderID: "knowledge-observatory.docs", Type: "doc", Description: "Project documentation and guides."},
		{ProviderID: "web-search.learnings", Type: "learning", Description: "Cited findings about the external world, software releases, and current events."},
	}

	got, err := c.Classify(context.Background(), "key features of Go 1.26", profiles)
	require.NoError(t, err)
	require.Contains(t, got.ProviderIDs, "web-search.learnings")

	got, err = c.Classify(context.Background(), "where is the retry logic in this project", profiles)
	require.NoError(t, err)
	require.NotContains(t, got.ProviderIDs, "web-search.learnings")
}

func TestBuildClassifierPromptPrefersEvidenceForImplementationQuestions(t *testing.T) {
	prompt := buildClassifierPrompt("where is the retry logic for agent runs", []ProviderProfile{
		{ProviderID: "code-reference.code", Type: "code", Group: "code-reference", Description: "source locations and call paths"},
		{ProviderID: "source-ledger.agent-memory", Type: "record", Group: "source-ledger", Description: "durable memory of prior implementation decisions"},
	})
	require.Contains(t, prompt, "implementation questions")
	require.Contains(t, prompt, "narrative record or memory corpus")
	require.Contains(t, prompt, "history, decisions, or prior work")
	require.Contains(t, prompt, "never route only to skill or record leaves")
}

func TestBuildClassifierPromptBoundsCandidateDescriptions(t *testing.T) {
	profiles := make([]ProviderProfile, classifierMaxProfiles+5)
	for i := range profiles {
		profiles[i] = ProviderProfile{ProviderID: fmt.Sprintf("provider-%d", i), Type: "code", Description: strings.Repeat("x", 500)}
	}
	prompt := buildClassifierPrompt("where is the answer", profiles)
	require.Contains(t, prompt, "candidate descriptions truncated")
	require.LessOrEqual(t, len(prompt), classifierMaxDescriptionBytes+3200, "prompt cap must remain bounded including instructions")
}

func TestBuildClassifierPromptNamesIndexOmissions(t *testing.T) {
	prompt := buildClassifierPrompt("where is the answer", []ProviderProfile{
		{ProviderID: "provider-0", Type: "code", Description: "code evidence", OmittedProviderIDs: []string{"provider-9", "provider-10"}},
	})
	require.Contains(t, prompt, "omitted provider_ids: provider-9, provider-10")
}

func TestOllamaClassifier_Classify_RunnerErrorPropagates(t *testing.T) {
	c := &OllamaClassifier{
		generate: func(_ context.Context, _, _ string, _ int) ([]byte, error) {
			return nil, errors.New("daemon down")
		},
	}
	_, err := c.Classify(context.Background(), "q", []ProviderProfile{{ProviderID: "command", Type: "command", Description: "d"}})
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
