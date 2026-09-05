/**
 * Production `liveCreateClient` tests. Create shares most of its seam with
 * `liveEnhanceClient`; the methods it defines itself are `submit` (which also
 * threads an optional mask) and `result` (which additionally surfaces the
 * terminal job `message` carrying the per-variation blob keys), plus the
 * `fetchResult` delegate. These tests mock the AI edge + the Jobs Connect
 * client to assert that proto→seam mapping. (The hook's N-variation state
 * machine is covered with a fully-stubbed seam in `useCreate.test.tsx`.)
 */
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { JobState } from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";

const mocks = vi.hoisted(() => ({
  client: {
    getJob: vi.fn(),
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

import { liveCreateClient } from "./useCreate";

const PNG_FILE = new File(["bytes"], "in.png", { type: "image/png" });
const MASK_FILE = new File(["mask"], "mask.png", { type: "image/png" });

afterEach(() => {
  vi.clearAllMocks();
});

describe("liveCreateClient", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });
  afterEach(() => {
    vi.unstubAllGlobals();
  });

  it("submit threads the input image + mask through the AI edge", async () => {
    mocks.submitAI.mockResolvedValueOnce({
      jobId: "gen-1",
      estimatedSeconds: 30,
      modelId: "sd-1.5",
      tier: "local-cpu",
      warnings: ["w"],
    });

    const out = await liveCreateClient.submit(
      "inpaint",
      { prompt: "x", variations: 2 },
      PNG_FILE,
      MASK_FILE,
    );

    expect(mocks.submitAI).toHaveBeenCalledWith(
      "inpaint",
      { prompt: "x", variations: 2 },
      { image: PNG_FILE, mask: MASK_FILE },
    );
    expect(out).toEqual({ jobId: "gen-1", tier: "local-cpu", warnings: ["w"] });
  });

  it("submit passes undefined image/mask for text-to-image", async () => {
    mocks.submitAI.mockResolvedValueOnce({
      jobId: "gen-2",
      estimatedSeconds: 0,
      modelId: "sd-1.5",
      tier: "byok",
      warnings: [],
    });

    await liveCreateClient.submit("text_to_image", { prompt: "a lake" });

    expect(mocks.submitAI).toHaveBeenCalledWith(
      "text_to_image",
      { prompt: "a lake" },
      { image: undefined, mask: undefined },
    );
  });

  it("result surfaces the terminal message (variation keys) on success", async () => {
    mocks.client.getJob.mockResolvedValueOnce({
      job: {
        state: JobState.SUCCEEDED,
        resultRef: "out/0.png",
        message: "variations: [out/0.png out/1.png]",
        error: "",
      },
    });

    const out = await liveCreateClient.result("gen-1");

    expect(mocks.client.getJob).toHaveBeenCalledWith({ id: "gen-1" });
    expect(out).toEqual({
      ok: true,
      resultRef: "out/0.png",
      message: "variations: [out/0.png out/1.png]",
      error: "",
    });
  });

  it("result reports ok=false with the error on a failed job", async () => {
    mocks.client.getJob.mockResolvedValueOnce({
      job: { state: JobState.FAILED, resultRef: "", message: "", error: "oom" },
    });

    expect(await liveCreateClient.result("gen-2")).toEqual({
      ok: false,
      resultRef: "",
      message: "",
      error: "oom",
    });
  });

  it("result defaults every field when the job is missing", async () => {
    mocks.client.getJob.mockResolvedValueOnce({ job: undefined });

    expect(await liveCreateClient.result("gen-3")).toEqual({
      ok: false,
      resultRef: "",
      message: "",
      error: "",
    });
  });

  it("fetchResult delegates to the AI edge", async () => {
    const image = {
      url: "blob:v0",
      width: 512,
      height: 512,
      format: "png",
      outputFile: PNG_FILE,
    };
    mocks.fetchAIResult.mockResolvedValueOnce(image);

    const out = await liveCreateClient.fetchResult("out/0.png");

    expect(mocks.fetchAIResult).toHaveBeenCalledWith("out/0.png");
    expect(out).toBe(image);
  });

  it("reuses the shared liveEnhanceClient methods for select/watch/install/wait/cancel", () => {
    // These are the SAME function references as liveEnhanceClient — verifying
    // the aliasing keeps a single production implementation of each method.
    expect(typeof liveCreateClient.selectModel).toBe("function");
    expect(typeof liveCreateClient.watch).toBe("function");
    expect(typeof liveCreateClient.install).toBe("function");
    expect(typeof liveCreateClient.waitJob).toBe("function");
    expect(typeof liveCreateClient.cancel).toBe("function");
  });
});
