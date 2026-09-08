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
    templateList: "page-template-list",
    templateDetail: "page-template-detail",
    runList: "page-run-list",
    runDetail: "page-run-detail",
    debtList: "page-debt-list",
    debtDetail: "page-debt-detail",
  },
  templateList: {
    root: "template-list",
    table: "template-list-table",
    kindFilter: "template-list-kind-filter",
    loading: "template-list-loading",
    error: "template-list-error",
  },
  runList: {
    root: "run-list",
    table: "run-list-table",
    statusFilter: "run-list-status-filter",
    loading: "run-list-loading",
    error: "run-list-error",
  },
  templateDetail: {
    header: "template-detail-header",
    overview: "template-detail-overview",
    runs: "template-detail-runs",
    drift: "template-detail-drift",
    debt: "template-detail-debt",
    loading: "template-detail-loading",
    error: "template-detail-error",
  },
  runDetail: {
    header: "run-detail-header",
    overview: "run-detail-overview",
    phases: "run-detail-phases",
    findings: "run-detail-findings",
    loading: "run-detail-loading",
    error: "run-detail-error",
  },
  debtList: {
    root: "debt-list",
    table: "debt-list-table",
    templateFilter: "debt-list-template-filter",
    statusFilter: "debt-list-status-filter",
    loading: "debt-list-loading",
    error: "debt-list-error",
  },
  debtDetail: {
    header: "debt-detail-header",
    overview: "debt-detail-overview",
    message: "debt-detail-message",
    provenance: "debt-detail-provenance",
    loading: "debt-detail-loading",
    error: "debt-detail-error",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  layout: {
    sidebarLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-sidebar-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "templates",
            "runs",
            "debt",
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
            "templates",
            "runs",
            "debt",
            "settings",
          ] as const,
        },
      },
    }),
  },
  templateList: {
    row: defineDynamicSelector({
      description: "Template list row link to a template detail view, by template id",
      testIdPattern: "template-list-row-${id}",
      params: { id: { type: "string" } },
    }),
  },
  runList: {
    row: defineDynamicSelector({
      description: "Run list row link to a validation run detail view, by run id",
      testIdPattern: "run-list-row-${id}",
      params: { id: { type: "string" } },
    }),
  },
  debtList: {
    row: defineDynamicSelector({
      description: "Debt list row link to a debt entry detail view, by debt key",
      testIdPattern: "debt-list-row-${key}",
      params: { key: { type: "string" } },
    }),
  },
  templateDetail: {
    runLink: defineDynamicSelector({
      description: "Template detail link to one of its validation runs, by run id",
      testIdPattern: "template-detail-run-link-${id}",
      params: { id: { type: "string" } },
    }),
    debtLink: defineDynamicSelector({
      description: "Template detail link to one of its debt entries, by debt key",
      testIdPattern: "template-detail-debt-link-${key}",
      params: { key: { type: "string" } },
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
