package domains

import (
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/vrooli/cli-core/cliapp"
	"github.com/vrooli/cli-core/cliapptest"
)

func TestCommandGroups(t *testing.T) {
	require.Nil(t, CommandGroups(&cliapp.ScenarioApp{}))
}

func TestSubcommandGroups(t *testing.T) {
	manifest := cliapptest.ReadManifest(t)
	groups, err := SubcommandGroups(&cliapp.ScenarioApp{}, manifest)
	require.NoError(t, err)

	cliapptest.RequireCommandGroups(t, groups, map[string][]string{
		"ios": {"targets", "generate", "build", "conformance-plan", "readiness", "distribution", "matrix", "matrix-catalog", "matrix-list", "matrix-create", "matrix-start", "matrix-wait", "compare", "matrix-rerun-failed"},
	})
}
