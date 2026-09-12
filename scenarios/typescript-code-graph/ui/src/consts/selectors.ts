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
  pages: {
    workbench: "page-workbench",
    settings: "page-settings",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  shared: {
    emptyState: {
      root: "empty-state-root",
      title: "empty-state-title",
      description: "empty-state-description",
      action: "empty-state-action",
    },
    errorState: {
      root: "error-state-root",
      title: "error-state-title",
      message: "error-state-message",
      retryButton: "error-state-retry",
    },
    loadingState: {
      root: "loading-state-root",
    },
  },
  ui: {
    tabs: {
      root: "tabs-root",
      list: "tabs-list",
    },
  },
  workbench: {
    extractBar: {
      root: "workbench-extract-bar",
      target: "workbench-extract-target",
      projectDir: "workbench-extract-project-dir",
      submit: "workbench-extract-submit",
    },
    stats: {
      root: "workbench-stats",
      files: "workbench-stat-files",
      modules: "workbench-stat-modules",
      symbols: "workbench-stat-symbols",
      imports: "workbench-stat-imports",
      warnings: "workbench-stat-warnings",
      hash: "workbench-stat-hash",
    },
    status: {
      loading: "workbench-status-loading",
      error: "workbench-status-error",
      empty: "workbench-status-empty",
      workspaceUnsupported: "workbench-status-workspace-unsupported",
    },
  },
  features: {
    explorer: {
      root: "explorer-root",
      canvas: {
        root: "explorer-canvas-root",
        summary: "explorer-canvas-summary",
      },
      accessibleList: {
        root: "explorer-accessible-list-root",
        empty: "explorer-accessible-list-empty",
      },
      legend: {
        root: "explorer-legend-root",
      },
      filterBar: {
        root: "explorer-filter-bar-root",
        allChip: "explorer-filter-bar-all",
        empty: "explorer-filter-bar-empty",
      },
      cycleBanner: "explorer-cycle-banner",
      drilldown: {
        root: "explorer-drilldown-root",
        title: "explorer-drilldown-title",
        empty: "explorer-drilldown-empty",
      },
    },
    sidecar: {
      root: "sidecar-status-root",
      loading: "sidecar-status-loading",
      error: "sidecar-status-error",
      message: "sidecar-status-message",
    },
    warnings: {
      root: "warnings-root",
      empty: "warnings-empty",
      list: "warnings-list",
      summary: "warnings-summary",
    },
    rewrite: {
      root: "rewrite-root",
      opsEditor: {
        root: "rewrite-ops-editor",
        empty: "rewrite-ops-empty",
        addFileMove: "rewrite-add-file-move",
        addImportRewrite: "rewrite-add-import-rewrite",
      },
      plan: {
        button: "rewrite-plan-button",
        result: "rewrite-plan-result",
        empty: "rewrite-plan-empty",
      },
      apply: {
        button: "rewrite-apply-button",
        result: "rewrite-apply-result",
        confirmDialog: {
          root: "rewrite-apply-confirm",
          confirm: "rewrite-apply-confirm-yes",
          cancel: "rewrite-apply-confirm-cancel",
        },
      },
    },
    fixtures: {
      root: "fixtures-root",
      list: "fixtures-list",
      empty: "fixtures-empty",
      loading: "fixtures-loading",
      error: "fixtures-error",
      result: "fixtures-result",
      diff: "fixtures-diff",
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
  shared: {
    severityBadge: {
      root: defineDynamicSelector({
        description: "Severity badge by level",
        testIdPattern: "severity-badge-${level}",
        params: { level: { type: "string" } },
      }),
    },
  },
  ui: {
    tabs: {
      trigger: defineDynamicSelector({
        description: "Tab trigger by tab value",
        testIdPattern: "tabs-trigger-${value}",
        params: { value: { type: "string" } },
      }),
      panel: defineDynamicSelector({
        description: "Tab panel by tab value",
        testIdPattern: "tabs-panel-${value}",
        params: { value: { type: "string" } },
      }),
    },
  },
  features: {
    explorer: {
      canvas: {
        node: defineDynamicSelector({
          description: "Explorer canvas node by node id",
          testIdPattern: "explorer-canvas-node-${id}",
          params: { id: { type: "string" } },
        }),
      },
      accessibleList: {
        item: defineDynamicSelector({
          description: "Accessible graph item by node id",
          testIdPattern: "explorer-accessible-item-${id}",
          params: { id: { type: "string" } },
        }),
      },
      filterBar: {
        chip: defineDynamicSelector({
          description: "Explorer filter chip by key",
          testIdPattern: "explorer-filter-chip-${key}",
          params: { key: { type: "string" } },
        }),
      },
      drilldown: {
        symbol: defineDynamicSelector({
          description: "Explorer symbol by symbol id",
          testIdPattern: "explorer-symbol-${id}",
          params: { id: { type: "string" } },
        }),
        symbolComments: defineDynamicSelector({
          description: "Explorer symbol comments by symbol id",
          testIdPattern: "explorer-symbol-comments-${id}",
          params: { id: { type: "string" } },
        }),
      },
      legend: {
        severity: defineDynamicSelector({
          description: "Explorer legend severity by level",
          testIdPattern: "explorer-legend-severity-${level}",
          params: { level: { type: "string" } },
        }),
      },
    },
    sidecar: {
      indicator: defineDynamicSelector({
        description: "Sidecar status indicator by status",
        testIdPattern: "sidecar-status-indicator-${status}",
        params: { status: { type: "string" } },
      }),
    },
    warnings: {
      item: defineDynamicSelector({
        description: "Warning item by index",
        testIdPattern: "warnings-item-${index}",
        params: { index: { type: "number" } },
      }),
    },
    rewrite: {
      opRow: defineDynamicSelector({
        description: "Rewrite operation row by index",
        testIdPattern: "rewrite-op-row-${index}",
        params: { index: { type: "number" } },
      }),
      opResult: defineDynamicSelector({
        description: "Rewrite operation result by index",
        testIdPattern: "rewrite-op-result-${index}",
        params: { index: { type: "number" } },
      }),
    },
    fixtures: {
      item: defineDynamicSelector({
        description: "Fixture item by name",
        testIdPattern: "fixtures-item-${name}",
        params: { name: { type: "string" } },
      }),
    },
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
            "workbench",
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
