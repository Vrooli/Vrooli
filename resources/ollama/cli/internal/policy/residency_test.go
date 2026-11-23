package policy

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func residencyPolicy() Policy {
	return Policy{
		Roles: map[string]Role{
			"embedding.default": {Model: "nomic-embed-text"},
			"chat.small":        {Model: "qwen3.5:4b"},
			"summarize.default": {Model: "qwen3.5:4b"},
			"chat.quality":      {Model: "qwen3.5:9b"},
		},
		Residency: Residency{ResidentRoles: []string{"embedding.default", "chat.small"}, IdleUnloadSeconds: 600},
	}
}

func TestKeepAliveKeepsResidentPrimariesAndUnloadsOthers(t *testing.T) {
	p := residencyPolicy()
	cases := map[string]int{
		"qwen3.5:4b":                            KeepAliveForever, // chat.small primary, shared by summarize
		"nomic-embed-text:latest":               KeepAliveForever, // implicit :latest matches the role ref
		"nomic-embed-text":                      KeepAliveForever,
		"Qwen3.5:4B":                            KeepAliveForever, // Ollama matches names case-insensitively
		"registry.ollama.ai/library/qwen3.5:4b": KeepAliveForever,
		"qwen3.5:9b":                            600,
		"llava:7b":                              600, // a model no role names still unloads
	}
	for model, want := range cases {
		got, ok := p.KeepAliveSeconds(model)
		if !ok || got != want {
			t.Errorf("KeepAliveSeconds(%q) = %d, %v; want %d, true", model, got, ok, want)
		}
	}
	if _, ok := p.KeepAliveSeconds(""); ok {
		t.Error("an empty model must not get a keep_alive")
	}
}

func TestKeepAliveIsUnsetWithoutIdleUnload(t *testing.T) {
	p := residencyPolicy()
	p.Residency.IdleUnloadSeconds = 0
	if _, ok := p.KeepAliveSeconds("qwen3.5:9b"); ok {
		t.Fatal("a policy without idle_unload_seconds must leave keep_alive to the daemon")
	}
}

func TestResidencyValidation(t *testing.T) {
	p := residencyPolicy()
	p.Residency = Residency{ResidentRoles: []string{"chat.small", "missing.role", "chat.small"}, IdleUnloadSeconds: 30}
	joined := ""
	for _, err := range p.Residency.validate(p) {
		joined += err.Error() + "\n"
	}
	for _, want := range []string{"below 60", `unknown role "missing.role"`, `lists "chat.small" twice`} {
		if !strings.Contains(joined, want) {
			t.Errorf("validation errors missing %q:\n%s", want, joined)
		}
	}
}

// The shipped policy keeps the shared small models loaded and lets the opt-in
// quality chat model unload, so an idle 9B never holds GPU memory between uses.
func TestShippedResidencyUnloadsLargeChatModel(t *testing.T) {
	p, err := LoadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	for _, role := range []string{"embedding.default", "chat.small"} {
		if got, _ := p.KeepAliveSeconds(p.Roles[role].Model); got != KeepAliveForever {
			t.Errorf("%s primary %s keep_alive = %d, want resident", role, p.Roles[role].Model, got)
		}
	}
	chat := p.Roles["chat.quality"].Model
	if got, ok := p.KeepAliveSeconds(chat); !ok || got <= 0 {
		t.Errorf("chat.quality primary %s keep_alive = %d, %v; want a positive idle unload", chat, got, ok)
	}
}

func TestLiveReloadsChangedPolicyAndKeepsLastGoodOnError(t *testing.T) {
	shipped, err := os.ReadFile(filepath.Join("..", "..", "..", "model-policy.json"))
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "model-policy.json")
	if err := os.WriteFile(path, shipped, 0o644); err != nil {
		t.Fatal(err)
	}
	clock := time.Date(2026, 11, 21, 12, 0, 0, 0, time.UTC)
	live := NewLive(path, nil)
	live.Now = func() time.Time { return clock }
	if got, ok := live.KeepAliveSeconds("qwen3.5:9b"); !ok || got != 600 {
		t.Fatalf("initial keep_alive = %d, %v; want 600", got, ok)
	}

	edited := strings.Replace(string(shipped), `"idle_unload_seconds": 600`, `"idle_unload_seconds": 1200`, 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	if got, _ := live.KeepAliveSeconds("qwen3.5:9b"); got != 600 {
		t.Fatalf("reload happened inside the check interval: %d", got)
	}
	clock = clock.Add(liveCheckInterval)
	if got, _ := live.KeepAliveSeconds("qwen3.5:9b"); got != 1200 {
		t.Fatalf("edited keep_alive = %d, want 1200", got)
	}

	if err := os.WriteFile(path, []byte(`{"roles":`), 0o644); err != nil {
		t.Fatal(err)
	}
	clock = clock.Add(liveCheckInterval)
	if got, ok := live.KeepAliveSeconds("qwen3.5:9b"); !ok || got != 1200 {
		t.Fatalf("broken edit must keep the last good policy, got %d, %v", got, ok)
	}
}

func TestAdmissionPoolsGiveEachResidentItsOwnRunnerAndShareTurnover(t *testing.T) {
	p := residencyPolicy()
	cases := []struct {
		model, pool string
		runners     int
	}{
		{"nomic-embed-text", "model-nomic-embed-text_latest", 1},
		{"Qwen3.5:4B", "model-qwen3.5_4b", 1},
		{"qwen3.5:9b", "turnover", 1},
		{"gemma4:12b", "turnover", 1},
	}
	for _, tc := range cases {
		if pool, runners := p.AdmissionPool(tc.model); pool != tc.pool || runners != tc.runners {
			t.Errorf("AdmissionPool(%q) = %q, %d; want %q, %d", tc.model, pool, runners, tc.pool, tc.runners)
		}
	}
}
