package admin

import (
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

func TestStripeSettingsCommandsUseVerifiedConnectPrimitives(t *testing.T) {
	commands := make(map[string]cliapp.Command)
	for _, command := range Register(support.Dependencies{}).Commands {
		commands[command.Name] = command
	}

	tests := []struct {
		name      string
		primitive cliapp.PrimitiveClass
		flag      string
		required  bool
	}{
		{name: "admin-stripe-settings", primitive: cliapp.PrimitiveProtoList},
		{name: "admin-stripe-settings-update", primitive: cliapp.PrimitiveProtoMutation, flag: "body", required: true},
		{name: "admin-stripe-secret", primitive: cliapp.PrimitiveOperational, flag: "field"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			command, ok := commands[tt.name]
			if !ok {
				t.Fatalf("command %q is not registered", tt.name)
			}
			if !command.NeedsAPI {
				t.Fatal("Connect command must require the API")
			}
			if command.Architecture.Primitive != tt.primitive {
				t.Fatalf("declared primitive = %q, want %q", command.Architecture.Primitive, tt.primitive)
			}
			if command.PrimitiveEvidence() != tt.primitive {
				t.Fatalf("observed primitive = %q, want %q", command.PrimitiveEvidence(), tt.primitive)
			}
			if tt.flag == "" {
				return
			}
			for _, flag := range command.Args.Flags {
				if flag.Name == tt.flag {
					if flag.Required != tt.required {
						t.Fatalf("--%s required = %t, want %t", tt.flag, flag.Required, tt.required)
					}
					return
				}
			}
			t.Fatalf("command is missing --%s", tt.flag)
		})
	}
}
