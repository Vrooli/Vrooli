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

type DynamicSelectorBranch = {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- Recursive type requires any for flexibility
  readonly [key: string]: DynamicSelectorBranch | DynamicSelectorDefinition<any>;
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
   
} & (D extends DynamicSelectorTree ? DynamicBranchResult<D> : Record<string, never>);

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match, token) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- Type guard requires any for flexibility
const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<any> =>
  Boolean(value && typeof value === "object" && (value as DynamicSelectorDefinition<ParamSchema>).kind === "dynamic-selector");

 
const normalizeParams = (
  definition: DynamicSelectorDefinition<ParamSchema | undefined>,
  raw: Record<string, string | number>,
  path: string,
) => {
  const schema = definition.params ?? ({} as ParamSchema);
  const normalized: Record<string, string | number> = {};

  for (const key of Object.keys(schema)) {
    if (!(key in raw)) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    const definitionEntry = schema[key];
    const value = raw[key];
    // Both are guaranteed by the key iteration and in-check above, but TypeScript
    // needs explicit guards with noUncheckedIndexedAccess
    if (!definitionEntry || value === undefined) {
      throw new Error(`Selector '${path}' has invalid schema or missing value for '${key}'`);
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

 
const createDynamicSelectorFn = (
  definition: DynamicSelectorDefinition<ParamSchema | undefined>,
  path: string,
) => {
  return (params?: Record<string, string | number>) => {
    const normalized = normalizeParams(definition, params ?? {}, path);
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

const literalSelectors: LiteralSelectorTree = {
  // Dashboard page selectors
  dashboard: {
    statsGrid: "dashboard-stats-grid",
    timelineSection: "dashboard-timeline-section",
    domainsSection: "dashboard-domains-section",
    eventsSection: "dashboard-events-section",
  },
  // Timeline chart selectors [REQ:LD-UI-TRENDS]
  timeline: {
    chart: "timeline-chart",
    empty: "timeline-empty",
    bars: "timeline-bars",
    periodSelector: "timeline-period-selector",
  },
  // Domain list selectors [REQ:LD-UI-DOMAINS]
  domains: {
    list: "domains-list",
    card: "domain-card",
    statusBadge: "domain-status-badge",
  },
  // Events list selectors
  events: {
    list: "events-list",
    row: "event-row",
  },
  // Stats selectors
  stats: {
    totalEvents: "stat-total-events",
    activeDomains: "stat-active-domains",
    lastActivity: "stat-last-activity",
    trend: "stat-7day-trend",
  },
  // Lifestyle Score [REQ:LD-UI-SCORE]
  score: {
    card: "lifestyle-score",
    value: "lifestyle-score-value",
  },
  // Error display
  error: {
    alert: "error-alert",
    retryButton: "error-retry-button",
  },
};

const dynamicSelectorDefinitions: DynamicSelectorTree = {
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
};

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
// Export defineDynamicSelector for use when adding dynamic selectors
export { defineDynamicSelector };

/**
 * Flat data-testid constants for direct use in components.
 * This provides a simpler API than the nested selectors object.
 * [REQ:LD-UI-TRENDS] [REQ:LD-UI-DOMAINS]
 */
export const DATA_SELECTORS = {
  // Dashboard
  DASHBOARD_STATS_GRID: "dashboard-stats-grid",
  DASHBOARD_TIMELINE_SECTION: "dashboard-timeline-section",
  DASHBOARD_DOMAINS_SECTION: "dashboard-domains-section",
  DASHBOARD_EVENTS_SECTION: "dashboard-events-section",

  // Timeline chart [REQ:LD-UI-TRENDS]
  TIMELINE_CHART: "timeline-chart",
  TIMELINE_EMPTY: "timeline-empty",
  TIMELINE_BARS: "timeline-bars",
  TIMELINE_PERIOD_SELECTOR: "timeline-period-selector",

  // Domains [REQ:LD-UI-DOMAINS]
  DOMAINS_LIST: "domains-list",
  DOMAIN_CARD: "domain-card",
  DOMAIN_STATUS_BADGE: "domain-status-badge",

  // Events
  EVENTS_LIST: "events-list",
  EVENT_ROW: "event-row",

  // Stats
  STAT_TOTAL_EVENTS: "stat-total-events",
  STAT_ACTIVE_DOMAINS: "stat-active-domains",
  STAT_LAST_ACTIVITY: "stat-last-activity",
  STAT_7DAY_TREND: "stat-7day-trend",

  // Lifestyle Score [REQ:LD-UI-SCORE]
  LIFESTYLE_SCORE: "lifestyle-score",
  LIFESTYLE_SCORE_VALUE: "lifestyle-score-value",

  // Error handling
  ERROR_ALERT: "error-alert",
  ERROR_RETRY_BUTTON: "error-retry-button",

  // Settings/Storage [REQ:LD-UI-STORAGE]
  SETTINGS_PAGE: "settings-page",
  STORAGE_SIZE: "storage-size",
  STORAGE_EVENTS: "storage-events",
  STORAGE_CLEAR_ALL: "storage-clear-all",

  // Briefs [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING]
  BRIEF_CARD: "brief-card",
  BRIEF_SUMMARY: "brief-summary",
  BRIEF_SCORE: "brief-score",
  BRIEF_SECTIONS: "brief-sections",
  BRIEF_SECTION: "brief-section",
  BRIEFS_PAGE: "briefs-page",
  BRIEF_MORNING_TAB: "brief-morning-tab",
  BRIEF_EVENING_TAB: "brief-evening-tab",
} as const;
