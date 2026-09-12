/**
 * Production `liveEnhanceClient` tests. The default `EnhanceClient` seam wraps
 * the AI submit edge (`submitAI`/`fetchAIResult`) plus the Jobs and Models
 * Connect clients. These tests mock those boundaries so the proto→seam field
 * mapping — and the `jobStateName` enum normalization driving each progress
 * tick — runs without a network. (The hook's own state machine is covered with
 * a fully-stubbed seam in `useEnhance.test.tsx`.)
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { JobState } from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";

// One shared Connect client fake stands in for both jobsClient and modelsClient
// (createClient returns it for every service). JobState stays the real enum.
const mocks = vi.hoisted(() => ({
  client: {
    selectModel: vi.fn(),
    installModel: vi.fn(),
    getJob: vi.fn(),
    waitJob: vi.fn(),
    watchJob: vi.fn(),
    cancelJob: vi.fn(),
  },
  submitAI: vi.fn(),
  fetchAIResult: vi.fn(),
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

vi.mock("../../api/ai", async (importActual) => {
  const actual = await importActual<typeof import("../../api/ai")>();
  return { ...actual, submitAI: mocks.submitAI, fetchAIResult: mocks.fetchAIResult };
});

import { liveEnhanceClient } from "./useEnhance";

const PNG_FILE = new File(["bytes"], "in.png", { type: "image/png" });

afterEach(() => {
  vi.clearAllMocks();
});

describe("liveEnhanceClient", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("selectModel maps the SelectModel response + hardware facts into SelectedModel", async () => {
    mocks.client.selectModel.mockResolvedValueOnce({
      model: {
        id: "rembg",
        name: "RemBG",
        sizeMbApprox: 45,
        install: { installed: true },
        hardware: {
          cpuCapable: true,
          gpuRequired: false,
          minVramGb: 0,
          speedNote: "~30s",
        },
      },
      gpuViable: true,
      reason: "CPU default",
      warnings: ["heads up"],
    });

    const out = await liveEnhanceClient.selectModel("background_removal");

    expect(mocks.client.selectModel).toHaveBeenCalledWith({ operation: "background_removal" });
    expect(out).toEqual({
      id: "rembg",
      name: "RemBG",
      installed: true,
      sizeMb: 45,
      cpuCapable: true,
      gpuRequired: false,
      minVramGb: 0,
      speedNote: "~30s",
      gpuViable: true,
      reason: "CPU default",
      warnings: ["heads up"],
    });
  });

  it("selectModel falls back to safe defaults when the model/hardware is absent", async () => {
    mocks.client.selectModel.mockResolvedValueOnce({
      model: undefined,
      gpuViable: false,
      reason: "",
      warnings: [],
    });

    const out = await liveEnhanceClient.selectModel("upscale");

    expect(out).toEqual({
      id: "",
      name: "",
      installed: false,
      sizeMb: 0,
      cpuCapable: false,
      gpuRequired: false,
      minVramGb: 0,
      speedNote: "",
      gpuViable: false,
      reason: "",
      warnings: [],
    });
  });

  it("submit forwards to the AI edge and projects the job id/tier/warnings", async () => {
    mocks.submitAI.mockResolvedValueOnce({
      jobId: "job-9",
      estimatedSeconds: 12,
      modelId: "rembg",
      tier: "local-cpu",
      warnings: ["slow on CPU"],
    });

    const out = await liveEnhanceClient.submit("upscale", { scale: 2 }, PNG_FILE);

    expect(mocks.submitAI).toHaveBeenCalledWith("upscale", { scale: 2 }, { image: PNG_FILE });
    expect(out).toEqual({ jobId: "job-9", tier: "local-cpu", warnings: ["slow on CPU"] });
  });

  it("watch streams ProgressEvents through onEvent with normalized state names", async () => {
    mocks.client.watchJob.mockReturnValueOnce(
      (async function* () {
        await Promise.resolve();
        yield { progress: 10, message: "queued", state: JobState.QUEUED };
        yield { progress: 50, message: "running", state: JobState.RUNNING };
        yield { progress: 100, message: "done", state: JobState.SUCCEEDED };
      })(),
    );
    const events: Array<{ percent: number; message: string; state: string }> = [];
    const controller = new AbortController();

    await liveEnhanceClient.watch("job-1", controller.signal, (e) => events.push(e));

    expect(mocks.client.watchJob).toHaveBeenCalledWith(
      { id: "job-1" },
      { signal: controller.signal },
    );
    expect(events).toEqual([
      { percent: 10, message: "queued", state: "queued" },
      { percent: 50, message: "running", state: "running" },
      { percent: 100, message: "done", state: "succeeded" },
    ]);
  });

  it("watch normalizes the failed/canceled/unspecified enum values", async () => {
    mocks.client.watchJob.mockReturnValueOnce(
      (async function* () {
        await Promise.resolve();
        yield { progress: 1, message: "", state: JobState.FAILED };
        yield { progress: 2, message: "", state: JobState.CANCELED };
        yield { progress: 3, message: "", state: JobState.UNSPECIFIED };
      })(),
    );
    const states: string[] = [];

    await liveEnhanceClient.watch("job-2", new AbortController().signal, (e) =>
      states.push(e.state),
    );

    expect(states).toEqual(["failed", "canceled", "unspecified"]);
  });

  it("watch stops emitting once the signal is aborted mid-stream", async () => {
    const controller = new AbortController();
    mocks.client.watchJob.mockReturnValueOnce(
      (async function* () {
        await Promise.resolve();
        yield { progress: 10, message: "a", state: JobState.RUNNING };
        controller.abort();
        yield { progress: 90, message: "b", state: JobState.RUNNING };
      })(),
    );
    const events: number[] = [];

    await liveEnhanceClient.watch("job-3", controller.signal, (e) => events.push(e.percent));

    expect(events).toEqual([10]);
  });

  it("watch swallows a thrown stream (cancel/unmount) without rejecting", async () => {
    mocks.client.watchJob.mockReturnValueOnce(
      (async function* () {
        await Promise.resolve();
        yield { progress: 5, message: "", state: JobState.RUNNING };
        throw new Error("stream aborted");
      })(),
    );
    const events: number[] = [];

    await expect(
      liveEnhanceClient.watch("job-4", new AbortController().signal, (e) =>
        events.push(e.percent),
      ),
    ).resolves.toBeUndefined();
    expect(events).toEqual([5]);
  });

  it("result reports ok=true with the result ref on a SUCCEEDED job", async () => {
    mocks.client.getJob.mockResolvedValueOnce({
      job: { state: JobState.SUCCEEDED, resultRef: "out/ok.png", error: "" },
    });

    const out = await liveEnhanceClient.result("job-1");

    expect(mocks.client.getJob).toHaveBeenCalledWith({ id: "job-1" });
    expect(out).toEqual({ ok: true, resultRef: "out/ok.png", error: "" });
  });

  it("result reports ok=false and surfaces the error for a non-succeeded job", async () => {
    mocks.client.getJob.mockResolvedValueOnce({
      job: { state: JobState.FAILED, resultRef: "", error: "backend exploded" },
    });

    const out = await liveEnhanceClient.result("job-2");

    expect(out).toEqual({ ok: false, resultRef: "", error: "backend exploded" });
  });

  it("result defaults to empty fields when the job is missing", async () => {
    mocks.client.getJob.mockResolvedValueOnce({ job: undefined });

    const out = await liveEnhanceClient.result("job-3");

    expect(out).toEqual({ ok: false, resultRef: "", error: "" });
  });

  it("fetchResult delegates to the AI edge", async () => {
    const image = {
      url: "blob:x",
      width: 200,
      height: 100,
      format: "png",
      outputFile: PNG_FILE,
    };
    mocks.fetchAIResult.mockResolvedValueOnce(image);

    const out = await liveEnhanceClient.fetchResult("out/ok.png");

    expect(mocks.fetchAIResult).toHaveBeenCalledWith("out/ok.png");
    expect(out).toBe(image);
  });

  it("install forwards the model id and returns the install job descriptor", async () => {
    mocks.client.installModel.mockResolvedValueOnce({
      jobId: "install-1",
      alreadyInstalled: false,
    });

    const out = await liveEnhanceClient.install("rembg");

    expect(mocks.client.installModel).toHaveBeenCalledWith({ id: "rembg" });
    expect(out).toEqual({ jobId: "install-1", alreadyInstalled: false });
  });

  it("waitJob maps a terminal SUCCEEDED job to ok=true", async () => {
    mocks.client.waitJob.mockResolvedValueOnce({
      job: { state: JobState.SUCCEEDED, error: "" },
    });

    const out = await liveEnhanceClient.waitJob("install-1");

    expect(mocks.client.waitJob).toHaveBeenCalledWith({ id: "install-1" });
    expect(out).toEqual({ ok: true, error: "" });
  });

  it("waitJob maps a failed/missing job to ok=false with its error", async () => {
    mocks.client.waitJob.mockResolvedValueOnce({
      job: { state: JobState.FAILED, error: "no disk" },
    });
    expect(await liveEnhanceClient.waitJob("install-2")).toEqual({ ok: false, error: "no disk" });

    mocks.client.waitJob.mockResolvedValueOnce({ job: undefined });
    expect(await liveEnhanceClient.waitJob("install-3")).toEqual({ ok: false, error: "" });
  });

  it("cancel forwards the job id to CancelJob", async () => {
    mocks.client.cancelJob.mockResolvedValueOnce({});

    await liveEnhanceClient.cancel("job-1");

    expect(mocks.client.cancelJob).toHaveBeenCalledWith({ id: "job-1" });
  });
});
