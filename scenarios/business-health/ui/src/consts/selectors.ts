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
    matrix: "page-matrix",
    fleet: "page-fleet",
    wizard: "page-wizard",
    findings: "page-findings",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  // Shared scenario picker (text input + persisted recents) reused by the
  // matrix, wizard, and findings surfaces.
  scenarioPicker: {
    form: "scenario-picker-form",
    input: "scenario-picker-input",
    submit: "scenario-picker-submit",
    recent: "scenario-picker-recent",
    clear: "scenario-picker-clear",
  },
  matrix: {
    loading: "matrix-loading",
    error: "matrix-error",
    empty: "matrix-empty",
    degradedBanner: "matrix-degraded-banner",
    registrySummary: "matrix-registry-summary",
    grid: "matrix-grid",
    drawer: "matrix-drawer",
    drawerClose: "matrix-drawer-close",
    drawerTitle: "matrix-drawer-title",
    evidenceDetail: "matrix-evidence-detail",
    attestForm: "matrix-attest-form",
    attestBy: "matrix-attest-by",
    attestNotes: "matrix-attest-notes",
    attestSubmit: "matrix-attest-submit",
    attestError: "matrix-attest-error",
  },
  fleet: {
    loading: "fleet-loading",
    error: "fleet-error",
    empty: "fleet-empty",
    refresh: "fleet-refresh",
    asOf: "fleet-as-of",
    table: "fleet-table",
    filterText: "fleet-filter-text",
    filterStarter: "fleet-filter-starter",
    filterLaggard: "fleet-filter-laggard",
    filterUnproven: "fleet-filter-unproven",
    tiles: "fleet-tiles",
    tileScanned: "fleet-tile-scanned",
    tilePassing: "fleet-tile-passing",
    tileStarter: "fleet-tile-starter",
    tileLaggard: "fleet-tile-laggard",
    errors: "fleet-errors",
  },
  wizard: {
    loading: "wizard-loading",
    error: "wizard-error",
    resetToggle: "wizard-reset-toggle",
    progress: "wizard-progress",
    question: "wizard-question",
    answerText: "wizard-answer-text",
    otList: "wizard-ot-list",
    otAdd: "wizard-ot-add",
    invalid: "wizard-invalid",
    submit: "wizard-submit",
    sectionPreview: "wizard-section-preview",
    previewButton: "wizard-preview-button",
    preview: "wizard-preview",
    apply: "wizard-apply",
    applyResult: "wizard-apply-result",
    hints: "wizard-hints",
  },
  findings: {
    loading: "findings-loading",
    error: "findings-error",
    empty: "findings-empty",
    list: "findings-list",
    remediation: "findings-remediation",
    docLink: "findings-doc-link",
    previewFix: "findings-preview-fix",
    applyFix: "findings-apply-fix",
    fixDiff: "findings-fix-diff",
    fixMessages: "findings-fix-messages",
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
          values: [
            "dashboard",
            "matrix",
            "fleet",
            "wizard",
            "findings",
            "settings",
          ] as const,
        },
      },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "matrix",
            "fleet",
            "wizard",
            "findings",
            "settings",
          ] as const,
        },
      },
    }),
  },
  scenarioPicker: {
    recentItem: defineDynamicSelector({
      description: "Recent-scenario chip in the scenario picker",
      testIdPattern: "scenario-picker-recent-${scenario}",
      params: { scenario: { type: "string" } },
    }),
  },
  matrix: {
    otGroup: defineDynamicSelector({
      description: "Operational-target group header row in the matrix",
      testIdPattern: "matrix-ot-group-${otId}",
      params: { otId: { type: "string" } },
    }),
    requirementRow: defineDynamicSelector({
      description: "Requirement row in the traceability matrix",
      testIdPattern: "matrix-row-${requirementId}",
      params: { requirementId: { type: "string" } },
    }),
    drillButton: defineDynamicSelector({
      description: "Drill-in trigger opening the requirement detail drawer",
      testIdPattern: "matrix-drill-${requirementId}",
      params: { requirementId: { type: "string" } },
    }),
  },
  fleet: {
    row: defineDynamicSelector({
      description: "Fleet table row by scenario slug",
      testIdPattern: "fleet-row-${scenario}",
      params: { scenario: { type: "string" } },
    }),
  },
  wizard: {
    step: defineDynamicSelector({
      description: "Wizard question step by question id",
      testIdPattern: "wizard-step-${id}",
      params: { id: { type: "string" } },
    }),
    otEntry: defineDynamicSelector({
      description: "One operational-target entry row in an ot_list answer",
      testIdPattern: "wizard-ot-entry-${index}",
      params: { index: { type: "number" } },
    }),
  },
  findings: {
    item: defineDynamicSelector({
      description: "Finding row by finding code",
      testIdPattern: "findings-item-${code}",
      params: { code: { type: "string" } },
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
