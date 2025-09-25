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
	builder.WriteString("You are an expert operator for the Vrooli Ecosystem Manager. Review the raw execution output of a single task run and produce an updated note plus a completion classification.\n")
	builder.WriteString("Respond ONLY in plain text using this exact layout:\n")
	builder.WriteString("classification: <full_complete|partial_progress|uncertain>\n")
	builder.WriteString("note:\n")
	builder.WriteString("<multi-line note content>\n")
	builder.WriteString("Do not include any other labels, JSON, or decorative formatting.\n")
	builder.WriteString("\nRules:\n")
	builder.WriteString("- Base every decision strictly on the provided output text. Assume no other context, counters, or previous notes.\n")
	builder.WriteString("- Use EXACTLY one of these note templates:\n")
	builder.WriteString("  1. 'Not sure current status' when the output is missing, corrupt, mostly errors, or fails to demonstrate forward progress. Classification MUST be 'uncertain'.\n")
	builder.WriteString("  2. 'Notes from last time:' followed by concise bullets when there is real progress but clear unfinished work or follow-up items. Classification MUST be 'partial_progress'.\n")
	builder.WriteString("  3. 'Already pretty good, but could use some additional validation/tidying. Notes from last time:' + bullets when the output claims substantial completion yet still mentions optional polish, validation gaps, or outstanding checks. Classification MUST be 'partial_progress'.\n")
	builder.WriteString("  4. 'Ready for finalization. Highlights:' + bullets when the output proves the task is truly finished with zero follow-up work, no failing tests, and complete validation evidence. Classification MUST be 'full_complete'.\n")
	builder.WriteString("- When the transcript supplies bullet lists or numbered sections, carry them forward verbatim; return as many bullets as appear in the source material.\n")
	builder.WriteString("\n- Keep the note under 1800 characters.\n")
	builder.WriteString("- Classification guidance:\n  * full_complete – output must prove the task is finished, all critical tests pass, and there are NO TODOs, next steps (even optional ones), or validation gaps.\n  * partial_progress – output shows momentum but highlights remaining work, optional follow-ups, pending validation, or incomplete testing.\n  * uncertain – output is inconclusive, broken, or dominated by failures.\n")
	builder.WriteString("- Never assign classification 'full_complete' if the output references TODOs, next steps, skipped tests, failing checks, missing coverage, open risks, or anything left to verify. Default to 'partial_progress' instead.\n")
	builder.WriteString("- Ignore orchestration boilerplate lines that begin with tokens like [HEADER], [INFO], [WARNING], or shell prompts — never mention them in the note.\n")
	builder.WriteString("- Do not repeat raw headings such as 'Task Completion Summary', 'Task ID', or 'Status'; blend the underlying facts into the required template.\n")
	builder.WriteString("- When the output shows blockers, crashes, failed checks, or ongoing outages, you MUST choose classification 'uncertain' and the 'Not sure current status' template.\n")
	builder.WriteString("- When constructing bullet lists, restate every substantive point, validation artifact, or risk with minimal compression. Prefer verbatim phrasing (minus boilerplate) so that downstream agents retain the nuance.\n")
	builder.WriteString("- There is no upper limit on bullet count if the transcript is dense; do not merge ideas together or omit quantitative details, command outputs, or follow-up guidance.\n")
	builder.WriteString("- Each bullet should be a complete sentence and may reference inline code, command output, metrics, or section labels exactly as provided.\n")
	builder.WriteString("- If the transcript includes multiple sections (Accomplishments, Validation, Pending work, etc.), mirror each section by emitting bullets in the same order so nothing is lost.\n")
	builder.WriteString("- Do not cut important details for brevity. The goal is to preserve nearly all of the substantive content without the outer headings.\n")
	builder.WriteString("- Never include fields outside note and classification.\n")

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
