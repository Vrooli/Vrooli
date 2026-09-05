package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/sirupsen/logrus"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/constants"
)

const (
	domExtractionNodeID        = "dom.extract"
	defaultDomExtractionWaitMs = 750
)

var domExtractionExpression = `(function() {
  const MAX_DEPTH = 6;
  const MAX_CHILDREN_PER_NODE = 12;
  const MAX_TOTAL_NODES = 800;
  const TEXT_LIMIT = 120;

  let nodeCount = 0;

  const trimText = (value) => {
    if (typeof value !== 'string') {
      return null;
    }
    const normalized = value.replace(/\s+/g, ' ').trim();
    if (!normalized) {
      return null;
    }
    if (normalized.length > TEXT_LIMIT) {
      return normalized.slice(0, TEXT_LIMIT - 1) + '…';
    }
    return normalized;
  };

  const shouldSkip = (element) => {
    const tag = element.tagName ? element.tagName.toLowerCase() : '';
    return ['script', 'style', 'noscript', 'template', 'meta', 'link'].includes(tag);
  };

  const buildSelector = (element) => {
    if (!element || element.nodeType !== Node.ELEMENT_NODE) {
      return '';
    }
    if (element.id) {
      return '#' + element.id;
    }
    const segments = [];
    let current = element;
    let guard = 0;
    while (current && current.nodeType === Node.ELEMENT_NODE && guard < 25) {
      if (current.id) {
        segments.unshift('#' + current.id);
        break;
      }
      const tag = current.tagName.toLowerCase();
      const parent = current.parentElement;
      if (!parent) {
        segments.unshift(tag);
        break;
      }
      const siblings = Array.from(parent.children).filter((sibling) => sibling.tagName === current.tagName);
      if (siblings.length > 1) {
        const index = siblings.indexOf(current);
        segments.unshift(tag + ':nth-of-type(' + (index + 1) + ')');
      } else {
        segments.unshift(tag);
      }
      current = parent;
      guard += 1;
    }
    return segments.join(' > ');
  };

  const buildNode = (element, depth) => {
    if (!element || element.nodeType !== Node.ELEMENT_NODE) {
      return null;
    }
    if (shouldSkip(element)) {
      return null;
    }
    if (nodeCount >= MAX_TOTAL_NODES) {
      return null;
    }
    nodeCount += 1;

    const tagName = element.tagName || 'UNKNOWN';
    const text = trimText(element.textContent || '');
    const selector = buildSelector(element) || tagName.toLowerCase();

    const node = {
      tagName,
      id: element.id || null,
      className: typeof element.className === 'string' && element.className ? element.className : null,
      text,
      type: element.type || null,
      href: typeof element.href === 'string' ? element.href : null,
      ariaLabel: element.getAttribute ? element.getAttribute('aria-label') : null,
      placeholder: element.placeholder || null,
      value: element.value || null,
      selector,
      children: []
    };

    if (depth < MAX_DEPTH) {
      const children = [];
      for (const child of Array.from(element.children)) {
        if (children.length >= MAX_CHILDREN_PER_NODE) {
          break;
        }
        const built = buildNode(child, depth + 1);
        if (built) {
          children.push(built);
        }
      }
      node.children = children;
    }

    return node;
  };

  const root = document.body || document.documentElement;
  const tree = buildNode(root, 0);

  if (tree) {
    return tree;
  }

  const fallback = document.body || document.documentElement;
  if (fallback) {
    return buildNode(fallback, 0);
  }

  return {
    tagName: 'BODY',
    id: null,
    className: null,
    text: null,
    type: null,
    href: null,
    ariaLabel: null,
    placeholder: null,
    value: null,
    selector: 'body',
    children: [],
  };
})()`

// DOMHandler handles DOM tree extraction operations
type DOMHandler struct {
	log    *logrus.Logger
	runner AutomationRunner
}

// DOMHandlerOption configures the DOMHandler.
type DOMHandlerOption func(*DOMHandler)

// WithDOMRunner sets a custom automation runner for DOM extraction.
func WithDOMRunner(runner AutomationRunner) DOMHandlerOption {
	return func(h *DOMHandler) {
		h.runner = runner
	}
}

// NewDOMHandler creates a new DOM handler with optional configuration.
func NewDOMHandler(log *logrus.Logger, opts ...DOMHandlerOption) *DOMHandler {
	handler := &DOMHandler{log: log}

	// Apply options first
	for _, opt := range opts {
		opt(handler)
	}

	// Create default runner if not provided
	if handler.runner == nil {
		runner, err := newAutomationRunner(log)
		if err != nil && log != nil {
			log.WithError(err).Warn("Failed to initialize automation runner for DOM extraction; requests will fail")
		}
		handler.runner = runner
	}

	return handler
}

// ExtractDOMTree extracts the DOM tree from a given URL
func (h *DOMHandler) ExtractDOMTree(ctx context.Context, url string) (string, error) {
	// Normalize URL - add protocol if missing
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	if h.runner == nil {
		return "", errors.New("automation runner not configured")
	}

	instructions, err := h.buildDOMExtractionInstructions(url)
	if err != nil {
		return "", fmt.Errorf("build dom extraction instructions: %w", err)
	}

	outcomes, _, err := h.runner.Run(ctx, previewDefaultViewportWidth, previewDefaultViewportHeight, instructions)
	if err != nil {
		return "", fmt.Errorf("automation run failed: %w", err)
	}

	for _, outcome := range outcomes {
		if outcome.NodeID != domExtractionNodeID {
			continue
		}
		if !outcome.Success {
			return "", fmt.Errorf("dom extraction failed: %s", failureMessage(outcome.Failure))
		}
		raw := outcome.ExtractedData
		if raw == nil {
			return "", errors.New("dom extraction returned no data")
		}
		value, ok := raw["result"]
		if !ok {
			return "", errors.New("dom extraction missing result payload")
		}
		encoded, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return "", fmt.Errorf("failed to encode dom extraction: %w", marshalErr)
		}
		return string(encoded), nil
	}

	return "", errors.New("no dom extraction outcome recorded")
}

// GetDOMTreeJSON is the transport-agnostic core of the dom-tree endpoint.
// It enforces the standard timeout and returns the raw JSON payload emitted
// by the page-side extractor. Callers (Connect handlers, tests) decode the
// JSON themselves.
func (h *DOMHandler) GetDOMTreeJSON(ctx context.Context, url string) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", ErrMissingURL
	}
	if h.runner == nil {
		return "", ErrAutomationRunnerNotReady
	}
	if h.log != nil {
		h.log.WithField("url", url).Info("Extracting DOM tree")
	}
	ctx, cancel := context.WithTimeout(ctx, constants.ElementAnalysisTimeout)
	defer cancel()
	return h.ExtractDOMTree(ctx, url)
}

// buildDOMExtractionInstructions creates the compiled instructions for DOM extraction.
// Returns an error if any action type fails to build (indicates a programming error).
func (h *DOMHandler) buildDOMExtractionInstructions(url string) ([]autocontracts.CompiledInstruction, error) {
	steps := []struct {
		nodeID   string
		stepType string
		params   map[string]any
	}{
		{
			nodeID:   "dom.navigate",
			stepType: "navigate",
			params: map[string]any{
				"url":       url,
				"waitUntil": defaultPreviewWaitUntil,
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			},
		},
		{
			nodeID:   "dom.wait",
			stepType: "wait",
			params: map[string]any{
				"waitType":   "time",
				"durationMs": defaultDomExtractionWaitMs,
			},
		},
		{
			nodeID:   domExtractionNodeID,
			stepType: "evaluate",
			params: map[string]any{
				"expression": domExtractionExpression,
				"timeoutMs":  defaultPreviewTimeoutMilliseconds,
			},
		},
	}

	instructions := make([]autocontracts.CompiledInstruction, 0, len(steps))
	for i, step := range steps {
		action, err := autocompiler.BuildActionDefinition(step.stepType, step.params)
		if err != nil {
			return nil, fmt.Errorf("build action %q (node %s): %w", step.stepType, step.nodeID, err)
		}
		instructions = append(instructions, autocontracts.CompiledInstruction{
			Index:  i,
			NodeID: step.nodeID,
			Action: action,
		})
	}

	return instructions, nil
}

func failureMessage(f *autocontracts.StepFailure) string {
	if f == nil {
		return "unknown failure"
	}
	if trimmed := strings.TrimSpace(f.Message); trimmed != "" {
		return trimmed
	}
	if f.Kind != "" {
		return string(f.Kind)
	}
	return "unknown failure"
}
