import { beforeEach, describe, expect, it, vi } from "vitest";

import { ExperimentStatus, ReplayLane } from "@vrooli/proto-types/audio-tools/v1/experiment/experiment_pb";
import { SpeakerMode } from "@vrooli/proto-types/audio-tools/v1/stt/stt_pb";

const {
  startExperimentRpc,
  listExperimentsRpc,
  getExperimentRpc,
  waitExperimentRpc,
  cancelExperimentRpc,
  getExperimentReportRpc,
  compareExperimentsRpc,
  streamExperimentEventsRpc,
  decodeEvalReport,
} = vi.hoisted(() => ({
  startExperimentRpc: vi.fn(),
  listExperimentsRpc: vi.fn(),
  getExperimentRpc: vi.fn(),
  waitExperimentRpc: vi.fn(),
  cancelExperimentRpc: vi.fn(),
  getExperimentReportRpc: vi.fn(),
  compareExperimentsRpc: vi.fn(),
  streamExperimentEventsRpc: vi.fn(),
  decodeEvalReport: vi.fn((report: unknown) => ({ decoded: report })),
}));

vi.mock("../api/client", () => ({ transport: {} }));
vi.mock("@connectrpc/connect", () => ({
  createClient: () => ({
    startExperiment: (request: unknown) => startExperimentRpc(request),
    listExperiments: (request: unknown) => listExperimentsRpc(request),
    getExperiment: (request: unknown) => getExperimentRpc(request),
    waitExperiment: (request: unknown) => waitExperimentRpc(request),
    cancelExperiment: (request: unknown) => cancelExperimentRpc(request),
    getExperimentReport: (request: unknown) => getExperimentReportRpc(request),
    compareExperiments: (request: unknown) => compareExperimentsRpc(request),
    streamExperimentEvents: (request: unknown, options: unknown) => streamExperimentEventsRpc(request, options),
  }),
}));
vi.mock("./corpus", () => ({ decodeEvalReport }));

import {
  cancelExperiment,
  compareExperiments,
  getExperiment,
  getExperimentReport,
  listExperiments,
  startExperiment,
  streamExperimentEvents,
  waitExperiment,
  type StartExperimentInput,
} from "./experiment";

const input: StartExperimentInput = {
  name: "provider parity",
  clipIds: ["clip-1"],
  engineIds: ["whisper-local", "kyutai"],
  strategies: ["overlap", "passthrough"],
  realtimeRepeats: 2,
  latencyTailSeconds: 4,
  chunkMs: 250,
  seed: 42,
  longForm: false,
  targetDurationSeconds: 60,
  gapMs: 10,
  tagContains: "speech",
  sweepDurationsCsv: "30, sixty, 60.4, -1",
  overlapMaxStallRejects: 3,
  overlapWindowMs: 500,
  overlapCommitRuns: 2,
  overlapMaxWindowMs: 1_000,
  vadSilenceMs: 250,
  noiseTypesCsv: "cafe, office",
  snrDbCsv: "10, invalid, 20.5",
  competingVoicesCsv: "voice-a, voice-b",
  competingText: "competing speech",
  speakerTargetProfileId: "speaker-1",
  speakerExtraction: true,
  speakerVerification: true,
  speakerMode: "filter",
  speakerThreshold: 0.8,
  speakerFallback: false,
  speakerAblation: true,
  droppedSpanThresholdWords: 0,
};

function experiment(status = ExperimentStatus.RUNNING) {
  return {
    id: "experiment-1",
    name: "provider parity",
    status,
    createdAt: { seconds: 1_700_000_000n, nanos: 500_000_000 },
    startedAt: { seconds: 1_700_000_001n },
    finishedAt: { seconds: 1_700_000_002n },
    error: "",
    resultRef: "result-1",
    machineJson: "{\"cpu\":\"test\"}",
    recipe: {
      clipIds: ["clip-1"],
      strategies: [{ kind: "overlap", label: "Overlap", overlapMaxWindowMs: 1_000, overlapMaxStallRejects: 3, overlapWindowMs: 500, overlapCommitRuns: 2, vadSilenceMs: 250 }],
      cells: [{ engineId: "kyutai", strategy: "passthrough", label: "Kyutai", replayLane: ReplayLane.DETERMINISTIC, repeatCount: 1 }],
      realtimeRepeats: 2,
      latencyTailSeconds: 4,
      chunkMs: 250,
      seed: 42n,
      longForm: { enabled: true, targetDurationSeconds: 60, gapMs: 10, tagContains: "speech", sweepDurationsSeconds: [30, 60] },
      realizedClipIds: ["clip-1"],
      realizedReference: "known reference",
      realizedDurationMs: 60_000n,
      augmentation: { noiseTypes: ["cafe"], snrDb: [10], competingVoiceIds: ["voice-a"], competingText: "competing" },
      realizedAugmentationConditions: [{ id: "noise", kind: "noise", source: "fixture", snrDb: 10, skipped: false, note: "applied" }],
      speaker: { targetProfileId: "speaker-1", extractionEnabled: true, verificationEnabled: true, verificationMode: SpeakerMode.FILTER, threshold: 0.8, ablationEnabled: true },
      realizedSpeakerConditions: [{ id: "filter", extractionEnabled: true, verificationEnabled: true, skipped: false, note: "applied" }],
      droppedSpanThresholdWords: 0,
    },
  };
}

beforeEach(() => vi.clearAllMocks());

describe("experiment service", () => {
  it("[REQ:ATD-P1-001] builds a provider-aware recipe and decodes its persisted evidence", async () => {
    startExperimentRpc.mockResolvedValue({ experiment: experiment() });

    const row = await startExperiment(input);
    const request = startExperimentRpc.mock.calls[0]![0];
    expect(request.recipe.strategies).toEqual([]);
    expect(request.recipe.cells).toEqual([
      { engineId: "whisper-local", strategy: "overlap", label: "whisper-local:overlap", replayLane: ReplayLane.REALTIME, repeatCount: 2 },
      { engineId: "whisper-local", strategy: "passthrough", label: "whisper-local:passthrough", replayLane: ReplayLane.REALTIME, repeatCount: 2 },
      { engineId: "kyutai", strategy: "overlap", label: "kyutai:overlap", replayLane: ReplayLane.REALTIME, repeatCount: 2 },
      { engineId: "kyutai", strategy: "passthrough", label: "kyutai:passthrough", replayLane: ReplayLane.REALTIME, repeatCount: 2 },
    ]);
	    expect(request.recipe.realtimeRepeats).toBe(0);
	    expect(request.recipe.latencyTailSeconds).toBe(0);
    expect(request.recipe.longForm).toMatchObject({ enabled: true, sweepDurationsSeconds: [30, 60] });
    expect(request.recipe.augmentation).toEqual({ noiseTypes: ["cafe", "office"], snrDb: [10, 20.5], competingVoiceIds: ["voice-a", "voice-b"], competingText: "competing speech" });
    expect(request.recipe.speaker).toMatchObject({ verificationMode: SpeakerMode.FILTER, ablationEnabled: true });
    expect(row.status).toBe("running");
    expect(row.createdAt).toBe("2023-11-14T22:13:20.500Z");
    expect(row.recipe.strategies).toEqual(["overlap", "Kyutai"]);
    expect(row.recipe.speakerMode).toBe("filter");
    expect(row.recipe.augmentationConditions[0]).toMatchObject({ id: "noise", skipped: false });
  });

  it("maps all experiment lifecycle endpoints and status values", async () => {
    listExperimentsRpc.mockResolvedValue({ experiments: [
      experiment(ExperimentStatus.QUEUED), experiment(ExperimentStatus.RUNNING), experiment(ExperimentStatus.SUCCEEDED),
      experiment(ExperimentStatus.FAILED), experiment(ExperimentStatus.CANCELED), experiment(ExperimentStatus.UNSPECIFIED),
    ] });
    getExperimentRpc.mockResolvedValue({ experiment: experiment(), runs: [{ id: "run-1", experimentId: "experiment-1", strategy: "overlap", conditionJson: "{}", createdAt: { seconds: 1n } }] });
    waitExperimentRpc.mockResolvedValue({ experiment: undefined, runs: [] });
    cancelExperimentRpc.mockResolvedValue({ experiment: experiment(ExperimentStatus.CANCELED) });
    getExperimentReportRpc.mockResolvedValue({ experiment: experiment(), report: { reportId: "report-1" }, runs: [] });
    compareExperimentsRpc.mockResolvedValue({ experiments: [{ experiment: experiment(), report: { reportId: "report-2" }, runs: [{ id: "run-2", experimentId: "experiment-1", strategy: "passthrough", conditionJson: "{}" }] }] });

    expect((await listExperiments()).map((row) => row.status)).toEqual(["queued", "running", "succeeded", "failed", "canceled", "unspecified"]);
    expect(listExperimentsRpc).toHaveBeenCalledWith({ status: ExperimentStatus.UNSPECIFIED, limit: 50, offset: 0 });
    expect((await getExperiment("experiment-1")).runs[0]).toMatchObject({ id: "run-1", createdAt: "1970-01-01T00:00:01.000Z" });
    expect(await waitExperiment("experiment-1")).toEqual({ experiment: null, runs: [] });
    expect((await cancelExperiment("experiment-1"))?.status).toBe("canceled");
    expect((await getExperimentReport("experiment-1")).report).toEqual({ decoded: { reportId: "report-1" } });
    expect((await compareExperiments(["experiment-1"]))[0]?.runs[0]).toMatchObject({ strategy: "passthrough" });
  });

  it("streams normalized provider-neutral experiment events with the caller signal", async () => {
    async function* events() {
      yield { experimentId: "experiment-1", status: ExperimentStatus.QUEUED, progress: 0.1, message: "queued", at: { seconds: 1n } };
      yield { experimentId: "experiment-1", status: ExperimentStatus.SUCCEEDED, progress: 1, message: "complete" };
    }
    streamExperimentEventsRpc.mockReturnValue(events());
    const controller = new AbortController();
    const observed: unknown[] = [];

    await streamExperimentEvents("experiment-1", (event) => observed.push(event), controller.signal);

    expect(streamExperimentEventsRpc).toHaveBeenCalledWith({ id: "experiment-1" }, { signal: controller.signal });
    expect(observed).toEqual([
      { experimentId: "experiment-1", status: "queued", progress: 0.1, message: "queued", at: "1970-01-01T00:00:01.000Z" },
      { experimentId: "experiment-1", status: "succeeded", progress: 1, message: "complete", at: "" },
    ]);
  });

  it("rejects a malformed start response instead of inventing an experiment", async () => {
    startExperimentRpc.mockResolvedValue({});
    await expect(startExperiment({ ...input, speakerMode: "advisory" })).rejects.toThrow("missing experiment");
  });
});
