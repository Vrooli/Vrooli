package ai

import (
	"context"
	"errors"
	"testing"
)

type adapterBackend struct {
	commandRaw, suggestRaw string
	err                    error
	configs                []ProviderConfig
	health                 []ProviderHealth
	updated                bool
	generated, suggested   int
}

func (b *adapterBackend) ExecuteCommand(context.Context, string) (string, string, error) {
	return b.commandRaw, "local", b.err
}

func (b *adapterBackend) ExecuteSuggest(context.Context, string) (string, string, error) {
	return b.suggestRaw, "local", b.err
}
func (b *adapterBackend) EmitGenerate(string, string)                 { b.generated++ }
func (b *adapterBackend) EmitSuggest(string, string, int)             { b.suggested++ }
func (b *adapterBackend) IncrGenerations()                            {}
func (b *adapterBackend) IncrSuggestions()                            {}
func (b *adapterBackend) GetConfigs(context.Context) []ProviderConfig { return b.configs }
func (b *adapterBackend) GetHealth(context.Context) []ProviderHealth  { return b.health }
func (b *adapterBackend) UpdateProviderConfig(context.Context, string, bool, int, int, int) bool {
	b.updated = true
	return true
}

func TestExtractCommandAndCommands(t *testing.T) {
	if got := ExtractCommand(" ```bash\necho hi\n``` "); got != "echo hi" {
		t.Fatalf("ExtractCommand() = %q", got)
	}
	if got := ExtractCommands("one\n\n```sh\ntwo\n```\nthree\nfour"); len(got) != 3 || got[1] != "two" {
		t.Fatalf("ExtractCommands() = %#v", got)
	}
}

func TestAdapterGenerateSuggestAndConfig(t *testing.T) {
	b := &adapterBackend{commandRaw: "```bash\nls\n```", suggestRaw: "pwd\nwhoami", configs: []ProviderConfig{{Name: "local", Enabled: true, Priority: 1, TimeoutSec: 30, MaxRetries: 1}}}
	a := &Adapter{Backend: b}
	if command, provider, err := a.Generate(context.Background(), "list", "cwd=/tmp"); err != nil || command != "ls" || provider != "local" || b.generated != 1 {
		t.Fatalf("Generate() = %q, %q, %v", command, provider, err)
	}
	commands, provider, err := a.Suggest(context.Background(), "inspect", "cwd=/tmp")
	if err != nil || provider != "local" || len(commands) != 2 || b.suggested != 1 {
		t.Fatalf("Suggest() = %#v, %q, %v", commands, provider, err)
	}
	if got := a.GetConfig(context.Background()); len(got.Providers) != 1 {
		t.Fatalf("GetConfig() = %#v", got)
	}
	if _, err := a.UpdateConfig(context.Background(), UpdateConfigRequest{Name: "missing"}); !errors.Is(err, ErrUnknownProvider) {
		t.Fatalf("missing provider error = %v", err)
	}
	if _, err := a.UpdateConfig(context.Background(), UpdateConfigRequest{Name: "local", HasTimeoutSec: true, TimeoutSec: 0}); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("timeout error = %v", err)
	}
	if _, err := a.UpdateConfig(context.Background(), UpdateConfigRequest{Name: "local", HasMaxRetries: true, MaxRetries: 6}); !errors.Is(err, ErrInvalidBody) {
		t.Fatalf("retry error = %v", err)
	}
	if _, err := a.UpdateConfig(context.Background(), UpdateConfigRequest{Name: "local", HasEnabled: true, Enabled: false}); err != nil || !b.updated {
		t.Fatalf("valid update = %v", err)
	}
	b.err = errors.New("provider down")
	if _, _, err := a.Generate(context.Background(), "x", ""); err == nil {
		t.Fatal("expected generation error")
	}
}

func TestAdapterGetHealth(t *testing.T) {
	want := []ProviderHealth{{Name: "local", Available: true}}
	got := (&Adapter{Backend: &adapterBackend{health: want}}).GetHealth(context.Background())
	if len(got) != 1 || got[0].Name != "local" || !got[0].Available {
		t.Fatalf("health=%+v", got)
	}
}
