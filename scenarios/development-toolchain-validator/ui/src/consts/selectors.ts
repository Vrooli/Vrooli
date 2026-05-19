/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. We deliberately model selectors as two
 * declarative maps (one literal, one dynamic) and rely on a small helper to
 * produce the typed `selectors` export plus the in-memory `selectorsManifest`
 * export consumed by workflow tooling. Do not hand-roll selector helpers or
 * change this structure—update the maps below so UI code, automation flows, and
 * the manifest builder all stay in sync across every scenario.
 *
 * ## Manifest Export
 *
 * If you need to add or modify selectors:
 *
 * 1. Update the `literalSelectors` object below for static selectors
 * 2. Update the `dynamicSelectorDefinitions` object for parameterized selectors
 * 3. `selectorsManifest` updates from the same source maps automatically
 */
import { LOCALE_CODES } from "../i18n/locales";

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
  readonly [key: string]:
    | DynamicSelectorBranch
    | DynamicSelectorDefinition<ParamSchema | undefined>;
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

const isDynamicDefinition = (
  value: unknown,
): value is DynamicSelectorDefinition<ParamSchema | undefined> =>
  Boolean(
    value &&
      typeof value === "object" &&
      (value as { kind?: unknown }).kind === "dynamic-selector",
  );

const normalizeParams = (
  definition: DynamicSelectorDefinition<ParamSchema | undefined>,
  raw: Record<string, string | number>,
  path: string,
): Record<string, string | number> => {
  const schema: ParamSchema = definition.params ?? {};
  const normalized: Record<string, string | number> = {};

  for (const [key, definitionEntry] of Object.entries(schema)) {
    if (!(key in raw)) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    const value = raw[key];
    // Defensive: `key in raw` doesn't narrow `raw[key]` under noUncheckedIndexedAccess.
    if (value === undefined) {
      throw new Error(`Selector '${path}' parameter '${key}' is undefined`);
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
        literalValue,
        isDynamicDefinition(dynamicValue) ? undefined : dynamicValue,
        nextPath,
      );
      return;
    }

    if (dynamicValue) {
      if (isDynamicDefinition(dynamicValue)) {
        merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
        return;
      }
      merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicValue, nextPath);
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

/**
 * Build a dynamic-selector definition. Exported so unit tests and downstream
 * tooling can construct registries; scenario authors should still edit the
 * `dynamicSelectorDefinitions` map at the bottom of this file rather than
 * calling this from elsewhere in the codebase.
 */
export const defineDynamicSelector = <P extends ParamSchema | undefined>(
  definition: Omit<DynamicSelectorDefinition<P>, "kind">,
): DynamicSelectorDefinition<P> => ({
  ...definition,
  kind: "dynamic-selector",
});

/**
 * Compose a typed selectors object plus a manifest from literal + dynamic
 * trees. Exported for unit tests; production registries are built from the
 * private `literalSelectors` and `dynamicSelectorDefinitions` maps below.
 */
export const createSelectorRegistry = <
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

// Use `satisfies` rather than `: LiteralSelectorTree` so TypeScript preserves
// the narrow literal shape. Without this the index signature widens every
// branch to `T | undefined` under `noUncheckedIndexedAccess` and breaks the
// `selectors.app.title` ergonomics this registry exists to provide.
const literalSelectors = {
  app: {
    title: "app-title",
    eyebrow: "app-eyebrow",
    description: "app-description",
  },
  health: {
    card: "health-card",
    loading: "health-loading",
    error: "health-error",
    statusValue: "health-status-value",
    serviceValue: "health-service-value",
    timestampValue: "health-timestamp-value",
    refreshButton: "health-refresh-button",
    refreshCount: "health-refresh-count",
  },
  notifications: {
    summary: "notifications-summary",
  },
  goldens: {
    card: "goldens-card",
    list: "goldens-list",
    loading: "goldens-loading",
    empty: "goldens-empty",
    error: "goldens-error",
    row: "goldens-row",
    indexHeading: "goldens-index-heading",
    detailHeading: "goldens-detail-heading",
    detailBack: "goldens-detail-back",
    registerOpen: "goldens-register-open",
    registerSheet: "goldens-register-sheet",
    registerForm: "goldens-register-form",
    registerSlug: "goldens-register-slug",
    registerTemplate: "goldens-register-template",
    registerVersion: "goldens-register-version",
    registerPath: "goldens-register-path",
    registerSubmit: "goldens-register-submit",
    registerError: "goldens-register-error",
    detail: "goldens-detail",
    detailRegenerate: "goldens-detail-regenerate",
    detailDelete: "goldens-detail-delete",
    detailClose: "goldens-detail-close",
    detailStatus: "goldens-detail-status",
    skillsGrid: "goldens-skills-grid",
    toolsGrid: "goldens-tools-grid",
    rowVerdictSummary: "goldens-row-verdict-summary",
    tupleDetail: "goldens-tuple-detail",
    tupleDetailBack: "goldens-tuple-back",
    tupleDetailHeading: "goldens-tuple-detail-heading",
    tupleDetailRunSummary: "goldens-tuple-run-summary",
    tupleDetailTabs: "goldens-tuple-tabs",
    tupleDetailDiff: "goldens-tuple-diff",
    tupleDetailManifest: "goldens-tuple-manifest",
    tupleDetailHistory: "goldens-tuple-history",
    tupleRow: "goldens-tuple-row",
  },
  skills: {
    surface: "skills-surface",
    list: "skills-list",
    loading: "skills-loading",
    empty: "skills-empty",
    error: "skills-error",
    row: "skills-row",
    detail: "skills-detail",
    detailBack: "skills-detail-back",
    detailHeading: "skills-detail-heading",
  },
  manifests: {
    surface: "manifests-surface",
    list: "manifests-list",
    loading: "manifests-loading",
    empty: "manifests-empty",
    error: "manifests-error",
    row: "manifests-row",
    editor: "manifests-editor",
    editorBack: "manifests-editor-back",
    editorHeading: "manifests-editor-heading",
    editorAllowedPaths: "manifests-editor-allowed-paths",
    editorWildcardAllowed: "manifests-editor-wildcard-allowed",
    editorConvergence: "manifests-editor-convergence",
    editorContentRules: "manifests-editor-content-rules",
    editorSave: "manifests-editor-save",
    editorClearStale: "manifests-editor-clear-stale",
    editorStatus: "manifests-editor-status",
  },
  nav: {
    sidebar: "nav-sidebar",
    sidebarLogo: "nav-sidebar-logo",
    sidebarCollapseToggle: "nav-sidebar-collapse",
    sidebarItemGoldens: "nav-sidebar-goldens",
    sidebarItemSkills: "nav-sidebar-skills",
    sidebarItemManifests: "nav-sidebar-manifests",
    sidebarItemSettings: "nav-sidebar-settings",
    topHeader: "nav-top-header",
    topHeaderConvergence: "nav-top-header-convergence",
    topHeaderStale: "nav-top-header-stale",
    topHeaderHealth: "nav-top-header-health",
    topHeaderMenu: "nav-top-header-menu",
    mobileBottomNav: "nav-mobile-bottom",
    mobileBottomItemGoldens: "nav-mobile-goldens",
    mobileBottomItemSkills: "nav-mobile-skills",
    mobileBottomItemManifests: "nav-mobile-manifests",
    mobileBottomItemSettings: "nav-mobile-settings",
    appShell: "app-shell",
  },
  settings: {
    surface: "settings-surface",
    themeDark: "settings-theme-dark",
    themeLight: "settings-theme-light",
    densityComfortable: "settings-density-comfortable",
    densityCompact: "settings-density-compact",
    sidebarCollapsed: "settings-sidebar-collapsed",
    catalogSyncCard: "settings-catalog-sync-card",
    catalogSyncButton: "settings-catalog-sync-button",
    catalogSyncSummary: "settings-catalog-sync-summary",
    catalogSyncError: "settings-catalog-sync-error",
    watcherCard: "settings-watcher-card",
    watcherSummary: "settings-watcher-summary",
    watcherError: "settings-watcher-error",
  },
  locale: {
    switcher: "locale-switcher",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

// Per-locale toggle test IDs are emitted by `locale.toggle({ code })` below.
// We deliberately do NOT also declare static `toggleEn` / `toggleJa` literals —
// the dynamic form is the single source of truth, and duplicating it here would
// drift the moment a new locale is added to LOCALE_CODES.
//
// `code` is constrained to `LOCALE_CODES` so `selectors.locale.toggle({ code: "fr" })`
// is a TypeScript error when "fr" isn't a supported locale. The runtime enum
// validation in `normalizeParams` provides the same guarantee at call time.
const dynamicSelectorDefinitions = {
  locale: {
    toggle: defineDynamicSelector({
      description: "Locale toggle button by language code",
      testIdPattern: "locale-toggle-${code}",
      params: { code: { type: "enum", values: LOCALE_CODES } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
