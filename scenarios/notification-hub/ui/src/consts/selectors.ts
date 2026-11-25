import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */
import { LOCALE_CODES } from "../i18n/locales";

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { createSelectorRegistry, defineDynamicSelector } from "@vrooli/ui-selectors";

/**
 * AlertDialog builds its test ids from a prefix: `<prefix>-dialog`, `-confirm`
 * and `-cancel`. Each dialog the cases drive has one prefix, used both by the
 * component and by the selectors below.
 */
export const dialogPrefixes = {
  quietRemoval: "quiet-hours-removal",
  deviceRemoval: "reach-device-removal",
  channelWiden: "reach-channel-widen",
} as const;

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
  },
  layout: {
    shell: "layout-shell",
    main: "layout-shell-main",
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
  },
  pages: {
    dashboard: "page-dashboard",
    settings: "page-settings",
    asks: "page-asks",
    ask: "page-ask",
    focus: "page-focus",
    history: "page-history",
    delivery: "page-delivery",
    deliveryDetail: "page-delivery-detail",
    notFound: "page-not-found",
  },
  delivery: {
    analyticsRetry: "delivery-analytics-retry",
  },
  history: {
    filterStrip: "history-filter-strip",
  },
  asks: {
    list: "asks-list",
    item: "asks-item",
    empty: "asks-empty",
    question: "ask-question",
    option: "ask-option",
    note: "ask-note",
    status: "ask-status",
    defaultNotice: "ask-default-notice",
    retryFileLinks: "ask-retry-file-links",
    loadRetry: "asks-load-retry",
    sort: "asks-sort",
    sortHow: "asks-sort-how",
    filtersOpen: "asks-filters-open",
    filtersSheet: "asks-filters-sheet",
    filtersShow: "asks-filters-show",
    filtersClear: "asks-filters-clear",
    activeFilters: "asks-active-filters",
    senderChip: "asks-filter-sender",
    group: "asks-group",
    desk: {
      page: "asks-desk",
      list: "asks-desk-list",
      row: "asks-desk-row",
      detail: "asks-desk-detail",
      title: "asks-desk-title",
    },
  },
  packet: {
    explainer: "packet-explainer",
    attachments: "packet-attachments",
    attachment: "packet-attachment",
    question: "packet-question",
    form: "packet-form",
    send: "packet-send",
    answerRecorded: "packet-answer-recorded",
    startAnswer: "packet-start-answer",
    acceptRecommended: "packet-accept-recommended",
    contextUnavailable: "packet-context-unavailable",
    loadRetry: "packet-load-retry",
    textLink: "packet-text-link",
    viewerOpenTab: "packet-viewer-open-tab",
    needContext: "packet-need-context",
    contextQuestion: "packet-context-question",
    sendContext: "packet-send-context",
    contextRequests: "packet-context-requests",
    nextDecision: "packet-next-decision",
    viewer: "packet-viewer",
    viewerClose: "file-preview-dock-close",
    viewerPrevious: "packet-viewer-previous",
    viewerNext: "packet-viewer-next",
    viewerComment: "packet-viewer-comment",
    viewerNote: "packet-viewer-note",
    viewerSaveNote: "packet-viewer-save-note",
  },
  session: {
    form: "session-form",
    email: "session-email",
    password: "session-password",
    submit: "session-submit",
    error: "session-error",
    signedIn: "session-signed-in",
    signOut: "session-sign-out",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  deviceApps: {
    root: "device-apps-root",
    removeButton: "device-apps-remove",
    message: "device-apps-message",
  },
  settings: {
    back: "settings-back",
  },
  quietHours: {
    add: "quiet-hours-add",
    editor: "quiet-hours-editor",
    save: "quiet-hours-save",
    remove: "quiet-hours-remove",
    start: "quiet-hours-start",
    end: "quiet-hours-end",
    timezone: "quiet-hours-timezone",
    critical: "quiet-hours-critical",
    issue: "quiet-hours-issue",
    overnight: "quiet-hours-overnight",
    removeDialog: `${dialogPrefixes.quietRemoval}-dialog`,
    removeConfirm: `${dialogPrefixes.quietRemoval}-confirm`,
    problem: "quiet-hours-problem",
    message: "quiet-hours-message",
  },
  escalation: {
    list: "escalation-list",
    add: "escalation-add",
    addSheet: "escalation-add-sheet",
    save: "escalation-save",
    discard: "escalation-discard",
    message: "escalation-message",
    problem: "escalation-problem",
  },
  reach: {
    root: "reach-settings",
    addBrowser: "reach-add-browser",
    refreshDevices: "reach-refresh-devices",
    connectionStatus: "connection-status",
    subscriptionValue: "connection-subscription-value",
    browserState: "reach-browser-state",
    enrollResult: "reach-enroll-result",
    addEmail: "reach-add-email",
    removeDevice: "reach-remove-device",
    removeDeviceDialog: `${dialogPrefixes.deviceRemoval}-dialog`,
    removeDeviceConfirm: `${dialogPrefixes.deviceRemoval}-confirm`,
    widenDialog: `${dialogPrefixes.channelWiden}-dialog`,
    widenConfirm: `${dialogPrefixes.channelWiden}-confirm`,
    deviceRemoved: "reach-device-removed",
    channelSaved: "reach-channel-saved",
    channelProblem: "reach-channel-problem",
    channelSave: "reach-channel-save",
    emailDestination: "reach-email-destination",
    unknownLabelPreserved: "reach-unknown-label-preserved",
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  layout: {
    navLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "dashboard",
            "asks",
            "history",
            "settings",
          ] as const,
        },
      },
    }),
    visibleNavLink: defineDynamicSelector({
      description: "Visible primary destination in the sidebar or the bottom tab bar",
      selectorPattern: '[data-testid="layout-nav-link-${key}"]:visible, [data-testid="layout-nav-link-${key}-tab"]:visible',
      params: { key: { type: "enum", values: ["dashboard", "asks", "history", "settings"] as const } },
    }),

  },
  delivery: {
    recordDetailsLink: defineDynamicSelector({
      description: "Delivery detail link for one notification record ID",
      testIdPattern: "delivery-record-${id}-details",
      params: { id: { type: "string" } },
    }),
  },
  updates: {
    tab: defineDynamicSelector({
      description: "Updates or Delivery tab in the Updates section switcher",
      testIdPattern: "updates-tab-${id}",
      params: { id: { type: "enum", values: ["updates", "delivery"] as const } },
    }),
  },
  asks: {
    urgencyChip: defineDynamicSelector({
      description: "Urgency filter chip in the decision filter sheet",
      testIdPattern: "asks-filter-urgency-${urgency}",
      params: { urgency: { type: "enum", values: ["critical", "high", "normal", "low"] as const } },
    }),
  },
  history: {
    row: defineDynamicSelector({
      description: "One request row in the decision list or History, by request ID",
      selectorPattern: '[data-request-id="${id}"]',
      params: { id: { type: "string" } },
    }),
    filterLink: defineDynamicSelector({
      description: "History filter link by request lifecycle state",
      testIdPattern: "history-filter-${state}",
      params: { state: { type: "enum", values: ["all", "awaiting", "answered", "defaulted", "expired", "withdrawn", "dismissed"] as const } },
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
  settings: {
    row: defineDynamicSelector({
      description: "Row for one settings screen, in the settings list",
      testIdPattern: "settings-row-${key}",
      params: { key: { type: "enum", values: ["reach", "quietHours", "escalation", "appearance", "session", "sources"] as const } },
    }),
    source: defineDynamicSelector({
      description: "Event source group by source scenario id",
      testIdPattern: "notification-source-${scenario}",
      params: { scenario: { type: "string" } },
    }),
    sourceToggle: defineDynamicSelector({
      description: "The on/off switch of one event category, by source scenario id and category key",
      testIdPattern: "notification-source-${scenario}-${category}-toggle",
      params: { scenario: { type: "string" }, category: { type: "string" } },
    }),
  },
  quietHours: {
    schedule: defineDynamicSelector({
      description: "Quiet-hours schedule row, by its first weekday index (Sunday is 0)",
      testIdPattern: "quiet-hours-schedule-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    day: defineDynamicSelector({
      description: "Weekday toggle in the quiet-hours editor, by weekday index (Sunday is 0)",
      testIdPattern: "quiet-hours-day-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
  },
  escalation: {
    step: defineDynamicSelector({
      description: "Escalation step row by channel key",
      testIdPattern: "escalation-step-${channel}",
      params: { channel: { type: "string" } },
    }),
    moveUp: defineDynamicSelector({
      description: "Move an escalation step up, by channel key",
      testIdPattern: "escalation-up-${channel}",
      params: { channel: { type: "string" } },
    }),
    moveDown: defineDynamicSelector({
      description: "Move an escalation step down, by channel key",
      testIdPattern: "escalation-down-${channel}",
      params: { channel: { type: "string" } },
    }),
    remove: defineDynamicSelector({
      description: "Remove an escalation step, by channel key",
      testIdPattern: "escalation-remove-${channel}",
      params: { channel: { type: "string" } },
    }),
    choice: defineDynamicSelector({
      description: "Channel choice in the Add a channel sheet, by channel key",
      testIdPattern: "escalation-choice-${channel}",
      params: { channel: { type: "string" } },
    }),
  },
  reach: {
    device: defineDynamicSelector({
      description: "Device row in the devices list, by device id",
      testIdPattern: "reach-device-${id}",
      params: { id: { type: "string" } },
    }),
    channel: defineDynamicSelector({
      description: "Channel row on a device page, by channel key",
      testIdPattern: "reach-channel-${channel}",
      params: { channel: { type: "string" } },
    }),
    channelEdit: defineDynamicSelector({
      description: "Edit link of a channel row on a device page, by channel key",
      testIdPattern: "reach-channel-edit-${channel}",
      params: { channel: { type: "string" } },
    }),
    label: defineDynamicSelector({
      description: "Content-label checkbox on the channel editor",
      testIdPattern: "reach-label-${label}",
      params: { label: { type: "enum", values: ["public", "private", "sensitive", "critical"] as const } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
