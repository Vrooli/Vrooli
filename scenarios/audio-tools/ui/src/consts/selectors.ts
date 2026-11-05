import { librarySelectors } from "./selectors.library";
export { librarySelectors };
/** Application selector definitions. Shared behavior lives in @vrooli/ui-selectors.
 * Run selector:manifest after editing these maps; UI builds regenerate the manifest.
 */
import { LOCALE_CODES } from "../i18n/locales";

import { createSelectorRegistry, defineDynamicSelector, type LiteralSelectorTree, type DynamicSelectorTree } from "@vrooli/ui-selectors";
export { createSelectorRegistry, defineDynamicSelector } from "@vrooli/ui-selectors";

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
    interimTranscript: "dictation-interim-transcript",
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

const registry = createSelectorRegistry(literalSelectors, dynamicSelectorDefinitions, librarySelectors);

export const selectors = registry.selectors;
export type Selectors = typeof selectors;
export const selectorsManifest = registry.manifest;
