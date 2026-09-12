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

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
