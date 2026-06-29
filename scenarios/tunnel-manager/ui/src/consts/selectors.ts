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
  // Shared query-state primitive (loading / error / empty) — reused by every surface.
  queryState: {
    loading: "query-state-loading",
    error: "query-state-error",
    empty: "query-state-empty",
  },
  overview: {
    page: "overview-page",
    readinessCard: "overview-readiness-card",
    readinessStatus: "overview-readiness-status",
    modeBadge: "overview-mode-badge",
    missingFields: "overview-missing-fields",
    tunnelCard: "overview-tunnel-card",
    tunnelStatus: "overview-tunnel-status",
    tunnelScore: "overview-tunnel-score",
    exposureCard: "overview-exposure-card",
    coreCount: "overview-core-count",
    leasedCount: "overview-leased-count",
    recoveryCard: "overview-recovery-card",
    recoveryStatus: "overview-recovery-status",
    circuitWarning: "overview-circuit-warning",
    modeCallout: "overview-mode-callout",
    driftCard: "overview-drift-card",
    driftManaged: "overview-drift-managed",
    driftExternal: "overview-drift-external",
    driftUnmanaged: "overview-drift-unmanaged",
  },
  drift: {
    panel: "drift-panel",
    summary: "drift-summary",
    modeHint: "drift-mode-hint",
    refreshButton: "drift-refresh-button",
    table: "drift-table",
    row: "drift-row",
    hostname: "drift-hostname",
    target: "drift-target",
    stateBadge: "drift-state-badge",
    sourceBadge: "drift-source-badge",
    actionError: "drift-action-error",
    confirmDialog: "drift-confirm-dialog",
    confirmButton: "drift-confirm-button",
    cancelButton: "drift-cancel-button",
  },
  routes: {
    panel: "routes-panel",
    addForm: "routes-add-form",
    subdomainInput: "routes-subdomain-input",
    targetInput: "routes-target-input",
    domainInput: "routes-domain-input",
    publicExposureSelect: "routes-public-exposure-select",
    addButton: "routes-add-button",
    addError: "routes-add-error",
    table: "routes-table",
    row: "routes-row",
    sourceBadge: "routes-source-badge",
    publicExposureBadge: "routes-public-exposure-badge",
    publicExposureError: "routes-public-exposure-error",
    url: "routes-url",
    deleteButton: "routes-delete-button",
    deleteError: "routes-delete-error",
  },
  access: {
    panel: "access-panel",
    summary: "access-summary",
    globalBadge: "access-global-badge",
    configuredBadge: "access-configured-badge",
    note: "access-note",
    refreshButton: "access-refresh-button",
    planCreate: "access-plan-create",
    planRemove: "access-plan-remove",
    table: "access-table",
    row: "access-row",
    hostName: "access-host-name",
    overrideBadge: "access-override-badge",
    bypassBadge: "access-bypass-badge",
    managedBadge: "access-managed-badge",
    appId: "access-app-id",
  },
  exposure: {
    panel: "exposure-panel",
    summary: "exposure-summary",
    coreCount: "exposure-core-count",
    leasedCount: "exposure-leased-count",
    unhealthyCount: "exposure-unhealthy-count",
    searchInput: "exposure-search-input",
    tierFilter: "exposure-tier-filter",
    reconcileButton: "exposure-reconcile-button",
    reconcileResult: "exposure-reconcile-result",
    table: "exposure-table",
    row: "exposure-row",
    tierBadge: "exposure-tier-badge",
    healthBadge: "exposure-health-badge",
    url: "exposure-url",
    leaseExpiry: "exposure-lease-expiry",
    exposeForm: "exposure-expose-form",
    exposeInput: "exposure-expose-input",
    exposeButton: "exposure-expose-button",
    exposeError: "exposure-expose-error",
    extendButton: "exposure-extend-button",
    revokeButton: "exposure-revoke-button",
    actionError: "exposure-action-error",
    refreshButton: "exposure-refresh-button",
  },
  recovery: {
    statePanel: "recovery-state-panel",
    summary: "recovery-summary",
    statusValue: "recovery-status-value",
    circuitValue: "recovery-circuit-value",
    nextAction: "recovery-next-action",
    policyNote: "recovery-policy-note",
    timeline: "recovery-timeline",
    timelineRow: "recovery-timeline-row",
    eventDetails: "recovery-event-details",
    recoverButton: "recovery-recover-button",
    forceToggle: "recovery-force-toggle",
    forceWarning: "recovery-force-warning",
    confirmDialog: "recovery-confirm-dialog",
    confirmButton: "recovery-confirm-button",
    cancelButton: "recovery-cancel-button",
    actionError: "recovery-action-error",
    refreshButton: "recovery-refresh-button",
  },
  metrics: {
    panel: "metrics-panel",
    summary: "metrics-summary",
    table: "metrics-table",
    row: "metrics-row",
    latest: "metrics-latest",
    scrapeButton: "metrics-scrape-button",
    probesPanel: "metrics-probes-panel",
    probesTable: "metrics-probes-table",
    probesRow: "metrics-probes-row",
    runProbesButton: "metrics-run-probes-button",
    classification: "metrics-classification",
    classCount: "metrics-class-count",
    limitation: "metrics-limitation",
  },
  audit: {
    panel: "audit-panel",
    summary: "audit-summary",
    table: "audit-table",
    row: "audit-row",
    statusBadge: "audit-status-badge",
    violationCount: "audit-violation-count",
    statusFilter: "audit-status-filter",
    remediation: "audit-remediation",
    runButton: "audit-run-button",
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
    overview: "page-overview",
    exposure: "page-exposure",
    recovery: "page-recovery",
    metrics: "page-metrics",
    audit: "page-audit",
    drift: "page-drift",
    settings: "page-settings",
  },
  settingsPage: {
    configPanel: "settings-config-panel",
    currentMode: "settings-current-mode",
    remoteAvailable: "settings-remote-available",
    credentialSource: "settings-credential-source",
    syncReady: "settings-sync-ready",
    missingFields: "settings-missing-fields",
    credentialsForm: "settings-credentials-form",
    accountIdInput: "settings-account-id-input",
    tunnelIdInput: "settings-tunnel-id-input",
    apiTokenInput: "settings-api-token-input",
    credentialsSaveButton: "settings-credentials-save-button",
    credentialsClearButton: "settings-credentials-clear-button",
    credentialPolicy: "settings-credential-policy",
    credentialNextAction: "settings-credential-next-action",
    credentialShadowWarning: "settings-credential-shadow-warning",
    credentialFields: "settings-credential-fields",
    localModeButton: "settings-local-mode-button",
    remoteModeButton: "settings-remote-mode-button",
    syncDryRunButton: "settings-sync-dry-run-button",
    syncApplyButton: "settings-sync-apply-button",
    syncResult: "settings-sync-result",
    actionError: "settings-action-error",
    switchModeNote: "settings-switch-mode-note",
    reconcileNowButton: "settings-reconcile-now-button",
    reconcileResult: "settings-reconcile-result",
    reviewDriftLink: "settings-review-drift-link",
    publicExposurePanel: "settings-public-exposure-panel",
    publicExposureState: "settings-public-exposure-state",
    publicExposureToggle: "settings-public-exposure-toggle",
    publicExposureStatusLink: "settings-public-exposure-status-link",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
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
      params: {
        key: {
          type: "enum",
          values: [
            "overview",
            "exposure",
            "recovery",
            "metrics",
            "audit",
            "drift",
            "settings",
          ] as const,
        },
      },
    }),
    bottomNavLink: defineDynamicSelector({
      description: "Bottom-nav link by canonical nav key",
      testIdPattern: "layout-bottom-nav-link-${key}",
      params: {
        key: {
          type: "enum",
          values: [
            "overview",
            "exposure",
            "recovery",
            "metrics",
            "audit",
            "drift",
            "settings",
          ] as const,
        },
      },
    }),
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
  routes: {
    publicExposureRowSelect: defineDynamicSelector({
      description: "Per-route public-exposure override select, by route id",
      testIdPattern: "routes-public-exposure-row-select-${id}",
      params: { id: { type: "string" } },
    }),
  },
  drift: {
    adoptButton: defineDynamicSelector({
      description: "Adopt action button for a drift row, by hostname",
      testIdPattern: "drift-adopt-${hostname}",
      params: { hostname: { type: "string" } },
    }),
    ignoreButton: defineDynamicSelector({
      description: "Ignore action button for a drift row, by hostname",
      testIdPattern: "drift-ignore-${hostname}",
      params: { hostname: { type: "string" } },
    }),
    pruneButton: defineDynamicSelector({
      description: "Prune action button for a drift row, by hostname",
      testIdPattern: "drift-prune-${hostname}",
      params: { hostname: { type: "string" } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
