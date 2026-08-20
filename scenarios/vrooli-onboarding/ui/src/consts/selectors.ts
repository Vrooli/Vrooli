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
  readonly [key: string]: DynamicSelectorBranch | DynamicSelectorDefinition<ParamSchema | undefined>;
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
  template.replace(TEMPLATE_TOKEN, (_match: string, token: string) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<ParamSchema | undefined> =>
  Boolean(value && typeof value === "object" && "kind" in value && value.kind === "dynamic-selector");

const isLiteralBranch = (value: LiteralNode): value is LiteralSelectorTree =>
  typeof value === "object";

const isDynamicBranch = (
  value: DynamicSelectorBranch | DynamicSelectorDefinition<ParamSchema | undefined>,
): value is DynamicSelectorBranch => !isDynamicDefinition(value);

const normalizeParams = (
  definition: DynamicSelectorDefinition<ParamSchema | undefined>,
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
    const value = raw[key];
    if (!definitionEntry || value === undefined) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
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
      const paramEntries = value.params
        ? Object.keys(value.params)
            .map((name) => {
              const config = value.params?.[name];
              return config ? { name, config } : undefined;
            })
            .filter((entry): entry is { name: string; config: ParamDefinition } => entry !== undefined)
        : [];
      target[manifestKey] = {
        description: value.description,
        selectorPattern:
          value.selectorPattern ?? (value.testIdPattern ? toDataTestIdSelector(value.testIdPattern) : ""),
        testIdPattern: value.testIdPattern,
        params: paramEntries.map(({ name, config }) => ({
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

    if (literalValue && isLiteralBranch(literalValue)) {
      merged[key] = mergeLiteralAndDynamicNodes(
        literalValue,
        dynamicValue && isDynamicBranch(dynamicValue) ? dynamicValue : undefined,
        nextPath,
      );
      return;
    }

    if (dynamicValue) {
      if (isDynamicDefinition(dynamicValue)) {
        merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
        return;
      }
      if (isDynamicBranch(dynamicValue)) {
        merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicValue, nextPath);
      }
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

// Safe cast helper: mergeLiteralAndDynamicNodes structurally builds SelectorTreeResult
// but TypeScript can't prove it statically. Isolated here so the single assertion is auditable.
const toSelectorResult = <L extends LiteralSelectorTree, D extends DynamicSelectorTree>(
  raw: ReturnType<typeof mergeLiteralAndDynamicNodes>,
  _literalTree: L, _dynamicTree: D,
): SelectorTreeResult<L, D> => raw as
  // structurally verified: merge walks both trees and produces the correct shape
  SelectorTreeResult<L, D>;

const createSelectorRegistry = <
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
>(literalTree: L, dynamicTree: D) => {
  const selectors = toSelectorResult(mergeLiteralAndDynamicNodes(literalTree, dynamicTree), literalTree, dynamicTree);
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: flattenDynamicSelectors(dynamicTree),
  };
  return { selectors, manifest };
};

const literalSelectors = {
  app: {
    skipToContent: "skip-to-content",
    nav: "app-nav",
  },
  nav: {
    wizard: "nav-wizard",
    wizardBadge: "nav-wizard-badge",
    dashboard: "nav-dashboard",
    glossary: "nav-glossary",
  },
  wizard: {
    shell: "wizard-shell",
    stepsDesktop: "wizard-steps-desktop",
    progressBar: "progress-bar",
    prev: "wizard-prev",
    next: "wizard-next",
    welcome: "step-welcome",
    scenarios: "step-select-scenarios",
    integrations: "step-integrations-deferred",
    operatingMode: "step-operating-mode",
    resources: "step-derived-resources",
    scenarioSearch: "scenario-search",
    scenarioFilter: "scenario-filter",
    host: "step-host-requirements",
    readiness: "step-readiness",
    selectResources: "step-select-resources",
    review: "step-review",
    complete: "step-complete",
    resumePrompt: "resume-prompt",
    resumeButton: "resume-button",
    setupOrderHint: "setup-order-hint",
    resourcesLoading: "step-resources-loading",
    resourcesError: "step-resources-error",
    validationLoading: "validation-loading",
    validationError: "validation-error",
    validationSuccess: "validation-success",
    validationInvalid: "validation-invalid",
    configLoading: "config-loading",
    configError: "config-error",
    configOutput: "config-output",
    copyConfig: "copy-config",
    downloadConfig: "download-config",
    startOver: "start-over",
    resourceSearch: "resource-search",
    resourceSearchClear: "resource-search-clear",
    resourceFilterCount: "resource-filter-count",
    resourceNoResults: "resource-no-results",
    startOverConfirm: "start-over-confirm",
    startOverCancel: "start-over-cancel",
    reviewEmptyState: "review-empty-state",
    reviewGoBack: "review-go-back",
    stepAnnouncement: "step-announcement",
  },
  apply: {
    plan: "apply-plan",
    privilegeWarning: "privilege-warning",
    confirm: "apply-confirm",
    progress: "apply-progress",
    report: "apply-report",
    skippedNote: "skipped-note",
    retry: "retry",
  },
  readiness: {
    summary: "readiness-summary",
    item: "readiness-item",
    remediation: "remediation",
    recheck: "recheck",
    continueDegraded: "readiness-continue-degraded",
  },
  host: {
    tools: "host-tools",
    safeguards: "host-safeguards",
    requirementEntry: "requirement-entry",
    riskIndicator: "risk-indicator",
    privilegeIndicator: "privilege-indicator",
    changeSummary: "change-summary",
  },
  credentials: {
    card: "credential-card",
    purpose: "credential-purpose",
    status: "credential-status",
    input: "credential-input",
    save: "credential-save",
    obtainLink: "credential-obtain-link",
  },
  capabilities: {
    actions: "capability-actions",
  },
  scenario: {
    list: "scenario-list",
    cascadeNote: "cascade-note",
    resourceRollup: "resource-rollup",
    catalogError: "catalog-error",
    lockedBadge: "locked-badge",
  },
  dashboard: {
    root: "health-dashboard",
    summary: "health-summary",
    grid: "health-grid",
    loading: "health-loading",
    error: "health-error",
    lastChecked: "health-last-checked",
    goToWizard: "health-go-to-wizard",
  },
  glossary: {
    root: "glossary-panel",
    search: "glossary-search",
    clearSearch: "glossary-clear-search",
    debounceIndicator: "glossary-debounce-indicator",
    count: "glossary-count",
    loading: "glossary-loading",
    empty: "glossary-empty",
    list: "glossary-list",
  },
} as const satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions: DynamicSelectorTree = {
  wizard: {
    stepIndicator: defineDynamicSelector({
      description: "Step indicator circle by index",
      testIdPattern: "step-indicator-${index}",
      params: { index: { type: "number" } },
    }),
    scenarioCard: defineDynamicSelector({
      description: "Scenario card by scenario name",
      testIdPattern: "scenario-card-${name}",
      params: { name: { type: "string" } },
    }),
    resourceCard: defineDynamicSelector({
      description: "Resource card by name",
      testIdPattern: "resource-card-${name}",
      params: { name: { type: "string" } },
    }),
    categoryToggle: defineDynamicSelector({
      description: "Select All / Deselect All toggle per category",
      testIdPattern: "category-toggle-${category}",
      params: { category: { type: "string" } },
    }),
    removeResource: defineDynamicSelector({
      description: "Remove resource chip button in review step",
      testIdPattern: "remove-resource-${name}",
      params: { name: { type: "string" } },
    }),
  },
  dashboard: {
    healthCard: defineDynamicSelector({
      description: "Health card by resource name",
      testIdPattern: "health-card-${name}",
      params: { name: { type: "string" } },
    }),
    statusIndicator: defineDynamicSelector({
      description: "Status indicator by resource name",
      testIdPattern: "status-indicator-${name}",
      params: { name: { type: "string" } },
    }),
  },
  glossary: {
    entry: defineDynamicSelector({
      description: "Glossary entry by term",
      testIdPattern: "glossary-entry-${term}",
      params: { term: { type: "string" } },
    }),
  },
};

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
