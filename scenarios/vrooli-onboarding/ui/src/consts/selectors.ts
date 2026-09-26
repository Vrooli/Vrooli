import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  app: {
    skipToContent: "skip-to-content",
    nav: "app-nav",
  },
  nav: {
    wizard: "nav-wizard",
    wizardBadge: "nav-wizard-badge",
    dashboard: "nav-dashboard",
    glossary: "nav-glossary",
  },
  wizard: {
    shell: "wizard-shell",
    prev: "wizard-prev",
    next: "wizard-next",
    welcome: "step-welcome",
    scenarios: "step-select-scenarios",
    integrations: "step-integrations-deferred",
    operatingMode: "step-operating-mode",
    resources: "step-derived-resources",
    host: "step-host-requirements",
    readiness: "step-readiness",
    stepAnnouncement: "step-announcement",
  },
  operatingMode: {
    toggle: "keep-running-toggle",
  },
  apply: {
    plan: "apply-plan",
    summary: "plan-summary",
    privilegeWarning: "privilege-warning",
    skippedNote: "skipped-note",
    retry: "retry",
  },
  readiness: {
    summary: "readiness-summary",
    item: "readiness-item",
    remediation: "remediation",
    recheck: "recheck",
    continueDegraded: "readiness-continue-degraded",
  },
  run: {
    ladder: "run-ladder",
    id: "run-id",
  },
  host: {
    tools: "host-tools",
    safeguards: "host-safeguards",
    requirementEntry: "requirement-entry",
    riskIndicator: "risk-indicator",
    privilegeIndicator: "privilege-indicator",
    changeSummary: "change-summary",
  },
  credentials: {
    card: "credential-card",
    purpose: "credential-purpose",
    status: "credential-status",
    input: "credential-input",
    save: "credential-save",
    obtainLink: "credential-obtain-link",
  },
  capabilities: {
    actions: "capability-actions",
  },
  scenario: {
    list: "scenario-list",
    search: "scenario-search",
    cascadeNote: "cascade-note",
    resourceRollup: "resource-rollup",
    catalogError: "catalog-error",
    lockedBadge: "locked-badge",
  },
  dashboard: {
    root: "health-dashboard",
    summary: "health-summary",
    grid: "health-grid",
    loading: "health-loading",
    error: "health-error",
    lastChecked: "health-last-checked",
    goToWizard: "health-go-to-wizard",
  },
  glossary: {
    root: "glossary-panel",
    search: "glossary-search",
    clearSearch: "glossary-clear-search",
    debounceIndicator: "glossary-debounce-indicator",
    count: "glossary-count",
    loading: "glossary-loading",
    empty: "glossary-empty",
    list: "glossary-list",
  },
} as const satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  wizard: {
    scenarioCard: defineDynamicSelector({
      description: "Scenario card by scenario name",
      testIdPattern: "scenario-card-${name}",
      params: { name: { type: "string" } },
    }),
  },
  dashboard: {
    healthCard: defineDynamicSelector({
      description: "Health card by resource name",
      testIdPattern: "health-card-${name}",
      params: { name: { type: "string" } },
    }),
    statusIndicator: defineDynamicSelector({
      description: "Status indicator by resource name",
      testIdPattern: "status-indicator-${name}",
      params: { name: { type: "string" } },
    }),
  },
  glossary: {
    entry: defineDynamicSelector({
      description: "Glossary entry by term",
      testIdPattern: "glossary-entry-${term}",
      params: { term: { type: "string" } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
