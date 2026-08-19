// DOC: docs/internal/SEAMS.md
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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- type erasure for generic selector definitions
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
// eslint-disable-next-line @typescript-eslint/no-empty-object-type -- intentional empty intersection for conditional type
} & (D extends DynamicSelectorTree ? DynamicBranchResult<D> : {});

const TEMPLATE_TOKEN = /\$\{([^}]+)\}/g;

const formatTemplate = (template: string, values: Record<string, string | number>, keyPath: string) =>
  template.replace(TEMPLATE_TOKEN, (_match, token: string) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

// eslint-disable-next-line @typescript-eslint/no-explicit-any -- type guard requires any for generic erasure
const isDynamicDefinition = (value: unknown): value is DynamicSelectorDefinition<any> =>
  Boolean(value && typeof value === "object" && (value as DynamicSelectorDefinition).kind === "dynamic-selector");

/* eslint-disable @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-argument -- runtime type infrastructure uses intentional any erasure */
const normalizeParams = (
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- generic type erasure at runtime boundary
  definition: DynamicSelectorDefinition<any>,
  raw: Record<string, string | number>,
  path: string,
) => {
  const schema: ParamSchema = definition.params ?? ({} as ParamSchema);
  const normalized: Record<string, string | number> = {};

  for (const key of Object.keys(schema)) {
    if (!(key in raw)) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    const definitionEntry = schema[key];
    const value = raw[key];
    if (!definitionEntry || value === undefined) continue;
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
  // eslint-disable-next-line @typescript-eslint/no-explicit-any -- generic type erasure at runtime boundary
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

export const defineDynamicSelector = <P extends ParamSchema | undefined>(
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

/* eslint-enable @typescript-eslint/no-unsafe-assignment, @typescript-eslint/no-unsafe-argument */

const literalSelectors: LiteralSelectorTree = {
  workspace: {
    paneGrid: 'pane-grid',
    newTerminalButton: 'new-terminal-button',
    paneContainer: 'terminal-pane-container',
    sidebarShell: 'workspace-sidebar-shell',
    sidebarToggle: 'workspace-sidebar-toggle',
    sidebarBackdrop: 'workspace-sidebar-backdrop',
    sidebarResizeHandle: 'workspace-sidebar-resize-handle',
    sidebarTopbar: 'workspace-sidebar-topbar',
    sidebarActiveTitle: 'workspace-sidebar-active-title',
    toggleView: 'workspace-toggle-view',
  },
  terminal: {
    pane: 'terminal-pane',
  },
  launcher: {
    dialog: 'terminal-launcher',
    emptyShell: 'launcher-empty-shell',
    customInput: 'launcher-custom-input',
    customLaunch: 'launcher-custom-launch',
  },
  nav: {
    settings: 'nav-settings',
  },
  settings: {
    error: 'settings-error',
    createProfile: 'create-profile',
  },
  toolbar: {
    container: 'mobile-toolbar',
  },
  voice: {
    micButton: 'voice-mic-btn',
    errorTooltip: 'voice-error-tooltip',
  },
  ai: {
    input: 'ai-input',
    prompt: 'ai-input-prompt',
    generate: 'ai-input-generate',
    result: 'ai-input-result',
    execute: 'ai-input-execute',
    copy: 'ai-input-copy',
    error: 'ai-input-error',
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
};

const dynamicSelectorDefinitions: DynamicSelectorTree = {
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
};

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
