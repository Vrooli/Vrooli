import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */
import { LOCALE_CODES } from "../i18n/locales";

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { createSelectorRegistry, defineDynamicSelector } from "@vrooli/ui-selectors";

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
  runs: {
    surface: "runs-surface",
    list: "runs-list",
    loading: "runs-loading",
    empty: "runs-empty",
    error: "runs-error",
    row: "runs-row",
    startCard: "runs-start-card",
    startForm: "runs-start-form",
    startKind: "runs-start-kind",
    startSubject: "runs-start-subject",
    startGolden: "runs-start-golden",
    startForce: "runs-start-force",
    startSubmit: "runs-start-submit",
    startError: "runs-start-error",
    detail: "runs-detail",
    detailBack: "runs-detail-back",
    detailHeading: "runs-detail-heading",
    detailStatus: "runs-detail-status",
    detailVerdict: "runs-detail-verdict",
    detailError: "runs-detail-error",
    runValidation: "runs-run-validation",
  },
  nav: {
    sidebar: "nav-sidebar",
    sidebarLogo: "nav-sidebar-logo",
    sidebarCollapseToggle: "nav-sidebar-collapse",
    sidebarItemGoldens: "nav-sidebar-goldens",
    sidebarItemSkills: "nav-sidebar-skills",
    sidebarItemManifests: "nav-sidebar-manifests",
    sidebarItemRuns: "nav-sidebar-runs",
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
    mobileBottomItemRuns: "nav-mobile-runs",
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

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
