package actions

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vrooli/api-core/discovery"
)

func TestCLIHealthCommandResolverRequiresCurrentCommandPath(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/vrooli.cli_health.v1.command.CommandReferenceValidationService/ValidateCommandReference" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"commandText":"prompt-manager skill read docs","verdict":"COMMAND_REFERENCE_VERDICT_PARTIAL","validationLevel":"COMMAND_REFERENCE_VALIDATION_LEVEL_COMMAND_EXISTS","canonicalCommand":"prompt-manager skill read","owner":"prompt-manager","issues":[{"code":"argument_schema_unavailable","message":"arguments unavailable"}],"guidance":["CLI Health proved the command path exists, but not every flag or positional argument."]}}`))
	}))
	defer server.Close()

	resolver := &CLIHealthCommandResolver{
		resolver:   discovery.NewStaticResolver(server.URL),
		httpClient: server.Client(),
	}
	resolution, err := resolver.ResolveCommand(context.Background(), []string{"prompt-manager", "skill", "read", "docs"})
	if err != nil {
		t.Fatalf("ResolveCommand: %v", err)
	}
	if resolution.Certainty != CertaintyOwnerOnly {
		t.Fatalf("certainty = %s, want %s; resolution=%#v", resolution.Certainty, CertaintyOwnerOnly, resolution)
	}
	if resolution.Owner.Type != "scenario" || resolution.Owner.ID != "prompt-manager" {
		t.Fatalf("owner = %#v", resolution.Owner)
	}
	if resolution.Message == "" {
		t.Fatalf("expected CLI Health guidance in message")
	}
}

func TestCLIHealthCommandResolverRejectsInvalidCommands(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"result":{"verdict":"COMMAND_REFERENCE_VERDICT_INVALID","validationLevel":"COMMAND_REFERENCE_VALIDATION_LEVEL_OWNER_IDENTIFIED","owner":"prompt-manager","issues":[{"code":"unknown_command","message":"command path was not found"}],"suggestions":[{"command":"prompt-manager skill read","reason":"closest catalog command"}]}}`))
	}))
	defer server.Close()

	resolver := &CLIHealthCommandResolver{
		resolver:   discovery.NewStaticResolver(server.URL),
		httpClient: server.Client(),
	}
	resolution, err := resolver.ResolveCommand(context.Background(), []string{"prompt-manager", "skill", "reed", "docs"})
	if err != nil {
		t.Fatalf("ResolveCommand: %v", err)
	}
	if resolution.Certainty != CertaintyNone {
		t.Fatalf("certainty = %s, want %s; resolution=%#v", resolution.Certainty, CertaintyNone, resolution)
	}
	if resolution.Message == "" {
		t.Fatalf("expected rejection detail")
	}
}
