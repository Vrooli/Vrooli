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
  realtime: {
    indicator: "realtime-indicator",
  },
  join: {
    screen: "join-screen",
    codeInput: "join-code-input",
    deviceNameInput: "join-device-name-input",
    redeemButton: "join-redeem-button",
    requestButton: "join-request-button",
    waiting: "join-waiting",
    approveThisDevice: "join-approve-this-device",
    error: "join-error",
  },
  receive: {
    panel: "receive-panel",
    loading: "receive-loading",
    empty: "receive-empty",
    error: "receive-error",
    search: "receive-search",
    sort: "receive-sort",
    filter: "receive-filter",
    viewToggle: "receive-view-toggle",
    list: "receive-list",
  },
  send: {
    panel: "send-panel",
    dropZone: "send-drop-zone",
    fileInput: "send-file-input",
    textInput: "send-text-input",
    addText: "send-add-text",
    staged: "send-staged",
    sendButton: "send-button",
    status: "send-status",
  },
  devices: {
    panel: "devices-panel",
    loading: "devices-loading",
    empty: "devices-empty",
    signInPrompt: "devices-sign-in-prompt",
    list: "devices-list",
    issuePanel: "devices-issue-panel",
    issueNameInput: "devices-issue-name-input",
    issueButton: "devices-issue-button",
    issuedCode: "devices-issued-code",
    issuedQr: "devices-issued-qr",
    pendingBanner: "devices-pending-banner",
  },
  owner: {
    panel: "owner-panel",
    signOutButton: "owner-sign-out-button",
    status: "owner-status",
  },
  onboarding: {
    screen: "onboarding-screen",
    setupChoice: "onboarding-setup-choice",
    joinChoice: "onboarding-join-choice",
    back: "onboarding-back",
  },
  login: {
    form: "login-form",
    tabSignIn: "login-tab-signin",
    tabCreate: "login-tab-create",
    emailInput: "login-email-input",
    passwordInput: "login-password-input",
    usernameInput: "login-username-input",
    submit: "login-submit",
    error: "login-error",
  },
  setupDevice: {
    panel: "setup-device-panel",
    nameInput: "setup-device-name-input",
    submit: "setup-device-submit",
    error: "setup-device-error",
    joinInstead: "setup-device-join-instead",
    signOut: "setup-device-sign-out",
  },
  session: {
    panel: "session-panel",
    signOutButton: "session-sign-out-button",
  },
  locale: {
    switcher: "locale-switcher",
  },
  layout: {
    shell: "layout-shell",
    topBar: "layout-top-bar",
    sidebar: "layout-sidebar",
    bottomNav: "layout-bottom-nav",
    main: "layout-main",
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
  },
  pages: {
    transfer: "page-transfer",
    devices: "page-devices",
    settings: "page-settings",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
} satisfies LiteralSelectorTree;

// Per-locale toggle test IDs are emitted by `locale.toggle({ code })` below.
// We deliberately do NOT also declare static `toggleEn` / `toggleJa` literals —
// the dynamic form is the single source of truth, and duplicating it here would
// drift the moment a new locale is added to LOCALE_CODES.
//
// `code` is constrained to `LOCALE_CODES` so `selectors.locale.toggle({ code: "fr" })`
// is a TypeScript error when "fr" isn't a supported locale. The runtime enum
// validation in `normalizeParams` provides the same guarantee at call time.
const dynamicSelectorDefinitions = {
  locale: {
    toggle: defineDynamicSelector({
      description: "Locale toggle button by language code",
      testIdPattern: "locale-toggle-${code}",
      params: { code: { type: "enum", values: LOCALE_CODES } },
    }),
  },
  layout: {
    sidebarLink: defineDynamicSelector({
      description: "Sidebar navigation link by canonical nav key",
      testIdPattern: "layout-sidebar-link-${key}",
      params: { key: { type: "enum", values: ["transfer", "devices", "settings"] as const } },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: { key: { type: "enum", values: ["transfer", "devices", "settings"] as const } },
    }),
  },
  receive: {
    deliveryInfo: defineDynamicSelector({
      description: "Hub relay storage and audience explanation for a received item",
      testIdPattern: "receive-delivery-info-${id}",
      params: { id: { type: "string" } },
    }),
    item: defineDynamicSelector({
      description: "A received item card/row by item id",
      testIdPattern: "receive-item-${id}",
      params: { id: { type: "string" } },
    }),
    download: defineDynamicSelector({
      description: "Download button for a received file item by id",
      testIdPattern: "receive-download-${id}",
      params: { id: { type: "string" } },
    }),
    copy: defineDynamicSelector({
      description: "Copy button for a received text item by id",
      testIdPattern: "receive-copy-${id}",
      params: { id: { type: "string" } },
    }),
    remove: defineDynamicSelector({
      description: "Remove button for a received item by id",
      testIdPattern: "receive-remove-${id}",
      params: { id: { type: "string" } },
    }),
  },
  send: {
    stagedItem: defineDynamicSelector({
      description: "A staged outgoing item by its local staging id",
      testIdPattern: "send-staged-${id}",
      params: { id: { type: "string" } },
    }),
    removeStaged: defineDynamicSelector({
      description: "Remove button for a staged outgoing item by its local id",
      testIdPattern: "send-remove-staged-${id}",
      params: { id: { type: "string" } },
    }),
  },
  devices: {
    row: defineDynamicSelector({
      description: "A device row by device id",
      testIdPattern: "devices-row-${id}",
      params: { id: { type: "string" } },
    }),
    rename: defineDynamicSelector({
      description: "Rename action for a device by id",
      testIdPattern: "devices-rename-${id}",
      params: { id: { type: "string" } },
    }),
    revoke: defineDynamicSelector({
      description: "Revoke action for a device by id",
      testIdPattern: "devices-revoke-${id}",
      params: { id: { type: "string" } },
    }),
    approve: defineDynamicSelector({
      description: "Approve action for a pending device by id",
      testIdPattern: "devices-approve-${id}",
      params: { id: { type: "string" } },
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
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
