import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  // Dashboard
  dashboard: {
    startNewButton: "dashboard-start-new-button",
    resumeButton: "dashboard-resume-button",
    discardButton: "dashboard-discard-button",
    apiStatus: "dashboard-api-status",
  },
  // Wizard
  wizard: {
    container: "wizard-container",
    stepper: "wizard-stepper",
    backButton: "wizard-back-button",
    nextButton: "wizard-next-button",
    dashboardButton: "wizard-dashboard-button",
    resetButton: "wizard-reset-button",
  },
  // Manifest Step (preserved from original)
  manifest: {
    input: "manifest-input",
    formModeButton: "manifest-form-mode-button",
    jsonModeButton: "manifest-json-mode-button",
    hostInput: "manifest-host-input",
    domainInput: "manifest-domain-input",
    scenarioIdInput: "manifest-scenario-id-input",
    uiPortInput: "manifest-ui-port-input",
    apiPortInput: "manifest-api-port-input",
    wsPortInput: "manifest-ws-port-input",
    includePackagesCheckbox: "manifest-include-packages-checkbox",
    includeAutohealCheckbox: "manifest-include-autoheal-checkbox",
    caddyEnabledCheckbox: "manifest-caddy-enabled-checkbox",
    validateButton: "manifest-validate-button",
    validateResult: "manifest-validate-result",
    bundleBuildButton: "manifest-bundle-build-button",
    bundleBuildResult: "manifest-bundle-build-result",
  },
  // Preflight Step
  preflight: {
    runButton: "preflight-run-button",
    result: "preflight-result",
    checkList: "preflight-check-list",
  },
  // Deploy Step
  deploy: {
    deployButton: "deploy-deploy-button",
    result: "deploy-result",
    startNewButton: "deploy-start-new-button",
    liveLink: "deploy-live-link",
  },
  // Docs
  docs: {
    sidebar: "docs-sidebar",
    viewer: "docs-viewer",
    copyPath: "docs-copy-path",
    searchInput: "docs-search-input",
  },
} as const satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  wizard: {
    stepByIndex: defineDynamicSelector({
      description: "Wizard step by index (0-based)",
      testIdPattern: "wizard-step-${index}",
      params: { index: { type: "number" } },
    }),
    stepByName: defineDynamicSelector({
      description: "Wizard step by name",
      testIdPattern: "wizard-step-${name}",
      params: { name: { type: "enum", values: ["manifest", "build", "preflight", "deploy"] } },
    }),
  },
  validation: {
    issueByIndex: defineDynamicSelector({
      description: "Validation issue by index (0-based)",
      testIdPattern: "validation-issue-${index}",
      params: { index: { type: "number" } },
    }),
  },
} as const satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
