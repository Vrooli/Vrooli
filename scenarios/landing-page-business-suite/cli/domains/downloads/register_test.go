package downloads

import (
	"strings"
	"testing"

	"github.com/vrooli/cli-core/cliapp"
	testutil "github.com/vrooli/cli-core/cliapptest"
	"landing-page-business-suite/cli/internal/support"
)

func TestRegisterExposesValidCommandGroup(t *testing.T) {
	group := Register(support.Dependencies{})
	if err := testutil.ValidateCommandGroup(group); err != nil {
		t.Fatalf("ValidateCommandGroup() error = %v", err)
	}
}

func TestDownloadAppCommandsUseGeneratedConnectPrimitives(t *testing.T) {
	commands := make(map[string]cliapp.Command)
	for _, command := range Register(support.Dependencies{}).Commands {
		commands[command.Name] = command
	}

	tests := []struct {
		name      string
		primitive cliapp.PrimitiveClass
		body      bool
		appKey    bool
	}{
		{name: "admin-download-apps-list", primitive: cliapp.PrimitiveProtoList},
		{name: "admin-download-apps-create", primitive: cliapp.PrimitiveProtoMutation, body: true},
		{name: "admin-download-apps-save", primitive: cliapp.PrimitiveProtoMutation, body: true, appKey: true},
		{name: "admin-download-apps-delete", primitive: cliapp.PrimitiveProtoMutation, appKey: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, ok := commands[tt.name]
			if !ok {
				t.Fatalf("command %q is not registered", tt.name)
			}
			if !command.NeedsAPI || command.Architecture.Primitive != tt.primitive || command.PrimitiveEvidence() != tt.primitive {
				t.Fatalf("command transport primitive = %q/%q, want %q", command.Architecture.Primitive, command.PrimitiveEvidence(), tt.primitive)
			}
			if tt.body {
				if len(command.Args.Flags) != 1 || command.Args.Flags[0].Name != "body" || !command.Args.Flags[0].Required {
					t.Fatalf("command body contract = %#v", command.Args.Flags)
				}
			}
			if tt.appKey {
				if len(command.Args.Positionals) != 1 || command.Args.Positionals[0].Name != "app_key" || !command.Args.Positionals[0].Required {
					t.Fatalf("command app_key contract = %#v", command.Args.Positionals)
				}
			}
		})
	}
}

func TestRunUploadManagedRejectsIncompleteInvocationBeforeNetwork(t *testing.T) {
	err := runUploadManaged(support.Dependencies{}, []string{"--file", "artifact.zip"})
	if err == nil || !strings.Contains(err.Error(), "platform is required") {
		t.Fatalf("error = %v, want platform validation error", err)
	}
}

func TestRunUploadManagedRejectsInvalidPlatformBeforeNetwork(t *testing.T) {
	err := runUploadManaged(support.Dependencies{}, []string{
		"--file", "artifact.zip", "--app-key", "desktop", "--platform", "ios", "--release-version", "1.2.3",
	})
	if err == nil || !strings.Contains(err.Error(), "platform") {
		t.Fatalf("error = %v, want platform validation error", err)
	}
}

func TestRunUploadManagedRejectsMissingArtifactBeforeNetwork(t *testing.T) {
	err := runUploadManaged(support.Dependencies{}, []string{
		"--file", "does-not-exist.zip", "--app-key", "desktop", "--platform", "windows", "--release-version", "1.2.3",
	})
	if err == nil || !strings.Contains(err.Error(), "artifact file not found") {
		t.Fatalf("error = %v, want missing artifact error", err)
	}
}
