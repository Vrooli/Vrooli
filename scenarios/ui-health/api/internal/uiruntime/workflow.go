package uiruntime

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// Node IDs for the handshake workflow. The timeline reader locates the
// handshake-gating assert and the frame screenshot by these ids.
const (
	nodeNavigate  = "smoke-navigate-host"
	nodeInject    = "smoke-inject-frame"
	nodeHandshake = "smoke-assert-handshake"
	nodeScreens   = "smoke-screenshot-frame"

	// hostFrameSelector is the id of the host iframe element embedding the
	// scenario UI; the screenshot step captures it.
	hostFrameSelector = "#ui-smoke-frame"
	// bridgeReadyMarker is the host-DOM attribute the injected listener sets once
	// the iframe-bridge child signals ready. The handshake assert gates on it.
	bridgeReadyMarker = "[data-smoke-bridge-ready]"

	// defaultHandshakeTimeout bounds how long the assert waits for readiness.
	defaultHandshakeTimeout = 15 * time.Second
	defaultViewportWidth    = 1280
	defaultViewportHeight   = 720
)

// defaultHandshakeSignals are the window-property paths polled, inside the
// embedded frame, for the iframe-bridge readiness signal. They mirror the
// historical smoke contract's KnownSignals.
var defaultHandshakeSignals = []string{
	"__vrooliBridgeChildInstalled",
	"IFRAME_BRIDGE_READY",
	"IframeBridge.ready",
	"iframeBridge.ready",
	"IframeBridge.getState().ready",
}

// buildHandshakeWorkflow authors the inline handshake workflow as a BAS
// workflow-definition map (proto-JSON node graph), reproducing test-genie
// smoke's proven shape:
//
//  1. navigate to about:blank (the host shell);
//  2. evaluate: install a postMessage listener + same-origin property poll,
//     inject <iframe src=scenarioURL>, and set [data-smoke-bridge-ready] on the
//     host DOM once the bridge child posts READY/HELLO (or a signal poll holds);
//  3. assert [data-smoke-bridge-ready] EXISTS (the hard-fail gate), bounded by
//     timeoutMs — succeeds the moment the marker appears, fails on timeout;
//  4. screenshot the host iframe element.
func buildHandshakeWorkflow(scenarioURL string, signals []string, timeout time.Duration, vw, vh int) map[string]any {
	if len(signals) == 0 {
		signals = defaultHandshakeSignals
	}
	if vw <= 0 {
		vw = defaultViewportWidth
	}
	if vh <= 0 {
		vh = defaultViewportHeight
	}
	if timeout <= 0 {
		timeout = defaultHandshakeTimeout
	}

	injectExpr := injectionScript(scenarioURL, signals)

	return map[string]any{
		"metadata": map[string]any{
			"name":        "ui-health-runtime",
			"description": "ui-health runtime/render: host-iframe embed + iframe-bridge handshake gate + frame screenshot",
		},
		"settings": map[string]any{
			"viewport": map[string]any{"width": vw, "height": vh},
		},
		"nodes": []any{
			node(nodeNavigate, "Navigate host shell", map[string]any{
				"type":     "ACTION_TYPE_NAVIGATE",
				"navigate": map[string]any{"url": "about:blank", "wait_until": "NAVIGATE_WAIT_EVENT_LOAD"},
			}),
			node(nodeInject, "Embed scenario UI + arm handshake listener", map[string]any{
				"type":     "ACTION_TYPE_EVALUATE",
				"evaluate": map[string]any{"expression": injectExpr},
			}),
			node(nodeHandshake, "Wait for iframe-bridge handshake", map[string]any{
				"type": "ACTION_TYPE_ASSERT",
				"assert": map[string]any{
					"selector":        bridgeReadyMarker,
					"mode":            "ASSERTION_MODE_EXISTS",
					"timeout_ms":      timeout.Milliseconds(),
					"failure_message": "Iframe bridge never signaled ready.",
				},
			}),
			node(nodeScreens, "Screenshot embedded UI", map[string]any{
				"type":       "ACTION_TYPE_SCREENSHOT",
				"screenshot": map[string]any{"selector": hostFrameSelector},
			}),
		},
		"edges": []any{
			edge(nodeNavigate, nodeInject),
			edge(nodeInject, nodeHandshake),
			edge(nodeHandshake, nodeScreens),
		},
	}
}

func node(id, label string, action map[string]any) map[string]any {
	action["metadata"] = map[string]any{"label": label}
	return map[string]any{"id": id, "action": action}
}

func edge(source, target string) map[string]any {
	return map[string]any{
		"id":     "edge-" + source + "-" + target,
		"source": source,
		"target": target,
		"type":   "WORKFLOW_EDGE_TYPE_SMOOTHSTEP",
	}
}

// injectionScript builds the page-context JS for the evaluate step: it arms a
// postMessage + property-poll listener and embeds the scenario UI in a host
// iframe shell, flipping [data-smoke-bridge-ready] once the bridge child signals
// ready. The marker is what the handshake assert gates on. Side-effect only —
// all observable state is exposed through the DOM marker (BAS's evaluate does
// not reliably return a value).
func injectionScript(scenarioURL string, signals []string) string {
	urlJSON, _ := json.Marshal(scenarioURL)
	return fmt.Sprintf(injectionTemplate, string(urlJSON), framePropertyPredicate(signals))
}

// framePropertyPredicate renders a JS boolean expression true when any readiness
// signal holds against the frame window `w`. Same-origin only; cross-origin
// access throws and is swallowed by the caller's try/catch.
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

// signalCheck renders one readiness-signal expression against frame window `w`,
// handling simple property, nested property, and guarded method-call shapes.
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

// injectionTemplate is the page-context script. Placeholders: %[1]s =
// JSON-quoted scenario URL, %[2]s = the frame-window readiness predicate body
// (references `w`).
const injectionTemplate = `(() => {
  var doc = document;
  var target = %[1]s;

  function signalReady() {
    try { doc.documentElement.setAttribute('data-smoke-bridge-ready', '1'); } catch (e) {}
  }

  window.addEventListener('message', function (ev) {
    var d = ev && ev.data;
    if (d && (d.t === 'READY' || d.t === 'HELLO')) { signalReady(); }
  });

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
