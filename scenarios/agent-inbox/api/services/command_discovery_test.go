package services

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-inbox/domain"
	"agent-inbox/integrations"
)

type fakeCommandDiscovery struct {
	result CommandDiscoveryResult
	err    error
	reqs   []CommandDiscoveryRequest
}

func (f *fakeCommandDiscovery) DiscoverCommands(_ context.Context, req CommandDiscoveryRequest) (CommandDiscoveryResult, error) {
	f.reqs = append(f.reqs, req)
	return f.result, f.err
}

func TestMaybeInjectCommandContext_AddsSearchHubCommandSystemMessage(t *testing.T) {
	discovery := &fakeCommandDiscovery{result: CommandDiscoveryResult{
		Commands: []DiscoveredCommand{{
			Title:      "agent-manager run",
			Path:       "agent-manager run",
			Snippet:    "agent-manager run - Manage run executions",
			ProviderID: "cli-health.commands",
		}},
	}}
	svc := &CompletionService{commandDiscovery: discovery}

	messages := []domain.Message{{Role: "user", Content: "spawn a coding agent"}}
	orMessages := []integrations.OpenRouterMessage{{Role: "user", Content: "spawn a coding agent"}}

	got, diagnostic := svc.maybeInjectCommandContext(context.Background(), "chat-1", messages, orMessages)
	if diagnostic != "" {
		t.Fatalf("diagnostic = %q, want empty", diagnostic)
	}
	if len(discovery.reqs) != 1 {
		t.Fatalf("DiscoverCommands calls = %d, want 1", len(discovery.reqs))
	}
	if discovery.reqs[0].Query != "spawn a coding agent" {
		t.Fatalf("query = %q, want latest user message", discovery.reqs[0].Query)
	}
	if len(got) != 2 {
		t.Fatalf("messages = %d, want 2", len(got))
	}
	if got[0].Role != "system" {
		t.Fatalf("first role = %q, want system", got[0].Role)
	}
	content, ok := got[0].Content.(string)
	if !ok {
		t.Fatalf("system content type = %T, want string", got[0].Content)
	}
	for _, want := range []string{"search-hub/cli-health", "agent-manager run", "OpenRouter function tools"} {
		if !strings.Contains(content, want) {
			t.Fatalf("system content missing %q: %s", want, content)
		}
	}
}

func TestMaybeInjectCommandContext_EmptyDiscoveryKeepsChatUsable(t *testing.T) {
	discovery := &fakeCommandDiscovery{result: CommandDiscoveryResult{
		Diagnostic: "command discovery returned no command records",
	}}
	svc := &CompletionService{commandDiscovery: discovery}
	orMessages := []integrations.OpenRouterMessage{{Role: "user", Content: "hello"}}

	got, diagnostic := svc.maybeInjectCommandContext(context.Background(), "chat-1", []domain.Message{{Role: "user", Content: "hello"}}, orMessages)
	if diagnostic == "" {
		t.Fatal("diagnostic empty, want degraded diagnostic")
	}
	if len(got) != len(orMessages) || got[0].Role != "user" {
		t.Fatalf("messages changed on empty discovery: %#v", got)
	}
}

func TestMaybeInjectCommandContext_SearchHubFailureKeepsChatUsable(t *testing.T) {
	discovery := &fakeCommandDiscovery{err: errors.New("search-hub unavailable")}
	svc := &CompletionService{commandDiscovery: discovery}
	orMessages := []integrations.OpenRouterMessage{{Role: "user", Content: "hello"}}

	got, diagnostic := svc.maybeInjectCommandContext(context.Background(), "chat-1", []domain.Message{{Role: "user", Content: "hello"}}, orMessages)
	if !strings.Contains(diagnostic, "search-hub unavailable") {
		t.Fatalf("diagnostic = %q, want search-hub failure", diagnostic)
	}
	if len(got) != len(orMessages) || got[0].Role != "user" {
		t.Fatalf("messages changed on discovery failure: %#v", got)
	}
}

func TestSearchHubCommandDiscoveryParsesCommandResults(t *testing.T) {
	dir := t.TempDir()
	binary := filepath.Join(dir, "search-hub-fixture")
	script := `#!/usr/bin/env bash
printf '%s\n' '{"ranked":[{"provider_id":"cli-health.commands","provider_group":"cli-health","type":"command","title":"test-genie runs","snippet":"test-genie runs - Inspect recorded runs","path":"test-genie runs","score":0.2,"rerank_score":0.9}]}'
`
	if err := os.WriteFile(binary, []byte(script), 0o755); err != nil {
		t.Fatalf("write fixture binary: %v", err)
	}

	discovery := &SearchHubCommandDiscovery{Binary: binary}
	result, err := discovery.DiscoverCommands(context.Background(), CommandDiscoveryRequest{Query: "wait for scenario tests", Limit: 1})
	if err != nil {
		t.Fatalf("DiscoverCommands returned error: %v", err)
	}
	if len(result.Commands) != 1 {
		t.Fatalf("commands = %d, want 1", len(result.Commands))
	}
	if result.Commands[0].Path != "test-genie runs" {
		t.Fatalf("path = %q, want test-genie runs", result.Commands[0].Path)
	}
	if result.Commands[0].ProviderID != "cli-health.commands" {
		t.Fatalf("provider = %q, want cli-health.commands", result.Commands[0].ProviderID)
	}
}

func TestBuildCommandDiscoveryQueryUsesLatestUserMessage(t *testing.T) {
	got := buildCommandDiscoveryQuery([]domain.Message{
		{Role: "user", Content: "older"},
		{Role: "assistant", Content: "assistant"},
		{Role: "user", Content: "  latest  "},
	})
	if got != "latest" {
		t.Fatalf("query = %q, want latest", got)
	}
}
