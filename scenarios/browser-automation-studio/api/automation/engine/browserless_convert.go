package engine

import (
	"encoding/base64"
	"encoding/json"
	"time"

	"github.com/google/uuid"
	"github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// buildStepOutcomeFromRuntime normalizes a runtime.ExecutionResponse into a
// StepOutcome that matches the automation contracts. This enables legacy
// Browserless orchestration to reuse the recorder/event sink path.
func buildStepOutcomeFromRuntime(executionID uuid.UUID, instruction contracts.CompiledInstruction, start time.Time, resp *runtime.ExecutionResponse) (contracts.StepOutcome, error) {
	if resp == nil || len(resp.Steps) == 0 {
		return contracts.StepOutcome{
			Success: false,
			Failure: &contracts.StepFailure{
				Kind:    contracts.FailureKindEngine,
				Message: "no step result returned",
				Source:  contracts.FailureSourceEngine,
			},
		}, nil
	}

	step := resp.Steps[0]
	outcome := contracts.StepOutcome{
		SchemaVersion:  contracts.StepOutcomeSchemaVersion,
		PayloadVersion: contracts.PayloadVersion,
		ExecutionID:    executionID,
		StepIndex:      step.Index,
		NodeID:         step.NodeID,
		StepType:       step.Type,
		Success:        step.Success,
		StartedAt:      start,
		DurationMs:     step.DurationMs,
		FinalURL:       step.FinalURL,
		ExtractedData: map[string]any{
			"value": step.ExtractedData,
		},
		Assertion:          convertAssertion(step.Assertion),
		Condition:          convertCondition(step.ConditionResult),
		ProbeResult:        step.ProbeResult,
		ElementBoundingBox: convertBoundingBox(step.ElementBoundingBox),
		ClickPosition:      convertPoint(step.ClickPosition),
		FocusedElement:     convertFocus(step.FocusedElement),
		HighlightRegions:   convertHighlights(step.HighlightRegions),
		MaskRegions:        convertMasks(step.MaskRegions),
		ZoomFactor:         step.ZoomFactor,
		CompletedAt: func() *time.Time {
			t := time.Now().UTC()
			return &t
		}(),
	}

	if !step.Success && step.Error != "" {
		outcome.Failure = &contracts.StepFailure{
			Kind:      contracts.FailureKindEngine,
			Message:   step.Error,
			Retryable: true,
			Source:    contracts.FailureSourceEngine,
		}
	}

	if len(step.ConsoleLogs) > 0 {
		outcome.ConsoleLogs = convertConsoleLogs(step.ConsoleLogs)
	}
	if len(step.NetworkEvents) > 0 {
		outcome.Network = convertNetwork(step.NetworkEvents)
	}
	if step.DOMSnapshot != "" {
		outcome.DOMSnapshot = &contracts.DOMSnapshot{
			HTML:        step.DOMSnapshot,
			CollectedAt: time.Now().UTC(),
		}
	}
	if step.ScreenshotBase64 != "" {
		data, err := base64.StdEncoding.DecodeString(step.ScreenshotBase64)
		if err == nil {
			outcome.Screenshot = &contracts.Screenshot{
				Data:        data,
				MediaType:   "image/png",
				CaptureTime: time.Now().UTC(),
			}
		}
	}

	return outcome, nil
}

func toRuntimeInstruction(instr contracts.CompiledInstruction) (runtime.Instruction, error) {
	rt := runtime.Instruction{
		Index:       instr.Index,
		NodeID:      instr.NodeID,
		Type:        instr.Type,
		PreloadHTML: instr.PreloadHTML,
	}
	raw, err := json.Marshal(instr.Params)
	if err != nil {
		return runtime.Instruction{}, err
	}
	var params runtime.InstructionParam
	if err := json.Unmarshal(raw, &params); err != nil {
		return runtime.Instruction{}, err
	}
	rt.Params = params
	return rt, nil
}

func convertConsoleLogs(logs []runtime.ConsoleLog) []contracts.ConsoleLogEntry {
	out := make([]contracts.ConsoleLogEntry, 0, len(logs))
	for _, l := range logs {
		out = append(out, contracts.ConsoleLogEntry{
			Type:      l.Type,
			Text:      l.Text,
			Timestamp: time.UnixMilli(l.Timestamp).UTC(),
		})
	}
	return out
}

func convertNetwork(events []runtime.NetworkEvent) []contracts.NetworkEvent {
	out := make([]contracts.NetworkEvent, 0, len(events))
	for _, ev := range events {
		out = append(out, contracts.NetworkEvent{
			Type:         ev.Type,
			URL:          ev.URL,
			Method:       ev.Method,
			ResourceType: ev.ResourceType,
			Status:       ev.Status,
			OK:           ev.OK,
			Failure:      ev.Failure,
			Timestamp:    time.UnixMilli(ev.Timestamp).UTC(),
		})
	}
	return out
}

func convertAssertion(a *runtime.AssertionResult) *contracts.AssertionOutcome {
	if a == nil {
		return nil
	}
	return &contracts.AssertionOutcome{
		Mode:          a.Mode,
		Selector:      a.Selector,
		Expected:      a.Expected,
		Actual:        a.Actual,
		Success:       a.Success,
		Negated:       a.Negated,
		CaseSensitive: a.CaseSensitive,
		Message:       a.Message,
	}
}

func convertCondition(c *runtime.ConditionResult) *contracts.ConditionOutcome {
	if c == nil {
		return nil
	}
	return &contracts.ConditionOutcome{
		Type:       c.Type,
		Outcome:    c.Outcome,
		Negated:    c.Negated,
		Operator:   c.Operator,
		Variable:   c.Variable,
		Selector:   c.Selector,
		Expression: c.Expression,
		Actual:     c.Actual,
		Expected:   c.Expected,
	}
}

func convertBoundingBox(b *runtime.BoundingBox) *contracts.BoundingBox {
	if b == nil {
		return nil
	}
	return &contracts.BoundingBox{
		X:      b.X,
		Y:      b.Y,
		Width:  b.Width,
		Height: b.Height,
	}
}

func convertPoint(p *runtime.Point) *contracts.Point {
	if p == nil {
		return nil
	}
	return &contracts.Point{X: p.X, Y: p.Y}
}

func convertFocus(f *runtime.ElementFocus) *contracts.ElementFocus {
	if f == nil {
		return nil
	}
	return &contracts.ElementFocus{
		Selector:    f.Selector,
		BoundingBox: convertBoundingBox(f.BoundingBox),
	}
}

func convertHighlights(regions []runtime.HighlightRegion) []contracts.HighlightRegion {
	out := make([]contracts.HighlightRegion, 0, len(regions))
	for _, r := range regions {
		out = append(out, contracts.HighlightRegion{
			Selector:    r.Selector,
			BoundingBox: convertBoundingBox(r.BoundingBox),
			Padding:     r.Padding,
			Color:       r.Color,
		})
	}
	return out
}

func convertMasks(regions []runtime.MaskRegion) []contracts.MaskRegion {
	out := make([]contracts.MaskRegion, 0, len(regions))
	for _, r := range regions {
		out = append(out, contracts.MaskRegion{
			Selector:    r.Selector,
			BoundingBox: convertBoundingBox(r.BoundingBox),
			Opacity:     r.Opacity,
		})
	}
	return out
}
