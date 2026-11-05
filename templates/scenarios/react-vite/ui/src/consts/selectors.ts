import { librarySelectors } from "./selectors.library";
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
  // EXAMPLE-DOMAIN:notes START
  notes: {
    surface: "notes-surface",
    card: "notes-card",
    list: "notes-list",
    loading: "notes-loading",
    empty: "notes-empty",
    error: "notes-error",
    createButton: "notes-create-button",
    createdAt: "notes-created-at",
    attachmentCount: "notes-attachment-count",
    attachmentUpload: "notes-attachment-upload",
    attachmentFile: "notes-attachment-file",
    attachmentButton: "notes-attachment-button",
    attachmentStatus: "notes-attachment-status",
    measure: {
      card: "notes-measure-card",
      value: "notes-measure-value",
      loading: "notes-measure-loading",
      error: "notes-measure-error",
    },
  },
  // EXAMPLE-DOMAIN:notes END
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
    notes: "page-notes", // EXAMPLE-DOMAIN:notes
    settings: "page-settings",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  layout: {
    navLink: defineDynamicSelector({
      description: "Primary navigation link by canonical nav key; the phone tab derives `-tab` from the same id",
      testIdPattern: "layout-nav-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "notes", // EXAMPLE-DOMAIN:notes
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
            "notes", // EXAMPLE-DOMAIN:notes
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
