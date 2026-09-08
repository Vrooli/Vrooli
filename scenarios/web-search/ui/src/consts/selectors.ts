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
    dashboard: "page-dashboard",
    settings: "page-settings",
    search: "page-search",
    findings: "page-findings",
    disputes: "page-disputes",
    ops: "page-ops",
  },
  search: {
    form: "search-form",
    input: "search-input",
    submit: "search-submit",
    modeLive: "search-mode-live",
    modeLearnings: "search-mode-learnings",
    synthesizeToggle: "search-synthesize-toggle",
    loading: "search-loading",
    error: "search-error",
    empty: "search-empty",
    method: "search-method",
    cached: "search-cached",
    degraded: "search-degraded",
    engineWarning: "search-engine-warning",
    results: "search-results",
    result: "search-result",
    synthesis: "search-synthesis",
    findingHits: "search-finding-hits",
    findingHit: "search-finding-hit",
  },
  history: {
    panel: "history-panel",
    empty: "history-empty",
    list: "history-list",
    item: "history-item",
    clear: "history-clear",
  },
  ops: {
    panel: "ops-panel",
    dependencies: "ops-dependencies",
    dependenciesLoading: "ops-dependencies-loading",
    dependenciesError: "ops-dependencies-error",
    dependenciesEmpty: "ops-dependencies-empty",
    dependency: "ops-dependency",
    lastQuery: "ops-last-query",
    lastQueryEmpty: "ops-last-query-empty",
    captureStatus: "ops-capture-status",
  },
  findings: {
    panel: "findings-panel",
    list: "findings-list",
    loading: "findings-loading",
    error: "findings-error",
    empty: "findings-empty",
    item: "findings-item",
    age: "findings-age",
    statusBadge: "findings-status-badge",
    includeArchived: "findings-include-archived",
    editButton: "findings-edit-button",
    editForm: "findings-edit-form",
    editClaim: "findings-edit-claim",
    editConfidence: "findings-edit-confidence",
    editSave: "findings-edit-save",
    editCancel: "findings-edit-cancel",
    supersedeButton: "findings-supersede-button",
    supersedeForm: "findings-supersede-form",
    flagButton: "findings-flag-button",
    flagForm: "findings-flag-form",
    correctionButton: "findings-correction-button",
    correctionForm: "findings-correction-form",
    correctionDisposition: "findings-correction-disposition",
    correctionReason: "findings-correction-reason",
    correctionSubmit: "findings-correction-submit",
    addForm: "findings-add-form",
    addClaim: "findings-add-claim",
    addConfidence: "findings-add-confidence",
    addQuery: "findings-add-query",
    addCitationUrl: "findings-add-citation-url",
    addCitationTitle: "findings-add-citation-title",
    addCitationRow: "findings-add-citation-row",
    addSubmit: "findings-add-submit",
    pruneDryRun: "findings-prune-dry-run",
    pruneApply: "findings-prune-apply",
    pruneResult: "findings-prune-result",
    effectivenessPanel: "findings-effectiveness-panel",
    effectivenessList: "findings-effectiveness-list",
    effectivenessItem: "findings-effectiveness-item",
    effectivenessLoading: "findings-effectiveness-loading",
    effectivenessError: "findings-effectiveness-error",
    effectivenessEmpty: "findings-effectiveness-empty",
  },
  research: {
    panel: "research-panel",
    query: "research-query",
    start: "research-start",
    resume: "research-resume",
    status: "research-status",
    resultStatus: "research-result-status",
    gaps: "research-gaps",
    summary: "research-summary",
    passageId: "research-passage-id",
    readPassage: "research-read-passage",
    passage: "research-passage",
    error: "research-error",
  },
  disputes: {
    panel: "disputes-panel",
    list: "disputes-list",
    loading: "disputes-loading",
    error: "disputes-error",
    empty: "disputes-empty",
    item: "disputes-item",
    note: "disputes-note",
    resolveButton: "disputes-resolve-button",
    resolveForm: "disputes-resolve-form",
    replacement: "disputes-replacement",
    reason: "disputes-reason",
    resolveSubmit: "disputes-resolve-submit",
    dismissButton: "disputes-dismiss-button",
    dismissError: "disputes-dismiss-error",
    reresearchButton: "disputes-reresearch-button",
    reresearchStatus: "disputes-reresearch-status",
    reresearchError: "disputes-reresearch-error",
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
          values: ["dashboard", "search", "findings", "disputes", "ops", "settings"] as const,
        },
      },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: ["dashboard", "search", "findings", "disputes", "ops", "settings"] as const,
        },
      },
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
