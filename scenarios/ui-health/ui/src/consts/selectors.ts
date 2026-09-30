import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */
import { LOCALE_CODES } from "../i18n/locales";

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { createSelectorRegistry, defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  app: {
    title: "app-title",
    brand: "app-brand",
    eyebrow: "app-eyebrow",
    description: "app-description",
  },
  health: {
    pill: "health-pill",
  },
  notifications: {
    summary: "notifications-summary",
  },
  locale: {
    switcher: "locale-switcher",
  },
  layout: {
    shell: "layout-shell",
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
    toggle: "theme-toggle",
  },
  inspector: {
    panel: "inspector-panel",
    title: "inspector-title",
    close: "inspector-close",
    backdrop: "inspector-backdrop",
  },
  pages: {
    dashboard: "page-dashboard",
    validation: "page-validation",
    validationDetail: "page-validation-detail",
    captures: "page-captures",
    search: "page-search",
    inventory: "page-inventory",
    surfaceDetail: "page-surface-detail",
    reindex: "page-reindex",
    reindexJob: "page-reindex-job",
    settings: "page-settings",
    notFound: "page-not-found",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  dashboard: {
    stats: {
      scenariosValidated: "dashboard-stat-scenarios-validated",
      surfacesIndexed: "dashboard-stat-surfaces-indexed",
      openIssues: "dashboard-stat-open-issues",
    },
    activity: {
      list: "dashboard-activity-list",
      empty: "dashboard-activity-empty",
    },
    quickActions: {
      search: "dashboard-quick-action-search",
      validate: "dashboard-quick-action-validate",
      reindex: "dashboard-quick-action-reindex",
      inventory: "dashboard-quick-action-inventory",
    },
    apiStatus: {
      card: "dashboard-api-status",
      dependency: "dashboard-api-dependency",
    },
  },
  validation: {
    form: "validation-form",
    scenarioInput: "validation-scenario-input",
    submit: "validation-submit",
    recentList: "validation-recent-list",
    emptyRecent: "validation-recent-empty",
    detail: {
      statusBadge: "validation-detail-status",
      summary: "validation-detail-summary",
      revalidate: "validation-detail-revalidate",
      findings: "validation-detail-findings",
      empty: "validation-detail-empty",
      error: "validation-detail-error",
      loading: "validation-detail-loading",
    },
  },
  captures: {
    gallery: "captures-gallery",
    scenarioSelect: "captures-scenario-select",
    overlayList: "captures-overlay-list",
    empty: "captures-empty",
  },
  search: {
    input: "search-input",
    filters: "search-filters",
    loading: "search-loading",
    error: "search-error",
    empty: "search-empty",
    shortQuery: "search-short-query",
    noResults: "search-no-results",
    noResultsForFilter: "search-no-results-filter",
    resultsList: "search-results-list",
    resultsSummary: "search-results-summary",
  },
  inventory: {
    form: "inventory-form",
    scenarioInput: "inventory-scenario-input",
    submit: "inventory-submit",
    filters: "inventory-filters",
    loading: "inventory-loading",
    error: "inventory-error",
    empty: "inventory-empty",
    noSurfaces: "inventory-no-surfaces",
    noResultsForFilter: "inventory-no-results-filter",
    surfacesTable: "inventory-surfaces-table",
    summary: "inventory-summary",
    detail: {
      back: "inventory-detail-back",
      meta: "inventory-detail-meta",
      provenance: "inventory-detail-provenance",
      widgets: "inventory-detail-widgets",
      empty: "inventory-detail-empty",
      error: "inventory-detail-error",
      loading: "inventory-detail-loading",
      notFound: "inventory-detail-not-found",
    },
  },
  reindex: {
    form: "reindex-form",
    scenarioInput: "reindex-scenario-input",
    dryRunInput: "reindex-dry-run-input",
    submit: "reindex-submit",
    confirmModal: "reindex-confirm-modal",
    confirmAccept: "reindex-confirm-accept",
    confirmCancel: "reindex-confirm-cancel",
    error: "reindex-error",
    jobsList: "reindex-jobs-list",
    emptyJobs: "reindex-jobs-empty",
    clearTerminal: "reindex-clear-terminal",
    detail: {
      back: "reindex-detail-back",
      meta: "reindex-detail-meta",
      progress: "reindex-detail-progress",
      error: "reindex-detail-error",
      cancel: "reindex-detail-cancel",
      notFound: "reindex-detail-not-found",
    },
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
    navLink: defineDynamicSelector({
      description: "Library shell navigation link by canonical nav key",
      testIdPattern: "layout-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: ["dashboard", "validation", "captures", "search", "inventory", "reindex", "settings"] as const,
        },
      },
    }),
  },
  validation: {
    recentRow: defineDynamicSelector({
      description: "Recent validation runs row by scenario name",
      testIdPattern: "validation-recent-row-${scenario}",
      params: { scenario: { type: "string" } },
    }),
    severityFilter: defineDynamicSelector({
      description: "Filter chip on validation detail by severity",
      testIdPattern: "validation-detail-filter-${severity}",
      params: {
        severity: { type: "enum", values: ["all", "error", "warning", "info"] as const },
      },
    }),
    findingRow: defineDynamicSelector({
      description: "Validation finding row by zero-based index",
      testIdPattern: "validation-detail-finding-${index}",
      params: { index: { type: "number" } },
    }),
  },
  captures: {
    captureCard: defineDynamicSelector({
      description: "Capture gallery card by capture id",
      testIdPattern: "captures-card-${captureId}",
      params: { captureId: { type: "string" } },
    }),
    overlay: defineDynamicSelector({
      description: "Capture gallery violation overlay by zero-based index",
      testIdPattern: "captures-overlay-${index}",
      params: { index: { type: "number" } },
    }),
  },
  search: {
    kindFilter: defineDynamicSelector({
      description: "Search surface-kind filter chip",
      testIdPattern: "search-kind-filter-${kind}",
      params: {
        kind: {
          type: "enum",
          values: ["all", "component", "page", "feature", "hook", "layout", "other"] as const,
        },
      },
    }),
    resultRow: defineDynamicSelector({
      description: "Search result row by zero-based index",
      testIdPattern: "search-result-row-${index}",
      params: { index: { type: "number" } },
    }),
    resultScore: defineDynamicSelector({
      description: "Search result score by zero-based index",
      testIdPattern: "search-result-score-${index}",
      params: { index: { type: "number" } },
    }),
    resultProvenance: defineDynamicSelector({
      description: "Search result provenance badge by zero-based index",
      testIdPattern: "search-result-provenance-${index}",
      params: { index: { type: "number" } },
    }),
    resultOpen: defineDynamicSelector({
      description: "Search result open-in-inventory link by zero-based index",
      testIdPattern: "search-result-open-${index}",
      params: { index: { type: "number" } },
    }),
  },
  dashboard: {
    activityRow: defineDynamicSelector({
      description: "Dashboard activity feed row by zero-based index",
      testIdPattern: "dashboard-activity-row-${index}",
      params: { index: { type: "number" } },
    }),
  },
  inventory: {
    kindFilter: defineDynamicSelector({
      description: "Inventory surface-kind filter chip",
      testIdPattern: "inventory-kind-filter-${kind}",
      params: {
        kind: {
          type: "enum",
          values: ["all", "component", "page", "feature", "hook", "layout", "other"] as const,
        },
      },
    }),
    surfaceRow: defineDynamicSelector({
      description: "Inventory surfaces table row by zero-based index",
      testIdPattern: "inventory-surface-row-${index}",
      params: { index: { type: "number" } },
    }),
    surfaceOpen: defineDynamicSelector({
      description: "Inventory surfaces table open-detail link by zero-based index",
      testIdPattern: "inventory-surface-open-${index}",
      params: { index: { type: "number" } },
    }),
  },
  reindex: {
    jobRow: defineDynamicSelector({
      description: "Reindex jobs list row by zero-based index",
      testIdPattern: "reindex-job-row-${index}",
      params: { index: { type: "number" } },
    }),
    jobOpen: defineDynamicSelector({
      description: "Reindex job detail link by zero-based index",
      testIdPattern: "reindex-job-open-${index}",
      params: { index: { type: "number" } },
    }),
    jobState: defineDynamicSelector({
      description: "Reindex job state badge by zero-based index",
      testIdPattern: "reindex-job-state-${index}",
      params: { index: { type: "number" } },
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

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
