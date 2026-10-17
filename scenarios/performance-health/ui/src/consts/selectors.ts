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
    dashboard: "page-dashboard",
    settings: "page-settings",
    audit: "page-audit",
    trends: "page-trends",
    fleet: "page-fleet",
    trace: "page-trace",
    readiness: "page-readiness",
    budgets: "page-budgets",
    overviewSnapshot: "overview-snapshot",
  },
  perf: {
    scenarioSelect: "perf-scenario-select",
  },
  audit: {
    runButton: "audit-run-button",
    error: "audit-error",
    runError: "audit-run-error",
    tierPanel: "audit-tier-panel",
    tierBadge: "audit-tier-badge",
    findings: "audit-findings",
    findingsEmpty: "audit-findings-empty",
    result: "audit-result",
    outcomeBadge: "audit-outcome-badge",
    analyzeTraceLink: "audit-analyze-trace-link",
  },
  trends: {
    error: "trends-error",
    empty: "trends-empty",
    charts: "trends-charts",
    cardGoBuild: "trends-card-go-build",
    cardUiBuild: "trends-card-ui-build",
    cardBundle: "trends-card-bundle",
    cardLcp: "trends-card-lcp",
    cardComponent: "trends-card-component",
    cardStartup: "trends-card-startup",
    samples: "trends-samples",
  },
  fleet: {
    refreshButton: "fleet-refresh-button",
    loading: "fleet-loading",
    error: "fleet-error",
    empty: "fleet-empty",
    summary: "fleet-summary",
    summaryScenarios: "fleet-summary-scenarios",
    summaryNoBudget: "fleet-summary-no-budget",
    summaryRegressed: "fleet-summary-regressed",
    tiers: "fleet-tiers",
    slowest: "fleet-slowest",
    regressed: "fleet-regressed",
    noBudget: "fleet-no-budget",
    errors: "fleet-errors",
  },
  trace: {
    artifactInput: "trace-artifact-input",
    analyzeButton: "trace-analyze-button",
    error: "trace-error",
    empty: "trace-empty",
    vitals: "trace-vitals",
    components: "trace-components",
    findings: "trace-findings",
    findingsEmpty: "trace-findings-empty",
  },
  readiness: {
    error: "readiness-error",
    summary: "readiness-summary",
    tierBadge: "readiness-tier-badge",
    autofixableCount: "readiness-autofixable-count",
    gaps: "readiness-gaps",
    gapsEmpty: "readiness-gaps-empty",
    previewButton: "readiness-preview-button",
    applyButton: "readiness-apply-button",
    applyError: "readiness-apply-error",
    fixResult: "readiness-fix-result",
  },
  budgets: {
    error: "budgets-error",
    form: "budgets-form",
    notDeclared: "budgets-not-declared",
    ratchet: "budgets-ratchet",
    saveButton: "budgets-save-button",
    checkButton: "budgets-check-button",
    saved: "budgets-saved",
    checkError: "budgets-check-error",
    checkResult: "budgets-check-result",
    checkVerdict: "budgets-check-verdict",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  state: {
    loading: "state-loading",
    error: "state-error",
    errorRetry: "state-error-retry",
    empty: "state-empty",
    emptyAction: "state-empty-action",
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
    navLink: defineDynamicSelector({
      description: "App shell navigation link by canonical nav key",
      testIdPattern: "layout-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "audit",
            "trends",
            "fleet",
            "trace",
            "readiness",
            "budgets",
            "settings",
          ] as const,
        },
      },
    }),
  },
  pages: {
    workflowCard: defineDynamicSelector({
      description: "Overview workflow entry-point card by destination route",
      testIdPattern: "overview-workflow-card-${to}",
      params: { to: { type: "string" } },
    }),
  },
  audit: {
    findingRow: defineDynamicSelector({
      description: "Readiness finding row in the audit workbench by finding code",
      testIdPattern: "audit-finding-${code}",
      params: { code: { type: "string" } },
    }),
  },
  fleet: {
    scenarioRow: defineDynamicSelector({
      description: "Fleet offender row by scenario id",
      testIdPattern: "fleet-scenario-${scenario}",
      params: { scenario: { type: "string" } },
    }),
    tierRow: defineDynamicSelector({
      description: "Fleet tier-distribution row by tier",
      testIdPattern: "fleet-tier-${tier}",
      params: { tier: { type: "string" } },
    }),
  },
  trace: {
    componentRow: defineDynamicSelector({
      description: "Per-component trace timing row by component name",
      testIdPattern: "trace-component-${component}",
      params: { component: { type: "string" } },
    }),
  },
  readiness: {
    gapRow: defineDynamicSelector({
      description: "Readiness gap row by finding code",
      testIdPattern: "readiness-gap-${code}",
      params: { code: { type: "string" } },
    }),
  },
  budgets: {
    field: defineDynamicSelector({
      description: "Budget threshold input by proto field name",
      testIdPattern: "budgets-field-${field}",
      params: { field: { type: "string" } },
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
