import { librarySelectors } from "./selectors.library.js";
export { librarySelectors };
// DOC: docs/internal/SEAMS.md
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { defineDynamicSelector } from "@vrooli/ui-selectors";

const literalSelectors = {
  workspace: {
    settings: 'toolbar-settings',
    paneGrid: 'pane-grid',
    newTerminalButton: 'new-terminal-button',
    paneContainer: 'terminal-pane-container',
    sidebarShell: 'workspace-sidebar',
    sidebarToggle: 'workspace-sidebar-toggle',
    sidebarBackdrop: 'workspace-sidebar-backdrop',
    sidebarResizeHandle: 'workspace-sidebar-resize-handle',
    sidebarTopbar: 'workspace-sidebar-topbar',
    sidebarActiveTitle: 'workspace-sidebar-active-title',
    toggleView: 'workspace-toggle-view',
    topEdge: 'workspace-top-edge',
  },
  terminal: {
    pane: 'terminal-pane',
  },
  launcher: {
    dialog: 'terminal-launcher',
    emptyShell: 'launcher-empty-shell',
    customInput: 'launcher-custom-input',
    customLaunch: 'launcher-custom-launch',
    // Destination and appearance disclosure — the dialog now states where the
    // session goes and what it will look like before anything is created.
    destination: 'launcher-destination',
    destinationTrigger: 'launcher-destination-trigger',
    appearance: 'launcher-appearance',
    machinePicker: 'launcher-machine-picker',
    machineMenu: 'launcher-machine-menu',
    machineList: 'launcher-machine-list',
    machineLink: 'launcher-machine-link',
    machineManage: 'launcher-machine-manage',
    targetUnavailable: 'launcher-target-unavailable',
    editShortcuts: 'launcher-edit-shortcuts',
    editTemplates: 'launcher-edit-templates',
    appearanceToggle: 'launcher-appearance-toggle',
    templateMenu: 'launcher-template-menu',
    groupRoleEmpty: 'launcher-group-role-empty',
    agentGrid: 'launcher-agent-grid',
    attributedToggle: 'launcher-attributed-toggle',
    // Group mode: one dialog trip creates a whole group.
    modeOneSession: 'launcher-mode-one-session',
    modeGroup: 'launcher-mode-group',
    templatePicker: 'launcher-template-picker',
    groupName: 'launcher-group-name',
    groupRoleList: 'launcher-group-role-list',
    groupRoleAdd: 'launcher-group-role-add',
    createGroup: 'launcher-create-group',
  },
  groups: {
    drawer: 'manage-groups-drawer',
    filter: 'manage-groups-filter',
    sectionActive: 'manage-groups-section-active',
    sectionEmpty: 'manage-groups-section-empty',
    closeAllEmpty: 'manage-groups-close-all-empty',
    bulkBar: 'manage-groups-bulk-bar',
    autoCloseToggle: 'manage-groups-auto-close',
    summary: 'manage-groups-summary',
    sort: 'manage-groups-sort',
    // Closing a group, reachable from the group header on both surfaces.
    closeMenuItem: 'group-ctx-close-group',
    closeConfirm: 'close-group-confirm',
    closeCancel: 'close-group-cancel',
    closeAlsoSessions: 'close-group-also-sessions',
    closeSummary: 'close-group-summary',
    closeConsequence: 'close-group-consequence',
    // One overlay serves both the launcher destination and the session
    // menu's assign action, so these ids are shared by both entry points.
    assignPicker: 'group-assign-picker',
    pickerList: 'group-picker-list',
    pickerFilter: 'group-picker-filter',
    pickerNone: 'group-picker-option-none',
    pickerEmpty: 'group-picker-empty',
    pickerNoMatches: 'group-picker-no-matches',
    pickerEditToggle: 'group-picker-edit-toggle',
    pickerSectionActive: 'group-picker-section-active',
    pickerSectionEmpty: 'group-picker-section-empty',
    pickerCloseAllEmpty: 'group-picker-close-all-empty',
    pickerUngroupNote: 'group-picker-ungroup-note',
    pickerCreateSubmit: 'group-picker-create-submit',
    undoBanner: 'group-undo-banner',
    undoAction: 'group-undo-action',
    undoDismiss: 'group-undo-dismiss',
  },
  roles: {
    addDialog: 'role-add-dialog',
    addLabel: 'role-add-label',
    addCommand: 'role-add-command',
    addPrompt: 'role-add-prompt',
    addSubmit: 'role-add-submit',
    menu: 'role-menu',
  },
  handoff: {
    composer: 'handoff-composer',
    trigger: 'handoff-trigger',
    targets: 'handoff-targets',
    message: 'handoff-message',
    send: 'handoff-send',
    results: 'handoff-results',
    suggestion: 'handoff-suggestion',
    suggestionDismiss: 'handoff-suggestion-dismiss',
    paneHeaderTrigger: 'handoff-pane-header',
    fileViewerTrigger: 'handoff-file-viewer',
    pendingStrip: 'pending-input-strip',
  },
  templates: {
    panel: 'group-templates-panel',
    create: 'group-templates-create',
    roleList: 'group-templates-role-list',
    roleAdd: 'group-templates-role-add',
    saveAs: 'group-templates-save-as',
  },
  handoffRules: {
    panel: 'handoff-rules-panel',
    create: 'handoff-rules-create',
    footer: 'handoff-rules-footer',
  },
  nav: {
    settings: 'nav-settings',
  },
  settings: {
    error: 'settings-error',
    createProfile: 'create-profile',
    account: 'settings-account',
    accountTab: 'settings-tab-account',
    integrationsTab: 'settings-tab-integrations',
    accountRefreshToken: 'account-refresh-token',
    accountPlan: 'account-plan',
    accountCredits: 'account-credits',
    accountPendingSync: 'account-pending-sync',
    openRouterKeyInput: 'openrouter-key-input',
    openRouterKeySave: 'openrouter-key-save',
    openRouterKeyTest: 'openrouter-key-test',
    openRouterKeyRemove: 'openrouter-key-remove',
    accountButton: 'toolbar-account',
    tunnelManagerAwareness: 'tunnel-manager-awareness',
  },
  toolbar: {
    container: 'mobile-toolbar',
  },
  fleet: {
    drawer: 'machines-drawer',
    // The machines shelf leads: starting a session on a named machine is the
    // errand that brings anyone to this drawer.
    railMachines: 'fleet-rail-machines',
    // "Screens", not "Devices". This shelf lists browsers attached to this
    // console; the device-control scenario owns the other meaning of the word.
    railScreens: 'fleet-rail-screens',
    card: 'fleet-card',
    deviceCard: '[data-testid^="fleet-card-device-"]',
    machineCard: '[data-testid^="fleet-card-machine-"]',
    machineStartSession: '[data-testid^="machines-start-session-"]',
    machineDetails: '[data-testid^="machines-details-"]',
    machineIssues: '[data-testid^="machines-issues-"]',
    machineDetail: '[data-testid^="machine-detail-"]',
    deviceSilhouette: '[data-testid^="fleet-card-device-"] [data-testid="device-silhouette"]',
    machineSilhouette: '[data-testid^="fleet-card-machine-"] [data-testid="machine-silhouette"]',
    deviceFrame: '[data-testid^="device-frame-"]',
    deviceCaption: '[data-testid^="device-caption-"]',
    takeOver: 'device-frame-take-over',
  },
  voice: {
    micButton: 'voice-mic-btn',
    errorTooltip: 'voice-error-tooltip',
  },
  ai: {
    open: 'toolbar-ai',
    input: 'ai-input',
    prompt: 'ai-input-prompt',
    generate: 'ai-input-generate',
    result: 'ai-input-result',
    execute: 'ai-input-execute',
    copy: 'ai-input-copy',
    error: 'ai-input-error',
    resolutionStrip: 'ai-resolution-strip',
    providerProvenance: 'ai-provider-provenance',
  },
  commercial: {
    sourceProvenanceAligned: 'body[data-source-provenance-aligned="true"]',
    localVoiceRefusal: 'body[data-local-voice-refusal="false"]',
  },
  provider: {
    refresh: 'provider-refresh',
    error: 'provider-error',
  },
  error: {
    banner: 'create-error-banner',
    retryButton: 'error-retry-button',
    recoveryHint: 'error-recovery-hint',
  },
  policy: {
    error: 'policy-error',
  },
  locale: {
    switcher: 'locale-switcher',
  },
  messages: {
    searchTrigger: 'messages-search-btn',
    navTrigger: 'msg-jump-trigger',
    navPanel: 'msg-jump-list',
    navScroll: 'msg-jump-scroll',
    searchInput: 'msg-nav-search',
    clearSearch: 'msg-nav-clear',
    resultCount: 'msg-nav-count',
    moreFilters: 'msg-nav-more',
    advancedPanel: 'msg-nav-advanced',
    emptyState: 'msg-nav-empty',
    // Convenience literal for the most common BAS chip; the full chip family
    // is modeled by the dynamic `messages.navChip` selector below.
    chipUser: 'msg-nav-chip-user',
  },
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  provider: {
    card: defineDynamicSelector({
      description: 'Provider health card',
      testIdPattern: 'provider-card-${providerName}',
      params: { providerName: { type: 'string' } },
    }),
    toggle: defineDynamicSelector({
      description: 'Provider enable/disable toggle',
      testIdPattern: 'provider-toggle-${providerName}',
      params: { providerName: { type: 'string' } },
    }),
  },
  policy: {
    select: defineDynamicSelector({
      description: 'Policy select dropdown for a session',
      testIdPattern: 'policy-select-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
    countdown: defineDynamicSelector({
      description: 'Policy countdown timer for a session',
      testIdPattern: 'policy-countdown-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
  },
  error: {
    boundary: defineDynamicSelector({
      description: 'Error boundary wrapper for a region',
      testIdPattern: 'error-boundary-${region}',
      params: { region: { type: 'string' } },
    }),
  },
  drawer: {
    session: defineDynamicSelector({
      description: 'Session row in drawer',
      testIdPattern: 'drawer-session-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
  },
  sidebar: {
    session: defineDynamicSelector({
      description: 'Session row in workspace sidebar',
      testIdPattern: 'sidebar-session-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
  },
  launcher: {
    shortcut: defineDynamicSelector({
      description: 'Launcher shortcut button',
      testIdPattern: 'launcher-shortcut-${label}',
      params: { label: { type: 'string' } },
    }),
    agentCard: defineDynamicSelector({
      description: 'Launcher agent card in the two-column grid',
      testIdPattern: 'launcher-agent-${label}',
      params: { label: { type: 'string' } },
    }),
    machineOption: defineDynamicSelector({
      description: 'Machine row inside the launcher machine picker listbox',
      testIdPattern: 'launcher-machine-option-${targetId}',
      params: { targetId: { type: 'string' } },
    }),
  },
  roles: {
    sidebarRow: defineDynamicSelector({
      description: 'Waiting role row in the workspace sidebar',
      testIdPattern: 'sidebar-waiting-role-${roleId}',
      params: { roleId: { type: 'string' } },
    }),
    sidebarStart: defineDynamicSelector({
      description: 'Start control on a waiting role row',
      testIdPattern: 'sidebar-waiting-role-start-${roleId}',
      params: { roleId: { type: 'string' } },
    }),
    sidebarHandoff: defineDynamicSelector({
      description: 'Handoff control on a waiting role row',
      testIdPattern: 'sidebar-waiting-role-handoff-${roleId}',
      params: { roleId: { type: 'string' } },
    }),
    sidebarMenu: defineDynamicSelector({
      description: 'Overflow menu control on a waiting role row',
      testIdPattern: 'sidebar-waiting-role-menu-${roleId}',
      params: { roleId: { type: 'string' } },
    }),
    tabRow: defineDynamicSelector({
      description: 'Waiting role chip in the tab strip',
      testIdPattern: 'tab-waiting-role-${roleId}',
      params: { roleId: { type: 'string' } },
    }),
  },
  handoff: {
    target: defineDynamicSelector({
      description: 'Selectable target row in the handoff composer',
      testIdPattern: 'handoff-target-${targetId}',
      params: { targetId: { type: 'string' } },
    }),
    result: defineDynamicSelector({
      description: 'Per-target result line after a handoff is sent',
      testIdPattern: 'handoff-result-${targetId}',
      params: { targetId: { type: 'string' } },
    }),
  },
  groups: {
    selectRow: defineDynamicSelector({
      description: 'Selection checkbox for a group row in the manager',
      testIdPattern: 'manage-groups-select-${groupId}',
      params: { groupId: { type: 'string' } },
    }),
    closeRow: defineDynamicSelector({
      description: 'Close control for a group row in the manager',
      testIdPattern: 'manage-groups-close-${groupId}',
      params: { groupId: { type: 'string' } },
    }),
    pickerOption: defineDynamicSelector({
      description: 'Group card in the group picker overlay',
      testIdPattern: 'group-picker-option-${groupId}',
      params: { groupId: { type: 'string' } },
    }),
    pickerEditRow: defineDynamicSelector({
      description: 'Group row in the picker overlay while editing',
      testIdPattern: 'group-picker-edit-${groupId}',
      params: { groupId: { type: 'string' } },
    }),
    pickerRename: defineDynamicSelector({
      description: 'Rename field for a group in the picker overlay',
      testIdPattern: 'group-picker-rename-${groupId}',
      params: { groupId: { type: 'string' } },
    }),
    pickerRecolor: defineDynamicSelector({
      description: 'Colour control for a group in the picker overlay',
      testIdPattern: 'group-picker-recolor-${groupId}',
      params: { groupId: { type: 'string' } },
    }),
    pickerCloseGroup: defineDynamicSelector({
      description: 'Close control for a group in the picker overlay',
      testIdPattern: 'group-picker-close-${groupId}',
      params: { groupId: { type: 'string' } },
    }),
  },
  templates: {
    row: defineDynamicSelector({
      description: 'Template row in the templates panel',
      testIdPattern: 'group-template-${templateId}',
      params: { templateId: { type: 'string' } },
    }),
    deleteRow: defineDynamicSelector({
      description: 'Delete control for a template row',
      testIdPattern: 'group-template-delete-${templateId}',
      params: { templateId: { type: 'string' } },
    }),
  },
  handoffRules: {
    row: defineDynamicSelector({
      description: 'Rule row in the handoff rules panel',
      testIdPattern: 'handoff-rule-${ruleId}',
      params: { ruleId: { type: 'string' } },
    }),
    toggleRow: defineDynamicSelector({
      description: 'Enable toggle for a rule row',
      testIdPattern: 'handoff-rule-toggle-${ruleId}',
      params: { ruleId: { type: 'string' } },
    }),
    deleteRow: defineDynamicSelector({
      description: 'Delete control for a rule row',
      testIdPattern: 'handoff-rule-delete-${ruleId}',
      params: { ruleId: { type: 'string' } },
    }),
  },
  toolbar: {
    key: defineDynamicSelector({
      description: 'Mobile toolbar key button',
      testIdPattern: 'toolbar-key-${label}',
      params: { label: { type: 'string' } },
    }),
  },
  sessions: {
    row: defineDynamicSelector({
      description: 'Session row in sessions page',
      testIdPattern: 'session-row-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
    policy: defineDynamicSelector({
      description: 'Policy select for session in sessions page',
      testIdPattern: 'session-policy-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
    deleteButton: defineDynamicSelector({
      description: 'Delete button for session in sessions page',
      testIdPattern: 'session-delete-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
  },
  settings: {
    profile: defineDynamicSelector({
      description: 'Shortcut profile editor card',
      testIdPattern: 'shortcut-profile-${profileId}',
      params: { profileId: { type: 'string' } },
    }),
    profileName: defineDynamicSelector({
      description: 'Profile name input',
      testIdPattern: 'profile-name-${profileId}',
      params: { profileId: { type: 'string' } },
    }),
    profileSave: defineDynamicSelector({
      description: 'Profile save button',
      testIdPattern: 'profile-save-${profileId}',
      params: { profileId: { type: 'string' } },
    }),
    profileDelete: defineDynamicSelector({
      description: 'Profile delete button',
      testIdPattern: 'profile-delete-${profileId}',
      params: { profileId: { type: 'string' } },
    }),
    entryLabel: defineDynamicSelector({
      description: 'Shortcut entry label input',
      testIdPattern: 'entry-label-${profileId}-${entryIdx}',
      params: { profileId: { type: 'string' }, entryIdx: { type: 'number' } },
    }),
    entryCommand: defineDynamicSelector({
      description: 'Shortcut entry command input',
      testIdPattern: 'entry-command-${profileId}-${entryIdx}',
      params: { profileId: { type: 'string' }, entryIdx: { type: 'number' } },
    }),
  },
  workspace: {
    paneContainerBySession: defineDynamicSelector({
      description: 'Pane container by session ID',
      selectorPattern: '[data-testid="terminal-pane-container"][data-session-id="${sessionId}"]',
      params: { sessionId: { type: 'string' } },
    }),
    closeButtonBySession: defineDynamicSelector({
      description: 'Close button for a specific session pane',
      testIdPattern: 'terminal-close-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
  },
  terminal: {
    paneBySession: defineDynamicSelector({
      description: 'Terminal host element by session ID',
      selectorPattern: '[data-testid="terminal-pane"][data-session-id="${sessionId}"]',
      params: { sessionId: { type: 'string' } },
    }),
  },
  locale: {
    toggle: defineDynamicSelector({
      description: 'Locale switcher toggle button for a specific locale code',
      testIdPattern: 'locale-toggle-${code}',
      params: { code: { type: 'string' } },
    }),
  },
  messages: {
    pane: defineDynamicSelector({
      description: 'Messages pane for a session',
      testIdPattern: 'messages-pane-${sessionId}',
      params: { sessionId: { type: 'string' } },
    }),
    card: defineDynamicSelector({
      description: 'Message card by event ID',
      testIdPattern: 'msg-card-${eventId}',
      params: { eventId: { type: 'string' } },
    }),
    speakFromHere: defineDynamicSelector({
      description: 'Read from here button on a message card',
      testIdPattern: 'msg-speak-from-${eventId}',
      params: { eventId: { type: 'string' } },
    }),
    speakOne: defineDynamicSelector({
      description: 'Read this message button on a message card',
      testIdPattern: 'msg-speak-one-${eventId}',
      params: { eventId: { type: 'string' } },
    }),
    navResultRow: defineDynamicSelector({
      description: 'Message navigator result row by event ID',
      testIdPattern: 'msg-jump-item-${eventId}',
      params: { eventId: { type: 'string' } },
    }),
    navChip: defineDynamicSelector({
      description: 'Message navigator primary filter chip',
      testIdPattern: 'msg-nav-chip-${id}',
      params: { id: { type: 'enum', values: ['all', 'user', 'assistant', 'failed', 'unheard'] } },
    }),
    navSourceOption: defineDynamicSelector({
      description: 'Message navigator source filter option',
      testIdPattern: 'msg-nav-source-${source}',
      params: { source: { type: 'enum', values: ['claude', 'codex', 'opencode', 'grok'] } },
    }),
    navStatusOption: defineDynamicSelector({
      description: 'Message navigator status filter option',
      testIdPattern: 'msg-nav-status-${status}',
      params: { status: { type: 'enum', values: ['all', 'unheard', 'played', 'failed', 'summarized'] } },
    }),
    navContentOption: defineDynamicSelector({
      description: 'Message navigator content filter option',
      testIdPattern: 'msg-nav-content-${content}',
      params: { content: { type: 'enum', values: ['all', 'code', 'fileReference', 'long'] } },
    }),
    navSortOption: defineDynamicSelector({
      description: 'Message navigator sort option',
      testIdPattern: 'msg-nav-sort-${mode}',
      params: { mode: { type: 'enum', values: ['oldest', 'newest', 'relevance'] } },
    }),
    navGroupOption: defineDynamicSelector({
      description: 'Message navigator grouping option',
      testIdPattern: 'msg-nav-group-${mode}',
      params: { mode: { type: 'enum', values: ['turn', 'flat', 'role'] } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
