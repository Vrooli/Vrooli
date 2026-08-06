package preview

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"html"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/gorilla/mux"

	"react-component-library/internal/components"
	internaldeps "react-component-library/internal/deps"
	"react-component-library/internal/preview"
)

func base64Encode(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }

// React/ReactDOM pin used in the importmap the harness HTML carries.
// Decision rationale and revisit triggers live in docs/RESEARCH.md.
const (
	defaultReactRuntimeVersion = "18.3.1"
)

var (
	reactRuntimeCandidates       = []string{"16.14.0", "17.0.2", "18.2.0", "18.3.1", "19.0.0", "19.1.0"}
	vendoredReactRuntimeVersions = map[string]bool{defaultReactRuntimeVersion: true}
	packageRuntimeCandidatesFor  = installedPackageVersionCandidates
)

// HarnessHandler serves the per-component HTML shell that the host UI
// loads into the live-preview iframe. Inlines the transpiled module so
// one request gives the browser everything it needs to render.
type HarnessHandler struct {
	service    preview.Service
	components components.Service
	logger     *log.Logger
	repoRoot   string
}

func NewHarnessHandler(svc preview.Service, logger *log.Logger) *HarnessHandler {
	return NewHarnessHandlerWithStories(svc, nil, logger)
}

func NewHarnessHandlerWithStories(svc preview.Service, comp components.Service, logger *log.Logger) *HarnessHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &HarnessHandler{service: svc, components: comp, logger: logger, repoRoot: discoverRepoRoot("")}
}

func NewHarnessHandlerWithStoriesAtRoot(svc preview.Service, comp components.Service, logger *log.Logger, repoRoot string) *HarnessHandler {
	if logger == nil {
		logger = log.Default()
	}
	return &HarnessHandler{service: svc, components: comp, logger: logger, repoRoot: discoverRepoRoot(repoRoot)}
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
	componentID, err := h.resolveComponentID(r, id)
	if err != nil {
		writeHarnessError(w, h.logger, id, err)
		return
	}
	story, err := h.resolveStory(r, componentID)
	if err != nil {
		writeHarnessError(w, h.logger, id, err)
		return
	}
	frame := story.Frame
	if strings.EqualFold(strings.TrimSpace(r.URL.Query().Get("frame")), "off") {
		frame = nil
	}
	bundle, err := h.service.GetBundleVersionWithFrame(r.Context(), componentID, strings.TrimSpace(r.URL.Query().Get("version")), frame)
	if err != nil {
		writeHarnessError(w, h.logger, id, err)
		return
	}
	kit := strings.TrimSpace(r.URL.Query().Get("kit"))
	if kit == "" {
		kit = defaultPreviewKit
	}
	css, err := previewDesignSystemCSS(h.repoRoot, kit)
	if err != nil {
		h.logger.Printf("preview.harness design kit %q: %v", kit, err)
		http.Error(w, "preview: requested design kit is unavailable", http.StatusInternalServerError)
		return
	}
	doc := renderHarnessHTML(id, bundle, story, css)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	// Same-origin: the host iframe controls the `src`, and same-origin
	// is required for the future iframe-bridge inspect flow. CSP stays
	// permissive in this dev tool by design.
	if _, err := w.Write([]byte(doc)); err != nil {
		h.logger.Printf("preview.harness write %q: %v", id, err)
	}
}

func (h *HarnessHandler) resolveComponentID(r *http.Request, id string) (string, error) {
	if !strings.Contains(id, ":") || h.components == nil {
		return id, nil
	}
	component, err := h.components.GetByLibraryID(r.Context(), id)
	if err != nil {
		return "", err
	}
	return component.ID, nil
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

type harnessStory struct {
	Name                  string
	Version               string
	DisplayName           string
	Kind                  components.StoryKind
	PropsJSON             string
	ArgsJSON              string
	EnvironmentJSON       string
	EnvironmentSchemaJSON string
	InteractionsJSON      string
	ExpectJSON            string
	Harness               string
	Frame                 *components.StoryFrame
}

// resolveStory uses the indexed story contract as the only harness baseline.
func (h *HarnessHandler) resolveStory(r *http.Request, id string) (harnessStory, error) {
	storyID := strings.TrimSpace(r.URL.Query().Get("story"))
	if storyID == "" || h.components == nil {
		return harnessStory{}, nil
	}
	version := strings.TrimSpace(r.URL.Query().Get("version"))
	stories, err := h.components.ListStories(r.Context(), components.StoryQuery{ComponentID: id, Version: version, Limit: 20})
	if err != nil {
		return harnessStory{}, err
	}
	if len(stories) == 0 {
		return harnessStory{}, fmt.Errorf("preview: story contract not found for component %q", id)
	}
	for _, projected := range stories {
		var contract components.StoryContract
		if err := json.Unmarshal([]byte(projected.ContractJSON), &contract); err != nil {
			return harnessStory{}, fmt.Errorf("preview: decode indexed story contract: %w", err)
		}
		for _, definition := range contract.Stories {
			if definition.ID != storyID {
				continue
			}
			args, err := json.Marshal(definition.Args)
			if err != nil {
				return harnessStory{}, fmt.Errorf("preview: encode story args: %w", err)
			}
			expect, err := json.Marshal(definition.Expect)
			if err != nil {
				return harnessStory{}, fmt.Errorf("preview: encode story expectations: %w", err)
			}
			interactions, err := json.Marshal(definition.Interactions)
			if err != nil {
				return harnessStory{}, fmt.Errorf("preview: encode story interactions: %w", err)
			}
			environment, err := json.Marshal(definition.Environment)
			if err != nil {
				return harnessStory{}, fmt.Errorf("preview: encode story environment: %w", err)
			}
			return harnessStory{Name: definition.ID, Version: projected.Version, DisplayName: definition.Name, Kind: contract.Kind, PropsJSON: string(args), ArgsJSON: projected.ArgsJSON, EnvironmentJSON: string(environment), EnvironmentSchemaJSON: projected.EnvironmentJSON, InteractionsJSON: string(interactions), ExpectJSON: string(expect), Harness: definition.Harness, Frame: components.EffectiveStoryFrame(&contract, &definition)}, nil
		}
	}
	return harnessStory{}, fmt.Errorf("preview: story %q not found for component %q", storyID, id)
}

func renderHarnessHTML(id string, b preview.Bundle, ex harnessStory, designSystemCSS string) string {
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
<meta name="story-id" content="`)
	sb.WriteString(html.EscapeString(ex.Name))
	sb.WriteString(`" />
<style>
`)
	sb.WriteString(designSystemCSS)
	sb.WriteString(`
  html, body { margin: 0; padding: 0; min-height: 100vh; background: var(--color-background); color: var(--color-foreground); font-family: var(--font-sans, ui-sans-serif), system-ui, sans-serif; }
  #root { min-height: 100vh; box-sizing: border-box; background: var(--color-background); }
  .rcl-preview-specimen {
    min-height: 100vh;
    box-sizing: border-box;
    display: grid;
    place-items: center;
    padding: clamp(24px, 6vw, 80px);
    background:
      radial-gradient(circle at 50% 0%, color-mix(in srgb, var(--color-primary) 10%, transparent), transparent 42%),
      linear-gradient(135deg, color-mix(in srgb, var(--color-surface) 54%, transparent), transparent 58%),
      var(--color-background);
  }
  .rcl-preview-well {
    width: min(100%, 760px);
    box-sizing: border-box;
    padding: clamp(24px, 5vw, 56px);
    border: var(--border-hairline) solid color-mix(in srgb, var(--color-border) 72%, transparent);
    border-radius: var(--radius-sheet);
    background: color-mix(in srgb, var(--color-surface) 88%, transparent);
    box-shadow: var(--elev-overlay);
    backdrop-filter: blur(18px);
  }
  .rcl-preview-well__meta {
    margin: 0 0 var(--space-md);
    color: var(--color-muted-foreground);
    font-size: var(--text-label-size);
    font-weight: 700;
    letter-spacing: .08em;
    line-height: var(--text-label-line);
    text-transform: uppercase;
  }
  .rcl-preview-well__content { display: grid; place-items: center; min-height: 96px; }
  .rcl-preview-well__content > * { max-width: 100%; }
  .rcl-preview-fixture {
    display: grid;
    gap: var(--space-sm);
    min-height: 260px;
    box-sizing: border-box;
    padding: var(--space-lg);
    border: var(--border-hairline) solid var(--color-border);
    border-radius: var(--radius-panel);
    background: var(--color-surface-raised);
    color: var(--color-foreground);
  }
  .rcl-preview-fixture__heading { color: var(--color-muted-foreground); font-size: var(--text-label-size); font-weight: 700; letter-spacing: .08em; text-transform: uppercase; }
  .rcl-preview-fixture__rows { display: grid; align-content: start; gap: var(--space-2xs); }
  .rcl-preview-fixture__row { block-size: var(--space-sm); border-radius: var(--radius-control); background: var(--color-surface-muted); }
  .rcl-preview-fixture__row:nth-child(1) { inline-size: 88%; }
  .rcl-preview-fixture__row:nth-child(2) { inline-size: 66%; }
  .rcl-preview-fixture__row:nth-child(3) { inline-size: 76%; }
  [data-frame-asset="navigation.page"] { min-height: 100vh; }
  @media (max-width: 520px) {
    .rcl-preview-specimen { padding: 24px 16px; }
  }
  #preview-importmap-diagnostics,
  #preview-error { padding: 16px; font-family: ui-monospace, SFMono-Regular, monospace; color: #ff8c8c; white-space: pre-wrap; }
</style>
<script type="importmap">
`)
	importMap, importWarnings := buildImportMapJSON(b)
	sb.WriteString(strings.ReplaceAll(importMap, "</script", "<\\/script"))
	sb.WriteString(`
</script>
<script>
// Signal the standard iframe bridge before the preview's module graph loads.
// The component bundle can be cold and take longer than the host's readiness
// budget, but the harness itself is ready to receive bridge/inspect messages.
(() => {
  if (window.parent === window) return;
  window.__vrooliBridgeChildInstalled = true;
  const post = (payload) => { try { window.parent.postMessage(payload, "*"); } catch (e) {} };
  post({ v: 1, t: "HELLO", appId: "react-component-library", title: document.title, caps: ["history", "hash", "title", "deeplink", "screenshot", "shortcuts", "inspect"] });
  // Re-announce READY while the host completes iframe attachment. This keeps
  // cold component imports from turning a valid harness into a timing race.
  const ready = () => post({ v: 1, t: "READY" });
  queueMicrotask(ready);
  [100, 500, 1500].forEach((delay) => setTimeout(ready, delay));
})();
</script>
</head>
<body>
<div id="root" data-testid="component-harness-root" data-experience-surface="component-harness" data-experience-state="loading"></div>
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
<pre id="rcl-story-result" hidden></pre>
<script type="module">
const componentModuleURL = "data:text/javascript;base64,`)
	// Embedded module loaded as a data: URL keeps everything on one
	// request and dodges any need for a second fetch. Base64 is safe
	// for the full ESM payload including non-ASCII characters.
	sb.WriteString(base64Encode(b.JS))
	sb.WriteString(`";
const storyHarnessModuleURL = "data:text/javascript;base64,`)
	sb.WriteString(base64Encode(b.HarnessJS))
	sb.WriteString(`";
const frameModuleURL = "data:text/javascript;base64,`)
	sb.WriteString(base64Encode(b.FrameJS))
	sb.WriteString(`";
const previewStory = {
  name: ` + jsString(ex.Name) + `,
  version: ` + jsString(ex.Version) + `,
  displayName: ` + jsString(ex.DisplayName) + `,
	kind: ` + jsString(string(ex.Kind)) + `,
  props: ` + jsonObjectLiteral(ex.PropsJSON) + `,
	args: ` + jsonObjectLiteral(ex.ArgsJSON) + `,
	environment: ` + jsonObjectLiteral(ex.EnvironmentJSON) + `,
	environmentSchema: ` + jsonObjectLiteral(ex.EnvironmentSchemaJSON) + `,
	interactions: ` + jsonArrayLiteral(ex.InteractionsJSON) + `,
  expect: ` + jsonArrayLiteral(ex.ExpectJSON) + `,
  harness: ` + jsString(ex.Harness) + `,
  frame: ` + frameJSON(ex.Frame) + `,
  fixture: ` + jsonObjectLiteral(b.FixtureJSON) + `,
};
// Resolved-theme bridge: the host owns the app/system decision and posts
// {type:"rcl-resolved-theme", theme:"light"|"dark"}. Stamping the root is
// essential: design-tokens.css deliberately guards its OS media fallback with
// :not([data-resolved-theme=light]), so a body class cannot override OS dark.
window.addEventListener("message", (ev) => {
  const data = ev && ev.data;
  if (!data || data.type !== "rcl-resolved-theme") return;
  const theme = data.theme;
  if (theme !== "light" && theme !== "dark") return;
  document.documentElement.dataset.resolvedTheme = theme;
  document.documentElement.style.colorScheme = theme;
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
})();
const errEl = document.getElementById("preview-error");
const storyResultEl = document.getElementById("rcl-story-result");
const harnessRoot = document.getElementById("root");
const setHarnessState = (state) => {
  if (harnessRoot) harnessRoot.dataset.experienceState = state;
};
const showPreviewError = (message) => {
  setHarnessState("error");
  errEl.hidden = false;
  errEl.textContent = message;
  try {
    parent.postMessage({ type: "preview-error", id: ` + jsString(id) + `, sha256: ` + jsString(b.SHA256) + `, story: previewStory.name || "", version: previewStory.version || "", message }, "*");
  } catch (e) {}
};
const normalizeText = (value) => String(value || "").replace(/\s+/g, " ").trim();
const readableText = (node) => {
  if (!node) return "";
  if (node.nodeType === Node.TEXT_NODE) return node.nodeValue || "";
  if (node.nodeType === Node.ELEMENT_NODE && node.getAttribute("aria-hidden") === "true") return "";
  return Array.from(node.childNodes || []).map(readableText).join(" ");
};
const implicitRole = (el) => {
const tag = (el && el.tagName ? el.tagName.toLowerCase() : "");
  if (tag === "button") return "button";
  if (tag === "img") return "img";
  if (tag === "select") return "combobox";
  if (tag === "textarea") return "textbox";
  if (tag === "input") {
    const type = (el.getAttribute("type") || "text").toLowerCase();
    if (["button", "submit", "reset"].includes(type)) return "button";
    if (["search", "text", "email", "password", "url", "tel"].includes(type)) return "textbox";
  }
  if (tag === "a" && el.hasAttribute("href")) return "link";
  if (tag === "main") return "main";
  if (tag === "form") return "form";
  if (tag === "table") return "table";
  if (/^h[1-6]$/.test(tag)) return "heading";
  return "";
};
const accessibleName = (el) => {
  const associatedLabel = el.id
    ? Array.from(document.querySelectorAll("label")).find((candidate) => candidate.htmlFor === el.id)
    : null;
  return normalizeText(
    el.getAttribute("aria-label") ||
    el.getAttribute("aria-labelledby")?.split(/\s+/).map((id) => readableText(document.getElementById(id))).join(" ") ||
    el.getAttribute("title") ||
    readableText(associatedLabel) ||
    el.value ||
    readableText(el) ||
    ""
  );
};
const elementsByRole = (role) => Array.from(document.querySelectorAll("body *"))
  .filter((el) => (el.getAttribute("role") || implicitRole(el)) === role);
const assertPreviewExpectations = () => {
  const failures = [];
  for (const [index, expectation] of (Array.isArray(previewStory.expect) ? previewStory.expect : []).entries()) {
    const kind = expectation && expectation.kind;
    if (kind === "text") {
      const value = normalizeText(expectation.value);
      if (value && !normalizeText(document.body.textContent).includes(value)) {
        failures.push("expect[" + index + "] text " + value + " not found");
      }
      continue;
    }
    if (kind === "role") {
      const role = expectation.role;
      const name = normalizeText(expectation.name);
      const matches = elementsByRole(role);
      const ok = matches.some((el) => !name || accessibleName(el).includes(name));
      if (!ok) failures.push("expect[" + index + "] role " + role + (name ? " named " + name : "") + " not found");
      continue;
    }
    if (kind === "selector") {
      if (!document.querySelector(expectation.selector || "")) {
        failures.push("expect[" + index + "] selector " + (expectation.selector || "<empty>") + " not found");
      }
      continue;
    }
    if (kind === "attribute") {
      const el = document.querySelector(expectation.selector || "");
      const attr = expectation.name || "";
      const actual = el ? el.getAttribute(attr) : null;
      const expected = expectation.value;
      if (!el || (expected !== undefined && actual !== String(expected))) {
        failures.push("expect[" + index + "] attribute " + attr + " expected " + expected + " got " + actual);
      }
      continue;
    }
    failures.push("expect[" + index + "] unsupported kind " + (kind || "<missing>"));
  }
  if (failures.length > 0) throw new Error(failures.join("; "));
};
const reportStoryResult = (passed, failures) => {
  const result = { passed, failures: Array.isArray(failures) ? failures : [] };
  // The DOM mirror is consumed only by the server-owned headless runner; the
  // normal iframe path continues to receive the typed postMessage below.
  storyResultEl.textContent = JSON.stringify(result);
  storyResultEl.hidden = true;
  parent.postMessage({ type: "rcl-story-result", id: ` + jsString(id) + `, story: previewStory.name || "", version: previewStory.version || "", ...result }, "*");
};
const createNodeFactory = (React, Icons, log) => {
  const resolve = (value) => {
    if (Array.isArray(value)) return value.map(resolve);
    if (!value || typeof value !== "object") return value;
    if (Object.prototype.hasOwnProperty.call(value, "$text")) return String(value.$text ?? "");
    if (Object.prototype.hasOwnProperty.call(value, "$handler")) {
      const name = String(value.$handler || "handler");
      return (...args) => log(name, ...args);
    }
    if (Object.prototype.hasOwnProperty.call(value, "$rowKey")) {
      const field = String(value.$rowKey || "id");
      return (row, index) => String((row && row[field]) ?? index);
    }
    if (Object.prototype.hasOwnProperty.call(value, "$icon")) {
      const Icon = Icons[String(value.$icon)] || Icons.Circle;
      return React.createElement(Icon || "span", { "aria-hidden": true, className: value.className || "h-4 w-4" });
    }
    if (Object.prototype.hasOwnProperty.call(value, "$node")) {
      const type = String(value.$node || "span");
      const props = resolve(value.props || {});
      const children = resolve(value.children || []);
      return React.createElement(type, props, ...(Array.isArray(children) ? children : [children]));
    }
    if (Object.prototype.hasOwnProperty.call(value, "$columns")) {
      return (Array.isArray(value.$columns) ? value.$columns : []).map((column) => ({
        id: column.id,
        header: column.header,
        className: column.className,
        accessor: (row) => column.badge
          ? React.createElement("span", { className: "inline-flex rounded-pill border border-app-border px-2 py-1 text-xs" }, row[column.field])
          : row[column.field],
        sortValue: column.sortable ? (row) => row[column.field] : undefined,
        searchValue: column.searchable ? (row) => String(row[column.field] ?? "") : undefined,
      }));
    }
    if (Object.prototype.hasOwnProperty.call(value, "$filters")) {
      return (Array.isArray(value.$filters) ? value.$filters : []).map((filter) => ({
        id: filter.id,
        label: filter.label,
        predicate: (row) => row[filter.field] === filter.equals,
      }));
    }
    const out = {};
    for (const [key, child] of Object.entries(value)) out[key] = resolve(child);
    return out;
  };
  return resolve;
};
const isRenderableComponent = (value) => (
  typeof value === "function" ||
  (value && typeof value === "object" && typeof value.$$typeof === "symbol")
);
try {
  const [{ createRoot }, Mod, React, Icons, HarnessMod, FrameMod] = await Promise.all([
    import("react-dom/client"),
    import(componentModuleURL),
    import("react"),
    import("lucide-react").catch(() => ({})),
		previewStory.harness ? import(storyHarnessModuleURL) : Promise.resolve({}),
		previewStory.frame ? import(frameModuleURL) : Promise.resolve({}),
  ]);
  const Cmp = isRenderableComponent(Mod.default)
    ? Mod.default
    : Mod[Object.keys(Mod).find(k => isRenderableComponent(Mod[k]))] ?? null;
  const hookEntry = Object.entries(Mod).find(([name, value]) => name.startsWith("use") && typeof value === "function");
  const Frame = Object.values(FrameMod).find((value) => isRenderableComponent(value));
  if (previewStory.kind === "hook" && !hookEntry) {
    showPreviewError("preview: hook file exports no callable use* hook");
  } else if (previewStory.kind !== "hook" && !Cmp) {
    showPreviewError("preview: component file exports neither a default nor a callable named export");
  } else {
    const root = createRoot(document.getElementById("root"));
    const postPreviewEvent = (name, ...args) => {
      const sanitize = (value, depth = 0) => {
        if (depth > 5) return "[depth limit]";
        if (value === null || typeof value === "string" || typeof value === "boolean") return value;
        if (typeof value === "number") return Number.isFinite(value) ? value : "[number]";
        if (Array.isArray(value)) return value.slice(0, 50).map((item) => sanitize(item, depth + 1));
        if (typeof value === "object") { const out = {}; for (const [key, item] of Object.entries(value).slice(0, 50)) out[key] = sanitize(item, depth + 1); return out; }
        return "[" + typeof value + "]";
      };
      parent.postMessage({ type: "rcl-preview-event", id: ` + jsString(id) + `, story: previewStory.name || "", version: previewStory.version || "", name: String(name), args: args.map((value) => sanitize(value)), ts: Date.now() }, "*");
    };
    const resolveProps = createNodeFactory(React, Icons, postPreviewEvent);
    const valueAtPath = (object, path) => path.split(".").reduce((value, key) => value && typeof value === "object" ? value[key] : undefined, object);
    const mergeStoryProps = (base, override) => {
      const result = { ...(base && typeof base === "object" && !Array.isArray(base) ? base : {}) };
      for (const [key, value] of Object.entries(override || {})) {
        result[key] = value && typeof value === "object" && !Array.isArray(value) && result[key] && typeof result[key] === "object" && !Array.isArray(result[key])
          ? mergeStoryProps(result[key], value)
          : value;
      }
      return result;
    };
    const validateOverride = (override) => {
      const fields = Array.isArray(previewStory.args && previewStory.args.fields) ? previewStory.args.fields : [];
      const merged = mergeStoryProps(previewStory.props, override);
      const errors = [];
      for (const field of fields) {
        if (!field || typeof field.path !== "string") continue;
        const value = valueAtPath(merged, field.path);
        if (value === undefined) {
          if (field.required && field.default === undefined) errors.push({ path: field.path, rule: "required", message: "A value is required." });
          continue;
        }
        if (field.kind === "text" && typeof value !== "string") errors.push({ path: field.path, rule: "text", message: "Must be text." });
        if (field.kind === "number" && (typeof value !== "number" || !Number.isFinite(value))) errors.push({ path: field.path, rule: "number", message: "Must be a finite number." });
        if (field.kind === "boolean" && typeof value !== "boolean") errors.push({ path: field.path, rule: "boolean", message: "Must be true or false." });
        if (field.kind === "enum" && !Array.isArray(field.options)) errors.push({ path: field.path, rule: "schema", message: "Enum options are unavailable." });
        if (field.kind === "enum" && Array.isArray(field.options) && !field.options.some((option) => JSON.stringify(option) === JSON.stringify(value))) errors.push({ path: field.path, rule: "enum", message: "Must be one of the declared options." });
        if (typeof field.minimum === "number" && typeof value === "number" && value < field.minimum) errors.push({ path: field.path, rule: "minimum", message: "Below the minimum." });
        if (typeof field.maximum === "number" && typeof value === "number" && value > field.maximum) errors.push({ path: field.path, rule: "maximum", message: "Above the maximum." });
        if (typeof field.minLength === "number" && typeof value === "string" && value.length < field.minLength) errors.push({ path: field.path, rule: "minLength", message: "Too short." });
        if (typeof field.maxLength === "number" && typeof value === "string" && value.length > field.maxLength) errors.push({ path: field.path, rule: "maxLength", message: "Too long." });
      }
      return errors;
    };
    const hookFixture = (hookProps, environment) => {
      const [hookName, Hook] = hookEntry;
      if (hookName === "useEscapeKey") return () => {
        const [outcome, setOutcome] = React.useState("waiting for Escape");
        Hook(hookProps.active !== false, () => setOutcome("escaped"));
        return React.createElement("div", { role: "status", tabIndex: -1, "data-rcl-hook-root": true }, outcome);
      };
      if (hookName === "useFocusTrap") return () => {
        const panelRef = React.useRef(null);
        Hook(hookProps.active !== false, panelRef);
        return React.createElement("div", { ref: panelRef, role: "status", tabIndex: -1, "data-rcl-hook-root": true }, React.createElement("button", null, "First"), React.createElement("button", null, "Last"));
      };
      if (hookName === "useVoiceInput") return () => {
        const fixture = environment && environment.voiceInput || "idle";
        const media = { acquire: () => fixture === "permission-denied" ? Promise.reject(new Error("permission denied")) : Promise.resolve({ stop() {}, onEnded() { return () => {}; } }) };
        const adapter = { connect: async () => {}, stop: () => {} };
        const voice = Hook({ adapter, media, mode: hookProps.mode || (fixture === "timeout" ? "timeout" : "always-on"), timeoutMs: 1 });
		const buttonClass = "min-h-11 min-w-11 rounded-control px-2 py-1";
		return React.createElement("div", { role: "status", "data-rcl-hook-root": true }, React.createElement("button", { type: "button", className: buttonClass, "data-rcl-hook-action": "start", onClick: () => void voice.start() }, "Start"), React.createElement("button", { type: "button", className: buttonClass, "data-rcl-hook-action": "stop", onClick: () => void voice.stop() }, "Stop"), React.createElement("output", null, voice.state));
      };
      throw new Error("preview: no registered fixture for hook " + hookName);
    };
    const validateEnvironment = (environment) => {
      if (!environment || typeof environment !== "object" || Array.isArray(environment)) return ["Environment override must be an object."];
      const declared = Array.isArray(previewStory.environmentSchema && previewStory.environmentSchema.fixtures) ? previewStory.environmentSchema.fixtures : [];
      const failures = [];
      for (const [key, value] of Object.entries(environment)) {
        const fixture = declared.find((candidate) => candidate && candidate.key === key);
        if (!fixture) failures.push(key + ": fixture is not declared.");
        else if (!Array.isArray(fixture.options) || !fixture.options.includes(value)) failures.push(key + ": fixture option is not declared.");
      }
      return failures;
    };
    const wrapStandalone = (subject) => previewStory.kind === "component"
      ? React.createElement(
        "div",
        { className: "rcl-preview-specimen", "data-preview-stage": "specimen" },
        React.createElement(
          "section",
          { className: "rcl-preview-well", "aria-label": "Component specimen" },
          React.createElement("p", { className: "rcl-preview-well__meta", "aria-hidden": "true" }, "Interactive specimen"),
          React.createElement("div", { className: "rcl-preview-well__content" }, subject),
        ),
      )
      : subject;
    const renderPreview = (override, environment = previewStory.environment) => {
      const safeOverride = override && typeof override === "object" && !Array.isArray(override) ? override : {};
      const props = resolveProps(mergeStoryProps(previewStory.props, safeOverride));
      const subject = previewStory.harness
        ? React.createElement(HarnessMod[previewStory.harness], { args: props, log: postPreviewEvent })
        : previewStory.kind === "hook" ? React.createElement(hookFixture(props, environment)) : React.createElement(Cmp, props);
      if (previewStory.frame && Frame) {
        const fixtureRegion = React.createElement(
          "section",
          { "data-frame-region": "fixture", "data-fixture-asset": previewStory.fixture?.asset || "", className: "rcl-preview-fixture", "aria-label": "Data source fixture" },
          React.createElement("span", { className: "rcl-preview-fixture__heading" }, "Live data surface"),
          React.createElement("strong", null, previewStory.fixture?.asset || "Fixture data"),
          React.createElement("div", { className: "rcl-preview-fixture__rows", "aria-hidden": "true" },
            React.createElement("span", { className: "rcl-preview-fixture__row" }),
            React.createElement("span", { className: "rcl-preview-fixture__row" }),
            React.createElement("span", { className: "rcl-preview-fixture__row" }),
          ),
        );
        const regions = { [previewStory.frame.region]: subject, content: fixtureRegion };
        // Catalog frames receive both the named regions and the original
        // region map. Named props keep simple frames such as Page ergonomic;
        // regions preserves the richer contract for frames that need to
        // inspect or iterate over all declared regions.
        root.render(React.createElement(Frame, { ...regions, regions, fixture: previewStory.fixture, children: subject, "data-frame-subject": previewStory.frame.asset }));
        return;
      }
      if (previewStory.harness) {
        const Harness = HarnessMod[previewStory.harness];
        if (typeof Harness !== "function") throw new Error("preview: harness export " + previewStory.harness + " was not found");
        root.render(wrapStandalone(React.createElement(Harness, { args: props, log: postPreviewEvent })));
        return;
      }
      const standaloneSubject = previewStory.kind === "hook"
        ? React.createElement(hookFixture(props, environment))
        : React.createElement(Cmp, props);
      root.render(wrapStandalone(standaloneSubject));
    };
    const locate = (target) => {
      if (!target || typeof target !== "object") return null;
      if (typeof target.selector === "string") return document.querySelector(target.selector);
      if (typeof target.role !== "string") return null;
      const candidates = Array.from(document.querySelectorAll("body *")).filter((candidate) => (candidate.getAttribute("role") || implicitRole(candidate)) === target.role);
      const accessibleName = (candidate) => {
        const labelledBy = candidate.getAttribute("aria-labelledby");
        if (labelledBy) return labelledBy.split(/\s+/).map((id) => readableText(document.getElementById(id))).join(" ").trim();
        const associatedLabel = candidate.id
          ? Array.from(document.querySelectorAll("label")).find((label) => label.htmlFor === candidate.id)
          : null;
        return (candidate.getAttribute("aria-label") || candidate.getAttribute("title") || readableText(associatedLabel) || candidate.value || readableText(candidate) || "").trim();
      };
      return Array.from(candidates).find((candidate) => typeof target.name !== "string" || accessibleName(candidate) === target.name) || null;
    };
    const expectationFailure = (expectation) => {
      const visible = (node) => !!node && !node.hasAttribute("hidden") && getComputedStyle(node).display !== "none" && getComputedStyle(node).visibility !== "hidden";
      let node = null;
      if (expectation.kind === "text") node = Array.from(document.querySelectorAll("body *")).find((candidate) => candidate.textContent && candidate.textContent.includes(expectation.value || ""));
      else if (expectation.kind === "role") node = locate({ role: expectation.role, name: expectation.name });
      else node = locate({ selector: expectation.selector });
	  if (!node && previewStory.kind === "hook" && expectation.kind === "visible") node = document.querySelector("[data-rcl-hook-root]");
      if (expectation.kind === "notVisible") return visible(node) ? "expected target not to be visible" : "";
      if (expectation.kind === "visible" || expectation.kind === "text" || expectation.kind === "role") return visible(node) ? "" : "expected target to be visible";
      if (expectation.kind === "attribute") {
        const attribute = expectation.attribute || expectation.name || "";
        if (!node) return "expected attribute value was not found";
        if (expectation.value === undefined) return node.hasAttribute(attribute) ? "" : "expected attribute value was not found";
        return node.getAttribute(attribute) === expectation.value ? "" : "expected attribute value was not found";
      }
      return "unsupported expectation";
    };
    const runStory = async () => {
      const failures = [];
      // React 18 schedules the initial root commit asynchronously. Interactions
      // must target the mounted specimen, not the scheduling frame following
      // root.render(); this matters especially for custom stateful harnesses.
      await new Promise((resolve) => setTimeout(resolve, 50));
      for (const interaction of previewStory.interactions || []) {
        let target = locate(interaction.target);
        if (interaction.kind === "settle") { await new Promise((resolve) => setTimeout(resolve, 20)); continue; }
        if (interaction.kind === "waitFor") { await new Promise((resolve) => setTimeout(resolve, Math.min(Math.max(Number(interaction.text) || 0, 0), 1000))); continue; }
        if (!target && previewStory.kind === "hook" && interaction.kind === "key") target = window;
        if (!target && previewStory.kind === "hook" && interaction.kind === "click") target = document.querySelector("[data-rcl-hook-action=start]");
        if (!target && previewStory.kind === "hook" && interaction.kind === "focus") target = document.querySelector("[data-rcl-hook-root]");
        if (!target) { failures.push({ kind: "interaction", message: "target not found for " + interaction.kind }); continue; }
        if (interaction.kind === "click") target.click();
        else if (interaction.kind === "focus") target.focus();
        else if (interaction.kind === "blur") target.blur();
        else if (interaction.kind === "key") target.dispatchEvent(new KeyboardEvent("keydown", { key: interaction.text || "", bubbles: true }));
        else if (interaction.kind === "type") {
          const text = interaction.text || "";
          // Bypass React's controlled-input value tracker before dispatching
          // the browser events. A plain assignment updates that tracker, so
          // React can correctly treat the following input/change as a no-op.
          const setter = Object.getOwnPropertyDescriptor(Object.getPrototypeOf(target), "value")?.set;
          if (setter) setter.call(target, text);
          else target.value = text;
          target.dispatchEvent(new Event("input", { bubbles: true }));
          target.dispatchEvent(new Event("change", { bubbles: true }));
        }
      }
      // React 18 may defer the commit past the first animation frame. Story
      // expectations run against the committed specimen, never the scheduling
      // frame that happened to follow root.render().
      // A bounded timer is the fallback for headless Chrome, where animation
      // frames may be throttled even after React has committed the root.
      await new Promise((resolve) => setTimeout(resolve, 50));
      for (const expectation of previewStory.expect || []) {
        const message = expectationFailure(expectation);
        if (message) failures.push({ kind: "expect", expectation, message });
      }
      reportStoryResult(failures.length === 0, failures);
    };
    renderPreview({});
    void runStory().then(() => {
      setHarnessState("ready");
      parent.postMessage({ type: "preview-ready", id: ` + jsString(id) + `, sha256: ` + jsString(b.SHA256) + `, story: previewStory.name || "", version: previewStory.version || "" }, "*");
    }).catch((error) => {
      showPreviewError("preview: story execution failed - " + (error && error.stack || error));
    });
    window.addEventListener("message", (ev) => {
      const data = ev && ev.data;
      if (!data || (data.type !== "rcl-preview-props-override" && data.type !== "rcl-preview-props-reset")) return;
      if (data.componentId !== ` + jsString(id) + ` || (data.story || "") !== (previewStory.name || "") || (data.version || "") !== (previewStory.version || "")) return;
      if (data.type === "rcl-preview-props-override") {
        const override = data.props;
        if (!override || typeof override !== "object" || Array.isArray(override)) {
          parent.postMessage({ type: "rcl-preview-props-error", id: ` + jsString(id) + `, story: previewStory.name || "", version: previewStory.version || "", message: "Props override must be a JSON object." }, "*");
          return;
        }
		const validationErrors = [...validateOverride(override), ...validateEnvironment(data.environment || previewStory.environment).map((message) => ({ path: "environment", message }))];
		if (validationErrors.length > 0) {
		  parent.postMessage({ type: "rcl-preview-props-error", id: ` + jsString(id) + `, story: previewStory.name || "", version: previewStory.version || "", message: validationErrors.map((error) => error.path + ": " + error.message).join(" "), fields: validationErrors }, "*");
		  return;
		}
        renderPreview(override, data.environment || previewStory.environment);
        parent.postMessage({ type: "rcl-preview-props-applied", id: ` + jsString(id) + `, story: previewStory.name || "", version: previewStory.version || "" }, "*");
        return;
      }
      renderPreview({});
      parent.postMessage({ type: "rcl-preview-props-reset", id: ` + jsString(id) + `, story: previewStory.name || "", version: previewStory.version || "" }, "*");
    });
  }
} catch (e) {
  showPreviewError("preview: render failed - " + (e && e.stack || e));
}

</script>
</body>
</html>
`)
	return sb.String()
}

func buildImportMapJSON(b preview.Bundle) (string, []string) {
	reactVersion, reactDOMVersion, warnings := resolveReactRuntimeVersions(b.Dependencies)
	imports := map[string]string{
		"react":                 runtimeURL("react", reactVersion, "", &warnings),
		"react/jsx-runtime":     runtimeURL("react", reactVersion, "jsx-runtime", &warnings),
		"react/jsx-dev-runtime": runtimeURL("react", reactVersion, "jsx-dev-runtime", &warnings),
		"react-dom":             runtimeURL("react-dom", reactDOMVersion, "", &warnings),
		"react-dom/client":      runtimeURL("react-dom", reactDOMVersion, "client", &warnings),
	}
	for _, d := range b.Dependencies {
		name := strings.TrimSpace(d.DepName)
		if name == "" || isReactRuntimeDep(name) {
			continue
		}
		version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, packageRuntimeCandidatesFor(name))
		if !ok {
			warnings = append(warnings, fmt.Sprintf("preview: dependency %q cannot be resolved from declared range %q; package is absent from the governed preview runtime store. Populate with: %s", name, d.VersionRange, previewDependencyPopulateCommand(name, d.VersionRange)))
			continue
		}
		imports[name] = packageRuntimeURL(name, version, "", &warnings)
	}
	if _, ok := imports["lucide-react"]; !ok {
		if version, resolved := internaldeps.ResolveRangeToLatest("^0.424.0", packageRuntimeCandidatesFor("lucide-react")); resolved {
			imports["lucide-react"] = packageRuntimeURL("lucide-react", version, "", &warnings)
		}
	}
	raw, err := json.MarshalIndent(map[string]map[string]string{"imports": imports}, "", "  ")
	if err != nil {
		return `{"imports":{}}`, append(warnings, "preview: failed to encode importmap: "+err.Error())
	}
	return string(raw), warnings
}

func resolveReactRuntimeVersions(declarations []internaldeps.Declaration) (string, string, []string) {
	reactVersion := defaultReactRuntimeVersion
	reactDOMVersion := ""
	var warnings []string
	for _, d := range declarations {
		name := strings.TrimSpace(d.DepName)
		switch name {
		case "react":
			version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, reactRuntimeCandidates)
			if !ok {
				warnings = append(warnings, fmt.Sprintf("preview: cannot pin dependency %q from range %q", name, d.VersionRange))
				continue
			}
			reactVersion = version
		case "react-dom":
			version, ok := internaldeps.ResolveRangeToLatest(d.VersionRange, reactRuntimeCandidates)
			if !ok {
				warnings = append(warnings, fmt.Sprintf("preview: cannot pin dependency %q from range %q", name, d.VersionRange))
				continue
			}
			reactDOMVersion = version
		}
	}
	if reactDOMVersion == "" {
		reactDOMVersion = reactVersion
	}
	return reactVersion, reactDOMVersion, warnings
}

func isReactRuntimeDep(name string) bool {
	return name == "react" || name == "react-dom" || strings.HasPrefix(name, "react/")
}

func runtimeURL(name, version, subpath string, warnings *[]string) string {
	resolvedVersion := strings.TrimSpace(version)
	if !vendoredReactRuntimeVersions[resolvedVersion] {
		*warnings = append(*warnings, fmt.Sprintf("preview: runtime %s@%s is not vendored; using %s", name, resolvedVersion, defaultReactRuntimeVersion))
		resolvedVersion = defaultReactRuntimeVersion
	}
	if subpath != "" {
		return "/preview/runtime/" + name + "@" + resolvedVersion + "/" + subpath + ".js"
	}
	return "/preview/runtime/" + name + "@" + resolvedVersion + "/index.js"
}

func packageRuntimeURL(name, version, subpath string, warnings *[]string) string {
	name = strings.TrimSpace(name)
	version = strings.TrimSpace(version)
	if name == "" || version == "" {
		*warnings = append(*warnings, "preview: dependency runtime URL missing package name or version")
		return ""
	}
	if subpath != "" {
		return "/preview/runtime/npm/" + name + "@" + version + "/" + strings.TrimPrefix(subpath, "/")
	}
	return "/preview/runtime/npm/" + name + "@" + version + "/index.js"
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

func jsonObjectLiteral(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "{}"
	}
	var obj map[string]any
	if err := json.Unmarshal([]byte(raw), &obj); err != nil {
		return "{}"
	}
	normalized, err := json.Marshal(obj)
	if err != nil {
		return "{}"
	}
	return string(normalized)
}

func frameJSON(frame *components.StoryFrame) string {
	if frame == nil {
		return "null"
	}
	raw, err := json.Marshal(frame)
	if err != nil {
		return "null"
	}
	return string(raw)
}

func jsonArrayLiteral(raw string) string {
	if strings.TrimSpace(raw) == "" {
		return "[]"
	}
	var arr []any
	if err := json.Unmarshal([]byte(raw), &arr); err != nil {
		return "[]"
	}
	normalized, err := json.Marshal(arr)
	if err != nil {
		return "[]"
	}
	return string(normalized)
}

const defaultPreviewKit = "vrooli-default"

var previewCSSCache struct {
	sync.Mutex
	key      string
	modified string
	css      string
}

func discoverRepoRoot(explicit string) string {
	candidates := []string{explicit, ".", "../..", "../../..", "../../../..", "../../../../.."}
	for _, candidate := range candidates {
		if strings.TrimSpace(candidate) == "" {
			continue
		}
		path := filepath.Join(candidate, "templates", "design", defaultPreviewKit, "adapters", "react-vite-tailwind", "tokens.css")
		if _, err := os.Stat(path); err == nil {
			return candidate
		}
	}
	return explicit
}

func previewDesignSystemCSS(repoRoot, kit string) (string, error) {
	if kit == "" || strings.Contains(kit, ".") || strings.ContainsAny(kit, `/\\`) {
		return "", fmt.Errorf("invalid design kit %q", kit)
	}
	adapter := filepath.Join(repoRoot, "templates", "design", kit, "adapters", "react-vite-tailwind")
	tokensPath := filepath.Join(adapter, "tokens.css")
	utilitiesPath := filepath.Join(adapter, "preview-utilities.css")
	tokensInfo, err := os.Stat(tokensPath)
	if err != nil {
		return "", fmt.Errorf("tokens stylesheet missing: %w", err)
	}
	utilitiesInfo, err := os.Stat(utilitiesPath)
	if err != nil {
		return "", fmt.Errorf("compiled utility stylesheet missing for %q: %w", kit, err)
	}
	key := kit
	modified := fmt.Sprintf("%d:%d", tokensInfo.ModTime().UnixNano(), utilitiesInfo.ModTime().UnixNano())
	previewCSSCache.Lock()
	defer previewCSSCache.Unlock()
	if previewCSSCache.key == key && previewCSSCache.modified == modified && previewCSSCache.css != "" {
		return previewCSSCache.css, nil
	}
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		return "", fmt.Errorf("read tokens stylesheet: %w", err)
	}
	utilities, err := os.ReadFile(utilitiesPath)
	if err != nil {
		return "", fmt.Errorf("read compiled utility stylesheet: %w", err)
	}
	if len(utilities) == 0 {
		return "", fmt.Errorf("compiled utility stylesheet for %q is empty", kit)
	}
	previewCSSCache.key = key
	previewCSSCache.modified = modified
	previewCSSCache.css = string(tokens) + "\n" + string(utilities)
	return previewCSSCache.css, nil
}
