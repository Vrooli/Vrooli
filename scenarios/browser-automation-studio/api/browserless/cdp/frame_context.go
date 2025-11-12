package cdp

import (
	"context"
	"fmt"
	"strings"

	"github.com/chromedp/cdproto/cdp"
	"github.com/chromedp/cdproto/dom"
	"github.com/chromedp/cdproto/page"
	"github.com/chromedp/cdproto/runtime"
	"github.com/chromedp/chromedp"
)

type frameScope struct {
	FrameID  cdp.FrameID
	Name     string
	URL      string
	Selector string
	Document *cdp.Node
}

func (s *Session) currentFrameScope() *frameScope {
	s.frameMu.RLock()
	defer s.frameMu.RUnlock()
	if len(s.frameStack) == 0 {
		return nil
	}
	return s.frameStack[len(s.frameStack)-1]
}

func (s *Session) currentFrameNode() *cdp.Node {
	if scope := s.currentFrameScope(); scope != nil {
		return scope.Document
	}
	return nil
}

func (s *Session) currentFrameID() (cdp.FrameID, bool) {
	if scope := s.currentFrameScope(); scope != nil && scope.FrameID != "" {
		return scope.FrameID, true
	}
	return "", false
}

func (s *Session) frameDepth() int {
	s.frameMu.RLock()
	defer s.frameMu.RUnlock()
	return len(s.frameStack)
}

func (s *Session) pushFrameScope(scope *frameScope) {
	if scope == nil {
		return
	}
	s.frameMu.Lock()
	s.frameStack = append(s.frameStack, scope)
	s.frameMu.Unlock()
}

func (s *Session) popFrameScope() (*frameScope, bool) {
	s.frameMu.Lock()
	defer s.frameMu.Unlock()
	if len(s.frameStack) == 0 {
		return nil, false
	}
	idx := len(s.frameStack) - 1
	scope := s.frameStack[idx]
	s.frameStack = s.frameStack[:idx]
	return scope, true
}

func (s *Session) clearFrameScopes() {
	s.frameMu.Lock()
	s.frameStack = nil
	s.frameMu.Unlock()
}

func (s *Session) frameQueryOptions(opts ...chromedp.QueryOption) []chromedp.QueryOption {
	scope := s.currentFrameScope()
	if scope == nil || scope.Document == nil {
		return opts
	}
	combined := make([]chromedp.QueryOption, 0, len(opts)+1)
	combined = append(combined, opts...)
	combined = append(combined, chromedp.FromNode(scope.Document))
	return combined
}

func (s *Session) evalWithFrame(ctx context.Context, expression string, out any) error {
	frameID, ok := s.currentFrameID()
	if !ok || frameID == "" {
		return chromedp.Run(ctx, chromedp.Evaluate(expression, out))
	}
	execID, err := s.createFrameExecutionContext(ctx, frameID)
	if err != nil {
		return err
	}
	return chromedp.Run(ctx, chromedp.Evaluate(expression, out, evaluateWithContext(execID)))
}

func (s *Session) createFrameExecutionContext(ctx context.Context, frameID cdp.FrameID) (runtime.ExecutionContextID, error) {
	var execID runtime.ExecutionContextID
	err := chromedp.Run(ctx, chromedp.ActionFunc(func(runCtx context.Context) error {
		id, err := page.CreateIsolatedWorld(frameID).WithGrantUniveralAccess(true).Do(runCtx)
		if err != nil {
			return err
		}
		execID = id
		return nil
	}))
	if err != nil {
		return 0, err
	}
	return execID, nil
}

func evaluateWithContext(id runtime.ExecutionContextID) chromedp.EvaluateOption {
	return func(params *runtime.EvaluateParams) *runtime.EvaluateParams {
		return params.WithContextID(id)
	}
}

func (s *Session) describeFrameNode(ctx context.Context, node *cdp.Node, origin string) (*frameScope, error) {
	if node == nil {
		return nil, fmt.Errorf("frame %s is unavailable", origin)
	}
	described, err := dom.DescribeNode().
		WithNodeID(node.NodeID).
		WithDepth(1).
		WithPierce(true).
		Do(ctx)
	if err != nil {
		return nil, err
	}
	if described == nil {
		return nil, fmt.Errorf("frame %s could not be described", origin)
	}
	name := strings.ToUpper(strings.TrimSpace(described.NodeName))
	if name != "IFRAME" && name != "FRAME" {
		return nil, fmt.Errorf("selector %s did not resolve to a frame element", origin)
	}
	doc := described.ContentDocument
	if doc == nil {
		if err := dom.RequestChildNodes(described.NodeID).WithDepth(1).Do(ctx); err != nil {
			return nil, fmt.Errorf("frame %s content unavailable: %w", origin, err)
		}
		described, err = dom.DescribeNode().
			WithNodeID(node.NodeID).
			WithDepth(1).
			WithPierce(true).
			Do(ctx)
		if err != nil {
			return nil, err
		}
		doc = described.ContentDocument
		if doc == nil {
			return nil, fmt.Errorf("frame %s has no accessible document", origin)
		}
	}
	return &frameScope{
		FrameID:  doc.FrameID,
		Name:     described.AttributeValue("name"),
		URL:      doc.DocumentURL,
		Selector: origin,
		Document: doc,
	}, nil
}
