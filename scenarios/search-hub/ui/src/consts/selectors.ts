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
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
  },
  search: {
    panel: "search-panel",
    input: "search-input",
    submit: "search-submit",
    expandToggle: "search-expand-toggle",
    typeFacets: "search-type-facets",
    loading: "search-loading",
    error: "search-error",
    empty: "search-empty",
    noResults: "search-no-results",
    summary: "search-summary",
    results: "search-results",
    routing: "search-routing",
    statusBar: "search-status-bar",
  },
  evals: {
    panel: "evals-panel",
    loading: "evals-loading",
    error: "evals-error",
    suiteList: "evals-suite-list",
    noSuites: "evals-no-suites",
    selectSuite: "evals-select-suite",
    runHistory: "evals-run-history",
    noRuns: "evals-no-runs",
    trend: "evals-trend",
    compareButton: "evals-compare-button",
    compareResult: "evals-compare-result",
  },
  pages: {
    search: "page-search",
    evals: "page-evals",
    dashboard: "page-dashboard",
    settings: "page-settings",
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
  search: {
    typeFacet: defineDynamicSelector({
      description: "Search type facet by result type",
      testIdPattern: "search-type-facet-${type}",
      params: { type: { type: "string" } },
    }),
    providerChip: defineDynamicSelector({
      description: "Search provider status chip by provider id",
      testIdPattern: "search-provider-chip-${providerId}",
      params: { providerId: { type: "string" } },
    }),
  },
  evals: {
    suiteItem: defineDynamicSelector({
      description: "Evaluation suite item by suite id",
      testIdPattern: "evals-suite-item-${suiteId}",
      params: { suiteId: { type: "string" } },
    }),
    runRow: defineDynamicSelector({
      description: "Evaluation run row by run id",
      testIdPattern: "evals-run-row-${runId}",
      params: { runId: { type: "string" } },
    }),
    runSelect: defineDynamicSelector({
      description: "Evaluation run selection control by run id",
      testIdPattern: "evals-run-select-${runId}",
      params: { runId: { type: "string" } },
    }),
  },
  layout: {
    navLink: defineDynamicSelector({
      description: "App shell navigation link by canonical nav key",
      testIdPattern: "layout-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "search",
            "evals",
            "dashboard",
            "settings",          ] as const,
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
