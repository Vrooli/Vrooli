package cdp

import (
	"context"
	"fmt"
	"math"
	"strings"
	"time"

	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

// ExecuteGesture simulates common mobile/touch gestures. [REQ:BAS-NODE-GESTURE-MOBILE]
func (s *Session) ExecuteGesture(ctx context.Context, opts gestureOptions) (*StepResult, error) {
	start := time.Now()
	gestureType := strings.ToLower(strings.TrimSpace(opts.Type))
	if gestureType == "" {
		gestureType = "tap"
	}
	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = defaultGestureTimeoutMs
	}
	if opts.DurationMs <= 0 {
		opts.DurationMs = defaultGestureDurationMs
	}
	if opts.Steps <= 0 {
		opts.Steps = defaultGestureSteps
	}
	if opts.Distance <= 0 {
		opts.Distance = defaultGestureDistance
	}
	if opts.Scale <= 0 {
		opts.Scale = 1
	}

	result := &StepResult{DebugContext: map[string]interface{}{
		"type":      gestureType,
		"direction": opts.Direction,
		"selector":  strings.TrimSpace(opts.Selector),
		"distance":  opts.Distance,
		"scale":     opts.Scale,
		"steps":     opts.Steps,
		"duration":  opts.DurationMs,
	}}

	timeoutCtx, cancel := context.WithTimeout(s.GetCurrentContext(), time.Duration(opts.TimeoutMs)*time.Millisecond)
	defer cancel()

	anchorX, anchorY, err := s.resolveGestureAnchor(timeoutCtx, opts)
	if err != nil {
		result.Error = fmt.Sprintf("failed to resolve gesture anchor: %v", err)
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}
	result.DebugContext["anchorX"] = anchorX
	result.DebugContext["anchorY"] = anchorY

	var execErr error
	switch gestureType {
	case "swipe":
		startX, startY, endX, endY := s.resolveSwipePath(anchorX, anchorY, opts)
		result.DebugContext["startX"] = startX
		result.DebugContext["startY"] = startY
		result.DebugContext["endX"] = endX
		result.DebugContext["endY"] = endY
		execErr = s.dispatchSwipe(timeoutCtx, startX, startY, endX, endY, opts.DurationMs, opts.Steps)
	case "pinch":
		execErr = s.dispatchPinch(timeoutCtx, anchorX, anchorY, opts)
	case "doubletap", "double_tap":
		execErr = s.dispatchDoubleTap(timeoutCtx, anchorX, anchorY, opts.DurationMs)
	case "longpress", "long_press", "long-press", "press":
		execErr = s.dispatchLongPress(timeoutCtx, anchorX, anchorY, opts.HoldMs)
	default:
		execErr = s.dispatchTap(timeoutCtx, anchorX, anchorY, opts.DurationMs)
	}

	if execErr != nil {
		result.Error = execErr.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, execErr
	}

	if err := chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	); err != nil {
		s.log.WithError(err).Debug("failed to capture page metadata after gesture")
	}

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func (s *Session) resolveGestureAnchor(ctx context.Context, opts gestureOptions) (float64, float64, error) {
	trimmed := strings.TrimSpace(opts.Selector)
	if trimmed != "" {
		box, err := s.resolveElementBox(ctx, trimmed)
		if err != nil {
			return 0, 0, err
		}
		x, y, err := centerFromBoxModel(box)
		if err != nil {
			return 0, 0, err
		}
		return x, y, nil
	}

	centerX, centerY := s.viewportCenter()
	if opts.HasExplicitStart {
		if opts.HasStartX {
			centerX = opts.StartX
		}
		if opts.HasStartY {
			centerY = opts.StartY
		}
	}
	return s.clampViewportX(centerX), s.clampViewportY(centerY), nil
}

func (s *Session) resolveSwipePath(anchorX, anchorY float64, opts gestureOptions) (float64, float64, float64, float64) {
	startX := anchorX
	startY := anchorY
	if opts.HasExplicitStart {
		if opts.HasStartX {
			startX = opts.StartX
		}
		if opts.HasStartY {
			startY = opts.StartY
		}
	}
	startX = s.clampViewportX(startX)
	startY = s.clampViewportY(startY)

	endX := startX
	endY := startY
	if opts.HasExplicitEnd {
		if opts.HasEndX {
			endX = opts.EndX
		}
		if opts.HasEndY {
			endY = opts.EndY
		}
	} else {
		delta := float64(opts.Distance)
		if delta <= 0 {
			delta = float64(defaultGestureDistance)
		}
		switch strings.ToLower(strings.TrimSpace(opts.Direction)) {
		case "up":
			endY = startY - delta
		case "left":
			endX = startX - delta
		case "right":
			endX = startX + delta
		default:
			endY = startY + delta
		}
	}

	return startX, startY, s.clampViewportX(endX), s.clampViewportY(endY)
}

func (s *Session) dispatchTap(ctx context.Context, x, y float64, holdMs int) error {
	if holdMs <= 0 {
		holdMs = minGestureDurationMs
	}
	point := s.touchPoint(1, x, y)
	if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{point}).Do(ctx); err != nil {
		return err
	}
	if err := waitWithContext(ctx, time.Duration(holdMs)*time.Millisecond); err != nil {
		return err
	}
	return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
}

func (s *Session) dispatchDoubleTap(ctx context.Context, x, y float64, durationMs int) error {
	if err := s.dispatchTap(ctx, x, y, durationMs); err != nil {
		return err
	}
	if err := waitWithContext(ctx, 120*time.Millisecond); err != nil {
		return err
	}
	return s.dispatchTap(ctx, x, y, durationMs)
}

func (s *Session) dispatchLongPress(ctx context.Context, x, y float64, holdMs int) error {
	if holdMs < 600 {
		holdMs = 600
	}
	point := s.touchPoint(1, x, y)
	if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{point}).Do(ctx); err != nil {
		return err
	}
	if err := waitWithContext(ctx, time.Duration(holdMs)*time.Millisecond); err != nil {
		return err
	}
	return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
}

func (s *Session) dispatchSwipe(ctx context.Context, startX, startY, endX, endY float64, durationMs, steps int) error {
	if steps < 1 {
		steps = 1
	}
	point := s.touchPoint(1, startX, startY)
	if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{point}).Do(ctx); err != nil {
		return err
	}

	stepDelay := time.Duration(durationMs) * time.Millisecond / time.Duration(steps)
	if durationMs <= 0 {
		stepDelay = 0
	}

	for i := 1; i <= steps; i++ {
		progress := float64(i) / float64(steps)
		point.X = s.clampViewportX(startX + (endX-startX)*progress)
		point.Y = s.clampViewportY(startY + (endY-startY)*progress)
		if err := input.DispatchTouchEvent(input.TouchMove, []*input.TouchPoint{point}).Do(ctx); err != nil {
			return err
		}
		if err := waitWithContext(ctx, stepDelay); err != nil {
			return err
		}
	}

	return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
}

func (s *Session) dispatchPinch(ctx context.Context, centerX, centerY float64, opts gestureOptions) error {
	steps := opts.Steps
	if steps < minGestureSteps {
		steps = minGestureSteps
	}
	baseRadius := math.Max(float64(opts.Distance)/2, 20)
	targetRadius := baseRadius * opts.Scale

	pointA := s.touchPoint(1, centerX-baseRadius, centerY)
	pointB := s.touchPoint(2, centerX+baseRadius, centerY)
	if err := input.DispatchTouchEvent(input.TouchStart, []*input.TouchPoint{pointA, pointB}).Do(ctx); err != nil {
		return err
	}

	stepDelay := time.Duration(opts.DurationMs) * time.Millisecond / time.Duration(steps)
	if opts.DurationMs <= 0 {
		stepDelay = 0
	}

	for i := 1; i <= steps; i++ {
		progress := float64(i) / float64(steps)
		radius := baseRadius + (targetRadius-baseRadius)*progress
		pointA.X = s.clampViewportX(centerX - radius)
		pointA.Y = s.clampViewportY(centerY)
		pointB.X = s.clampViewportX(centerX + radius)
		pointB.Y = s.clampViewportY(centerY)
		if err := input.DispatchTouchEvent(input.TouchMove, []*input.TouchPoint{pointA, pointB}).Do(ctx); err != nil {
			return err
		}
		if err := waitWithContext(ctx, stepDelay); err != nil {
			return err
		}
	}

	return input.DispatchTouchEvent(input.TouchEnd, []*input.TouchPoint{}).Do(ctx)
}

func (s *Session) touchPoint(id float64, x, y float64) *input.TouchPoint {
	return &input.TouchPoint{
		ID:      id,
		X:       s.clampViewportX(x),
		Y:       s.clampViewportY(y),
		RadiusX: 8,
		RadiusY: 8,
		Force:   1,
	}
}

func (s *Session) viewportCenter() (float64, float64) {
	width := float64(s.viewportWidth)
	height := float64(s.viewportHeight)
	if width <= 0 {
		width = 1920
	}
	if height <= 0 {
		height = 1080
	}
	return width / 2, height / 2
}

func (s *Session) clampViewportX(value float64) float64 {
	if value < 0 {
		return 0
	}
	maxX := float64(s.viewportWidth)
	if maxX <= 0 {
		maxX = 1920
	}
	if value > maxX {
		return maxX
	}
	return value
}

func (s *Session) clampViewportY(value float64) float64 {
	if value < 0 {
		return 0
	}
	maxY := float64(s.viewportHeight)
	if maxY <= 0 {
		maxY = 1080
	}
	if value > maxY {
		return maxY
	}
	return value
}

func (s *Session) pointerStartPosition(targetX, targetY float64) (float64, float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if !s.pointerInitialized {
		s.pointerX = targetX
		s.pointerY = targetY
		s.pointerInitialized = true
		return targetX, targetY
	}
	return s.pointerX, s.pointerY
}

func (s *Session) setPointerPosition(x, y float64) {
	s.mu.Lock()
	s.pointerX = x
	s.pointerY = y
	s.pointerInitialized = true
	s.mu.Unlock()
}
