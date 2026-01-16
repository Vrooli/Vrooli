/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. We deliberately model selectors as two
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
  readonly [key: string]: DynamicSelectorBranch | DynamicSelectorDefinition<ParamSchema | undefined>;
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
} & (D extends DynamicSelectorTree ? DynamicBranchResult<D> : Record<string, never>);

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match, token) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<ParamSchema | undefined> =>
  Boolean(value && typeof value === "object" && (value as DynamicSelectorDefinition<ParamSchema | undefined>).kind === "dynamic-selector");

const normalizeParams = (
  definition: DynamicSelectorDefinition<ParamSchema | undefined>,
  raw: Record<string, string | number>,
  path: string,
) => {
  const schema: ParamSchema = definition.params ?? {};
  const normalized: Record<string, string | number> = {};

  for (const key of Object.keys(schema)) {
    if (!(key in raw)) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    const definitionEntry = schema[key];
    const value = raw[key];
    // These should never be undefined since we iterate Object.keys and check `key in raw`
    if (!definitionEntry || value === undefined) {
      throw new Error(`Selector '${path}' has invalid parameter '${key}'`);
    }
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
  definition: DynamicSelectorDefinition<ParamSchema | undefined>,
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

const literalSelectors: LiteralSelectorTree = {
  // Main layout
  app: {
    container: 'inbox-container',
    mobileMenuButton: 'mobile-menu-button',
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
    confirmDeleteButton: 'confirm-delete-button',
  },
  // Message input
  messageInput: {
    container: 'message-input-container',
    input: 'message-input',
    sendButton: 'send-message-button',
  },
  // Empty state
  emptyState: {
    container: 'empty-state',
    newChatButton: 'empty-state-new-chat',
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
