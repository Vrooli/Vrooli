package codecs

import (
	"slices"
	"testing"
	"time"

	"agent-manager/internal/adapters/runner"
	"agent-manager/internal/domain"
	"github.com/google/uuid"
)

func TestAntigravityBuildArgsUsesRunnerDefaultSymbol(t *testing.T) {
	c := NewAntigravityForTest()
	args := c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeAntigravity, Model: "default"},
		Prompt:         "safe fixture",
	})
	if !slices.Equal(args, []string{"--print", "safe fixture"}) {
		t.Fatalf("default model args = %q, want no explicit model flag", args)
	}

	args = c.BuildArgs(c.NewState(), runner.ExecuteRequest{
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeAntigravity, Model: "gemini-test"},
		Prompt:         "safe fixture",
	})
	if !slices.Equal(args, []string{"--print", "safe fixture", "--model", "gemini-test"}) {
		t.Fatalf("explicit model args = %q", args)
	}
}

func TestAntigravityCapabilitiesAreHonestForPrintMode(t *testing.T) {
	caps := NewAntigravityForTest().Capabilities()
	if !caps.SupportsMessages || !caps.SupportsStreaming || !caps.SupportsContinuation || !caps.SupportsCancellation || !caps.SupportsRunnerDefault {
		t.Fatalf("basic capabilities = %+v", caps)
	}
	if caps.SupportsToolEvents || caps.SupportsCostTracking || caps.SupportsImageAttachments || caps.SupportsToolRestriction || caps.SupportsEffort {
		t.Fatalf("unstable print capabilities claimed: %+v", caps)
	}
}

func TestAntigravityBuildContinueArgsUsesSharedResumeContract(t *testing.T) {
	c := NewAntigravityForTest()
	prompt := `follow "up"; $() stays literal`
	args := c.BuildContinueArgs(c.NewState(), runner.ContinueRequest{
		SessionID: "native-session", Prompt: prompt,
		ResolvedConfig: &domain.RunConfig{RunnerType: domain.RunnerTypeAntigravity, Model: "gemini-test"},
	})
	want := []string{"--conversation", "native-session", "--print", prompt, "--model", "gemini-test"}
	if !slices.Equal(args, want) {
		t.Fatalf("continuation args = %q, want %q", args, want)
	}
}

func TestAntigravityGenericTranscriptRecordPreservesSourceTime(t *testing.T) {
	line := `{"step_index":9,"type":"GENERIC","source":"MODEL","status":"DONE","created_at":"2026-11-24T10:00:03Z","content":"sanitized generic assistant fixture"}`
	result := NewAntigravityForTest().NewTranscriptParser().ParseTranscriptLine(uuid.Nil, line)
	want := time.Date(2026, 11, 24, 10, 0, 3, 0, time.UTC)
	if result.Timestamp.IsZero() || !result.Timestamp.Equal(want) {
		t.Fatalf("generic transcript timestamp = %v, want %v", result.Timestamp, want)
	}
	if len(result.Events) != 1 || !result.Events[0].Timestamp.Equal(want) {
		t.Fatalf("generic transcript events = %+v, want one source-timed event at %v", result.Events, want)
	}
	message, ok := result.Events[0].Data.(*domain.MessageEventData)
	if !ok || message.Content != "sanitized generic assistant fixture" || message.NativeRecordSequence != 9 {
		t.Fatalf("generic transcript event data = %#v", result.Events[0].Data)
	}
}
