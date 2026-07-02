import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import { cleanup, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";

import { renderWithProviders } from "../../test-utils";
import { selectors } from "../../consts/selectors";
import { strings } from "../../consts/strings";

const startExperiment = vi.fn();
const listExperiments = vi.fn();
const waitExperiment = vi.fn();
const cancelExperiment = vi.fn();
const getExperimentReport = vi.fn();
const compareExperiments = vi.fn();
const getExperiment = vi.fn();
const streamExperimentEvents = vi.fn();

vi.mock("../../services/experiment", () => ({
  startExperiment: (input: unknown) => startExperiment(input),
  listExperiments: () => listExperiments(),
  getExperiment: (id: string) => getExperiment(id),
  waitExperiment: (id: string) => waitExperiment(id),
  cancelExperiment: (id: string) => cancelExperiment(id),
  getExperimentReport: (id: string) => getExperimentReport(id),
  compareExperiments: (ids: string[]) => compareExperiments(ids),
  streamExperimentEvents: (id: string, onEvent: (event: unknown) => void, signal?: AbortSignal) =>
    streamExperimentEvents(id, onEvent, signal),
}));

const listClips = vi.fn();
vi.mock("../../services/corpus", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../../services/corpus")>();
  return { ...actual, listClips: (...args: unknown[]) => listClips(...args) };
});

const getSpeakerStatus = vi.fn();
vi.mock("../../services/speakerAdmin", () => ({
  getSpeakerStatus: () => getSpeakerStatus(),
}));

const pushToast = vi.fn();
vi.mock("../../components/ui/toast", () => ({
  pushToast: (...args: unknown[]) => pushToast(...args),
}));

import { ExperimentLabView } from "./ExperimentLabView";

function clip(id: string, over = {}) {
  return {
    id,
    referenceText: `reference for ${id}`,
    tags: ["news"],
    durationMs: 4200,
    sampleRateHz: 16000,
    format: "pcm_s16le",
    source: 0,
    createdAt: "",
    ...over,
  };
}

function experiment(over = {}) {
  return {
    id: "exp-1",
    name: "Long run",
    status: "succeeded",
    createdAt: "",
    startedAt: "",
    finishedAt: "",
    error: "",
    resultRef: "blob",
    machineJson: "",
    recipe: {
      clipIds: [],
      strategies: ["batch", "overlap_agree"],
      strategyDetails: [
        {
          kind: "overlap_agree",
          label: "overlap agree",
          overlapMaxWindowMs: 25000,
          overlapMaxStallRejects: 3,
          overlapWindowMs: 1500,
          overlapCommitRuns: 2,
          vadSilenceMs: 0,
        },
      ],
      realtimeRepeats: 0,
      latencyTailSeconds: 8,
      chunkMs: 100,
      seed: 42,
      longFormEnabled: true,
      targetDurationSeconds: 180,
      gapMs: 5000,
      tagContains: "",
      sweepDurationsSeconds: [],
      realizedClipIds: ["clip-1"],
      realizedDurationMs: 180000,
      realizedReference: "hello world",
      noiseTypes: ["white"],
      snrDb: [12],
      competingVoiceIds: [],
      competingText: "",
      augmentationConditions: [
        { id: "noise-white", kind: "noise", source: "white", snrDb: 12, skipped: false, note: "" },
        { id: "kokoro", kind: "voice", source: "kokoro", snrDb: 0, skipped: true, note: "kokoro down" },
      ],
      speakerTargetProfileId: "",
      speakerExtraction: false,
      speakerVerification: false,
      speakerMode: "filter",
      speakerThreshold: 0.5,
      speakerAblation: false,
      speakerConditions: [
        { id: "speaker-missing", extraction: true, verification: true, skipped: true, note: "profile unavailable" },
      ],
      droppedSpanThresholdWords: 4,
    },
    ...over,
  };
}

function report() {
  return {
    experiment: experiment(),
    runs: [],
    report: {
      perStrategy: [
        {
          strategy: "batch",
          label: "batch",
          wer: 0.1,
          substitutions: 1,
          insertions: 0,
          deletions: 0,
          refWords: 10,
          whisperCalls: 1,
          whisperAudioSeconds: 3,
          rtf: 0.4,
          finalizationLatencyP50Ms: 20,
          finalizationLatencyP95Ms: 30,
          partialRevisions: 0,
          perClip: [],
          werDeltaVsWinner: 0,
          p95DeltaMsVsWinner: 0,
          callMultiplierVsWinner: 1,
          verdict: "winner",
          reasons: [],
          warnings: [],
          safety: {
            passed: true,
            retractionFree: true,
            droppedSpanFree: true,
            maxDroppedSpanWords: 0,
            droppedSpanThresholdWords: 4,
            retractionEvents: [],
            reasons: [],
          },
          stageAttribution: {
            ingressLostWords: 0,
            strategyLostWords: 0,
            egressLostWords: 0,
            egressRejectEvents: 0,
            notes: [],
          },
          lengthCurves: [
            {
              bucket: "3m",
              minDurationMs: 60000,
              maxDurationMs: 180000,
              clipCount: 1,
              wer: 0.1,
              finalizationLatencyP95Ms: 30,
              meanTimeToFirstCommitMs: 10,
              maxDroppedSpanWords: 0,
            },
          ],
          commitCount: 1,
          speakerRejectionCount: 0,
        },
      ],
      qualityMeasured: true,
      latencyMeasured: true,
      summary: {
        winnerStrategy: "batch",
        winnerLabel: "batch",
        recommendation: "Prefer batch.",
        confidence: "medium",
        reasons: [],
        confidenceNotes: [],
      },
      warnings: [],
      normalizationPolicy: null,
      latencyHonesty: "Latency is intra-experiment only.",
    },
  };
}

function unsafeRecommendedReport() {
  const base = report();
  return {
    ...base,
    report: {
      ...base.report,
      perStrategy: [
        {
          ...base.report.perStrategy[0],
          safety: {
            passed: false,
            retractionFree: false,
            droppedSpanFree: false,
            maxDroppedSpanWords: 4,
            droppedSpanThresholdWords: 4,
            retractionEvents: [],
            reasons: ["dropped span threshold exceeded"],
          },
          verdict: "recommended-despite-safety-failure",
          warnings: [
            {
              code: "safety_failed",
              severity: "error",
              message: "Safety gate failed for every evaluated strategy; this winner is diagnostic only.",
            },
          ],
        },
      ],
      summary: {
        ...base.report.summary,
        recommendation: "Prefer batch for this corpus, recommended despite safety failure.",
        confidence: "low",
        confidenceNotes: ["Every evaluated strategy failed the safety gate; treat this only as the least-bad diagnostic result."],
      },
    },
  };
}

beforeEach(() => {
  vi.clearAllMocks();
  listExperiments.mockResolvedValue([experiment()]);
  listClips.mockResolvedValue([clip("clip-1"), clip("clip-2")]);
  startExperiment.mockResolvedValue(experiment({ id: "exp-2", name: "New run", status: "queued" }));
  waitExperiment.mockResolvedValue({ experiment: experiment(), runs: [] });
  cancelExperiment.mockResolvedValue(experiment({ status: "canceled" }));
  getExperimentReport.mockResolvedValue(report());
  getExperiment.mockResolvedValue({ experiment: experiment(), runs: [] });
  getSpeakerStatus.mockResolvedValue({
    config: {},
    capability: "speaker_id",
    capabilityLabel: "Speaker ID",
    resourceReady: true,
    profileConfigured: true,
    profileExists: true,
    profileCount: 1,
    profiles: [
      {
        id: "profile-1",
        displayName: "Alice",
        createdAt: "",
        modelName: "ecapa",
        sampleRate: 16000,
        clipCount: 2,
        totalVoicedSeconds: 8,
      },
    ],
  });
  streamExperimentEvents.mockImplementation(async (id: string, onEvent: (event: unknown) => void) => {
    onEvent({ experimentId: id, status: "running", progress: 45, message: "evaluating strategies", at: "" });
    await new Promise((resolve) => window.setTimeout(resolve, 25));
    onEvent({ experimentId: id, status: "succeeded", progress: 100, message: "storing report", at: "" });
  });
  compareExperiments.mockResolvedValue([report(), { ...report(), experiment: experiment({ id: "exp-2", name: "New run" }) }]);
});
afterEach(cleanup);

describe("ExperimentLabView", () => {
  it("starts an async experiment from builder controls", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExperimentLabView />);

    await user.clear(screen.getByTestId(selectors.dictationStudio.experimentName));
    await user.type(screen.getByTestId(selectors.dictationStudio.experimentName), "Noise sweep");
    await user.type(screen.getByTestId(selectors.dictationStudio.experimentNoiseTypes), "white,fan");
    await user.click(screen.getByTestId(selectors.dictationStudio.startExperiment));

    await waitFor(() =>
      expect(startExperiment).toHaveBeenCalledWith(
        expect.objectContaining({
          name: "Noise sweep",
          longForm: true,
          latencyTailSeconds: 8,
          noiseTypesCsv: "white,fan",
          strategies: expect.arrayContaining(["batch", "vad_segment", "overlap_agree"]),
        }),
      ),
    );
    expect(pushToast).toHaveBeenCalledWith(expect.objectContaining({ title: strings.dictationStudio.experimentStarted }));
    expect(streamExperimentEvents).toHaveBeenCalledWith("exp-2", expect.any(Function), expect.any(AbortSignal));
    await waitFor(() =>
      expect(screen.getByTestId(selectors.dictationStudio.experimentLiveProgress)).toHaveTextContent("storing report"),
    );
    await waitFor(() => expect(getExperimentReport).toHaveBeenCalledWith("exp-2"));
  });

  it("loads history, report, and safety envelope", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExperimentLabView />);

    const row = await screen.findByTestId(selectors.dictationStudio.experimentRow({ id: "exp-1" }));
    expect(within(row).getByText(/Long run/)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.dictationStudio.experimentReport({ id: "exp-1" })));
    expect(await screen.findByTestId(selectors.dictationStudio.experimentResults)).toHaveTextContent(strings.dictationStudio.safetySafe);
    expect(screen.getByTestId(selectors.dictationStudio.lengthCurveChart)).toBeInTheDocument();
    expect(screen.getByText(strings.dictationStudio.realizedClipsLabel)).toBeInTheDocument();
    const conditions = screen.getByTestId(selectors.dictationStudio.experimentConditions);
    expect(conditions).toHaveTextContent("kokoro down");
    expect(conditions).toHaveTextContent("profile unavailable");
    expect(getExperimentReport).toHaveBeenCalledWith("exp-1");
  });

  it("warns when the recommended strategy is unsafe", async () => {
    getExperimentReport.mockResolvedValue(unsafeRecommendedReport());
    const user = userEvent.setup();
    renderWithProviders(<ExperimentLabView />);

    await user.click(await screen.findByTestId(selectors.dictationStudio.experimentReport({ id: "exp-1" })));

    const summary = await screen.findByTestId(selectors.dictationStudio.evalSummary);
    expect(summary).toHaveTextContent("recommended despite safety failure");
    expect(summary).toHaveTextContent("low");
    expect(await screen.findByTestId(selectors.dictationStudio.experimentResults)).toHaveTextContent(strings.dictationStudio.safetyUnsafe);
  });

  it("compares experiments selected from the history list", async () => {
    listExperiments.mockResolvedValue([experiment(), experiment({ id: "exp-2", name: "New run" })]);
    compareExperiments.mockResolvedValue([
      report(),
      {
        ...report(),
        experiment: experiment({
          id: "exp-2",
          name: "New run",
          recipe: {
            ...experiment().recipe,
            strategyDetails: [
              {
                kind: "overlap_agree",
                label: "overlap agree",
                overlapMaxWindowMs: 25000,
                overlapMaxStallRejects: 8,
                overlapWindowMs: 1500,
                overlapCommitRuns: 2,
                vadSilenceMs: 0,
              },
            ],
          },
        }),
        report: {
          ...report().report,
          perStrategy: [
            {
              ...report().report.perStrategy[0],
              strategy: "overlap_agree",
              label: "overlap agree",
              wer: 0.08,
              finalizationLatencyP95Ms: 24,
            },
          ],
          summary: {
            ...report().report.summary,
            winnerStrategy: "overlap_agree",
            winnerLabel: "overlap agree",
          },
        },
      },
    ]);
    const user = userEvent.setup();
    renderWithProviders(<ExperimentLabView />);

    await screen.findByTestId(selectors.dictationStudio.experimentRow({ id: "exp-1" }));

    const compareButton = screen.getByTestId(selectors.dictationStudio.compareExperiments);
    expect(compareButton).toBeDisabled();

    await user.click(screen.getByTestId(selectors.dictationStudio.experimentCompare({ id: "exp-1" })));
    await user.click(screen.getByTestId(selectors.dictationStudio.experimentCompare({ id: "exp-2" })));
    expect(compareButton).not.toBeDisabled();

    await user.click(compareButton);
    expect(compareExperiments).toHaveBeenCalledWith(["exp-1", "exp-2"]);
    const results = await screen.findByTestId(selectors.dictationStudio.compareResults);
    expect(results).toHaveTextContent(strings.dictationStudio.safetySafe);
    expect(results).toHaveTextContent(strings.dictationStudio.compareRecipeDiffTitle);
    expect(results).toHaveTextContent("strategy.overlap_agree.overlap_max_stall_rejects: 3 -> 8");
    expect(results).toHaveTextContent(strings.dictationStudio.compareStrategyAlignmentTitle);
    expect(results).toHaveTextContent("8.0% / 24");
  });

  it("wires the clip picker, dropped-span threshold, and length sweep into the recipe", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExperimentLabView />);

    await user.click(await screen.findByTestId(selectors.dictationStudio.clipPick({ id: "clip-1" })));
    expect(screen.getByTestId(selectors.dictationStudio.clipPickerCount)).toBeInTheDocument();

    const threshold = screen.getByTestId(selectors.dictationStudio.experimentDroppedSpanThreshold);
    await user.clear(threshold);
    await user.type(threshold, "7");

    const sweep = screen.getByTestId(selectors.dictationStudio.experimentSweepDurations);
    await user.type(sweep, "30,60,120");

    await user.click(screen.getByTestId(selectors.dictationStudio.startExperiment));

    await waitFor(() =>
      expect(startExperiment).toHaveBeenCalledWith(
        expect.objectContaining({
          clipIds: ["clip-1"],
          droppedSpanThresholdWords: 7,
          sweepDurationsCsv: "30,60,120",
        }),
      ),
    );
  });

  it("prevents a no-input clip run and uses the speaker profile picker", async () => {
    const user = userEvent.setup();
    renderWithProviders(<ExperimentLabView />);

    await user.click(screen.getByTestId(selectors.dictationStudio.experimentLongForm));
    const startButton = screen.getByTestId(selectors.dictationStudio.startExperiment);
    expect(startButton).toBeDisabled();
    expect(screen.getByText(strings.dictationStudio.startInputRequired)).toBeInTheDocument();

    await user.click(screen.getByTestId(selectors.dictationStudio.experimentLongForm));
    await screen.findByRole("option", { name: /Alice/ });
    await user.selectOptions(screen.getByTestId(selectors.dictationStudio.experimentSpeakerProfile), "profile-1");
    await user.click(startButton);

    await waitFor(() =>
      expect(startExperiment).toHaveBeenCalledWith(expect.objectContaining({ speakerTargetProfileId: "profile-1" })),
    );
  });

  it("requires confirmation before canceling a running experiment", async () => {
    listExperiments.mockResolvedValue([experiment({ id: "exp-run", name: "Running run", status: "running" })]);
    const user = userEvent.setup();
    renderWithProviders(<ExperimentLabView />);

    await screen.findByTestId(selectors.dictationStudio.experimentRow({ id: "exp-run" }));

    await user.click(screen.getByTestId(selectors.dictationStudio.experimentCancel({ id: "exp-run" })));
    expect(cancelExperiment).not.toHaveBeenCalled();

    // Dismissing keeps the run untouched.
    await user.click(screen.getByTestId(selectors.dictationStudio.experimentCancelDismiss({ id: "exp-run" })));
    expect(cancelExperiment).not.toHaveBeenCalled();

    // Confirming actually cancels.
    await user.click(screen.getByTestId(selectors.dictationStudio.experimentCancel({ id: "exp-run" })));
    await user.click(screen.getByTestId(selectors.dictationStudio.experimentCancelConfirm({ id: "exp-run" })));
    expect(cancelExperiment).toHaveBeenCalledWith("exp-run");
  });
});
