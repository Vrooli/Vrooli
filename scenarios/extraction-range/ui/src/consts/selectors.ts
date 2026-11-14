import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

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
  layout: {
    // The library AppShell derives every part id from the root id it is given.
    shell: "layout-shell",
    navigation: "layout-shell-navigation",
    tabs: "layout-shell-tabs",
    main: "layout-shell-main",
    brand: "layout-shell-brand",
    skip: "layout-shell-skip",
  },
  settingsPage: {
    themeSelect: "page-settings-theme",
    localeSelect: "page-settings-locale",
  },
  pages: {
    dashboard: "page-dashboard",
    dashboardHeader: "page-dashboard-header",
    dashboardPlaceholder: "page-dashboard-placeholder",
    settings: "page-settings",
  },
  range: {
    catalog: "range-catalog",
    familyFilter: "range-family-filter",
    levelList: "range-level-list",
    challenge: "range-selected-challenge",
    toolInput: "range-tool-input",
    recordsInput: "range-records-input",
    submitButton: "range-submit-button",
    scoreboard: "range-scoreboard",
    scoreList: "range-score-list",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  range: {
    level: defineDynamicSelector({
      description: "Challenge row by stable level id",
      testIdPattern: "range-level-${key}",
      params: { key: { type: "string" } },
    }),
  },
  layout: {
    navLink: defineDynamicSelector({
      description: "Primary navigation link by canonical nav key; the phone tab derives `-tab` from the same id",
      testIdPattern: "layout-nav-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "settings",
          ] as const,
        },
      },
    }),
    navTab: defineDynamicSelector({
      description: "Phone tab-bar item by canonical nav key",
      testIdPattern: "layout-nav-${key}-tab",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "settings",
          ] as const,
        },
      },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
