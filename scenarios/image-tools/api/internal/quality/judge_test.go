package quality

import (
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/require"
	routingv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/routing"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
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
