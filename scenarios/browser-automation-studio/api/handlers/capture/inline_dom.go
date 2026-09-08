package capture

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
)

// defaultInlineDomMaxBytes caps CaptureResponse.dom_html so a pathological
// page cannot balloon the RPC response. Truncation is silent (documented
// on the proto field); readable-text consumers tolerate a cut-off tail by
// design.
const defaultInlineDomMaxBytes = 2 << 20

// defaultInlineDomExpression is evaluated in-page to read the rendered DOM.
// An EVALUATE node is used (not EXTRACT) because the Playwright driver's
// extract handler only does textContent today, while evaluate returns the
// raw expression result through extracted_data.
const defaultInlineDomExpression = "document.documentElement.outerHTML"

// defaultInlineDomTreeExpression is a compact, bounded layout snapshot used
// by the single-session capture surface. It intentionally mirrors the
// browser AI DOM contract without depending on the AI handler package.
const defaultInlineDomTreeExpression = `(function() {
  const MAX_DEPTH = 20;
  const MAX_NODES = 4000;
  let count = 0;
  const role = (el) => el.getAttribute('role') || ({H1:'heading',H2:'heading',H3:'heading',H4:'heading',H5:'heading',H6:'heading',BUTTON:'button',A:el.hasAttribute('href')?'link':null,NAV:'navigation',MAIN:'main',IMG:'img'}[el.tagName] || null);
  const inScrollContainer = (el) => {
    for (let parent = el.parentElement; parent; parent = parent.parentElement) {
      const style = getComputedStyle(parent);
      if (/(auto|scroll|overlay)/.test(style.overflowY || '') && parent.scrollHeight > parent.clientHeight + 1) return true;
    }
    const root = el.ownerDocument && el.ownerDocument.documentElement;
    const body = el.ownerDocument && el.ownerDocument.body;
    return !!(root && body && Math.max(root.scrollHeight, body.scrollHeight) > window.innerHeight + 1);
  };
  const selector = (el) => el.id ? '#' + el.id : el.tagName.toLowerCase();
  const build = (el, depth) => {
    if (!el || el.nodeType !== Node.ELEMENT_NODE || ['SCRIPT','STYLE','NOSCRIPT','TEMPLATE','META','LINK'].includes(el.tagName) || count >= MAX_NODES) return null;
    count += 1;
    const style = getComputedStyle(el);
    const rect = el.getBoundingClientRect();
    const ariaLabel = el.getAttribute('aria-label');
    const text = (ariaLabel && ['BUTTON','A','INPUT','TEXTAREA'].includes(el.tagName) ? ariaLabel : (el.innerText || el.textContent || '')).replace(/\s+/g, ' ').trim().slice(0, 120) || null;
    const node = {tagName: el.tagName, role: role(el), id: el.id || null, text, ariaLabel, selector: selector(el), inScrollContainer: inScrollContainer(el), clientWidth: el.clientWidth, clientHeight: el.clientHeight, scrollWidth: el.scrollWidth, scrollHeight: el.scrollHeight, computed: {display: style.display, position: style.position, margin: style.margin, padding: style.padding, gap: style.gap, flexDirection: style.flexDirection, gridTemplateColumns: style.gridTemplateColumns, color: style.color, backgroundColor: style.backgroundColor, font: style.font, overflow: style.overflow, textOverflow: style.textOverflow, whiteSpace: style.whiteSpace, zIndex: style.zIndex}, rect: {x: rect.x, y: rect.y, width: rect.width, height: rect.height}, children: []};
    if (depth < MAX_DEPTH) for (const child of Array.from(el.children)) { const built = build(child, depth + 1); if (built) node.children.push(built); }
    return node;
  };
  const tree = build(document.body || document.documentElement, 0);
  if (tree) tree.truncated = count >= MAX_NODES;
  return tree;
})()`

// InlineDomConfig holds the tunables for inline-DOM capture. Zero values
// fall back to the package defaults via withDefaults so callers can wire
// only the fields they want to override.
type InlineDomConfig struct {
	// Expression is the in-page JS evaluated to read the rendered DOM.
	Expression string
	// MaxBytes caps the returned dom_html length (silent truncation).
	MaxBytes int
}

// withDefaults returns a copy with any unset field filled from the
// package defaults.
func (c InlineDomConfig) withDefaults() InlineDomConfig {
	if c.Expression == "" {
		c.Expression = defaultInlineDomExpression
	}
	if c.MaxBytes <= 0 {
		c.MaxBytes = defaultInlineDomMaxBytes
	}
	return c
}

// readInlineDom pulls the evaluate node's expression result out of the
// exported timeline.json. The driver's evaluate handler returns the raw
// result under the "result" key of extracted_data, which the execution
// writer persists verbatim as the frame's extracted_data_preview.
func (c InlineDomConfig) readInlineDom(outDir, nodeID string) (string, error) {
	raw, err := os.ReadFile(filepath.Join(outDir, "timeline.json"))
	if err != nil {
		return "", fmt.Errorf("read timeline.json: %w", err)
	}
	var timeline struct {
		Frames []struct {
			NodeID               string         `json:"node_id"`
			ExtractedDataPreview map[string]any `json:"extracted_data_preview"`
		} `json:"frames"`
	}
	if err := json.Unmarshal(raw, &timeline); err != nil {
		return "", fmt.Errorf("decode timeline.json: %w", err)
	}
	for _, frame := range timeline.Frames {
		if frame.NodeID != nodeID {
			continue
		}
		if value, ok := frame.ExtractedDataPreview["result"].(string); ok && value != "" {
			if len(value) > c.MaxBytes {
				value = value[:c.MaxBytes]
			}
			return value, nil
		}
		if value := frame.ExtractedDataPreview["result"]; value != nil {
			encoded, marshalErr := json.Marshal(value)
			if marshalErr != nil {
				return "", fmt.Errorf("encode DOM evaluate result: %w", marshalErr)
			}
			if len(encoded) > c.MaxBytes {
				encoded = encoded[:c.MaxBytes]
			}
			return string(encoded), nil
		}
	}
	return "", errors.New("timeline has no DOM evaluate result")
}
