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
    fleet: "page-fleet",
    validation: "page-validation",
    settings: "page-settings",
  },
  fleet: {
    view: "fleet-view",
    refreshButton: "fleet-refresh-button",
    loading: "fleet-loading",
    error: "fleet-error",
    empty: "fleet-empty",
    summary: "fleet-summary",
    summaryScenarios: "fleet-summary-scenarios",
    summaryPassing: "fleet-summary-passing",
    summaryAutofixable: "fleet-summary-autofixable",
    profiles: "fleet-profiles",
    rules: "fleet-rules",
    rulesEmpty: "fleet-rules-empty",
    scenarios: "fleet-scenarios",
    scenariosEmpty: "fleet-scenarios-empty",
    errors: "fleet-errors",
    targets: "fleet-targets",
  },
  validation: {
    view: "validation-view",
    kind: "validation-kind",
    id: "validation-id",
    root: "validation-root",
    path: "validation-path",
    submit: "validation-submit",
    loading: "validation-loading",
    error: "validation-error",
    result: "validation-result",
    findings: "validation-findings",
    emptyFindings: "validation-findings-empty",
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
            "fleet",
            "validate",
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
            "fleet",
            "validate",
            "settings",
          ] as const,
        },
      },
    }),
  },
  fleet: {
    targetRow: defineDynamicSelector({
      description: "Typed fleet target row by kind and id",
      testIdPattern: "fleet-target-row-${kind}-${id}",
      params: { kind: { type: "string" }, id: { type: "string" } },
    }),
    scenarioRow: defineDynamicSelector({
      description: "Scenario offenders table row by scenario slug",
      testIdPattern: "fleet-scenario-row-${scenario}",
      params: { scenario: { type: "string" } },
    }),
    ruleRow: defineDynamicSelector({
      description: "Rule conformance table row by finding code",
      testIdPattern: "fleet-rule-row-${code}",
      params: { code: { type: "string" } },
    }),
    profileRow: defineDynamicSelector({
      description: "Profile distribution row by profile id",
      testIdPattern: "fleet-profile-row-${profileId}",
      params: { profileId: { type: "string" } },
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
