package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/constants"
	"github.com/vrooli/browser-automation-studio/internal/httpjson"
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

	instructions := []autocontracts.CompiledInstruction{
		{
			Index:  0,
			NodeID: "dom.navigate",
			Type:   "navigate",
			Params: map[string]any{
				"url":       url,
				"waitUntil": defaultPreviewWaitUntil,
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			},
		},
		{
			Index:  1,
			NodeID: "dom.wait",
			Type:   "wait",
			Params: map[string]any{
				"waitType":   "time",
				"durationMs": defaultDomExtractionWaitMs,
			},
		},
		{
			Index:  2,
			NodeID: domExtractionNodeID,
			Type:   "evaluate",
			Params: map[string]any{
				"expression": domExtractionExpression,
				"timeoutMs":  defaultPreviewTimeoutMilliseconds,
			},
		},
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
		value, ok := raw["value"]
		if !ok {
			return "", errors.New("dom extraction missing value payload")
		}
		encoded, marshalErr := json.Marshal(value)
		if marshalErr != nil {
			return "", fmt.Errorf("failed to encode dom extraction: %w", marshalErr)
		}
		return string(encoded), nil
	}

	return "", errors.New("no dom extraction outcome recorded")
}

// GetDOMTree handles POST /api/v1/dom-tree - returns the DOM structure for Browser Inspector
func (h *DOMHandler) GetDOMTree(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := httpjson.Decode(w, r, &req); err != nil {
		h.log.WithError(err).Error("Failed to decode DOM tree request")
		RespondError(w, ErrInvalidRequest)
		return
	}

	if req.URL == "" {
		RespondError(w, ErrMissingRequiredField.WithDetails(map[string]string{"field": "url"}))
		return
	}

	h.log.WithField("url", req.URL).Info("Extracting DOM tree")

	ctx, cancel := context.WithTimeout(r.Context(), constants.ElementAnalysisTimeout)
	defer cancel()

	domData, err := h.ExtractDOMTree(ctx, req.URL)
	if err != nil {
		h.log.WithError(err).Error("Failed to extract DOM tree")
		RespondError(w, ErrInternalServer.WithDetails(map[string]string{"operation": "extract_dom", "error": err.Error()}))
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(domData))
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
