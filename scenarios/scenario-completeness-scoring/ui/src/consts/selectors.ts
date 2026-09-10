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
  scoring: {
    dashboard: "scoring-dashboard",
    form: "scoring-form",
    input: "scoring-input",
    submit: "scoring-submit",
    loading: "scoring-loading",
    error: "scoring-error",
    empty: "scoring-empty",
    maturity: {
      card: "scoring-maturity-card",
      workingRung: "scoring-maturity-working-rung",
      ladderClean: "scoring-maturity-ladder-clean",
      satisfiedThrough: "scoring-maturity-satisfied-through",
      build: "scoring-maturity-build",
      digest: "scoring-maturity-digest",
      dimensions: "scoring-maturity-dimensions",
    },
    composite: {
      card: "scoring-composite-card",
      score: "scoring-composite-score",
      classification: "scoring-composite-classification",
    },
    freshness: {
      card: "scoring-freshness-card",
      refreshCommand: "scoring-freshness-refresh-command",
    },
    importance: {
      card: "scoring-importance-card",
      score: "scoring-importance-score",
      signals: "scoring-importance-signals",
    },
    recommendations: {
      card: "scoring-recommendations-card",
    },
    actionPlan: {
      card: "scoring-action-plan-card",
      projected: "scoring-action-plan-projected",
    },
    degradations: {
      card: "scoring-degradations-card",
    },
    trend: {
      card: "scoring-trend-card",
      delta: "scoring-trend-delta",
      series: "scoring-trend-series",
    },
    fleet: {
      card: "scoring-fleet-card",
      table: "scoring-fleet-table",
      next: "scoring-fleet-next",
      empty: "scoring-fleet-empty",
    },
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
  pages: {
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
  scoring: {
    freshnessPhaseRow: defineDynamicSelector({
      description: "Freshness verdict row by required-phase name",
      testIdPattern: "scoring-freshness-phase-${phase}",
      params: { phase: { type: "string" } },
    }),
    compositeGroup: defineDynamicSelector({
      description: "Composite score group section by stable group id",
      testIdPattern: "scoring-composite-group-${id}",
      params: { id: { type: "enum", values: ["quality", "coverage", "quantity", "ui"] as const } },
    }),
  },
  locale: {
    toggle: defineDynamicSelector({
      description: "Locale toggle button by language code",
      testIdPattern: "locale-toggle-${code}",
      params: { code: { type: "enum", values: LOCALE_CODES } },
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
