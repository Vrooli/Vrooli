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
  projects: {
    surface: "projects-surface",
    loading: "projects-loading",
    error: "projects-error",
    empty: "projects-empty",
    list: "projects-list",
    nameInput: "projects-name-input",
    createButton: "projects-create-button",
    renameInput: "project-rename-input",
    renameButton: "project-rename-button",
    conflict: "project-rename-conflict",
    reloadButton: "project-reload-button",
  },
  media: {
    panel: "project-source-media",
    fileInput: "project-source-media-file",
    fileHelp: "project-source-media-file-help",
    uploadButton: "project-source-media-upload",
    pending: "project-source-media-pending",
    uploadError: "project-source-media-upload-error",
    listHeading: "project-source-media-heading",
    loading: "project-source-media-loading",
    empty: "project-source-media-empty",
    list: "project-source-media-list",
    loadMore: "project-source-media-load-more",
    preview: "project-source-media-preview",
    image: "project-source-media-image",
    videoPreview: "project-source-media-video",
  },
  shot: {
    videoTiming: "shot-video-timing",
    videoSampleAccept: "shot-video-sample-accept",
    videoFinalStage: "shot-video-final-stage",
    videoSourcePreview: "shot-video-source-preview",
  },
  physical: {
    create: "physical-shot-create",
    editor: "physical-shot-editor",
    source: "physical-shot-source",
    backend: "physical-shot-backend",
    compile: "physical-shot-compile",
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
    accountPanel: "page-settings-account",
    accountStatus: "page-settings-account-status",
    accountConnect: "page-settings-account-connect",
    accountDisconnect: "page-settings-account-disconnect",
  },
  pages: {
    dashboard: "page-dashboard",
    dashboardHeader: "page-dashboard-header",
    dashboardPlaceholder: "page-dashboard-placeholder",
    project: "page-project",
    settings: "page-settings",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  projects: {
    openButton: defineDynamicSelector({
      description: "Open action for one project by stable project id",
      testIdPattern: "projects-open-${id}",
      params: { id: { type: "string" } },
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
  media: {
    selectButton: defineDynamicSelector({
      description: "Select one saved source image by stable media ID",
      testIdPattern: "project-source-media-select-${id}",
      params: { id: { type: "string" } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
