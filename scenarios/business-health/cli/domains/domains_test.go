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
		"validate":   {"scenario"},
		"matrix":     {"show"},
		"drift":      {"show"},
		"manual-log": {"add"},
		"fix":        {"preview", "apply"},
		"fleet":      {"scan"},
		"wizard":     {"start", "answer", "preview", "apply"},
		"search":     {"query", "status"},
		"reindex":    {"run", "status", "cancel"},
	})
}
