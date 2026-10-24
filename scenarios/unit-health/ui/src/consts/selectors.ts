import { librarySelectors } from "./selectors.library";
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
  validationWorkbench: {
    root: "validation-workbench",
    scenarioInput: "validation-workbench-scenario-input",
    runButton: "validation-workbench-run-button",
    loading: "validation-workbench-loading",
    error: "validation-workbench-error",
    idle: "validation-workbench-idle",
    status: "validation-workbench-status",
    maturity: "validation-workbench-maturity",
    counts: "validation-workbench-counts",
    surfaces: "validation-workbench-surfaces",
    rationale: "validation-workbench-rationale",
    degraded: "validation-workbench-degraded",
    findings: "validation-workbench-findings",
    empty: "validation-workbench-empty",
    maturitySummary: "validation-workbench-maturity-summary",
    nextLevel: "validation-workbench-next-level",
    testPlan: "validation-workbench-test-plan",
    testPlanEmpty: "validation-workbench-test-plan-empty",
    execution: "validation-workbench-execution",
    executionEmpty: "validation-workbench-execution-empty",
    coverage: "validation-workbench-coverage",
    coverageEmpty: "validation-workbench-coverage-empty",
    projections: "validation-workbench-projections",
    projectionsEmpty: "validation-workbench-projections-empty",
    diagnostics: "validation-workbench-diagnostics",
    diagnosticsEmpty: "validation-workbench-diagnostics-empty",
    globalImpact: "validation-workbench-global-impact",
    globalImpactEmpty: "validation-workbench-global-impact-empty",
    recommendedSkills: "validation-workbench-recommended-skills",
    recommendedSkillsEmpty: "validation-workbench-recommended-skills-empty",
    nextSteps: "validation-workbench-next-steps",
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
      params: { key: { type: "enum", values: ["dashboard", "settings"] as const } },
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
  validationWorkbench: {
    findingRow: defineDynamicSelector({
      description: "Validation finding row by stable finding id",
      testIdPattern: "validation-workbench-finding-${id}",
      params: { id: { type: "string" } },
    }),
    workspaceRow: defineDynamicSelector({
      description: "Test-plan workspace row by stable workspace id",
      testIdPattern: "validation-workbench-workspace-${id}",
      params: { id: { type: "string" } },
    }),
    commandRow: defineDynamicSelector({
      description: "Execution result row by command name",
      testIdPattern: "validation-workbench-command-${name}",
      params: { name: { type: "string" } },
    }),
    commandOutputToggle: defineDynamicSelector({
      description: "Toggle button for a command result's captured output",
      testIdPattern: "validation-workbench-command-toggle-${name}",
      params: { name: { type: "string" } },
    }),
    commandOutput: defineDynamicSelector({
      description: "Expanded captured output panel for a command result",
      testIdPattern: "validation-workbench-command-output-${name}",
      params: { name: { type: "string" } },
    }),
    coverageSurface: defineDynamicSelector({
      description: "Coverage surface roll-up panel by surface id",
      testIdPattern: "validation-workbench-coverage-surface-${id}",
      params: { id: { type: "string" } },
    }),
    coverageRow: defineDynamicSelector({
      description: "Coverage target row by stable target id",
      testIdPattern: "validation-workbench-coverage-row-${id}",
      params: { id: { type: "string" } },
    }),
    findingCategory: defineDynamicSelector({
      description: "Findings category group by category name",
      testIdPattern: "validation-workbench-finding-category-${category}",
      params: { category: { type: "string" } },
    }),
    projectionRow: defineDynamicSelector({
      description: "Unit policy projection row by stable projection id",
      testIdPattern: "validation-workbench-projection-${id}",
      params: { id: { type: "string" } },
    }),
    impactRow: defineDynamicSelector({
      description: "Global-impact grouping row by impact key",
      testIdPattern: "validation-workbench-impact-${key}",
      params: { key: { type: "string" } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
