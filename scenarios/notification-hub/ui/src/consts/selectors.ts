import { librarySelectors } from "./selectors.library";
export { librarySelectors };
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
  layout: {
    shell: "layout-shell",
    main: "layout-shell-main",
    menu: "layout-shell-menu",
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
    history: "page-history",
    delivery: "page-delivery",
    deliveryDetail: "page-delivery-detail",
  },
  delivery: {
    analyticsRecordsAction: "delivery-analytics-records-link",
    analyticsRetry: "delivery-analytics-retry",
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
  },
  reach: {
    root: "reach-settings",
    emptyDevices: "reach-empty-devices",
    addBrowser: "reach-add-browser",
    refreshDevices: "reach-refresh-devices",
    permissionRecovery: "reach-permission-recovery",
    permissionRecoveryBack: "reach-permission-recovery-back",
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
  quietHours: {
    root: "quiet-hours-root",
    openWeek: "quiet-hours-open-week",
    saveButton: "quiet-hours-save",
    stageRemoval: "quiet-hours-stage-removal",
    review: "quiet-hours-review",
    reviewButton: "quiet-hours-review-button",
    message: "quiet-hours-message",
    checkSavedButton: "quiet-hours-check-saved",
    timezoneReview: "quiet-hours-timezone-review",
    timezoneReviewStage: "quiet-hours-timezone-review-stage",
    timezoneReviewKeepEditing: "quiet-hours-timezone-review-keep-editing",
  },
  escalationOrder: {
    root: "escalation-order-root",
    openButton: "escalation-order-open",
    list: "escalation-order-list",
    addSelect: "escalation-order-add-select",
    addButton: "escalation-order-add",
    clearButton: "escalation-order-clear",
    confirmClear: "escalation-order-confirm-clear",
    cancelClear: "escalation-order-cancel-clear",
    saveButton: "escalation-order-save",
    checkSavedButton: "escalation-order-check-saved",
    editButton: "escalation-order-edit",
    review: "escalation-order-review",
    message: "escalation-order-message",
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
            "delivery",
            "settings",
          ] as const,
        },
      },
    }),
    visibleNavLink: defineDynamicSelector({
      description: "Visible primary destination in sidebar, compact menu or bottom navigation",
      selectorPattern: '[data-testid="layout-nav-link-${key}"]:visible, [data-testid="layout-nav-link-${key}-tab"]:visible',
      params: { key: { type: "enum", values: ["dashboard", "asks", "history", "delivery", "settings"] as const } },
    }),

  },
  delivery: {
    recordDetailsLink: defineDynamicSelector({
      description: "Delivery detail link for one notification record ID",
      testIdPattern: "delivery-record-${id}-details",
      params: { id: { type: "string" } },
    }),
  },
  history: {
    filterLink: defineDynamicSelector({
      description: "History filter link by request lifecycle state",
      testIdPattern: "history-filter-${state}",
      params: { state: { type: "enum", values: ["all", "awaiting", "answered", "defaulted", "expired", "withdrawn"] as const } },
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
  quietHours: {
    timezoneReviewInput: defineDynamicSelector({
      description: "New timezone input on a quiet-window timezone review",
      testIdPattern: "quiet-hours-timezone-review-input-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    timezoneReviewOpen: defineDynamicSelector({
      description: "Open the explicit quiet-window timezone review for a weekday index",
      testIdPattern: "quiet-hours-timezone-review-open-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    editDay: defineDynamicSelector({
      description: "Open the quiet-hours editor for a weekday index",
      testIdPattern: "quiet-hours-edit-day-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    doneDay: defineDynamicSelector({
      description: "Stage and close the quiet-hours editor for a weekday index",
      testIdPattern: "quiet-hours-done-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    cancelDay: defineDynamicSelector({
      description: "Discard and close the quiet-hours editor for a weekday index",
      testIdPattern: "quiet-hours-cancel-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    dayEnabled: defineDynamicSelector({
      description: "Quiet-hours enabled checkbox by weekday index",
      testIdPattern: "quiet-hours-enabled-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    dayStart: defineDynamicSelector({
      description: "Quiet-hours start-time input by weekday index",
      testIdPattern: "quiet-hours-start-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    dayEnd: defineDynamicSelector({
      description: "Quiet-hours end-time input by weekday index",
      testIdPattern: "quiet-hours-end-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    dayTimezone: defineDynamicSelector({
      description: "Quiet-hours timezone input by weekday index",
      testIdPattern: "quiet-hours-timezone-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
    removeDay: defineDynamicSelector({
      description: "Stage removal of the saved quiet window for a weekday index",
      testIdPattern: "quiet-hours-remove-${day}",
      params: { day: { type: "enum", values: ["0", "1", "2", "3", "4", "5", "6"] as const } },
    }),
  },
  escalationOrder: {
    channelChoice: defineDynamicSelector({
      description: "Supported channel choice in the escalation-order Add a channel view",
      testIdPattern: "escalation-order-channel-choice-${channel}",
      params: { channel: { type: "enum", values: ["browser", "email", "webhook", "web_push"] as const } },
    }),
    channelRow: defineDynamicSelector({
      description: "Escalation channel row by stable channel key",
      testIdPattern: "escalation-order-channel-${channel}",
      params: { channel: { type: "string" } },
    }),
    moveUp: defineDynamicSelector({
      description: "Escalation channel move-up control by stable channel key",
      testIdPattern: "escalation-order-up-${channel}",
      params: { channel: { type: "string" } },
    }),
    moveDown: defineDynamicSelector({
      description: "Escalation channel move-down control by stable channel key",
      testIdPattern: "escalation-order-down-${channel}",
      params: { channel: { type: "string" } },
    }),
    remove: defineDynamicSelector({
      description: "Escalation channel removal control by stable channel key",
      testIdPattern: "escalation-order-remove-${channel}",
      params: { channel: { type: "string" } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
