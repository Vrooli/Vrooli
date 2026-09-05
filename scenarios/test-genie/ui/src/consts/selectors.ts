/**
 * Test Genie selector registry
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

type AnyDynamicSelectorDefinition = DynamicSelectorDefinition<ParamSchema | undefined>;

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
} & (D extends DynamicSelectorTree ? DynamicBranchResult<D> : unknown);

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match: string, token: string) => {
    const resolved = values[token];
    if (resolved === undefined) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(resolved);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isRecord = (value: unknown): value is Record<string, unknown> =>
  typeof value === "object" && value !== null;

const isLiteralSelectorTree = (value: unknown): value is LiteralSelectorTree =>
  isRecord(value) && Object.values(value).every((entry) => typeof entry === "string" || isLiteralSelectorTree(entry));

const isDynamicDefinition = (value: unknown): value is AnyDynamicSelectorDefinition =>
  isRecord(value) && value.kind === "dynamic-selector";

const isDynamicSelectorTree = (value: unknown): value is DynamicSelectorTree =>
  isRecord(value) && Object.values(value).every((entry) => isDynamicDefinition(entry) || isDynamicSelectorTree(entry));

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
      continue;
    }
    const value = raw[key];
    if (value === undefined) {
      throw new Error(`Selector '${path}' parameter '${key}' resolved to undefined`);
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
      const paramEntries = Object.entries(value.params ?? {});
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

function mergeLiteralAndDynamicNodes<
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
>(
  literalNode: L,
  dynamicNode: D,
  path?: string[],
): SelectorTreeResult<L, D>;
function mergeLiteralAndDynamicNodes(
  literalNode: LiteralSelectorTree | undefined,
  dynamicNode: DynamicSelectorTree | undefined,
  path?: string[],
): Record<string, unknown>;
function mergeLiteralAndDynamicNodes(
  literalNode: LiteralSelectorTree | undefined,
  dynamicNode: DynamicSelectorTree | undefined,
  path: string[] = [],
): Record<string, unknown> {
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

    if (isLiteralSelectorTree(literalValue)) {
      const dynamicBranch = isDynamicSelectorTree(dynamicValue) ? dynamicValue : undefined;
      merged[key] = mergeLiteralAndDynamicNodes(
        literalValue,
        isDynamicDefinition(dynamicValue) ? undefined : dynamicBranch,
        nextPath,
      );
      return;
    }

    if (dynamicValue) {
      if (isDynamicDefinition(dynamicValue)) {
        merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
        return;
      }
      if (isDynamicSelectorTree(dynamicValue)) {
        merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicValue, nextPath);
      }
    }
  });

  return merged;
}

const createDynamicSelectorFn = (
  definition: AnyDynamicSelectorDefinition,
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

const createSelectorRegistry = <
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
>(literalTree: L, dynamicTree: D) => {
  const selectors = mergeLiteralAndDynamicNodes(literalTree, dynamicTree);
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: flattenDynamicSelectors(dynamicTree),
  };
  return { selectors, manifest };
};

const literalSelectors = {
  // Top-level tab navigation
  tabs: {
    nav: "test-genie-tab-nav",
    dashboard: "test-genie-tab-dashboard",
    runs: "test-genie-tab-runs",
    docs: "test-genie-tab-docs",
    health: "test-genie-tab-health"
  },
  // Dashboard page
  dashboard: {
    continueSection: "test-genie-continue-section",
    header: "test-genie-header",
    lastExecution: "test-genie-last-execution"
  },
  // Self-Health page
  health: {
    page: "test-genie-health-page",
    catalog: "test-genie-health-catalog",
    conformance: "test-genie-health-conformance",
    ledger: "test-genie-health-ledger",
    providers: "test-genie-health-providers",
    trend: "test-genie-health-trend",
    empty: "test-genie-health-empty"
  },
  // Runs page
  runs: {
    subtabScenarios: "test-genie-subtab-scenarios",
    subtabHistory: "test-genie-subtab-history",
    scenarioTable: "test-genie-scenario-table",
    testGenieScenario: "test-genie-scenario-row-test-genie",
    historyTable: "test-genie-history-table",
    scenarioDetail: "test-genie-scenario-detail",
    scenarioDetailBack: "test-genie-scenario-detail-back",
    scenarioTabOverview: "test-genie-scenario-tab-overview",
    scenarioTabRequirements: "test-genie-scenario-tab-requirements",
    scenarioTabHistory: "test-genie-scenario-tab-history",
    phaseCard: "test-genie-phase-card",
    remediationPanel: "test-genie-remediation-panel"
  },
  // Requirements
  requirements: {
    panel: "test-genie-requirements-panel",
    syncBanner: "test-genie-sync-banner",
    syncButton: "test-genie-sync-button",
    coverageStats: "test-genie-coverage-stats",
    tree: "test-genie-requirements-tree",
    filterAll: "test-genie-filter-all",
    filterPassed: "test-genie-filter-passed",
    filterFailed: "test-genie-filter-failed",
    filterNotRun: "test-genie-filter-not-run",
    searchInput: "test-genie-requirements-search",
    // Help section
    helpSection: "test-genie-requirements-help",
    helpToggle: "test-genie-requirements-help-toggle",
  },
  // Docs page
  docs: {
    sidebar: "test-genie-docs-sidebar",
    viewer: "test-genie-docs-viewer",
    copyPath: "test-genie-docs-copy-path",
    searchInput: "test-genie-docs-search"
  },
  // Forms (used in Runs detail)
  forms: {
    executionForm: "test-genie-execution-form",
    submitExecution: "test-genie-submit-execution"
  },
  // Actions
  actions: {
    runTests: "test-genie-action-run-tests",
    viewScenario: "test-genie-action-view-scenario"
  }
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  scenarios: {
    rowByName: {
      kind: "dynamic-selector",
      description: "Scenario directory row by scenario name",
      testIdPattern: "test-genie-scenario-row-${name}",
      params: { name: { type: "string" } }
    }
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
