/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. Selectors are defined as typed constant objects
 * to ensure TypeScript can statically verify all accesses.
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

// =============================================================================
// Dynamic Selector Types
// =============================================================================

type ParamType = "string" | "number" | "enum";

type ParamDefinition =
  | { readonly type: "string" }
  | { readonly type: "number" }
  | { readonly type: "enum"; readonly values: readonly (string | number)[] };

type ParamSchema = Readonly<Record<string, ParamDefinition>>;

interface DynamicSelectorDefinition<P extends ParamSchema | undefined = undefined> {
  readonly kind: "dynamic-selector";
  readonly description: string;
  readonly params?: P;
  readonly testIdPattern?: string;
  readonly selectorPattern?: string;
}

// =============================================================================
// Literal Selectors - Static test IDs with deterministic types
// =============================================================================

/**
 * Literal (static) selectors organized by UI area.
 * Each value is a data-testid string.
 */
export const literalSelectors = {
  // Layout selectors
  layout: {
    main: "main-layout",
    header: "header",
    desktopTabs: "desktop-tabs",
    mobileNav: "mobile-nav",
    agentsToggle: "layout-agents-toggle",
    agentsDropdown: "layout-agents-dropdown",
  },
  // Error state selectors (shared across pages)
  error: {
    container: "error-state",
    icon: "error-icon",
    title: "error-title",
    message: "error-message",
    retryButton: "error-retry",
  },
  // Error boundary selectors (for runtime errors)
  errorBoundary: {
    container: "error-boundary",
    title: "error-boundary-title",
    message: "error-boundary-message",
    refreshButton: "error-boundary-refresh",
    showDetailsButton: "error-boundary-show-details",
    diagnosticsPanel: "error-boundary-diagnostics",
    copyButton: "error-boundary-copy",
    copyConfirmation: "error-boundary-copy-confirmation",
    errorName: "error-boundary-error-name",
    errorMessage: "error-boundary-error-message",
    componentStack: "error-boundary-component-stack",
    timestamp: "error-boundary-timestamp",
    userAgent: "error-boundary-user-agent",
    errorCategory: "error-boundary-error-category",
  },
  // Not Found page selectors
  notFound: {
    page: "not-found-page",
    title: "not-found-title",
    message: "not-found-message",
    homeButton: "not-found-home",
  },
  // Desktop tab selectors
  tabs: {
    backlog: "tab-backlog",
    scenarios: "tab-scenarios",
    execution: "tab-execution",
    prompts: "tab-prompts",
    settings: "tab-settings",
  },
  // Mobile tab selectors
  mobileTabs: {
    backlog: "mobile-tab-backlog",
    scenarios: "mobile-tab-scenarios",
    execution: "mobile-tab-execution",
    prompts: "mobile-tab-prompts",
    settings: "mobile-tab-settings",
  },
  // Backlog page selectors
  backlog: {
    page: "backlog-page",
    search: "backlog-search",
    filter: "backlog-filter",
    createButton: "create-backlog",
    createFirstButton: "create-first-backlog",
    empty: "backlog-empty",
    grid: "backlog-grid",
    noResults: "backlog-no-results",
    sortButton: "backlog-sort-button",
    batchToggle: "backlog-batch-toggle",
    summaryStats: "backlog-summary-stats",
    readyCount: "backlog-ready-count",
    cliHint: "backlog-cli-hint",
    showFinishedToggle: "backlog-show-finished-toggle",
    // Iteration 5 additions
    statusLegend: "backlog-status-legend",
    welcomeHint: "backlog-welcome-hint",
    kindTabs: "backlog-kind-tabs",
  },
  // Captures (unified feed) selectors
  captures: {
    quickInput: "captures-quick-input",
    quickInputSubmit: "captures-quick-input-submit",
    quickInputAttach: "captures-quick-input-attach",
    quickInputSend: "captures-quick-input-send",
    card: "capture-card",
    retryButton: "capture-retry-button",
    acceptAllButton: "capture-accept-all-button",
    dismissButton: "capture-dismiss-button",
    itemAcceptButton: "capture-item-accept-button",
    itemEditButton: "capture-item-edit-button",
    itemDismissButton: "capture-item-dismiss-button",
  },
  // Inline question stepper selectors
  questionStepper: {
    container: "question-stepper",
    prevButton: "question-stepper-prev",
    nextButton: "question-stepper-next",
    skipButton: "question-stepper-skip",
    progress: "question-stepper-progress",
    workshopOption: "question-stepper-workshop-option",
    reviewApprove: "question-stepper-review-approve",
    reviewFlag: "question-stepper-review-flag",
  },
  // Backlog details page selectors
  backlogDetails: {
    page: "backlog-details-page",
    header: "backlog-details-header",
    title: "backlog-details-title",
    description: "backlog-details-description",
    backButton: "backlog-details-back",
    editButton: "backlog-details-edit",
    deleteButton: "backlog-details-delete",
    queueButton: "backlog-details-queue",
    agentButton: "backlog-details-agent",
    deleteDialog: "backlog-delete-dialog",
    deleteConfirmButton: "backlog-delete-confirm",
    deleteCancelButton: "backlog-delete-cancel",
    fileTree: "backlog-details-file-tree",
    filePreview: "backlog-details-file-preview",
    fileUpload: "backlog-details-file-upload",
    uploadDropzone: "file-upload-dropzone",
    uploadList: "file-upload-list",
    clarifyPanel: "backlog-clarify-panel",
    clarifyNextMode: "backlog-clarify-next-mode",
    clarifyNextModeNone: "backlog-clarify-next-mode-none",
    clarifySubmit: "backlog-clarify-submit",
    suggestionsPanel: "backlog-suggestions-panel",
    suggestionsSubmit: "backlog-suggestions-submit",
    // Experience architecture additions (Phase 29)
    breadcrumb: "backlog-details-breadcrumb",
    statusSelect: "backlog-details-status-select",
    activityTimeline: "backlog-details-activity-timeline",
    activeRunBanner: "backlog-details-active-run-banner",
    tabRow: "backlog-details-tab-row",
    tabInfo: "backlog-details-tab-info",
    tabPrompt: "backlog-details-tab-prompt",
    tabFiles: "backlog-details-tab-files",
    tabOutput: "backlog-details-tab-output",
    tabActivity: "backlog-details-tab-activity",
    outputTab: "backlog-details-output-tab",
    activityTab: "backlog-details-activity-tab",
    promptPanel: "backlog-details-prompt-panel",
    initiativeChip: "backlog-details-initiative-chip",
  },
  // Backlog form dialog selectors
  backlogForm: {
    dialog: "backlog-form-dialog",
    titleInput: "backlog-form-title",
    nameInput: "backlog-form-name",
    descriptionInput: "backlog-form-description",
    statusSelect: "backlog-form-status",
    priorityInput: "backlog-form-priority",
    tagsInput: "backlog-form-tags",
    kindSelect: "backlog-form-kind",
    submitButton: "backlog-form-submit",
    cancelButton: "backlog-form-cancel",
    agentDialog: "backlog-agent-dialog",
    agentMode: "backlog-agent-mode",
    agentContext: "backlog-agent-context",
    agentSubmit: "backlog-agent-submit",
  },
  // Scenarios page selectors
  scenarios: {
    page: "scenarios-page",
    search: "scenarios-search",
    filter: "scenarios-filter",
    filterDropdown: "scenarios-filter-dropdown",
    statusFilter: "scenarios-status-filter",
    count: "scenarios-count",
    empty: "scenarios-empty",
    noResults: "scenarios-no-results",
    list: "scenarios-list",
    // Experience architecture additions (Phase 29)
    statusSummary: "scenarios-status-summary",
    runningCount: "scenarios-running-count",
    stoppedCount: "scenarios-stopped-count",
    errorCount: "scenarios-error-count",
  },
  // Scenario details page selectors
  // [REQ:REQ-P0-007] Selectors for scenario metadata management UI
  // [REQ:REQ-P0-008] Selectors for scenario deletion UI
  scenarioDetails: {
    page: "scenario-details-page",
    header: "scenario-details-header",
    title: "scenario-details-title",
    description: "scenario-details-description",
    backButton: "scenario-details-back",
    metadataSection: "scenario-details-metadata",
    actionsSection: "scenario-details-actions",
    startButton: "scenario-details-start",
    stopButton: "scenario-details-stop",
    restartButton: "scenario-details-restart",
    actionError: "scenario-details-action-error",
    greenfieldToggle: "scenario-greenfield-toggle",
    saveButton: "scenario-details-save",
    priority: "scenario-details-priority",
    tags: "scenario-details-tags",
    status: "scenario-details-status",
    deleteButton: "scenario-details-delete",
    deleteDialog: "scenario-delete-dialog",
    deleteConfirmButton: "scenario-delete-confirm",
    deleteCancelButton: "scenario-delete-cancel",
    archiveCheckbox: "scenario-delete-archive",
    // Experience architecture additions (Phase 29)
    breadcrumb: "scenario-details-breadcrumb",
    // Iteration 5 additions
    cliHint: "scenario-details-cli-hint",
  },
  // Settings page selectors
  settings: {
    page: "settings-page",
    themeSettings: "theme-settings",
    themeDark: "theme-dark",
    themeLight: "theme-light",
    themeSystem: "theme-system",
    executionDefaults: "execution-defaults",
    workshopSettings: "workshop-settings",
    agentSettings: "agent-settings",
    uiPreferences: "ui-preferences",
    reviewSettings: "review-settings",
    settingsTabs: "settings-tabs",
    tabGeneral: "settings-tab-general",
    tabExecution: "settings-tab-execution",
    tabWorkshop: "settings-tab-workshop",
    tabReview: "settings-tab-review",
    saveButton: "settings-save",
  },
  // Execution list page selectors
  execution: {
    page: "execution-page",
    tabs: "execution-tabs",
    search: "execution-search",
    filter: "execution-filter",
    empty: "execution-empty",
    noResults: "execution-no-results",
    activeSection: "execution-active-section",
    activeList: "execution-active-list",
    grid: "execution-grid",
    promptTrace: "execution-prompt-trace",
  },
  // Execution detail page selectors
  executionDetails: {
    page: "execution-details-page",
    tabRow: "execution-details-tab-row",
    tabOverview: "execution-details-tab-overview",
    tabChanges: "execution-details-tab-changes",
    tabReview: "execution-details-tab-review",
    tabPrompt: "execution-details-tab-prompt",
    overviewMetadata: "execution-details-overview-metadata",
    overviewActions: "execution-details-overview-actions",
    changesFileList: "execution-details-changes-file-list",
    changesEmpty: "execution-details-changes-empty",
    reviewSection: "execution-details-review-section",
    reviewEmpty: "execution-details-review-empty",
    promptTrace: "execution-details-prompt-trace",
    promptEmpty: "execution-details-prompt-empty",
    followUpButton: "execution-details-follow-up",
    retryButton: "execution-details-retry",
    cancelButton: "execution-details-cancel",
    runChecksButton: "execution-details-run-checks",
    viewRunButton: "execution-details-view-run",
  },
  // Run backlog modal selectors
  runBacklog: {
    dialog: "run-backlog-dialog",
    submitButton: "run-backlog-submit",
    blockingReasons: "run-backlog-blocking-reasons",
    readinessWarning: "run-backlog-readiness-warning",
    error: "run-backlog-error",
  },
  followUp: {
    dialog: "follow-up-dialog",
    typeFixup: "follow-up-type-fixup",
    typeFollowup: "follow-up-type-followup",
    typeCustom: "follow-up-type-custom",
    runModeContinue: "follow-up-run-mode-continue",
    runModeNew: "follow-up-run-mode-new",
    contextInput: "follow-up-context-input",
    submitButton: "follow-up-submit-button",
    error: "follow-up-error",
    reviewSummary: "follow-up-review-summary",
    runHealth: "follow-up-run-health",
  },
  // Shared review flow selectors
  review: {
    flow: "review-flow",
    statusHeader: "review-status-header",
    primaryAction: "review-primary-action",
    rerunAction: "review-rerun-action",
    stopAction: "review-stop-action",
    gatherEvidenceAction: "review-gather-evidence-action",
    scenarioChips: "review-scenario-chips",
    scenarioResultCards: "review-scenario-result-cards",
    scenarioResultCard: "review-scenario-result-card",
    launchSheet: "review-launch-sheet",
    launchSheetFullReview: "review-launch-sheet-full-review",
    launchSheetGatherEvidence: "review-launch-sheet-gather-evidence",
    activityTimelineLabel: "review-activity-timeline-label",
    phaseProgress: "review-phase-progress",
    evidenceContextSummary: "review-evidence-context-summary",
    followUpSheet: "review-follow-up-sheet",
  },
  // Initiative details page selectors
  initiativeDetails: {
    page: "initiative-details-page",
    title: "initiative-details-title",
    status: "initiative-details-status",
    description: "initiative-details-description",
    rollup: "initiative-details-rollup",
    itemsList: "initiative-details-items-list",
    itemsViewToggle: "initiative-items-view-toggle",
    itemsListView: "initiative-items-list-view",
    itemsGraphView: "initiative-items-graph-view",
    backLink: "initiative-details-back-link",
    tabRow: "initiative-details-tab-row",
    tabInfo: "initiative-details-tab-info",
    tabFiles: "initiative-details-tab-files",
  },
  prompts: {
    page: "prompts-page",
    tabs: "prompts-tabs",
    tabMap: "prompts-tab-map",
    tabViewer: "prompts-tab-viewer",
    mapPanel: "prompts-map-panel",
    viewerPanel: "prompts-viewer-panel",
    usageMatrix: "prompts-usage-matrix",
    bindingMap: "prompts-binding-map",
    skillsList: "prompts-skills-list",
    editor: "prompts-editor",
    contentInput: "prompts-content-input",
    saveDraft: "prompts-save-draft",
    publish: "prompts-publish",
    versions: "prompts-versions",
    preview: "prompts-preview",
  },
  // Command Post selectors
  commandPost: {
    overlay: "command-post-overlay",
    overlayHeader: "command-post-overlay-header",
    close: "command-post-close",
    summary: "command-post-summary",
    decisionStream: {
      container: "ds-container",
      header: "ds-header",
      backButton: "ds-back",
      counter: "ds-counter",
      contextToggle: "ds-context-toggle",
      contextPanel: "ds-context-panel",
      questionArea: "ds-question-area",
      navBar: "ds-nav-bar",
      navBack: "ds-nav-back",
      navSkip: "ds-nav-skip",
      navSnooze: "ds-nav-snooze",
      navNext: "ds-nav-next",
      progressBar: "ds-progress-bar",
      openItemLink: "ds-open-item",
      deleteButton: "ds-delete-question",
      deleteConfirm: "ds-delete-confirm",
    },
  },
  graphNavControls: {
    container: "graph-nav-controls",
    panUp: "graph-nav-pan-up",
    panDown: "graph-nav-pan-down",
    panLeft: "graph-nav-pan-left",
    panRight: "graph-nav-pan-right",
    zoomIn: "graph-nav-zoom-in",
    zoomOut: "graph-nav-zoom-out",
    fitView: "graph-nav-fit-view",
  },
  // Evidence renderer selectors
  evidence: {
    cliOutput: "evidence-cli-output",
    configDiff: "evidence-config-diff",
    workflowRecording: "evidence-workflow-recording",
    lightbox: "media-lightbox",
    truncatedLink: "evidence-truncated-link",
    reviewCheckbox: "evidence-review-checkbox",
    markAllReviewed: "evidence-mark-all-reviewed",
    agentAssessment: "evidence-agent-assessment",
  },
  // Evidence request panel selectors
  evidenceRequest: {
    panel: "evidence-request-panel",
    messageList: "evidence-request-messages",
    textInput: "evidence-request-input",
    sendButton: "evidence-request-send",
    dismissButton: "evidence-request-dismiss",
    staleWarning: "evidence-request-stale-warning",
    error: "evidence-request-error",
    targetContext: "evidence-request-target-context",
  },
} as const;

// =============================================================================
// Dynamic Selectors - Parameterized selectors for data-driven elements
// =============================================================================

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match, token: string) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const defineDynamicSelector = <P extends ParamSchema | undefined>(
  definition: Omit<DynamicSelectorDefinition<P>, "kind">,
): DynamicSelectorDefinition<P> => ({
  ...definition,
  kind: "dynamic-selector",
});

// Dynamic selector definitions (used for manifest generation)
export const dynamicSelectorDefinitions = {
  backlog: {
    cardByName: defineDynamicSelector({
      description: "Backlog card filtered by kind and name",
      testIdPattern: "backlog-card-${kind}-${name}",
      params: {
        kind: { type: "enum", values: ["idea", "research", "fix", "execute", "chore"] },
        name: { type: "string" },
      },
    }),
  },
  scenarios: {
    cardByName: defineDynamicSelector({
      description: "Scenario card filtered by name",
      testIdPattern: "scenario-card-${name}",
      params: { name: { type: "string" } },
    }),
    actionStart: defineDynamicSelector({
      description: "Scenario start action button",
      testIdPattern: "scenario-action-start-${name}",
      params: { name: { type: "string" } },
    }),
    actionStop: defineDynamicSelector({
      description: "Scenario stop action button",
      testIdPattern: "scenario-action-stop-${name}",
      params: { name: { type: "string" } },
    }),
    actionRestart: defineDynamicSelector({
      description: "Scenario restart action button",
      testIdPattern: "scenario-action-restart-${name}",
      params: { name: { type: "string" } },
    }),
  },
} as const;

// =============================================================================
// Dynamic Selector Functions
// =============================================================================

/**
 * Dynamic selectors - functions that generate test IDs from parameters
 */
export const dynamicSelectors = {
  backlog: {
    cardByName: (params: { kind: string; name: string }) =>
      formatTemplate("backlog-card-${kind}-${name}", params, "backlog.cardByName"),
  },
  scenarios: {
    cardByName: (params: { name: string }) =>
      formatTemplate("scenario-card-${name}", params, "scenarios.cardByName"),
    actionStart: (params: { name: string }) =>
      formatTemplate("scenario-action-start-${name}", params, "scenarios.actionStart"),
    actionStop: (params: { name: string }) =>
      formatTemplate("scenario-action-stop-${name}", params, "scenarios.actionStop"),
    actionRestart: (params: { name: string }) =>
      formatTemplate("scenario-action-restart-${name}", params, "scenarios.actionRestart"),
  },
} as const;

// =============================================================================
// Combined Selectors Export
// =============================================================================

/**
 * Combined selector registry - literal selectors merged with dynamic functions.
 * This is the primary export for UI components.
 *
 * Usage:
 * - Literal: selectors.backlog.page
 * - Dynamic: selectors.backlog.cardByName({ kind: "idea", name: "my-idea" })
 */
export const selectors = {
  ...literalSelectors,
  backlog: {
    ...literalSelectors.backlog,
    ...dynamicSelectors.backlog,
  },
  scenarios: {
    ...literalSelectors.scenarios,
    ...dynamicSelectors.scenarios,
  },
} as const;

export type Selectors = typeof selectors;

// =============================================================================
// Manifest Generation (for workflow tools)
// =============================================================================

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

type LiteralSelectorTree = { readonly [key: string]: string | LiteralSelectorTree };

const flattenLiteralSelectors = (
  tree: LiteralSelectorTree,
  prefix: string[] = [],
  target: Record<string, { testId: string; selector: string }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const nextPath = [...prefix, key];
    if (typeof value === "string") {
      const manifestKey = nextPath.join(".");
      target[manifestKey] = {
        testId: value,
        selector: toDataTestIdSelector(value),
      };
      continue;
    }
    flattenLiteralSelectors(value as LiteralSelectorTree, nextPath, target);
  }
  return target;
};

const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<ParamSchema | undefined> =>
  Boolean(value && typeof value === "object" && (value as DynamicSelectorDefinition<ParamSchema | undefined>).kind === "dynamic-selector");

type DynamicSelectorBranch = {
  readonly [key: string]: DynamicSelectorBranch | DynamicSelectorDefinition<ParamSchema | undefined>;
};

const flattenDynamicSelectors = (
  tree: DynamicSelectorBranch,
  prefix: string[] = [],
  target: Record<string, {
    description: string;
    selectorPattern: string;
    testIdPattern?: string;
    params: Array<{ name: string; type: ParamType; values?: readonly (string | number)[] }>;
  }> = {},
) => {
  for (const [key, value] of Object.entries(tree)) {
    const nextPath = [...prefix, key];
    if (isDynamicDefinition(value)) {
      const manifestKey = nextPath.join(".");
      const paramEntries = Object.entries(value.params ?? {}) as Array<[string, ParamDefinition]>;
      target[manifestKey] = {
        description: value.description,
        selectorPattern:
          value.selectorPattern ?? (value.testIdPattern ? toDataTestIdSelector(value.testIdPattern) : ""),
        testIdPattern: value.testIdPattern,
        params: paramEntries.map(([name, config]) => ({
          name,
          type: config.type,
          values: config.type === "enum" ? config.values : undefined,
        })),
      };
      continue;
    }
    flattenDynamicSelectors(value as DynamicSelectorBranch, nextPath, target);
  }
  return target;
};

export const selectorsManifest = {
  selectors: flattenLiteralSelectors(literalSelectors),
  dynamicSelectors: flattenDynamicSelectors(dynamicSelectorDefinitions),
};
