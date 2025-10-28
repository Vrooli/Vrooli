package runtime

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/browserless/compiler"
)

// Instruction represents a normalized execution step that can be shipped to Browserless.
type Instruction struct {
	Index  int              `json:"index"`
	NodeID string           `json:"nodeId"`
	Type   string           `json:"type"`
	Params InstructionParam `json:"params"`
}

// InstructionParam captures the parameter payload for a Browserless instruction.
type InstructionParam struct {
	URL            string `json:"url,omitempty"`
	WaitUntil      string `json:"waitUntil,omitempty"`
	TimeoutMs      int    `json:"timeoutMs,omitempty"`
	WaitForMs      int    `json:"waitForMs,omitempty"`
	WaitType       string `json:"waitType,omitempty"`
	DurationMs     int    `json:"durationMs,omitempty"`
	Selector       string `json:"selector,omitempty"`
	Name           string `json:"name,omitempty"`
	FullPage       *bool  `json:"fullPage,omitempty"`
	ViewportWidth  int    `json:"viewportWidth,omitempty"`
	ViewportHeight int    `json:"viewportHeight,omitempty"`
}

// InstructionsFromPlan converts a compiled execution plan into Browserless instructions.
func InstructionsFromPlan(plan *compiler.ExecutionPlan) ([]Instruction, error) {
	if plan == nil {
		return nil, fmt.Errorf("execution plan is nil")
	}

	instructions := make([]Instruction, 0, len(plan.Steps))
	for _, step := range plan.Steps {
		instr, err := instructionFromStep(step)
		if err != nil {
			return nil, err
		}
		instructions = append(instructions, instr)
	}

	return instructions, nil
}

type navigateConfig struct {
	URL       string `json:"url"`
	WaitUntil string `json:"waitUntil"`
	TimeoutMs int    `json:"timeoutMs"`
	WaitForMs int    `json:"waitForMs"`
}

type waitConfig struct {
	Type      string `json:"type"`
	Duration  int    `json:"duration"`
	Selector  string `json:"selector"`
	TimeoutMs int    `json:"timeoutMs"`
}

type screenshotConfig struct {
	Name           string `json:"name"`
	FullPage       *bool  `json:"fullPage"`
	ViewportWidth  int    `json:"viewportWidth"`
	ViewportHeight int    `json:"viewportHeight"`
	WaitForMs      int    `json:"waitForMs"`
}

func instructionFromStep(step compiler.ExecutionStep) (Instruction, error) {
	base := Instruction{
		Index:  step.Index,
		NodeID: step.NodeID,
		Type:   string(step.Type),
		Params: InstructionParam{},
	}

	switch step.Type {
	case compiler.StepNavigate:
		var cfg navigateConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("navigate node %s has invalid data: %w", step.NodeID, err)
		}
		if strings.TrimSpace(cfg.URL) == "" {
			return Instruction{}, fmt.Errorf("navigate node %s missing url", step.NodeID)
		}
		base.Params.URL = cfg.URL
		if wait := strings.TrimSpace(cfg.WaitUntil); wait != "" {
			base.Params.WaitUntil = wait
		}
		if cfg.TimeoutMs > 0 {
			base.Params.TimeoutMs = cfg.TimeoutMs
		}
		if cfg.WaitForMs > 0 {
			base.Params.WaitForMs = cfg.WaitForMs
		}
	case compiler.StepWait:
		var cfg waitConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("wait node %s has invalid data: %w", step.NodeID, err)
		}
		waitType := strings.ToLower(strings.TrimSpace(cfg.Type))
		if waitType == "" || waitType == "time" {
			base.Params.WaitType = "time"
			if cfg.Duration > 0 {
				base.Params.DurationMs = cfg.Duration
			} else {
				base.Params.DurationMs = 1000
			}
		} else if waitType == "element" {
			base.Params.WaitType = "element"
			base.Params.Selector = cfg.Selector
			base.Params.TimeoutMs = cfg.TimeoutMs
			if strings.TrimSpace(base.Params.Selector) == "" {
				return Instruction{}, fmt.Errorf("wait node %s requires selector for element wait", step.NodeID)
			}
		} else {
			return Instruction{}, fmt.Errorf("wait node %s has unsupported type %q", step.NodeID, cfg.Type)
		}
	case compiler.StepScreenshot:
		var cfg screenshotConfig
		if err := decodeParams(step.Params, &cfg); err != nil {
			return Instruction{}, fmt.Errorf("screenshot node %s has invalid data: %w", step.NodeID, err)
		}
		base.Params.Name = cfg.Name
		base.Params.ViewportWidth = cfg.ViewportWidth
		base.Params.ViewportHeight = cfg.ViewportHeight
		base.Params.WaitForMs = cfg.WaitForMs
		base.Params.FullPage = cfg.FullPage
	default:
		return Instruction{}, fmt.Errorf("unsupported step type %q in execution plan", step.Type)
	}

	return base, nil
}

func decodeParams(src map[string]any, target interface{}) error {
	raw, err := json.Marshal(src)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}
