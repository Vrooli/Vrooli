/**
 * Canonical registry of every `data-testid` string used by the Browser Automation Studio UI.
 *
 * Components import from this file to avoid hardcoded selectors, and the workflow tooling
 * consumes the same registry to validate BAS playbooks. When adding a new test id, update
 * this file so both the UI and automation helpers stay in sync.
 */

const toSelector = (testId: string) => `[data-testid="${testId}"]`;

type DynamicParamType = "string" | "number";

type DynamicSelectorDefinition = {
  pattern: string;
  selectorPattern?: string;
  params: readonly string[];
  allowedValues?: Record<string, readonly string[]>;
  valueTypes?: Record<string, DynamicParamType>;
  description?: string;
};

type SelectorTree<T> = {
  [K in keyof T]: T[K] extends string ? string : SelectorTree<T[K]>;
};

export const testIds = {
  aiGenerateButton: "ai-generate-button",
  aiPromptInput: "ai-prompt-input",
  aiPromptModal: "ai-prompt-modal",
  aiWorkflowNameInput: "ai-workflow-name-input",
  appReady: "app-ready",
  appBackground: "background",
  breadcrumbProject: "breadcrumb-project",
  dashboardNewProjectButton: "dashboard-new-project-button",
  dialogCloseButton: "dialog-close-button",
  dialogContent: "dialog-content",
  executionCard: "execution-card",
  executionFilterAll: "execution-filter-all",
  executionHistory: "execution-history",
  executionHistoryList: "execution-history-list",
  executionHistoryMock: "execution-history-mock",
  executionHistoryRefresh: "execution-history-refresh",
  executionItem: "execution-item",
  executionLogs: "execution-logs",
  executionScreenshot: "execution-screenshot",
  executionScreenshots: "execution-screenshots",
  executionStatus: "execution-status",
  executionStatusCancelled: "execution-status-cancelled",
  executionStatusCompleted: "execution-status-completed",
  executionStatusFailed: "execution-status-failed",
  executionStopButton: "execution-stop-button",
  executionTabExecutions: "execution-tab-executions",
  executionTabLogs: "execution-tab-logs",
  executionTabReplay: "execution-tab-replay",
  executionTabScreenshots: "execution-tab-screenshots",
  executionViewButton: "execution-view-button",
  executionViewer: "execution-viewer",
  executionViewerMock: "execution-viewer-mock",
  executionsTab: "executions-tab",
  exportReplayButton: "export-replay-button",
  exportConfirmButton: "export-confirm-button",
  exportInProgress: "export-in-progress",
  formError: "form-error",
  header: "header",
  headerDebugButton: "header-debug-button",
  headerBackToProjectButton: "header-back-to-project-button",
  headerBackToDashboardButton: "header-back-to-dashboard-button",
  headerExecuteButton: "header-execute-button",
  headerInfoButton: "header-info-button",
  headerInfoCloseButton: "header-info-close-button",
  headerInfoPopover: "header-info-popover",
  headerSaveErrorDetailsButton: "header-save-error-details-button",
  headerSaveErrorDialog: "header-save-error-dialog",
  headerSaveErrorDialogCloseButton: "header-save-error-dialog-close-button",
  headerSaveErrorDismissButton: "header-save-error-dismiss-button",
  headerSaveRetryButton: "header-save-retry-button",
  headerSaveStatusIndicator: "header-save-status-indicator",
  headerTitleCancelButton: "header-title-cancel-button",
  headerTitleEditButton: "header-title-edit-button",
  headerTitleInput: "header-title-input",
  headerTitleSaveButton: "header-title-save-button",
  headerVersionConflictDetailsButton: "header-version-conflict-details-button",
  headerVersionConflictForceSaveButton:
    "header-version-conflict-force-save-button",
  headerVersionConflictRefreshButton: "header-version-conflict-refresh-button",
  headerVersionConflictReloadButton: "header-version-conflict-reload-button",
  headerVersionHistoryButton: "header-version-history-button",
  headerVersionHistoryCloseButton: "header-version-history-close-button",
  headerVersionHistoryDialog: "header-version-history-dialog",
  headerWorkflowTitle: "header-workflow-title",
  heartbeatIndicator: "heartbeat-indicator",
  heartbeatStatus: "heartbeat-status",
  heartbeatLagWarning: "heartbeat-lag-warning",
  logEntry: "log-entry",
  minimap: "minimap",
  monacoEditor: "monaco-editor",
  newWorkflowButton: "new-workflow-button",
  newWorkflowButtonFab: "new-workflow-button-fab",
  nodePaletteAssertCard: "node-palette-assert-card",
  nodePaletteCategoryNavigationToggle:
    "node-palette-category-navigation-toggle",
  nodePaletteClearRecentsButton: "node-palette-clear-recents-button",
  nodePaletteClickCard: "node-palette-click-card",
  nodePaletteContainer: "node-palette-container",
  nodePaletteDragDropCard: "node-palette-dragDrop-card",
  nodePaletteDragdropCard: "node-palette-dragdrop-card",
  nodePaletteFavoritesSection: "node-palette-favorites-section",
  nodePaletteNavigateCard: "node-palette-navigate-card",
  nodePaletteNavigateFavoriteButton: "node-palette-navigate-favorite-button",
  nodePaletteNoResults: "node-palette-no-results",
  nodePaletteQuickAccess: "node-palette-quick-access",
  nodePaletteQuickAccessToggle: "node-palette-quick-access-toggle",
  nodePaletteRecentsSection: "node-palette-recents-section",
  nodePaletteScreenshotCard: "node-palette-screenshot-card",
  nodePaletteSearchInput: "node-palette-search-input",
  nodePaletteSetVariableCard: "node-palette-setVariable-card",
  nodePaletteSetvariableCard: "node-palette-setvariable-card",
  nodePaletteUseVariableCard: "node-palette-useVariable-card",
  nodePaletteUsevariableCard: "node-palette-usevariable-card",
  nodePaletteWaitCard: "node-palette-wait-card",
  nodePropertyUrlInput: "node-property-url-input",
  projectCard: "project-card",
  projectCardTitle: "project-card-title",
  projectEditButton: "project-edit-button",
  projectModal: "project-modal",
  projectModalCancel: "project-modal-cancel",
  projectModalDescriptionInput: "project-modal-description-input",
  projectModalForm: "project-modal-form",
  projectModalNameError: "project-modal-name-error",
  projectModalNameInput: "project-modal-name-input",
  projectModalSubmit: "project-modal-submit",
  projectTabExecutions: "project-tab-executions",
  projectsGrid: "projects-grid",
  reactFlowCanvas: "react-flow-canvas",
  replayPlayer: "replay-player",
  replayScreenshot: "replay-screenshot",
  replayTimeline: "replay-timeline",
  replayViewer: "replay-viewer",
  responsiveDialogContent: "responsive-dialog-content",
  responsiveDialogOverlay: "responsive-dialog-overlay",
  switchToManualButton: "switch-to-manual-button",
  timelineFrame: "timeline-frame",
  toolbarConfigureViewportButton: "toolbar-configure-viewport-button",
  toolbarConfigureViewportMenuItem: "toolbar-configure-viewport-menu-item",
  toolbarDeleteButton: "toolbar-delete-button",
  toolbarDeleteMenuItem: "toolbar-delete-menu-item",
  toolbarDuplicateButton: "toolbar-duplicate-button",
  toolbarDuplicateMenuItem: "toolbar-duplicate-menu-item",
  toolbarFitViewButton: "toolbar-fit-view-button",
  toolbarFitViewMenuItem: "toolbar-fit-view-menu-item",
  toolbarLayoutMenu: "toolbar-layout-menu",
  toolbarLayoutMenuButton: "toolbar-layout-menu-button",
  toolbarLockButton: "toolbar-lock-button",
  toolbarUnlockButton: "toolbar-unlock-button",
  toolbarRedoButton: "toolbar-redo-button",
  toolbarSelectionMenu: "toolbar-selection-menu",
  toolbarSelectionMenuButton: "toolbar-selection-menu-button",
  toolbarToggleLockMenuItem: "toolbar-toggle-lock-menu-item",
  toolbarUndoButton: "toolbar-undo-button",
  toolbarZoomInButton: "toolbar-zoom-in-button",
  toolbarZoomOutButton: "toolbar-zoom-out-button",
  urlModeButton: "url-mode-button",
  variableSuggestions: "variable-suggestions",
  variableSuggestionsEmpty: "variable-suggestions-empty",
  viewportDialogCancelButton: "viewport-dialog-cancel-button",
  viewportDialogError: "viewport-dialog-error",
  viewportDialogHeightInput: "viewport-dialog-height-input",
  viewportDialogSaveButton: "viewport-dialog-save-button",
  viewportDialogTitle: "viewport-dialog-title",
  viewportDialogWidthInput: "viewport-dialog-width-input",
  workflowBuilderCanvas: "workflow-builder-canvas",
  workflowBuilderCodeApplyButton: "workflow-builder-code-apply-button",
  workflowBuilderCodeEditor: "workflow-builder-code-editor",
  workflowBuilderCodeEditorContainer: "workflow-builder-code-editor-container",
  workflowBuilderCodeError: "workflow-builder-code-error",
  workflowBuilderCodeLineCount: "workflow-builder-code-line-count",
  workflowBuilderCodeModeButton: "workflow-builder-code-mode-button",
  workflowBuilderCodeResetButton: "workflow-builder-code-reset-button",
  workflowBuilderCodeToolbar: "workflow-builder-code-toolbar",
  workflowBuilderCodeValidation: "workflow-builder-code-validation",
  workflowBuilderCodeView: "workflow-builder-code-view",
  workflowBuilderViewModeToggle: "workflow-builder-view-mode-toggle",
  workflowBuilderViewportDialog: "workflow-builder-viewport-dialog",
  workflowBuilderVisualModeButton: "workflow-builder-visual-mode-button",
  workflowCard: "workflow-card",
  workflowExecuteButton: "workflow-execute-button",
  workflowSearchClear: "workflow-search-clear",
  workflowSearchInput: "workflow-search-input",
  workflowToolbar: "workflow-toolbar",
  workflowsTab: "workflows-tab",
} as const;

export type TestIdRegistry = typeof testIds;
export type TestIdKey = keyof TestIdRegistry;

const mapSelectors = <T>(registry: T): SelectorTree<T> => {
  const result: Record<string, any> = {};
  for (const [key, value] of Object.entries(registry as Record<string, any>)) {
    result[key] =
      typeof value === "string" ? toSelector(value) : mapSelectors(value);
  }
  return result as SelectorTree<T>;
};

export const workflowSelectors = mapSelectors(testIds);
export const selectorFromTestId = toSelector;

const DYNAMIC_TEMPLATE_PATTERN = /\$\{([^}]+)\}/g;

const formatDynamicTestId = (
  pattern: string,
  params: Record<string, string | number>,
) =>
  pattern.replace(DYNAMIC_TEMPLATE_PATTERN, (_match, key) => {
    if (!(key in params)) {
      throw new Error(`Missing parameter '${key}' for dynamic test id`);
    }
    return String(params[key]);
  });

export type ExecutionFilterKey =
  | "all"
  | "completed"
  | "failed"
  | "running"
  | "cancelled";

export type ViewportPresetKey = "desktop" | "mobile" | "custom";

export const dynamicTestIds = {
  promptExample: (index: number) =>
    formatDynamicTestId("ai-example-prompt-${index}", { index }),
  executionFilter: (filter: ExecutionFilterKey) =>
    formatDynamicTestId("execution-filter-${filter}", { filter }),
  nodePaletteCard: (type: string) =>
    formatDynamicTestId("node-palette-${type}-card", { type }),
  nodePaletteFavoriteButton: (type: string) =>
    formatDynamicTestId("node-palette-${type}-favorite-button", { type }),
  nodePaletteCategory: (category: string) =>
    formatDynamicTestId("node-palette-category-${category}", { category }),
  nodePaletteCategoryToggle: (category: string) =>
    formatDynamicTestId("node-palette-category-${category}-toggle", {
      category,
    }),
  versionHistoryItem: (versionLabel: string | number) =>
    formatDynamicTestId("version-history-item-${version}", {
      version: versionLabel,
    }),
  versionRestoreButton: (versionLabel: string | number) =>
    formatDynamicTestId("version-restore-button-${version}", {
      version: versionLabel,
    }),
  viewportPresetButton: (preset: ViewportPresetKey) =>
    formatDynamicTestId("viewport-dialog-preset-${preset}-button", { preset }),
  uploadNodePathCount: (nodeId: string | number) =>
    formatDynamicTestId("upload-node-${id}-path-count", { id: nodeId }),
} as const;

const mapDynamicSelectorDefinitions = <
  T extends Record<string, DynamicSelectorDefinition>,
>(definitions: T) => {
  const result: Record<string, DynamicSelectorDefinition & { selectorPattern: string }> = {};
  for (const [key, value] of Object.entries(definitions)) {
    const selectorPattern = value.selectorPattern ?? toSelector(value.pattern);
    result[key] = { ...value, selectorPattern };
  }
  return result as {
    [K in keyof T]: DynamicSelectorDefinition & { selectorPattern: string };
  };
};

export const dynamicSelectors = mapDynamicSelectorDefinitions({
  promptExample: {
    description: "AI modal example prompts rendered with ai-example-prompt-${index}",
    pattern: "ai-example-prompt-${index}",
    params: ["index"] as const,
    valueTypes: { index: "number" },
  },
  executionFilter: {
    description: "Execution history filter buttons (all/completed/failed/etc.)",
    pattern: "execution-filter-${filter}",
    params: ["filter"] as const,
    allowedValues: {
      filter: ["all", "completed", "failed", "running", "cancelled"] as const,
    },
  },
  nodePaletteCard: {
    description: "Node palette card for a workflow node type",
    pattern: "node-palette-${type}-card",
    params: ["type"] as const,
  },
  nodePaletteFavoriteButton: {
    description: "Favorite toggle button for a workflow node card",
    pattern: "node-palette-${type}-favorite-button",
    params: ["type"] as const,
  },
  nodePaletteCategory: {
    description: "Category container in the node palette sidebar",
    pattern: "node-palette-category-${category}",
    params: ["category"] as const,
  },
  nodePaletteCategoryToggle: {
    description: "Category toggle button in the node palette sidebar",
    pattern: "node-palette-category-${category}-toggle",
    params: ["category"] as const,
  },
  versionHistoryItem: {
    description: "Version history row in header modal",
    pattern: "version-history-item-${version}",
    params: ["version"] as const,
  },
  versionRestoreButton: {
    description: "Restore button for a version history entry",
    pattern: "version-restore-button-${version}",
    params: ["version"] as const,
  },
  projectCardByName: {
    description: "Dashboard/project list card filtered by data-project-name",
    pattern: "project-card-by-name-${name}",
    selectorPattern: '[data-testid="project-card"][data-project-name="${name}"]',
    params: ["name"] as const,
  },
  workflowCardByName: {
    description: "Workflow list card filtered by data-workflow-name",
    pattern: "workflow-card-by-name-${name}",
    selectorPattern: '[data-testid="workflow-card"][data-workflow-name="${name}"]',
    params: ["name"] as const,
  },
  viewportPresetButton: {
    description: "Viewport preset button inside the execution dimensions dialog",
    pattern: "viewport-dialog-preset-${preset}-button",
    params: ["preset"] as const,
    allowedValues: {
      preset: ["desktop", "mobile", "custom"] as const,
    },
  },
  uploadNodePathCount: {
    description: "Upload node selected file count chip",
    pattern: "upload-node-${id}-path-count",
    params: ["id"] as const,
  },
  nodePaletteCardList: {
    description: "Matches all node palette cards regardless of node type",
    pattern: "node-palette-card-list",
    selectorPattern: '[data-testid^="node-palette-"][data-testid$="-card"]',
    params: [] as const,
  },
  nodePaletteVisibleCardList: {
    description: "Matches visible node palette cards (after filters)",
    pattern: "node-palette-card-visible",
    selectorPattern:
      '[data-testid^="node-palette-"][data-testid$="-card"]:not([style*="display: none"])',
    params: [] as const,
  },
});
