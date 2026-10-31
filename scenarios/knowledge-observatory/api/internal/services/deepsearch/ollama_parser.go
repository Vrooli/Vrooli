package deepsearch

// DOC: docs/reference/configuration.md#api-runtime-configuration

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"strings"
)

// OllamaParser uses the resource-ollama gateway CLI to coerce unstructured
// agent output into JSON results. All daemon traffic is funnelled through the
// CLI so the host-wide semaphore bounds fleet-wide parallelism.
type OllamaParser struct {
	Role string

	// Runner is an optional seam for tests. Production callers leave it nil and
	// the default exec-based runner is used.
	Runner func(ctx context.Context, args []string, stdin string) ([]byte, error)
}

func (o *OllamaParser) Parse(ctx context.Context, raw string) ([]DeepSearchResult, error) {
	role := strings.TrimSpace(o.Role)
	if role == "" {
		return nil, fmt.Errorf("ollama parser role not configured")
	}
	prompt := buildOllamaPrompt(raw)

	args := []string{"gateway", "generate", "--role", role, "--json", "--prompt-stdin"}
	out, err := o.run(ctx, args, prompt)
	if err != nil {
		return nil, fmt.Errorf("resource-ollama gateway generate failed: %w", err)
	}

	var decoded struct {
		Response string `json:"response"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(out), &decoded); err != nil {
		return nil, fmt.Errorf("decode gateway generate response: %w", err)
	}

	parsed, ok := parseJSONResults(decoded.Response)
	if !ok {
		return nil, fmt.Errorf("ollama response did not contain valid JSON")
	}
	return parsed, nil
}

func (o *OllamaParser) run(ctx context.Context, args []string, stdin string) ([]byte, error) {
	if o.Runner != nil {
		return o.Runner(ctx, args, stdin)
	}
	cmd := exec.CommandContext(ctx, "resource-ollama", args...)
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

func buildOllamaPrompt(raw string) string {
	return strings.TrimSpace(fmt.Sprintf(`Convert the following agent output into JSON only.

Return a JSON array of objects with fields:
- path (string)
- relevance (number 0-1)
- summary (string)
- match_reason (string)
- references (array of strings)
- snippet (string)

If a field is missing, use an empty string or empty array. Return JSON only.

OUTPUT:
%s`, raw))
}
