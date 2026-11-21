package health

import (
	"context"
	"net/http"
	"testing"

	"connectrpc.com/connect"
	"github.com/stretchr/testify/require"

	libraryhealthv1 "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth"
	libraryhealthconnect "github.com/vrooli/vrooli/packages/proto/gen/go/react-component-library/v1/libraryhealth/libraryhealth_v1connect"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestHealthManifestCoverage(t *testing.T) {
	raw := cliapptest.ReadManifest(t)
	cliapp.RequireProtoServiceCoverage(t, raw, libraryhealthv1.File_react_component_library_v1_libraryhealth_library_health_proto, "LibraryHealthService")
}

func TestHealthGroupRegistersEveryManifestCommand(t *testing.T) {
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

func TestHealthShowForwardsWindowAndNeverPrintsUnavailableAsZero(t *testing.T) {
	var got *libraryhealthv1.GetLibraryHealthRequest
	mux := http.NewServeMux()
	mux.Handle(libraryhealthconnect.LibraryHealthServiceGetLibraryHealthProcedure, connect.NewUnaryHandlerSimple(
		libraryhealthconnect.LibraryHealthServiceGetLibraryHealthProcedure,
		func(_ context.Context, req *libraryhealthv1.GetLibraryHealthRequest) (*libraryhealthv1.GetLibraryHealthResponse, error) {
			got = req
			return &libraryhealthv1.GetLibraryHealthResponse{Health: &libraryhealthv1.LibraryHealth{
				Overview: &libraryhealthv1.HealthSection{
					State:   libraryhealthv1.SectionState_SECTION_STATE_AVAILABLE,
					Metrics: []*libraryhealthv1.HealthMetric{{Key: "assets", Label: "assets", Value: 3}},
				},
				Quality: &libraryhealthv1.HealthSection{
					State:  libraryhealthv1.SectionState_SECTION_STATE_UNAVAILABLE,
					Reason: "quality provider is not configured",
					// A metric from an unavailable section must never be rendered as
					// if it were a measured value.
					Metrics: []*libraryhealthv1.HealthMetric{{Key: "quality-secret", Label: "quality-secret", Value: 77}},
				},
				Activity: &libraryhealthv1.HealthSection{State: libraryhealthv1.SectionState_SECTION_STATE_EMPTY},
				Services: &libraryhealthv1.HealthSection{
					State:   libraryhealthv1.SectionState_SECTION_STATE_PARTIAL,
					Reason:  "capability registry is stale",
					Metrics: []*libraryhealthv1.HealthMetric{{Key: "wired", Label: "wired", Value: 2, Total: 5}},
				},
				Counts:      &libraryhealthv1.HealthOverview{Assets: 3, Workspaces: 1},
				GeneratedAt: "2026-11-19T10:00:00Z",
			}}, nil
		},
	))
	core := cliapptest.NewTestApp(t, mux)
	group, err := Register(core, cliapptest.ReadManifest(t))
	require.NoError(t, err)
	command := cliapptest.FindCommand(t, []cliapp.SubcommandGroup{group}, GroupName, "show")

	out, err := cliapptest.RunCommand(t, command, core, "--activity-days", "7")
	require.NoError(t, err)
	require.NotNil(t, got)
	require.Equal(t, int32(7), got.GetActivityDays())

	require.Contains(t, out, "overview: available")
	require.Contains(t, out, "assets: 3")
	require.Contains(t, out, "quality: unavailable — quality provider is not configured")
	require.NotContains(t, out, "quality-secret", "an unavailable section must not render its metrics")
	require.Contains(t, out, "activity: empty (measured zero)")
	require.Contains(t, out, "services: partial — capability registry is stale")
	require.Contains(t, out, "wired: 2/5")
	require.Contains(t, out, "counts:")
}
