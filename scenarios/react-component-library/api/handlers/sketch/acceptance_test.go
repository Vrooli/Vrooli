package sketch

import (
	"connectrpc.com/connect"
	"context"
	"github.com/stretchr/testify/require"
	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"
	"os"
	"path/filepath"
	internal "react-component-library/internal/sketch"
	"strings"
	"testing"
)

func TestAcceptancePreflightDetectsStaleInputsWithoutAccepting(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "scenarios", "demo", "experience", "pages", "home.json")
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0700))
	require.NoError(t, os.WriteFile(path, []byte(`{"sketch":{}}`), 0600))
	store := internal.NewStore(root)
	base, err := store.Read("demo", "home")
	require.NoError(t, err)
	doc := internal.Document{Template: &internal.AssetRef{Asset: "templates.page", Version: "1.0.0"}, Render: &internal.RenderSettings{TemplateExport: "Page", Bindings: map[string]any{"$template": map[string]any{}}}}
	candidate, err := store.SaveCandidate("demo", "home", "candidate-one", base.ContentHash, doc)
	require.NoError(t, err)
	h := NewConnectHandler(Deps{Store: store, Renderer: &renderProbe{}})
	request := &sketchv1.CheckCandidateAcceptanceRequest{Render: &sketchv1.RenderCandidateRequest{Candidate: &sketchv1.CandidateReference{Scenario: "demo", DesignId: candidate.DesignID, Hash: candidate.Hash}, MissingLabel: "Missing", FailedLabel: "Failed"}, ExpectedRenderHash: candidate.Hash}
	check := func() map[string]string {
		result, e := h.CheckCandidateAcceptance(context.Background(), connect.NewRequest(request))
		require.NoError(t, e)
		require.False(t, result.Msg.Ready)
		require.False(t, result.Msg.AcceptanceEstablished)
		rows := map[string]string{}
		for _, r := range result.Msg.Requirements {
			rows[r.Code] = r.Status
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
	decisionRequest := connect.NewRequest(&sketchv1.AcceptCandidateRequest{Check: request, Actor: "test-agent", IdempotencyKey: "decision-one"})
	decision, e := h.AcceptCandidate(context.Background(), decisionRequest)
	require.NoError(t, e)
	require.Equal(t, "needs_evidence", decision.Msg.State)
	require.Equal(t, candidate.Hash, decision.Msg.Intent.CandidateHash)
	require.NotEmpty(t, decision.Msg.Hash)
	calls := h.deps.Renderer.(*renderProbe).calls
	replay, e := h.AcceptCandidate(context.Background(), decisionRequest)
	require.NoError(t, e)
	require.Equal(t, decision.Msg.Hash, replay.Msg.Hash)
	require.Equal(t, calls, h.deps.Renderer.(*renderProbe).calls)
	stored, e := h.GetAcceptance(context.Background(), connect.NewRequest(&sketchv1.GetAcceptanceRequest{Scenario: "demo", DesignId: candidate.DesignID, Id: decision.Msg.Id}))
	require.NoError(t, e)
	require.Equal(t, decision.Msg.Hash, stored.Msg.Hash)

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
	h := NewConnectHandler(Deps{})
	_, err := h.CheckCandidateAcceptance(context.Background(), connect.NewRequest(&sketchv1.CheckCandidateAcceptanceRequest{}))
	require.Equal(t, connect.CodeInvalidArgument, connect.CodeOf(err))
}
