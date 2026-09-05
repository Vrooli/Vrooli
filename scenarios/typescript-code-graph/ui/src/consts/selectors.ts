/**
 * Vrooli Ascension selector registry
 *
 * This file is the single source of truth for every selector used by the UI and
 * by Vrooli Ascension workflows. We deliberately model selectors as two
 * declarative maps (one literal, one dynamic) and rely on a small helper to
 * produce the typed `selectors` export plus the in-memory `selectorsManifest`
 * export consumed by workflow tooling. Do not hand-roll selector helpers or
 * change this structure—update the maps below so UI code, automation flows, and
 * the manifest builder all stay in sync across every scenario.
 *
 * ## Manifest Export
 *
 * If you need to add or modify selectors:
 *
 * 1. Update the `literalSelectors` object below for static selectors
 * 2. Update the `dynamicSelectorDefinitions` object for parameterized selectors
 * 3. `selectorsManifest` updates from the same source maps automatically
 */
import { LOCALE_CODES } from "../i18n/locales";

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
  readonly [key: string]:
    | DynamicSelectorBranch
    | DynamicSelectorDefinition<ParamSchema | undefined>;
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
  template.replace(TEMPLATE_TOKEN, (_match: string, token: string) => {
    if (!(token in values)) {
      throw new Error(`Missing parameter '${token}' for selector '${keyPath}'`);
    }
    return String(values[token]);
  });

const toDataTestIdSelector = (testId: string) => `[data-testid="${testId}"]`;

const isDynamicDefinition = (
  value: unknown,
): value is DynamicSelectorDefinition<ParamSchema | undefined> =>
  Boolean(
    value &&
      typeof value === "object" &&
      (value as { kind?: unknown }).kind === "dynamic-selector",
  );

const normalizeParams = (
  definition: DynamicSelectorDefinition<ParamSchema | undefined>,
  raw: Record<string, string | number>,
  path: string,
): Record<string, string | number> => {
  const schema: ParamSchema = definition.params ?? {};
  const normalized: Record<string, string | number> = {};

  for (const [key, definitionEntry] of Object.entries(schema)) {
    if (!(key in raw)) {
      throw new Error(`Selector '${path}' is missing parameter '${key}'`);
    }
    const value = raw[key];
    // Defensive: `key in raw` doesn't narrow `raw[key]` under noUncheckedIndexedAccess.
    if (value === undefined) {
      throw new Error(`Selector '${path}' parameter '${key}' is undefined`);
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
      const paramEntries = Object.entries(value.params ?? {});
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
        literalValue,
        isDynamicDefinition(dynamicValue) ? undefined : dynamicValue,
        nextPath,
      );
      return;
    }

    if (dynamicValue) {
      if (isDynamicDefinition(dynamicValue)) {
        merged[key] = createDynamicSelectorFn(dynamicValue, nextPath.join("."));
        return;
      }
      merged[key] = mergeLiteralAndDynamicNodes(undefined, dynamicValue, nextPath);
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

/**
 * Build a dynamic-selector definition. Exported so unit tests and downstream
 * tooling can construct registries; scenario authors should still edit the
 * `dynamicSelectorDefinitions` map at the bottom of this file rather than
 * calling this from elsewhere in the codebase.
 */
export const defineDynamicSelector = <P extends ParamSchema | undefined>(
  definition: Omit<DynamicSelectorDefinition<P>, "kind">,
): DynamicSelectorDefinition<P> => ({
  ...definition,
  kind: "dynamic-selector",
});

/**
 * Compose a typed selectors object plus a manifest from literal + dynamic
 * trees. Exported for unit tests; production registries are built from the
 * private `literalSelectors` and `dynamicSelectorDefinitions` maps below.
 */
export const createSelectorRegistry = <
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

// Use `satisfies` rather than `: LiteralSelectorTree` so TypeScript preserves
// the narrow literal shape. Without this the index signature widens every
// branch to `T | undefined` under `noUncheckedIndexedAccess` and breaks the
// `selectors.app.title` ergonomics this registry exists to provide.
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
    workbench: "page-workbench",
    settings: "page-settings",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  shared: {
    emptyState: {
      root: "empty-state-root",
      title: "empty-state-title",
      description: "empty-state-description",
      action: "empty-state-action",
    },
    errorState: {
      root: "error-state-root",
      title: "error-state-title",
      message: "error-state-message",
      retryButton: "error-state-retry",
    },
    loadingState: {
      root: "loading-state-root",
    },
  },
  ui: {
    tabs: {
      root: "tabs-root",
      list: "tabs-list",
    },
  },
  workbench: {
    extractBar: {
      root: "workbench-extract-bar",
      target: "workbench-extract-target",
      projectDir: "workbench-extract-project-dir",
      submit: "workbench-extract-submit",
    },
    stats: {
      root: "workbench-stats",
      files: "workbench-stat-files",
      modules: "workbench-stat-modules",
      symbols: "workbench-stat-symbols",
      imports: "workbench-stat-imports",
      warnings: "workbench-stat-warnings",
      hash: "workbench-stat-hash",
    },
    status: {
      loading: "workbench-status-loading",
      error: "workbench-status-error",
      empty: "workbench-status-empty",
      workspaceUnsupported: "workbench-status-workspace-unsupported",
    },
  },
  features: {
    explorer: {
      root: "explorer-root",
      canvas: {
        root: "explorer-canvas-root",
        summary: "explorer-canvas-summary",
      },
      accessibleList: {
        root: "explorer-accessible-list-root",
        empty: "explorer-accessible-list-empty",
      },
      legend: {
        root: "explorer-legend-root",
      },
      filterBar: {
        root: "explorer-filter-bar-root",
        allChip: "explorer-filter-bar-all",
        empty: "explorer-filter-bar-empty",
      },
      cycleBanner: "explorer-cycle-banner",
      drilldown: {
        root: "explorer-drilldown-root",
        title: "explorer-drilldown-title",
        empty: "explorer-drilldown-empty",
      },
    },
    sidecar: {
      root: "sidecar-status-root",
      loading: "sidecar-status-loading",
      error: "sidecar-status-error",
      message: "sidecar-status-message",
    },
    warnings: {
      root: "warnings-root",
      empty: "warnings-empty",
      list: "warnings-list",
      summary: "warnings-summary",
    },
    rewrite: {
      root: "rewrite-root",
      opsEditor: {
        root: "rewrite-ops-editor",
        empty: "rewrite-ops-empty",
        addFileMove: "rewrite-add-file-move",
        addImportRewrite: "rewrite-add-import-rewrite",
      },
      plan: {
        button: "rewrite-plan-button",
        result: "rewrite-plan-result",
        empty: "rewrite-plan-empty",
      },
      apply: {
        button: "rewrite-apply-button",
        result: "rewrite-apply-result",
        confirmDialog: {
          root: "rewrite-apply-confirm",
          confirm: "rewrite-apply-confirm-yes",
          cancel: "rewrite-apply-confirm-cancel",
        },
      },
    },
    fixtures: {
      root: "fixtures-root",
      list: "fixtures-list",
      empty: "fixtures-empty",
      loading: "fixtures-loading",
      error: "fixtures-error",
      result: "fixtures-result",
      diff: "fixtures-diff",
    },
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
      params: { key: { type: "enum", values: ["workbench", "settings"] as const } },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: { key: { type: "enum", values: ["workbench", "settings"] as const } },
    }),
  },
  ui: {
    tabs: {
      trigger: defineDynamicSelector({
        description: "Tab trigger button by tab value",
        testIdPattern: "tabs-trigger-${value}",
        params: { value: { type: "string" } },
      }),
      panel: defineDynamicSelector({
        description: "Tab panel by tab value",
        testIdPattern: "tabs-panel-${value}",
        params: { value: { type: "string" } },
      }),
    },
  },
  shared: {
    severityBadge: {
      root: defineDynamicSelector({
        description: "Severity badge by level",
        testIdPattern: "severity-badge-${level}",
        params: {
          level: {
            type: "enum",
            values: ["info", "low", "medium", "high", "critical"] as const,
          },
        },
      }),
    },
  },
  features: {
    explorer: {
      canvas: {
        node: defineDynamicSelector({
          description: "Graph canvas node group by node id",
          testIdPattern: "explorer-canvas-node-${id}",
          params: { id: { type: "string" } },
        }),
      },
      accessibleList: {
        item: defineDynamicSelector({
          description: "Accessible-list row by node id",
          testIdPattern: "explorer-accessible-item-${id}",
          params: { id: { type: "string" } },
        }),
      },
      filterBar: {
        chip: defineDynamicSelector({
          description: "Domain filter chip by package key",
          testIdPattern: "explorer-filter-chip-${key}",
          params: { key: { type: "string" } },
        }),
      },
      legend: {
        severity: defineDynamicSelector({
          description: "Legend severity row by level",
          testIdPattern: "explorer-legend-${level}",
          params: {
            level: {
              type: "enum",
              values: ["info", "low", "medium", "high", "critical"] as const,
            },
          },
        }),
      },
      drilldown: {
        symbol: defineDynamicSelector({
          description: "File-drilldown symbol row by node id",
          testIdPattern: "explorer-drilldown-symbol-${id}",
          params: { id: { type: "string" } },
        }),
        symbolComments: defineDynamicSelector({
          description: "Leading-comments / JSDoc block for a symbol row by node id",
          testIdPattern: "explorer-drilldown-symbol-comments-${id}",
          params: { id: { type: "string" } },
        }),
      },
    },
    sidecar: {
      indicator: defineDynamicSelector({
        description: "Sidecar status indicator by status token",
        testIdPattern: "sidecar-status-indicator-${status}",
        params: {
          status: {
            type: "enum",
            values: [
              "unspecified",
              "ready",
              "unhealthy",
              "restarting",
              "permanently_unhealthy",
            ] as const,
          },
        },
      }),
    },
    warnings: {
      item: defineDynamicSelector({
        description: "Warning row by index",
        testIdPattern: "warnings-item-${index}",
        params: { index: { type: "number" } },
      }),
    },
    rewrite: {
      opRow: defineDynamicSelector({
        description: "Rewrite operation editor row by index",
        testIdPattern: "rewrite-op-row-${index}",
        params: { index: { type: "number" } },
      }),
      opResult: defineDynamicSelector({
        description: "Rewrite apply per-operation result row by index",
        testIdPattern: "rewrite-op-result-${index}",
        params: { index: { type: "number" } },
      }),
    },
    fixtures: {
      item: defineDynamicSelector({
        description: "Fixture list row by fixture name",
        testIdPattern: "fixtures-item-${name}",
        params: { name: { type: "string" } },
      }),
    },
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

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
