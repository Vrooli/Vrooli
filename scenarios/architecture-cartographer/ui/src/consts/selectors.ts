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
    subnav: "layout-workspace-subnav",
  },
  theme: {
    switcher: "theme-switcher",
    select: "theme-select",
  },
  pages: {
    overview: "page-overview",
    newTarget: "page-new-target",
    targetWorkspace: "page-target-workspace",
    targetConflicts: "page-target-conflicts",
    targetConflictDetail: "page-target-conflict-detail",
    targetGraph: "page-target-graph",
    targetApply: "page-target-apply",
    targetApplyDomain: "page-target-apply-domain",
    targetManifest: "page-target-manifest",
    targetAnalytics: "page-target-analytics",
    history: "page-history",
    settings: "page-settings",
  },
  features: {
    graph: {
      canvas: {
        root: "feature-graph-canvas",
        loading: "feature-graph-canvas-loading",
        error: "feature-graph-canvas-error",
        empty: "feature-graph-canvas-empty",
        fallback: "feature-graph-canvas-fallback",
        summary: "feature-graph-canvas-summary",
      },
      accessibleList: {
        root: "feature-graph-accessible-list",
        empty: "feature-graph-accessible-list-empty",
      },
      filterBar: {
        root: "feature-graph-filter-bar",
        loading: "feature-graph-filter-bar-loading",
        error: "feature-graph-filter-bar-error",
        empty: "feature-graph-filter-bar-empty",
        allChip: "feature-graph-filter-bar-all",
      },
      legend: {
        root: "feature-graph-legend",
        noConflict: "feature-graph-legend-no-conflict",
      },
    },
    conflicts: {
      list: {
        root: "feature-conflicts-list",
        loading: "feature-conflicts-list-loading",
        error: "feature-conflicts-list-error",
        empty: "feature-conflicts-list-empty",
        detectButton: "feature-conflicts-list-detect",
        validateButton: "feature-conflicts-list-validate",
      },
      detail: {
        root: "feature-conflicts-detail",
        loading: "feature-conflicts-detail-loading",
        notFound: "feature-conflicts-detail-not-found",
        evidence: "feature-conflicts-detail-evidence",
        fixes: "feature-conflicts-detail-fixes",
        locations: "feature-conflicts-detail-locations",
        backLink: "feature-conflicts-detail-back",
        actions: "feature-conflicts-detail-actions",
        assignedDomain: "feature-conflicts-detail-assigned-domain",
      },
      workbench: {
        root: "feature-conflicts-workbench",
        emptyDetail: "feature-conflicts-workbench-empty-detail",
      },
    },
    apply: {
      overview: {
        root: "feature-apply-overview",
        baseline: "feature-apply-overview-baseline",
        state: "feature-apply-overview-state",
      },
      plan: {
        root: "feature-apply-plan",
        loading: "feature-apply-plan-loading",
        error: "feature-apply-plan-error",
        empty: "feature-apply-plan-empty",
        planButton: "feature-apply-plan-button",
        dryRunButton: "feature-apply-plan-dry-run-button",
        applyButton: "feature-apply-plan-apply-button",
      },
      dryRun: {
        root: "feature-apply-dry-run",
        empty: "feature-apply-dry-run-empty",
      },
      confirmDialog: {
        root: "feature-apply-confirm-dialog",
        noteInput: "feature-apply-confirm-dialog-note",
        noteError: "feature-apply-confirm-dialog-note-error",
        confirmButton: "feature-apply-confirm-dialog-confirm",
        cancelButton: "feature-apply-confirm-dialog-cancel",
      },
      history: {
        root: "feature-apply-history",
        loading: "feature-apply-history-loading",
        error: "feature-apply-history-error",
        empty: "feature-apply-history-empty",
      },
    },
    manifest: {
      view: {
        root: "feature-manifest-view",
        loading: "feature-manifest-view-loading",
        error: "feature-manifest-view-error",
        empty: "feature-manifest-view-empty",
        validateButton: "feature-manifest-view-validate",
        copyButton: "feature-manifest-view-copy",
      },
      validation: {
        root: "feature-manifest-validation",
        validBanner: "feature-manifest-validation-valid",
        invalidBanner: "feature-manifest-validation-invalid",
      },
      inventory: {
        root: "feature-manifest-inventory",
        empty: "feature-manifest-inventory-empty",
      },
    },
    analytics: {
      stats: {
        root: "feature-analytics-stats",
        loading: "feature-analytics-stats-loading",
        error: "feature-analytics-stats-error",
        suppressed: "feature-analytics-stats-suppressed",
      },
      events: {
        root: "feature-analytics-events",
        loading: "feature-analytics-events-loading",
        error: "feature-analytics-events-error",
        empty: "feature-analytics-events-empty",
      },
      placements: {
        root: "feature-analytics-placements",
        loading: "feature-analytics-placements-loading",
        error: "feature-analytics-placements-error",
        empty: "feature-analytics-placements-empty",
      },
      buildDeltas: {
        root: "feature-analytics-build-deltas",
        empty: "feature-analytics-build-deltas-empty",
      },
    },
    settingsExt: {
      root: "feature-settings-ext",
      densityToggle: "feature-settings-ext-density",
      reducedMotionToggle: "feature-settings-ext-reduced-motion",
      handednessToggle: "feature-settings-ext-handedness",
      defaultScenarioInput: "feature-settings-ext-default-scenario",
      defaultDomainInput: "feature-settings-ext-default-domain",
    },
    targets: {
      recent: {
        root: "feature-targets-recent",
        empty: "feature-targets-recent-empty",
      },
      newForm: {
        root: "feature-targets-new-form",
        scenarioInput: "feature-targets-new-form-scenario-input",
        scenarioInputError: "feature-targets-new-form-scenario-error",
        submitButton: "feature-targets-new-form-submit",
        successBanner: "feature-targets-new-form-success",
        errorBanner: "feature-targets-new-form-error",
        openWorkspaceLink: "feature-targets-new-form-open-workspace",
      },
      activeSnapshots: {
        root: "feature-targets-active-snapshots",
        loading: "feature-targets-active-snapshots-loading",
        error: "feature-targets-active-snapshots-error",
        empty: "feature-targets-active-snapshots-empty",
      },
    },
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  shared: {
    emptyState: {
      root: "shared-empty-state",
      title: "shared-empty-state-title",
      description: "shared-empty-state-description",
      action: "shared-empty-state-action",
    },
    loadingState: {
      root: "shared-loading-state",
    },
    errorState: {
      root: "shared-error-state",
      title: "shared-error-state-title",
      message: "shared-error-state-message",
      retryButton: "shared-error-state-retry",
    },
    routeErrorFallback: {
      root: "shared-route-error-fallback",
      retryButton: "shared-route-error-fallback-retry",
      homeButton: "shared-route-error-fallback-home",
    },
    dataTable: {
      root: "shared-data-table",
      empty: "shared-data-table-empty",
      rowCount: "shared-data-table-row-count",
    },
    splitPane: {
      root: "shared-split-pane",
      primary: "shared-split-pane-primary",
      secondary: "shared-split-pane-secondary",
      handle: "shared-split-pane-handle",
    },
    diffView: {
      root: "shared-diff-view",
    },
    keyboardShortcut: {
      root: "shared-keyboard-shortcut",
    },
  },
  ui: {
    badge: { root: "ui-badge" },
    card: { root: "ui-card", header: "ui-card-header", body: "ui-card-body" },
    dialog: {
      root: "ui-dialog",
      backdrop: "ui-dialog-backdrop",
      panel: "ui-dialog-panel",
      title: "ui-dialog-title",
      closeButton: "ui-dialog-close",
    },
    select: { root: "ui-select" },
    tabs: { root: "ui-tabs", list: "ui-tabs-list" },
    tooltip: { root: "ui-tooltip" },
    checkbox: { root: "ui-checkbox" },
    radio: { root: "ui-radio" },
    code: { root: "ui-code" },
    kbd: { root: "ui-kbd" },
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
      params: { key: { type: "enum", values: ["overview", "targets", "history", "settings"] as const } },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: { key: { type: "enum", values: ["overview", "targets", "history", "settings"] as const } },
    }),
    workspaceSubnavLink: defineDynamicSelector({
      description: "Workspace sub-nav tab link by canonical key",
      testIdPattern: "layout-workspace-subnav-${key}",
      params: {
        key: {
          type: "enum",
          values: ["graph", "manifest", "conflicts", "apply", "analytics"] as const,
        },
      },
    }),
  },
  shared: {
    dataTable: {
      row: defineDynamicSelector({
        description: "Data-table row by stable id",
        testIdPattern: "shared-data-table-row-${id}",
        params: { id: { type: "string" } },
      }),
      cell: defineDynamicSelector({
        description: "Data-table cell by row id + column key",
        testIdPattern: "shared-data-table-cell-${id}-${column}",
        params: { id: { type: "string" }, column: { type: "string" } },
      }),
    },
    severityBadge: {
      root: defineDynamicSelector({
        description: "Severity badge by level",
        testIdPattern: "shared-severity-badge-${level}",
        params: {
          level: {
            type: "enum",
            values: ["info", "low", "medium", "high", "critical"] as const,
          },
        },
      }),
    },
  },
  ui: {
    tabs: {
      trigger: defineDynamicSelector({
        description: "Tab trigger by tab value",
        testIdPattern: "ui-tabs-trigger-${value}",
        params: { value: { type: "string" } },
      }),
      panel: defineDynamicSelector({
        description: "Tab panel by tab value",
        testIdPattern: "ui-tabs-panel-${value}",
        params: { value: { type: "string" } },
      }),
    },
  },
  features: {
    graph: {
      canvas: {
        node: defineDynamicSelector({
          description: "Graph canvas node by node id",
          testIdPattern: "feature-graph-canvas-node-${id}",
          params: { id: { type: "string" } },
        }),
      },
      accessibleList: {
        item: defineDynamicSelector({
          description: "Graph accessible-list item by node id",
          testIdPattern: "feature-graph-accessible-list-item-${id}",
          params: { id: { type: "string" } },
        }),
      },
      filterBar: {
        chip: defineDynamicSelector({
          description: "Graph filter-bar chip by domain key",
          testIdPattern: "feature-graph-filter-bar-chip-${key}",
          params: { key: { type: "string" } },
        }),
      },
      legend: {
        severity: defineDynamicSelector({
          description: "Graph legend row by severity level",
          testIdPattern: "feature-graph-legend-${level}",
          params: {
            level: {
              type: "enum",
              values: ["info", "low", "medium", "high", "critical"] as const,
            },
          },
        }),
      },
    },
    conflicts: {
      list: {
        row: defineDynamicSelector({
          description: "Conflict list row by conflict id",
          testIdPattern: "feature-conflicts-list-row-${id}",
          params: { id: { type: "string" } },
        }),
        openButton: defineDynamicSelector({
          description: "Conflict list open detail link by conflict id",
          testIdPattern: "feature-conflicts-list-open-${id}",
          params: { id: { type: "string" } },
        }),
      },
      detail: {
        actionButton: defineDynamicSelector({
          description: "Conflict detail action button by flow event",
          testIdPattern: "feature-conflicts-detail-action-${event}",
          params: {
            event: {
              type: "enum",
              values: [
                "assign",
                "split",
                "resolve",
                "force_resolve",
                "validate",
                "commit",
                "reopen",
              ] as const,
            },
          },
        }),
        evidenceItem: defineDynamicSelector({
          description: "Conflict detail evidence item by index",
          testIdPattern: "feature-conflicts-detail-evidence-${index}",
          params: { index: { type: "number" } },
        }),
        fixItem: defineDynamicSelector({
          description: "Conflict detail suggested-fix item by id",
          testIdPattern: "feature-conflicts-detail-fix-${id}",
          params: { id: { type: "string" } },
        }),
      },
    },
    apply: {
      plan: {
        operationRow: defineDynamicSelector({
          description: "Apply plan operation row by operation id",
          testIdPattern: "feature-apply-plan-op-${id}",
          params: { id: { type: "string" } },
        }),
        domainLink: defineDynamicSelector({
          description: "Apply plan per-domain link by domain key",
          testIdPattern: "feature-apply-plan-domain-${key}",
          params: { key: { type: "string" } },
        }),
      },
      history: {
        row: defineDynamicSelector({
          description: "Apply history row by apply run id",
          testIdPattern: "feature-apply-history-row-${id}",
          params: { id: { type: "string" } },
        }),
      },
    },
    manifest: {
      diagnostic: defineDynamicSelector({
        description: "Manifest diagnostic row by diagnostic index",
        testIdPattern: "feature-manifest-diagnostic-${index}",
        params: { index: { type: "number" } },
      }),
      domainRow: defineDynamicSelector({
        description: "Manifest domain inventory row by domain name",
        testIdPattern: "feature-manifest-domain-${name}",
        params: { name: { type: "string" } },
      }),
    },
    analytics: {
      eventRow: defineDynamicSelector({
        description: "Analytics event row by event id",
        testIdPattern: "feature-analytics-event-${id}",
        params: { id: { type: "string" } },
      }),
      placementRow: defineDynamicSelector({
        description: "Analytics placement row by placement id",
        testIdPattern: "feature-analytics-placement-${id}",
        params: { id: { type: "string" } },
      }),
    },
    targets: {
      recent: {
        item: defineDynamicSelector({
          description: "Recent-target list item by scenario name",
          testIdPattern: "feature-targets-recent-item-${scenario}",
          params: { scenario: { type: "string" } },
        }),
        openButton: defineDynamicSelector({
          description: "Recent-target open button by scenario name",
          testIdPattern: "feature-targets-recent-open-${scenario}",
          params: { scenario: { type: "string" } },
        }),
        removeButton: defineDynamicSelector({
          description: "Recent-target remove button by scenario name",
          testIdPattern: "feature-targets-recent-remove-${scenario}",
          params: { scenario: { type: "string" } },
        }),
      },
      activeSnapshots: {
        item: defineDynamicSelector({
          description: "Active-snapshot row by snapshot id",
          testIdPattern: "feature-targets-active-snapshot-${id}",
          params: { id: { type: "string" } },
        }),
      },
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
