import { existsSync, readFileSync, writeFileSync } from "node:fs";
import path from "node:path";
import { fileURLToPath } from "node:url";

const scenarioRoot = path.resolve(path.dirname(fileURLToPath(import.meta.url)), "..");
const outputRoot = path.join(scenarioRoot, "bas", "cases", "composition");
const checkOnly = process.argv.includes("--check");

const assets = [
  { slug: "alert-dialog", name: "AlertDialog", version: "2.0.5", role: "alertdialog", selector: '[role="alertdialog"]' },
  { slug: "bottom-sheet", name: "BottomSheet", version: "1.0.6", role: "dialog", selector: '[data-rcl-bottom-sheet-panel], [role="dialog"]', mobileStory: "mobile-open", desktopStory: "desktop-open" },
  { slug: "command-palette", name: "CommandPalette", version: "1.1.4", role: "dialog", selector: "[data-rcl-command-palette-panel]" },
  { slug: "context-menu", name: "ContextMenu", version: "1.1.2", role: "menu", selector: '[role="menu"]', mobileStory: "mobile-open", desktopStory: "desktop-open" },
  { slug: "dialog", name: "Dialog", version: "1.3.5", role: "dialog", selector: '[data-rcl-dialog-panel], [role="dialog"]' },
  { slug: "full-page-drawer", name: "FullPageDrawer", version: "1.0.5", role: "dialog", selector: '[data-rcl-full-page-drawer-panel], [role="dialog"]', mobileStory: "mobile-open", desktopStory: "desktop-open" },
  { slug: "menu", name: "Menu", version: "1.2.2", role: "menu", selector: '[role="menu"]' },
  { slug: "popover", name: "Popover", version: "1.2.3", role: "dialog", selector: "[data-rcl-popover-content]" },
  { slug: "responsive-dialog", name: "ResponsiveDialog", version: "1.0.5", role: "dialog", selector: '[data-rcl-responsive-dialog-panel], [role="dialog"]', mobileStory: "mobile-open", desktopStory: "desktop-open" },
];

const consumers = ["browser-automation-studio", "web-console"];
const viewports = {
  mobile: { width: 390, height: 844 },
  desktop: { width: 1440, height: 900 },
};

const edge = (index, source, target) => ({
  id: `e${index}`,
  source,
  target,
  type: "WORKFLOW_EDGE_TYPE_SMOOTHSTEP",
});

function measureLightExpression(asset, consumer, viewport) {
  return `(() => {
    const surface = document.querySelector(${JSON.stringify(asset.selector)});
    if (!surface) return { ok: false, reason: "surface missing" };
    const rootStyle = getComputedStyle(document.documentElement);
    const style = getComputedStyle(surface);
    const rect = surface.getBoundingClientRect();
    const visible = style.display !== "none" && style.visibility !== "hidden" && rect.width > 0 && rect.height > 0;
    const interactiveSelector = 'button,[href],input,select,textarea,[role="button"],[tabindex]:not([tabindex="-1"])';
    const controls = [...surface.querySelectorAll(interactiveSelector)].filter((element) => {
      const bounds = element.getBoundingClientRect();
      const computed = getComputedStyle(element);
      return bounds.width > 0 && bounds.height > 0 && computed.display !== "none" && computed.visibility !== "hidden";
    });
    const tapMin = parseFloat(rootStyle.getPropertyValue("--tap-target-min")) || 44;
    const undersized = controls.flatMap((element) => {
      const bounds = element.getBoundingClientRect();
      return bounds.width < tapMin || bounds.height < tapMin
        ? [{ tag: element.tagName.toLowerCase(), name: element.getAttribute("aria-label") || element.textContent?.trim().slice(0, 60) || "", width: bounds.width, height: bounds.height }]
        : [];
    });
    const labelledBy = (surface.getAttribute("aria-labelledby") || "").split(/\\s+/).filter(Boolean).map((id) => document.getElementById(id)?.textContent?.trim() || "").filter(Boolean).join(" ");
    const accessibleName = surface.getAttribute("aria-label") || labelledBy || "";
    const expectedRole = ${JSON.stringify(asset.role)};
    const actualRole = surface.getAttribute("role") || "";
    const parseColor = (value) => {
      const match = value.match(/[\\d.]+/g);
      if (!match || match.length < 3) return null;
      return { r: Number(match[0]), g: Number(match[1]), b: Number(match[2]), a: match[3] === undefined ? 1 : Number(match[3]) };
    };
    const opaque = (foreground, background) => foreground.a >= 1 ? foreground : ({
      r: foreground.r * foreground.a + background.r * (1 - foreground.a),
      g: foreground.g * foreground.a + background.g * (1 - foreground.a),
      b: foreground.b * foreground.a + background.b * (1 - foreground.a),
      a: 1,
    });
    const luminance = (color) => {
      const channel = (value) => { const normalized = value / 255; return normalized <= 0.04045 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4; };
      return 0.2126 * channel(color.r) + 0.7152 * channel(color.g) + 0.0722 * channel(color.b);
    };
    const pageBackground = parseColor(getComputedStyle(document.body).backgroundColor) || { r: 255, g: 255, b: 255, a: 1 };
    const background = parseColor(style.backgroundColor);
    const foreground = parseColor(style.color);
    const resolvedBackground = background ? opaque(background, pageBackground) : null;
    const resolvedForeground = foreground && resolvedBackground ? opaque(foreground, resolvedBackground) : null;
    const contrastRatio = resolvedBackground && resolvedForeground
      ? ((Math.max(luminance(resolvedBackground), luminance(resolvedForeground)) + 0.05) / (Math.min(luminance(resolvedBackground), luminance(resolvedForeground)) + 0.05))
      : 0;
    const contrastFloor = Number(document.querySelector('meta[name="consumer-contrast-floor"]')?.content || 0);
    const active = document.activeElement;
    const activeStyle = active instanceof Element ? getComputedStyle(active) : null;
    const focusIndicator = !!activeStyle && surface.contains(active) && (
      (activeStyle.outlineStyle !== "none" && parseFloat(activeStyle.outlineWidth) > 0) ||
      (activeStyle.boxShadow !== "none" && activeStyle.boxShadow !== "")
    );
    const tokenNames = ["--color-foreground", "--color-surface", "--color-surface-raised", "--tap-target-min", "--color-focus"];
    const tokenValues = Object.fromEntries(tokenNames.map((name) => [name, rootStyle.getPropertyValue(name).trim()]));
    const tokensResolved = Object.values(tokenValues).every(Boolean);
    const styleResolved = style.backgroundColor !== "rgba(0, 0, 0, 0)" && style.backgroundColor !== "transparent" && style.color !== "rgba(0, 0, 0, 0)";
    const portalBoundary = !document.querySelector("#root")?.contains(surface);
    const semanticTree = [surface, ...surface.querySelectorAll("[role],button,input,select,textarea,a[href]")].slice(0, 32).map((element) => ({
      role: element.getAttribute("role") || ({ BUTTON: "button", INPUT: "textbox", SELECT: "combobox", TEXTAREA: "textbox", A: "link" }[element.tagName] || "generic"),
      name: element.getAttribute("aria-label") || (element.getAttribute("aria-labelledby") || "").split(/\\s+/).filter(Boolean).map((id) => document.getElementById(id)?.textContent?.trim() || "").filter(Boolean).join(" ") || element.textContent?.trim().replace(/\\s+/g, " ").slice(0, 100) || "",
    }));
    const presentationHost = surface.closest("[data-presentation]") || surface.closest("[data-responsive]") || surface;
    const presentation = presentationHost.getAttribute("data-presentation") || presentationHost.getAttribute("data-responsive") || "fixed";
    const ok = visible && actualRole === expectedRole && accessibleName.length > 0 && undersized.length === 0 && tokensResolved && styleResolved && portalBoundary && contrastFloor > 0 && contrastRatio >= contrastFloor && focusIndicator;
    document.body.dataset.compositionLight = ok ? "yes" : "no";
    document.body.dataset.lightSurface = style.backgroundColor;
    document.body.dataset.lightForeground = style.color;
    return {
      ok, asset: ${JSON.stringify(asset.name)}, consumer: ${JSON.stringify(consumer)}, viewport: [innerWidth, innerHeight], expectedViewport: [${viewport.width}, ${viewport.height}],
      visible, actualRole, expectedRole, accessibleName, controls: controls.length, undersized, tapMin, contrastRatio, contrastFloor,
      focusIndicator, activeName: active instanceof Element ? active.getAttribute("aria-label") || active.textContent?.trim().slice(0, 80) || active.tagName : "",
      tokenValues, tokensResolved, styleResolved, portalBoundary, presentation, semanticTree,
      surfaceStyle: { color: style.color, backgroundColor: style.backgroundColor, position: style.position, zIndex: style.zIndex, borderRadius: style.borderRadius },
    };
  })()`;
}

function measureDarkExpression(asset) {
  return `(() => {
    document.documentElement.classList.add("dark");
    document.documentElement.dataset.resolvedTheme = "dark";
    document.documentElement.style.colorScheme = "dark";
    const surface = document.querySelector(${JSON.stringify(asset.selector)});
    if (!surface) return { ok: false, reason: "surface missing" };
    const style = getComputedStyle(surface);
    const lightBackground = document.body.dataset.lightSurface || "";
    const lightForeground = document.body.dataset.lightForeground || "";
    const darkBackground = style.backgroundColor;
    const darkForeground = style.color;
    const changed = lightBackground !== darkBackground || lightForeground !== darkForeground;
    const ok = document.body.dataset.compositionLight === "yes" && changed;
    document.body.dataset.compositionValid = ok ? "yes" : "no";
    return { ok, changed, lightBackground, darkBackground, lightForeground, darkForeground, viewport: [innerWidth, innerHeight] };
  })()`;
}

function workflow(asset, consumer, viewportName) {
  const viewport = viewports[viewportName];
  const story = viewportName === "mobile" ? asset.mobileStory ?? "default" : asset.desktopStory ?? "default";
  const name = `${asset.slug}-${consumer}-${viewportName}`;
  const nodes = [
    {
      id: "navigate",
      action: {
        type: "ACTION_TYPE_NAVIGATE",
        navigate: {
          destination_type: "NAVIGATE_DESTINATION_TYPE_SCENARIO",
          scenario: "react-component-library",
          scenario_path: `/preview/react-component-library:${asset.name}/harness.html?version=${asset.version}&story=${story}&consumer=${consumer}&theme=light`,
          wait_until: "NAVIGATE_WAIT_EVENT_DOMCONTENTLOADED",
          timeout_ms: 30000,
        },
        metadata: { label: `Open ${asset.name} in ${consumer} ${viewportName} context` },
      },
    },
    {
      id: "wait-surface",
      action: { type: "ACTION_TYPE_WAIT", wait: { selector: asset.selector, state: "WAIT_STATE_VISIBLE", timeout_ms: 15000 }, metadata: { label: "Wait for the overlay surface" } },
      execution_settings: { wait_after_ms: 300 },
    },
    {
      id: "prime-keyboard-focus",
      action: { type: "ACTION_TYPE_KEYBOARD", keyboard: { keys: ["Tab"] }, metadata: { label: "Enter keyboard modality and expose the focus-visible treatment" } },
    },
    {
      id: "measure-light",
      action: { type: "ACTION_TYPE_EVALUATE", evaluate: { expression: measureLightExpression(asset, consumer, viewport), store_result: "lightEvidence" }, metadata: { label: "Emit light-theme semantic, style, token, contrast, focus, portal, target, and viewport evidence" } },
    },
    {
      id: "capture-light",
      action: { type: "ACTION_TYPE_SCREENSHOT", screenshot: { full_page: true }, metadata: { label: "Capture light-theme composition evidence" } },
    },
    {
      id: "focus-last-control",
      action: {
        type: "ACTION_TYPE_EVALUATE",
        evaluate: {
          expression: `(() => { const surface = document.querySelector(${JSON.stringify(asset.selector)}); const controls = [...(surface?.querySelectorAll('button,[href],input,select,textarea,[role="button"],[tabindex]:not([tabindex="-1"])') || [])].filter((element) => { const bounds = element.getBoundingClientRect(); const style = getComputedStyle(element); return bounds.width > 0 && bounds.height > 0 && style.display !== "none" && style.visibility !== "hidden"; }); controls.at(-1)?.focus(); return { count: controls.length, focused: document.activeElement?.getAttribute("aria-label") || document.activeElement?.textContent?.trim().slice(0, 80) || "" }; })()`,
          store_result: "focusCycleStart",
        },
        metadata: { label: "Focus the final control before cycling past the surface boundary" },
      },
    },
    {
      id: "cycle-focus",
      action: { type: "ACTION_TYPE_KEYBOARD", keyboard: { keys: ["Tab"] }, metadata: { label: "Cycle focus past the final control" } },
    },
    {
      id: "measure-focus-containment",
      action: {
        type: "ACTION_TYPE_EVALUATE",
        evaluate: {
          expression: `(() => { const surface = document.querySelector(${JSON.stringify(asset.selector)}); const contained = !!surface?.contains(document.activeElement); document.body.dataset.focusContained = contained ? "yes" : "no"; return { contained, activeRole: document.activeElement?.getAttribute("role") || document.activeElement?.tagName || "", activeName: document.activeElement?.getAttribute("aria-label") || document.activeElement?.textContent?.trim().slice(0, 80) || "" }; })()`,
          store_result: "focusContainmentEvidence",
        },
        metadata: { label: "Emit keyboard focus-containment evidence" },
      },
    },
    {
      id: "assert-focus-containment",
      action: { type: "ACTION_TYPE_ASSERT", assert: { selector: 'body[data-focus-contained="yes"]', mode: "ASSERTION_MODE_EXISTS", timeout_ms: 5000, failure_message: `${asset.name} allowed keyboard focus to escape its open surface in ${consumer} at ${viewport.width}x${viewport.height}.` }, metadata: { label: "Keyboard focus remains in the open surface" } },
    },
    {
      id: "measure-dark",
      action: { type: "ACTION_TYPE_EVALUATE", evaluate: { expression: measureDarkExpression(asset), store_result: "darkEvidence" }, metadata: { label: "Switch theme and emit dark-parity evidence" } },
      execution_settings: { wait_after_ms: 150 },
    },
    {
      id: "assert-composition",
      action: { type: "ACTION_TYPE_ASSERT", assert: { selector: 'body[data-composition-valid="yes"]', mode: "ASSERTION_MODE_EXISTS", timeout_ms: 5000, failure_message: `${asset.name} failed semantic, styling, token, contrast, focus, portal, target, viewport, or theme checks for ${consumer} at ${viewport.width}x${viewport.height}.` }, metadata: { label: "Composition contract holds" } },
    },
    {
      id: "capture-dark",
      action: { type: "ACTION_TYPE_SCREENSHOT", screenshot: { full_page: true }, metadata: { label: "Capture dark-theme composition evidence" } },
    },
  ];

  return {
    metadata: {
      name,
      description: `Renders ${asset.name}@${asset.version} inside ${consumer}'s current compiled CSS and token contract at ${viewport.width}x${viewport.height}; emits semantic-tree, computed-style, token, contrast, focus, portal, target, viewport, trace, and light/dark screenshot evidence.`,
      requirement: "OVERLAY-COMPOSITION",
      version: "2",
      labels: {
        reset: "none",
        surface: "overlay-composition",
        asset: asset.name,
        consumer,
        viewport: viewportName,
        generated_by: "tools/generate-overlay-composition-workflows.mjs",
        requirements_json: '["OVERLAY-COMPOSITION","TC-004","TC-006"]',
      },
      execution_mode: "EXECUTION_MODE_OBSERVER",
    },
    settings: { viewport_width: viewport.width, viewport_height: viewport.height },
    nodes,
    edges: nodes.slice(1).map((node, index) => edge(index + 1, nodes[index].id, node.id)),
  };
}

const drift = [];
for (const asset of assets) {
  for (const consumer of consumers) {
    for (const viewportName of Object.keys(viewports)) {
      const file = path.join(outputRoot, `${asset.slug}-${consumer}-${viewportName}.json`);
      const expected = `${JSON.stringify(workflow(asset, consumer, viewportName), null, 2)}\n`;
      if (checkOnly) {
        if (!existsSync(file) || readFileSync(file, "utf8") !== expected) drift.push(path.relative(scenarioRoot, file));
      } else {
        writeFileSync(file, expected);
      }
    }
  }
}

if (drift.length > 0) {
  throw new Error(`overlay composition workflows are stale:\n${drift.join("\n")}`);
}

console.log(`${checkOnly ? "Checked" : "Generated"} ${assets.length * consumers.length * Object.keys(viewports).length} overlay composition workflows.`);
