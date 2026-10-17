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
  debtList: {
    row: defineDynamicSelector({
      description: "Debt list row by debt id",
      testIdPattern: "debt-list-row-${key}",
      params: { key: { type: "string" } },
    }),
  },
  templateDetail: {
    runLink: defineDynamicSelector({
      description: "Template detail run link by run id",
      testIdPattern: "template-detail-run-${id}",
      params: { id: { type: "string" } },
    }),
    debtLink: defineDynamicSelector({
      description: "Template detail debt link by debt id",
      testIdPattern: "template-detail-debt-${key}",
      params: { key: { type: "string" } },
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
            "templates",
            "runs",
            "debt",
            "settings",          ] as const,
        },
      },
    }),
  },
  templateList: {
    row: defineDynamicSelector({
      description: "Template list row by template id",
      testIdPattern: "template-list-row-${id}",
      params: { id: { type: "string" } },
    }),
  },
  runList: {
    row: defineDynamicSelector({
      description: "Run list row by run id",
      testIdPattern: "run-list-row-${id}",
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
