package activity

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	activityv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity"
	activityconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/activity/activity_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestActivityManifestCoverage(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	cliapp.RequireProtoServiceCoverage(t, raw, activityv1.File_react_component_library_v1_activity_activity_proto, "ActivityService")
}

func TestActivityGroupRegistersEveryManifestCommand(t *testing.T) {
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

func TestActivityEventsForwardsFilters(t *testing.T) {
	var got *activityv1.ListActivityRequest
	mux := http.NewServeMux()
	mux.Handle(activityconnect.ActivityServiceListActivityProcedure, connect.NewUnaryHandlerSimple(
		activityconnect.ActivityServiceListActivityProcedure,
		func(_ context.Context, req *activityv1.ListActivityRequest) (*activityv1.ListActivityResponse, error) {
			got = req
			return &activityv1.ListActivityResponse{Events: []*activityv1.ActivityEvent{{
				Id:          "evt-1",
				Actor:       "human",
				Action:      "workspace.save",
				SubjectKind: "workspace",
				SubjectId:   "ws-1",
				Summary:     "saved Dashboard",
				At:          "2026-11-19T10:00:00Z",
			}}}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "events")

	out, err := cliapptest.RunCommand(t, command, core,
		"--subject-kind", "workspace",
		"--subject-id", "ws-1",
		"--actor", "human",
		"--limit", "10",
		"--since", "2026-01-01T00:00:00Z",
	)
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, "workspace", got.GetSubjectKind())
	require.Equal(t, "ws-1", got.GetSubjectId())
	require.Equal(t, "human", got.GetActor())
	require.Equal(t, int32(10), got.GetLimit())
	require.Equal(t, "2026-01-01T00:00:00Z", got.GetSince())

	require.Contains(t, out, "Found 1 activity event(s).")
	require.Contains(t, out, "workspace.save")
	require.Contains(t, out, "saved Dashboard")
}
