package review

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	reviewv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review"
	reviewconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/review/review_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestReviewManifestCoverage(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	cliapp.RequireProtoServiceCoverage(t, raw, reviewv1.File_react_component_library_v1_review_review_proto, "ReviewService")
}

func TestReviewGroupRegistersEveryManifestCommand(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	group, err := Register(&cliapp.ScenarioApp{}, raw)
	require.NoError(t, err)

	manifest, err := cliapp.ParseManifest(raw)
	require.NoError(t, err)
	declared := manifest.FindGroup(GroupName)
	require.NotNil(t, declared)
	require.Len(t, group.Subcommands, len(declared.Commands))

	for _, manifestCommand := range declared.Commands {
		command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, manifestCommand.Name)
		require.NotNilf(t, command.RunCtx, "command %s/%s has no bound handler", GroupName, manifestCommand.Name)
		require.NotNilf(t, manifestCommand.Architecture, "command %s/%s declares no architecture", GroupName, manifestCommand.Name)
		require.Equalf(t, manifestCommand.Architecture.Primitive, string(command.PrimitiveEvidence()),
			"command %s/%s primitive evidence does not match its manifest declaration", GroupName, manifestCommand.Name)
	}
}

func TestReviewDispositionForwardsDecisionAndActor(t *testing.T) {
	var got *reviewv1.RecordDispositionRequest
	mux := http.NewServeMux()
	mux.Handle(reviewconnect.ReviewServiceRecordDispositionProcedure, connect.NewUnaryHandlerSimple(
		reviewconnect.ReviewServiceRecordDispositionProcedure,
		func(_ context.Context, req *reviewv1.RecordDispositionRequest) (*reviewv1.RecordDispositionResponse, error) {
			got = req
			return &reviewv1.RecordDispositionResponse{Finding: &reviewv1.Finding{
				Id:       req.GetId(),
				Severity: reviewv1.FindingSeverity_FINDING_SEVERITY_MEDIUM,
				Status:   reviewv1.FindingStatus_FINDING_STATUS_ACCEPTED,
				Message:  "contrast below threshold",
				Owner:    "design-systems",
			}}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "disposition")

	out, err := cliapptest.RunCommand(t, command, core, "fnd-1", "--decision", "accept-risk", "--reason", "accepted for launch", "--actor", "human")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "fnd-1", got.GetId())
	require.Equal(t, reviewv1.DispositionDecision_DISPOSITION_DECISION_ACCEPT_RISK, got.GetDecision())
	require.Equal(t, "accepted for launch", got.GetReason())
	require.Equal(t, "human", got.GetActor())

	require.Contains(t, out, "Recorded accept-risk disposition on finding fnd-1.")
	require.Contains(t, out, "fnd-1 [medium/accepted]")
}
