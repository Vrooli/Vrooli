package fix

import (
	"context"
	"encoding/json"
	"strings"

	"scenario-to-cloud/domain"
	"scenario-to-cloud/tasks/shared"
)

// Handler implements TaskHandler for fix tasks.
type Handler struct{}

// NewHandler creates a new fix handler.
func NewHandler() *Handler {
	return &Handler{}
}

// TaskType returns the task type this handler processes.
func (h *Handler) TaskType() domain.TaskType {
	return domain.TaskTypeFix
}

// BuildPromptAndContext creates the prompt and attachments for one fix iteration.
func (h *Handler) BuildPromptAndContext(ctx context.Context, input shared.TaskInput) (shared.PromptResult, error) {
	return BuildPromptAndContext(input)
}

// AgentTag returns the tag for fix runs.
func (h *Handler) AgentTag() string {
	return "scenario-to-cloud-fix"
}

// ShouldContinue determines if the fix loop should continue.
// Returns (continue, reason) based on the agent's output.
func (h *Handler) ShouldContinue(ctx context.Context, task *domain.Investigation, result *shared.AgentResult) (bool, string) {
	if result == nil || result.Output == "" {
		return false, "no output from agent"
	}

	// Parse the iteration report from the agent's output
	report, err := ParseIterationReport(result.Output)
	if err != nil {
		// If we can't parse the report, check if the output indicates success
		output := strings.ToLower(result.Output)
		if strings.Contains(output, "deployment is healthy") ||
			strings.Contains(output, "fix successful") ||
			strings.Contains(output, "verification passed") {
			return false, "success indicated in output"
		}
		// Continue if we can't determine status
		return true, "could not parse iteration report"
	}

	switch report.Outcome {
	case "success":
		return false, "fix successful"
	case "gave_up":
		return false, "agent determined issue cannot be fixed"
	case "continue":
		return true, "agent requests continuation"
	default:
		// Unknown outcome, check verification result
		if report.VerificationResult == "pass" {
			return false, "verification passed"
		}
		return true, "continuing due to unknown outcome"
	}
}

// IterationReport represents the structured report from a fix iteration.
type IterationReport struct {
	Diagnosis          string   `json:"diagnosis"`
	ChangesMade        []string `json:"changes_made"`
	DeployTriggered    bool     `json:"deploy_triggered"`
	VerificationResult string   `json:"verification_result"` // pass, fail, skip
	Outcome            string   `json:"outcome"`             // success, continue, gave_up
	Notes              string   `json:"notes"`
}

// ParseIterationReport extracts the iteration report from agent output.
func ParseIterationReport(output string) (*IterationReport, error) {
	// Look for the JSON block in the output
	// The agent is instructed to output: {"iteration_report": {...}}

	// Find JSON in the output
	startIdx := strings.Index(output, `{"iteration_report"`)
	if startIdx == -1 {
		// Try without the wrapper
		startIdx = strings.Index(output, `"iteration_report"`)
		if startIdx == -1 {
			return nil, nil
		}
		// Find the opening brace before it
		for i := startIdx - 1; i >= 0; i-- {
			if output[i] == '{' {
				startIdx = i
				break
			}
		}
	}

	// Find the matching closing brace
	braceCount := 0
	endIdx := startIdx
	for i := startIdx; i < len(output); i++ {
		if output[i] == '{' {
			braceCount++
		} else if output[i] == '}' {
			braceCount--
			if braceCount == 0 {
				endIdx = i + 1
				break
			}
		}
	}

	if endIdx <= startIdx {
		return nil, nil
	}

	jsonStr := output[startIdx:endIdx]

	// Parse the wrapper
	var wrapper struct {
		IterationReport IterationReport `json:"iteration_report"`
	}
	if err := json.Unmarshal([]byte(jsonStr), &wrapper); err != nil {
		// Try parsing as direct report
		var report IterationReport
		if err := json.Unmarshal([]byte(jsonStr), &report); err != nil {
			return nil, err
		}
		return &report, nil
	}

	return &wrapper.IterationReport, nil
}

// ExtractIterationRecord converts an agent result to an iteration record.
func ExtractIterationRecord(iteration int, result *shared.AgentResult) domain.FixIterationRecord {
	record := domain.FixIterationRecord{
		Number:     iteration,
		AgentRunID: result.RunID,
	}

	report, err := ParseIterationReport(result.Output)
	if err == nil && report != nil {
		record.DiagnosisSummary = report.Diagnosis
		if len(report.ChangesMade) > 0 {
			record.ChangesSummary = strings.Join(report.ChangesMade, "; ")
		}
		record.DeployTriggered = report.DeployTriggered
		record.VerifyResult = report.VerificationResult
		record.Outcome = report.Outcome
	} else {
		// Couldn't parse report, use defaults
		record.Outcome = "continue"
		record.VerifyResult = "skip"
	}

	return record
}
