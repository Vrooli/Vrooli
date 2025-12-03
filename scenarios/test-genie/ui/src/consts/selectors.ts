/**
 * Test Genie selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Browser Automation Studio workflows. We deliberately model selectors as two
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
} & (D extends DynamicSelectorTree ? DynamicBranchResult<D> : {});

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match, token) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<any> =>
  Boolean(value && typeof value === "object" && (value as DynamicSelectorDefinition).kind === "dynamic-selector");

const normalizeParams = (
  definition: DynamicSelectorDefinition<any>,
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
  definition: DynamicSelectorDefinition<any>,
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
  const selectors = mergeLiteralAndDynamicNodes(literalTree, dynamicTree) as SelectorTreeResult<L, D>;
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: flattenDynamicSelectors(dynamicTree),
  };
  return { selectors, manifest };
};

const literalSelectors: LiteralSelectorTree = {
  // Top-level tab navigation
  tabs: {
    nav: "test-genie-tab-nav",
    dashboard: "test-genie-tab-dashboard",
    runs: "test-genie-tab-runs",
    generate: "test-genie-tab-generate",
    docs: "test-genie-tab-docs"
  },
  // Dashboard page
  dashboard: {
    continueSection: "test-genie-continue-section",
    header: "test-genie-header",
    guidedFlows: "test-genie-guided-flows",
    stats: "test-genie-stats",
    queueHealth: "test-genie-queue-health",
    lastExecution: "test-genie-last-execution"
  },
  // Runs page
  runs: {
    subtabScenarios: "test-genie-subtab-scenarios",
    subtabHistory: "test-genie-subtab-history",
    scenarioTable: "test-genie-scenario-table",
    historyTable: "test-genie-history-table",
    scenarioDetail: "test-genie-scenario-detail",
    scenarioDetailBack: "test-genie-scenario-detail-back",
    scenarioTabOverview: "test-genie-scenario-tab-overview",
    scenarioTabRequirements: "test-genie-scenario-tab-requirements",
    scenarioTabHistory: "test-genie-scenario-tab-history"
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
    searchInput: "test-genie-requirements-search"
  },
  // Generate page
  generate: {
    phaseSelector: "test-genie-phase-selector",
    promptEditor: "test-genie-prompt-editor",
    copyButton: "test-genie-copy-prompt",
    spawnButton: "test-genie-spawn-agent",
    presetSelector: "test-genie-preset-selector",
    scopeButton: "test-genie-scope-button",
    targetSelector: "test-genie-target-selector",
    targetItem: "test-genie-target-item"
  },
  // Docs page
  docs: {
    sidebar: "test-genie-docs-sidebar",
    viewer: "test-genie-docs-viewer",
    copyPath: "test-genie-docs-copy-path",
    searchInput: "test-genie-docs-search"
  },
  // Forms (used in Dashboard and Runs detail)
  forms: {
    queueForm: "test-genie-queue-form",
    executionForm: "test-genie-execution-form",
    submitQueue: "test-genie-submit-queue",
    submitExecution: "test-genie-submit-execution"
  },
  // Actions
  actions: {
    queueTests: "test-genie-action-queue-tests",
    runTests: "test-genie-action-run-tests",
    viewScenario: "test-genie-action-view-scenario",
    copyPrompt: "test-genie-action-copy-prompt"
  }
};

const dynamicSelectorDefinitions: DynamicSelectorTree = {
  /*
  Example dynamic selectors:
  scenarios: {
    rowByName: defineDynamicSelector({
      description: 'Scenario row by name',
      testIdPattern: 'test-genie-scenario-row-${name}',
      params: { name: { type: 'string' } },
    }),
  },
  */
};

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

// Export with explicit type to fix inference issues with empty dynamic selectors
export const selectors = registry.selectors as unknown as {
  tabs: {
    nav: string;
    dashboard: string;
    runs: string;
    generate: string;
    docs: string;
  };
  dashboard: {
    continueSection: string;
    header: string;
    guidedFlows: string;
    stats: string;
    queueHealth: string;
    lastExecution: string;
  };
  runs: {
    subtabScenarios: string;
    subtabHistory: string;
    scenarioTable: string;
    historyTable: string;
    scenarioDetail: string;
    scenarioDetailBack: string;
    scenarioTabOverview: string;
    scenarioTabRequirements: string;
    scenarioTabHistory: string;
  };
  requirements: {
    panel: string;
    syncBanner: string;
    syncButton: string;
    coverageStats: string;
    tree: string;
    filterAll: string;
    filterPassed: string;
    filterFailed: string;
    filterNotRun: string;
    searchInput: string;
  };
  generate: {
    phaseSelector: string;
    promptEditor: string;
    copyButton: string;
    spawnButton: string;
    presetSelector: string;
    scopeButton: string;
    targetSelector: string;
    targetItem: string;
  };
  docs: {
    sidebar: string;
    viewer: string;
    copyPath: string;
    searchInput: string;
  };
  forms: {
    queueForm: string;
    executionForm: string;
    submitQueue: string;
    submitExecution: string;
  };
  actions: {
    queueTests: string;
    runTests: string;
    viewScenario: string;
    copyPrompt: string;
  };
};
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
