package preview

import (
	"encoding/base64"
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
		notFound       components.ErrComponentNotFound
		pathEscape     components.ErrPathEscape
		bundleErr      preview.ErrBundle
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
  #preview-error { padding: 16px; font-family: ui-monospace, SFMono-Regular, monospace; color: #ff8c8c; white-space: pre-wrap; }
</style>
<script type="importmap">
{
  "imports": {
    "react": "`)
	sb.WriteString(reactPinESMSh)
	sb.WriteString(`",
    "react/jsx-runtime": "https://esm.sh/react@18.3.1/jsx-runtime?dev",
    "react/jsx-dev-runtime": "https://esm.sh/react@18.3.1/jsx-dev-runtime?dev",
    "react-dom": "`)
	sb.WriteString(reactDOMPinESMSh)
	sb.WriteString(`",
    "react-dom/client": "`)
	sb.WriteString(clientPinESMSh)
	sb.WriteString(`"
  }
}
</script>
</head>
<body>
<div id="root"></div>
<div id="preview-error" hidden></div>
<script type="module">
import { createRoot } from "react-dom/client";
import * as Mod from "data:text/javascript;base64,`)
	// Embedded module loaded as a data: URL keeps everything on one
	// request and dodges any need for a second fetch. Base64 is safe
	// for the full ESM payload including non-ASCII characters.
	sb.WriteString(base64Encode(b.JS))
	sb.WriteString(`";
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
