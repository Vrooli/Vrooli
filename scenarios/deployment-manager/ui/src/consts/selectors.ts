import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  layout: {
    rootContainer: 'app-root',
    navDashboard: 'nav-dashboard',
    navProfiles: 'nav-profiles',
    navAnalyze: 'nav-analyze',
    navDeployments: 'nav-deployments',
  },
  dashboard: {
    heading: 'dashboard-heading',
    newProfileButton: 'new-profile-button',
    profileCard: 'profile-card',
  },
  profiles: {
    list: 'profiles-list',
    item: 'profile-item',
  },
  profileForm: {
    form: 'profile-form',
    nameInput: 'profile-name-input',
    scenarioInput: 'profile-scenario-input',
    tierSelector: 'tier-selector',
    createSubmit: 'create-profile-submit',
  },
  analyze: {
    form: 'analyze-form',
    scenarioInput: 'scenario-input',
    analyzeButton: 'analyze-button',
    fitnessScores: 'fitness-scores',
    dependencyTree: 'dependency-tree',
  },
  profileDetail: {
    detail: 'profile-detail',
    deployButton: 'deploy-button',
    config: 'profile-config',
  },
  deployments: {
    list: 'deployments-list',
    item: 'deployment-item',
    status: 'deployment-status',
    logs: 'deployment-logs',
  },
  health: {
    status: 'health-status',
  },
  evidence: {
    heading: 'evidence-heading',
    commitInput: 'evidence-commit-input',
    reviewIntro: 'evidence-review-intro',
    gateStatus: 'evidence-gate-status',
    producerRecording: 'evidence-producer-recording',
  },
  releases: {
    heading: 'releases-heading',
    recordsIntro: 'releases-records-intro',
  },
};

const dynamicSelectorDefinitions = {
  /*
  Example dynamic selectors:
  projects: {
    cardByName: defineDynamicSelector({
      description: 'Project card filtered by name',
      selectorPattern: '[data-testid="project-card"][data-project-name="${name}"]',
      params: { name: { type: 'string' } },
    }),
  },
  */
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
