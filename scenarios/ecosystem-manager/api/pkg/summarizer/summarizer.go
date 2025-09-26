package summarizer

import (
	"bufio"
	"context"
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

// Input carries the execution output that should be summarized.
type Input struct {
	Output string
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

	var builder strings.Builder
	builder.WriteString("You are an expert operator for the Vrooli Ecosystem Manager. Review the raw execution output of a single task run and tidy the final response without removing information.\n")
	builder.WriteString("Respond ONLY in plain text using this exact layout:\n")
	builder.WriteString("classification: <full_complete|partial_progress|uncertain>\n")
	builder.WriteString("note:\n")
	builder.WriteString("<multi-line note content that keeps the required sections>\n")
	builder.WriteString("Do not include any other labels, JSON, decorative formatting, or summary sentences outside the required structure.\n")
	builder.WriteString("\nOutput Format Requirements:\n")
	builder.WriteString("1. Present the four required sections exactly once and in this order, each on its own line using the agent format:\n")
	builder.WriteString("   - **What was accomplished:** <detailed content>\n")
	builder.WriteString("   - **Current status:** <detailed content>\n")
	builder.WriteString("   - **Remaining issues or limitations:** <detailed content>\n")
	builder.WriteString("   - **Validation evidence:** <detailed content>\n")
	builder.WriteString("   Keep the field labels identical (including capitalization and punctuation).\n")
	builder.WriteString("2. Preserve every concrete fact, metric, command output, log line, checklist item, or nuance from the transcript by restating it inside the relevant section. When the source supplies multiple bullets or numbered items, carry each item forward as its own sub-bullet (using '- ') beneath the appropriate section.\n")
	builder.WriteString("3. Only tidy grammar, spacing, or obvious duplication. Never drop, merge, or generalize distinct data points.\n")
	builder.WriteString("4. If the transcript lacks information for a section, write 'None reported' or 'Not captured in output' rather than omitting the section.\n")
	builder.WriteString("5. Do not add any introductory sentence or lead-in; the system will prepend one automatically based on your classification.\n")
	builder.WriteString("6. Classification guidance:\n")
	builder.WriteString("   - full_complete – transcript proves the task is finished, all critical tests pass, and there are no TODOs or pending validation.\n")
	builder.WriteString("   - partial_progress – transcript shows progress but leaves unfinished work, TODOs, optional polish, or pending validation.\n")
	builder.WriteString("   - uncertain – transcript is inconclusive, dominated by failures, or missing.\n")
	builder.WriteString("   Never assign 'full_complete' if the output references TODOs, skipped tests, failing checks, open risks, or follow-up actions.\n")
	builder.WriteString("7. Ignore orchestration boilerplate lines that begin with tokens like [HEADER], [INFO], or shell prompts, but retain any meaningful diagnostics that follow them.\n")
	builder.WriteString("8. Keep the entire note under 1800 characters while still capturing all substantive details.\n")
	builder.WriteString("9. Never include fields outside 'classification' and 'note'.\n")

	if output == "" {
		builder.WriteString("\nNo execution output was captured. Respond accordingly.\n")
	} else {
		builder.WriteString("\nRaw execution output:\n")
		builder.WriteString(output)
	}

	return builder.String()
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
	scanner := bufio.NewScanner(strings.NewReader(raw))
	scanner.Split(bufio.ScanLines)

	var firstLine string
	for scanner.Scan() {
		candidate := strings.TrimSpace(scanner.Text())
		if candidate == "" {
			continue
		}
		firstLine = candidate
		break
	}

	if firstLine == "" {
		return Result{}, errors.New("summarizer returned empty output")
	}
	if !strings.HasPrefix(strings.ToLower(firstLine), "classification:") {
		return Result{}, fmt.Errorf("unexpected summarizer format: %s", firstLine)
	}

	classification := strings.TrimSpace(firstLine[len("classification:"):])
	classification = strings.ToLower(classification)
	switch classification {
	case classificationFull, classificationPartial, classificationUncertain:
	default:
		classification = classificationUncertain
	}

	noteStarted := false
	var builder strings.Builder

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if !noteStarted {
			if trimmed == "" {
				continue
			}
			if strings.EqualFold(trimmed, "note:") {
				noteStarted = true
				continue
			}

			// No explicit note header; treat this line as the start of the note content.
			noteStarted = true
			builder.WriteString(line)
			builder.WriteRune('\n')
			continue
		}

		builder.WriteString(line)
		builder.WriteRune('\n')
	}

	if !noteStarted {
		return Result{}, errors.New("summarizer note content missing")
	}

	note := strings.TrimSpace(builder.String())
	if note == "" {
		note = "Not sure current status"
		classification = classificationUncertain
	}

	return Result{Note: decorateNote(classification, note), Classification: classification}, nil
}

func decorateNote(classification, note string) string {
	trimmed := strings.TrimSpace(note)
	var lead string
	switch classification {
	case classificationFull:
		lead = "Likely complete, but may benefit from additional validation/tidying. Notes from last time:"
	case classificationPartial:
		lead = choosePartialLead(trimmed)
	case classificationUncertain:
		lead = "Not sure current status"
	default:
		lead = "Notes from last time:"
	}

	if trimmed == "" {
		return lead
	}

	if strings.HasPrefix(trimmed, lead) {
		return trimmed
	}

	return lead + "\n\n" + trimmed
}

func choosePartialLead(note string) string {
	const (
		standardLead = "Notes from last time:"
		nuanceLead   = "Already pretty good, but could use some additional validation/tidying. Notes from last time:"
	)

	lower := strings.ToLower(note)
	marker := strings.ToLower("**Remaining issues or limitations:**")
	idx := strings.Index(lower, marker)
	if idx == -1 {
		return standardLead
	}

	segment := lower[idx+len(marker):]
	if newline := strings.Index(segment, "\n"); newline != -1 {
		segment = segment[:newline]
	}

	segment = strings.TrimSpace(segment)
	if segment == "" {
		return nuanceLead
	}

	if strings.Contains(segment, "none") || strings.Contains(segment, "not captured") || strings.Contains(segment, "n/a") {
		return nuanceLead
	}

	return standardLead
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
