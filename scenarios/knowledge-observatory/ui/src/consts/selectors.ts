/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. We deliberately model selectors as two
 * declarative maps (one literal, one dynamic) and rely on a small helper to
 * produce the typed `selectors` export plus the manifest consumed by workflow
 * linting. Do not hand-roll selector helpers or change this structure—update the
 * maps below so UI code, automation flows, and the manifest builder all stay in
 * sync across every scenario.
 *
 * ## Auto-Generated Manifest
 *
 * The `selectors.manifest.json` file is automatically generated from this file
 * during the testing process. If you need to add or modify selectors:
 *
 * 1. Update the `literalSelectors` object below for static selectors
 * 2. Update the `dynamicSelectorDefinitions` object for parameterized selectors
 * 3. The manifest will be regenerated automatically when tests run
 *
 * DO NOT manually edit `selectors.manifest.json` - your changes will be overwritten!
 */

// DOC: docs/concepts/ARCHITECTURE.md#ui-surface
type LiteralSelectorTree = { readonly [key: string]: string | LiteralSelectorTree };
type LiteralNode = string | LiteralSelectorTree;

type ParamType = "string" | "number" | "enum";

type ParamDefinition =
  | { readonly type: "string" }
  | { readonly type: "number" }
  | { readonly type: "enum"; readonly values: readonly (string | number)[] };

type ParamSchema = Readonly<Record<string, ParamDefinition>>;

type ParamValueType<T extends ParamDefinition> = T extends { type: "number" }
  ? number
  : T extends { type: "enum"; values: readonly (infer V)[] }
  ? V
  : string;

type ParamValues<P extends ParamSchema | undefined> = P extends ParamSchema
  ? { [K in keyof P]: ParamValueType<P[K]> }
  : Record<string, never>;

interface DynamicSelectorDefinition<P extends ParamSchema | undefined = undefined> {
  readonly kind: "dynamic-selector";
  readonly description: string;
  readonly params?: P;
  readonly testIdPattern?: string;
  readonly selectorPattern?: string;
}

type AnyParamSchema = ParamSchema | undefined;
type AnyDynamicSelectorDefinition = DynamicSelectorDefinition<AnyParamSchema>;

type DynamicSelectorBranch = {
  readonly [key: string]: DynamicSelectorBranch | AnyDynamicSelectorDefinition;
};

type DynamicSelectorTree = DynamicSelectorBranch;

type DynamicSelectorFn<P extends ParamSchema | undefined> = keyof ParamValues<P> extends never
  ? () => string
  : (params: ParamValues<P>) => string;

type DynamicBranchResult<D extends DynamicSelectorTree> = {
  [K in keyof D]: D[K] extends DynamicSelectorDefinition<infer P>
  ? DynamicSelectorFn<P>
  : D[K] extends DynamicSelectorTree
  ? DynamicBranchResult<D[K]>
  : never;
};

type SelectorTreeResult<
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
> = {
  [K in keyof L]: L[K] extends string
    ? string
    : SelectorTreeResult<
        Extract<L[K], LiteralSelectorTree>,
        K extends keyof D ? Extract<D[K], DynamicSelectorTree> : DynamicSelectorTree
      >;
} & (D extends DynamicSelectorTree ? DynamicBranchResult<D> : {});

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match: string, token: string) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const isDynamicDefinition = (value: unknown): value is AnyDynamicSelectorDefinition =>
  isRecord(value) && value.kind === "dynamic-selector";

const normalizeParams = (
  definition: AnyDynamicSelectorDefinition,
  raw: Record<string, string | number>,
  path: string,
) => {
  const schema: ParamSchema = definition.params ?? {};
  const normalized: Record<string, string | number> = {};

  for (const key of Object.keys(schema)) {
    if (!(key in raw)) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    const definitionEntry = schema[key];
    if (!definitionEntry) {
      throw new Error(`Selector '${path}' is missing parameter definition for '${key}'`);
    }
    const value = raw[key];
    if (value === undefined) {
      throw new Error(`Selector '${path}' parameter '${key}' must be provided`);
    }
    if (definitionEntry.type === "number") {
      if (typeof value !== "number") {
        throw new Error(`Selector '${path}' parameter '${key}' must be numeric`);
      }
      normalized[key] = value;
      continue;
    }
    if (definitionEntry.type === "enum") {
      if (!definitionEntry.values.includes(value)) {
        throw new Error(
          `Selector '${path}' parameter '${key}' must be one of: ${definitionEntry.values.join(", ")}`,
        );
      }
      normalized[key] = value;
      continue;
    }
    normalized[key] = value;
  }

  const extras = Object.keys(raw).filter((key) => !(key in schema));
  if (extras.length > 0) {
    throw new Error(`Selector '${path}' received unknown parameter(s): ${extras.join(", ")}`);
  }

  return normalized;
};

const flattenLiteralSelectors = (
  tree: LiteralSelectorTree,
  prefix: string[] = [],
  target: Record<string, { testId: string; selector: string }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const nextPath = [...prefix, key];
    if (typeof value === "string") {
      const manifestKey = nextPath.join(".");
      target[manifestKey] = {
        testId: value,
        selector: toDataTestIdSelector(value),
      };
      continue;
    }
    flattenLiteralSelectors(value, nextPath, target);
  }
  return target;
};

const flattenDynamicSelectors = (
  tree: DynamicSelectorTree,
  prefix: string[] = [],
  target: Record<string, {
    description: string;
    selectorPattern: string;
    testIdPattern?: string;
    params: Array<{ name: string; type: ParamType; values?: readonly (string | number)[] }>;
  }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const nextPath = [...prefix, key];
    if (isDynamicDefinition(value)) {
      const manifestKey = nextPath.join(".");
      const paramEntries = Object.entries(value.params ?? {}) as Array<[string, ParamDefinition]>;
      target[manifestKey] = {
        description: value.description,
        selectorPattern:
          value.selectorPattern ?? (value.testIdPattern ? toDataTestIdSelector(value.testIdPattern) : ""),
        testIdPattern: value.testIdPattern,
        params: paramEntries.map(([name, config]) => ({
          name,
          type: config.type,
          values: config.type === "enum" ? config.values : undefined,
        })),
      };
      continue;
    }
    flattenDynamicSelectors(value, nextPath, target);
  }
  return target;
};

const mergeLiteralAndDynamicNodes = (
  literalNode: LiteralSelectorTree | undefined,
  dynamicNode: DynamicSelectorTree | undefined,
  path: string[] = [],
): Record<string, unknown> => {
  const merged: Record<string, unknown> = {};
  const keys = new Set([
    ...Object.keys(literalNode ?? {}),
    ...Object.keys(dynamicNode ?? {}),
  ]);

  keys.forEach((key) => {
    const literalValue: LiteralNode | undefined = literalNode?.[key];
    const dynamicValue = dynamicNode?.[key];
    const nextPath = [...path, key];

    if (typeof literalValue === "string") {
      merged[key] = literalValue;
      return;
    }

    if (literalValue && typeof literalValue === "object") {
      merged[key] = mergeLiteralAndDynamicNodes(
        literalValue as LiteralSelectorTree,
        isDynamicDefinition(dynamicValue) ? undefined : (dynamicValue as DynamicSelectorTree | undefined),
        nextPath,
      );
      return;
    }

    if (dynamicValue) {
      if (isDynamicDefinition(dynamicValue)) {
        merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
        return;
      }
      merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicValue as DynamicSelectorTree, nextPath);
    }
  });

  return merged;
};

const createDynamicSelectorFn = <P extends ParamSchema | undefined>(
  definition: DynamicSelectorDefinition<P>,
  path: string,
): DynamicSelectorFn<P> => {
  return (params?: ParamValues<P>) => {
    const rawParams: Record<string, string | number> = { ...(params ?? {}) };
    const normalized = normalizeParams(definition, rawParams, path);
    const template = definition.testIdPattern ?? definition.selectorPattern;
    if (!template) {
      throw new Error(`Selector '${path}' is missing both testIdPattern and selectorPattern`);
    }
    return formatTemplate(template, normalized, path);
  };
};

const defineDynamicSelector = <P extends ParamSchema | undefined>(
  definition: Omit<DynamicSelectorDefinition<P>, "kind">,
): DynamicSelectorDefinition<P> => ({
  ...definition,
  kind: "dynamic-selector",
});

const createSelectorRegistry = <
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
>(literalTree: L, dynamicTree: D) => {
  const selectors = mergeLiteralAndDynamicNodes(literalTree, dynamicTree) as SelectorTreeResult<L, D>;
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: flattenDynamicSelectors(dynamicTree),
  };
  return { selectors, manifest };
};

const literalSelectors = {
  header: {
    title: "ko-header-title",
    statusBadge: "ko-status-badge",
    pageTitle: "ko-page-title",
    mobileMenuButton: "ko-header-mobile-menu-button",
    mobileMenuPanel: "ko-header-mobile-menu-panel",
  },
  nav: {
    dashboard: "ko-nav-dashboard",
    search: "ko-nav-search",
    explorer: "ko-nav-explorer",
    viewer: "ko-nav-viewer",
    graph: "ko-nav-graph",
    metrics: "ko-nav-metrics",
  },
  dashboard: {
    quickActions: "ko-dashboard-quick-actions",
    quickSearch: "ko-dashboard-quick-search",
    quickSearchForm: "ko-dashboard-quick-search-form",
    quickSearchMode: "ko-dashboard-quick-search-mode",
    quickSearchInput: "ko-dashboard-quick-search-input",
    quickSearchSubmit: "ko-dashboard-quick-search-submit",
    quickMetrics: "ko-dashboard-quick-metrics",
    quickGraph: "ko-dashboard-quick-graph",
    healthSection: "ko-health-section",
    healthRefresh: "ko-health-refresh",
    healthError: "ko-health-error",
    featureSearch: "ko-feature-search",
    featureExplorer: "ko-feature-explorer",
    featureGraph: "ko-feature-graph",
    featureMetrics: "ko-feature-metrics",
    activityFeed: "ko-dashboard-activity-feed",
    cliSection: "ko-cli-section",
  },
  search: {
    form: "ko-search-form",
    input: "ko-search-input",
    submit: "ko-search-submit",
    clear: "ko-search-clear",
    sampleGroup: "ko-search-samples",
    sampleButton: "ko-search-sample",
    resultsSummary: "ko-search-summary",
    resultsList: "ko-search-results",
    emptyState: "ko-search-empty",
    error: "ko-search-error",
    modeSelector: "ko-search-mode-selector",
    docSearchForm: "ko-doc-search-form",
    docSearchQuery: "ko-doc-search-query",
    docSearchPattern: "ko-doc-search-pattern",
    docSearchScope: "ko-doc-search-scope",
    docSearchScenario: "ko-doc-search-scenario",
    docSearchBasePath: "ko-doc-search-base-path",
    docSearchFileTypes: "ko-doc-search-file-types",
    docSearchContextLines: "ko-doc-search-context-lines",
    docSearchCaseSensitive: "ko-doc-search-case-sensitive",
    docSearchIncludeContent: "ko-doc-search-include-content",
    docSearchUseSemantic: "ko-doc-search-use-semantic",
    docSearchSummary: "ko-doc-search-summary",
    docSearchResults: "ko-doc-search-results",
    docSearchEmpty: "ko-doc-search-empty",
    docSearchError: "ko-doc-search-error",
  },
  deepSearch: {
    form: "ko-deep-search-form",
    input: "ko-deep-search-input",
    submit: "ko-deep-search-submit",
    clear: "ko-deep-search-clear",
    status: "ko-deep-search-status",
    results: "ko-deep-search-results",
    error: "ko-deep-search-error",
  },
  metrics: {
    overall: "ko-metrics-overall",
    refresh: "ko-metrics-refresh",
    legend: "ko-metrics-legend",
    collections: "ko-metrics-collections",
    summary: "ko-metrics-summary",
  },
  graph: {
    container: "ko-graph-container",
    form: "ko-graph-form",
    centerInput: "ko-graph-center-input",
    collectionInput: "ko-graph-collection-input",
    visibilityInput: "ko-graph-visibility-input",
    namespacesInput: "ko-graph-namespaces-input",
    tagsInput: "ko-graph-tags-input",
    depthInput: "ko-graph-depth-input",
    limitInput: "ko-graph-limit-input",
    thresholdInput: "ko-graph-threshold-input",
    submit: "ko-graph-submit",
    clear: "ko-graph-clear",
    refresh: "ko-graph-refresh",
    results: "ko-graph-results",
    canvasPanel: "ko-graph-canvas-panel",
    canvas: "ko-graph-canvas",
    canvasViewport: "ko-graph-canvas-viewport",
    fit: "ko-graph-fit",
    zoomIn: "ko-graph-zoom-in",
    zoomOut: "ko-graph-zoom-out",
    resetViewport: "ko-graph-reset-viewport",
    zoomLabel: "ko-graph-zoom-label",
    minWeightInput: "ko-graph-min-weight-input",
    layoutRadial: "ko-graph-layout-radial",
    layoutForce: "ko-graph-layout-force",
    layoutColumn: "ko-graph-layout-column",
    highlightNeighbors: "ko-graph-highlight-neighbors",
    expandToggle: "ko-graph-expand-toggle",
    expand: "ko-graph-expand",
    truncatedWarning: "ko-graph-truncated-warning",
    legend: "ko-graph-legend",
    details: "ko-graph-details",
    nodePrefix: "ko-graph-node",
    nodes: "ko-graph-nodes",
    edges: "ko-graph-edges",
    error: "ko-graph-error",
    emptyState: "ko-graph-empty",
  },
  explorer: {
    scenarioList: "ko-explorer-scenario-list",
    scenarioFilter: "ko-explorer-scenario-filter",
    docTree: "ko-explorer-doc-tree",
    healthPanel: "ko-explorer-health-panel",
  },
  viewer: {
    pathInput: "ko-viewer-path-input",
    loadButton: "ko-viewer-load-button",
    modeToggle: "ko-viewer-mode-toggle",
    codeView: "ko-viewer-code-view",
    previewView: "ko-viewer-preview-view",
    resetPanel: "ko-viewer-reset-panel",
  },
} as const satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  search: {
    sampleByQuery: defineDynamicSelector({
      description: "Sample query button filtered by query text",
      selectorPattern: '[data-testid="ko-search-sample"][data-query="${query}"]',
      params: { query: { type: "string" } },
    }),
    modeButton: defineDynamicSelector({
      description: "Search mode selector button by mode",
      selectorPattern: '[data-testid="ko-search-mode-selector-${mode}"]',
      params: { mode: { type: "enum", values: ["semantic", "files", "text", "unified", "deep"] } },
    }),
  },
} as const satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
