package cdp

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/chromedp"
)

// ExecuteFrameSwitch updates the active frame context for subsequent selectors. [REQ:BAS-NODE-FRAME-SWITCH]
func (s *Session) ExecuteFrameSwitch(ctx context.Context, opts frameSwitchOptions) (*StepResult, error) {
	start := time.Now()
	mode := strings.ToLower(strings.TrimSpace(opts.SwitchBy))
	if mode == "" {
		mode = "selector"
	}
	if opts.TimeoutMs <= 0 {
		opts.TimeoutMs = 30000
	}

	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(opts.TimeoutMs)*time.Millisecond)
	defer cancel()

	result := &StepResult{DebugContext: map[string]interface{}{
		"switchBy": mode,
	}}

	var scope *frameScope
	var err error

	switch mode {
	case "main":
		s.clearFrameScopes()
	case "parent":
		if _, ok := s.popFrameScope(); !ok {
			err = fmt.Errorf("already at main document")
		}
	case "selector":
		scope, err = s.resolveFrameBySelector(timeoutCtx, opts.Selector)
	case "index":
		scope, err = s.resolveFrameByIndex(timeoutCtx, opts.Index)
	case "name":
		scope, err = s.resolveFrameByName(timeoutCtx, opts.Name)
	case "url":
		scope, err = s.resolveFrameByURL(timeoutCtx, opts.URLMatch)
	default:
		err = fmt.Errorf("unsupported frame switch mode %q", opts.SwitchBy)
	}

	if err != nil {
		result.Error = err.Error()
		result.DurationMs = int(time.Since(start).Milliseconds())
		return result, err
	}

	if scope != nil {
		s.pushFrameScope(scope)
		result.DebugContext["frameId"] = string(scope.FrameID)
		result.DebugContext["frameName"] = scope.Name
		result.DebugContext["frameURL"] = scope.URL
		result.DebugContext["origin"] = scope.Selector
	}

	result.DebugContext["depth"] = s.frameDepth()

	chromedp.Run(s.ctx,
		chromedp.Location(&result.URL),
		chromedp.Title(&result.Title),
	)

	result.Success = true
	result.DurationMs = int(time.Since(start).Milliseconds())
	result.ConsoleLogs, result.NetworkEvents = s.GetTelemetry()
	return result, nil
}

func (s *Session) resolveFrameBySelector(ctx context.Context, selector string) (*frameScope, error) {
	trimmed := strings.TrimSpace(selector)
	if trimmed == "" {
		return nil, fmt.Errorf("frame selector is required")
	}
	var nodes []*cdp.Node
	actions := []chromedp.Action{
		chromedp.Nodes(trimmed, &nodes, s.frameQueryOptions(chromedp.ByQueryAll, chromedp.AtLeast(1))...),
	}
	if err := chromedp.Run(ctx, actions...); err != nil {
		return nil, err
	}
	return s.describeFrameNode(ctx, nodes[0], fmt.Sprintf("selector:%s", trimmed))
}

func (s *Session) resolveFrameByIndex(ctx context.Context, index int) (*frameScope, error) {
	if index < 0 {
		return nil, fmt.Errorf("frame index must be non-negative")
	}
	frames, err := s.collectFrameNodes(ctx)
	if err != nil {
		return nil, err
	}
	if index >= len(frames) {
		return nil, fmt.Errorf("frame index %d out of range (found %d frames)", index, len(frames))
	}
	return s.describeFrameNode(ctx, frames[index], fmt.Sprintf("index:%d", index))
}

func (s *Session) resolveFrameByName(ctx context.Context, name string) (*frameScope, error) {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return nil, fmt.Errorf("frame name is required")
	}
	frames, err := s.collectFrameNodes(ctx)
	if err != nil {
		return nil, err
	}
	for _, node := range frames {
		if node.AttributeValue("name") == trimmed {
			return s.describeFrameNode(ctx, node, fmt.Sprintf("name:%s", trimmed))
		}
	}
	return nil, fmt.Errorf("frame with name %q not found", trimmed)
}

func (s *Session) resolveFrameByURL(ctx context.Context, pattern string) (*frameScope, error) {
	trimmed := strings.TrimSpace(pattern)
	if trimmed == "" {
		return nil, fmt.Errorf("frame urlMatch is required")
	}
	frames, err := s.collectFrameNodes(ctx)
	if err != nil {
		return nil, err
	}
	matcher, err := buildURLMatcher(trimmed)
	if err != nil {
		return nil, err
	}
	for _, node := range frames {
		scope, err := s.describeFrameNode(ctx, node, fmt.Sprintf("url:%s", trimmed))
		if err != nil {
			continue
		}
		if matcher(scope.URL) {
			return scope, nil
		}
	}
	return nil, fmt.Errorf("no frame matches url pattern %q", trimmed)
}

func (s *Session) collectFrameNodes(ctx context.Context) ([]*cdp.Node, error) {
	var nodes []*cdp.Node
	if err := chromedp.Run(ctx,
		chromedp.Nodes("iframe,frame", &nodes, s.frameQueryOptions(chromedp.ByQueryAll)...),
	); err != nil {
		return nil, err
	}
	if len(nodes) == 0 {
		return nil, fmt.Errorf("no frames are available in the current context")
	}
	return nodes, nil
}

func buildURLMatcher(pattern string) (func(string) bool, error) {
	if len(pattern) >= 2 && strings.HasPrefix(pattern, "/") && strings.HasSuffix(pattern, "/") {
		expr := pattern[1 : len(pattern)-1]
		re, err := regexp.Compile(expr)
		if err != nil {
			return nil, fmt.Errorf("invalid frame url regex: %w", err)
		}
		return func(candidate string) bool {
			return re.MatchString(candidate)
		}, nil
	}
	lowered := strings.ToLower(pattern)
	return func(candidate string) bool {
		return strings.Contains(strings.ToLower(candidate), lowered)
	}, nil
}
