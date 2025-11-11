package cdp

import (
	"context"
	"fmt"
	"strings"

	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// ExecuteSetVariable resolves a setVariable instruction by delegating to the correct primitive.
func (s *Session) ExecuteSetVariable(ctx context.Context, instruction runtime.Instruction) (*StepResult, error) {
	source := strings.ToLower(strings.TrimSpace(instruction.Params.VariableSource))
	timeout := instruction.Params.TimeoutMs
	if timeout <= 0 {
		timeout = defaultVariableTimeoutMs
	}

	switch source {
	case "static":
		return &StepResult{Success: true, ExtractedData: instruction.Params.VariableValue}, nil
	case "expression":
		return s.ExecuteEvaluate(ctx, instruction.Params.Expression, timeout)
	case "extract":
		return s.ExecuteExtract(ctx, instruction.Params.Selector, instruction.Params.ExtractType, instruction.Params.Attribute, boolFromPointer(instruction.Params.AllMatches), timeout)
	default:
		return nil, fmt.Errorf("unsupported variable source %s", source)
	}
}
