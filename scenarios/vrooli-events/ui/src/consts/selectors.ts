import { librarySelectors } from "./selectors.library";
// DOC: docs/internal/EXPERIENCE-AUDIT.md
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  nav: {
    stream: 'nav-stream',
    analytics: 'nav-analytics',
    events: 'nav-events',
    settings: 'nav-settings',
    healthIndicator: 'nav-health-indicator',
  },
  stream: {
    pauseButton: 'stream-pause-button',
    clearButton: 'stream-clear-button',
    typeFilter: 'stream-type-filter',
    sourceFilter: 'stream-source-filter',
    resetFilters: 'stream-reset-filters',
    eventCount: 'stream-event-count',
    connectionStatus: 'stream-connection-status',
  },
  analytics: {
    totalEvents: 'analytics-total-events',
    storeSize: 'analytics-store-size',
    subscribers: 'analytics-subscribers',
    systemStatus: 'analytics-system-status',
  },
  eventLog: {
    table: 'event-log-table',
    typeFilter: 'event-log-type-filter',
    sourceFilter: 'event-log-source-filter',
    correlationFilter: 'event-log-correlation-filter',
    limitSelect: 'event-log-limit-select',
    resetFilters: 'event-log-reset-filters',
    refreshButton: 'event-log-refresh-button',
  },
  eventDetail: {
    panel: 'event-detail-panel',
    closeButton: 'event-detail-close',
    eventId: 'event-detail-id',
    eventType: 'event-detail-type',
    payload: 'event-detail-payload',
    metadata: 'event-detail-metadata',
  },
  settings: {
    retentionTime: 'settings-retention-time',
    retentionSize: 'settings-retention-size',
    pruneInterval: 'settings-prune-interval',
  },
  scenarioMetrics: {
    page: 'scenario-metrics-page',
    table: 'scenario-metrics-table',
    sortOutbound: 'sort-outbound',
    sortInbound: 'sort-inbound',
    sortErrorRate: 'sort-errorRate',
  },
  correlationTrace: {
    page: 'correlation-trace-page',
    correlationInput: 'trace-correlation-input',
    searchButton: 'trace-search-button',
    timeline: 'trace-timeline',
  },
  policies: {
    page: 'policies-page',
    table: 'policies-table',
    newButton: 'policies-new-button',
    newForm: 'policies-new-form',
  },
  circuitBreakers: {
    page: 'circuit-breakers-page',
    table: 'circuit-breakers-table',
    overrideForm: 'cb-override-form',
  },
  subscriptions: {
    page: 'subscriptions-page',
    table: 'subscriptions-table',
    newButton: 'subscriptions-new-button',
    newForm: 'subscriptions-new-form',
  },
  compliance: {
    page: 'compliance-page',
    table: 'compliance-table',
  },
} as const satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  eventTable: {
    rowByIndex: defineDynamicSelector({
      description: 'Event table row by index',
      selectorPattern: '[data-testid="event-row-${index}"]',
      params: { index: { type: 'number' } },
    }),
    rowByEventId: defineDynamicSelector({
      description: 'Event table row by event ID',
      selectorPattern: '[data-testid="event-row"][data-event-id="${eventId}"]',
      params: { eventId: { type: 'string' } },
    }),
  },
} as const satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
