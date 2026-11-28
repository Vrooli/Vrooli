package orchestration

import (
	"strings"
	"testing"

	"agent-manager/internal/adapters/runner/codecs"
	"agent-manager/internal/domain"
	"agent-manager/internal/orchestration/phases"

	"github.com/vrooli/cli-core/usagemode"
)

func TestUsageModeContextLimitIsAppliedAndRecordedForCodex(t *testing.T) {
	f := newModeFixture(t, nil)
	writeMode(t, func(m *usagemode.Mode) {
		s := m.Runners["codex"]
		s.ContextLimitTokens = 180000
		m.Runners["codex"] = s
	})
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	if cfg.ContextLimitTokens != 180000 {
		t.Fatalf("ContextLimitTokens = %d, want the mode's 180000", cfg.ContextLimitTokens)
	}
	if !strings.Contains(cfg.PolicySnapshot.SelectionReason, "usage_mode_context_limit:180000") {
		t.Fatalf("the snapshot must record the limit: %q", cfg.PolicySnapshot.SelectionReason)
	}
	args, err := codecs.NewCodexForTest().ControlArgs(cfg)
	if err != nil || !containsPair(args, "-c", "model_auto_compact_token_limit=180000") {
		t.Fatalf("codex launch args %v (%v) must carry the limit", args, err)
	}
}

func TestUsageModeContextLimitAloneStillApplies(t *testing.T) {
	// A mode that maps no model and sets no effort still carries its limit.
	f := newModeFixture(t, nil)
	f.o.roleResolver = modeRoleResolver{model: "gpt-6-luna"}
	writeMode(t, func(m *usagemode.Mode) {
		m.Runners["codex"] = usagemode.RunnerSettings{ContextLimitTokens: 150000}
	})
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	if cfg.ContextLimitTokens != 150000 || cfg.Model != "gpt-6-luna" {
		t.Fatalf("limit-only mode: limit=%d model=%q", cfg.ContextLimitTokens, cfg.Model)
	}
}

func TestNoModeLimitMeansNoLimitArgs(t *testing.T) {
	f := newModeFixture(t, nil)
	writeMode(t, nil)
	cfg := f.create(t, "code.default", CreateRunRequest{}, nil)
	args, _ := codecs.NewCodexForTest().ControlArgs(cfg)
	for _, arg := range args {
		if strings.Contains(arg, "auto_compact") {
			t.Fatalf("a mode without a limit must not add one: %v", args)
		}
	}
}

func TestClaudeContextLimitBelowMinimumIsRaisedAndStated(t *testing.T) {
	limit, note := modeContextLimit(domain.RunnerTypeClaudeCode, 60000)
	if limit != 100000 || !strings.Contains(note, "raised:60000_to_claude_minimum_100000") {
		t.Fatalf("limit=%d note=%q: Claude's 100k minimum must be applied and stated", limit, note)
	}
	if limit, note = modeContextLimit(domain.RunnerTypeClaudeCode, 250000); limit != 250000 || note != "" {
		t.Fatalf("explicit limit above the minimum changed: %d %q", limit, note)
	}
	if limit, _ = modeContextLimit(domain.RunnerTypeClaudeCode, 0); limit != 0 {
		t.Fatalf("no limit requested must stay unset, got %d", limit)
	}
	if limit, _ = modeContextLimit(domain.RunnerTypeOpenCode, 150000); limit != 0 {
		t.Fatalf("a runner without a control got a limit: %d", limit)
	}
}

func TestClaudeRunEnvCarriesTheLimitAndExplicitEnvWins(t *testing.T) {
	cfg := &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, ContextLimitTokens: 150000}
	env := phases.AssembleRunEnv(phases.AssembleRunEnvInput{Config: cfg})
	if env[phases.ClaudeAutoCompactEnv] != "150000" {
		t.Fatalf("env = %v, want the limit in CLAUDE_CODE_AUTO_COMPACT_WINDOW", env)
	}
	explicit := phases.MergeEnvVars(phases.MergeEnvInput{Runner: phases.ContextLimitEnv(cfg), Custom: map[string]string{phases.ClaudeAutoCompactEnv: "300000"}})
	if explicit[phases.ClaudeAutoCompactEnv] != "300000" {
		t.Fatalf("an explicit value must win over the mode's: %v", explicit)
	}
	if env := phases.AssembleRunEnv(phases.AssembleRunEnvInput{Config: &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode}}); env != nil {
		t.Fatalf("no limit must add nothing: %v", env)
	}
	if env := phases.AssembleRunEnv(phases.AssembleRunEnvInput{Config: &domain.RunConfig{RunnerType: domain.RunnerTypeCodex, ContextLimitTokens: 150000}}); env != nil {
		t.Fatalf("codex takes its limit as an argument, not env: %v", env)
	}
	small := &domain.RunConfig{RunnerType: domain.RunnerTypeClaudeCode, ContextLimitTokens: 50000}
	if got := phases.ContextLimitEnv(small)[phases.ClaudeAutoCompactEnv]; got != "100000" {
		t.Fatalf("a stored sub-minimum limit must still reach Claude as 100000, got %q", got)
	}
}

func containsPair(args []string, key, value string) bool {
	for i := 0; i+1 < len(args); i++ {
		if args[i] == key && args[i+1] == value {
			return true
		}
	}
	return false
}
