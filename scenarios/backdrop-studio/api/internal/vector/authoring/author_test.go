package authoring

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"

	"backdrop-studio/internal/vector"
)

// fakeClient stands in for ai-gateway. Authoring costs money, so every path
// below has to be reachable without spending any.
type fakeClient struct {
	answer string
	model  string
	err    error
	prompt string
	calls  int
}

func (c *fakeClient) Author(_ context.Context, prompt string) (string, string, error) {
	c.calls++
	c.prompt = prompt
	if c.err != nil {
		return "", "", c.err
	}
	return c.answer, c.model, nil
}

const modelAnswer = `{
  "name": "Authored Drift",
  "template": "<rect width=\"{{f .W}}\" height=\"{{f .H}}\" fill=\"{{paper}}\"/>{{$fr := .}}{{range $i := seq ($fr.Param \"marks\")}}<circle cx=\"{{f (mul $fr.W ($fr.Rand $i))}}\" cy=\"{{f (mul $fr.H (add 0.15 (mul 0.7 ($fr.Rand (add $i 17)))))}}\" r=\"{{f (mul $fr.H 0.009)}}\" fill=\"{{ink}}\"/>{{end}}",
  "params": [{"name": "marks", "min": 40, "max": 800, "default": 300, "description": "mark count"}],
  "inks": ["$brand.background", "$brand.primary"]
}`

func TestAuthoringValidatesAndRecordsItsProvenance(t *testing.T) {
	client := &fakeClient{answer: modelAnswer, model: "frontier/some-model"}
	generator, report, err := Author(context.Background(), client, Request{ID: "authored-drift", Brief: "a drifting field of marks"})
	require.NoError(t, err)
	require.Truef(t, report.Passed, "refusals: %v", report.Refusals)

	require.Equal(t, "authored-drift", generator.ID, "the caller's id wins over anything the model returned")
	require.Equal(t, "frontier/some-model", generator.ModelID, "the model the gateway resolved, never one we asked for")
	require.NotEmpty(t, generator.Prompt, "the prompt is stored so the generator can be re-authored and reviewed")
	require.Equal(t, report, generator.Validation)
}

// The prompt and the validator are two halves of one specification: every
// rejection path the validator has corresponds to a sentence in the prompt. A
// model told nothing about a variable frame places marks in pixels.
func TestThePromptStatesEveryRuleTheValidatorEnforces(t *testing.T) {
	prompt := Prompt(Request{ID: "x", Brief: "a brief"})
	for _, required := range []string{
		"fraction of .W or .H",  // composition_holds
		"must appear in params", // params_declared
		"Deterministic",         // deterministic
		"<script>",              // no_active_content
		"http://",               // no_active_content
		"paper/ink/accent",      // inks_resolve
		"density of marks",      // the reason this lane exists at all
	} {
		require.Containsf(t, prompt, required,
			"the prompt must state the rule the validator enforces, or the rejection rate is the prompt's fault")
	}
	require.Contains(t, prompt, "a brief", "the operator's art direction reaches the model")
}

func TestAuthoringRefusesWithNoClient(t *testing.T) {
	_, _, err := Author(context.Background(), nil, Request{ID: "x", Brief: "y"})
	require.ErrorContains(t, err, "unavailable on this host")
}

func TestAuthoringRefusesAnEmptyBrief(t *testing.T) {
	client := &fakeClient{answer: modelAnswer, model: "m"}
	_, _, err := Author(context.Background(), client, Request{ID: "x", Brief: "   "})
	require.ErrorContains(t, err, "brief is required")
	require.Zero(t, client.calls, "an unusable request must not spend a model call")
}

// A gateway failure is reported with the role named, because the operator's
// next action — configure a credential, install a local model — depends on it.
func TestAGatewayFailureNamesTheRole(t *testing.T) {
	client := &fakeClient{err: errors.New("no eligible route")}
	_, _, err := Author(context.Background(), client, Request{ID: "x", Brief: "y"})
	require.ErrorContains(t, err, Role)
	require.ErrorContains(t, err, "no eligible route")
}

// Models habitually wrap JSON in a fenced block. Tolerating that is worth one
// helper; tolerating arbitrary prose around it is not, and an unreadable answer
// says what was expected rather than producing an empty generator.
func TestAFencedAnswerIsReadAndAnUnreadableOneIsNamed(t *testing.T) {
	fenced := &fakeClient{answer: "Here you go:\n```json\n" + modelAnswer + "\n```\n", model: "m"}
	generator, report, err := Author(context.Background(), fenced, Request{ID: "authored-drift", Brief: "b"})
	require.NoError(t, err)
	require.Truef(t, report.Passed, "refusals: %v", report.Refusals)
	require.NotEmpty(t, generator.Template)

	prose := &fakeClient{answer: "I would love to help, but I need more detail.", model: "m"}
	_, _, err = Author(context.Background(), prose, Request{ID: "x", Brief: "b"})
	require.ErrorContains(t, err, "declared JSON object")
}

// A refused generator comes back with its verdict rather than as a bare error.
// The refusals are the actionable part, and the plan asks for a measured
// failure rate that nobody can compute from silence.
func TestARefusedGeneratorIsReturnedWithItsVerdict(t *testing.T) {
	answer := strings.Replace(modelAnswer, `"params": [{"name": "marks", "min": 40, "max": 800, "default": 300, "description": "mark count"}]`, `"params": []`, 1)
	client := &fakeClient{answer: answer, model: "m"}
	generator, report, err := Author(context.Background(), client, Request{ID: "authored-drift", Brief: "b"})
	require.NoError(t, err, "a refusal is a verdict, not a transport failure")
	require.False(t, report.Passed)
	require.NotEmpty(t, report.Refusals)
	require.NotEmpty(t, generator.Template, "the refused generator is returned so its author can be shown what to fix")
	require.Equal(t, report, generator.Validation)
}

// The role is a role. A concrete model slug here would be refused by
// ai-gateway's own conformance policy, and would need editing every time the
// provider catalog moved.
func TestTheRoleIsARoleAndNotAModelSlug(t *testing.T) {
	require.Equal(t, "author.generator", Role)
	require.NotContains(t, Role, "/", "a slash is how a provider names a concrete model")
	require.NotContains(t, Role, ":")
}

// An authored generator must never be able to shadow a hand-written one, which
// is asserted against the real preset list rather than a copy of it.
func TestNoBuiltInPresetCanBeAuthoredOver(t *testing.T) {
	client := &fakeClient{answer: modelAnswer, model: "m"}
	for _, preset := range vector.Presets {
		_, report, err := Author(context.Background(), client, Request{ID: preset, Brief: "b"})
		require.NoError(t, err)
		require.Falsef(t, report.Passed, "authoring over built-in preset %q was allowed", preset)
	}
}
