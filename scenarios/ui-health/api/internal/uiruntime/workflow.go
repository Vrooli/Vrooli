package uiruntime

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	urlpkg "net/url"
	"strings"
	"time"
)

// Node IDs for the handshake workflow. The timeline reader locates the
// handshake-gating assert and the frame screenshot by these ids.
const (
	nodeNavigate     = "smoke-navigate-host"
	nodeInject       = "smoke-inject-frame"
	nodeHandshake    = "smoke-assert-handshake"
	nodeReadiness    = "smoke-assert-experience-settled"
	nodeRenderSettle = "smoke-wait-for-first-paint"
	nodeArtifacts    = "visual-capture-artifacts"
	nodeScreens      = "smoke-screenshot-frame"

	// hostFrameSelector is the id of the host iframe element embedding the
	// scenario UI; the screenshot step captures it.
	hostFrameSelector = "#ui-smoke-frame"
	// bridgeReadyMarker is the host-DOM attribute the injected listener sets once
	// the iframe-bridge child signals ready. The handshake assert gates on it.
	bridgeReadyMarker = "[data-smoke-bridge-ready]"

	// defaultHandshakeTimeout bounds how long the assert waits for readiness.
	defaultHandshakeTimeout = 15 * time.Second
	// renderSettleDelay gives a committed React frame time to paint after the
	// bridge handshake. The iframe bridge is initialized during boot, which is
	// intentionally earlier than lazy route content becoming visible.
	renderSettleDelay     = 750 * time.Millisecond
	defaultViewportWidth  = 1280
	defaultViewportHeight = 720
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
//  1. navigate to the scenario origin so the host shell and embedded child are
//     same-origin (required for DOM/layout artifact capture);
//  2. evaluate: install a postMessage listener + same-origin property poll,
//     inject <iframe src=scenarioURL>, and set [data-smoke-bridge-ready] on the
//     host DOM once the bridge child posts READY/HELLO (or a signal poll holds);
//  3. assert [data-smoke-bridge-ready] EXISTS (the hard-fail gate), bounded by
//     timeoutMs — succeeds the moment the marker appears, fails on timeout;
//  4. wait for the first post-handshake render to paint;
//  5. evaluate: best-effort DOM/layout/viewport artifact capture;
//  6. screenshot the host iframe element.
func buildHandshakeWorkflow(scenarioURL string, signals []string, expected []requiredSurface, timeout time.Duration, vw, vh int) map[string]any {
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

	injectExpr := injectionScript(scenarioURL, signals, expected)
	nodes := []any{
		node(nodeNavigate, "Navigate host shell", map[string]any{
			"type":     "ACTION_TYPE_NAVIGATE",
			"navigate": map[string]any{"url": scenarioURL, "wait_until": "NAVIGATE_WAIT_EVENT_LOAD"},
		}),
		node(nodeInject, "Embed scenario UI + arm handshake listener", map[string]any{
			"type":     "ACTION_TYPE_EVALUATE",
			"evaluate": map[string]any{"expression": injectExpr},
		}),
		node(nodeHandshake, "Wait for iframe-bridge handshake", map[string]any{
			"type":   "ACTION_TYPE_ASSERT",
			"assert": map[string]any{"selector": bridgeReadyMarker, "mode": "ASSERTION_MODE_EXISTS", "timeout_ms": timeout.Milliseconds(), "failure_message": "Iframe bridge never signaled ready."},
		}),
	}
	edges := []any{edge(nodeNavigate, nodeInject), edge(nodeInject, nodeHandshake)}
	previous := nodeHandshake
	if len(expected) > 0 {
		nodes = append(nodes, node(nodeReadiness, "Wait for declared experience surfaces to settle", map[string]any{
			"type":   "ACTION_TYPE_ASSERT",
			"assert": map[string]any{"selector": "[data-smoke-experience-settled]", "mode": "ASSERTION_MODE_EXISTS", "timeout_ms": timeout.Milliseconds(), "failure_message": "Declared required experience surfaces did not settle to a terminal state."},
		}))
		edges = append(edges, edge(previous, nodeReadiness))
		previous = nodeReadiness
	}
	nodes = append(nodes, node(nodeRenderSettle, "Wait for first post-handshake paint", map[string]any{
		"type": "ACTION_TYPE_WAIT",
		"wait": map[string]any{"duration_ms": renderSettleDelay.Milliseconds()},
	}))
	edges = append(edges, edge(previous, nodeRenderSettle))
	previous = nodeRenderSettle
	nodes = append(nodes,
		node(nodeArtifacts, "Capture visual health artifacts", map[string]any{"type": "ACTION_TYPE_EVALUATE", "evaluate": map[string]any{"expression": artifactCaptureScript(), "storeResult": "visual_artifacts"}}),
		node(nodeScreens, "Screenshot embedded UI", map[string]any{"type": "ACTION_TYPE_SCREENSHOT", "screenshot": map[string]any{"selector": hostFrameSelector}}),
	)
	edges = append(edges, edge(previous, nodeArtifacts), edge(nodeArtifacts, nodeScreens))

	return map[string]any{
		"metadata": map[string]any{
			"name":        "ui-health-runtime",
			"description": "ui-health runtime/render: host-iframe embed + iframe-bridge handshake gate + frame screenshot",
			"labels": map[string]string{
				// Every profile check owns a fresh execution lease. This prevents a
				// prior profile's browser state from becoming validation evidence.
				"session_reuse_mode":  "fresh",
				"validation_route":    scenarioURL,
				"validation_viewport": fmt.Sprintf("%dx%d", vw, vh),
			},
		},
		"settings": map[string]any{
			"viewport_width":  vw,
			"viewport_height": vh,
		},
		"nodes": nodes,
		"edges": edges,
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
func injectionScript(scenarioURL string, signals []string, expected []requiredSurface) string {
	target, origin, nonce := bridgeTarget(scenarioURL)
	urlJSON, _ := json.Marshal(target)
	originJSON, _ := json.Marshal(origin)
	nonceJSON, _ := json.Marshal(nonce)
	type expectedSurface struct {
		ID     string   `json:"id"`
		Kind   string   `json:"kind"`
		States []string `json:"states"`
	}
	items := make([]expectedSurface, 0, len(expected))
	for _, surface := range expected {
		if !surface.required || strings.TrimSpace(surface.id) == "" {
			continue
		}
		states := make([]string, 0, len(surface.states))
		for state := range surface.states {
			if state != "loading" {
				states = append(states, state)
			}
		}
		items = append(items, expectedSurface{ID: surface.id, Kind: surface.kind, States: states})
	}
	expectedJSON, _ := json.Marshal(items)
	return fmt.Sprintf(injectionTemplate, string(urlJSON), string(originJSON), string(nonceJSON), framePropertyPredicate(signals), string(expectedJSON))
}

// bridgeTarget gives every validation iframe an unguessable, per-invocation
// nonce. The child receives it in its URL and must echo it in READY/HELLO;
// the host never treats an arbitrary same-page postMessage as readiness.
func bridgeTarget(raw string) (target, origin, nonce string) {
	target, origin = raw, ""
	if parsed, err := urlpkg.Parse(raw); err == nil {
		origin = parsed.Scheme + "://" + parsed.Host
		nonce = bridgeNonce()
		query := parsed.Query()
		query.Set("__vrooli_bridge_nonce", nonce)
		parsed.RawQuery = query.Encode()
		target = parsed.String()
	}
	return target, origin, nonce
}

func bridgeNonce() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		// The nonce is a validation correlation token, not an authorization
		// secret. Preserve a non-empty per-call fallback if OS entropy is down.
		return fmt.Sprintf("runtime-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
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

// injectionTemplate is the page-context script. Placeholders: %[1]s = target
// URL, %[2]s = expected origin, %[3]s = nonce, %[4]s = frame readiness
// predicate, %[5]s = declared required surface terminal-state expectations.
// (references `w`).
const injectionTemplate = `(() => {
  var doc = document;
  var target = %[1]s;
  var expectedOrigin = %[2]s;
  var nonce = %[3]s;
  var expectedSurfaces = %[5]s;
  var observedDocument = null;

  function signalReady() {
    try { doc.documentElement.setAttribute('data-smoke-bridge-ready', '1'); } catch (e) {}
  }

  window.addEventListener('message', function (ev) {
    var d = ev && ev.data;
    if (!d || (d.t !== 'READY' && d.t !== 'HELLO')) { return; }
    if (!frame || ev.source !== frame.contentWindow) { return; }
    if (expectedOrigin && ev.origin !== expectedOrigin) { return; }
    if (!nonce || d.nonce !== nonce) { return; }
    signalReady();
  });

  function frameReady(w) {
    try { return (%[4]s); } catch (e) { return false; }
  }

  var style = doc.createElement('style');
  style.textContent = 'html,body{margin:0;padding:0;background:#050505;height:100%%}#ui-smoke-frame{border:0;width:100%%;height:100vh}';
  doc.head.appendChild(style);

  var frame = doc.createElement('iframe');
  frame.id = 'ui-smoke-frame';
  frame.setAttribute('allow', 'clipboard-read; clipboard-write');
  frame.src = target;
  var frameLoaded = false;
  frame.addEventListener('load', function () { frameLoaded = true; });
  doc.body.appendChild(frame);

  function updateExperienceSettlement() {
    if (!expectedSurfaces.length || !frame.contentDocument) { return; }
    var child = frame.contentDocument;
    for (var i = 0; i < expectedSurfaces.length; i++) {
      var expected = expectedSurfaces[i];
      var nodes = child.querySelectorAll('[data-experience-surface]');
      var matched = null;
      for (var j = 0; j < nodes.length; j++) {
        if (nodes[j].getAttribute('data-experience-surface') === expected.id) { matched = nodes[j]; break; }
      }
      if (!matched) { doc.documentElement.removeAttribute('data-smoke-experience-settled'); return; }
      var state = matched.getAttribute('data-experience-state') || '';
      if (expected.states.length && expected.states.indexOf(state) === -1) { doc.documentElement.removeAttribute('data-smoke-experience-settled'); return; }
      if (expected.kind === 'async' && (state === 'loading' || state === 'static' || !state)) { doc.documentElement.removeAttribute('data-smoke-experience-settled'); return; }
      if (expected.kind === 'static' && state !== 'static') { doc.documentElement.removeAttribute('data-smoke-experience-settled'); return; }
    }
    doc.documentElement.setAttribute('data-smoke-experience-settled', '1');
  }

  function observeExperienceSurfaces() {
    var child = null;
    try { child = frame.contentDocument; } catch (e) { child = null; }
    if (!child || !child.documentElement || child === observedDocument) { return; }
    observedDocument = child;
    new MutationObserver(updateExperienceSettlement).observe(child.documentElement, { childList: true, subtree: true, attributes: true, attributeFilter: ['data-experience-surface', 'data-experience-state'] });
    updateExperienceSettlement();
  }

  var poll = setInterval(function () {
    if (!frameLoaded) { return; }
    var w = null;
    try { w = frame.contentWindow; } catch (e) { w = null; }
    if (w && frameReady(w)) { signalReady(); }
    observeExperienceSurfaces();
    updateExperienceSettlement();
    if (doc.documentElement.hasAttribute('data-smoke-bridge-ready') && (!expectedSurfaces.length || doc.documentElement.hasAttribute('data-smoke-experience-settled'))) { clearInterval(poll); }
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

  function inScrollContainer(el) {
    for (var parent = el.parentElement; parent; parent = parent.parentElement) {
      var style = window.getComputedStyle(parent);
      var scrollsY = /(auto|scroll|overlay)/.test(style.overflowY || '');
      if (scrollsY && parent.scrollHeight > parent.clientHeight + 1) { return true; }
    }
    return false;
  }

  function elementRecord(el) {
    var rect = el.getBoundingClientRect();
    var style = window.getComputedStyle(el);
    var tag = String(el.tagName || '').toLowerCase();
    var role = el.getAttribute('role') || '';
    var tabIndexRaw = el.getAttribute('tabindex');
    var tabIndex = tabIndexRaw === null ? -1 : Number(tabIndexRaw);
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
      ariaModal: el.getAttribute('aria-modal') === 'true',
      inScrollContainer: inScrollContainer(el)
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

  // Shell readiness only proves the iframe booted. Semantic surfaces are
  // collected separately so adopted experience contracts can validate
  // functional lifecycle without treating every DOM primitive as a surface.
  function experienceSurfaces(doc) {
    return Array.prototype.slice.call(doc.querySelectorAll('[data-experience-surface]')).map(function (el) {
      var rect = el.getBoundingClientRect();
      var style = window.getComputedStyle(el);
      return {
        id: el.getAttribute('data-experience-surface') || '',
        state: el.getAttribute('data-experience-state') || '',
        visible: !!(rect.width || rect.height) && style.display !== 'none' && style.visibility !== 'hidden',
        role: el.getAttribute('role') || ''
      };
    });
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
    elements: collectElements(doc),
    experienceSurfaces: experienceSurfaces(doc)
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
