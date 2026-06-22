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
  jobs: {
    card: "jobs-card",
    list: "jobs-list",
    loading: "jobs-loading",
    empty: "jobs-empty",
    error: "jobs-error",
    operation: "jobs-operation",
    lane: "jobs-lane",
    state: "jobs-state",
    progress: "jobs-progress",
    message: "jobs-message",
    result: "jobs-result",
    errorDetail: "jobs-error-detail",
    liveBadge: "jobs-live-badge",
    cancelButton: "jobs-cancel-button",
  },
  home: {
    root: "home-root",
    hero: "home-hero",
    dropzone: "home-dropzone",
    fileInput: "home-file-input",
    cameraInput: "home-camera-input",
    chooseButton: "home-choose-button",
    cameraButton: "home-camera-button",
    sampleButton: "home-sample-button",
    groups: "home-groups",
    recent: "home-recent",
    recentList: "home-recent-list",
    recentEmpty: "home-recent-empty",
    viewLibrary: "home-view-library",
    firstRun: "home-first-run",
    samples: "home-samples",
  },
  library: {
    root: "library-root",
    loading: "library-loading",
    error: "library-error",
    empty: "library-empty",
    noMatches: "library-no-matches",
    grid: "library-grid",
    search: "library-search",
    count: "library-count",
    selectAll: "library-select-all",
    clearSelection: "library-clear-selection",
    downloadSelected: "library-download-selected",
  },
  looks: {
    root: "looks-root",
    loading: "looks-loading",
    error: "looks-error",
    empty: "looks-empty",
    grid: "looks-grid",
  },
  select: {
    root: "select-root",
    dropzone: "select-dropzone",
    fileInput: "select-file-input",
    image: "select-image",
    maskOverlay: "select-mask-overlay",
    autoButton: "select-auto-button",
    pointXInput: "select-point-x",
    pointYInput: "select-point-y",
    selectPointButton: "select-point-button",
    toleranceInput: "select-tolerance",
    status: "select-status",
    error: "select-error",
    regionClass: "select-region-class",
    editsHeading: "select-edits-heading",
    editsList: "select-edits-list",
    promptInput: "select-prompt-input",
    applyResult: "select-apply-result",
    reset: "select-reset",
  },
  compare: {
    root: "compare-root",
    baseDropzone: "compare-base-dropzone",
    baseFileInput: "compare-base-file-input",
    baseImage: "compare-base-image",
    compareDropzone: "compare-compare-dropzone",
    compareFileInput: "compare-compare-file-input",
    compareImage: "compare-compare-image",
    modeToggle: "compare-mode-toggle",
    toleranceInput: "compare-tolerance",
    runButton: "compare-run-button",
    reset: "compare-reset",
    status: "compare-status",
    error: "compare-error",
    result: "compare-result",
    verdict: "compare-verdict",
    heatmap: "compare-heatmap",
    metrics: "compare-metrics",
    warnings: "compare-warnings",
  },
  activity: {
    card: "activity-card",
    list: "activity-list",
    loading: "activity-loading",
    empty: "activity-empty",
    error: "activity-error",
  },
  backends: {
    card: "backends-card",
    list: "backends-list",
    loading: "backends-loading",
    empty: "backends-empty",
    error: "backends-error",
    name: "backends-name",
    operations: "backends-operations",
    state: "backends-state",
    installButton: "backends-install-button",
    installNotice: "backends-install-notice",
    manualHint: "backends-manual-hint",
    remediation: "backends-remediation",
  },
  models: {
    card: "models-card",
    list: "models-list",
    loading: "models-loading",
    empty: "models-empty",
    error: "models-error",
    name: "models-name",
    tier: "models-tier",
    backend: "models-backend",
    operations: "models-operations",
    size: "models-size",
    hardware: "models-hardware",
    hardwareChip: "models-hardware-chip",
    license: "models-license",
    commercial: "models-commercial",
    commercialNotes: "models-commercial-notes",
    nsfwBadge: "models-nsfw-badge",
    customBadge: "models-custom-badge",
    defaultBadge: "models-default-badge",
    defaultFor: "models-default-for",
    downloadSize: "models-download-size",
    diskSize: "models-disk-size",
    installState: "models-install-state",
    installButton: "models-install-button",
    installNotice: "models-install-notice",
    removeButton: "models-remove-button",
    enabledState: "models-enabled-state",
    toggleButton: "models-toggle-button",
    blocklist: {
      card: "models-blocklist-card",
      list: "models-blocklist-list",
      loading: "models-blocklist-loading",
      empty: "models-blocklist-empty",
      error: "models-blocklist-error",
      entry: "models-blocklist-entry",
      onnxWarning: "models-blocklist-onnx-warning",
    },
    addCustom: {
      form: "models-add-custom-form",
      id: "models-add-custom-id",
      operations: "models-add-custom-operations",
      backend: "models-add-custom-backend",
      localPath: "models-add-custom-local-path",
      downloadUrl: "models-add-custom-download-url",
      submit: "models-add-custom-submit",
      success: "models-add-custom-success",
      error: "models-add-custom-error",
    },
    defaults: {
      card: "models-defaults-card",
      list: "models-defaults-list",
      loading: "models-defaults-loading",
      empty: "models-defaults-empty",
      error: "models-defaults-error",
      row: "models-defaults-row",
      operation: "models-defaults-operation",
      select: "models-defaults-select",
      source: "models-defaults-source",
      clearButton: "models-defaults-clear-button",
    },
    // Host-aware model picker (the menu behind every AI action). The static
    // ids live here; the per-model-id dynamic selectors are in the dynamic tree.
    pickerTrigger: "models-picker-trigger",
    picker: {
      sheet: "models-picker-sheet",
      host: "models-picker-host",
      loading: "models-picker-loading",
      error: "models-picker-error",
      footer: "models-picker-footer",
    },
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
  workspace: {
    root: "workspace-root",
    loading: "workspace-loading",
    error: "workspace-error",
    dropzone: "workspace-dropzone",
    fileInput: "workspace-file-input",
    operationSelect: "workspace-operation-select",
    opDescription: "workspace-op-description",
    overlayInput: "workspace-overlay-input",
    paramsForm: "workspace-params-form",
    applyButton: "workspace-apply-button",
    runError: "workspace-run-error",
    empty: "workspace-empty",
    modeSwitcher: "workspace-mode-switcher",
    inspector: "workspace-inspector",
    canvas: {
      root: "workspace-canvas",
      image: "workspace-canvas-image",
      zoomIn: "workspace-canvas-zoom-in",
      zoomOut: "workspace-canvas-zoom-out",
      zoomFit: "workspace-canvas-zoom-fit",
      zoomActual: "workspace-canvas-zoom-actual",
      compareToggle: "workspace-canvas-compare-toggle",
      replace: "workspace-canvas-replace",
      meta: "workspace-canvas-meta",
      metadataOutput: "workspace-canvas-metadata-output",
      progress: "workspace-canvas-progress",
    },
    history: {
      rail: "workspace-history-rail",
      list: "workspace-history-list",
      empty: "workspace-history-empty",
      source: "workspace-history-source",
    },
    actions: {
      bar: "workspace-action-bar",
      undo: "workspace-undo-button",
      redo: "workspace-redo-button",
      reset: "workspace-reset-button",
      download: "workspace-download-link",
    },
    compare: {
      root: "workspace-compare-root",
      slider: "workspace-compare-slider",
      before: "workspace-compare-before",
      after: "workspace-compare-after",
    },
    crop: {
      advanced: "workspace-crop-advanced",
      box: "workspace-crop-box",
      aspect: "workspace-crop-aspect",
    },
    enhance: {
      panel: "workspace-enhance-panel",
      intro: "workspace-enhance-intro",
      loading: "workspace-enhance-loading",
      error: "workspace-enhance-error",
      actions: "workspace-enhance-actions",
      scale: "workspace-enhance-scale",
      realism: "workspace-enhance-realism",
      faceAware: "workspace-enhance-face-aware",
      suggest: "workspace-enhance-naturalize-suggest",
      modelBadge: "workspace-enhance-model-badge",
      run: "workspace-enhance-run",
      progress: "workspace-enhance-progress",
      cancel: "workspace-enhance-cancel",
      retry: "workspace-enhance-retry",
      succeeded: "workspace-enhance-succeeded",
      failed: "workspace-enhance-failed",
      warnings: "workspace-enhance-warnings",
      installGate: "workspace-enhance-install-gate",
      install: "workspace-enhance-install",
    },
    create: {
      panel: "workspace-create-panel",
      intro: "workspace-create-intro",
      loading: "workspace-create-loading",
      error: "workspace-create-error",
      actions: "workspace-create-actions",
      prompt: "workspace-create-prompt",
      negative: "workspace-create-negative",
      size: "workspace-create-size",
      seed: "workspace-create-seed",
      seedLock: "workspace-create-seed-lock",
      seedRandomize: "workspace-create-seed-randomize",
      variations: "workspace-create-variations",
      model: "workspace-create-model",
      byok: "workspace-create-byok",
      advanced: "workspace-create-advanced",
      usesCurrent: "workspace-create-uses-current",
      needsImage: "workspace-create-needs-image",
      needsMask: "workspace-create-needs-mask",
      modelBadge: "workspace-create-model-badge",
      run: "workspace-create-run",
      progress: "workspace-create-progress",
      cancel: "workspace-create-cancel",
      retry: "workspace-create-retry",
      succeeded: "workspace-create-succeeded",
      failed: "workspace-create-failed",
      warnings: "workspace-create-warnings",
      installGate: "workspace-create-install-gate",
      install: "workspace-create-install",
      results: "workspace-create-results",
      consent: "workspace-create-consent",
      consentRequired: "workspace-create-consent-required",
      consentBlocked: "workspace-create-consent-blocked",
    },
    mask: {
      root: "workspace-mask",
      canvas: "workspace-mask-canvas",
      brushSize: "workspace-mask-brush-size",
      clear: "workspace-mask-clear",
      upload: "workspace-mask-upload",
      status: "workspace-mask-status",
    },
    analyze: {
      panel: "workspace-analyze-panel",
      intro: "workspace-analyze-intro",
      loading: "workspace-analyze-loading",
      error: "workspace-analyze-error",
      actions: "workspace-analyze-actions",
      run: "workspace-analyze-run",
      progress: "workspace-analyze-progress",
      installGate: "workspace-analyze-install-gate",
      install: "workspace-analyze-install",
      failed: "workspace-analyze-failed",
      retry: "workspace-analyze-retry",
      result: "workspace-analyze-result",
      probe: "workspace-analyze-probe",
      ocr: "workspace-analyze-ocr",
      ocrText: "workspace-analyze-ocr-text",
      copy: "workspace-analyze-copy",
      nsfw: "workspace-analyze-nsfw",
      overlay: "workspace-analyze-overlay",
      duplicate: "workspace-analyze-duplicate",
      duplicatePhash: "workspace-analyze-duplicate-phash",
      duplicateAhash: "workspace-analyze-duplicate-ahash",
      quality: "workspace-analyze-quality",
    },
  },
  pages: {
    home: "page-home",
    library: "page-library",
    activity: "page-activity",
    models: "page-models",
    settings: "page-settings",
    workspace: "page-workspace",
    select: "page-select",
    compare: "page-compare",
  },
  responsibleUse: {
    panel: "settings-responsible-use-panel",
    loading: "settings-responsible-use-loading",
    error: "settings-responsible-use-error",
    tier: "settings-responsible-use-tier",
    weights: "settings-responsible-use-weights",
    summary: "settings-responsible-use-summary",
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
      params: { key: { type: "enum", values: ["home", "workspace", "library", "select", "compare", "activity", "models", "settings"] as const } },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: { key: { type: "enum", values: ["home", "workspace", "library", "select", "compare", "activity", "models", "settings"] as const } },
    }),
  },
  select: {
    editButton: defineDynamicSelector({
      description: "Smart-select contextual edit button by edit id",
      testIdPattern: "select-edit-${id}",
      params: { id: { type: "string" } },
    }),
  },
  compare: {
    modeOption: defineDynamicSelector({
      description: "Compare mode segmented-control option by mode key",
      testIdPattern: "compare-mode-${mode}",
      params: { mode: { type: "enum", values: ["pixel", "perceptual"] as const } },
    }),
  },
  workspace: {
    fieldInput: defineDynamicSelector({
      description: "Workspace params-form input by proto field name",
      testIdPattern: "workspace-field-${name}",
      params: { name: { type: "string" } },
    }),
    modeOption: defineDynamicSelector({
      description: "Workspace mode tab by mode key",
      testIdPattern: "workspace-mode-${mode}",
      params: { mode: { type: "enum", values: ["edit", "enhance", "create", "analyze"] as const } },
    }),
    historyStep: defineDynamicSelector({
      description: "Workspace history step by 1-based index",
      testIdPattern: "workspace-history-step-${index}",
      params: { index: { type: "number" } },
    }),
    opOption: defineDynamicSelector({
      description: "Operation-picker radio button by op name",
      testIdPattern: "workspace-op-${name}",
      params: { name: { type: "string" } },
    }),
    enhanceAction: defineDynamicSelector({
      description: "Enhance one-tap action button by AI op name",
      testIdPattern: "workspace-enhance-action-${name}",
      params: { name: { type: "string" } },
    }),
    createAction: defineDynamicSelector({
      description: "Create generator-op radio button by AI op name",
      testIdPattern: "workspace-create-action-${name}",
      params: { name: { type: "string" } },
    }),
    createVariation: defineDynamicSelector({
      description: "Create result-grid variation slot by 1-based index",
      testIdPattern: "workspace-create-variation-${index}",
      params: { index: { type: "number" } },
    }),
    createSend: defineDynamicSelector({
      description: "Create variation 'send to canvas' button by 1-based index",
      testIdPattern: "workspace-create-send-${index}",
      params: { index: { type: "number" } },
    }),
    analyzeAction: defineDynamicSelector({
      description: "Analyze one-tap action button by analysis op name",
      testIdPattern: "workspace-analyze-action-${name}",
      params: { name: { type: "string" } },
    }),
    crop: {
      handle: defineDynamicSelector({
        description: "Crop-box resize handle by corner",
        testIdPattern: "workspace-crop-handle-${corner}",
        params: {
          corner: { type: "enum", values: ["nw", "ne", "sw", "se"] as const },
        },
      }),
    },
  },
  home: {
    tile: defineDynamicSelector({
      description: "Home intent tile by op/action name",
      testIdPattern: "home-tile-${name}",
      params: { name: { type: "string" } },
    }),
    sample: defineDynamicSelector({
      description: "Home sample-image button by sample key",
      testIdPattern: "home-sample-${key}",
      params: { key: { type: "enum", values: ["product", "portrait", "blurry", "receipt"] as const } },
    }),
    recentItem: defineDynamicSelector({
      description: "Home recent-rail thumbnail link by 1-based index",
      testIdPattern: "home-recent-${index}",
      params: { index: { type: "number" } },
    }),
  },
  library: {
    item: defineDynamicSelector({
      description: "Library grid item by 1-based index",
      testIdPattern: "library-item-${index}",
      params: { index: { type: "number" } },
    }),
    open: defineDynamicSelector({
      description: "Library 'open in workspace' button by 1-based index",
      testIdPattern: "library-open-${index}",
      params: { index: { type: "number" } },
    }),
    select: defineDynamicSelector({
      description: "Library item selection checkbox by 1-based index",
      testIdPattern: "library-select-${index}",
      params: { index: { type: "number" } },
    }),
  },
  looks: {
    card: defineDynamicSelector({
      description: "Look gallery card by 1-based index",
      testIdPattern: "looks-card-${index}",
      params: { index: { type: "number" } },
    }),
    preview: defineDynamicSelector({
      description: "Look 'render preview' button by 1-based index",
      testIdPattern: "looks-preview-${index}",
      params: { index: { type: "number" } },
    }),
  },
  activity: {
    openOutput: defineDynamicSelector({
      description: "Activity row 'open output' button by 1-based index",
      testIdPattern: "activity-open-${index}",
      params: { index: { type: "number" } },
    }),
  },
  models: {
    pickerRow: defineDynamicSelector({
      description: "Model-picker candidate row by model id",
      testIdPattern: "models-picker-row-${id}",
      params: { id: { type: "string" } },
    }),
    pickerSelect: defineDynamicSelector({
      description: "Model-picker 'use this model' button by model id",
      testIdPattern: "models-picker-select-${id}",
      params: { id: { type: "string" } },
    }),
    pickerInUse: defineDynamicSelector({
      description: "Model-picker 'in use' marker by model id",
      testIdPattern: "models-picker-inuse-${id}",
      params: { id: { type: "string" } },
    }),
    pickerInstallModel: defineDynamicSelector({
      description: "Model-picker download-weights button by model id",
      testIdPattern: "models-picker-install-model-${id}",
      params: { id: { type: "string" } },
    }),
    pickerInstallBackend: defineDynamicSelector({
      description: "Model-picker install-engine button by model id",
      testIdPattern: "models-picker-install-backend-${id}",
      params: { id: { type: "string" } },
    }),
    pickerEnable: defineDynamicSelector({
      description: "Model-picker enable-model button by model id",
      testIdPattern: "models-picker-enable-${id}",
      params: { id: { type: "string" } },
    }),
    pickerManualToggle: defineDynamicSelector({
      description: "Model-picker manual-setup toggle by model id",
      testIdPattern: "models-picker-manual-toggle-${id}",
      params: { id: { type: "string" } },
    }),
    pickerManual: defineDynamicSelector({
      description: "Model-picker manual-setup panel by model id",
      testIdPattern: "models-picker-manual-${id}",
      params: { id: { type: "string" } },
    }),
    pickerRowError: defineDynamicSelector({
      description: "Model-picker per-row error by model id",
      testIdPattern: "models-picker-row-error-${id}",
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
    fontScaleOption: defineDynamicSelector({
      description: "Font-scale choice radio button on the settings page",
      testIdPattern: "page-settings-font-scale-${choice}",
      params: { choice: { type: "enum", values: ["small", "default", "large", "xlarge"] as const } },
    }),
    reducedMotionOption: defineDynamicSelector({
      description: "Reduced-motion choice radio button on the settings page",
      testIdPattern: "page-settings-reduced-motion-${choice}",
      params: { choice: { type: "enum", values: ["system", "always", "never"] as const } },
    }),
    textDirectionOption: defineDynamicSelector({
      description: "Text-direction choice radio button on the settings page",
      testIdPattern: "page-settings-text-direction-${choice}",
      params: { choice: { type: "enum", values: ["auto", "ltr", "rtl"] as const } },
    }),
    handednessOption: defineDynamicSelector({
      description: "Handedness choice radio button on the settings page",
      testIdPattern: "page-settings-handedness-${choice}",
      params: { choice: { type: "enum", values: ["left", "right"] as const } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
