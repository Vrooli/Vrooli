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
  notifications: {
    summary: "notifications-summary",
  },
  integrations: {
    modeIndicator: "integrations-mode-indicator",
    modeValue: "integrations-mode-value",
    settingsPanel: "integrations-settings-panel",
    overrideSelect: "integrations-override-select",
    statusTable: "integrations-status-table",
    deploymentProfilePanel: "integrations-deployment-profile-panel",
    deploymentProfileSelect: "integrations-deployment-profile-select",
    deploymentEndpointInput: "integrations-deployment-endpoint-input",
    deploymentProfileSave: "integrations-deployment-profile-save",
    deploymentProfileStatus: "integrations-deployment-profile-status",
  },
  search: {
    omnibox: "search-omnibox",
    input: "search-omnibox-input",
    status: "search-omnibox-status",
    results: "search-omnibox-results",
  },
  chat: {
    workspace: "chat-workspace",
    sidebar: "chat-sidebar",
    newChatButton: "chat-new-chat-button",
    newAgentChatButton: "chat-new-agent-chat-button",
    newGroupButton: "chat-new-group-button",
    messageList: "chat-message-list",
    emptyState: "chat-empty-state",
    composer: "chat-composer",
    composerInput: "chat-composer-input",
    sendButton: "chat-send-button",
    stopButton: "chat-stop-button",
    modelSelect: "chat-model-select",
    modeSelect: "chat-mode-select",
    harnessSelect: "chat-harness-select",
    webSearchToggle: "chat-web-search-toggle",
    skillInput: "chat-skill-input",
    branchPrevious: "chat-branch-previous",
    branchNext: "chat-branch-next",
    editButton: "chat-edit-button",
    regenerateButton: "chat-regenerate-button",
  },
  locale: {
    switcher: "locale-switcher",
  },
  layout: {
    shell: "layout-shell",
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
  },
  pages: {
    dashboard: "page-dashboard",
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
    navLink: defineDynamicSelector({
      description: "App shell navigation link by canonical nav key",
      testIdPattern: "layout-nav-link-${key}",
      params: {
        key: { type: "enum", values: ["dashboard", "settings"] as const },
      },
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
  integrations: {
    statusRow: defineDynamicSelector({
      description: "Integration readiness table row by integration id",
      testIdPattern: "integrations-status-row-${id}",
      params: { id: { type: "string" } },
    }),
    deploymentProfileOption: defineDynamicSelector({
      description: "Deployment profile option by profile id",
      testIdPattern: "integrations-deployment-profile-option-${id}",
      params: { id: { type: "enum", values: ["client", "local-control", "automation"] as const } },
    }),
  },
  search: {
    result: defineDynamicSelector({
      description: "Search omnibox result row by visible index",
      testIdPattern: "search-omnibox-result-${index}",
      params: { index: { type: "number" } },
    }),
  },
  chat: {
    group: defineDynamicSelector({
      description: "Chat sidebar group row by group id",
      testIdPattern: "chat-group-${id}",
      params: { id: { type: "string" } },
    }),
    chat: defineDynamicSelector({
      description: "Chat sidebar conversation row by chat id",
      testIdPattern: "chat-item-${id}",
      params: { id: { type: "string" } },
    }),
    message: defineDynamicSelector({
      description: "Chat message row by message id",
      testIdPattern: "chat-message-${id}",
      params: { id: { type: "string" } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
