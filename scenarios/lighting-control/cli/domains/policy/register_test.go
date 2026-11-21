package policy

import (
	"os"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/vrooli/cli-core/cliapp"
	policyv1 "github.com/vrooli/vrooli/packages/proto/gen/go/lighting-control/v1/policy"
)

func TestManifestCoversPolicyService(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	require.NoError(t, err)
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{Name: "lighting-control", AllowAnonymous: true})
	require.NoError(t, err)
	group, err := Register(app, manifest)
	require.NoError(t, err)
	require.Len(t, group.Subcommands, 5)
}

func TestManifestPolicyBindingMatchesGeneratedService(t *testing.T) {
	manifest, err := os.ReadFile("../../manifest.json")
	require.NoError(t, err)
	cliapp.RequireProtoServiceCoverage(t, manifest, policyv1.File_lighting_control_v1_policy_policy_proto, "PolicyService")
}
