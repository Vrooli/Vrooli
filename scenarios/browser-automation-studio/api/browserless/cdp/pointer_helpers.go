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

	s.log.WithFields(map[string]interface{}{
		"ctx_err":    ctx.Err(),
		"steps":      steps,
		"durationMs": durationMs,
		"startX":     startX,
		"startY":     startY,
		"targetX":    targetX,
		"targetY":    targetY,
	}).Info("[POINTER] Starting movePointerSmoothWithButtons")

	for i := 1; i <= steps; i++ {
		nextX := startX + deltaX*float64(i)
		nextY := startY + deltaY*float64(i)

		s.log.WithFields(map[string]interface{}{
			"step":    i,
			"ctx_err": ctx.Err(),
			"x":       nextX,
			"y":       nextY,
		}).Info("[POINTER] About to dispatch mouse event via chromedp.Run")

		// Use chromedp.MouseEvent to create an Action, then execute via chromedp.Run
		// This ensures the context has proper executor/target metadata
		mouseAction := chromedp.MouseEvent(input.MouseMoved, nextX, nextY)
		if buttons > 0 {
			mouseAction = chromedp.MouseEvent(input.MouseMoved, nextX, nextY, func(p *input.DispatchMouseEventParams) *input.DispatchMouseEventParams {
				return p.WithButtons(buttons)
			})
		}

		if err := chromedp.Run(ctx, mouseAction); err != nil {
			s.log.WithFields(map[string]interface{}{
				"step":    i,
				"ctx_err": ctx.Err(),
				"err":     err.Error(),
			}).Error("[POINTER] Mouse event dispatch failed")
			return err
		}
		s.log.WithField("step", i).Info("[POINTER] Mouse event dispatched successfully")

		if err := waitWithContext(ctx, stepDelay); err != nil {
			s.log.WithFields(map[string]interface{}{
				"step":      i,
				"ctx_err":   ctx.Err(),
				"stepDelay": stepDelay,
			}).Error("[POINTER] Wait failed")
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
