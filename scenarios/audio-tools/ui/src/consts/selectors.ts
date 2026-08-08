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
 * ## Auto-Generated Manifest
 *
 * If you need to add or modify selectors:
 *
 * 1. Update the `literalSelectors` object below for static selectors
 * 2. Update the `dynamicSelectorDefinitions` object for parameterized selectors
 * 3. Run `pnpm selector:manifest` to regenerate `selectors.manifest.json`
 *
 * Do not hand-edit the generated manifest.
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
  locale: {
    switcher: "locale-switcher",
  },
  errorBoundary: {
    root: "error-boundary-root",
    retryButton: "error-boundary-retry",
  },
  diagnostics: {
    suiteRun: "suite-run",
    suiteLastRun: "suite-last-run",
  },
  streamConfig: {
    enginePicker: "stream-config-engine-picker",
    switchPrompt: "stream-config-switch-prompt",
    confirmSwitch: "stream-config-confirm-switch",
    cancelSwitch: "stream-config-cancel-switch",
    stallRejectsInput: "stream-config-stall-rejects-input",
    saveOverlap: "stream-config-save-overlap",
  },
  dictationStudio: {
    modeFree: "dictation-mode-free",
    modeScripted: "dictation-mode-scripted",
    recordStart: "dictation-record-start",
    recordCancel: "dictation-record-cancel",
    recordState: "dictation-record-state",
		streamStatus: "dictation-stream-status",
		recordError: "dictation-record-error",
    audioMeter: "dictation-audio-meter",
    turnDetails: "dictation-turn-details",
    turnCaptureStatus: "dictation-turn-capture-status",
    turnSentStatus: "dictation-turn-sent-status",
    turnDoneStatus: "dictation-turn-done-status",
    turnDoneReady: "dictation-turn-done-ready",
    turnProcessedStatus: "dictation-turn-processed-status",
    turnProcessedReady: "dictation-turn-processed-ready",
    exportDiagnostic: "dictation-export-diagnostic",
    finalTranscript: "dictation-final-transcript",
    transcriptEditor: "dictation-transcript-editor",
    tagInput: "dictation-tag-input",
    saveClip: "dictation-save-clip",
    corpusList: "dictation-corpus-list",
    evalTable: "dictation-eval-table",
    evalSummary: "dictation-eval-summary",
    evalClips: "dictation-eval-clips",
    lengthCurveChart: "dictation-length-curve-chart",
    promptInput: "dictation-prompt-input",
    scriptPicker: "dictation-script-picker",
    scriptDetails: "dictation-script-details",
    startExperiment: "dictation-start-experiment",
    refreshExperiments: "dictation-refresh-experiments",
    experimentName: "dictation-experiment-name",
    experimentEngines: "dictation-experiment-engines",
    experimentLongForm: "dictation-experiment-long-form",
    experimentSeed: "dictation-experiment-seed",
    experimentTargetDuration: "dictation-experiment-target-duration",
    experimentGapMs: "dictation-experiment-gap-ms",
    experimentTagContains: "dictation-experiment-tag-contains",
    experimentRealtimeRepeats: "dictation-experiment-realtime-repeats",
    experimentLatencyTailSeconds: "dictation-experiment-latency-tail-seconds",
    experimentOverlapMaxWindow: "dictation-experiment-overlap-max-window",
    experimentSweepDurations: "dictation-experiment-sweep-durations",
    experimentDroppedSpanThreshold: "dictation-experiment-dropped-span-threshold",
    experimentChunkMs: "dictation-experiment-chunk-ms",
    experimentOverlapMaxStall: "dictation-experiment-overlap-max-stall",
    experimentOverlapWindow: "dictation-experiment-overlap-window",
    experimentOverlapCommitRuns: "dictation-experiment-overlap-commit-runs",
    experimentVadSilence: "dictation-experiment-vad-silence",
    experimentSpeakerFallback: "dictation-experiment-speaker-fallback",
    experimentAdvanced: "dictation-experiment-advanced",
    clipPicker: "dictation-clip-picker",
    clipPickerCount: "dictation-clip-picker-count",
    clipPickerSelectAll: "dictation-clip-picker-select-all",
    clipPickerClear: "dictation-clip-picker-clear",
    experimentNoiseTypes: "dictation-experiment-noise-types",
    experimentSnrDb: "dictation-experiment-snr-db",
    experimentCompetingVoices: "dictation-experiment-competing-voices",
    experimentSpeakerProfile: "dictation-experiment-speaker-profile",
    experimentLiveProgress: "dictation-experiment-live-progress",
    experimentResults: "dictation-experiment-results",
    experimentConditions: "dictation-experiment-conditions",
    compareExperiments: "dictation-compare-experiments",
    compareResults: "dictation-compare-results",
  },
  wakeWord: {
    label: "wake-word-label",
    threshold: "wake-word-threshold",
    record: "wake-word-record",
    save: "wake-word-save",
    delete: "wake-word-delete",
    sampleCount: "wake-word-sample-count",
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
  speakerAdmin: {
    profileName: defineDynamicSelector({
      description: "Speaker profile display-name cell by profile id",
      testIdPattern: "speaker-profile-name-${id}",
      params: { id: { type: "string" } },
    }),
    profileModel: defineDynamicSelector({
      description: "Speaker profile model cell by profile id",
      testIdPattern: "speaker-profile-model-${id}",
      params: { id: { type: "string" } },
    }),
    clipRow: defineDynamicSelector({
      description: "Enrollment clip row by clip id",
      testIdPattern: "speaker-clip-row-${id}",
      params: { id: { type: "string" } },
    }),
  },
  streamConfig: {
    engineRow: defineDynamicSelector({
      description: "Selectable STT engine row by engine id",
      testIdPattern: "engine-row-${id}",
      params: { id: { type: "string" } },
    }),
    engineSelect: defineDynamicSelector({
      description: "Select-engine button by engine id",
      testIdPattern: "engine-select-${id}",
      params: { id: { type: "string" } },
    }),
  },
  wakeWord: {
    sampleRow: defineDynamicSelector({
      description: "Recorded wake-word sample row by index",
      testIdPattern: "wake-word-sample-${index}",
      params: { index: { type: "string" } },
    }),
  },
  dictationStudio: {
    clipRow: defineDynamicSelector({
      description: "Corpus clip row by clip id",
      testIdPattern: "dictation-clip-row-${id}",
      params: { id: { type: "string" } },
    }),
    clipDelete: defineDynamicSelector({
      description: "Delete-clip button by clip id",
      testIdPattern: "dictation-clip-delete-${id}",
      params: { id: { type: "string" } },
    }),
    evalRow: defineDynamicSelector({
      description: "Eval comparison table row by strategy key",
      testIdPattern: "dictation-eval-row-${strategy}",
      params: { strategy: { type: "string" } },
    }),
    evalClip: defineDynamicSelector({
      description: "Eval per-clip drilldown row by strategy and clip id",
      testIdPattern: "dictation-eval-clip-${strategy}-${clipId}",
      params: { strategy: { type: "string" }, clipId: { type: "string" } },
    }),
    experimentRow: defineDynamicSelector({
      description: "Experiment history row by experiment id",
      testIdPattern: "dictation-experiment-row-${id}",
      params: { id: { type: "string" } },
    }),
    experimentWait: defineDynamicSelector({
      description: "Wait button by experiment id",
      testIdPattern: "dictation-experiment-wait-${id}",
      params: { id: { type: "string" } },
    }),
    experimentReport: defineDynamicSelector({
      description: "Report button by experiment id",
      testIdPattern: "dictation-experiment-report-${id}",
      params: { id: { type: "string" } },
    }),
    experimentCancel: defineDynamicSelector({
      description: "Cancel button by experiment id",
      testIdPattern: "dictation-experiment-cancel-${id}",
      params: { id: { type: "string" } },
    }),
    experimentCancelConfirm: defineDynamicSelector({
      description: "Confirm-cancel button by experiment id",
      testIdPattern: "dictation-experiment-cancel-confirm-${id}",
      params: { id: { type: "string" } },
    }),
    experimentCancelDismiss: defineDynamicSelector({
      description: "Dismiss-cancel button by experiment id",
      testIdPattern: "dictation-experiment-cancel-dismiss-${id}",
      params: { id: { type: "string" } },
    }),
    experimentCompare: defineDynamicSelector({
      description: "Compare-selection checkbox by experiment id",
      testIdPattern: "dictation-experiment-compare-${id}",
      params: { id: { type: "string" } },
    }),
    clipPick: defineDynamicSelector({
      description: "Corpus clip-picker checkbox by clip id",
      testIdPattern: "dictation-clip-pick-${id}",
      params: { id: { type: "string" } },
    }),
  },
} satisfies DynamicSelectorTree;

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
