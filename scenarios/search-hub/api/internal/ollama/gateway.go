// Package ollama is search-hub's one client to the shared local Ollama daemon.
// Every LLM call search-hub makes — query classification, candidate reranking,
// and corpus-generation query inversion — goes through `resource-ollama
// gateway`, the same throttled resource CLI cli-health and agent-manager use, so
// the daemon is never hit directly and load stays governed in one place.
//
// The package is deliberately transport-only: it shells one completion, unwraps
// the gateway's JSON envelope, and strips the reasoning blocks small qwen3
// models emit even under `/no_think`. Prompt construction and response PARSING
// (each caller's JSON shape) stay with the caller — this package holds no
// task-specific knowledge, so a new LLM caller reuses it without widening it.
package ollama

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// Bin is the resource CLI that fronts the shared Ollama daemon.
const Bin = "resource-ollama"

// Generate shells one completion through `resource-ollama gateway generate
// --json` and returns the raw stdout envelope (use UnwrapResponse to read the
// text). temperature is pinned to 0 for determinism; maxTokens caps the reply.
// An error carries the gateway's stderr (one line) when present.
func Generate(ctx context.Context, role, prompt string, maxTokens int) ([]byte, error) {
	args := []string{
		"gateway", "generate",
		"--role", role,
		"--json",
		"--max-tokens", fmt.Sprintf("%d", maxTokens),
		"--temperature", "0",
		"--prompt-stdin",
	}
	cmd := exec.CommandContext(ctx, Bin, args...)
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%s: %w", oneLine(msg), err)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}

// Available reports whether the Ollama daemon is reachable (`resource-ollama
// status` exits 0). Callers on a hot path rely on Generate's error for graceful
// degradation instead; this is for status surfaces and pre-flight checks.
func Available(ctx context.Context) bool {
	return exec.CommandContext(ctx, Bin, "status").Run() == nil
}

// UnwrapResponse returns the model's text. `resource-ollama gateway generate
// --json` wraps the completion as {"response":"…","eval_count":N}; if the output
// is not that envelope (e.g. a plain-text run) it is returned as-is.
func UnwrapResponse(raw []byte) string {
	trimmed := bytes.TrimSpace(raw)
	var env struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(trimmed, &env); err == nil && env.Response != "" {
		return env.Response
	}
	return string(trimmed)
}

// StripThink removes qwen3 reasoning blocks (<think>…</think>) that survive even
// with /no_think (they come back empty but present). An unterminated block drops
// everything from it onward.
func StripThink(s string) string {
	for {
		start := strings.Index(s, "<think>")
		if start == -1 {
			break
		}
		end := strings.Index(s, "</think>")
		if end == -1 || end < start {
			s = s[:start]
			break
		}
		s = s[:start] + s[end+len("</think>"):]
	}
	return s
}

// ExtractJSONObject returns the first balanced {…} object in s, ignoring braces
// inside JSON strings. Empty string when none is found.
func ExtractJSONObject(s string) string {
	start := strings.Index(s, "{")
	if start == -1 {
		return ""
	}
	depth := 0
	inStr := false
	escaped := false
	for i := start; i < len(s); i++ {
		c := s[i]
		if escaped {
			escaped = false
			continue
		}
		if c == '\\' && inStr {
			escaped = true
			continue
		}
		if c == '"' {
			inStr = !inStr
			continue
		}
		if inStr {
			continue
		}
		switch c {
		case '{':
			depth++
		case '}':
			depth--
			if depth == 0 {
				return s[start : i+1]
			}
		}
	}
	return ""
}

// oneLine collapses whitespace so a multi-line stderr becomes a single log line.
func oneLine(s string) string {
	return strings.Join(strings.Fields(s), " ")
}
