import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  app: {
    root: "app-root",
  },
  scheme: {
    list: "scheme-list",
    createBtn: "create-scheme-btn",
    item: "scheme-item",
  },
  capture: {
    container: "text-capture",
    input: "text-capture-input",
    sendBtn: "text-capture-send",
  },
  canvas: {
    view: "canvas-view",
    node: "canvas-node",
    shortcutHelp: "keyboard-shortcut-help",
  },
  graph: {
    view: "graph-view",
    thoughtTitleInput: "thought-title-input",
    createThoughtBtn: "create-thought-btn",
    linkModeBtn: "link-mode-btn",
    linkModeHint: "link-mode-hint",
    thoughtNode: "thought-node",
    edgeItem: "edge-item",
  },
  viewToggle: {
    canvasBtn: "view-canvas-btn",
    graphBtn: "view-graph-btn",
  },
  provider: {
    status: "provider-status",
  },
  export: {
    btn: "export-btn",
  },
  error: {
    banner: "error-banner",
    retryBtn: "error-retry-btn",
    boundaryFallback: "error-boundary-fallback",
    boundaryRetry: "error-boundary-retry",
  },
  connection: {
    status: "connection-status",
  },
  suggestion: {
    list: "suggestion-list",
    item: "suggestion-item",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  /*
  Example dynamic selectors:
  projects: {
    cardByName: defineDynamicSelector({
      description: 'Project card filtered by name',
      selectorPattern: '[data-testid="project-card"][data-project-name="${name}"]',
      params: { name: { type: 'string' } },
    }),
  },
  */
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
