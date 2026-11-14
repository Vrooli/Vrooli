import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  // Main dashboard container
  dashboard: 'autoheal-dashboard',
  shell: {
    header: 'autoheal-shell-header',
    nav: 'autoheal-shell-nav',
    content: 'autoheal-shell-content',
  },
  // Run tick button in header
  runTickButton: 'autoheal-run-tick-button',
  runTickButtonDesktop: 'autoheal-run-tick-button-desktop',
  settingsButton: 'autoheal-settings-button',
  // Health check card
  checkCard: 'autoheal-check-card',
  // Summary section
  summary: {
    grid: 'autoheal-summary-grid',
    total: 'autoheal-summary-total',
    ok: 'autoheal-summary-ok',
    warning: 'autoheal-summary-warning',
    critical: 'autoheal-summary-critical',
  },
  // Platform info section
  platform: 'autoheal-platform-info',
  // Events timeline section
  eventsTimeline: 'autoheal-events-timeline',
  // Uptime stats section
  uptimeStats: 'autoheal-uptime-stats',
  // Check history drawer
  checkHistory: 'autoheal-check-history',
  // Tab navigation
  tabs: {
    dashboard: 'autoheal-tab-dashboard',
    trends: 'autoheal-tab-trends',
    timeline: 'autoheal-tab-timeline',
    incidents: 'autoheal-tab-incidents',
    docs: 'autoheal-tab-docs',
  },
  // Trends page
  trends: {
    page: 'autoheal-trends-page',
    chart: 'autoheal-trends-chart',
    checkGrid: 'autoheal-trends-check-grid',
  },
  // System Protection section [REQ:WATCH-DETECT-001]
  systemProtection: 'autoheal-system-protection',
  systemProtectionCompact: 'autoheal-system-protection-compact',
} as const satisfies LiteralSelectorTree;

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
} as const satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
