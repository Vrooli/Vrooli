package preview

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"strings"

	"github.com/gorilla/mux"

	"react-component-library/internal/components"
	"react-component-library/internal/preview"
)

func base64Encode(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// React/ReactDOM pin used in the importmap the harness HTML carries.
// Decision rationale and revisit triggers live in docs/RESEARCH.md.
const (
	reactPinESMSh    = "https://esm.sh/react@18.3.1?dev"
	reactDOMPinESMSh = "https://esm.sh/react-dom@18.3.1?dev"
	clientPinESMSh   = "https://esm.sh/react-dom@18.3.1/client?dev"
)

// HarnessHandler serves the per-component HTML shell that the host UI
// loads into the live-preview iframe. Inlines the transpiled module so
// one request gives the browser everything it needs to render.
type HarnessHandler struct {
	service preview.Service
	logger  *log.Logger
}

func NewHarnessHandler(svc preview.Service, logger *log.Logger) *HarnessHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &HarnessHandler{service: svc, logger: logger}
}

// ServeHTTP returns the iframe-friendly HTML shell. Status mapping
// mirrors the Connect handler: NotFound → 404, ErrBundle/PathEscape →
// 400, everything else → 500.
func (h *HarnessHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimSpace(mux.Vars(r)["id"])
	if id == "" {
		http.Error(w, "missing component id", http.StatusBadRequest)
		return
	}
	bundle, err := h.service.GetBundle(r.Context(), id)
	if err != nil {
		writeHarnessError(w, h.logger, id, err)
		return
	}
	doc := renderHarnessHTML(id, bundle)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// Same-origin: the host iframe controls the `src`, and same-origin
	// is required for the future iframe-bridge inspect flow. CSP stays
	// permissive in this dev tool by design.
	if _, err := w.Write([]byte(doc)); err != nil {
		h.logger.Printf("preview.harness write %q: %v", id, err)
	}
}

func writeHarnessError(w http.ResponseWriter, logger *log.Logger, id string, err error) {
	var (
		notFound   components.ErrComponentNotFound
		pathEscape components.ErrPathEscape
		bundleErr  preview.ErrBundle
	)
	switch {
	case errors.As(err, &notFound):
		http.Error(w, fmt.Sprintf("preview: component %q not found", id), http.StatusNotFound)
	case errors.As(err, &pathEscape):
		http.Error(w, fmt.Sprintf("preview: source_path escapes root for %q", id), http.StatusBadRequest)
	case errors.As(err, &bundleErr):
		// Render the diagnostic inside the iframe so the user sees the
		// error in-place rather than a blank pane.
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Header().Set("Cache-Control", "no-store")
		_, _ = w.Write([]byte(renderBundleErrorHTML(bundleErr)))
	default:
		logger.Printf("preview.harness internal %q: %v", id, err)
		http.Error(w, "preview: internal error", http.StatusInternalServerError)
	}
}

func renderHarnessHTML(id string, b preview.Bundle) string {
	var sb strings.Builder
	sb.WriteString(`<!doctype html>
<html lang="en">
<head>
<meta charset="utf-8" />
<title>preview: `)
	sb.WriteString(html.EscapeString(b.SourcePath))
	sb.WriteString(`</title>
<meta name="component-id" content="`)
	sb.WriteString(html.EscapeString(id))
	sb.WriteString(`" />
<meta name="bundle-sha256" content="`)
	sb.WriteString(html.EscapeString(b.SHA256))
	sb.WriteString(`" />
<style>
  html, body { margin: 0; padding: 0; min-height: 100vh; background: #0b0d12; color: #f5f7fa; font-family: ui-sans-serif, system-ui, sans-serif; }
  #root { padding: 16px; }
  #preview-importmap-diagnostics,
  #preview-error { padding: 16px; font-family: ui-monospace, SFMono-Regular, monospace; color: #ff8c8c; white-space: pre-wrap; }
</style>
<script type="importmap">
`)
	importMap, importWarnings := buildImportMapJSON(b)
	sb.WriteString(strings.ReplaceAll(importMap, "</script", "<\\/script"))
	sb.WriteString(`
</script>
</head>
<body>
<div id="root"></div>
`)
	if len(importWarnings) > 0 {
		sb.WriteString(`<div id="preview-importmap-diagnostics">`)
		for _, warning := range importWarnings {
			sb.WriteString(html.EscapeString(warning))
			sb.WriteString("\n")
		}
		sb.WriteString(`</div>
`)
	}
	sb.WriteString(`
<div id="preview-error" hidden></div>
<script type="module">
import { createRoot } from "react-dom/client";
import * as Mod from "data:text/javascript;base64,`)
	// Embedded module loaded as a data: URL keeps everything on one
	// request and dodges any need for a second fetch. Base64 is safe
	// for the full ESM payload including non-ASCII characters.
	sb.WriteString(base64Encode(b.JS))
	sb.WriteString(`";
// Color-scheme bridge (req DV-001): the host posts
// {type:"rcl-color-scheme", colorScheme:"system"|"light"|"dark"}; we
// apply CSS color-scheme on :root and toggle a "dark" class on body.
window.addEventListener("message", (ev) => {
  const data = ev && ev.data;
  if (!data || data.type !== "rcl-color-scheme") return;
  const cs = data.colorScheme;
  if (cs !== "system" && cs !== "light" && cs !== "dark") return;
  document.documentElement.style.colorScheme = cs === "system" ? "light dark" : cs;
  document.body.classList.toggle("dark", cs === "dark");
});
// Theme bridge (req TH-003): the host posts
// {type:"rcl-theme-apply", themeId:"<id>", tokens:{"--color-primary":"#..."}}
// and we set each token as a CSS custom property on :root so the
// component re-styles immediately. Tokens with keys missing the
// canonical "--" prefix are ignored.
(() => {
  let appliedTokens = [];
  window.addEventListener("message", (ev) => {
    const data = ev && ev.data;
    if (!data || data.type !== "rcl-theme-apply") return;
    const tokens = (data && data.tokens) || {};
    // Clear tokens from a prior theme so switching never leaves stragglers.
    for (const key of appliedTokens) {
      document.documentElement.style.removeProperty(key);
    }
    appliedTokens = [];
    for (const key of Object.keys(tokens)) {
      if (typeof key !== "string" || !key.startsWith("--")) continue;
      const val = tokens[key];
      if (typeof val !== "string") continue;
      document.documentElement.style.setProperty(key, val);
      appliedTokens.push(key);
    }
    try {
      parent.postMessage({ type: "rcl-theme-applied", themeId: data.themeId || "" }, "*");
    } catch (e) {}
  });
})();
// Inspector (req IS-001..003): minimal implementation of the
// @vrooli/iframe-bridge INSPECT wire protocol. The host sends
// {v:1,t:"INSPECT",cmd:"START"|"STOP"}; we reply with INSPECT_STATE,
// INSPECT_HOVER, and INSPECT_RESULT messages whose payload shape
// matches BridgeInspectHoverPayload / BridgeInspectResultPayload.
(() => {
  let active = false;
  let lastHoverEl = null;
  const post = (payload) => { try { parent.postMessage(payload, "*"); } catch (e) {} };
  const truncate = (s, n) => (s && s.length > n) ? s.slice(0, n) + "…" : s || "";
  const buildSelector = (el) => {
    if (!el || el.nodeType !== 1) return "";
    if (el.id) return "#" + el.id;
    const parts = [el.tagName.toLowerCase()];
    const cls = (el.className && typeof el.className === "string")
      ? el.className.trim().split(/\s+/).filter(Boolean).slice(0, 2)
      : [];
    for (const c of cls) parts.push("." + c);
    return parts.join("");
  };
  const buildMeta = (el) => {
    const tag = el.tagName ? el.tagName.toLowerCase() : "";
    const classes = (el.className && typeof el.className === "string")
      ? el.className.trim().split(/\s+/).filter(Boolean)
      : [];
    const text = (el.textContent || "").trim();
    return {
      tag,
      id: el.id || "",
      classes,
      selector: buildSelector(el),
      label: el.getAttribute ? (el.getAttribute("aria-label") || "") : "",
      ariaLabel: el.getAttribute ? (el.getAttribute("aria-label") || "") : "",
      ariaDescription: el.getAttribute ? (el.getAttribute("aria-description") || "") : "",
      title: el.getAttribute ? (el.getAttribute("title") || "") : "",
      role: el.getAttribute ? (el.getAttribute("role") || "") : "",
      text: truncate(text, 500),
    };
  };
  const buildRect = (el) => {
    const r = el.getBoundingClientRect ? el.getBoundingClientRect() : null;
    if (!r) return null;
    return { x: r.x, y: r.y, width: r.width, height: r.height };
  };
  const buildAncestors = (el) => {
    const out = [];
    let cur = el;
    let depth = 0;
    while (cur && cur.nodeType === 1 && depth < 12 && cur !== document.documentElement) {
      out.push({
        depth,
        tag: cur.tagName.toLowerCase(),
        selector: buildSelector(cur),
        id: cur.id || "",
        classes: (cur.className && typeof cur.className === "string")
          ? cur.className.trim().split(/\s+/).filter(Boolean) : [],
        rect: buildRect(cur),
        documentRect: buildRect(cur),
      });
      cur = cur.parentElement;
      depth++;
    }
    return out;
  };
  const buildPayload = (el) => {
    if (!el) return null;
    return {
      meta: buildMeta(el),
      rect: buildRect(el),
      documentRect: buildRect(el),
      ancestors: buildAncestors(el),
      selectedAncestorIndex: 0,
      pointerType: "mouse",
    };
  };
  const onMove = (ev) => {
    if (!active) return;
    const el = document.elementFromPoint(ev.clientX, ev.clientY);
    if (!el || el === lastHoverEl) return;
    lastHoverEl = el;
    const payload = buildPayload(el);
    if (payload) post({ v: 1, t: "INSPECT_HOVER", payload });
  };
  const onClick = (ev) => {
    if (!active) return;
    ev.preventDefault();
    ev.stopPropagation();
    const el = document.elementFromPoint(ev.clientX, ev.clientY);
    if (!el) return;
    const payload = buildPayload(el);
    if (!payload) return;
    post({ v: 1, t: "INSPECT_RESULT", payload: { ...payload, method: "pointer" } });
    setActive(false, "complete");
  };
  const onKey = (ev) => {
    if (!active || ev.key !== "Escape") return;
    ev.preventDefault();
    setActive(false, "cancel");
  };
  function setActive(next, reason) {
    active = next;
    if (next) {
      document.addEventListener("mousemove", onMove, true);
      document.addEventListener("click", onClick, true);
      document.addEventListener("keydown", onKey, true);
      document.body.style.cursor = "crosshair";
    } else {
      document.removeEventListener("mousemove", onMove, true);
      document.removeEventListener("click", onClick, true);
      document.removeEventListener("keydown", onKey, true);
      document.body.style.cursor = "";
      lastHoverEl = null;
    }
    post({ v: 1, t: "INSPECT_STATE", active, reason: reason || (next ? "start" : "stop") });
  }
  window.addEventListener("message", (ev) => {
    const d = ev && ev.data;
    if (!d || d.v !== 1 || d.t !== "INSPECT") return;
    if (d.cmd === "START") setActive(true, "start");
    else if (d.cmd === "STOP") setActive(false, "stop");
  });
  // Announce capability so the host knows the harness supports inspect.
  post({ v: 1, t: "HELLO", caps: ["inspect"] });
})();
const Cmp = Mod.default ?? Mod[Object.keys(Mod).find(k => typeof Mod[k] === "function")] ?? null;
const errEl = document.getElementById("preview-error");
if (!Cmp) {
  errEl.hidden = false;
  errEl.textContent = "preview: component file exports neither a default nor a callable named export";
} else {
  try {
    const React = await import("react");
    createRoot(document.getElementById("root")).render(React.createElement(Cmp));
    parent.postMessage({ type: "preview-ready", id: ` + jsString(id) + `, sha256: ` + jsString(b.SHA256) + ` }, "*");
  } catch (e) {
    errEl.hidden = false;
    errEl.textContent = "preview: render failed — " + (e && e.stack || e);
  }
}

</script>
</body>
</html>
`)
	return sb.String()
}

func buildImportMapJSON(b preview.Bundle) (string, []string) {
	imports := map[string]string{
		"react":                 reactPinESMSh,
		"react/jsx-runtime":     "https://esm.sh/react@18.3.1/jsx-runtime?dev",
		"react/jsx-dev-runtime": "https://esm.sh/react@18.3.1/jsx-dev-runtime?dev",
		"react-dom":             reactDOMPinESMSh,
		"react-dom/client":      clientPinESMSh,
	}
	var warnings []string
	for _, d := range b.Dependencies {
		name := strings.TrimSpace(d.DepName)
		if name == "" || strings.HasPrefix(name, "react") {
			continue
		}
		version, ok := esmVersionFromRange(d.VersionRange)
		if !ok {
			warnings = append(warnings, fmt.Sprintf("preview: cannot pin dependency %q from range %q", name, d.VersionRange))
			continue
		}
		imports[name] = "https://esm.sh/" + name + "@" + version + "?dev"
	}
	raw, err := json.MarshalIndent(map[string]map[string]string{"imports": imports}, "", "  ")
	if err != nil {
		return `{"imports":{}}`, append(warnings, "preview: failed to encode importmap: "+err.Error())
	}
	return string(raw), warnings
}

func esmVersionFromRange(raw string) (string, bool) {
	v := strings.TrimSpace(raw)
	v = strings.TrimPrefix(v, "^")
	v = strings.TrimPrefix(v, "~")
	v = strings.TrimPrefix(v, ">=")
	v = strings.TrimPrefix(v, "=")
	v = strings.TrimSpace(v)
	if v == "" || v == "*" || strings.ContainsAny(v, " <>|") {
		return "", false
	}
	return v, true
}

func renderBundleErrorHTML(err preview.ErrBundle) string {
	var sb strings.Builder
	sb.WriteString(`<!doctype html>
<html lang="en"><head><meta charset="utf-8" /><title>preview: bundle error</title>
<style>body{margin:0;padding:16px;background:#0b0d12;color:#ff8c8c;font:13px/1.5 ui-monospace,SFMono-Regular,monospace;white-space:pre-wrap;}</style>
</head><body>`)
	sb.WriteString(html.EscapeString(fmt.Sprintf("bundle failed: %s\n\n", err.SourcePath)))
	for _, m := range err.Messages {
		sb.WriteString(html.EscapeString(m))
		sb.WriteString("\n")
	}
	sb.WriteString(`</body></html>`)
	return sb.String()
}

// jsString emits a JS-safe double-quoted string literal. Keeps the
// inline `<script>` literal-free of template-injection risk.
func jsString(s string) string {
	var sb strings.Builder
	sb.WriteByte('"')
	for _, r := range s {
		switch r {
		case '\\':
			sb.WriteString(`\\`)
		case '"':
			sb.WriteString(`\"`)
		case '\n':
			sb.WriteString(`\n`)
		case '\r':
			sb.WriteString(`\r`)
		default:
			if r < 0x20 || r == '<' || r == '>' || r == '&' {
				fmt.Fprintf(&sb, `\u%04x`, r)
			} else {
				sb.WriteRune(r)
			}
		}
	}
	sb.WriteByte('"')
	return sb.String()
}
