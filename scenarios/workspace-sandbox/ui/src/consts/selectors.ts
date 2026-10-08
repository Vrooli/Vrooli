import { librarySelectors } from "./selectors.library";
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";

const literalSelectors = {
  // Main app container
  app: "workspace-sandbox-app",

  // Header
  header: {
    root: "status-header",
    healthIndicator: "health-indicator",
    statsPanel: "stats-panel",
    createButton: "create-sandbox-button",
    refreshButton: "refresh-button",
  },

  // Sandbox list
  sandboxList: "sandbox-list",
  sandboxItem: "sandbox-item",
  emptyState: "empty-state",

  // Detail panel
  detailPanel: "detail-panel",
  detailEmpty: "detail-empty-state",

  // Diff viewer
  diffViewer: {
    root: "diff-viewer",
    fileList: "diff-file-list",
    fileItem: "diff-file-item",
    content: "diff-content",
    hunk: "diff-hunk",
    line: "diff-line",
    stats: "diff-stats",
    loading: "diff-loading",
    error: "diff-error",
    empty: "diff-empty",
  },

  // Actions
  actions: {
    approve: "approve-button",
    reject: "reject-button",
    stop: "stop-button",
    delete: "delete-button",
    confirmApprove: "confirm-approve",
    confirmReject: "confirm-reject",
    cancelAction: "cancel-action",
  },

  // Create dialog
  createDialog: {
    root: "create-sandbox-dialog",
    scopePathInput: "scope-path-input",
    projectRootInput: "project-root-input",
    ownerInput: "owner-input",
    ownerTypeSelect: "owner-type-select",
    submitButton: "submit-create",
    cancelButton: "cancel-create",
  },

  // Launch agent dialog
  launchDialog: {
    root: "launch-agent-dialog",
    commandInput: "launch-command-input",
    modeSelector: "launch-mode-selector",
    isolationSelector: "launch-isolation-selector",
    submitButton: "launch-submit",
    cancelButton: "launch-cancel",
  },

  // Mobile
  mobileNav: "mobile-nav",
  mobileHeader: "mobile-header",
  mobileHeaderMore: "mobile-header-more",
  bottomSheet: "bottom-sheet",

  // Error/loading states
  loading: "loading-spinner",
  errorToast: "error-toast",
} satisfies LiteralSelectorTree;

const dynamicSelectorDefinitions = {
  // Sandbox group by status
  sandboxGroup: defineDynamicSelector({
    description: "Sandbox group filtered by status",
    testIdPattern: "sandbox-group-${status}",
    params: {
      status: {
        type: "enum",
        values: ["creating", "active", "stopped", "checkpointing", "checkpointed", "approved", "rejected", "deleted", "error"],
      },
    },
  }),

  // Sandbox by ID
  sandboxById: defineDynamicSelector({
    description: "Sandbox item by ID",
    selectorPattern: '[data-testid="sandbox-item"][data-sandbox-id="${id}"]',
    params: { id: { type: "string" } },
  }),

  // Diff file by path
  diffFileByPath: defineDynamicSelector({
    description: "Diff file item by path",
    selectorPattern: '[data-testid="diff-file-item"][data-file-path="${path}"]',
    params: { path: { type: "string" } },
  }),

  // Mobile nav tab by panel
  mobileNavTab: defineDynamicSelector({
    description: "Mobile navigation tab by panel name",
    testIdPattern: "mobile-nav-${panel}",
    params: {
      panel: {
        type: "enum",
        values: ["sandboxes", "details", "changes"],
      },
    },
  }),

  // Diff hunk by index
  diffHunkByIndex: defineDynamicSelector({
    description: "Diff hunk by index",
    testIdPattern: "diff-hunk-${index}",
    params: { index: { type: "number" } },
  }),
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;

// Simple flat export for components to use directly
export const SELECTORS = {
  // Main app
  app: "workspace-sandbox-app",

  // Header
  statusHeader: "status-header",
  healthIndicator: "health-indicator",
  statsPanel: "stats-panel",
  createButton: "create-sandbox-button",
  refreshButton: "refresh-button",

  // Sandbox list
  sandboxList: "sandbox-list",
  sandboxItem: "sandbox-item",
  sandboxGroup: (status: string) => `sandbox-group-${status}`,
  emptyState: "empty-state",

  // Detail panel
  detailPanel: "detail-panel",
  detailEmpty: "detail-empty-state",
  detailsCollapseToggle: "details-collapse-toggle",

  // Diff viewer
  diffViewer: "diff-viewer",
  diffFileList: "diff-file-list",
  diffFileItem: "diff-file-item",
  diffContent: "diff-content",
  diffHunk: (index: number) => `diff-hunk-${index}`,
  diffLine: "diff-line",
  diffStats: "diff-stats",
  diffLoading: "diff-loading",
  diffError: "diff-error",
  diffEmpty: "diff-empty",

  // Actions
  approveButton: "approve-button",
  rejectButton: "reject-button",
  stopButton: "stop-button",
  startButton: "start-button",
  deleteButton: "delete-button",
  confirmApprove: "confirm-approve",
  confirmReject: "confirm-reject",
  cancelAction: "cancel-action",

  // Selection mode
  selectionModeToggle: "selection-mode-toggle",
  selectAllButton: "select-all-button",
  approveSelectedButton: "approve-selected-button",
  fileCheckbox: (fileId: string) => `file-checkbox-${fileId}`,
  hunkCheckbox: (fileId: string, hunkIndex: number) => `hunk-checkbox-${fileId}-${hunkIndex}`,

  // Create dialog
  createDialog: "create-sandbox-dialog",
  nameInput: "sandbox-name-input",
  scopePathInput: "scope-path-input",
  projectRootInput: "project-root-input",
  ownerInput: "owner-input",
  ownerTypeSelect: "owner-type-select",
  submitCreate: "submit-create",
  cancelCreate: "cancel-create",

  // Launch agent dialog
  launchDialog: "launch-agent-dialog",
  launchCommandInput: "launch-command-input",
  launchModeSelector: "launch-mode-selector",
  launchIsolationSelector: "launch-isolation-selector",
  launchSubmit: "launch-submit",
  launchCancel: "launch-cancel",
  launchAgentButton: "launch-agent-button",

  // Mobile navigation
  mobileNav: "mobile-nav",
  mobileNavTab: (panel: string) => `mobile-nav-${panel}`,
  mobileHeader: "mobile-header",
  mobileHeaderMore: "mobile-header-more",

  // Bottom sheet
  bottomSheet: "bottom-sheet",

  // Error/loading
  loading: "loading-spinner",
  errorToast: "error-toast",
} as const;
