/**
 * Browser Automation Studio selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Browser Automation Studio workflows. We deliberately model selectors as two
 * declarative maps (one literal, one dynamic) and rely on a small helper to
 * produce the typed `selectors` export plus the manifest consumed by workflow
 * linting. Do not hand-roll selector helpers or change this structure—update the
 * maps below so UI code, automation flows, and the manifest builder all stay in
 * sync across every scenario.
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

type LiteralSelectorTree = { readonly [key: string]: string | LiteralSelectorTree };
type LiteralNode = string | LiteralSelectorTree;

type ParamType = "string" | "number" | "enum";

type ParamDefinition =
  | { readonly type: "string" }
  | { readonly type: "number" }
  | { readonly type: "enum"; readonly values: readonly (string | number)[] };

type ParamSchema = Readonly<Record<string, ParamDefinition>>;

type ParamValueType<T extends ParamDefinition> = T extends { type: "number" }
  ? number
  : T extends { type: "enum"; values: readonly (infer V)[] }
  ? V
  : string;

type ParamValues<P extends ParamSchema | undefined> = P extends ParamSchema
  ? { [K in keyof P]: ParamValueType<P[K]> }
  : Record<string, never>;

interface DynamicSelectorDefinition<P extends ParamSchema | undefined = undefined> {
  readonly kind: "dynamic-selector";
  readonly description: string;
  readonly params?: P;
  readonly testIdPattern?: string;
  readonly selectorPattern?: string;
}

type DynamicSelectorBranch = {
  readonly [key: string]: DynamicSelectorBranch | DynamicSelectorDefinition<any>;
};

type DynamicSelectorTree = DynamicSelectorBranch;

type DynamicSelectorFn<P extends ParamSchema | undefined> = keyof ParamValues<P> extends never
  ? () => string
  : (params: ParamValues<P>) => string;

type DynamicBranchResult<D extends DynamicSelectorTree> = {
  [K in keyof D]: D[K] extends DynamicSelectorDefinition<infer P>
  ? DynamicSelectorFn<P>
  : D[K] extends DynamicSelectorTree
  ? DynamicBranchResult<D[K]>
  : never;
};

type SelectorTreeResult<
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
> = {
  [K in keyof L]: L[K] extends string
    ? string
    : SelectorTreeResult<
        Extract<L[K], LiteralSelectorTree>,
        K extends keyof D ? Extract<D[K], DynamicSelectorTree> : DynamicSelectorTree
      >;
} & (D extends DynamicSelectorTree ? DynamicBranchResult<D> : {});

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match, token) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<any> =>
  Boolean(value && typeof value === "object" && (value as DynamicSelectorDefinition).kind === "dynamic-selector");

const normalizeParams = (
  definition: DynamicSelectorDefinition<any>,
  raw: Record<string, string | number>,
  path: string,
) => {
  const schema = definition.params ?? ({} as ParamSchema);
  const normalized: Record<string, string | number> = {};

  for (const key of Object.keys(schema)) {
    if (!(key in raw)) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    const definitionEntry = schema[key];
    const value = raw[key];
    if (definitionEntry.type === "number") {
      if (typeof value !== "number") {
        throw new Error(`Selector '${path}' parameter '${key}' must be numeric`);
      }
      normalized[key] = value;
      continue;
    }
    if (definitionEntry.type === "enum") {
      if (!definitionEntry.values.includes(value)) {
        throw new Error(
          `Selector '${path}' parameter '${key}' must be one of: ${definitionEntry.values.join(", ")}`,
        );
      }
      normalized[key] = value;
      continue;
    }
    normalized[key] = value;
  }

  const extras = Object.keys(raw).filter((key) => !(key in schema));
  if (extras.length > 0) {
    throw new Error(`Selector '${path}' received unknown parameter(s): ${extras.join(", ")}`);
  }

  return normalized;
};

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
    flattenLiteralSelectors(value, nextPath, target);
  }
  return target;
};

const flattenDynamicSelectors = (
  tree: DynamicSelectorTree,
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
    flattenDynamicSelectors(value, nextPath, target);
  }
  return target;
};

const mergeLiteralAndDynamicNodes = (
  literalNode: LiteralSelectorTree | undefined,
  dynamicNode: DynamicSelectorTree | undefined,
  path: string[] = [],
): Record<string, unknown> => {
  const merged: Record<string, unknown> = {};
  const keys = new Set([
    ...Object.keys(literalNode ?? {}),
    ...Object.keys(dynamicNode ?? {}),
  ]);

  keys.forEach((key) => {
    const literalValue: LiteralNode | undefined = literalNode?.[key];
    const dynamicValue = dynamicNode?.[key];
    const nextPath = [...path, key];

    if (typeof literalValue === "string") {
      merged[key] = literalValue;
      return;
    }

    if (literalValue && typeof literalValue === "object") {
      merged[key] = mergeLiteralAndDynamicNodes(
        literalValue as LiteralSelectorTree,
        isDynamicDefinition(dynamicValue) ? undefined : (dynamicValue as DynamicSelectorTree | undefined),
        nextPath,
      );
      return;
    }

    if (dynamicValue) {
      if (isDynamicDefinition(dynamicValue)) {
        merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
        return;
      }
      merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicValue as DynamicSelectorTree, nextPath);
    }
  });

  return merged;
};

const createDynamicSelectorFn = (
  definition: DynamicSelectorDefinition<any>,
  path: string,
) => {
  return (params?: Record<string, string | number>) => {
    const normalized = normalizeParams(definition, params ?? {}, path);
    const template = definition.testIdPattern ?? definition.selectorPattern;
    if (!template) {
      throw new Error(`Selector '${path}' is missing both testIdPattern and selectorPattern`);
    }
    return formatTemplate(template, normalized, path);
  };
};

const defineDynamicSelector = <P extends ParamSchema | undefined>(
  definition: Omit<DynamicSelectorDefinition<P>, "kind">,
): DynamicSelectorDefinition<P> => ({
  ...definition,
  kind: "dynamic-selector",
});

const createSelectorRegistry = <
  L extends LiteralSelectorTree,
  D extends DynamicSelectorTree,
>(literalTree: L, dynamicTree: D) => {
  const selectors = mergeLiteralAndDynamicNodes(literalTree, dynamicTree) as SelectorTreeResult<L, D>;
  const manifest = {
    selectors: flattenLiteralSelectors(literalTree),
    dynamicSelectors: flattenDynamicSelectors(dynamicTree),
  };
  return { selectors, manifest };
};

const literalSelectors = {
  app: {
    shell: {
      ready: "app-ready",
    },
    background: "background",
  },
  ai: {
    modal: {
      root: "ai-prompt-modal",
      generateButton: "ai-generate-button",
      promptInput: "ai-prompt-input",
      workflowNameInput: "ai-workflow-name-input",
      switchToManualButton: "switch-to-manual-button",
      examplePrompts: {
        first: "ai-example-prompt-0",
      },
    },
  },
  breadcrumbs: {
    project: "breadcrumb-project",
  },
  dashboard: {
    newProjectButton: "dashboard-new-project-button",
  },
  dialogs: {
    base: {
      closeButton: "dialog-close-button",
      content: "dialog-content",
    },
    responsive: {
      content: "responsive-dialog-content",
      overlay: "responsive-dialog-overlay",
    },
    project: {
      root: "project-modal",
      form: "project-modal-form",
      nameInput: "project-modal-name-input",
      nameError: "project-modal-name-error",
      descriptionInput: "project-modal-description-input",
      cancelButton: "project-modal-cancel",
      submitButton: "project-modal-submit",
    },
  },
  forms: {
    error: "form-error",
  },
  executions: {
    tabs: {
      executions: "execution-tab-executions",
      logs: "execution-tab-logs",
      replay: "execution-tab-replay",
      screenshots: "execution-tab-screenshots",
    },
    filters: {
      all: "execution-filter-all",
      completed: "execution-filter-completed",
      failed: "execution-filter-failed",
      running: "execution-filter-running",
      cancelled: "execution-filter-cancelled",
    },
    list: {
      root: "execution-history",
      list: "execution-history-list",
      refreshButton: "execution-history-refresh",
      item: "execution-item",
      card: "execution-card",
    },
    mock: {
      history: "execution-history-mock",
      viewer: "execution-viewer-mock",
    },
    viewer: {
      root: "execution-viewer",
      status: "execution-status",
      statusCancelled: "execution-status-cancelled",
      statusCompleted: "execution-status-completed",
      statusFailed: "execution-status-failed",
      stopButton: "execution-stop-button",
      logs: "execution-logs",
      screenshots: "execution-screenshots",
      screenshot: "execution-screenshot",
    },
    navigation: {
      executionsTab: "executions-tab",
    },
    actions: {
      viewButton: "execution-view-button",
      exportReplayButton: "export-replay-button",
      exportConfirmButton: "export-confirm-button",
    },
    export: {
      inProgress: "export-in-progress",
    },
    logEntry: "log-entry",
  },
  header: {
    root: "header",
    buttons: {
      debug: "header-debug-button",
      execute: "header-execute-button",
      info: "header-info-button",
      infoClose: "header-info-close-button",
      versionHistory: "header-version-history-button",
      backToProject: "header-back-to-project-button",
      backToDashboard: "header-back-to-dashboard-button",
      saveRetry: "header-save-retry-button",
      saveErrorDetails: "header-save-error-details-button",
      saveErrorDismiss: "header-save-error-dismiss-button",
      versionConflictDetails: "header-version-conflict-details-button",
      versionConflictRefresh: "header-version-conflict-refresh-button",
      versionConflictReload: "header-version-conflict-reload-button",
      versionConflictForceSave: "header-version-conflict-force-save-button",
    },
    title: {
      input: "header-title-input",
      saveButton: "header-title-save-button",
      cancelButton: "header-title-cancel-button",
      editButton: "header-title-edit-button",
      text: "header-workflow-title",
    },
    info: {
      popover: "header-info-popover",
    },
    saveStatus: {
      indicator: "header-save-status-indicator",
    },
    saveError: {
      dialog: "header-save-error-dialog",
      dialogCloseButton: "header-save-error-dialog-close-button",
    },
    versionHistory: {
      dialog: "header-version-history-dialog",
      closeButton: "header-version-history-close-button",
    },
  },
  heartbeat: {
    indicator: "heartbeat-indicator",
    status: "heartbeat-status",
    lagWarning: "heartbeat-lag-warning",
  },
  nodePalette: {
    container: "node-palette-container",
    searchInput: "node-palette-search-input",
    quickAccess: "node-palette-quick-access",
    quickAccessToggle: "node-palette-quick-access-toggle",
    favorites: {
      section: "node-palette-favorites-section",
    },
    recents: {
      section: "node-palette-recents-section",
      clearButton: "node-palette-clear-recents-button",
    },
    cards: {
      assert: "node-palette-assert-card",
      click: "node-palette-click-card",
      dragDrop: "node-palette-dragDrop-card",
      navigate: "node-palette-navigate-card",
      screenshot: "node-palette-screenshot-card",
      setVariable: "node-palette-setVariable-card",
      useVariable: "node-palette-useVariable-card",
      wait: "node-palette-wait-card",
    },
    categories: {
      navigationToggle: "node-palette-category-navigation-toggle",
    },
    noResults: "node-palette-no-results",
    favoriteButtons: {
      navigate: "node-palette-navigate-favorite-button",
    },
  },
  nodeProperties: {
    urlInput: "node-property-url-input",
    urlModeButton: "url-mode-button",
  },
  projects: {
    grid: "projects-grid",
    card: "project-card",
    cardTitle: "project-card-title",
    editButton: "project-edit-button",
    search: {
      input: "projects-search-input",
      clearButton: "projects-search-clear",
    },
    tabs: {
      executions: "project-tab-executions",
    },
  },
  replay: {
    player: "replay-player",
    screenshot: "replay-screenshot",
    timeline: "replay-timeline",
    viewer: "replay-viewer",
  },
  timeline: {
    frame: "timeline-frame",
  },
  toolbar: {
    root: "workflow-toolbar",
    zoomInButton: "toolbar-zoom-in-button",
    zoomOutButton: "toolbar-zoom-out-button",
    fitViewButton: "toolbar-fit-view-button",
    configureViewportButton: "toolbar-configure-viewport-button",
    configureViewportMenuItem: "toolbar-configure-viewport-menu-item",
    layoutMenu: "toolbar-layout-menu",
    layoutMenuButton: "toolbar-layout-menu-button",
    lockButton: "toolbar-lock-button",
    unlockButton: "toolbar-unlock-button",
    redoButton: "toolbar-redo-button",
    undoButton: "toolbar-undo-button",
    duplicateButton: "toolbar-duplicate-button",
    deleteButton: "toolbar-delete-button",
    selectionMenu: "toolbar-selection-menu",
    selectionMenuButton: "toolbar-selection-menu-button",
    selectionMenuDuplicate: "toolbar-duplicate-menu-item",
    selectionMenuDelete: "toolbar-delete-menu-item",
    toggleLockMenuItem: "toolbar-toggle-lock-menu-item",
    fitViewMenuItem: "toolbar-fit-view-menu-item",
  },
  variables: {
    suggestions: "variable-suggestions",
    suggestionsEmpty: "variable-suggestions-empty",
  },
  viewport: {
    dialog: {
      root: "workflow-builder-viewport-dialog",
      cancelButton: "viewport-dialog-cancel-button",
      error: "viewport-dialog-error",
      heightInput: "viewport-dialog-height-input",
      saveButton: "viewport-dialog-save-button",
      title: "viewport-dialog-title",
      widthInput: "viewport-dialog-width-input",
    },
  },
  workflowBuilder: {
    canvas: {
      root: "workflow-builder-canvas",
      reactFlow: "react-flow-canvas",
      minimap: "minimap",
    },
    code: {
      view: "workflow-builder-code-view",
      toolbar: "workflow-builder-code-toolbar",
      editor: "workflow-builder-code-editor",
      editorContainer: "workflow-builder-code-editor-container",
      error: "workflow-builder-code-error",
      lineCount: "workflow-builder-code-line-count",
      modeButton: "workflow-builder-code-mode-button",
      resetButton: "workflow-builder-code-reset-button",
      applyButton: "workflow-builder-code-apply-button",
      validation: "workflow-builder-code-validation",
    },
    viewModeToggle: "workflow-builder-view-mode-toggle",
    visualModeButton: "workflow-builder-visual-mode-button",
    executeButton: "workflow-execute-button",
    search: {
      input: "workflow-search-input",
      clearButton: "workflow-search-clear",
    },
    monacoEditor: "monaco-editor",
    formError: "form-error",
  },
  workflows: {
    tab: "workflows-tab",
    card: "workflow-card",
    newButton: "new-workflow-button",
    newButtonFab: "new-workflow-button-fab",
  },
} as const;

const dynamicSelectorDefinitions = {
  ai: {
    modal: {
      promptExample: defineDynamicSelector({
        description: "AI modal example prompts rendered as ai-example-prompt-${index}",
        testIdPattern: "ai-example-prompt-${index}",
        params: { index: { type: "number" } },
      }),
    },
  },
  executions: {
    filters: {
      filter: defineDynamicSelector({
        description: "Execution history filter buttons",
        testIdPattern: "execution-filter-${filter}",
        params: {
          filter: { type: "enum", values: ["all", "completed", "failed", "running", "cancelled"] },
        },
      }),
    },
  },
  nodePalette: {
    card: defineDynamicSelector({
      description: "Node palette card for a workflow node type",
      testIdPattern: "node-palette-${type}-card",
      params: { type: { type: "string" } },
    }),
    favoriteButton: defineDynamicSelector({
      description: "Favorite toggle button for a node palette card",
      testIdPattern: "node-palette-${type}-favorite-button",
      params: { type: { type: "string" } },
    }),
    category: defineDynamicSelector({
      description: "Node palette category section",
      testIdPattern: "node-palette-category-${category}",
      params: { category: { type: "string" } },
    }),
    categoryToggle: defineDynamicSelector({
      description: "Toggle button for a node palette category",
      testIdPattern: "node-palette-category-${category}-toggle",
      params: { category: { type: "string" } },
    }),
    cardList: defineDynamicSelector({
      description: "Matches all node palette cards",
      selectorPattern: '[data-testid^="node-palette-"][data-testid$="-card"]',
    }),
    visibleCardList: defineDynamicSelector({
      description: "Visible node palette cards once filters are applied",
      selectorPattern:
        '[data-testid^="node-palette-"][data-testid$="-card"]:not([style*="display: none"])',
    }),
  },
  header: {
    versionHistory: {
      item: defineDynamicSelector({
        description: "Version history entries in the header dialog",
        testIdPattern: "version-history-item-${version}",
        params: { version: { type: "string" } },
      }),
      restoreButton: defineDynamicSelector({
        description: "Restore button inside version history rows",
        testIdPattern: "version-restore-button-${version}",
        params: { version: { type: "string" } },
      }),
    },
  },
  viewport: {
    dialog: {
      presetButton: defineDynamicSelector({
        description: "Viewport dialog preset buttons",
        testIdPattern: "viewport-dialog-preset-${preset}-button",
        params: {
          preset: { type: "enum", values: ["desktop", "mobile", "custom"] },
        },
      }),
    },
  },
  workflowBuilder: {
    reactFlowReady: defineDynamicSelector({
      description: "React Flow canvas when fully initialized and ready for interaction",
      selectorPattern: '[data-testid="react-flow-canvas"][data-builder-ready="true"]',
    }),
    nodes: {
      upload: {
        pathCount: defineDynamicSelector({
          description: "Upload node selected path count chip",
          testIdPattern: "upload-node-${id}-path-count",
          params: { id: { type: "string" } },
        }),
      },
    },
  },
  projects: {
    cardByName: defineDynamicSelector({
      description: "Dashboard project card filtered by name",
      selectorPattern: '[data-testid="project-card"][data-project-name="${name}"]',
      params: { name: { type: "string" } },
    }),
    cardById: defineDynamicSelector({
      description: "Dashboard project card filtered by id",
      selectorPattern: '[data-testid="project-card"][data-project-id="${id}"]',
      params: { id: { type: "string" } },
    }),
  },
  workflows: {
    cardByName: defineDynamicSelector({
      description: "Workflow list card filtered by name",
      selectorPattern: '[data-testid="workflow-card"][data-workflow-name="${name}"]',
      params: { name: { type: "string" } },
    }),
    cardById: defineDynamicSelector({
      description: "Workflow list card filtered by id",
      selectorPattern: '[data-testid="workflow-card"][data-workflow-id="${id}"]',
      params: { id: { type: "string" } },
    }),
  },
} as const satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
