package summarizer

import (
	"context"
	_ "embed"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

const (
	providerOllama     = "ollama"
	providerOpenRouter = "openrouter"

	classificationFull        = "full_complete"
	classificationSignificant = "significant_progress"
	classificationSome        = "some_progress"
	classificationUncertain   = "uncertain"

	defaultTimeout = 45 * time.Second
	maxOutputChars = 12000
)

const rawSectionPlaceholder = "{{RAW_SECTION}}"

//go:embed prompt_template.txt
var promptTemplate string

// Config defines summarizer execution settings.
type Config struct {
	Provider string
	Model    string
}

// Input carries the execution output that should be summarized.
type Input struct {
	Output         string
	PromptOverride string
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

	prompt := strings.TrimSpace(input.PromptOverride)
	if prompt == "" {
		prompt = BuildPrompt(input.Output)
	}

	switch trimmedProvider {
	case providerOllama:
		return runOllama(ctx, cfg.Model, prompt)
	case providerOpenRouter:
		return runOpenRouter(ctx, cfg.Model, prompt)
	default:
		return Result{}, fmt.Errorf("unsupported summarizer provider: %s", trimmedProvider)
	}
}

// BuildPrompt generates the default recycler summarizer prompt for the provided output text.
func BuildPrompt(rawOutput string) string {
	output := strings.TrimSpace(rawOutput)
	if len(output) > maxOutputChars {
		output = output[:maxOutputChars] + "\n[TRUNCATED]"
	}

	template := promptTemplate
	if template == "" {
		return ""
	}
	var rawSection string
	if output == "" {
		rawSection = "\nNo execution output was captured. Respond accordingly.\n"
	} else {
		rawSection = "\nRaw execution output:\n" + output
	}

	if !strings.Contains(template, rawSectionPlaceholder) {
		return template + rawSection
	}

	return strings.ReplaceAll(template, rawSectionPlaceholder, rawSection)
}

func runOllama(ctx context.Context, model, prompt string) (Result, error) {
	model = strings.TrimSpace(model)
	if model == "" {
		model = "llama3.1:8b"
	}

	args := []string{"query"}
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

	return parseResult(trimmed)
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

	// Note: RecyclerPromptPrefix is defined in queue package but we don't import it
	// to avoid circular dependency. Using direct constant here.
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

	return parseResult(string(output))
}

func parseResult(raw string) (Result, error) {
	// NEW: No classification parsing - just extract the note content
	// The completeness metrics system will provide classification

	note := strings.TrimSpace(raw)

	if note == "" {
		return Result{}, errors.New("summarizer returned empty output")
	}

	// Return note content only, no classification
	return Result{
		Note:           note,
		Classification: "", // Not used - metrics system provides classification
	}, nil
}

// decorateNote is no longer used - removed in favor of metrics-based classification
// The completeness system provides the prefix based on objective scores

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
		Note:           "Unable to generate summary - no execution output was captured",
		Classification: "", // Not used - metrics system provides classification
	}
}

// SupportedClassifications exposes the canonical classification values.
func SupportedClassifications() []string {
	return []string{classificationFull, classificationSignificant, classificationSome, classificationUncertain}
}
