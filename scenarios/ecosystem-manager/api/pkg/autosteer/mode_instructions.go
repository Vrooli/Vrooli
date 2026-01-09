package autosteer

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

// ModeInstructions loads and serves mode-specific guidance from markdown files.
type ModeInstructions struct {
	baseDir string
	cache   map[SteerMode]phasePromptData
	mu      sync.RWMutex
}

type phasePromptData struct {
	Instructions        string
	SuccessCriteria     []string
	ToolRecommendations []string
	Raw                 string
}

// NewModeInstructions creates a new mode instructions provider that reads prompts from disk.
func NewModeInstructions(baseDir string) *ModeInstructions {
	return &ModeInstructions{
		baseDir: baseDir,
		cache:   make(map[SteerMode]phasePromptData),
	}
}

// GetInstructions returns the detailed instructions for a given mode.
func (m *ModeInstructions) GetInstructions(mode SteerMode) string {
	data, err := m.loadPrompt(mode)
	if err != nil {
		return fmt.Sprintf("Phase prompt unavailable for %s: %v", mode, err)
	}
	return data.Instructions
}

// GetToolRecommendations returns recommended tools for a mode.
func (m *ModeInstructions) GetToolRecommendations(mode SteerMode) []string {
	data, err := m.loadPrompt(mode)
	if err != nil {
		return []string{}
	}
	return data.ToolRecommendations
}

// GetSuccessCriteria returns success criteria for a mode.
func (m *ModeInstructions) GetSuccessCriteria(mode SteerMode) []string {
	data, err := m.loadPrompt(mode)
	if err != nil {
		return []string{}
	}
	return data.SuccessCriteria
}

// FormatConditionProgress renders stop conditions with their evaluated status.
// If no evaluator is provided, a new one is created to keep callers simple.
func (m *ModeInstructions) FormatConditionProgress(conditions []StopCondition, metrics MetricsSnapshot, evaluator *ConditionEvaluator) string {
	if len(conditions) == 0 {
		return ""
	}

	eval := evaluator
	if eval == nil {
		eval = NewConditionEvaluator()
	}

	var builder strings.Builder
	for _, condition := range conditions {
		builder.WriteString("- ")
		builder.WriteString(eval.FormatCondition(condition, metrics))
		builder.WriteString("\n")
	}

	return strings.TrimSpace(builder.String())
}

func (m *ModeInstructions) loadPrompt(mode SteerMode) (phasePromptData, error) {
	m.mu.RLock()
	if data, ok := m.cache[mode]; ok {
		m.mu.RUnlock()
		return data, nil
	}
	m.mu.RUnlock()

	if strings.TrimSpace(m.baseDir) == "" {
		return phasePromptData{}, fmt.Errorf("phase prompts directory not configured")
	}

	filePath := filepath.Join(m.baseDir, fmt.Sprintf("%s.md", strings.ToLower(string(mode))))
	content, err := os.ReadFile(filePath)
	if err != nil {
		return phasePromptData{}, fmt.Errorf("failed to read %s: %w", filePath, err)
	}

	data, err := parsePhasePrompt(string(content))
	if err != nil {
		return phasePromptData{}, err
	}

	m.mu.Lock()
	m.cache[mode] = data
	m.mu.Unlock()
	return data, nil
}

func parsePhasePrompt(content string) (phasePromptData, error) {
	scanner := bufio.NewScanner(strings.NewReader(content))
	section := "instructions"
	var instructions []string
	var success []string
	var tools []string

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)
		lower := strings.ToLower(trimmed)

		switch {
		case strings.HasPrefix(lower, "## success criteria"):
			section = "success"
			continue
		case strings.HasPrefix(lower, "## recommended tools"), strings.HasPrefix(lower, "## tools"):
			section = "tools"
			continue
		}

		switch section {
		case "instructions":
			instructions = append(instructions, line)
		case "success":
			if strings.HasPrefix(trimmed, "-") {
				success = append(success, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
			}
		case "tools":
			if strings.HasPrefix(trimmed, "-") {
				tools = append(tools, strings.TrimSpace(strings.TrimPrefix(trimmed, "-")))
			}
		}
	}

	if err := scanner.Err(); err != nil {
		return phasePromptData{}, fmt.Errorf("failed to parse phase prompt: %w", err)
	}

	instructionText := strings.TrimSpace(strings.Join(instructions, "\n"))
	if instructionText == "" {
		return phasePromptData{}, fmt.Errorf("phase prompt missing instructions section")
	}

	return phasePromptData{
		Instructions:        instructionText,
		SuccessCriteria:     success,
		ToolRecommendations: tools,
		Raw:                 content,
	}, nil
}
