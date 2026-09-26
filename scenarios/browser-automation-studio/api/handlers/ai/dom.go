package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/sirupsen/logrus"
	autocompiler "github.com/vrooli/browser-automation-studio/automation/compiler"
	autocontracts "github.com/vrooli/browser-automation-studio/automation/contracts"
	"github.com/vrooli/browser-automation-studio/constants"
)

const (
	domExtractionNodeID        = "dom.extract"
	defaultDomExtractionWaitMs = 750 // retained for the legacy timing contract test
	defaultDOMMaxNodes         = 4000
	maxDOMMaxNodes             = 4000
)

var domExtractionExpression = `(function() {
	const MAX_DEPTH = 20;
  const MAX_TOTAL_NODES = 4000;
  const MAX_DATA_ATTRS = 32;
  const MAX_DATA_VALUE = 1024;
  const INCLUDE_COMPUTED = true;
  const TEXT_LIMIT = 120;

  let nodeCount = 0;

  const readDataAttributes = (element) => {
    const data = Object.create(null);
    for (const key of Object.keys(element.dataset || {}).slice(0, MAX_DATA_ATTRS)) {
      const value = element.dataset[key];
      if (typeof value === 'string') data[key] = value.slice(0, MAX_DATA_VALUE);
    }
    return Object.keys(data).length ? data : null;
  };

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

  const implicitRole = (element) => {
    const explicit = element.getAttribute && element.getAttribute('role');
    if (explicit) {
      return explicit;
    }
    const tag = element.tagName ? element.tagName.toLowerCase() : '';
    const roles = {
      a: element.hasAttribute('href') ? 'link' : null,
      article: 'article',
      button: 'button',
      dialog: 'dialog',
      form: 'form',
      h1: 'heading', h2: 'heading', h3: 'heading', h4: 'heading', h5: 'heading', h6: 'heading',
      img: 'img',
      input: 'textbox',
      li: 'listitem',
      main: 'main',
      nav: 'navigation',
      option: 'option',
      progress: 'progressbar',
      section: 'region',
      table: 'table',
      textarea: 'textbox',
      ul: 'list', ol: 'list'
    };
    return roles[tag] || null;
  };

  const inScrollContainer = (element) => {
    for (let parent = element.parentElement; parent; parent = parent.parentElement) {
      const style = getComputedStyle(parent);
      const scrollsY = /(auto|scroll|overlay)/.test(style.overflowY || '');
      if (scrollsY && parent.scrollHeight > parent.clientHeight + 1) {
        return true;
      }
    }
    const root = element.ownerDocument && element.ownerDocument.documentElement;
    const body = element.ownerDocument && element.ownerDocument.body;
    return !!(root && body && Math.max(root.scrollHeight, body.scrollHeight) > window.innerHeight + 1);
  };

  const readComputed = (element) => {
    if (!INCLUDE_COMPUTED) {
      return null;
    }
    const styles = getComputedStyle(element);
    const rect = element.getBoundingClientRect();
    return {
      computed: {
        display: styles.display,
        position: styles.position,
        margin: styles.margin,
        padding: styles.padding,
        gap: styles.gap,
        flexDirection: styles.flexDirection,
        flexWrap: styles.flexWrap,
        gridTemplateColumns: styles.gridTemplateColumns,
        gridTemplateRows: styles.gridTemplateRows,
        color: styles.color,
        backgroundColor: styles.backgroundColor,
        font: styles.font,
        overflow: styles.overflow,
        textOverflow: styles.textOverflow,
        whiteSpace: styles.whiteSpace,
        zIndex: styles.zIndex
      },
      clientWidth: element.clientWidth,
      clientHeight: element.clientHeight,
      scrollWidth: element.scrollWidth,
      scrollHeight: element.scrollHeight,
      rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height }
    };
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
		const ariaLabel = element.getAttribute ? element.getAttribute('aria-label') : null;
		const text = trimText((ariaLabel && ['BUTTON', 'A', 'INPUT', 'TEXTAREA'].includes(tagName)) ? ariaLabel : (element.innerText || element.textContent || ''));
		const selector = buildSelector(element) || tagName.toLowerCase();

    const node = {
      tagName,
      role: implicitRole(element),
      id: element.id || null,
      className: typeof element.className === 'string' && element.className ? element.className : null,
      text,
      type: element.type || null,
      href: typeof element.href === 'string' ? element.href : null,
		ariaLabel,
      placeholder: element.placeholder || null,
      value: element.value || null,
      selector,
      inScrollContainer: inScrollContainer(element),
      children: []
    };
    const data = readDataAttributes(element);
    if (data) node.data = data;

    const layout = readComputed(element);
    if (layout) {
      node.computed = layout.computed;
      node.clientWidth = layout.clientWidth;
      node.clientHeight = layout.clientHeight;
      node.scrollWidth = layout.scrollWidth;
      node.scrollHeight = layout.scrollHeight;
      node.rect = layout.rect;
    }

    if (depth < MAX_DEPTH) {
      const children = [];
      for (const child of Array.from(element.children)) {
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
    tree.truncated = nodeCount >= MAX_TOTAL_NODES;
    return tree;
  }

  const fallback = document.body || document.documentElement;
  if (fallback) {
    return buildNode(fallback, 0);
  }

  return {
    tagName: 'BODY',
    role: 'document',
    id: null,
    className: null,
    text: null,
    type: null,
    href: null,
    ariaLabel: null,
    placeholder: null,
    value: null,
    selector: 'body',
    truncated: false,
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
	return h.extractDOMTree(ctx, url, defaultPreviewWaitUntil, "", 0, true, defaultDOMMaxNodes)
}

// ExtractDOMTreeWithOptions extracts a settled DOM tree without exposing the
// browser runner to transport callers.
func (h *DOMHandler) ExtractDOMTreeWithOptions(ctx context.Context, url, waitUntil, waitFor string, settleMs int) (string, error) {
	return h.extractDOMTree(ctx, url, normalizeWaitUntil(waitUntil), waitFor, settleMs, true, defaultDOMMaxNodes)
}

func (h *DOMHandler) extractDOMTree(ctx context.Context, url, waitUntil, waitFor string, settleMs int, computed bool, maxNodes int) (string, error) {
	// Normalize URL - add protocol if missing
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	if h.runner == nil {
		return "", errors.New("automation runner not configured")
	}
	if settleMs < 0 || settleMs > 15000 {
		return "", fmt.Errorf("settle_ms must be between 0 and 15000")
	}
	if maxNodes <= 0 || maxNodes > maxDOMMaxNodes {
		return "", fmt.Errorf("max_nodes must be between 1 and %d", maxDOMMaxNodes)
	}

	instructions, err := h.buildDOMExtractionInstructions(url, waitUntil, waitFor, settleMs, computed, maxNodes)
	if err != nil {
		return "", fmt.Errorf("build dom extraction instructions: %w", err)
	}

	outcomes, _, err := h.runner.Run(ctx, previewDefaultViewportWidth, previewDefaultViewportHeight, instructions)
	if err != nil {
		return "", fmt.Errorf("automation run failed: %w", err)
	}

	byNodeID := make(map[string]autocontracts.StepOutcome, len(outcomes))
	for _, outcome := range outcomes {
		byNodeID[outcome.NodeID] = outcome
	}
	if err := waitOutcomeError(byNodeID, "dom.wait-selector", waitFor); err != nil {
		return "", err
	}
	if err := waitOutcomeError(byNodeID, "dom.settle", "settle_ms"); err != nil {
		return "", err
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
	return h.GetDOMTreeJSONWithOptions(ctx, url, defaultPreviewWaitUntil, "", 0)
}

// GetDOMTreeJSONWithOptions is the option-aware DOM snapshot contract used by
// the AI CLI and Connect service.
func (h *DOMHandler) GetDOMTreeJSONWithOptions(ctx context.Context, url, waitUntil, waitFor string, settleMs int) (string, error) {
	return h.GetDOMTreeJSONWithCaptureOptions(ctx, url, waitUntil, waitFor, settleMs, true, defaultDOMMaxNodes)
}

// GetDOMTreeJSONWithCaptureOptions adds explicit layout metadata and a bounded
// node budget to the settled DOM snapshot contract.
func (h *DOMHandler) GetDOMTreeJSONWithCaptureOptions(ctx context.Context, url, waitUntil, waitFor string, settleMs int, computed bool, maxNodes int) (string, error) {
	if strings.TrimSpace(url) == "" {
		return "", ErrMissingURL
	}
	if h.runner == nil {
		return "", ErrAutomationRunnerNotReady
	}
	if maxNodes == 0 {
		maxNodes = defaultDOMMaxNodes
	}
	if h.log != nil {
		h.log.WithField("url", url).Info("Extracting DOM tree")
	}
	ctx, cancel := context.WithTimeout(ctx, constants.ElementAnalysisTimeout)
	defer cancel()
	return h.extractDOMTree(ctx, url, waitUntil, waitFor, settleMs, computed, maxNodes)
}

// buildDOMExtractionInstructions creates the compiled instructions for DOM extraction.
// Returns an error if any action type fails to build (indicates a programming error).
func (h *DOMHandler) buildDOMExtractionInstructions(url, waitUntil, waitFor string, settleMs int, computed bool, maxNodes int) ([]autocontracts.CompiledInstruction, error) {
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
				"waitUntil": normalizeWaitUntil(waitUntil),
				"timeoutMs": defaultPreviewTimeoutMilliseconds,
			},
		},
	}
	if selector := strings.TrimSpace(waitFor); selector != "" {
		steps = append(steps, struct {
			nodeID   string
			stepType string
			params   map[string]any
		}{"dom.wait-selector", "wait", map[string]any{"selector": selector, "timeoutMs": defaultPreviewTimeoutMilliseconds}})
	}
	if settleMs > 0 {
		steps = append(steps, struct {
			nodeID   string
			stepType string
			params   map[string]any
		}{"dom.settle", "wait", map[string]any{"timeoutMs": settleMs}})
	}
	steps = append(steps, struct {
		nodeID   string
		stepType string
		params   map[string]any
	}{domExtractionNodeID, "evaluate", map[string]any{
		"expression": domExtractionExpressionForOptions(computed, maxNodes),
		"timeoutMs":  defaultPreviewTimeoutMilliseconds,
	}})

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

func domExtractionExpressionForOptions(computed bool, maxNodes int) string {
	expression := domExtractionExpression
	expression = strings.Replace(expression, "const MAX_TOTAL_NODES = 4000;", "const MAX_TOTAL_NODES = "+strconv.Itoa(maxNodes)+";", 1)
	if !computed {
		expression = strings.Replace(expression, "const INCLUDE_COMPUTED = true;", "const INCLUDE_COMPUTED = false;", 1)
	}
	return expression
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
