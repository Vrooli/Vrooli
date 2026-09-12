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
  brands: {
    card: "brands-card",
    list: "brands-list",
    loading: "brands-loading",
    empty: "brands-empty",
    error: "brands-error",
    createButton: "brands-create-button",
    updatedAt: "brands-updated-at",
    version: "brands-version",
  },
  assignments: {
    card: "assignments-card",
    list: "assignments-list",
    loading: "assignments-loading",
    empty: "assignments-empty",
    error: "assignments-error",
    scenario: "assignments-scenario",
    brand: "assignments-brand",
    version: "assignments-version",
  },
  assets: {
    card: "assets-card",
    list: "assets-list",
    loading: "assets-loading",
    empty: "assets-empty",
    error: "assets-error",
    filename: "assets-filename",
    brand: "assets-brand",
    mimeType: "assets-mime-type",
    size: "assets-size",
  },
  generation: {
    card: "generation-card",
    loading: "generation-loading",
    error: "generation-error",
    summary: "generation-summary",
    list: "generation-list",
    providerName: "generation-provider-name",
    providerStatus: "generation-provider-status",
    imageCard: "generation-image-card",
    imageSummary: "generation-image-summary",
    imageList: "generation-image-list",
    imageOpName: "generation-image-op-name",
    imageOpStatus: "generation-image-op-status",
  },
  apply: {
    card: "apply-card",
    brandInput: "apply-brand-input",
    scenarioInput: "apply-scenario-input",
    previewButton: "apply-preview-button",
    results: "apply-results",
    summary: "apply-summary",
    appliedList: "apply-applied-list",
    skippedList: "apply-skipped-list",
    empty: "apply-empty",
    error: "apply-error",
  },
  discovery: {
    card: "discovery-card",
    scenarioInput: "discovery-scenario-input",
    scanButton: "discovery-scan-button",
    results: "discovery-results",
    summary: "discovery-summary",
    sourcesList: "discovery-sources-list",
    draft: "discovery-draft",
    suggestionsList: "discovery-suggestions-list",
    empty: "discovery-empty",
    error: "discovery-error",
  },
  design: {
    card: "design-card",
    brandInput: "design-brand-input",
    generateButton: "design-generate-button",
    result: "design-result",
    markdown: "design-markdown",
    error: "design-error",
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
    brands: "page-brands",
    assignments: "page-assignments",
    assets: "page-assets",
    generation: "page-generation",
    apply: "page-apply",
    discovery: "page-discovery",
    design: "page-design",
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
  layout: {
    sidebarLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-sidebar-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "brands",
            "assignments",
            "assets",
            "generation",
            "apply",
            "discovery",
            "design",
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
            "brands",
            "assignments",
            "assets",
            "generation",
            "apply",
            "discovery",
            "design",
            "settings",
          ] as const,
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
