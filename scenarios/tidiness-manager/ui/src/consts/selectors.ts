import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  // Dashboard
  scenarioTable: 'scenario-table',
  scenarioTableHeader: {
    name: 'scenario-table-header-name',
    health: 'scenario-table-header-health',
    lightIssues: 'scenario-table-header-light-issues',
    aiIssues: 'scenario-table-header-ai-issues',
    longFiles: 'scenario-table-header-long-files',
    visitPercent: 'scenario-table-header-visit-percent',
    campaignStatus: 'scenario-table-header-campaign-status',
  },
  scenarioTableRow: 'scenario-table-row',

  // Scenario Detail
  fileTable: 'file-table',
  fileTableHeader: {
    path: 'file-table-header-path',
    lines: 'file-table-header-lines',
    issues: 'file-table-header-issues',
    visitCount: 'file-table-header-visit-count',
  },
  fileTableRow: 'file-table-row',

  // Actions
  runLightScanButton: 'run-light-scan-button',
  viewAllIssuesButton: 'view-all-issues-button',
  viewFileDetailsButton: 'view-file-details-button',

  // Filters
  statusFilter: 'status-filter',
  categoryFilter: 'category-filter',
  severityFilter: 'severity-filter',

  // Campaign
  newCampaignButton: 'new-campaign-button',
  pauseCampaignButton: 'pause-campaign-button',
  resumeCampaignButton: 'resume-campaign-button',
  stopCampaignButton: 'stop-campaign-button',

  // Settings
  longFileThresholdInput: 'long-file-threshold-input',
  maxScansPerFileInput: 'max-scans-per-file-input',
  saveSettingsButton: 'save-settings-button',
} satisfies LiteralSelectorTree;

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
