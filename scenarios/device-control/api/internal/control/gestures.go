package control

import (
	"context"
	"fmt"
	"strings"

	"device-control/strategy"
)

func pixelStep(step Step) bool {
	space, _ := step.Arguments["coordinate_space"].(string)
	return strings.EqualFold(space, "pixel")
}

func gesturePoint(step Step, prefix string) (float64, float64, error) {
	x, xok := number(step.Arguments[prefix+"x"])
	y, yok := number(step.Arguments[prefix+"y"])
	if prefix == "" {
		x, xok = number(step.Arguments["x"])
		y, yok = number(step.Arguments["y"])
	}
	if !xok || !yok || x < 0 || x > 1 || y < 0 || y > 1 {
		return 0, 0, fmt.Errorf("gesture %s requires normalized %s coordinates", step.ID, prefix)
	}
	return x, y, nil
}

func gestureDuration(step Step, fallback int) int {
	if value, ok := number(step.Arguments["duration_ms"]); ok && value > 0 {
		return int(value)
	}
	return fallback
}

func executeGesture(ctx context.Context, dispatch func(context.Context, strategy.Actuation) error, step Step) error {
	x0, y0, err := gesturePoint(step, "start_")
	if step.Kind == "long-press" || step.Kind == "double-tap" {
		x0, y0, err = gesturePoint(step, "")
	}
	if err != nil {
		return err
	}
	x1, y1 := x0, y0
	if step.Kind == "drag" || step.Kind == "fling" || step.Kind == "swipe" {
		x1, y1, err = gesturePoint(step, "end_")
		if err != nil {
			return err
		}
	}
	duration := gestureDuration(step, 300)
	if step.Kind == "long-press" {
		duration = gestureDuration(step, 800)
	}
	if step.Kind == "double-tap" {
		for i := 0; i < 2; i++ {
			if err := dispatch(ctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: "tap", X: x0, Y: y0, Normalized: true}}); err != nil {
				return err
			}
		}
		return nil
	}
	return dispatch(ctx, strategy.Actuation{Pointer: &strategy.PointerEvent{Kind: step.Kind, X: x0, Y: y0, Button: fmt.Sprintf("%f,%f", x1, y1), Normalized: true, DurationMS: duration}})
}

func executePinch(ctx context.Context, dispatch func(context.Context, strategy.Actuation) error, step Step) error {
	x0, y0, err := gesturePoint(step, "start_")
	if err != nil {
		return err
	}
	x1, y1, err := gesturePoint(step, "end_")
	if err != nil {
		return err
	}
	return dispatch(ctx, strategy.Actuation{Action: "pinch", Value: fmt.Sprintf("%f,%f,%f,%f", x0, y0, x1, y1)})
}

func executeScrollTo(ctx context.Context, dispatch func(context.Context, strategy.Actuation) error, step Step) error {
	// Target re-resolution is owned by the flow layer; this bounded adapter
	// action only performs one scroll attempt.
	return dispatch(ctx, strategy.Actuation{Action: "scroll-to", Value: step.Target})
}
