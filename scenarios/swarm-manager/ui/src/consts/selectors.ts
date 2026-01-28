/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. Selectors are defined as typed constant objects
 * to ensure TypeScript can statically verify all accesses.
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

// =============================================================================
// Dynamic Selector Types
// =============================================================================

type ParamType = "string" | "number" | "enum";

type ParamDefinition =
  | { readonly type: "string" }
  | { readonly type: "number" }
  | { readonly type: "enum"; readonly values: readonly (string | number)[] };

type ParamSchema = Readonly<Record<string, ParamDefinition>>;

interface DynamicSelectorDefinition<P extends ParamSchema | undefined = undefined> {
  readonly kind: "dynamic-selector";
  readonly description: string;
  readonly params?: P;
  readonly testIdPattern?: string;
  readonly selectorPattern?: string;
}

// =============================================================================
// Literal Selectors - Static test IDs with deterministic types
// =============================================================================

/**
 * Literal (static) selectors organized by UI area.
 * Each value is a data-testid string.
 */
export const literalSelectors = {
  // Layout selectors
  layout: {
    main: "main-layout",
    header: "header",
    desktopTabs: "desktop-tabs",
    mobileNav: "mobile-nav",
  },
  // Error state selectors (shared across pages)
  error: {
    container: "error-state",
    icon: "error-icon",
    title: "error-title",
    message: "error-message",
    retryButton: "error-retry",
  },
  // Error boundary selectors (for runtime errors)
  errorBoundary: {
    container: "error-boundary",
    title: "error-boundary-title",
    message: "error-boundary-message",
    refreshButton: "error-boundary-refresh",
  },
  // Not Found page selectors
  notFound: {
    page: "not-found-page",
    title: "not-found-title",
    message: "not-found-message",
    homeButton: "not-found-home",
  },
  // Desktop tab selectors
  tabs: {
    ideas: "tab-ideas",
    scenarios: "tab-scenarios",
    recommendations: "tab-recommendations",
    settings: "tab-settings",
  },
  // Mobile tab selectors
  mobileTabs: {
    ideas: "mobile-tab-ideas",
    scenarios: "mobile-tab-scenarios",
    recommendations: "mobile-tab-recommendations",
    settings: "mobile-tab-settings",
  },
  // Ideas page selectors
  ideas: {
    page: "ideas-page",
    search: "ideas-search",
    filter: "ideas-filter",
    createButton: "create-idea",
    createFirstButton: "create-first-idea",
    empty: "ideas-empty",
    grid: "ideas-grid",
    // Experience architecture additions (Phase 29)
    continueSection: "ideas-continue-section",
    continueList: "ideas-continue-list",
    summaryStats: "ideas-summary-stats",
    readyCount: "ideas-ready-count",
    cliHint: "ideas-cli-hint",
    // Iteration 5 additions
    statusLegend: "ideas-status-legend",
    welcomeHint: "ideas-welcome-hint",
  },
  // Idea details page selectors
  ideaDetails: {
    page: "idea-details-page",
    header: "idea-details-header",
    title: "idea-details-title",
    description: "idea-details-description",
    backButton: "idea-details-back",
    editButton: "idea-details-edit",
    deleteButton: "idea-details-delete",
    queueButton: "idea-details-queue",
    fileTree: "idea-details-file-tree",
    filePreview: "idea-details-file-preview",
    fileUpload: "idea-details-file-upload",
    uploadDropzone: "file-upload-dropzone",
    uploadList: "file-upload-list",
    // Experience architecture additions (Phase 29)
    breadcrumb: "idea-details-breadcrumb",
  },
  // Scenarios page selectors
  scenarios: {
    page: "scenarios-page",
    search: "scenarios-search",
    filter: "scenarios-filter",
    filterDropdown: "scenarios-filter-dropdown",
    statusFilter: "scenarios-status-filter",
    count: "scenarios-count",
    empty: "scenarios-empty",
    noResults: "scenarios-no-results",
    list: "scenarios-list",
    // Experience architecture additions (Phase 29)
    statusSummary: "scenarios-status-summary",
    runningCount: "scenarios-running-count",
    stoppedCount: "scenarios-stopped-count",
    errorCount: "scenarios-error-count",
  },
  // Scenario details page selectors
  // [REQ:REQ-P0-007] Selectors for scenario metadata management UI
  // [REQ:REQ-P0-008] Selectors for scenario deletion UI
  scenarioDetails: {
    page: "scenario-details-page",
    header: "scenario-details-header",
    title: "scenario-details-title",
    description: "scenario-details-description",
    backButton: "scenario-details-back",
    metadataSection: "scenario-details-metadata",
    greenfieldToggle: "scenario-greenfield-toggle",
    recommendationsToggle: "scenario-recommendations-toggle",
    saveButton: "scenario-details-save",
    priority: "scenario-details-priority",
    tags: "scenario-details-tags",
    status: "scenario-details-status",
    deleteButton: "scenario-details-delete",
    deleteDialog: "scenario-delete-dialog",
    deleteConfirmButton: "scenario-delete-confirm",
    deleteCancelButton: "scenario-delete-cancel",
    archiveCheckbox: "scenario-delete-archive",
    // Experience architecture additions (Phase 29)
    breadcrumb: "scenario-details-breadcrumb",
    // Iteration 5 additions
    cliHint: "scenario-details-cli-hint",
  },
  // Recommendations page selectors
  recommendations: {
    page: "recommendations-page",
    filter: "recommendations-filter",
    empty: "recommendations-empty",
    list: "recommendations-list",
    // Experience architecture additions (Phase 29)
    settingsLink: "recommendations-settings-link",
  },
  // Settings page selectors
  settings: {
    page: "settings-page",
    themeSettings: "theme-settings",
    themeDark: "theme-dark",
    themeLight: "theme-light",
    themeSystem: "theme-system",
    recommendationSettings: "recommendation-settings",
    recModeOff: "rec-mode-off",
    recModeSuggestions: "rec-mode-suggestions",
    recModeYolo: "rec-mode-yolo",
    customFocus: "custom-focus",
    insightsSettings: "insights-settings",
    insightsEnabled: "insights-enabled",
    insightsAutoAnalyze: "insights-auto-analyze",
    saveButton: "settings-save",
    // Experience architecture additions (Phase 29)
    recModeHint: "rec-mode-hint",
  },
} as const;

// =============================================================================
// Dynamic Selectors - Parameterized selectors for data-driven elements
// =============================================================================

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match, token: string) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const defineDynamicSelector = <P extends ParamSchema | undefined>(
  definition: Omit<DynamicSelectorDefinition<P>, "kind">,
): DynamicSelectorDefinition<P> => ({
  ...definition,
  kind: "dynamic-selector",
});

// Dynamic selector definitions (used for manifest generation)
export const dynamicSelectorDefinitions = {
  ideas: {
    cardByName: defineDynamicSelector({
      description: "Idea card filtered by name",
      testIdPattern: "idea-card-${name}",
      params: { name: { type: "string" } },
    }),
  },
  scenarios: {
    cardByName: defineDynamicSelector({
      description: "Scenario card filtered by name",
      testIdPattern: "scenario-card-${name}",
      params: { name: { type: "string" } },
    }),
  },
  recommendations: {
    cardByName: defineDynamicSelector({
      description: "Recommendation card filtered by name",
      testIdPattern: "recommendation-card-${name}",
      params: { name: { type: "string" } },
    }),
  },
} as const;

// =============================================================================
// Dynamic Selector Functions
// =============================================================================

/**
 * Dynamic selectors - functions that generate test IDs from parameters
 */
export const dynamicSelectors = {
  ideas: {
    cardByName: (params: { name: string }) =>
      formatTemplate("idea-card-${name}", params, "ideas.cardByName"),
  },
  scenarios: {
    cardByName: (params: { name: string }) =>
      formatTemplate("scenario-card-${name}", params, "scenarios.cardByName"),
  },
  recommendations: {
    cardByName: (params: { name: string }) =>
      formatTemplate("recommendation-card-${name}", params, "recommendations.cardByName"),
  },
} as const;

// =============================================================================
// Combined Selectors Export
// =============================================================================

/**
 * Combined selector registry - literal selectors merged with dynamic functions.
 * This is the primary export for UI components.
 *
 * Usage:
 * - Literal: selectors.ideas.page
 * - Dynamic: selectors.ideas.cardByName({ name: "my-idea" })
 */
export const selectors = {
  ...literalSelectors,
  ideas: {
    ...literalSelectors.ideas,
    ...dynamicSelectors.ideas,
  },
  scenarios: {
    ...literalSelectors.scenarios,
    ...dynamicSelectors.scenarios,
  },
  recommendations: {
    ...literalSelectors.recommendations,
    ...dynamicSelectors.recommendations,
  },
} as const;

export type Selectors = typeof selectors;

// =============================================================================
// Manifest Generation (for workflow tools)
// =============================================================================

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

type LiteralSelectorTree = { readonly [key: string]: string | LiteralSelectorTree };

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
    flattenLiteralSelectors(value as LiteralSelectorTree, nextPath, target);
  }
  return target;
};

const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<ParamSchema | undefined> =>
  Boolean(value && typeof value === "object" && (value as DynamicSelectorDefinition<ParamSchema | undefined>).kind === "dynamic-selector");

type DynamicSelectorBranch = {
  readonly [key: string]: DynamicSelectorBranch | DynamicSelectorDefinition<ParamSchema | undefined>;
};

const flattenDynamicSelectors = (
  tree: DynamicSelectorBranch,
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
    flattenDynamicSelectors(value as DynamicSelectorBranch, nextPath, target);
  }
  return target;
};

export const selectorsManifest = {
  selectors: flattenLiteralSelectors(literalSelectors),
  dynamicSelectors: flattenDynamicSelectors(dynamicSelectorDefinitions),
};
