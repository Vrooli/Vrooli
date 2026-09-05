package ai

import (
	"context"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

type testProvider struct {
	name   string
	output string
	err    error
	calls  int
}

func (p *testProvider) Name() string { return p.name }
func (p *testProvider) Generate(context.Context, string, string) (string, error) {
	p.calls++
	return p.output, p.err
}

type testEmitter struct {
	event   string
	details map[string]string
}

func (e *testEmitter) Emit(kind, _ string, details map[string]string) {
	e.event, e.details = kind, details
}

func TestChainProviderSelectionAndEmptyCases(t *testing.T) {
	fail := &testProvider{name: "first", err: errors.New("down")}
	ok := &testProvider{name: "second", output: "ok"}
	chain := NewChain(fail, ok)
	if got, provider, err := chain.Generate(context.Background(), "sys", "user"); err != nil || got != "ok" || provider != "second" || fail.calls != 1 || ok.calls != 1 {
		t.Fatalf("chain: %q %q %v", got, provider, err)
	}
	if _, _, err := NewChain(fail).Generate(context.Background(), "", ""); err == nil || !strings.Contains(err.Error(), "all providers failed") {
		t.Fatalf("failure: %v", err)
	}
	if _, _, err := NewChain().Generate(context.Background(), "", ""); err == nil || !strings.Contains(err.Error(), "no providers") {
		t.Fatalf("empty: %v", err)
	}
	if len(chain.Providers()) != 2 {
		t.Fatal("providers not exposed")
	}
}

func TestOllamaProviderRunnerAndServicePolicy(t *testing.T) {
	var args []string
	ollama := &OllamaProvider{Role: " ", Runner: func(_ context.Context, got []string, stdin string) ([]byte, error) {
		args = got
		if !strings.Contains(stdin, "System:") {
			t.Errorf("stdin=%q", stdin)
		}
		return []byte(`{"response":"ls -la"}`), nil
	}}
	if got, err := ollama.Generate(context.Background(), "sys", "user"); err != nil || got != "ls -la" || args[2] != "--role" || args[3] != "chat.default" {
		t.Fatalf("ollama: %q %v args=%v", got, err, args)
	}
	badJSON := &OllamaProvider{Runner: func(context.Context, []string, string) ([]byte, error) { return []byte("bad"), nil }}
	if _, err := badJSON.Generate(context.Background(), "", ""); err == nil {
		t.Fatal("invalid JSON should fail")
	}
	store := NewMemConfigStore()
	store.UpdateConfig(context.Background(), "openrouter", false, 2, 30, 0)
	provider := &testProvider{name: "ollama", output: "command"}
	var generated, suggested atomic.Int64
	emitter := &testEmitter{}
	svc := NewService(NewChain(provider), store, &SystemContext{OS: "linux", Arch: "amd64"}, emitter, &generated, &suggested)
	if got, name, err := svc.Execute(context.Background(), "s", "u"); err != nil || got != "command" || name != "ollama" {
		t.Fatalf("execute: %q %q %v", got, name, err)
	}
	if got, name, err := svc.ExecuteCommand(context.Background(), "u"); err != nil || got != "command" || name != "ollama" {
		t.Fatalf("command: %q %q %v", got, name, err)
	}
	if got, name, err := svc.ExecuteSuggest(context.Background(), "u"); err != nil || got != "command" || name != "ollama" {
		t.Fatalf("suggest: %q %q %v", got, name, err)
	}
	svc.EmitGenerate("ollama", "p")
	svc.EmitSuggest("ollama", "p", 2)
	svc.IncrGenerations()
	svc.IncrSuggestions()
	if generated.Load() != 1 || suggested.Load() != 1 || emitter.event != eventAISuggest || emitter.details["count"] != "2" {
		t.Fatalf("events/counters: %+v %d %d", emitter, generated.Load(), suggested.Load())
	}
	if len(svc.GetConfigs(context.Background())) == 0 || len(svc.GetHealth(context.Background())) == 0 || !svc.UpdateProviderConfig(context.Background(), "ollama", true, 1, 20, 1) {
		t.Fatal("service config methods")
	}
}

func TestServiceSkipsDisabledAndRecordsFailures(t *testing.T) {
	store := NewMemConfigStore()
	store.UpdateConfig(context.Background(), "ollama", false, 1, 1, 0)
	fail := &testProvider{name: "openrouter", err: errors.New("failed")}
	svc := NewService(NewChain(fail), store, nil, nil, nil, nil)
	if _, _, err := svc.Execute(context.Background(), "", ""); err == nil {
		t.Fatal("expected provider failure")
	}
	health := store.GetHealth(context.Background())
	var found bool
	for _, h := range health {
		if h.Name == "openrouter" {
			found = true
			if h.ErrorCount != 1 || h.Available {
				t.Fatalf("health=%+v", h)
			}
		}
	}
	if !found {
		t.Fatal("missing health")
	}
	if store.GetProviderTimeout(context.Background(), "unknown") != DefaultProviderTimeout || store.IsEnabled(context.Background(), "unknown") {
		t.Fatal("unknown config semantics")
	}
	store.RecordSuccess(context.Background(), "unknown", time.Millisecond)
}
