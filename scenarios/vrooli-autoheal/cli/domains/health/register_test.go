package health

import (
	"testing"
	"vrooli-autoheal/cli/internal/support"

	"github.com/vrooli/cli-core/cliapp"
)

func TestRegisterProvidesOperationsCommands(t *testing.T) {
	group := Register(nil, support.Dependencies{})
	want := []string{"status", "tick", "loop", "platform", "diagnose-port"}
	if group.Title != "Operations" {
		t.Fatalf("Register().Title = %q, want Operations", group.Title)
	}
	if len(group.Commands) != len(want) {
		t.Fatalf("Register() command count = %d, want %d", len(group.Commands), len(want))
	}
	for i, name := range want {
		if group.Commands[i].Name != name {
			t.Fatalf("Register() command[%d] = %q, want %q", i, group.Commands[i].Name, name)
		}
	}
}

func TestRegisterStampsPassthroughEntrypoints(t *testing.T) {
	group := Register(nil, support.Dependencies{
		RunLoop:      func(args []string) error { return nil },
		DiagnosePort: func(args []string) error { return nil },
	})
	for _, name := range []string{"loop", "diagnose-port"} {
		cmd := findCommand(t, group, name)
		if cmd.PrimitiveEvidence() != cliapp.PrimitivePassthrough {
			t.Fatalf("%s primitive evidence = %q, want passthrough", name, cmd.PrimitiveEvidence())
		}
		if cmd.Architecture.Exception != cliapp.ExceptionPassthrough {
			t.Fatalf("%s declared exception = %q, want passthrough", name, cmd.Architecture.Exception)
		}
		if cmd.Run == nil {
			t.Fatalf("%s should remain an argv handler", name)
		}
	}
}

func findCommand(t *testing.T, group cliapp.CommandGroup, name string) cliapp.Command {
	t.Helper()
	for _, cmd := range group.Commands {
		if cmd.Name == name {
			return cmd
		}
	}
	t.Fatalf("command %q not registered", name)
	return cliapp.Command{}
}
