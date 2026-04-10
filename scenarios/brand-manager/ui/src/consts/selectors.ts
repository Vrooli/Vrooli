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
  template.replace(TEMPLATE_TOKEN, (_match, token: string) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    const val = values[token];
    if (val === undefined) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(val);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<ParamSchema | undefined> => {
  if (!value || typeof value !== "object") return false;
  if (!("kind" in value)) return false;
  const record: Record<string, unknown> = value;
  return record.kind === "dynamic-selector";
};

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
    if (value === undefined) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    if (definitionEntry?.type === "number") {
      if (typeof value !== "number") {
        throw new Error(`Selector '${path}' parameter '${key}' must be numeric`);
      }
      normalized[key] = value;
      continue;
    }
    if (definitionEntry?.type === "enum") {
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
      const paramSchema: ParamSchema = value.params ?? {};
      const paramEntries: Array<[string, ParamDefinition]> = Object.entries(paramSchema);
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
      const literalBranch: LiteralSelectorTree = literalValue;
      const dynamicBranch: DynamicSelectorTree | undefined =
        isDynamicDefinition(dynamicValue) ? undefined : (dynamicValue ?? undefined);
      merged[key] = mergeLiteralAndDynamicNodes(literalBranch, dynamicBranch, nextPath);
      return;
    }

    if (dynamicValue) {
      if (isDynamicDefinition(dynamicValue)) {
        merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
        return;
      }
      const dynamicBranch: DynamicSelectorTree = dynamicValue;
      merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicBranch, nextPath);
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

// Runtime type guard for the merged selector tree. The merge function builds the correct
// shape at runtime (strings for literals, functions for dynamics) but returns Record<string, unknown>.
// This guard narrows to the strongly-typed consumer API without using an `as` cast.
function isSelectorTreeResult<L extends LiteralSelectorTree, D extends DynamicSelectorTree>(
  value: unknown,
): value is SelectorTreeResult<L, D> {
  return value !== null && typeof value === "object" && !Array.isArray(value);
}

const createSelectorRegistry = <
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
>(literalTree: L, dynamicTree: D) => {
  const merged: unknown = mergeLiteralAndDynamicNodes(literalTree, dynamicTree);
  if (!isSelectorTreeResult<L, D>(merged)) {
    throw new Error("Selector registry merge produced an invalid result");
  }
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: flattenDynamicSelectors(dynamicTree),
  };
  return { selectors: merged, manifest };
};

const literalSelectors = {
  app: {
    root: "app-root",
    healthIndicator: "health-indicator",
    navHome: "nav-home",
  },
  brandList: {
    page: "brand-list-page",
    createBtn: "create-brand-btn",
    searchInput: "brand-search-input",
    refreshBtn: "refresh-brands-btn",
    grid: "brand-list-grid",
    empty: "brand-list-empty",
    error: "brand-list-error",
  },
  brandDetail: {
    page: "brand-detail-page",
    backBtn: "back-to-brands",
    editBtn: "edit-brand-btn",
    deleteBtn: "delete-brand-btn",
    colorsSection: "brand-colors-section",
    identitySection: "brand-identity-section",
    typographySection: "brand-typography-section",
    voiceSection: "brand-voice-section",
    versionsSection: "brand-versions-section",
    themePreview: "theme-preview-section",
    applyPreview: "apply-preview-section",
  },
  brandForm: {
    page: "brand-form-page",
    backBtn: "back-from-form",
    nameInput: "brand-name-input",
    descriptionInput: "brand-description-input",
    saveBtn: "save-brand-btn",
    error: "form-error",
    generateOptions: "generate-options-section",
  },
  contrast: {
    badge: "contrast-badge",
  },
  scanner: {
    page: "scanner-page",
    input: "scanner-input",
    scanBtn: "scan-btn",
    backBtn: "back-to-brands",
    loading: "scanner-loading",
    scanResults: "scan-results",
    scanTotal: "scan-total",
    scanCss: "scan-css",
    scanJson: "scan-json",
    scanFindings: "scan-findings",
    scanNoFindings: "scan-no-findings",
    scanError: "scan-error",
    auditResults: "audit-results",
    auditPass: "audit-pass",
    auditFail: "audit-fail",
    auditItems: "audit-items",
    auditError: "audit-error",
    auditRulesSection: "audit-rules-section",
  },
  standards: {
    page: "standards-page",
    backBtn: "back-to-brands",
    loading: "standards-loading",
    error: "standards-error",
    list: "standards-list",
    empty: "standards-empty",
  },
  nav: {
    home: "nav-home",
    scanner: "nav-scanner",
    standards: "nav-standards",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  brands: {
    cardById: defineDynamicSelector({
      description: "Brand card by brand ID",
      testIdPattern: "brand-card-${id}",
      params: { id: { type: "string" } },
    }),
  },
  standards: {
    ruleById: defineDynamicSelector({
      description: "Standard rule card by rule ID",
      testIdPattern: "standard-${id}",
      params: { id: { type: "string" } },
    }),
  },
  colors: {
    swatchByLabel: defineDynamicSelector({
      description: "Color swatch by label",
      testIdPattern: "color-swatch-${label}",
      params: { label: { type: "string" } },
    }),
    pickerByKey: defineDynamicSelector({
      description: "Color picker input by key",
      testIdPattern: "color-picker-${key}",
      params: { key: { type: "string" } },
    }),
    inputByKey: defineDynamicSelector({
      description: "Color hex input by key",
      testIdPattern: "color-input-${key}",
      params: { key: { type: "string" } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
