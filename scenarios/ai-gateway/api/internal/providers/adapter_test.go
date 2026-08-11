package providers

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	sharedv1 "github.com/vrooli/vrooli/packages/proto/gen/go/ai-gateway/v1/shared"
)

func TestAdapter_ListRoles_ParsesResourcePolicyOutput(t *testing.T) { // [REQ:AIGW-PROVIDER-CLI-ONLY] [REQ:AIGW-INVENTORY-ROLES]
	runner := &fakeRunner{results: map[string]Result{
		"resource-ollama policy roles --json": {Stdout: `{
			"roles": [
				{
					"schema_version": "2026-06-10",
					"role": "embedding.default",
					"required_capabilities": ["embedding"],
					"capabilities": ["embedding"]
				},
				{
					"schema_version": "2026-06-10",
					"role": "chat.default",
					"required_capabilities": ["chat", "generate"],
					"capabilities": ["chat", "generate", "summarize"]
				}
			]
		}`},
	}}

	inventory, err := (Adapter{
		Provider:    ProviderOllama,
		CommandName: "resource-ollama",
		Locality:    "local",
		Runner:      runner,
	}).ListRoles(context.Background())
	if err != nil {
		t.Fatalf("ListRoles() error = %v", err)
	}
	if got, want := len(inventory.Roles), 2; got != want {
		t.Fatalf("roles len = %d, want %d", got, want)
	}
	if inventory.Roles[0].Role != "chat.default" {
		t.Fatalf("roles sorted by role, first = %q", inventory.Roles[0].Role)
	}
	role := inventory.Roles[1]
	if role.Provider != ProviderOllama || role.Locality != "local" || role.Status != "available" {
		t.Fatalf("role metadata = %+v", role)
	}
	if strings.Join(role.Capabilities, ",") != "embedding" {
		t.Fatalf("capabilities = %v", role.Capabilities)
	}
	if role.PolicySchemaVersion != "2026-06-10" {
		t.Fatalf("policy schema = %q", role.PolicySchemaVersion)
	}
	if len(runner.Commands) != 1 || runner.Commands[0].String() != "resource-ollama policy roles --json" {
		t.Fatalf("commands = %+v", runner.Commands)
	}
}

func TestAdapter_ListRoles_MapsMalformedJSON(t *testing.T) { // [REQ:AIGW-PROVIDER-FAILURES]
	runner := &fakeRunner{results: map[string]Result{
		"resource-openrouter policy roles --json": {Stdout: `not-json`},
	}}

	_, err := (Adapter{
		Provider:    ProviderOpenRouter,
		CommandName: "resource-openrouter",
		Locality:    "remote",
		Runner:      runner,
	}).ListRoles(context.Background())
	var cmdErr *CommandError
	if !errors.As(err, &cmdErr) {
		t.Fatalf("error = %T %v, want CommandError", err, err)
	}
	if cmdErr.Code != "malformed_json" {
		t.Fatalf("code = %q", cmdErr.Code)
	}
}

func TestAdapter_Smoke_MapsMissingBinary(t *testing.T) { // [REQ:AIGW-INVENTORY-SMOKE] [REQ:AIGW-PROVIDER-FAILURES]
	runner := &fakeRunner{errors: map[string]error{
		"resource-openrouter policy roles --json": &CommandError{Code: "missing_binary", Command: "resource-openrouter policy roles --json", ExitCode: -1},
	}}

	smoke := (Adapter{
		Provider:    ProviderOpenRouter,
		CommandName: "resource-openrouter",
		Locality:    "remote",
		Runner:      runner,
	}).Smoke(context.Background())
	if smoke.Status != "unavailable" || smoke.Code != "missing_binary" {
		t.Fatalf("smoke = %+v", smoke)
	}
}

func TestAdapterMultimodalUsesJSONStdinAndNeverArgv(t *testing.T) { // [REQ:AIGW-MULTIMODAL-CONTRACT]
	runner := &fakeRunner{results: map[string]Result{
		"resource-ollama gateway generate --role vision.default --json --input-json-stdin": {Stdout: `{"response":"ok"}`},
	}}
	image := []byte{1, 2, 3}
	command, err := (Adapter{Provider: ProviderOllama, CommandName: "resource-ollama", Locality: "local", Runner: runner}).executionCommand(ExecutionRequest{
		Kind:        sharedv1.RequestKind_REQUEST_KIND_TEXT_GENERATION,
		Role:        "vision.default",
		InputText:   "describe",
		Timeout:     time.Second,
		Attachments: []*sharedv1.Attachment{{MediaType: "image/png", Payload: &sharedv1.Attachment_InlineBytes{InlineBytes: image}}},
	})
	if err != nil {
		t.Fatalf("executionCommand: %v", err)
	}
	if command.String() != "resource-ollama gateway generate --role vision.default --json --input-json-stdin" {
		t.Fatalf("command = %q", command.String())
	}
	var envelope struct {
		Prompt string `json:"prompt"`
		Images []struct {
			DataB64 string `json:"data_b64"`
		} `json:"images"`
	}
	if err := json.Unmarshal([]byte(command.Stdin), &envelope); err != nil {
		t.Fatalf("decode stdin: %v", err)
	}
	if envelope.Prompt != "describe" || len(envelope.Images) != 1 || envelope.Images[0].DataB64 != base64.StdEncoding.EncodeToString(image) {
		t.Fatalf("envelope = %+v", envelope)
	}
}

func TestRedactRemovesSecretishOutput(t *testing.T) { // [REQ:AIGW-PROVIDER-FAILURES]
	got := redact(`Authorization: Bearer abc123 token="secret-value" safe`)
	if strings.Contains(got, "abc123") || strings.Contains(got, "secret-value") {
		t.Fatalf("redact leaked secretish values: %q", got)
	}
}

type fakeRunner struct {
	results  map[string]Result
	errors   map[string]error
	Commands []Command
}

var _ CommandRunner = (*fakeRunner)(nil)

func (f *fakeRunner) Run(_ context.Context, command Command) (Result, error) {
	f.Commands = append(f.Commands, command)
	key := command.String()
	if err, ok := f.errors[key]; ok {
		return f.results[key], err
	}
	if result, ok := f.results[key]; ok {
		return result, nil
	}
	return Result{}, &CommandError{Code: "missing_fixture", Command: key, ExitCode: -1}
}
