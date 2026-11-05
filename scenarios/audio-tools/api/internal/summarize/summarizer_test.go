package summarize

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"strings"
	"testing"
	"time"
)

func testSummarizer(t *testing.T, response string, inspect func(args []string, stdin string)) *Summarizer {
	t.Helper()
	return NewSummarizerWithRunner("resource-ollama-test", func(_ context.Context, args []string, stdin string) ([]byte, error) {
		if inspect != nil {
			inspect(args, stdin)
		}
		out, err := json.Marshal(map[string]any{
			"response":    response,
			"done_reason": "stop",
			"eval_count":  42,
		})
		if err != nil {
			t.Fatalf("marshal fake response: %v", err)
		}
		return out, nil
	})
}

func argValue(args []string, flag string) string {
	for i, arg := range args {
		if arg == flag && i+1 < len(args) {
			return args[i+1]
		}
	}
	return ""
}

func hasArg(args []string, want string) bool {
	for _, arg := range args {
		if arg == want {
			return true
		}
	}
	return false
}

func TestSummarizer_Summarize(t *testing.T) {
	s := testSummarizer(t, "summarized output", func(args []string, stdin string) {
		if strings.Join(args[:2], " ") != "gateway chat" {
			t.Fatalf("args prefix = %v", args[:2])
		}
		if argValue(args, "--role") != "test-role" {
			t.Errorf("--role = %q", argValue(args, "--role"))
		}
		if !strings.Contains(argValue(args, "--system"), "Tighten") {
			t.Errorf("expected light-level prompt, got %q", argValue(args, "--system"))
		}
		if stdin != "long text input" {
			t.Errorf("stdin = %q", stdin)
		}
	})
	result, err := s.Summarize(context.Background(), "long text input", "test-role", "light")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "summarized output" {
		t.Errorf("expected %q, got %q", "summarized output", result.Content)
	}
}

func TestSummarizer_UnknownLevel(t *testing.T) {
	s := testSummarizer(t, "ok", func(args []string, _ string) {
		if !strings.Contains(argValue(args, "--system"), "Rewrite") {
			t.Errorf("expected moderate-level prompt for unknown level, got %q", argValue(args, "--system"))
		}
	})
	if _, err := s.Summarize(context.Background(), "text", "model", "unknown_level"); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestSummarizer_EachLevel(t *testing.T) {
	levels := map[string]string{
		"light":    "Tighten",
		"moderate": "Rewrite",
		"heavy":    "brief spoken summary",
	}
	for level, expected := range levels {
		t.Run(level, func(t *testing.T) {
			s := testSummarizer(t, "ok", func(args []string, _ string) {
				if !strings.Contains(argValue(args, "--system"), expected) {
					t.Errorf("level %s: expected prompt containing %q, got %q", level, expected, argValue(args, "--system"))
				}
			})
			if _, err := s.Summarize(context.Background(), "text", "model", level); err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
		})
	}
}

func TestSummarizer_ServerError(t *testing.T) {
	s := NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		return nil, errors.New("chat: HTTP 500: internal error")
	})
	_, err := s.Summarize(context.Background(), "text", "model", "moderate")
	if err == nil {
		t.Error("expected error for gateway failure")
	}
}

func TestSummarizer_ModelNotInstalled(t *testing.T) {
	s := NewSummarizerWithRunner("resource-ollama-test", func(context.Context, []string, string) ([]byte, error) {
		return nil, errors.New(`chat: HTTP 404: {"error":"model \"missing:latest\" not found, try pulling it first"}`)
	})
	_, err := s.Summarize(context.Background(), "text", "missing:latest", "moderate")
	if !errors.Is(err, ErrSummarizeModelNotInstalled) {
		t.Fatalf("error = %v, want ErrSummarizeModelNotInstalled", err)
	}
}

func TestStripThinkTags(t *testing.T) {
	tests := []struct {
		name, input, want string
	}{
		{"no tags", "hello world", "hello world"},
		{"with think block", "<think>\nreasoning here\n</think>\nactual answer", "actual answer"},
		{"unclosed tag", "<think>partial reasoning", ""},
		{"multiple blocks", "<think>a</think>first<think>b</think>second", "firstsecond"},
		{"empty think", "<think></think>answer", "answer"},
		{"prefilled think", "reasoning text without opener\n</think>\nactual answer", "actual answer"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := StripThinkTags(tt.input)
			if got != tt.want {
				t.Errorf("StripThinkTags(%q) = %q, want %q", tt.input, got, tt.want)
			}
		})
	}
}

func TestSummarizer_StripsThinkTags(t *testing.T) {
	s := testSummarizer(t, "<think>\nlong reasoning\n</think>\nThe quick summary.", nil)
	result, err := s.Summarize(context.Background(), "text", "fixture-reasoning-model", "moderate")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Content != "The quick summary." {
		t.Errorf("expected think tags stripped, got %q", result.Content)
	}
}

func TestSummarizer_SendsTokenAndTemperatureOptions(t *testing.T) {
	cases := []struct {
		level     string
		inputText string
	}{
		{"light", strings.Repeat("word ", 400)},
		{"moderate", strings.Repeat("word ", 400)},
		{"heavy", strings.Repeat("word ", 400)},
		{"moderate", "tiny"},
	}
	for _, tc := range cases {
		t.Run(tc.level+"/"+strconv.Itoa(len(tc.inputText)), func(t *testing.T) {
			s := testSummarizer(t, "ok", func(args []string, _ string) {
				if got := argValue(args, "--temperature"); got != "0.2" {
					t.Errorf("temperature = %q, want 0.2", got)
				}
				want := strconv.Itoa(summarizeTokenBudget(tc.level, len(tc.inputText)))
				if got := argValue(args, "--max-tokens"); got != want {
					t.Errorf("max tokens = %q, want %s", got, want)
				}
			})
			if _, err := s.Summarize(context.Background(), tc.inputText, "fixture-safe-model", tc.level); err != nil {
				t.Fatalf("summarize: %v", err)
			}
		})
	}
}

func TestSummarizer_AddsReasoningHeadroomOnlyForReasoningModels(t *testing.T) {
	cases := []struct {
		model string
		want  int
	}{
		{"fixture-safe-model", summarizeTokenBudget("moderate", len("hello"))},
		{"fixture-reasoning-model", summarizeTokenBudget("moderate", len("hello")) + reasoningHeadroomTokens},
		{"fixture-reasoning-alt-model", summarizeTokenBudget("moderate", len("hello")) + reasoningHeadroomTokens},
	}
	for _, tc := range cases {
		t.Run(tc.model, func(t *testing.T) {
			s := testSummarizer(t, "ok", func(args []string, _ string) {
				got, _ := strconv.Atoi(argValue(args, "--max-tokens"))
				if got != tc.want {
					t.Errorf("max tokens: got %d, want %d", got, tc.want)
				}
			})
			if _, err := s.Summarize(context.Background(), "hello", tc.model, "moderate"); err != nil {
				t.Fatalf("summarize: %v", err)
			}
		})
	}
}

func TestSummarizeTokenBudget_LevelShape(t *testing.T) {
	if got := summarizeTokenBudget("heavy", 100); got != 120 {
		t.Errorf("heavy budget: got %d, want 120", got)
	}
	if got := summarizeTokenBudget("heavy", 100000); got != 120 {
		t.Errorf("heavy budget with large input: got %d, want 120", got)
	}

	smallModerate := summarizeTokenBudget("moderate", 10)
	if smallModerate != 90 {
		t.Errorf("moderate floor: got %d, want 90", smallModerate)
	}
	largeModerate := summarizeTokenBudget("moderate", 10000)
	if largeModerate <= smallModerate {
		t.Errorf("moderate budget should scale: got %d <= floor %d", largeModerate, smallModerate)
	}

	inputChars := 10000
	if summarizeTokenBudget("light", inputChars) <= summarizeTokenBudget("moderate", inputChars) {
		t.Errorf("light budget should exceed moderate for the same input")
	}

	if summarizeTokenBudget("unknown", inputChars) != summarizeTokenBudget("moderate", inputChars) {
		t.Errorf("unknown level should default to moderate budget")
	}
}

func TestSummarizer_SendsThinkFalse(t *testing.T) {
	s := testSummarizer(t, "ok", func(args []string, _ string) {
		if !hasArg(args, "--think=false") {
			t.Fatalf("expected --think=false in argv, got %v", args)
		}
	})
	if _, err := s.Summarize(context.Background(), "text", "fixture-reasoning-model", "moderate"); err != nil {
		t.Fatalf("summarize: %v", err)
	}
}

func TestSummarizer_UsesDirectModelFlagOnlyForExplicitModelSelectors(t *testing.T) {
	s := testSummarizer(t, "ok", func(args []string, _ string) {
		if argValue(args, "--model") != "custom:latest" {
			t.Fatalf("--model = %q, args=%v", argValue(args, "--model"), args)
		}
		if hasArg(args, "--role") {
			t.Fatalf("did not expect --role for direct selector: %v", args)
		}
	})
	if _, err := s.Summarize(context.Background(), "text", "custom:latest", "moderate"); err != nil {
		t.Fatalf("summarize: %v", err)
	}
}

func TestSummarizer_ReturnsDiagnostics(t *testing.T) {
	s := testSummarizer(t, "<think>\ndropped\n</think>\nkept", nil)
	result, err := s.Summarize(context.Background(), "text", "fixture-reasoning-model", "moderate")
	if err != nil {
		t.Fatalf("summarize: %v", err)
	}
	if result.Content != "kept" {
		t.Errorf("Content: got %q, want %q", result.Content, "kept")
	}
	if !strings.Contains(result.RawContent, "<think>") {
		t.Errorf("RawContent should be pre-strip, got %q", result.RawContent)
	}
	if result.DoneReason != "stop" {
		t.Errorf("DoneReason: got %q, want stop", result.DoneReason)
	}
	if result.EvalCount != 42 {
		t.Errorf("EvalCount: got %d, want 42", result.EvalCount)
	}
}

func TestSummarizer_Timeout(t *testing.T) {
	s := NewSummarizerWithRunner("resource-ollama-test", func(ctx context.Context, _ []string, _ string) ([]byte, error) {
		<-ctx.Done()
		return nil, ctx.Err()
	})
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	_, err := s.Summarize(ctx, "text", "model", "moderate")
	if err == nil {
		t.Error("expected error for timeout")
	}
}
