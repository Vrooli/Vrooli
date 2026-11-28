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
		"control": {"backend-status", "controllers", "enroll", "set", "off", "desired", "off-clear", "reconcile", "configure-zone"},
		"policy":  {"get", "set", "set-manual", "clear-manual", "evaluate"},
		"setup":   {"status", "install", "enable", "disable", "devices", "plan-devices", "apply-devices", "rollback-devices"},
	})
}
