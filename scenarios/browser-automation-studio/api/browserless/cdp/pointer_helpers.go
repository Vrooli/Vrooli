package cdp

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/input"
	"github.com/chromedp/chromedp"
)

func (s *Session) movePointerSmooth(ctx context.Context, targetX, targetY float64, steps, durationMs int) error {
	return s.movePointerSmoothWithButtons(ctx, targetX, targetY, steps, durationMs, 0)
}

func (s *Session) movePointerSmoothWithButtons(ctx context.Context, targetX, targetY float64, steps, durationMs int, buttons int64) error {
	startX, startY := s.pointerStartPosition(targetX, targetY)
	if steps < 1 {
		steps = 1
	}
	deltaX := (targetX - startX) / float64(steps)
	deltaY := (targetY - startY) / float64(steps)

	var stepDelay time.Duration
	if durationMs > 0 {
		stepDelay = time.Duration(durationMs) * time.Millisecond / time.Duration(steps)
	}

	for i := 1; i <= steps; i++ {
		nextX := startX + deltaX*float64(i)
		nextY := startY + deltaY*float64(i)
		event := input.DispatchMouseEvent(input.MouseMoved, nextX, nextY)
		if buttons > 0 {
			event = event.WithButtons(buttons)
		}
		if err := event.Do(ctx); err != nil {
			return err
		}
		if err := waitWithContext(ctx, stepDelay); err != nil {
			return err
		}
	}

	s.setPointerPosition(targetX, targetY)
	return nil
}

func (s *Session) resolveElementBox(ctx context.Context, selector string) (*dom.BoxModel, error) {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return nil, fmt.Errorf("selector is required")
	}

	var nodes []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.WaitVisible(trimmed, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.ScrollIntoView(trimmed, s.frameQueryOptions(chromedp.ByQuery)...),
		chromedp.Nodes(trimmed, &nodes, s.frameQueryOptions(chromedp.ByQuery)...),
	); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("selector %s not found", trimmed)
	}

	var box *dom.BoxModel
	if err := chromedp.Run(ctx, chromedp.ActionFunc(func(ctx context.Context) error {
		model, err := dom.GetBoxModel().WithNodeID(nodes[0].NodeID).Do(ctx)
		if err != nil {
			return err
		}
		box = model
		return nil
	})); err != nil {
		return nil, err
	}

	return box, nil
}
