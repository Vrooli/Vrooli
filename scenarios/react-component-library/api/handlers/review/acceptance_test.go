package review

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	sketchh "react-component-library/handlers/sketch"
	"react-component-library/internal/preview"
	internal "react-component-library/internal/sketch"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	previewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/preview"
	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
)

type acceptanceRenderProbe struct{ calls int }

func (p *acceptanceRenderProbe) RenderPrepared(_ context.Context, prepared preview.PreparedComposition, _ *previewv1.RenderCompositionRequest) (*connect.Response[sharedv1.RenderCompositionResponse], error) {
	p.calls++
	return connect.NewResponse(&sharedv1.RenderCompositionResponse{Html: "<main>fixture</main>", RenderHash: prepared.Composition.Revision}), nil
}

func TestAcceptancePreflightDetectsStaleInputsWithoutAccepting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o700))
	require.NoError(t, os.WriteFile(path, []byte(`{"sketch":{}}`), 0o600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	doc := internal.Document{Template: &internal.AssetRef{Asset: "templates.page", Version: "1.0.0"}, Render: &internal.RenderSettings{TemplateExport: "Page", Bindings: map[string]any{"$template": map[string]any{}}}}
	candidate, err := store.SaveCandidate("demo", "home", "candidate-one", base.ContentHash, doc)
	require.NoError(t, err)
	renderer := &acceptanceRenderProbe{}
	evaluator := sketchh.NewConnectHandler(sketchh.Deps{Store: store, Renderer: renderer})
	h := NewHandlerWithPorts(nil, Ports{
		AcceptanceEvaluator: evaluator,
		AcceptanceStoreFor:  func(context.Context) (AcceptanceStore, error) { return store, nil },
	})
	request := &reviewv1.CheckCandidateAcceptanceRequest{Render: &sharedv1.RenderCandidateRequest{Candidate: &sharedv1.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash}, MissingLabel: "Missing", FailedLabel: "Failed"}, ExpectedRenderHash: candidate.Hash}
	check := func() map[string]string {
		result, e := h.CheckCandidateAcceptance(context.Background(), connect.NewRequest(request))
		require.NoError(t, e)
		require.False(t, result.Msg.GetReady())
		require.False(t, result.Msg.GetAcceptanceEstablished())
		rows := map[string]string{}
		for _, r := range result.Msg.GetRequirements() {
			rows[r.GetCode()] = r.GetStatus()
		}
		return rows
	}
	rows := check()
	require.Equal(t, "passed", rows["page_freshness"])
	require.Equal(t, "passed", rows["render_freshness"])
	require.Equal(t, "unavailable", rows["render_completeness"])
	require.Equal(t, "blocked", rows["visual_review"])
	require.Equal(t, "unavailable", rows["behavior_evidence"])
	require.Equal(t, "unavailable", rows["rubric_calibration"])
	require.Equal(t, "unavailable", rows["independent_review"])
	decisionRequest := connect.NewRequest(&reviewv1.AcceptCandidateRequest{Check: request, Actor: "test-agent", IdempotencyKey: "decision-one"})
	decision, e := h.AcceptCandidate(context.Background(), decisionRequest)
	require.NoError(t, e)
	require.Equal(t, "needs_evidence", decision.Msg.GetState())
	require.Equal(t, candidate.Hash, decision.Msg.GetIntent().GetCandidateHash())
	require.NotEmpty(t, decision.Msg.GetHash())
	calls := renderer.calls
	replay, e := h.AcceptCandidate(context.Background(), decisionRequest)
	require.NoError(t, e)
	require.Equal(t, decision.Msg.GetHash(), replay.Msg.GetHash())
	require.Equal(t, calls, renderer.calls)
	stored, e := h.GetAcceptance(context.Background(), connect.NewRequest(&reviewv1.GetAcceptanceRequest{Scenario: "demo", DesignId: candidate.DesignID, Id: decision.Msg.GetId()}))
	require.NoError(t, e)
	require.Equal(t, decision.Msg.GetHash(), stored.Msg.GetHash())

	_, err = store.Save("demo", "home", base.ContentHash, internal.Document{Viewport: "phone"})
	require.NoError(t, err)
	request.ExpectedRenderHash = strings.Repeat("f", 64)
	rows = check()
	require.Equal(t, "blocked", rows["page_freshness"])
	require.Equal(t, "blocked", rows["render_freshness"])
	current, err := store.Read("demo", "home")
	require.NoError(t, err)
	require.Equal(t, "phone", current.Document.Viewport)
}

func TestAcceptancePreflightRequiresExactRender(t *testing.T) {
	h := NewHandlerWithPorts(nil, Ports{})
	_, err := h.CheckCandidateAcceptance(context.Background(), connect.NewRequest(&reviewv1.CheckCandidateAcceptanceRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
