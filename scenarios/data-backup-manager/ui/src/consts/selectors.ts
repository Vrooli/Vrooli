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

function mergeLiteralAndDynamicNodes<
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
>(
  literalNode: L | undefined,
  dynamicNode: D | undefined,
  path?: string[],
): SelectorTreeResult<L, D>;
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
}

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
  const selectors = mergeLiteralAndDynamicNodes(literalTree, dynamicTree);
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
  locale: {
    switcher: "locale-switcher",
  },
  layout: {
    shell: "layout-shell",
    topBar: "layout-top-bar",
    sidebar: "layout-sidebar",
    bottomNav: "layout-bottom-nav",
    main: "layout-main",
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
  },
  pages: {
    overview: "page-overview",
    targets: "page-targets",
    destinations: "page-destinations",
    plans: "page-plans",
    runs: "page-runs",
    restores: "page-restores",
    drills: "page-drills",
    settings: "page-settings",
  },
  // Shared async-section affordances (loading/error/empty/retry) used across
  // every surface so e2e flows can wait on a stable state.
  async: {
    loading: "async-loading",
    error: "async-error",
    retry: "async-retry",
    empty: "async-empty",
  },
  overview: {
    posture: "overview-posture",
    postureHeadline: "overview-posture-headline",
    storage: "overview-storage",
    storageEmpty: "overview-storage-empty",
    coverage: "overview-coverage",
    coverageEmpty: "overview-coverage-empty",
    setupCta: "overview-setup-cta",
    metrics: "overview-metrics",
  },
  coverage: {
    banner: "coverage-banner",
    registerRecommended: "coverage-register-recommended",
    recommendedList: "coverage-recommended-list",
    sensitiveList: "coverage-sensitive-list",
    registerSensitive: "coverage-register-sensitive",
    complete: "coverage-complete",
  },
  discovery: {
    panel: "discovery-panel",
    targetsGroup: "discovery-targets-group",
    destinationsGroup: "discovery-destinations-group",
    empty: "discovery-empty",
    destinationReview: "discovery-destination-review",
    reviewCreateButton: "discovery-destination-review-create",
  },
  targets: {
    table: "targets-table",
    registerButton: "targets-register-button",
    form: "targets-form",
    formOwner: "targets-form-owner",
    formName: "targets-form-name",
    formKind: "targets-form-kind",
    formLocator: "targets-form-locator",
    formCritical: "targets-form-critical",
    formSubmit: "targets-form-submit",
    inspector: "targets-inspector",
    verifyLatest: "targets-verify-latest",
    deregisterButton: "targets-deregister-button",
    deregisterConfirm: "targets-deregister-confirm",
  },
  destinations: {
    list: "destinations-list",
    createButton: "destinations-create-button",
    form: "destinations-form",
    formName: "destinations-form-name",
    formBackend: "destinations-form-backend",
    formLocation: "destinations-form-location",
    formRepoPreview: "destinations-form-repo-preview",
    formCap: "destinations-form-cap",
    formPolicy: "destinations-form-policy",
    formSubmit: "destinations-form-submit",
    editButton: "destinations-edit-button",
    deleteButton: "destinations-delete-button",
    deleteConfirm: "destinations-delete-confirm",
    deleteRepoToggle: "destinations-delete-repo-toggle",
  },
  plans: {
    list: "plans-list",
    createButton: "plans-create-button",
    form: "plans-form",
    formName: "plans-form-name",
    formSchedule: "plans-form-schedule",
    formDrillSchedule: "plans-form-drill-schedule",
    formTier: "plans-form-tier",
    formKeepLatest: "plans-form-keep-latest",
    formEnabled: "plans-form-enabled",
    targetPicker: "plans-target-picker",
    destinationPicker: "plans-destination-picker",
    summary: "plans-summary",
    formSubmit: "plans-form-submit",
    runNowButton: "plans-run-now-button",
    deleteButton: "plans-delete-button",
    coverageWarning: "plans-coverage-warning",
    proceedIncompleteCoverage: "plans-proceed-incomplete-coverage",
  },
  runs: {
    table: "runs-table",
  },
  restores: {
    list: "restores-list",
    startButton: "restores-start-button",
    verifyButton: "restores-verify-button",
    restoreButton: "restores-restore-button",
    restoreConfirm: "restores-restore-confirm",
    restoreLocation: "restores-restore-location",
    restoreConfirmButton: "restores-restore-confirm-button",
  },
  snapshot: {
    browser: "snapshot-browser",
    up: "snapshot-up",
  },
  audits: {
    runButton: "audits-run-button",
    report: "audits-report",
    verdict: "audits-verdict",
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
  layout: {
    sidebarLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-sidebar-link-${key}",
      params: {
        key: {
          type: "enum",
          values: ["overview", "targets", "destinations", "plans", "runs", "restores", "drills", "settings"] as const,
        },
      },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: ["overview", "targets", "destinations", "plans", "runs", "restores", "drills", "settings"] as const,
        },
      },
    }),
  },
  overview: {
    coverageRow: defineDynamicSelector({
      description: "Coverage-grid row for one target by id",
      testIdPattern: "overview-coverage-row-${targetId}",
      params: { targetId: { type: "string" } },
    }),
  },
  discovery: {
    suggestionRow: defineDynamicSelector({
      description: "Suggestion row (target or destination) by stable suggestion id",
      testIdPattern: "discovery-suggestion-${id}",
      params: { id: { type: "string" } },
    }),
    enableButton: defineDynamicSelector({
      description: "Enable (accept) button for a suggestion by stable id",
      testIdPattern: "discovery-enable-${id}",
      params: { id: { type: "string" } },
    }),
    dismissButton: defineDynamicSelector({
      description: "Dismiss button for a suggestion by stable id",
      testIdPattern: "discovery-dismiss-${id}",
      params: { id: { type: "string" } },
    }),
  },
  targets: {
    row: defineDynamicSelector({
      description: "Targets table row by target id",
      testIdPattern: "targets-row-${id}",
      params: { id: { type: "string" } },
    }),
  },
  destinations: {
    row: defineDynamicSelector({
      description: "Destinations list row by destination id",
      testIdPattern: "destinations-row-${id}",
      params: { id: { type: "string" } },
    }),
  },
  plans: {
    row: defineDynamicSelector({
      description: "Plans list row by plan id",
      testIdPattern: "plans-row-${id}",
      params: { id: { type: "string" } },
    }),
  },
  runs: {
    row: defineDynamicSelector({
      description: "Runs table row by run id",
      testIdPattern: "runs-row-${id}",
      params: { id: { type: "string" } },
    }),
    outcomeRow: defineDynamicSelector({
      description: "Per-target outcome row within an expanded run, by target id",
      testIdPattern: "runs-outcome-row-${targetId}",
      params: { targetId: { type: "string" } },
    }),
  },
  restores: {
    row: defineDynamicSelector({
      description: "Restores list row by restore id",
      testIdPattern: "restores-row-${id}",
      params: { id: { type: "string" } },
    }),
  },
  settingsPage: {
    themeOption: defineDynamicSelector({
      description: "Theme choice radio button on the settings page",
      testIdPattern: "page-settings-theme-${choice}",
      params: { choice: { type: "enum", values: ["light", "dark", "system"] as const } },
    }),
    localeOption: defineDynamicSelector({
      description: "Locale choice radio button on the settings page",
      testIdPattern: "page-settings-locale-${code}",
      params: { code: { type: "enum", values: LOCALE_CODES } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
