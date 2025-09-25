package summarizer

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/ecosystem-manager/api/pkg/tasks"
)

const (
	providerOllama     = "ollama"
	providerOpenRouter = "openrouter"

	classificationFull      = "full_complete"
	classificationPartial   = "partial_progress"
	classificationUncertain = "uncertain"

	defaultTimeout = 45 * time.Second
	maxOutputChars = 12000
)

// Config defines summarizer execution settings.
type Config struct {
	Provider string
	Model    string
}

// Input bundles task metadata and execution output for summarization.
type Input struct {
	Task         tasks.TaskItem
	Output       string
	PreviousNote string
}

// Result captures the summarizer outcome.
type Result struct {
	Note           string `json:"note"`
	Classification string `json:"classification"`
}

// GenerateNote produces a structured note and classification using the configured provider.
func GenerateNote(ctx context.Context, cfg Config, input Input) (Result, error) {
	trimmedProvider := strings.ToLower(strings.TrimSpace(cfg.Provider))
	if trimmedProvider == "" {
		return Result{}, errors.New("summarizer provider not configured")
	}

	if ctx == nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), defaultTimeout)
		defer cancel()
	}

	prompt := buildPrompt(input)

	switch trimmedProvider {
	case providerOllama:
		return runOllama(ctx, cfg.Model, prompt)
	case providerOpenRouter:
		return runOpenRouter(ctx, cfg.Model, prompt)
	default:
		return Result{}, fmt.Errorf("unsupported summarizer provider: %s", trimmedProvider)
	}
}

func buildPrompt(input Input) string {
	output := strings.TrimSpace(input.Output)
	if len(output) > maxOutputChars {
		output = output[:maxOutputChars] + "\n[TRUNCATED]"
	}

	previousNote := strings.TrimSpace(input.PreviousNote)
	if previousNote == "" {
		previousNote = "(none)"
	}

	var builder strings.Builder
	builder.WriteString("You are an expert operator for the Vrooli Ecosystem Manager. Summarize the latest execution of a task into the canonical note patterns and classify the output.\n")
	builder.WriteString("Always respond with compact JSON using this schema:\n")
	builder.WriteString("{\n  \"note\": \"string\",\n  \"classification\": \"full_complete|partial_progress|uncertain\"\n}\n")
	builder.WriteString("\nRules:\n")
	builder.WriteString("- Use EXACTLY one of the following note templates based on the situation:\n")
	builder.WriteString("  1. 'Not sure current status' when output is missing, corrupt, or inconclusive.\n")
	builder.WriteString("  2. 'Notes from last time:' + concise bullet list when there was useful output but follow-up work required.\n")
	builder.WriteString("  3. 'Already pretty good, but could use some additional validation/tidying. Notes from last time:' + bullets when the output claims completion once.\n")
	builder.WriteString("  4. 'Likely complete, but could use some additional validation/tidying. Notes from last time:' + bullets when the output repeatedly claims completion.\n")
	builder.WriteString("- For bullet lists, limit to the most actionable 4-6 points derived from the output.\n")
	builder.WriteString("- Keep the note under 1200 characters.\n")
	builder.WriteString("- The classification meaning:\n  * full_complete – agent asserts task is finished.\n  * partial_progress – useful progress but more work remains.\n  * uncertain – output missing, contradictory, or failure details only.\n")
	builder.WriteString("- If output references blockers or failures, prefer classification 'uncertain'.\n")
	builder.WriteString("- When you see errors, stack traces, failed health checks, or statements that the task is still broken/not running, you MUST choose classification 'uncertain' and use the 'Not sure current status' note template.\n")
	builder.WriteString("- Never include additional fields outside note and classification.\n")

	builder.WriteString("\nTask metadata:\n")
	builder.WriteString(fmt.Sprintf("- ID: %s\n", input.Task.ID))
	builder.WriteString(fmt.Sprintf("- Title: %s\n", input.Task.Title))
	builder.WriteString(fmt.Sprintf("- Type: %s\n", input.Task.Type))
	builder.WriteString(fmt.Sprintf("- Operation: %s\n", input.Task.Operation))
	builder.WriteString(fmt.Sprintf("- Completion count: %d\n", input.Task.CompletionCount))
	builder.WriteString(fmt.Sprintf("- Consecutive completion claims: %d\n", input.Task.ConsecutiveCompletionClaims))
	builder.WriteString(fmt.Sprintf("- Consecutive failures: %d\n", input.Task.ConsecutiveFailures))
	builder.WriteString(fmt.Sprintf("- Previous note: %s\n", previousNote))

	if output == "" {
		builder.WriteString("\nNo execution output was captured. Respond accordingly.\n")
	} else {
		builder.WriteString("\nRaw execution output (analyze carefully):\n")
		builder.WriteString(output)
	}

	return builder.String()
}

func runOllama(ctx context.Context, model, prompt string) (Result, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = "llama3.1:8b"
	}

	args := []string{"query", "--json"}
	if model != "" {
		args = append(args, "--model", model)
	}
	args = append(args, prompt)

	cmd := exec.CommandContext(ctx, "resource-ollama", args...)
	cmd.Env = os.Environ()

	output, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("resource-ollama query failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	trimmed := strings.TrimSpace(string(output))
	if trimmed == "" {
		return Result{}, errors.New("resource-ollama query returned empty output")
	}

	return parseResult([]byte(trimmed))
}

func runOpenRouter(ctx context.Context, model, prompt string) (Result, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = "openai/gpt-4o-mini"
	}

	root := resolveVrooliRoot()
	promptDir := filepath.Join(root, "data", "resources", "openrouter", "content", "prompts")
	if err := os.MkdirAll(promptDir, 0o775); err != nil {
		return Result{}, fmt.Errorf("failed to ensure OpenRouter prompt directory: %w", err)
	}

	promptID := fmt.Sprintf("ecosystem-recycler-%d", time.Now().UnixNano())
	promptFile := filepath.Join(promptDir, promptID+".txt")
	if err := os.WriteFile(promptFile, []byte(prompt), 0o600); err != nil {
		return Result{}, fmt.Errorf("failed to write OpenRouter prompt file: %w", err)
	}
	defer os.Remove(promptFile)

	cmd := exec.CommandContext(ctx, "resource-openrouter", "content", "execute", "--name", promptID, "--model", model)
	cmd.Dir = root
	cmd.Env = os.Environ()
	if os.Getenv("VROOLI_ROOT") == "" {
		cmd.Env = append(cmd.Env, fmt.Sprintf("VROOLI_ROOT=%s", root))
	}

	output, err := cmd.CombinedOutput()
	if err != nil {
		return Result{}, fmt.Errorf("resource-openrouter failed: %w (output: %s)", err, strings.TrimSpace(string(output)))
	}

	return parseResult(output)
}

func parseResult(output []byte) (Result, error) {
	sanitized := sanitizeLLMJSON(output)
	var payload struct {
		Note           string `json:"note"`
		Classification string `json:"classification"`
	}

	if err := json.Unmarshal(sanitized, &payload); err != nil {
		trimmed := strings.TrimSpace(string(sanitized))
		return Result{}, fmt.Errorf("summarizer returned invalid JSON: %w (payload: %s)", err, trimmed)
	}

	classification := strings.ToLower(strings.TrimSpace(payload.Classification))
	switch classification {
	case classificationFull, classificationPartial, classificationUncertain:
	default:
		classification = classificationUncertain
	}

	note := strings.TrimSpace(payload.Note)
	if note == "" {
		note = "Not sure current status"
		classification = classificationUncertain
	}

	return Result{Note: note, Classification: classification}, nil
}

func resolveVrooliRoot() string {
	if root := os.Getenv("VROOLI_ROOT"); root != "" {
		return root
	}

	if wd, err := os.Getwd(); err == nil {
		dir := wd
		for dir != string(filepath.Separator) && dir != "." {
			if _, statErr := os.Stat(filepath.Join(dir, ".vrooli")); statErr == nil {
				return dir
			}
			parent := filepath.Dir(dir)
			if parent == dir {
				break
			}
			dir = parent
		}
	}

	return "."
}

// DefaultResult returns a safe fallback result when summarization cannot be performed.
func DefaultResult() Result {
	return Result{
		Note:           "Not sure current status",
		Classification: classificationUncertain,
	}
}

// SupportedClassifications exposes the canonical classification values.
func SupportedClassifications() []string {
	return []string{classificationFull, classificationPartial, classificationUncertain}
}

func sanitizeLLMJSON(raw []byte) []byte {
	trimmed := strings.TrimSpace(string(raw))
	trimmed = strings.TrimPrefix(trimmed, "\ufeff")
	trimmed = strings.TrimSpace(trimmed)

	// Remove optional Markdown code fences (```json ... ```)
	if strings.HasPrefix(trimmed, "```") {
		for strings.HasPrefix(trimmed, "```") {
			trimmed = strings.TrimPrefix(trimmed, "```")
			trimmed = strings.TrimSpace(trimmed)
			lowered := strings.ToLower(trimmed)
			switch {
			case strings.HasPrefix(lowered, "json\n"):
				trimmed = strings.TrimSpace(trimmed[5:])
			case strings.HasPrefix(lowered, "json\r\n"):
				trimmed = strings.TrimSpace(trimmed[7:])
			case strings.HasPrefix(lowered, "json"):
				trimmed = strings.TrimSpace(trimmed[4:])
			}
		}

		if idx := strings.LastIndex(trimmed, "```"); idx >= 0 {
			trimmed = trimmed[:idx]
		}
		trimmed = strings.TrimSpace(trimmed)
	}

	var builder strings.Builder
	builder.Grow(len(trimmed) + 16)

	inString := false
	escaped := false

	for _, r := range trimmed {
		switch r {
		case '\\':
			builder.WriteRune(r)
			if inString {
				escaped = !escaped
			}
			continue
		case '"':
			if !escaped {
				inString = !inString
			}
			builder.WriteRune(r)
			if escaped {
				escaped = false
			}
			continue
		case '\n', '\r', '\u2028', '\u2029', '\u0085':
			if inString {
				builder.WriteString("\\n")
			} else {
				builder.WriteRune('\n')
			}
			escaped = false
			continue
		default:
			builder.WriteRune(r)
			if escaped {
				escaped = false
			}
		}
	}

	sanitized := strings.TrimSpace(builder.String())
	if sanitized == "" {
		return raw
	}

	return []byte(sanitized)
}
