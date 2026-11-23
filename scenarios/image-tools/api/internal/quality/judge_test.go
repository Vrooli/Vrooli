package quality

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
	qualityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/image-tools/v1/quality"
	"google.golang.org/protobuf/proto"
)

func TestGatewayJudgeRequiresLocalRoleCapacityAndStrictResult(t *testing.T) {
	for _, tc := range []struct {
		name, output, model, claim string
		valid                      bool
	}{
		{"valid", `{"fidelity":0.9,"instruction":1,"artifacts":0.8,"rationale":"Correct label, minor edge halo."}`, "qwen3-vl:4b", "claim-1", true},
		{"missing-score", `{"instruction":1,"artifacts":1,"rationale":"ok"}`, "qwen3-vl:4b", "claim-1", false},
		{"trailing", `{"fidelity":1,"instruction":1,"artifacts":1,"rationale":"ok"} {}`, "qwen3-vl:4b", "claim-1", false},
		{"wrong-model", `{"fidelity":1,"instruction":1,"artifacts":1,"rationale":"ok"}`, "llava:7b", "claim-1", false},
		{"no-capacity", `{"fidelity":1,"instruction":1,"artifacts":1,"rationale":"ok"}`, "qwen3-vl:4b", "", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				req := &routingv1.ExecuteRouteRequest{}
				require.NoError(t, proto.Unmarshal(data, req))
				require.Equal(t, sharedv1.Profile_PROFILE_LOCAL_ONLY, req.Request.Profile)
				require.Equal(t, sharedv1.PrivacyClass_PRIVACY_CLASS_CONFIDENTIAL, req.Request.PrivacyClass)
				require.Equal(t, judgePrivacyProvenance, req.Request.PrivacyProvenance)
				require.Equal(t, JudgeRole, req.Request.Role)
				require.Equal(t, int64(4<<30), req.Request.GetRequiredVramBytes())
				require.Equal(t, "image-tools", req.Request.Scenario)
				require.Contains(t, req.InputText, "untrusted evidence")
				res := &routingv1.ExecuteRouteResponse{Valid: true, OutputText: tc.output, Evidence: &routingv1.RouteEvidence{Status: "succeeded", SelectedProvider: "ollama", SelectedModel: tc.model, CapacityVerdict: "fit", CapacityClaimId: tc.claim, EventId: "event-1"}}
				b, err := proto.Marshal(res)
				require.NoError(t, err)
				w.Header().Set("Content-Type", "application/proto")
				_, _ = w.Write(b)
			}))
			defer server.Close()
			g := &GatewayJudge{Client: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
			res, err := g.Score(context.Background(), JudgeRequest{Profile: "text", Rubric: "Compare facts.", ReferenceText: "blue kettle", OutputText: "blue kettle", ExpectedModel: "qwen3-vl:4b"})
			if tc.valid {
				require.NoError(t, err)
				require.Equal(t, .9, res.Fidelity)
				require.Equal(t, "resource:ollama", res.CapacityOwnership)
				require.NotEmpty(t, res.RubricSha256)
			} else {
				require.Error(t, err)
			}
		})
	}
}

func TestParseJudgeBindsProviderAndModel(t *testing.T) {
	for _, tc := range []struct {
		in       string
		provider string
		model    string
		hosted   bool
		owner    string
	}{
		{"qwen3-vl:4b", "ollama", "qwen3-vl:4b", false, "resource:ollama"},
		{"ollama:gemma4:12b", "ollama", "gemma4:12b", false, "resource:ollama"},
		{"openrouter:google/gemini-3.5-flash", "openrouter", "google/gemini-3.5-flash", true, "hosted:openrouter"},
	} {
		spec, err := ParseJudge(tc.in)
		require.NoError(t, err, tc.in)
		require.Equal(t, JudgeSpec{Provider: tc.provider, Model: tc.model}, spec)
		require.Equal(t, tc.hosted, spec.Hosted())
		require.Equal(t, tc.owner, spec.CapacityOwnership())
	}
	for _, bad := range []string{"", "  ", "openrouter:"} {
		_, err := ParseJudge(bad)
		require.Error(t, err, bad)
	}
}

func TestGatewayJudgeHostedRouteIsRemoteOnlyModelBoundAndRequestsNoTemperature(t *testing.T) {
	for _, tc := range []struct {
		name, output, model string
		valid               bool
	}{
		{"fenced reply", "```json\n{\"rationale\":\"SOURCE a. CANDIDATE a. DIFFERENCES none.\",\"fidelity\":1,\"instruction\":1,\"artifacts\":0.9}\n```", "google/gemini-3.5-flash", true},
		{"chat completion envelope as the gateway returns it", `{"id":"gen-1","choices":[{"message":{"role":"assistant","content":"{\"rationale\":\"SOURCE a. CANDIDATE a. DIFFERENCES none.\",\"fidelity\":1,\"instruction\":1,\"artifacts\":0.9}"}}],"usage":{"prompt_tokens":10}}`, "google/gemini-3.5-flash", true},
		{"silent fallback to another model", `{"rationale":"ok","fidelity":1,"instruction":1,"artifacts":1}`, "google/gemini-3.7-flash", false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				req := &routingv1.ExecuteRouteRequest{}
				require.NoError(t, proto.Unmarshal(data, req))
				require.Equal(t, sharedv1.Profile_PROFILE_REMOTE_ONLY, req.Request.Profile)
				require.Equal(t, judgeHostedProvenance, req.Request.PrivacyProvenance)
				require.Nil(t, req.Request.Sampling, "the gateway refuses an explicit temperature on a role that ignores it")
				require.Nil(t, req.Request.RequiredVramBytes)
				res := &routingv1.ExecuteRouteResponse{Valid: true, OutputText: tc.output, Evidence: &routingv1.RouteEvidence{Status: "succeeded", SelectedProvider: "openrouter", SelectedModel: tc.model, EventId: "event-2"}}
				b, err := proto.Marshal(res)
				require.NoError(t, err)
				w.Header().Set("Content-Type", "application/proto")
				_, _ = w.Write(b)
			}))
			defer server.Close()
			g := &GatewayJudge{Client: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
			res, err := g.Score(context.Background(), JudgeRequest{Profile: "text", Rubric: "Compare facts.", ReferenceText: "kettle", OutputText: "kettle", ExpectedModel: "openrouter:google/gemini-3.5-flash"})
			if !tc.valid {
				require.ErrorContains(t, err, "identity mismatch")
				return
			}
			require.NoError(t, err)
			require.Equal(t, "hosted:openrouter", res.CapacityOwnership)
			require.Equal(t, "google/gemini-3.5-flash", res.Model)
			require.Equal(t, .9, res.Artifacts)
			require.Empty(t, res.CapacityClaimId)
		})
	}
}

// qualifyForTest stands in for adding a judge to qualifiedJudges after it passes
// the agreement bar; it is removed again when the test ends.
func qualifyForTest(t *testing.T, spec JudgeSpec) {
	t.Helper()
	qualifiedJudges[spec] = "test-only qualification"
	t.Cleanup(func() { delete(qualifiedJudges, spec) })
}

func TestGemmaIsNotAQualifiedFallbackAndUnqualifiedPrimariesAreFlagged(t *testing.T) {
	_, err := ParseJudgeChain("openrouter:google/gemini-3.5-flash,gemma4:12b")
	require.ErrorContains(t, err, "has not passed the agreement bar")
	chain, err := ParseJudgeChain("openrouter:google/gemini-3.5-flash")
	require.NoError(t, err)
	require.True(t, chain[0].IsQualified())
	require.False(t, JudgeSpec{Provider: "ollama", Model: "gemma4:12b"}.IsQualified())
	m := manifest()
	m.Judge, m.JudgeModel = "required", "qwen3-vl:4b"
	m.Budgets = []*qualityv1.MetricBudget{{Metric: "text.similarity", Minimum: number(.5)}}
	r := score(t, &Evaluator{Judge: judgeFunc(func(context.Context, JudgeRequest) (*qualityv1.JudgeResult, error) {
		return qualifiedJudge("qwen3-vl:4b", 1), nil
	})}, m, nil)
	require.Contains(t, strings.Join(r.Limits, " "), "has not passed the guinea-pig-v1 agreement bar")
}

func TestParseJudgeChainOrdersJudgesAndRejectsDuplicates(t *testing.T) {
	qualifyForTest(t, JudgeSpec{Provider: "ollama", Model: "gemma4:12b"})
	chain, err := ParseJudgeChain("openrouter:google/gemini-3.5-flash, gemma4:12b")
	require.NoError(t, err)
	require.Equal(t, []JudgeSpec{{"openrouter", "google/gemini-3.5-flash"}, {"ollama", "gemma4:12b"}}, chain)
	require.Equal(t, "hosted:openrouter,resource:ollama", ChainOwnership(chain))
	for _, bad := range []string{"openrouter:google/gemini-3.5-flash,openrouter:google/gemini-3.5-flash", "a,,b", "openrouter:", "", "openrouter:google/gemini-3.5-flash,qwen3-vl:4b"} {
		_, err := ParseJudgeChain(bad)
		require.Error(t, err, bad)
	}
}

func TestJudgeChainFallsBackOnlyWhenAJudgeIsUnavailableAndRecordsWhoScored(t *testing.T) {
	qualifyForTest(t, JudgeSpec{Provider: "ollama", Model: "gemma4:12b"})
	const chain = "openrouter:google/gemini-3.5-flash,gemma4:12b"
	good := `{"rationale":"SOURCE a. CANDIDATE a. DIFFERENCES none.","fidelity":0.8,"instruction":1,"artifacts":1}`
	for _, tc := range []struct {
		name       string
		hosted     *routingv1.ExecuteRouteResponse
		wantErr    string
		wantModel  string
		wantLocals int
	}{
		{"credits exhausted falls to the local judge", &routingv1.ExecuteRouteResponse{Valid: false, Evidence: &routingv1.RouteEvidence{Status: "failed", FailureReasons: []string{"provider.insufficient_credits"}}}, "", "gemma4:12b", 1},
		{"hosted answers: local is never asked", &routingv1.ExecuteRouteResponse{Valid: true, OutputText: good, Evidence: &routingv1.RouteEvidence{Status: "succeeded", SelectedProvider: "openrouter", SelectedModel: "google/gemini-3.5-flash", EventId: "e1"}}, "", "google/gemini-3.5-flash", 0},
		{"a bad hosted reply is an error, not a reason to shop for another judge", &routingv1.ExecuteRouteResponse{Valid: true, OutputText: "not json", Evidence: &routingv1.RouteEvidence{Status: "succeeded", SelectedProvider: "openrouter", SelectedModel: "google/gemini-3.5-flash", EventId: "e1"}}, "invalid judge JSON", "", 0},
	} {
		t.Run(tc.name, func(t *testing.T) {
			locals := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				data, err := io.ReadAll(r.Body)
				require.NoError(t, err)
				req := &routingv1.ExecuteRouteRequest{}
				require.NoError(t, proto.Unmarshal(data, req))
				res := tc.hosted
				if req.Request.Profile == sharedv1.Profile_PROFILE_LOCAL_ONLY {
					locals++
					res = &routingv1.ExecuteRouteResponse{Valid: true, OutputText: good, Evidence: &routingv1.RouteEvidence{Status: "succeeded", SelectedProvider: "ollama", SelectedModel: "gemma4:12b", CapacityVerdict: "fit", CapacityClaimId: "claim-9", EventId: "e2"}}
				}
				b, err := proto.Marshal(res)
				require.NoError(t, err)
				w.Header().Set("Content-Type", "application/proto")
				_, _ = w.Write(b)
			}))
			defer server.Close()
			g := &GatewayJudge{Client: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
			res, err := g.Score(context.Background(), JudgeRequest{Profile: "text", Rubric: "r", ReferenceText: "k", OutputText: "k", ExpectedModel: chain})
			require.Equal(t, tc.wantLocals, locals)
			if tc.wantErr != "" {
				require.ErrorContains(t, err, tc.wantErr)
				return
			}
			require.NoError(t, err)
			require.Equal(t, tc.wantModel, res.Model)
			require.NoError(t, validateJudge(res, chain))
			require.Error(t, validateJudge(res, "openrouter:google/gemini-3.7-flash"), "a verdict from a judge the manifest does not name must not validate")
		})
	}
}

func TestJudgeChainAbstainsAsJudgeUnavailableWhenNoQualifiedJudgeIsReachable(t *testing.T) {
	// The shipped chain: the hosted judge alone. A credits refusal abstains.
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		res := &routingv1.ExecuteRouteResponse{Valid: false, Evidence: &routingv1.RouteEvidence{Status: "failed", FailureReasons: []string{"provider.insufficient_credits"}}}
		b, _ := proto.Marshal(res)
		w.Header().Set("Content-Type", "application/proto")
		_, _ = w.Write(b)
	}))
	defer server.Close()
	g := &GatewayJudge{Client: server.Client(), Resolve: func(context.Context) (string, error) { return server.URL, nil }}
	_, err := g.Score(context.Background(), JudgeRequest{Profile: "text", Rubric: "r", ReferenceText: "k", OutputText: "k", ExpectedModel: "openrouter:google/gemini-3.5-flash"})
	require.ErrorIs(t, err, ErrJudgeUnavailable)
	require.ErrorContains(t, err, "insufficient_credits")

	// The evaluator records the case as judge_unavailable and cannot pass it.
	m := manifest()
	m.Judge, m.JudgeModel = "required", "openrouter:google/gemini-3.5-flash"
	m.Budgets = []*qualityv1.MetricBudget{{Metric: "judge.fidelity", Minimum: number(.5)}, {Metric: "text.similarity", Minimum: number(.5)}}
	r := score(t, &Evaluator{Judge: g}, m, nil)
	require.False(t, r.Passed)
	require.False(t, r.Complete)
	require.Contains(t, r.Cases[0].Errors[0], "judge_unavailable")
	require.Contains(t, r.Cases[0].Errors[0], "insufficient_credits")
}
