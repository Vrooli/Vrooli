/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. Types, helpers, and the registry builder are
 * imported from ./selectorTypes.ts. This file defines the literal and dynamic
 * selector maps and exports the final `selectors` and `selectorsManifest`.
 *
 * ## Auto-Generated Manifest
 *
 * The `selectors.manifest.json` file is automatically generated from this file
 * during the testing process. If you need to add or modify selectors:
 *
 * 1. Update the `literalSelectors` object below for static selectors
 * 2. Update the `dynamicSelectorDefinitions` object for parameterized selectors
 * 3. The manifest will be regenerated automatically when tests run
 *
 * DO NOT manually edit `selectors.manifest.json` - your changes will be overwritten!
 */

import type { LiteralSelectorTree, DynamicSelectorTree } from "./selectorTypes";
import { defineDynamicSelector, createSelectorRegistry } from "./selectorTypes";

const literalSelectors: LiteralSelectorTree = {
  // Main layout
  app: {
    container: 'inbox-container',
    mobileMenuButton: 'mobile-menu-button',
    mobileBackButton: 'mobile-back-button',
    mobileStarButton: 'mobile-star-button',
    mobileSidebarOverlay: 'mobile-sidebar-overlay',
    closeSidebarButton: 'close-sidebar-button',
  },
  // Sidebar
  sidebar: {
    container: 'sidebar',
    newChatButton: 'new-chat-button',
    nav: 'sidebar-nav',
    manageLabelsButton: 'manage-labels-button',
    addLabelsButton: 'add-labels-button',
    mobileActionsButton: 'sidebar-mobile-actions',
  },
  // Navigation
  nav: {
    inbox: 'nav-inbox',
    starred: 'nav-starred',
    archived: 'nav-archived',
  },
  // Chat list panel
  chatListPanel: {
    container: 'chat-list-panel',
    searchInput: 'chat-search-input',
    clearSearchButton: 'clear-search-button',
    searchModeToggle: 'search-mode-toggle',
    searchModeQuick: 'search-mode-quick',
    searchModeContent: 'search-mode-content',
    switchToContentSearchButton: 'switch-to-content-search',
    list: 'chat-list',
  },
  // Chat view
  chatView: {
    container: 'chat-view',
    loading: 'chat-view-loading',
    header: 'chat-header',
    messageList: 'message-list',
    emptyMessages: 'empty-messages',
    streamingMessage: 'streaming-message',
  },
  // Chat header actions
  chatHeader: {
    renameChatButton: 'rename-chat-button',
    modelSelectorButton: 'model-selector-button',
    addLabelButton: 'add-label-button',
    toggleReadButton: 'toggle-read-button',
    toggleStarButton: 'toggle-star-button',
    toggleArchiveButton: 'toggle-archive-button',
    moreActionsButton: 'chat-more-actions',
    mobileActionsButton: 'chat-mobile-actions',
    confirmDeleteButton: 'confirm-delete-button',
  },
  // Message input
  messageInput: {
    container: 'message-input-container',
    input: 'message-input',
    suggestionsToggle: 'suggestions-toggle',
    sendButton: 'send-message-button',
  },
  // Empty state
  emptyState: {
    container: 'empty-state',
    title: 'empty-state-title',
    subtitle: 'empty-state-subtitle',
    modeHint: 'empty-state-mode-hint',
    mobileTips: 'empty-state-mobile-tips',
  },
  // Dialogs
  dialog: {
    overlay: 'dialog-overlay',
    content: 'dialog-content',
    closeButton: 'dialog-close-button',
  },
  // Rename dialog
  renameDialog: {
    input: 'rename-chat-input',
  },
  // Label manager
  labelManager: {
    newLabelInput: 'new-label-input',
    createButton: 'create-label-button',
  },
  // Dropdown
  dropdown: {
    menu: 'dropdown-menu',
  },
  // Indicators
  indicators: {
    unread: 'unread-indicator',
  },
};

const dynamicSelectorDefinitions: DynamicSelectorTree = {
  chat: {
    item: defineDynamicSelector({
      description: 'Chat list item by chat ID',
      testIdPattern: 'chat-item-${chatId}',
      params: { chatId: { type: 'string' } },
    }),
    message: defineDynamicSelector({
      description: 'Message by message ID',
      testIdPattern: 'message-${messageId}',
      params: { messageId: { type: 'string' } },
    }),
  },
  label: {
    filterButton: defineDynamicSelector({
      description: 'Label filter button in sidebar',
      testIdPattern: 'label-filter-${labelId}',
      params: { labelId: { type: 'string' } },
    }),
    item: defineDynamicSelector({
      description: 'Label item in label manager',
      testIdPattern: 'label-item-${labelId}',
      params: { labelId: { type: 'string' } },
    }),
    deleteButton: defineDynamicSelector({
      description: 'Delete label button',
      testIdPattern: 'delete-label-${labelId}',
      params: { labelId: { type: 'string' } },
    }),
    confirmDeleteButton: defineDynamicSelector({
      description: 'Confirm delete label button',
      testIdPattern: 'confirm-delete-label-${labelId}',
      params: { labelId: { type: 'string' } },
    }),
  },
  model: {
    option: defineDynamicSelector({
      description: 'Model option in dropdown',
      testIdPattern: 'model-option-${modelId}',
      params: { modelId: { type: 'string' } },
    }),
  },
  color: {
    button: defineDynamicSelector({
      description: 'Color picker button',
      testIdPattern: 'color-${color}',
      params: { color: { type: 'string' } },
    }),
  },
};

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
