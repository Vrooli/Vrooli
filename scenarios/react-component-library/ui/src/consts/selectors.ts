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
  components: {
    card: "components-card",
    list: "components-list",
    loading: "components-loading",
    empty: "components-empty",
    error: "components-error",
    indexButton: "components-index-button",
    create: {
      button: "components-create-button",
      dialog: "components-create-dialog",
      slug: "components-create-slug",
      libraryId: "components-create-library-id",
      displayName: "components-create-display-name",
      description: "components-create-description",
      tags: "components-create-tags",
      version: "components-create-version",
      fileName: "components-create-file-name",
      initialSource: "components-create-initial-source",
      submit: "components-create-submit",
      cancel: "components-create-cancel",
      error: "components-create-error",
    },
    searchInput: "components-search-input",
    tagInput: "components-tag-input",
    tagsInput: "components-tags-input",
    categoryInput: "components-category-input",
    styleInput: "components-style-input",
    affinityInput: "components-affinity-input",
    summary: "components-summary",
    item: "components-item",
    itemLibraryId: "components-item-library-id",
    itemDisplayName: "components-item-display-name",
    itemVersion: "components-item-version",
    itemSlot: "components-item-slot",
    itemDesignStyles: "components-item-design-styles",
    itemTags: "components-item-tags",
    itemEditButton: "components-item-edit-button",
    editor: {
      panel: "components-editor-panel",
      title: "components-editor-title",
      loading: "components-editor-loading",
      error: "components-editor-error",
      surface: "components-editor-surface",
      saveButton: "components-editor-save-button",
      closeButton: "components-editor-close-button",
      infoButton: "components-editor-info-button",
      infoDialog: "components-editor-info-dialog",
      dirtyBadge: "components-editor-dirty-badge",
      shaHash: "components-editor-sha-hash",
      savedToast: "components-editor-saved-toast",
      modeSwitch: "components-editor-mode-switch",
      previewModeButton: "components-editor-preview-mode-button",
      codeModeButton: "components-editor-code-mode-button",
      preview: "components-editor-preview",
      gallery: "components-editor-gallery",
      exampleCard: "components-editor-example-card",
      exampleTitle: "components-editor-example-title",
      previewFrame: "components-editor-preview-frame",
      previewBadge: "components-editor-preview-badge",
      previewError: "components-editor-preview-error",
      previewRetryButton: "components-editor-preview-retry-button",
    },
    emulator: {
      root: "components-emulator",
      toolbar: "components-emulator-toolbar",
      presetSelect: "components-emulator-preset-select",
      dimensions: "components-emulator-dimensions",
      zoomValue: "components-emulator-zoom-value",
      zoomIn: "components-emulator-zoom-in",
      zoomOut: "components-emulator-zoom-out",
      resetZoom: "components-emulator-reset-zoom",
      rotate: "components-emulator-rotate",
      reset: "components-emulator-reset",
      viewport: "components-emulator-viewport",
      colorSchemeSelect: "components-emulator-color-scheme",
      visionFilterSelect: "components-emulator-vision-filter",
      blurSlider: "components-emulator-blur-slider",
      blurValue: "components-emulator-blur-value",
      filterDefs: "components-emulator-filter-defs",
    },
    themeSwitcher: {
      root: "components-theme-switcher",
      select: "components-theme-switcher-select",
      scenarioInput: "components-theme-switcher-scenario-input",
      scenarioApply: "components-theme-switcher-scenario-apply",
      status: "components-theme-switcher-status",
      error: "components-theme-switcher-error",
    },
    inspector: {
      panel: "components-inspector-panel",
      toggleButton: "components-inspector-toggle",
      statusBadge: "components-inspector-status",
      breadcrumb: "components-inspector-breadcrumb",
      breadcrumbItem: "components-inspector-breadcrumb-item",
      selectedTag: "components-inspector-selected-tag",
      selectedSelector: "components-inspector-selected-selector",
      selectedRect: "components-inspector-selected-rect",
      selectedText: "components-inspector-selected-text",
      empty: "components-inspector-empty",
      overlay: "components-inspector-overlay",
    },
  },
  adoptions: {
    card: "adoptions-card",
    list: "adoptions-list",
    loading: "adoptions-loading",
    error: "adoptions-error",
    empty: "adoptions-empty",
    summary: "adoptions-summary",
    refreshButton: "adoptions-refresh-button",
    scenarioFilter: "adoptions-scenario-filter",
    item: "adoptions-item",
    itemId: "adoptions-item-id",
    itemScenario: "adoptions-item-scenario",
    itemPath: "adoptions-item-path",
    itemVersion: "adoptions-item-version",
    itemLibraryId: "adoptions-item-library-id",
    itemStatus: "adoptions-item-status",
    itemStatusDetail: "adoptions-item-status-detail",
    itemRefreshedAt: "adoptions-item-refreshed-at",
    itemDeleteButton: "adoptions-item-delete-button",
    createButton: "adoptions-create-button",
    createDialog: "adoptions-create-dialog",
    createComponentId: "adoptions-create-component-id",
    createScenario: "adoptions-create-scenario",
    createAdoptedPath: "adoptions-create-adopted-path",
    createAdoptedVersion: "adoptions-create-adopted-version",
    createCancel: "adoptions-create-cancel",
    createConfirm: "adoptions-create-confirm",
    createVerdict: "adoptions-create-verdict",
    createVerdictKind: "adoptions-create-verdict-kind",
    createVerdictIssue: "adoptions-create-verdict-issue",
    createVerdictAck: "adoptions-create-verdict-ack",
    createStyleVerdict: "adoptions-create-style-verdict",
    createStyleVerdictKind: "adoptions-create-style-verdict-kind",
    createStyleVerdictDetail: "adoptions-create-style-verdict-detail",
    createError: "adoptions-create-error",
    createPathSource: "adoptions-create-path-source",
    createPathWarning: "adoptions-create-path-warning",
  },
  versions: {
    card: "versions-card",
    list: "versions-list",
    loading: "versions-loading",
    error: "versions-error",
    empty: "versions-empty",
    item: "versions-item",
    itemId: "versions-item-id",
    itemVersion: "versions-item-version",
    itemSha: "versions-item-sha",
    itemRecordedAt: "versions-item-recorded-at",
    itemChangelog: "versions-item-changelog",
    itemDiffButton: "versions-item-diff-button",
    diff: {
      card: "versions-diff-card",
      fromSelect: "versions-diff-from",
      toSelect: "versions-diff-to",
      runButton: "versions-diff-run",
      table: "versions-diff-table",
      row: "versions-diff-row",
      summary: "versions-diff-summary",
      empty: "versions-diff-empty",
      loading: "versions-diff-loading",
      error: "versions-diff-error",
    },
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
