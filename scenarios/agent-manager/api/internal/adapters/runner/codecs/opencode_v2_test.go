package codecs

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"

	"github.com/google/uuid"
	opencodeversion "github.com/vrooli/vrooli/packages/opencode-version"
)

func TestOpenCodeV2_BuildArgs(t *testing.T) {
	c := NewOpenCodeForTestWithCapabilities(opencodeversion.FromMajor(2))
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		RunID:      uuid.New(),
		Prompt:     "do the thing",
		WorkingDir: "/work",
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeOpenCode,
			Model:      "openai/gpt-5",
			Effort:     domain.EffortHigh,
		},
	})
	want := []string{
		"run", "do the thing", "--format", "json", "--print-logs", "--standalone",
		"-m", "openai/gpt-5#high",
	}
	if len(args) != len(want) {
		t.Fatalf("args = %q, want %q", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q (args=%q)", i, args[i], want[i], args)
		}
	}
	if _, ok := findDirArg(args); ok {
		t.Fatal("OpenCode 2 does not accept --dir; the runner pins the process cwd")
	}
}

func TestOpenCodeV2_BuildContinueArgs(t *testing.T) {
	c := NewOpenCodeForTestWithCapabilities(opencodeversion.FromMajor(2))
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		RunID:      uuid.New(),
		SessionID:  "ses_abc",
		Prompt:     "follow up",
		WorkingDir: "/work",
		ResolvedConfig: &domain.RunConfig{
			RunnerType: domain.RunnerTypeOpenCode,
			Model:      "openai/gpt-5",
			ExtraFlags: domain.RunnerExtraFlags{domain.RunnerTypeOpenCode: []string{"--verbose"}},
		},
	})
	want := []string{
		"run", "follow up", "--session", "ses_abc", "--format", "json", "--print-logs",
		"--standalone", "-m", "openai/gpt-5", "--verbose",
	}
	if len(args) != len(want) {
		t.Fatalf("args = %q, want %q", args, want)
	}
	for i := range want {
		if args[i] != want[i] {
			t.Fatalf("args[%d] = %q, want %q (args=%q)", i, args[i], want[i], args)
		}
	}
}

func TestOpenCodeV2_ControlArgs(t *testing.T) {
	caps := opencodeversion.FromMajor(2)
	t.Run("model and effort", func(t *testing.T) {
		args, err := opencodeControlArgsFor(caps, &domain.RunConfig{Model: "google/gemini-4", Effort: domain.EffortLow})
		if err != nil {
			t.Fatal(err)
		}
		if len(args) != 2 || args[0] != "-m" || args[1] != "google/gemini-4#low" {
			t.Fatalf("args = %q", args)
		}
	})
	t.Run("effort without model fails closed", func(t *testing.T) {
		if _, err := opencodeControlArgsFor(caps, &domain.RunConfig{Effort: domain.EffortHigh}); err == nil {
			t.Fatal("expected an error when a variant has no model to attach to")
		}
	})
	t.Run("model only", func(t *testing.T) {
		args, err := opencodeControlArgsFor(caps, &domain.RunConfig{Model: "ollama/qwen3.5:4b"})
		if err != nil {
			t.Fatal(err)
		}
		if len(args) != 2 || args[0] != "-m" || args[1] != "ollama/qwen3.5:4b" {
			t.Fatalf("args = %q", args)
		}
	})
}

func TestOpenCodeV2_ValidateContinuationUsesV2Table(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "opencode"), 0o755); err != nil {
		t.Fatal(err)
	}
	dbPath := filepath.Join(dir, "opencode", "opencode.db")
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`CREATE TABLE session_v2 (id TEXT PRIMARY KEY)`); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO session_v2 (id) VALUES ('ses_present')`); err != nil {
		t.Fatal(err)
	}
	_ = db.Close()

	c := NewOpenCodeForTestWithCapabilities(opencodeversion.FromMajor(2))
	env := map[string]string{"XDG_DATA_HOME": dir}
	if err := c.ValidateContinuation(context.Background(), runner.ContinueRequest{SessionID: "ses_present", Environment: env}); err != nil {
		t.Fatalf("present V2 session should validate: %v", err)
	}
	if err := c.ValidateContinuation(context.Background(), runner.ContinueRequest{SessionID: "ses_missing", Environment: env}); err == nil {
		t.Fatal("missing V2 session should be reported as expired")
	}
}

func TestOpenCodeV2_ProbeModelSkipsLocalCatalog(t *testing.T) {
	c := NewOpenCodeForTestWithBinary("/bin/true")
	c.caps = opencodeversion.FromMajor(2)
	if err := c.ProbeModel(context.Background(), "openrouter/missing/model"); err != nil {
		t.Fatalf("V2 has no stable local catalog; probe must degrade to nil, got %v", err)
	}
}

// TestOpenCodeV2_DecodeCapturedStream uses a real 2.3.2 `run --format json`
// capture (local Ollama) so the decoder contract is anchored to the shipped
// wire shape, not a hand-written approximation.
func TestOpenCodeV2_DecodeCapturedStream(t *testing.T) {
	lines := []string{
		`{"type":"step_start","timestamp":1794972661757,"sessionID":"ses_e134acda9ffedXySGAVjW8rV6O","part":{"id":"prt_1ecb537fd0010Nmf1sZ7Gg8mnW","sessionID":"ses_e134acda9ffedXySGAVjW8rV6O","messageID":"msg_1ecb537f7001OOfM4eLeMey34V","type":"step-start"}}`,
		`{"type":"text","timestamp":1794972677635,"sessionID":"ses_e134acda9ffedXySGAVjW8rV6O","part":{"id":"prt_1ecb575f1001oN3szn6fbeik5y","sessionID":"ses_e134acda9ffedXySGAVjW8rV6O","messageID":"msg_1ecb537f7001OOfM4eLeMey34V","type":"text","text":"{\"ok\"}","time":{"start":1794972677617,"end":1794972677635}}}`,
		`{"type":"step_finish","timestamp":1794972677669,"sessionID":"ses_e134acda9ffedXySGAVjW8rV6O","part":{"id":"prt_1ecb576250017zcc8pTzflwt8b","sessionID":"ses_e134acda9ffedXySGAVjW8rV6O","messageID":"msg_1ecb537f7001OOfM4eLeMey34V","type":"step-finish","reason":"stop","cost":0,"tokens":{"input":12,"output":4,"reasoning":0,"cache":{"read":3,"write":0}}}}`,
	}
	c := NewOpenCodeForTestWithCapabilities(opencodeversion.FromMajor(2))
	events := decodeLines(t, c, uuid.New(), lines)
	messages := eventsOfType[*domain.MessageEventData](events)
	if len(messages) != 1 || !strings.Contains(messages[0].Content, "ok") {
		t.Fatalf("messages = %#v", messages)
	}
	usages := eventsOfType[*domain.UsageEventData](events)
	if len(usages) != 1 || usages[0].InputTokens != 12 || usages[0].OutputTokens != 4 || usages[0].CacheReadTokens != 3 {
		t.Fatalf("usages = %#v", usages)
	}
}
