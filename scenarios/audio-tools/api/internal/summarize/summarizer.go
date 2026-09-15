package summarize

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strconv"
	"strings"
)

var ErrSummarizeModelNotInstalled = errors.New("summarize model is not installed")

const defaultOllamaGatewayBin = "resource-ollama"

// GatewayRunner runs resource-ollama gateway commands. Tests inject a fake
// runner; production uses exec.CommandContext.
type GatewayRunner func(ctx context.Context, args []string, stdin string) ([]byte, error)

// Summarizer calls the resource-ollama gateway to summarize text for TTS
// consumption.
type Summarizer struct {
	Bin    string
	Runner GatewayRunner
}

// NewSummarizer creates a summarizer backed by the resource-ollama gateway.
// The legacy parameters are ignored; they are retained so existing bootstrap
// call sites can migrate without a wider constructor churn.
func NewSummarizer(_ string, _ any) *Summarizer {
	return &Summarizer{
		Bin:    defaultOllamaGatewayBin,
		Runner: nil,
	}
}

// NewSummarizerWithRunner creates a gateway-backed summarizer with an injected
// runner for tests.
func NewSummarizerWithRunner(bin string, runner GatewayRunner) *Summarizer {
	if strings.TrimSpace(bin) == "" {
		bin = defaultOllamaGatewayBin
	}
	return &Summarizer{Bin: bin, Runner: runner}
}

// summarizeSystemPrompts maps summarization levels to system prompts.
// Prompts use explicit word/sentence budgets plus anti-preamble guards because
// small instruction-tuned models (qwen3 family) routinely ignore soft
// percentage targets. The hard cap still comes from options.num_predict below.
var summarizeSystemPrompts = map[string]string{
	"light":    "Tighten the following text for text-to-speech. Budget: at most 55% of the source word count. Remove filler, examples, and redundant phrasing. Keep technical details verbatim. End on a complete sentence. No preamble, no greeting, no restating the request — output only the tightened text.",
	"moderate": "Rewrite the following text as a spoken summary. Budget: at most 35% of the source word count. Keep only the single most important conclusion and the facts required to act on it. No lists unless the source is itself a list. End on a complete sentence. No preamble, no greeting, no restating the request — output only the summary.",
	"heavy":    "Write a brief spoken summary of the following text. Budget: at most 2 sentences and 40 words total. Focus on the single actionable takeaway. No preamble, no greeting, no restating the request — output only the summary.",
}

// summarizeTokenBudget returns the hard max-output-tokens (Ollama num_predict)
// for a given summarization level, sized against the input text. Token count is
// estimated as len(text)/4 characters per token, which matches the rough
// heuristic used by the qwen3 and llama tokenizers.
func summarizeTokenBudget(level string, inputChars int) int {
	inputTokens := inputChars / 4
	if inputTokens < 1 {
		inputTokens = 1
	}
	switch level {
	case "heavy":
		return 120
	case "light":
		budget := inputTokens * 55 / 100
		if budget < 90 {
			return 90
		}
		return budget
	default: // moderate and unknown
		budget := inputTokens * 35 / 100
		if budget < 90 {
			return 90
		}
		return budget
	}
}

// reasoningHeadroomTokens is extra num_predict budget reserved only when an
// operator explicitly chooses a reasoning model. The default summarizer model
// is non-reasoning, so normal TTS summaries stay fast and tightly bounded.
const reasoningHeadroomTokens = 2048

// SummarizerResponse carries the answer plus the diagnostic signals we need
// to distinguish a real empty response from a truncated/stripped one.
type SummarizerResponse struct {
	// Content is the final post-strip, trimmed summary. May be empty — callers
	// must inspect RawContent/DoneReason to classify the failure.
	Content string
	// RawContent is the pre-strip, trimmed model output. Used for diagnostics
	// (logging a short snippet, detecting think-tag truncation).
	RawContent string
	// DoneReason is Ollama's completion reason ("stop", "length", "load", ...).
	DoneReason string
	// EvalCount is the number of tokens Ollama generated.
	EvalCount int
}

// Summarize sends text to resource-ollama with a level-appropriate system prompt and
// returns the stripped summary plus diagnostic fields. We pass `think: false`
// through the gateway so reasoning models skip their <think> block entirely.
func (s *Summarizer) Summarize(ctx context.Context, text, selector, level string) (SummarizerResponse, error) {
	if s == nil {
		return SummarizerResponse{}, fmt.Errorf("summarizer is nil")
	}
	systemPrompt, ok := summarizeSystemPrompts[level]
	if !ok {
		systemPrompt = summarizeSystemPrompts["moderate"]
	}
	numPredict := summarizeTokenBudget(level, len(text))
	if isReasoningModel(selector) {
		numPredict += reasoningHeadroomTokens
	}

	selector = strings.TrimSpace(selector)
	if selector == "" {
		selector = DefaultSummarizeModel
	}
	selectorFlag := "--model"
	if SelectorIsRole(selector) {
		selectorFlag = "--role"
	}
	args := []string{
		"gateway", "chat",
		selectorFlag, selector,
		"--system", systemPrompt,
		"--max-tokens", strconv.Itoa(numPredict),
		"--temperature", "0.2",
		"--think=false",
		"--json",
		"--prompt-stdin",
	}

	runner := s.Runner
	if runner == nil {
		bin := strings.TrimSpace(s.Bin)
		if bin == "" {
			bin = defaultOllamaGatewayBin
		}
		runner = func(ctx context.Context, args []string, stdin string) ([]byte, error) {
			return runGatewayCLI(ctx, bin, args, stdin)
		}
	}
	out, err := runner(ctx, args, text)
	if err != nil {
		if looksLikeMissingOllamaModel(err.Error()) {
			return SummarizerResponse{}, fmt.Errorf("%w: %s", ErrSummarizeModelNotInstalled, err.Error())
		}
		return SummarizerResponse{}, fmt.Errorf("resource-ollama gateway chat: %w", err)
	}

	var result struct {
		Response   string `json:"response"`
		DoneReason string `json:"done_reason"`
		EvalCount  int    `json:"eval_count"`
	}
	if err := json.Unmarshal(out, &result); err != nil {
		return SummarizerResponse{}, fmt.Errorf("decode response: %w", err)
	}

	raw := strings.TrimSpace(result.Response)
	return SummarizerResponse{
		Content:    StripThinkTags(raw),
		RawContent: raw,
		DoneReason: result.DoneReason,
		EvalCount:  result.EvalCount,
	}, nil
}

func runGatewayCLI(ctx context.Context, bin string, args []string, stdin string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err != nil {
		if exitErr, ok := err.(*exec.ExitError); ok && len(exitErr.Stderr) > 0 {
			return nil, fmt.Errorf("%w: %s", err, strings.TrimSpace(string(exitErr.Stderr)))
		}
		return nil, err
	}
	return out, nil
}

func isReasoningModel(model string) bool {
	return IsReasoningModel(model)
}

func looksLikeMissingOllamaModel(body string) bool {
	body = strings.ToLower(body)
	return strings.Contains(body, "model") &&
		(strings.Contains(body, "not found") || strings.Contains(body, "pull"))
}

// StripThinkTags removes <think>...</think> blocks that reasoning models
// (e.g. qwen3) emit before their actual answer.
//
// qwen3's chat template prefills "<think>\n" into the prompt, so the
// generated content frequently *starts inside* the think block — there
// is a closing </think> but no opening <think>. We treat any text
// before the first </think> as reasoning to strip.
func StripThinkTags(s string) string {
	if firstClose := strings.Index(s, "</think>"); firstClose >= 0 {
		firstOpen := strings.Index(s, "<think>")
		if firstOpen < 0 || firstOpen > firstClose {
			s = s[firstClose+len("</think>"):]
		}
	}
	for {
		start := strings.Index(s, "<think>")
		if start < 0 {
			break
		}
		end := strings.Index(s, "</think>")
		if end < 0 {
			// Unclosed tag — strip from <think> to end
			s = s[:start]
			break
		}
		s = s[:start] + s[end+len("</think>"):]
	}
	return strings.TrimSpace(s)
}
