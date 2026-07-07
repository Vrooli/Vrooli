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
	nodeArtifacts = "visual-capture-artifacts"
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
//  4. evaluate: best-effort DOM/layout/viewport artifact capture;
//  5. screenshot the host iframe element.
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
			node(nodeArtifacts, "Capture visual health artifacts", map[string]any{
				"type": "ACTION_TYPE_EVALUATE",
				"evaluate": map[string]any{
					"expression":   artifactCaptureScript(),
					"store_result": "visual_artifacts",
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
			edge(nodeHandshake, nodeArtifacts),
			edge(nodeArtifacts, nodeScreens),
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

func artifactCaptureScript() string {
	return artifactCaptureTemplate
}

const artifactCaptureTemplate = `(() => {
  function candidateDocument() {
    var frame = document.querySelector('#ui-smoke-frame');
    if (frame) {
      try {
        if (frame.contentDocument && frame.contentDocument.documentElement) {
          return frame.contentDocument;
        }
      } catch (e) {}
    }
    return document;
  }

  function selectorFor(el) {
    if (!el || !el.tagName) { return ''; }
    if (el.id) { return '#' + el.id; }
    var tag = String(el.tagName).toLowerCase();
    var name = el.getAttribute && el.getAttribute('name');
    if (name) { return tag + '[name="' + name + '"]'; }
    var role = el.getAttribute && el.getAttribute('role');
    if (role) { return tag + '[role="' + role + '"]'; }
    var cls = el.classList && el.classList.length ? Array.prototype.slice.call(el.classList, 0, 2).join('.') : '';
    return cls ? tag + '.' + cls : tag;
  }

  function visibleText(el) {
    var text = (el.innerText || el.textContent || '').trim();
    return text.length > 160 ? text.slice(0, 160) : text;
  }

  function elementRecord(el) {
    var rect = el.getBoundingClientRect();
    var style = window.getComputedStyle(el);
    var tag = String(el.tagName || '').toLowerCase();
    var role = el.getAttribute('role') || '';
    var tabIndex = Number(el.getAttribute('tabindex'));
    var interactive = /^(a|button|input|select|textarea|summary)$/.test(tag) ||
      /^(button|link|checkbox|combobox|menuitem|radio|searchbox|slider|switch|tab|textbox)$/.test(role) ||
      el.isContentEditable || tabIndex >= 0;
    return {
      selector: selectorFor(el),
      tag: tag,
      role: role,
      type: el.getAttribute('type') || '',
      text: visibleText(el),
      interactive: !!interactive,
      contentEditable: !!el.isContentEditable,
      rect: { x: rect.x, y: rect.y, width: rect.width, height: rect.height },
      clientWidth: el.clientWidth,
      clientHeight: el.clientHeight,
      scrollWidth: el.scrollWidth,
      scrollHeight: el.scrollHeight,
      fontSize: parseFloat(style.fontSize) || 0,
      position: style.position,
      overflowX: style.overflowX,
      overflowY: style.overflowY,
      pointerEvents: style.pointerEvents,
      visibility: style.visibility,
      display: style.display,
      opacity: parseFloat(style.opacity || '1'),
      ariaModal: el.getAttribute('aria-modal') === 'true'
    };
  }

  function collectElements(doc) {
    var selector = [
      'a[href]', 'button', 'input', 'select', 'textarea', 'summary',
      '[role]', '[tabindex]', '[contenteditable="true"]',
      'p', 'h1', 'h2', 'h3', 'h4', 'h5', 'h6', 'label', 'span', 'div'
    ].join(',');
    return Array.prototype.slice.call(doc.querySelectorAll(selector), 0, 250).map(elementRecord);
  }

  function metaContent(doc, name) {
    try {
      var el = doc.querySelector('meta[name="' + name + '"]');
      return el ? (el.getAttribute('content') || '').trim() : '';
    } catch (e) {
      return '';
    }
  }

  function readSafeAreaInsets(doc) {
    var win = doc.defaultView || window;
    var probe = doc.createElement('div');
    probe.style.cssText = [
      'position:fixed',
      'left:0',
      'top:0',
      'visibility:hidden',
      'pointer-events:none',
      'padding-top:env(safe-area-inset-top)',
      'padding-right:env(safe-area-inset-right)',
      'padding-bottom:env(safe-area-inset-bottom)',
      'padding-left:env(safe-area-inset-left)'
    ].join(';');
    (doc.body || doc.documentElement).appendChild(probe);
    var style = win.getComputedStyle(probe);
    var out = {
      top: parseFloat(style.paddingTop) || 0,
      right: parseFloat(style.paddingRight) || 0,
      bottom: parseFloat(style.paddingBottom) || 0,
      left: parseFloat(style.paddingLeft) || 0
    };
    probe.remove();
    return out;
  }

  function declaredChrome(doc) {
    var themeColor = metaContent(doc, 'theme-color');
    var statusStyle = metaContent(doc, 'apple-mobile-web-app-status-bar-style');
    var statusBarColor = themeColor;
    if (statusStyle === 'black') { statusBarColor = '#000000'; }
    return {
      themeColor: themeColor,
      statusBarColor: statusBarColor,
      safeAreaColor: themeColor,
      statusBarStyle: statusStyle
    };
  }

  function resourceType(entry) {
    if (entry.initiatorType) { return entry.initiatorType; }
    var name = entry.name || '';
    if (/\.(png|jpe?g|gif|webp|svg|ico)(\?|$)/i.test(name)) { return 'image'; }
    if (/\.(css)(\?|$)/i.test(name)) { return 'stylesheet'; }
    if (/\.(woff2?|ttf|otf)(\?|$)/i.test(name)) { return 'font'; }
    return 'resource';
  }

  var doc = candidateDocument();
  var root = doc.documentElement;
  var body = doc.body || root;
  var viewport = {
    width: window.innerWidth || root.clientWidth || 0,
    height: window.innerHeight || root.clientHeight || 0,
    deviceScaleFactor: window.devicePixelRatio || 1,
    visualViewportWidth: window.visualViewport ? window.visualViewport.width : 0,
    visualViewportHeight: window.visualViewport ? window.visualViewport.height : 0,
    visualViewportScale: window.visualViewport ? window.visualViewport.scale : 0
  };
  var layout = {
    viewport: viewport,
    document: {
      scrollWidth: Math.max(root.scrollWidth || 0, body.scrollWidth || 0),
      scrollHeight: Math.max(root.scrollHeight || 0, body.scrollHeight || 0),
      clientWidth: root.clientWidth || 0,
      clientHeight: root.clientHeight || 0
    },
    chrome: declaredChrome(doc),
    safeArea: readSafeAreaInsets(doc),
    elements: collectElements(doc)
  };
  var network = [];
  try {
    network = performance.getEntriesByType('resource').slice(-200).map(function (entry) {
      return {
        url: entry.name || '',
        method: 'GET',
        resourceType: resourceType(entry),
        status: 0,
        errorText: ''
      };
    });
  } catch (e) {}
  return {
    domHtml: root ? root.outerHTML : '',
    layout: layout,
    viewport: viewport,
    network: network
  };
})();`
