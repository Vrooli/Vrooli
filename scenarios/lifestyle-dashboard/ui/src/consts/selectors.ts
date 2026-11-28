import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  // Dashboard page selectors
  dashboard: {
    statsGrid: "dashboard-stats-grid",
    timelineSection: "dashboard-timeline-section",
    domainsSection: "dashboard-domains-section",
    eventsSection: "dashboard-events-section",
  },
  // Timeline chart selectors [REQ:LD-UI-TRENDS]
  timeline: {
    chart: "timeline-chart",
    empty: "timeline-empty",
    bars: "timeline-bars",
    periodSelector: "timeline-period-selector",
  },
  // Domain list selectors [REQ:LD-UI-DOMAINS]
  domains: {
    list: "domains-list",
    card: "domain-card",
    statusBadge: "domain-status-badge",
  },
  // Events list selectors
  events: {
    list: "events-list",
    row: "event-row",
  },
  // Stats selectors
  stats: {
    totalEvents: "stat-total-events",
    activeDomains: "stat-active-domains",
    lastActivity: "stat-last-activity",
    trend: "stat-7day-trend",
  },
  // Lifestyle Score [REQ:LD-UI-SCORE]
  score: {
    card: "lifestyle-score",
    value: "lifestyle-score-value",
  },
  // Error display
  error: {
    alert: "error-alert",
    retryButton: "error-retry-button",
  },
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
// Export defineDynamicSelector for use when adding dynamic selectors
export { defineDynamicSelector };

/**
 * Flat data-testid constants for direct use in components.
 * This provides a simpler API than the nested selectors object.
 * [REQ:LD-UI-TRENDS] [REQ:LD-UI-DOMAINS]
 */
export const DATA_SELECTORS = {
  // Dashboard
  DASHBOARD_STATS_GRID: "dashboard-stats-grid",
  DASHBOARD_TIMELINE_SECTION: "dashboard-timeline-section",
  DASHBOARD_DOMAINS_SECTION: "dashboard-domains-section",
  DASHBOARD_EVENTS_SECTION: "dashboard-events-section",

  // Timeline chart [REQ:LD-UI-TRENDS]
  TIMELINE_CHART: "timeline-chart",
  TIMELINE_EMPTY: "timeline-empty",
  TIMELINE_BARS: "timeline-bars",
  TIMELINE_PERIOD_SELECTOR: "timeline-period-selector",

  // Domains [REQ:LD-UI-DOMAINS]
  DOMAINS_LIST: "domains-list",
  DOMAIN_CARD: "domain-card",
  DOMAIN_STATUS_BADGE: "domain-status-badge",

  // Events
  EVENTS_LIST: "events-list",
  EVENT_ROW: "event-row",

  // Stats
  STAT_TOTAL_EVENTS: "stat-total-events",
  STAT_ACTIVE_DOMAINS: "stat-active-domains",
  STAT_LAST_ACTIVITY: "stat-last-activity",
  STAT_7DAY_TREND: "stat-7day-trend",

  // Lifestyle Score [REQ:LD-UI-SCORE]
  LIFESTYLE_SCORE: "lifestyle-score",
  LIFESTYLE_SCORE_VALUE: "lifestyle-score-value",

  // Error handling
  ERROR_ALERT: "error-alert",
  ERROR_RETRY_BUTTON: "error-retry-button",

  // Settings/Storage [REQ:LD-UI-STORAGE]
  SETTINGS_PAGE: "settings-page",
  STORAGE_SIZE: "storage-size",
  STORAGE_EVENTS: "storage-events",
  STORAGE_CLEAR_ALL: "storage-clear-all",

  // Briefs [REQ:LD-BRIEF-MORNING] [REQ:LD-BRIEF-EVENING]
  BRIEF_CARD: "brief-card",
  BRIEF_SUMMARY: "brief-summary",
  BRIEF_SCORE: "brief-score",
  BRIEF_SECTIONS: "brief-sections",
  BRIEF_SECTION: "brief-section",
  BRIEFS_PAGE: "briefs-page",
  BRIEF_MORNING_TAB: "brief-morning-tab",
  BRIEF_EVENING_TAB: "brief-evening-tab",
} as const;
