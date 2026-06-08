package measureindex

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// ollamaBin is the resource CLI that fronts the shared Ollama daemon — the same
// throttled gateway cli-health, search-hub, and agent-manager use, so the daemon
// is never hit directly and load stays governed in one place.
const ollamaBin = "resource-ollama"

// defaultExtractModel is the local model the constrained param extractor uses.
// llama3.2:3b is small, fast, already pulled, and (per the aisearch score-regime
// rehab) extracts reliably at low latency where a reasoning model (qwen3) is
// slow. Override with MEASURES_HEALTH_EXTRACT_MODEL.
const defaultExtractModel = "llama3.2:3b"

// extractMaxTokens caps the extractor's reply. It only emits a short JSON object
// ({"found":..,"value":..,"confidence":..}); 256 is ample headroom.
const extractMaxTokens = 256

// ollamaCompleter is the production measures.Completer: it shells one completion
// through `resource-ollama gateway generate --json` and returns the raw stdout
// envelope. The measures-go LLMExtractor owns the prompt construction and the
// response PARSING (unwrap envelope, strip <think>, decode strict JSON) — this
// type is transport-only, holding no extraction-task knowledge.
type ollamaCompleter struct {
	model     string
	maxTokens int
	// run executes the gateway command; seamed so tests inject a deterministic
	// runner instead of the real daemon.
	run func(ctx context.Context, model, prompt string, maxTokens int) ([]byte, error)
}

// newOllamaCompleter returns the gateway-backed completer. Model resolves from
// MEASURES_HEALTH_EXTRACT_MODEL or falls back to defaultExtractModel.
func newOllamaCompleter() *ollamaCompleter {
	model := strings.TrimSpace(os.Getenv("MEASURES_HEALTH_EXTRACT_MODEL"))
	if model == "" {
		model = defaultExtractModel
	}
	return &ollamaCompleter{model: model, maxTokens: extractMaxTokens, run: ollamaGenerate}
}

// Complete runs one constrained completion and returns the raw gateway stdout
// (the LLMExtractor unwraps + parses it).
func (c *ollamaCompleter) Complete(ctx context.Context, prompt string) (string, error) {
	out, err := c.run(ctx, c.model, prompt, c.maxTokens)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

// ollamaAvailable reports whether the Ollama daemon is reachable
// (`resource-ollama status` exits 0). Used only by the Status surface — the
// query hot path relies on the extractor's error for graceful degradation.
func ollamaAvailable(ctx context.Context) bool {
	return exec.CommandContext(ctx, ollamaBin, "status").Run() == nil
}

// ollamaGenerate shells one completion through the gateway with temperature
// pinned to 0 for determinism. An error carries the gateway's stderr (one line).
func ollamaGenerate(ctx context.Context, model, prompt string, maxTokens int) ([]byte, error) {
	args := []string{
		"gateway", "generate",
		"--model", model,
		"--json",
		"--max-tokens", fmt.Sprintf("%d", maxTokens),
		"--temperature", "0",
		"--prompt-stdin",
	}
	cmd := exec.CommandContext(ctx, ollamaBin, args...)
	cmd.Stdin = strings.NewReader(prompt)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("%s: %w", strings.Join(strings.Fields(msg), " "), err)
		}
		return nil, err
	}
	return stdout.Bytes(), nil
}
