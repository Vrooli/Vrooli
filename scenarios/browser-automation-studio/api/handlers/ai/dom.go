package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/vrooli/browser-automation-studio/constants"
)

// DOMHandler handles DOM tree extraction operations
type DOMHandler struct {
	log *logrus.Logger
}

// NewDOMHandler creates a new DOM handler
func NewDOMHandler(log *logrus.Logger) *DOMHandler {
	return &DOMHandler{log: log}
}

// ExtractDOMTree extracts the DOM tree from a given URL
func (h *DOMHandler) ExtractDOMTree(ctx context.Context, url string) (string, error) {
	// Normalize URL - add protocol if missing
	if !strings.HasPrefix(url, "http://") && !strings.HasPrefix(url, "https://") {
		url = "https://" + url
	}

	// JavaScript to extract DOM tree with selectors - explore meaningful nodes
	script := `
try {
  const MAX_DEPTH = 6;
  const MAX_CHILDREN_PER_NODE = 12;
  const MAX_TOTAL_NODES = 800;
  const TEXT_LIMIT = 120;

  await new Promise((resolve) => setTimeout(resolve, 750));

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
      ariaLabel: element.getAttribute('aria-label') || null,
      placeholder: element.placeholder || null,
      value: typeof element.value === 'string' && element.value !== '' ? element.value : null,
      selector,
      children: [],
    };

    if (depth >= MAX_DEPTH) {
      return node;
    }

    const childElements = Array.from(element.children || []);
    let included = 0;
    for (const child of childElements) {
      if (included >= MAX_CHILDREN_PER_NODE) {
        break;
      }
      const childNode = buildNode(child, depth + 1);
      if (childNode) {
        node.children.push(childNode);
        included += 1;
      }
    }

    return node;
  };

  const root = document.documentElement || document.body;
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
} catch (error) {
  return {
    tagName: 'ERROR',
    selector: 'error',
    text: error && error.message ? String(error.message) : 'Failed to extract DOM',
    children: [],
  };
}`

	// Create temp file for output
	tmpFile, err := os.CreateTemp("", "dom-extract-*.json")
	if err != nil {
		return "", fmt.Errorf("failed to create temp file: %w", err)
	}
	defer os.Remove(tmpFile.Name())
	defer tmpFile.Close()

	// Use browserless extract to get the DOM
	extractCmd := exec.CommandContext(ctx, "resource-browserless", "extract", url,
		"--script", script,
		"--output", tmpFile.Name())

	output, err := extractCmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("failed to extract DOM: %w, output: %s", err, string(output))
	}

	// Read the extracted DOM data
	domData, err := os.ReadFile(tmpFile.Name())
	if err != nil {
		return "", fmt.Errorf("failed to read DOM data: %w", err)
	}

	// Return the extracted DOM data directly (browserless now returns proper JSON)
	return string(domData), nil
}

// GetDOMTree handles POST /api/v1/dom-tree - returns the DOM structure for Browser Inspector
func (h *DOMHandler) GetDOMTree(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
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
