package browsercapture

import (
	"encoding/json"
	"fmt"
	"strings"
)

// Node IDs for the smoke workflow. They are referenced by the timeline mapper
// to locate the handshake-gating assert and the frame screenshot.
const (
	nodeNavigate  = "smoke-navigate-host"
	nodeInject    = "smoke-inject-frame"
	nodeHandshake = "smoke-assert-handshake"
	nodeScreens   = "smoke-screenshot-frame"

	// hostFrameSelector is the id of the host iframe element that embeds the
	// scenario UI. The screenshot step captures this element.
	hostFrameSelector = "#ui-smoke-frame"
	// bridgeReadyMarker is the host-DOM attribute the injected listener sets
	// once the iframe-bridge child signals ready. The handshake assert checks
	// for its presence — this is the smoke contract's hard-fail gate.
	bridgeReadyMarker = "[data-smoke-bridge-ready]"
)

// DefaultHandshakeSignals are the window property paths checked, inside the
// embedded frame, for the iframe-bridge readiness signal when no custom signals
// are configured. They mirror the historical smoke contract's KnownSignals.
var DefaultHandshakeSignals = []string{
	"__vrooliBridgeChildInstalled",
	"IFRAME_BRIDGE_READY",
	"IframeBridge.ready",
	"iframeBridge.ready",
	"IframeBridge.getState().ready",
}

// workflowParams configures the inline smoke workflow.
type workflowParams struct {
	// ScenarioURL is the UI URL embedded inside the host iframe.
	ScenarioURL string
	// HandshakeSignals are the window property paths polled for readiness
	// inside the embedded frame; DefaultHandshakeSignals when empty.
	HandshakeSignals []string
	// HandshakeTimeoutMs bounds how long the handshake assert waits for the
	// readiness marker.
	HandshakeTimeoutMs int64
	// ViewportWidth / ViewportHeight size the browser viewport.
	ViewportWidth  int
	ViewportHeight int
}

// buildWorkflow authors the inline smoke workflow as a BAS workflow-definition
// map (the proto-JSON node graph). It reproduces the proven keystone shape:
//
//  1. navigate to about:blank (the host shell);
//  2. evaluate: install console/error/network observers on the host window,
//     inject <iframe src=scenarioURL>, and register a message listener that
//     sets the [data-smoke-bridge-ready] marker on the host DOM when the
//     iframe-bridge child posts READY/HELLO (and a same-origin property poll as
//     a fallback);
//  3. assert [data-smoke-bridge-ready] EXISTS (the handshake hard-fail gate),
//     bounded by HandshakeTimeoutMs — this reproduces the old waitForFunction
//     semantics: the assert succeeds the moment the marker appears and fails on
//     timeout;
//  4. screenshot the host iframe element.
func buildWorkflow(p workflowParams) map[string]any {
	signals := p.HandshakeSignals
	if len(signals) == 0 {
		signals = DefaultHandshakeSignals
	}
	width := p.ViewportWidth
	if width <= 0 {
		width = DefaultViewportWidth
	}
	height := p.ViewportHeight
	if height <= 0 {
		height = DefaultViewportHeight
	}
	timeoutMs := p.HandshakeTimeoutMs
	if timeoutMs <= 0 {
		timeoutMs = DefaultHandshakeTimeout.Milliseconds()
	}

	injectExpr := injectionScript(p.ScenarioURL, signals)

	return map[string]any{
		"metadata": map[string]any{
			"name":        "ui-smoke",
			"description": "test-genie UI smoke: host-iframe embed + iframe-bridge handshake gate + frame screenshot",
		},
		"settings": map[string]any{
			"viewport": map[string]any{
				"width":  width,
				"height": height,
			},
		},
		"nodes": []any{
			node(nodeNavigate, "Navigate host shell", map[string]any{
				"type": "ACTION_TYPE_NAVIGATE",
				"navigate": map[string]any{
					"url":        "about:blank",
					"wait_until": "NAVIGATE_WAIT_EVENT_LOAD",
				},
			}),
			node(nodeInject, "Embed scenario UI + arm handshake listener", map[string]any{
				"type": "ACTION_TYPE_EVALUATE",
				"evaluate": map[string]any{
					"expression": injectExpr,
				},
			}),
			node(nodeHandshake, "Wait for iframe-bridge handshake", map[string]any{
				"type": "ACTION_TYPE_ASSERT",
				"assert": map[string]any{
					"selector":   bridgeReadyMarker,
					"mode":       "ASSERTION_MODE_EXISTS",
					"timeout_ms": timeoutMs,
					"failure_message": "Iframe bridge never signaled ready. " +
						"See: docs/phases/structure/ui-smoke.md#handshake-timeout",
				},
			}),
			node(nodeScreens, "Screenshot embedded UI", map[string]any{
				"type": "ACTION_TYPE_SCREENSHOT",
				"screenshot": map[string]any{
					"selector": hostFrameSelector,
				},
			}),
		},
		"edges": []any{
			edge(nodeNavigate, nodeInject),
			edge(nodeInject, nodeHandshake),
			edge(nodeHandshake, nodeScreens),
		},
	}
}

// node builds one workflow node with a stable id and a labeled action.
func node(id, label string, action map[string]any) map[string]any {
	action["metadata"] = map[string]any{"label": label}
	return map[string]any{
		"id":     id,
		"action": action,
	}
}

// edge builds a linear edge from source to target.
func edge(source, target string) map[string]any {
	return map[string]any{
		"id":     "edge-" + source + "-" + target,
		"source": source,
		"target": target,
		"type":   "WORKFLOW_EDGE_TYPE_SMOOTHSTEP",
	}
}

// injectionScript builds the page-context JS for the evaluate step. It installs
// the host-window observers, embeds the scenario UI in a same-origin host iframe
// shell, and arms a postMessage + property-poll listener that flips the
// [data-smoke-bridge-ready] marker once the bridge child signals ready. The
// marker is what the handshake assert gates on.
func injectionScript(scenarioURL string, signals []string) string {
	urlJSON, _ := json.Marshal(scenarioURL)
	predicate := framePropertyPredicate(signals)
	return fmt.Sprintf(injectionTemplate, string(urlJSON), predicate)
}

// framePropertyPredicate renders a JS boolean expression that is true when any
// configured readiness signal holds against the supplied frame `w` (the iframe's
// contentWindow). Same-origin only; cross-origin access is swallowed by the
// caller's try/catch. This mirrors the historical per-signal check generator.
func framePropertyPredicate(signals []string) string {
	var checks []string
	for _, sig := range signals {
		if c := signalCheck(sig); c != "" {
			checks = append(checks, c)
		}
	}
	if len(checks) == 0 {
		return "false"
	}
	return strings.Join(checks, " || ")
}

// signalCheck renders one readiness-signal expression against frame window `w`.
// It handles three shapes, matching the legacy generator:
//   - simple property: "IFRAME_BRIDGE_READY"            → w.IFRAME_BRIDGE_READY === true
//   - nested property: "IframeBridge.ready"             → w.IframeBridge && w.IframeBridge.ready === true
//   - method call:     "IframeBridge.getState().ready"  → guarded getState() call
func signalCheck(signal string) string {
	signal = strings.TrimSpace(signal)
	if signal == "" {
		return ""
	}
	switch {
	case strings.Contains(signal, "()"):
		parts := strings.Split(signal, ".")
		methodIdx := -1
		for i, p := range parts {
			if strings.HasSuffix(p, "()") {
				methodIdx = i
				break
			}
		}
		if methodIdx < 0 {
			return ""
		}
		method := strings.TrimSuffix(parts[methodIdx], "()")
		tail := strings.Join(parts[methodIdx+1:], ".")
		if methodIdx == 0 {
			return fmt.Sprintf("(typeof w.%s === 'function' && w.%s().%s === true)", method, method, tail)
		}
		objPath := "w." + strings.Join(parts[:methodIdx], ".")
		return fmt.Sprintf("(%s && typeof %s.%s === 'function' && %s.%s().%s === true)",
			objPath, objPath, method, objPath, method, tail)
	case strings.Contains(signal, "."):
		obj := "w." + strings.SplitN(signal, ".", 2)[0]
		return fmt.Sprintf("(%s && w.%s === true)", obj, signal)
	default:
		return fmt.Sprintf("w.%s === true", signal)
	}
}

// injectionTemplate is the page-context script run by the evaluate step.
// Placeholders: %[1]s = JSON-quoted scenario URL, %[2]s = the frame-window
// readiness predicate body (references `w`).
//
// The script is intentionally side-effect-only: BAS's workflow evaluate does not
// reliably return a value, so all observable state is exposed through the DOM
// (the [data-smoke-bridge-ready] marker the handshake assert checks) and through
// the engine's own console/network capture (collected via the timeline).
const injectionTemplate = `(() => {
  var doc = document;
  var target = %[1]s;

  // Mark readiness on the host DOM; the handshake assert gates on this.
  function signalReady() {
    try { doc.documentElement.setAttribute('data-smoke-bridge-ready', '1'); } catch (e) {}
  }

  // The iframe-bridge child posts {v,t:'READY'|'HELLO'} to window.parent.
  window.addEventListener('message', function (ev) {
    var d = ev && ev.data;
    if (d && (d.t === 'READY' || d.t === 'HELLO')) { signalReady(); }
  });

  // Same-origin fallback: poll the configured window-property signals inside the
  // embedded frame. Cross-origin access throws and is swallowed.
  function frameReady(w) {
    try { return (%[2]s); } catch (e) { return false; }
  }

  var style = doc.createElement('style');
  style.textContent = 'html,body{margin:0;padding:0;background:#050505;height:100%%}#ui-smoke-frame{border:0;width:100%%;height:100vh}';
  doc.head.appendChild(style);

  var frame = doc.createElement('iframe');
  frame.id = 'ui-smoke-frame';
  frame.setAttribute('allow', 'clipboard-read; clipboard-write');
  frame.src = target;
  doc.body.appendChild(frame);

  var poll = setInterval(function () {
    var w = null;
    try { w = frame.contentWindow; } catch (e) { w = null; }
    if (w && frameReady(w)) { signalReady(); clearInterval(poll); }
    if (doc.documentElement.hasAttribute('data-smoke-bridge-ready')) { clearInterval(poll); }
  }, 100);
})();`
