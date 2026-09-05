import { librarySelectors } from "./selectors.library";
export { librarySelectors };
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
    dashboard: "page-dashboard",
    settings: "page-settings",
    plans: "page-plans",
    planDetail: "page-plan-detail",
    authoring: "page-authoring",
    execution: "page-execution",
    validation: "page-validation",
    triage: "page-triage",
    velocity: "page-velocity",
  },
  asyncSuffix: {
    loading: "loading",
    error: "error",
    empty: "empty",
  },
  plans: {
    list: "plans-list",
    createForm: "plans-create-form",
    templateSelect: "plans-template-select",
    titleInput: "plans-title-input",
    createButton: "plans-create-button",
    detailMarkdownToggle: "plan-detail-markdown-toggle",
    detailMarkdown: "plan-detail-markdown",
    detailGraph: "plan-detail-graph",
    detailPhases: "plan-detail-phases",
    relevantContext: "plan-relevant-context",
    archiveButton: "plan-detail-archive-button",
  },
  authoring: {
    startForm: "authoring-start-form",
    titleInput: "authoring-title-input",
    templateInput: "authoring-template-input",
    startButton: "authoring-start-button",
    sections: "authoring-sections",
    contentInput: "authoring-content-input",
    submitButton: "authoring-submit-button",
    nextButton: "authoring-next-button",
    validateButton: "authoring-validate-button",
    violations: "authoring-violations",
    autofillButton: "authoring-autofill-button",
    autofillResults: "authoring-autofill-results",
    guidance: "authoring-guidance",
    phases: "authoring-phases",
    phaseTitleInput: "authoring-phase-title-input",
    phaseIntentInput: "authoring-phase-intent-input",
    phaseAddButton: "authoring-phase-add-button",
    phaseNextButton: "authoring-phase-next-button",
    phaseFieldSelect: "authoring-phase-field-select",
    phaseFieldInput: "authoring-phase-field-input",
    phaseSubmitButton: "authoring-phase-submit-button",
    contextItems: "authoring-context-items",
    contextKindSelect: "authoring-context-kind-select",
    contextPhaseToggle: "authoring-context-phase-toggle",
    contextLabelInput: "authoring-context-label-input",
    contextReasonInput: "authoring-context-reason-input",
    contextInstructionInput: "authoring-context-instruction-input",
    contextCommandInput: "authoring-context-command-input",
    contextTargetInput: "authoring-context-target-input",
    contextSubmitButton: "authoring-context-submit-button",
    contextRemoveButton: "authoring-context-remove-button",
    contextConceptsInput: "authoring-context-concepts-input",
    contextComplexityInput: "authoring-context-complexity-input",
    skillPackButton: "authoring-skill-pack-button",
    finalizeButton: "authoring-finalize-button",
    finalizedBanner: "authoring-finalized-banner",
  },
  execution: {
    startForm: "execution-start-form",
    planSelect: "execution-plan-select",
    runIdInput: "execution-run-id-input",
    startButton: "execution-start-button",
    resumeForm: "execution-resume-form",
    resumeExecutionIdInput: "execution-resume-execution-id-input",
    resumeButton: "execution-resume-button",
    guidedStep: "execution-guided-step",
    context: "execution-context",
    contextButton: "execution-context-button",
    setupContext: "execution-setup-context",
    freshenStatus: "execution-freshen-status",
    transitionSelect: "execution-transition-select",
    transitionButton: "execution-transition-button",
    decisionSummary: "execution-decision-summary",
    decisionDetail: "execution-decision-detail",
    recordDecisionButton: "execution-record-decision-button",
    findingTitle: "execution-finding-title",
    findingDetail: "execution-finding-detail",
    recordFindingButton: "execution-record-finding-button",
    feedbackCheckpoint: "execution-feedback-checkpoint",
    noFeedbackButton: "execution-no-feedback-button",
    bugTitle: "execution-bug-title",
    bugDetail: "execution-bug-detail",
    recordBugButton: "execution-record-bug-button",
    recordTitle: "execution-record-title",
    recordDetail: "execution-record-detail",
    recordRecordButton: "execution-record-record-button",
    noteTitle: "execution-note-title",
    noteDetail: "execution-note-detail",
    recordNoteButton: "execution-record-note-button",
    logSummary: "execution-log-summary",
    completeButton: "execution-complete-button",
    handoff: "execution-handoff",
  },
  validation: {
    planSelect: "validation-plan-select",
    phaseSelect: "validation-phase-select",
    resolveButton: "validation-resolve-button",
    references: "validation-references",
    stalenessButton: "validation-staleness-button",
    staleness: "validation-staleness",
    baselineButton: "validation-baseline-button",
    baseline: "validation-baseline",
    runButton: "validation-run-button",
    result: "validation-result",
    commandFindings: "validation-command-findings",
    dodButton: "validation-dod-button",
    dod: "validation-dod",
  },
  triage: {
    list: "triage-list",
  },
  velocity: {
    planSelect: "velocity-plan-select",
    chart: "velocity-chart",
    table: "velocity-table",
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
            "dashboard",
            "plans",
            "authoring",
            "execution",
            "validation",
            "triage",
            "velocity",
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
            "dashboard",
            "plans",
            "authoring",
            "execution",
            "validation",
            "triage",
            "velocity",
            "settings",
          ] as const,
        },
      },
    }),
  },
  plans: {
    row: defineDynamicSelector({
      description: "Plan list row by plan id",
      testIdPattern: "plans-row-${id}",
      params: { id: { type: "string" } },
    }),
    phase: defineDynamicSelector({
      description: "Plan-detail phase card by phase id",
      testIdPattern: "plan-phase-${id}",
      params: { id: { type: "string" } },
    }),
  },
  authoring: {
    section: defineDynamicSelector({
      description: "Authoring section row by section key",
      testIdPattern: "authoring-section-${key}",
      params: { key: { type: "string" } },
    }),
  },
  execution: {
    logEntry: defineDynamicSelector({
      description: "Execution log entry by durable ledger id",
      testIdPattern: "execution-log-entry-${id}",
      params: { id: { type: "string" } },
    }),
    retrySync: defineDynamicSelector({
      description: "Retry downstream sync for a durable ledger entry",
      testIdPattern: "execution-retry-sync-${id}",
      params: { id: { type: "string" } },
    }),
  },
  triage: {
    row: defineDynamicSelector({
      description: "Triage finding row by finding id",
      testIdPattern: "triage-row-${id}",
      params: { id: { type: "string" } },
    }),
    promote: defineDynamicSelector({
      description: "Triage promote button by finding id",
      testIdPattern: "triage-promote-${id}",
      params: { id: { type: "string" } },
    }),
    dismiss: defineDynamicSelector({
      description: "Triage dismiss button by finding id",
      testIdPattern: "triage-dismiss-${id}",
      params: { id: { type: "string" } },
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
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry({ library: librarySelectors, ...literalSelectors }, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
