package domains

import (
	"scenario-to-desktop/cli/internal/support"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisteredCommandSurface(t *testing.T) {
	app, err := cliapp.NewStandardScenarioApp(cliapp.StandardScenarioOptions{
		Name:        "scenario-to-desktop-test",
		Version:     "test",
		Description: "test command registration",
	})
	if err != nil {
		t.Fatalf("NewStandardScenarioApp() error: %v", err)
	}
	deps := support.Dependencies{Core: func() *cliapp.ScenarioApp { return app }}
	commandGroups := CommandGroups(deps)
	if len(commandGroups) != 4 {
		t.Fatalf("CommandGroups() returned %d groups, want 4", len(commandGroups))
	}

	subcommandGroups := SubcommandGroups(deps)
	if len(subcommandGroups) != 12 {
		t.Fatalf("SubcommandGroups() returned %d groups, want 12", len(subcommandGroups))
	}
	for _, group := range subcommandGroups {
		if len(group.Subcommands) == 0 {
			t.Errorf("group %q has no executable subcommands", group.Name)
		}
		for _, command := range group.Subcommands {
			if command.Run == nil && command.RunCtx == nil {
				t.Errorf("command %s %s has no executable handler", group.Name, command.Name)
			}
		}
	}
}
