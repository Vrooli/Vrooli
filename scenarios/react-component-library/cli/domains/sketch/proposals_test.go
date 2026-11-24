package sketch

import (
	"context"
	"net/http"
	"os"
	"sync"
	"testing"

	sketchv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/sketch"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
	experienceconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/experience/experience_v1connect"
	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/shared"
	workspaceassistconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspaceassist/workspaceassist_v1connect"
	workspacecandidateconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/workspacecandidate/workspacecandidate_v1connect"
)

type proposalServer struct {
	workspaceassistconnect.UnimplementedWorkspaceAssistServiceHandler
	workspacecandidateconnect.UnimplementedWorkspaceCandidateServiceHandler
	experienceconnect.UnimplementedExperienceServiceHandler
	mu               sync.Mutex
	proposed         *sharedv1.ProposeSketchRequest
	inferred         *sketchv1.InferSketchRequest
	recovered        *sketchv1.GetSketchInferenceRequest
	gets, dispatches int
}

func (s *proposalServer) GetSketch(context.Context, *connect.Request[sketchv1.GetSketchRequest]) (*connect.Response[sketchv1.GetSketchResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.gets++
	return connect.NewResponse(&sketchv1.GetSketchResponse{ContentHash: "current-revision"}), nil
}

func (s *proposalServer) ProposeSketch(_ context.Context, r *connect.Request[sharedv1.ProposeSketchRequest]) (*connect.Response[sharedv1.ProposeSketchResponse], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.proposed = r.Msg
	return connect.NewResponse(&sharedv1.ProposeSketchResponse{ContentHash: r.Msg.ExpectedContentHash, RetrievalMode: "lexical"}), nil
}

func (s *proposalServer) InferSketch(_ context.Context, r *connect.Request[sketchv1.InferSketchRequest]) (*connect.Response[sketchv1.SketchInferenceOperation], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.inferred = r.Msg
	s.dispatches++
	return nil, connect.NewError(connect.CodeUnavailable, nil)
}

func (s *proposalServer) GetSketchInference(_ context.Context, r *connect.Request[sketchv1.GetSketchInferenceRequest]) (*connect.Response[sketchv1.SketchInferenceOperation], error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.recovered = r.Msg
	return connect.NewResponse(&sketchv1.SketchInferenceOperation{Id: "existing", State: "dispatch_unknown"}), nil
}

func proposalCommands(t *testing.T, s *proposalServer) (*cliapp.ScenarioApp, []cliapp.SubcommandGroup) {
	t.Helper()
	assistPath, assistHandler := workspaceassistconnect.NewWorkspaceAssistServiceHandler(s)
	candidatePath, candidateHandler := workspacecandidateconnect.NewWorkspaceCandidateServiceHandler(s)
	experiencePath, experienceHandler := experienceconnect.NewExperienceServiceHandler(s)
	mux := http.NewServeMux()
	mux.Handle(assistPath, assistHandler)
	mux.Handle(candidatePath, candidateHandler)
	mux.Handle(experiencePath, experienceHandler)
	core := cliapptest.NewTestApp(t, mux)
	manifest, err := os.ReadFile("../../manifest.json")
	require.NoError(t, err)
	group, err := Register(core, manifest)
	require.NoError(t, err)
	return core, []cliapp.SubcommandGroup{group}
}

func TestProposeCommandCarriesTypedIntentAndReadsRevision(t *testing.T) {
	s := &proposalServer{}
	core, groups := proposalCommands(t, s)
	command := cliapptest.FindCommand(t, groups, "sketch", "propose")
	output, err := cliapptest.RunCommand(t, command, core, "demo/conversations", "--intent", "Triage and respond", "--user", "Operator", "--user", "Reviewer", "--task", "Find urgent threads", "--task", "Respond", "--design-source", "DESIGN.md", "--json")
	require.NoError(t, err)
	require.Contains(t, output, "current-revision")
	require.Equal(t, 1, s.gets)
	require.Equal(t, []string{"Operator", "Reviewer"}, s.proposed.Intent.Users)
	require.Equal(t, []string{"Find urgent threads", "Respond"}, s.proposed.Intent.PrimaryTasks)
	require.Equal(t, []string{"phone", "desktop"}, s.proposed.Intent.Viewports)
	require.Equal(t, "current-revision", s.proposed.ExpectedContentHash)
	require.Equal(t, "DESIGN.md", s.proposed.Intent.Constraints.DesignSource)
	require.True(t, s.proposed.Intent.Constraints.GetPreserveRoutes())
	require.True(t, s.proposed.Intent.Constraints.GetPreserveBusinessBehavior())
	require.Equal(t, 0, s.dispatches)
}

func TestInferErrorProvidesRecoveryWithoutRepeatingDispatch(t *testing.T) {
	s := &proposalServer{}
	core, groups := proposalCommands(t, s)
	_, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "sketch", "infer"), core, "demo/conversations", "--intent", "Respond", "--user", "Operator", "--task", "Respond", "--key", "stable-attempt", "--allow-route-changes")
	require.ErrorContains(t, err, "recover with sketch inference --key")
	require.Equal(t, "stable-attempt", s.inferred.IdempotencyKey)
	require.False(t, s.inferred.Proposal.Intent.Constraints.GetPreserveRoutes())
	output, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "sketch", "inference"), core, "--key", "stable-attempt", "--json")
	require.NoError(t, err)
	require.Contains(t, output, "dispatch_unknown")
	require.Equal(t, "stable-attempt", s.recovered.IdempotencyKey)
	require.Equal(t, 1, s.dispatches)
}

func (s *proposalServer) RenderCandidate(_ context.Context, r *connect.Request[sharedv1.RenderCandidateRequest]) (*connect.Response[sharedv1.RenderCompositionResponse], error) {
	return connect.NewResponse(&sharedv1.RenderCompositionResponse{RenderHash: "rendered-" + r.Msg.Candidate.Hash}), nil
}

func TestCandidateRenderCommandDoesNotRequireAcceptanceFlags(t *testing.T) {
	s := &proposalServer{}
	core, groups := proposalCommands(t, s)
	out, err := cliapptest.RunCommand(t, cliapptest.FindCommand(t, groups, "sketch", "render-candidate"), core, "--scenario", "demo", "--design", "home", "--candidate", "one", "--json")
	require.NoError(t, err)
	require.Contains(t, out, "rendered-one")
}
