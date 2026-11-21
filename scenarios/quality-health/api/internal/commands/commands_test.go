package commands

import (
	"testing"

	"quality-health/internal/surfaces"

	"github.com/stretchr/testify/require"
)

func TestResolveBuildsCommandsForKnownSurfaces(t *testing.T) {
	got := Resolve(surfaces.Inventory{
		RootPath: "/repo/scenarios/demo",
		Surfaces: []surfaces.Surface{
			{ID: "ui", Language: "typescript", RootPath: "/repo/scenarios/demo/ui"},
			{ID: "api", Language: "go", RootPath: "/repo/scenarios/demo/api"},
		},
	})

	var commands []string
	for _, cmd := range got {
		commands = append(commands, cmd.Name+" "+cmd.Args[0])
	}
	require.Contains(t, commands, "pnpm run")
	require.Contains(t, commands, "golangci-lint run")
	require.Contains(t, commands, "make lint")
}
