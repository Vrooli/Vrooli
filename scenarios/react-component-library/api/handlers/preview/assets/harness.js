const componentModuleURL = __COMPONENT_MODULE_URL__;
const storyHarnessModuleURL = __STORY_HARNESS_MODULE_URL__;
const frameModuleURL = __FRAME_MODULE_URL__;
const previewStory = {
  name: __STORY_NAME__,
  version: __STORY_VERSION__,
  displayName: __STORY_DISPLAY_NAME__,
  kind: __STORY_KIND__,
  props: __STORY_PROPS__,
  args: __STORY_ARGS__,
  environment: __STORY_ENVIRONMENT__,
  environmentSchema: __STORY_ENVIRONMENT_SCHEMA__,
  interactions: __STORY_INTERACTIONS__,
  expect: __STORY_EXPECT__,
  composition: __STORY_COMPOSITION__,
  geometry: __STORY_GEOMETRY__,
  mode: __STORY_MODE__,
  slot: __STORY_SLOT__,
  assetKind: __STORY_ASSET_KIND__,
  archetype: __STORY_ARCHETYPE__,
  fixture: __STORY_FIXTURE__,
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
// Route isolation: component navigation is part of the specimen, not a
// request to replace the preview harness with the library application. Keep
// same-origin links and history updates inside this harness document while
// preserving a readable hash route for stories that want to observe it.
(() => {
  const harnessPath = window.location.pathname;
  const routeFor = (raw) => {
    if (raw === undefined || raw === null || raw === "") return null;
    let resolved;
    try { resolved = new URL(String(raw), window.location.href); } catch (e) { return null; }
    if (resolved.origin !== window.location.origin) return null;
    if (resolved.pathname === harnessPath && resolved.hash.startsWith("#/")) return resolved.pathname + resolved.search + resolved.hash;
    if (resolved.pathname === harnessPath && !resolved.search && !resolved.hash) return null;
    return harnessPath + "#" + resolved.pathname + resolved.search + resolved.hash;
  };
  const applyRoute = (raw, replace) => {
    const local = routeFor(raw);
    if (!local) return false;
    const method = replace ? "replaceState" : "pushState";
    window.history[method]({}, "", local);
    window.dispatchEvent(new PopStateEvent("popstate"));
    document.documentElement.dataset.previewRoute = local.slice(local.indexOf("#") + 1);
    return true;
  };
  const nativePushState = window.history.pushState.bind(window.history);
  const nativeReplaceState = window.history.replaceState.bind(window.history);
  window.history.pushState = (state, title, raw) => {
    if (routeFor(raw)) {
      nativePushState(state, title, routeFor(raw));
      window.dispatchEvent(new PopStateEvent("popstate"));
      return;
    }
    nativePushState(state, title, raw);
  };
  window.history.replaceState = (state, title, raw) => {
    if (routeFor(raw)) {
      nativeReplaceState(state, title, routeFor(raw));
      window.dispatchEvent(new PopStateEvent("popstate"));
      return;
    }
    nativeReplaceState(state, title, raw);
  };
  document.addEventListener("click", (event) => {
    if (event.defaultPrevented || event.button !== 0 || event.metaKey || event.ctrlKey || event.shiftKey || event.altKey) return;
    const anchor = event.target && event.target.closest ? event.target.closest("a[href]") : null;
    if (!anchor || anchor.target === "_blank" || !applyRoute(anchor.getAttribute("href"), false)) return;
    event.preventDefault();
    event.stopPropagation();
  }, true);
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
const performanceEntries = [];
const longTasks = [];
let commitCount = 0;
let rerenderCount = 0;
let mountMs = 0;
let mountStartedAt = performance.now();
const captureParams = new URLSearchParams(window.location.search);
const captureTheme = captureParams.get("theme");
const captureMotion = captureParams.get("motion");
const captureSeed = captureParams.get("seed");
const captureMode = captureParams.get("runner") === "1" ? "isolated" : "workbench";
const captureFixtureShape = captureParams.get("fixtureShape") || (String(previewStory.name || "").toLowerCase().includes("failure") ? "failure" : "typical");
document.documentElement.dataset.rclCaptureMode = captureMode;
if (captureMotion === "reduce") {
  document.documentElement.dataset.rclCapture = "deterministic";
  document.documentElement.style.setProperty("--rcl-capture-motion", "reduced");
}
if (captureSeed) {
  document.documentElement.dataset.rclCaptureSeed = captureSeed;
  window.__RCL_CAPTURE_SEED__ = captureSeed;
}
if (captureTheme === "light" || captureTheme === "dark") {
  document.documentElement.dataset.resolvedTheme = captureTheme;
  document.documentElement.style.colorScheme = captureTheme;
}
const setHarnessState = (state) => {
  if (harnessRoot) harnessRoot.dataset.experienceState = state;
  const requestedTheme = captureTheme;
  if (harnessRoot && (requestedTheme === "light" || requestedTheme === "dark")) {
    harnessRoot.dataset.rclTheme = requestedTheme;
  }
};
const showPreviewError = (message) => {
  setHarnessState("error");
  errEl.hidden = false;
  errEl.textContent = message;
  try {
    parent.postMessage({ type: "preview-error", id: __PREVIEW_ID__, sha256: __BUNDLE_SHA256__, story: previewStory.name || "", version: previewStory.version || "", message }, "*");
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
  if (tag === "output") return "status";
  if (tag === "input") {
    const type = (el.getAttribute("type") || "text").toLowerCase();
    if (["button", "submit", "reset"].includes(type)) return "button";
    if (["search", "text", "email", "password", "url", "tel"].includes(type)) return "textbox";
  }
  if (tag === "a" && el.hasAttribute("href")) return "link";
  if (tag === "main") return "main";
  if (tag === "form") return "form";
  if (tag === "table") return "table";
  if (tag === "nav") return "navigation";
  if (tag === "aside") return "complementary";
  if (tag === "section" && el.hasAttribute("aria-label")) return "region";
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
    if (kind === "layout") {
      const el = document.querySelector(expectation.selector || "");
      if (!el) {
        failures.push("expect[" + index + "] layout selector " + (expectation.selector || "<empty>") + " not found");
        continue;
      }
      const rect = el.getBoundingClientRect();
      const checks = [
        ["width", expectation.minWidth, rect.width >= Number(expectation.minWidth)],
        ["height", expectation.minHeight, rect.height >= Number(expectation.minHeight)],
        ["width", expectation.maxWidth, rect.width <= Number(expectation.maxWidth)],
        ["height", expectation.maxHeight, rect.height <= Number(expectation.maxHeight)],
      ];
      for (const [dimension, bound, passed] of checks) {
        if (bound !== undefined && !passed) {
          failures.push("expect[" + index + "] layout " + dimension + "=" + Math.round(rect[dimension]) + " violates bound " + bound);
        }
      }
      if (expectation.noOverflow && el.scrollWidth > el.clientWidth + 1) {
        failures.push("expect[" + index + "] layout overflows horizontally: scrollWidth=" + el.scrollWidth + " clientWidth=" + el.clientWidth);
      }
      continue;
    }
    failures.push("expect[" + index + "] unsupported kind " + (kind || "<missing>"));
  }
  if (failures.length > 0) throw new Error(failures.join("; "));
};
const reportStoryResult = (passed, failures) => {
  performanceEntries.push(...performance.getEntriesByType("measure").map((entry) => ({ name: entry.name, duration: entry.duration })));
  const result = {
    passed,
    failures: Array.isArray(failures) ? failures : [],
    performance: {
      mountMs: mountMs || Math.max(0, performance.now() - mountStartedAt),
      commitCount,
      rerenderCount,
      longTasks: [...longTasks],
      nodeCount: harnessRoot ? harnessRoot.querySelectorAll("*").length : 0,
      measures: performanceEntries,
    },
  };
  if (harnessRoot) harnessRoot.dataset.rclStoryStatus = passed ? "passed" : "failed";
  // The DOM mirror is consumed only by the server-owned headless runner; the
  // normal iframe path continues to receive the typed postMessage below.
  storyResultEl.textContent = JSON.stringify(result);
  storyResultEl.hidden = true;
  parent.postMessage({ type: "rcl-story-result", id: __PREVIEW_ID__, story: previewStory.name || "", version: previewStory.version || "", ...result }, "*");
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
      const childValues = Array.isArray(children) ? children : [children];
      const keyedChildren = childValues.map((child, index) => (
        React.isValidElement(child) && child.key == null
          ? React.cloneElement(child, { key: "rcl-" + type + "-" + index })
          : child
      ));
      return React.createElement(type, props, ...keyedChildren);
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
		previewStory.composition?.specimen || previewStory.composition?.harness ? import(storyHarnessModuleURL) : Promise.resolve({}),
		previewStory.composition?.frame ? import(frameModuleURL) : Promise.resolve({}),
  ]);
  const Cmp = isRenderableComponent(Mod.default)
    ? Mod.default
    : Mod[Object.keys(Mod).find(k => isRenderableComponent(Mod[k]))] ?? null;
  const hookEntry = Object.entries(Mod).find(([name, value]) => name.startsWith("use") && typeof value === "function");
  const Frame = Object.values(FrameMod).find((value) => isRenderableComponent(value));
	if (previewStory.mode === "live" && !previewStory.composition?.specimen && !previewStory.composition?.harness) {
    showPreviewError("preview: live stories must declare an explicit harness");
  } else if (previewStory.kind === "hook" && !hookEntry) {
    showPreviewError("preview: hook file exports no callable use* hook");
  } else if (previewStory.kind !== "hook" && !Cmp) {
    showPreviewError("preview: component file exports neither a default nor a callable named export");
  } else {
    const root = createRoot(document.getElementById("root"));
    try {
      const observer = new PerformanceObserver((list) => {
        for (const entry of list.getEntries()) {
          if (entry.entryType === "longtask") longTasks.push(entry.duration);
        }
      });
      observer.observe({ type: "longtask", buffered: true });
    } catch (e) {}
    const onProfile = (_id, phase, actualDuration) => {
      commitCount += 1;
      if (phase === "update") rerenderCount += 1;
      if (phase === "mount") mountMs += Number(actualDuration) || 0;
    };
    const renderElement = (element) => {
      mountStartedAt = performance.now();
      root.render(React.createElement(React.Profiler, { id: "story", onRender: onProfile }, element));
    };
    const renderSheet = (element, kind) => renderElement(
      React.createElement(
        "div",
        { "data-preview-sheet": kind || "standalone", "data-preview-capture-boundary": "component-sheet" },
        element,
      ),
    );
    const postPreviewEvent = (name, ...args) => {
      const sanitize = (value, depth = 0) => {
        if (depth > 5) return "[depth limit]";
        if (value === null || typeof value === "string" || typeof value === "boolean") return value;
        if (typeof value === "number") return Number.isFinite(value) ? value : "[number]";
        if (Array.isArray(value)) return value.slice(0, 50).map((item) => sanitize(item, depth + 1));
        if (typeof value === "object") { const out = {}; for (const [key, item] of Object.entries(value).slice(0, 50)) out[key] = sanitize(item, depth + 1); return out; }
        return "[" + typeof value + "]";
      };
      parent.postMessage({ type: "rcl-preview-event", id: __PREVIEW_ID__, story: previewStory.name || "", version: previewStory.version || "", name: String(name), args: args.map((value) => sanitize(value)), ts: Date.now() }, "*");
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
      if (hookName === "useFileAttach" || hookName === "useClipboard" || hookName === "useNetworkStatus") return () => {
        const key = hookName === "useFileAttach" ? "fileAttach" : hookName === "useClipboard" ? "clipboard" : "network";
        const state = environment && environment[key] || "idle";
        return React.createElement("div", { role: "status", "data-rcl-hook-root": true }, hookName + ": " + state);
      };
      throw new Error("preview: no registered fixture for hook " + hookName);
    };
    const resolveFixtureContext = (environment) => {
      const declared = Array.isArray(previewStory.environmentSchema && previewStory.environmentSchema.fixtures) ? previewStory.environmentSchema.fixtures : [];
      return Object.fromEntries(declared.map((fixture) => [fixture.key, {
        adapter: fixture.adapter,
        value: environment && environment[fixture.key] || fixture.options && fixture.options[0] || "idle",
      }]));
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
    const wrapStandalone = (subject) => {
      const archetype = previewStory.geometry?.archetype || previewStory.archetype || "pattern";
      if (archetype !== "primitive") {
        const className = "rcl-preview-stage rcl-preview-stage--" + archetype;
        return React.createElement("div", { className, "data-preview-stage": "stage", "data-preview-archetype": archetype }, subject);
      }
      return previewStory.kind === "component"
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
    };
    const renderPreview = (override, environment = previewStory.environment) => {
      const safeOverride = override && typeof override === "object" && !Array.isArray(override) ? override : {};
      const props = resolveProps(mergeStoryProps(previewStory.props, safeOverride));
      if (Array.isArray(props.children)) props.children = React.Children.toArray(props.children);
      const fixtures = resolveFixtureContext(environment);
	      const subject = previewStory.composition?.harness
	        ? React.createElement(HarnessMod[previewStory.composition.harness.export], { subject: Cmp, args: props, config: previewStory.composition.harness.config || {}, environment, fixtures, log: postPreviewEvent })
	        : previewStory.composition?.specimen
	        ? React.createElement(HarnessMod[previewStory.composition.specimen.export], { args: props, environment, fixtures, log: postPreviewEvent })
        : previewStory.kind === "hook" ? React.createElement(hookFixture(props, environment)) : React.createElement(Cmp, props);
      if (previewStory.composition?.frame && Frame) {
        const fixtureFailure = captureFixtureShape === "failure" && (previewStory.fixture?.dataShapes || []).includes("failure");
        const fixtureRegion = React.createElement(
          "section",
          { "data-frame-region": "fixture", "data-fixture-asset": previewStory.fixture?.asset || "", "data-fixture-shape": captureFixtureShape, className: "rcl-preview-fixture", "aria-label": "Data source fixture" },
          React.createElement("span", { className: "rcl-preview-fixture__heading" }, fixtureFailure ? "Fixture failure" : "Live data surface"),
          fixtureFailure
            ? React.createElement("div", { role: "alert", "data-fixture-state": "failure" }, "Fixture data source failed to load")
            : React.createElement(React.Fragment, null,
                React.createElement("strong", null, previewStory.fixture?.asset || "Fixture data"),
                React.createElement("div", { className: "rcl-preview-fixture__rows", "aria-hidden": "true" },
                  React.createElement("span", { className: "rcl-preview-fixture__row" }),
                  React.createElement("span", { className: "rcl-preview-fixture__row" }),
                  React.createElement("span", { className: "rcl-preview-fixture__row" }),
                ),
              ),
        );
        const regions = { [previewStory.composition.frame.region]: subject, content: fixtureRegion };
        // Catalog frames receive both the named regions and the original
        // region map. Named props keep simple frames such as Page ergonomic;
        // regions preserves the richer contract for frames that need to
        // inspect or iterate over all declared regions.
        renderSheet(React.createElement(Frame, { ...regions, regions, fixture: previewStory.fixture, children: subject, "data-frame-subject": previewStory.composition.frame.asset }), "frame");
        return;
      }
	  if (previewStory.composition?.harness) {
		const Harness = HarnessMod[previewStory.composition.harness.export];
		if (typeof Harness !== "function") throw new Error("preview: composition harness export " + previewStory.composition.harness.export + " was not found");
		// A local harness owns its own Preview composition. Do not add the
		// host's legacy specimen well around it: that duplicates the shared
		// PreviewShowcase when a local harness has adopted the foundation and
		// still leaves the capture boundary explicit and host-owned.
		renderSheet(React.createElement(Harness, { subject: Cmp, args: props, config: previewStory.composition.harness.config || {}, environment, fixtures: resolveFixtureContext(environment), log: postPreviewEvent }), "composition-harness");
		return;
	  }
	  if (previewStory.composition?.specimen) {
		const Specimen = HarnessMod[previewStory.composition.specimen.export];
		if (typeof Specimen !== "function") throw new Error("preview: specimen export " + previewStory.composition.specimen.export + " was not found");
		renderSheet(React.createElement(Specimen, { args: props, environment, fixtures: resolveFixtureContext(environment), log: postPreviewEvent }), "specimen");
		return;
	  }
      const standaloneSubject = previewStory.kind === "hook"
        ? React.createElement(hookFixture(props, environment))
        : React.createElement(Cmp, props);
      renderSheet(wrapStandalone(standaloneSubject), "standalone");
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
      if (expectation.kind === "count") {
        if (!expectation.selector) return "count expectation requires a selector";
        const expected = Number(expectation.value);
        if (!Number.isInteger(expected) || expected < 0) return "count expectation requires a non-negative integer value";
        return document.querySelectorAll(expectation.selector).length === expected ? "" : "expected element count was not met";
      }
      if (expectation.kind === "layout") {
        if (!node) return "expected layout target was not found";
        const rect = node.getBoundingClientRect();
        if (expectation.minWidth !== undefined && rect.width < Number(expectation.minWidth)) return "expected layout width minimum was not met";
        if (expectation.minHeight !== undefined && rect.height < Number(expectation.minHeight)) return "expected layout height minimum was not met";
        if (expectation.maxWidth !== undefined && rect.width > Number(expectation.maxWidth)) return "expected layout width maximum was not met";
        if (expectation.maxHeight !== undefined && rect.height > Number(expectation.maxHeight)) return "expected layout height maximum was not met";
        if (expectation.noOverflow && node.scrollWidth > node.clientWidth + 1) return "expected layout target not to overflow horizontally";
        return "";
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
		if (interaction.kind === "waitFor") {
		  const waitText = String(interaction.text || "").trim();
		  const deadline = Date.now() + 2000;
		  while (waitText && Date.now() < deadline) {
		    const visibleText = Array.from(document.querySelectorAll("body *")).some((candidate) => {
		      const style = getComputedStyle(candidate);
		      return candidate.textContent?.includes(waitText) && style.display !== "none" && style.visibility !== "hidden";
		    });
		    if (visibleText) break;
		    await new Promise((resolve) => setTimeout(resolve, 20));
		  }
		  continue;
		}
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
      parent.postMessage({ type: "preview-ready", id: __PREVIEW_ID__, sha256: __BUNDLE_SHA256__, story: previewStory.name || "", version: previewStory.version || "" }, "*");
    }).catch((error) => {
      showPreviewError("preview: story execution failed - " + (error && error.stack || error));
    });
    window.addEventListener("message", (ev) => {
      const data = ev && ev.data;
      if (!data || (data.type !== "rcl-preview-props-override" && data.type !== "rcl-preview-props-reset")) return;
      if (data.componentId !== __PREVIEW_ID__ || (data.story || "") !== (previewStory.name || "") || (data.version || "") !== (previewStory.version || "")) return;
      if (data.type === "rcl-preview-props-override") {
        const override = data.props;
        if (!override || typeof override !== "object" || Array.isArray(override)) {
          parent.postMessage({ type: "rcl-preview-props-error", id: __PREVIEW_ID__, story: previewStory.name || "", version: previewStory.version || "", message: "Props override must be a JSON object." }, "*");
          return;
        }
		const validationErrors = [...validateOverride(override), ...validateEnvironment(data.environment || previewStory.environment).map((message) => ({ path: "environment", message }))];
		if (validationErrors.length > 0) {
		  parent.postMessage({ type: "rcl-preview-props-error", id: __PREVIEW_ID__, story: previewStory.name || "", version: previewStory.version || "", message: validationErrors.map((error) => error.path + ": " + error.message).join(" "), fields: validationErrors }, "*");
		  return;
		}
        renderPreview(override, data.environment || previewStory.environment);
        parent.postMessage({ type: "rcl-preview-props-applied", id: __PREVIEW_ID__, story: previewStory.name || "", version: previewStory.version || "" }, "*");
        return;
      }
      renderPreview({});
      parent.postMessage({ type: "rcl-preview-props-reset", id: __PREVIEW_ID__, story: previewStory.name || "", version: previewStory.version || "" }, "*");
    });
  }
} catch (e) {
  showPreviewError("preview: render failed - " + (e && e.stack || e));
}

