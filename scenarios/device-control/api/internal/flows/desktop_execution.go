package flows

import (
	"context"
	"math"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"device-control/internal/execution"
	"github.com/google/uuid"
	desktopv1 "github.com/vrooli/vrooli/packages/proto/gen/go/device-control/v1/desktop"
)

type desktopTextStep struct {
	id, window, field, text string
	assertion               bool
	position                int32
	timeout                 time.Duration
}

// ExecuteDesktopFlow uses one admitted application/session binding. Preflight
// validates the entire flow before any observation or mutation. The caller owns
// Stop and run persistence; no flow content or credentials enter chapter text.
func ExecuteDesktopFlow(ctx context.Context, owner DesktopStepOwner, binding *desktopv1.OwnerObserveRequest, flow execution.Flow) (execution.RunResult, error) {
	return ExecuteDesktopFlowWithID(ctx, owner, binding, flow, uuid.NewString())
}

// ExecuteDesktopFlowWithID requires the caller to hold fresh durable admission
// for runID. Duplicate claims must return their journal record without calling it.
func ExecuteDesktopFlowWithID(ctx context.Context, owner DesktopStepOwner, binding *desktopv1.OwnerObserveRequest, flow execution.Flow, runID string) (execution.RunResult, error) {
	if runID == "" || len(runID) > 120 || strings.ContainsRune(runID, ':') {
		return execution.RunResult{}, ErrInvalidRequest
	}

	if owner == nil || binding == nil || binding.Session == nil || binding.ApplicationId == "" || binding.ApplicationRevision == "" || binding.ProcessId != 0 || len(flow.Steps) == 0 || len(flow.Steps) > 32 || flow.Transport != "desktop" || flow.SuppressActuation || flow.AllowUnredactedCapture || flow.AuthProfileID != "" {
		return execution.RunResult{}, ErrInvalidRequest
	}
	prepared, err := prepareDesktopFlow(flow)
	if err != nil {
		return execution.RunResult{}, err
	}
	result := execution.RunResult{RunID: runID, Disposition: "passed"}
	ctx, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	for index, step := range prepared {
		stepctx, stop := context.WithTimeout(ctx, step.timeout)
		receipt, err := executeDesktopText(stepctx, owner, binding, result.RunID+":"+strconv.Itoa(index), step.window, step.field, step.text, step.position, step.assertion)
		stop()
		chapter := execution.Chapter{ID: step.id, Title: "Desktop semantic text", Disposition: "passed"}
		if err != nil || receipt == nil || receipt.Receipt == nil || receipt.Receipt.Outcome != "applied" {
			chapter.Disposition = "failed"
			chapter.Message = "Desktop step unconfirmed; inspect state before retrying"
			result.Disposition = "failed"
			result.Incomplete = true
			result.DisconnectStep = step.id
			result.Chapters = append(result.Chapters, chapter)
			return result, nil
		}
		result.Chapters = append(result.Chapters, chapter)
		result.Resolutions = append(result.Resolutions, execution.Resolution{Target: step.field, Rung: "semantic", Confidence: 1})
	}
	return result, nil
}

func ValidateDesktopFlow(flow execution.Flow) error { _, err := prepareDesktopFlow(flow); return err }
func prepareDesktopFlow(flow execution.Flow) ([]desktopTextStep, error) {
	if len(flow.Steps) == 0 || len(flow.Steps) > 32 || flow.Transport != "desktop" || flow.SuppressActuation || flow.AllowUnredactedCapture || flow.AuthProfileID != "" {
		return nil, ErrInvalidRequest
	}
	prepared := make([]desktopTextStep, 0, len(flow.Steps))
	seen := map[string]bool{}
	for _, step := range flow.Steps {
		if step.ID == "" || len(step.ID) > 128 || seen[step.ID] || (step.Kind != "desktop-text" && step.Kind != "desktop-text-assert") || step.Target == "" || len(step.Target) > 4096 || step.TimeoutMS < 0 || step.TimeoutMS > 10000 || len(step.RequiredCapabilities) > 0 {
			return nil, ErrInvalidRequest
		}
		seen[step.ID] = true
		for key := range step.Arguments {
			if key != "window" && key != "text" && (key != "position" || step.Kind == "desktop-text-assert") {
				return nil, ErrInvalidRequest
			}
		}
		window, wok := step.Arguments["window"].(string)
		text, tok := step.Arguments["text"].(string)
		if !wok || window == "" || len(window) > 4096 || !tok || (text == "" && step.Kind != "desktop-text-assert") || len(text) > 16*1024 || !utf8.ValidString(text) || strings.ContainsRune(text, 0) {
			return nil, ErrInvalidRequest
		}
		position := float64(0)
		if raw, ok := step.Arguments["position"]; ok {
			switch n := raw.(type) {
			case float64:
				position = n
			case int:
				position = float64(n)
			case int32:
				position = float64(n)
			default:
				return nil, ErrInvalidRequest
			}
		}
		if math.IsNaN(position) || math.IsInf(position, 0) || position < 0 || position > math.MaxInt32 || math.Trunc(position) != position {
			return nil, ErrInvalidRequest
		}
		timeout := 5 * time.Second
		if step.TimeoutMS > 0 {
			timeout = time.Duration(step.TimeoutMS) * time.Millisecond
		}
		prepared = append(prepared, desktopTextStep{id: step.ID, window: window, field: step.Target, text: text, assertion: step.Kind == "desktop-text-assert", position: int32(position), timeout: timeout})
	}
	return prepared, nil
}
