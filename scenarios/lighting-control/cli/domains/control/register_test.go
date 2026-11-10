package control

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	controlv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/control"
)

func TestManifestCoversControlService(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	require.NoError(t, err)
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "lighting-control", AllowAnonymous: true})
	require.NoError(t, err)
	group, err := Register(app, manifest)
	require.NoError(t, err)
	require.Len(t, group.Subcommands, 8)
}

func TestManifestControlBindingMatchesGeneratedService(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	require.NoError(t, err)
	cliapp.RequireProtoServiceCoverage(t, manifest, controlv1.File_lighting_control_v1_control_control_proto, "ControlService")
}
