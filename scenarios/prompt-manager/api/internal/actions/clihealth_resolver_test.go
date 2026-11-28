package actions

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"prompt-manager/internal/store"

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

// TestCoreActionPacksRemainOwnedByTheRecordedCLIHealthCatalog makes catalog
// drift explicit. The fixture is a recorded response set from CLI Health for
// every core action, rather than a hand-maintained approximation of command
// paths in prompt-manager. An action may receive a partial verdict when CLI
// Health has no argument metadata, but it must never lose command ownership.
func TestCoreActionPacksRemainOwnedByTheRecordedCLIHealthCatalog(t *testing.T) {
	repoRoot := filepath.Dir(filepath.Dir(filepath.Dir(locateRealSchema(t))))
	fixturePath := filepath.Join(repoRoot, "scenarios", "prompt-manager", "api", "internal", "actions", "testdata", "core_action_cli_health_catalog.json")
	rawFixture, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatalf("read catalog fixture: %v", err)
	}
	var fixture struct {
		Records []struct {
			ID     string                    `json:"id"`
			Result cliHealthValidationResult `json:"result"`
		} `json:"records"`
	}
	if err := json.Unmarshal(rawFixture, &fixture); err != nil {
		t.Fatalf("parse catalog fixture: %v", err)
	}
	catalog := make(map[string]cliHealthValidationResult, len(fixture.Records))
	for _, record := range fixture.Records {
		if record.ID == "" || record.Result.CommandText == "" {
			t.Fatalf("invalid catalog record: %#v", record)
		}
		if _, exists := catalog[record.Result.CommandText]; exists {
			t.Fatalf("duplicate command text in catalog fixture: %s", record.Result.CommandText)
		}
		catalog[record.Result.CommandText] = record.Result
	}

	requested := map[string]bool{}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request struct {
			CommandText string `json:"commandText"`
		}
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatalf("decode CLI Health request: %v", err)
		}
		result, ok := catalog[request.CommandText]
		if !ok {
			t.Fatalf("action command is absent from recorded CLI Health catalog: %s", request.CommandText)
		}
		requested[request.CommandText] = true
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(map[string]any{"result": result}); err != nil {
			t.Fatalf("encode CLI Health response: %v", err)
		}
	}))
	defer server.Close()

	resolver := &CLIHealthCommandResolver{resolver: discovery.NewStaticResolver(server.URL), httpClient: server.Client()}
	service := NewService(nil, resolver)
	actionPaths, err := filepath.Glob(filepath.Join(repoRoot, "scenarios", "prompt-manager", "store", "actions", "packs", "core", "*", "action.json"))
	if err != nil {
		t.Fatalf("list core actions: %v", err)
	}
	if len(actionPaths) == 0 {
		t.Fatal("expected core action packs")
	}
	for _, actionPath := range actionPaths {
		rawAction, err := os.ReadFile(actionPath)
		if err != nil {
			t.Fatalf("read %s: %v", actionPath, err)
		}
		var action store.Action
		if err := json.Unmarshal(rawAction, &action); err != nil {
			t.Fatalf("parse %s: %v", actionPath, err)
		}
		t.Run(action.ID, func(t *testing.T) {
			validation := service.Validate(context.Background(), &action)
			if !validation.Valid || !validation.Runnable {
				t.Fatalf("core action lost CLI Health ownership: valid=%t runnable=%t checks=%#v", validation.Valid, validation.Runnable, validation.Checks)
			}
			if validation.Command == nil || validation.Command.Certainty == CertaintyNone {
				t.Fatalf("expected owned command resolution, got %#v", validation.Command)
			}
		})
	}
	if len(requested) != len(catalog) {
		t.Fatalf("catalog fixture records=%d, requests=%d; remove stale records or add the missing core action", len(catalog), len(requested))
	}
}
