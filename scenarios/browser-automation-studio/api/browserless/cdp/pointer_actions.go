package cdp

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
	"github.com/vrooli/browser-automation-studio/browserless/runtime"
)

// ExecuteHover hovers over an element to trigger pointer interactions.
func (s *Session) ExecuteHover(ctx context.Context, selector string, timeoutMs, waitAfterMs, steps, durationMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if steps < 1 {
		steps = 1
	} else if steps > 50 {
		steps = 50
	}
	if durationMs < 0 {
		durationMs = 0
	} else if durationMs > 10000 {
		durationMs = 10000
	}

	timeoutCtx, cancel := context.WithTimeout(s.GetCurrentContext(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()

	box, err := s.resolveElementBox(timeoutCtx, selector)
	if err != nil {
		result.Error = fmt.Sprintf("hover target %s unavailable: %v", selector, err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	centerX, centerY, err := centerFromBoxModel(box)
	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	err = chromedp.Run(timeoutCtx,
		chromedp.ActionFunc(func(ctx context.Context) error {
			return s.movePointerSmooth(ctx, centerX, centerY, steps, durationMs)
		}),
	)
	if err != nil {
		result.Error = fmt.Sprintf("hover movement failed: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if waitAfterMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(waitAfterMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("hover wait interrupted: %v", err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

// ExecuteDragAndDrop simulates HTML5 drag interactions, including pointer motion and drop events.
func (s *Session) ExecuteDragAndDrop(ctx context.Context, opts dragDropOptions, timeoutMs int) (*StepResult, error) {
	start := time.Now()
	result := &StepResult{DebugContext: map[string]interface{}{
		"sourceSelector": opts.sourceSelector,
		"targetSelector": opts.targetSelector,
		"holdMs":         opts.holdMs,
		"steps":          opts.steps,
		"durationMs":     opts.durationMs,
		"offsetX":        opts.offsetX,
		"offsetY":        opts.offsetY,
	}}

	if timeoutMs <= 0 {
		timeoutMs = 30000
	}
	if opts.steps < 1 {
		opts.steps = 1
	}
	if opts.durationMs < 0 {
		opts.durationMs = 0
	}
	if opts.holdMs < 0 {
		opts.holdMs = 0
	}

	baseCtx := s.BeginOperation()
	defer s.EndOperation()
	s.log.WithField("baseCtx_err", baseCtx.Err()).Info("[DRAG] Got base context via BeginOperation")

	timeoutCtx, cancel := context.WithTimeout(baseCtx, time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	s.log.WithField("timeoutCtx_err", timeoutCtx.Err()).Info("[DRAG] Created timeout context")

	sourceBox, err := s.resolveElementBox(timeoutCtx, opts.sourceSelector)
	s.log.WithField("timeoutCtx_err_after_resolve", timeoutCtx.Err()).WithField("err", err).Info("[DRAG] Resolved source element box")
	if err != nil {
		result.Error = fmt.Sprintf("drag source %s unavailable: %v", opts.sourceSelector, err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	targetBox, err := s.resolveElementBox(timeoutCtx, opts.targetSelector)
	if err != nil {
		result.Error = fmt.Sprintf("drag target %s unavailable: %v", opts.targetSelector, err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	sourceX, sourceY, err := centerFromBoxModel(sourceBox)
	if err != nil {
		result.Error = fmt.Sprintf("unable to compute drag source center: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	targetX, targetY, err := centerFromBoxModel(targetBox)
	if err != nil {
		result.Error = fmt.Sprintf("unable to compute drag target center: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	dropX := targetX + float64(opts.offsetX)
	dropY := targetY + float64(opts.offsetY)
	result.DebugContext["dropX"] = dropX
	result.DebugContext["dropY"] = dropY

	approachSteps := opts.steps // default to steps
	if approachSteps < 1 {
		approachSteps = 1
	}
	approachDuration := opts.durationMs / 3
	s.log.WithField("timeoutCtx_err_before_move", timeoutCtx.Err()).Info("[DRAG] About to call movePointerSmooth")
	if err := s.movePointerSmooth(timeoutCtx, sourceX, sourceY, approachSteps, approachDuration); err != nil {
		s.log.WithField("timeoutCtx_err_after_move_fail", timeoutCtx.Err()).WithError(err).Error("[DRAG] movePointerSmooth failed")
		result.Error = fmt.Sprintf("failed to move pointer to source: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	// Use chromedp.MouseEvent wrapper to ensure proper context metadata
	if err := chromedp.Run(timeoutCtx, chromedp.MouseEvent(input.MousePressed, sourceX, sourceY, func(p *input.DispatchMouseEventParams) *input.DispatchMouseEventParams {
		return p.WithButton(input.Left).WithButtons(1).WithClickCount(1)
	})); err != nil {
		result.Error = fmt.Sprintf("failed to press mouse: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if opts.holdMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(opts.holdMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("drag hold interrupted: %v", err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
	}

	if err := s.movePointerSmoothWithButtons(timeoutCtx, dropX, dropY, opts.steps, opts.durationMs, 1); err != nil {
		result.Error = fmt.Sprintf("failed to drag pointer: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	// Use chromedp.MouseEvent wrapper to ensure proper context metadata
	if err := chromedp.Run(timeoutCtx, chromedp.MouseEvent(input.MouseReleased, dropX, dropY, func(p *input.DispatchMouseEventParams) *input.DispatchMouseEventParams {
		return p.WithButton(input.Left).WithButtons(0).WithClickCount(1)
	})); err != nil {
		result.Error = fmt.Sprintf("failed to release mouse: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	dragEventScript := fmt.Sprintf(`(() => {
        const sourceSelector = %s;
        const targetSelector = %s;
        const dropPoint = { x: %f, y: %f };
        const source = document.querySelector(sourceSelector);
        const target = document.querySelector(targetSelector);
        if (!source || !target) {
            throw new Error('Drag selectors missing during HTML5 dispatch');
        }
        const dataTransfer = typeof DataTransfer === 'function' ? new DataTransfer() : {
            _data: {},
            setData(type, value) { this._data[type] = String(value); },
            getData(type) { return this._data[type] ?? ''; },
            dropEffect: 'move',
            effectAllowed: 'all',
        };
        const fire = (node, type) => {
            const eventInit = { bubbles: true, cancelable: true, dataTransfer, clientX: dropPoint.x, clientY: dropPoint.y };
            let event;
            try {
                event = new DragEvent(type, eventInit);
            } catch (err) {
                event = document.createEvent('CustomEvent');
                event.initCustomEvent(type, true, true, null);
                Object.assign(event, eventInit);
            }
            node.dispatchEvent(event);
        };
        fire(source, 'dragstart');
        fire(source, 'drag');
        fire(target, 'dragenter');
        fire(target, 'dragover');
        fire(target, 'drop');
        fire(source, 'dragend');
        return true;
    })()`, strconv.Quote(opts.sourceSelector), strconv.Quote(opts.targetSelector), dropX, dropY)

	if err := s.evalWithFrame(timeoutCtx, dragEventScript, nil); err != nil {
		result.Error = fmt.Sprintf("failed to dispatch HTML5 drag events: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	result.DebugContext["html5Dispatch"] = true

	if targetBox != nil {
		result.ElementBoundingBox = &runtime.BoundingBox{
			X:      dropX - float64(targetBox.Width)/2,
			Y:      dropY - float64(targetBox.Height)/2,
			Width:  float64(targetBox.Width),
			Height: float64(targetBox.Height),
		}
	}

	if opts.waitAfterMs > 0 {
		if err := waitWithContext(timeoutCtx, time.Duration(opts.waitAfterMs)*time.Millisecond); err != nil {
			result.Error = fmt.Sprintf("post-drag wait interrupted: %v", err)
			result.DurationMs = int(time.Since(start).Milliseconds())
			return result, err
		}
	}

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}
