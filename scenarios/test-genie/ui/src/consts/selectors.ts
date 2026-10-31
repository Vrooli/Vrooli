import { librarySelectors } from "./selectors.library";
/**
 * Test Genie selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. We deliberately model selectors as two
 * declarative maps (one literal, one dynamic) and rely on a small helper to
 * produce the typed `selectors` export plus the manifest consumed by workflow
 * linting. Do not hand-roll selector helpers or change this structure—update the
 * maps below so UI code, automation flows, and the manifest builder all stay in
 * sync across every scenario.
 *
 * ## Auto-Generated Manifest
 *
 * The `selectors.manifest.json` file is automatically generated from this file
 * during the testing process. If you need to add or modify selectors:
 *
 * 1. Update the `literalSelectors` object below for static selectors
 * 2. Update the `dynamicSelectorDefinitions` object for parameterized selectors
 * 3. The manifest will be regenerated automatically when tests run
 *
 * DO NOT manually edit `selectors.manifest.json` - your changes will be overwritten!
 */

import { createSelectorRegistry, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  // Top-level tab navigation
  tabs: {
    nav: "test-genie-tab-nav",
    dashboard: "test-genie-tab-dashboard",
    runs: "test-genie-tab-runs",
    docs: "test-genie-tab-docs",
    health: "test-genie-tab-health"
  },
  // Dashboard page
  dashboard: {
    continueSection: "test-genie-continue-section",
    header: "test-genie-header",
    lastExecution: "test-genie-last-execution"
  },
  // Self-Health page
  health: {
    page: "test-genie-health-page",
    catalog: "test-genie-health-catalog",
    conformance: "test-genie-health-conformance",
    ledger: "test-genie-health-ledger",
    providers: "test-genie-health-providers",
    trend: "test-genie-health-trend",
    empty: "test-genie-health-empty"
  },
  // Runs page
  runs: {
    subtabScenarios: "test-genie-subtab-scenarios",
    subtabHistory: "test-genie-subtab-history",
    scenarioTable: "test-genie-scenario-table",
    testGenieScenario: "test-genie-scenario-row-test-genie",
    historyTable: "test-genie-history-table",
    scenarioDetail: "test-genie-scenario-detail",
    scenarioDetailBack: "test-genie-scenario-detail-back",
    scenarioTabOverview: "test-genie-scenario-tab-overview",
    scenarioTabRequirements: "test-genie-scenario-tab-requirements",
    scenarioTabHistory: "test-genie-scenario-tab-history",
    phaseCard: "test-genie-phase-card",
    remediationPanel: "test-genie-remediation-panel"
  },
  // Requirements
  requirements: {
    panel: "test-genie-requirements-panel",
    syncBanner: "test-genie-sync-banner",
    syncButton: "test-genie-sync-button",
    coverageStats: "test-genie-coverage-stats",
    tree: "test-genie-requirements-tree",
    filterAll: "test-genie-filter-all",
    filterPassed: "test-genie-filter-passed",
    filterFailed: "test-genie-filter-failed",
    filterNotRun: "test-genie-filter-not-run",
    searchInput: "test-genie-requirements-search",
    // Help section
    helpSection: "test-genie-requirements-help",
    helpToggle: "test-genie-requirements-help-toggle",
  },
  // Docs page
  docs: {
    sidebar: "test-genie-docs-sidebar",
    viewer: "test-genie-docs-viewer",
    copyPath: "test-genie-docs-copy-path",
    searchInput: "test-genie-docs-search"
  },
  // Forms (used in Runs detail)
  forms: {
    executionForm: "test-genie-execution-form",
    submitExecution: "test-genie-submit-execution"
  },
  // Actions
  actions: {
    runTests: "test-genie-action-run-tests",
    viewScenario: "test-genie-action-view-scenario"
  }
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  scenarios: {
    rowByName: {
      kind: "dynamic-selector",
      description: "Scenario directory row by scenario name",
      testIdPattern: "test-genie-scenario-row-${name}",
      params: { name: { type: "string" } }
    }
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
