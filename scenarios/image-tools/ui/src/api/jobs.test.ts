import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    getJob: vi.fn(),
    waitJob: vi.fn(),
    listJobs: vi.fn(),
    cancelJob: vi.fn(),
  },
}));

vi.mock("@connectrpc/connect", () => ({
  createClient: vi.fn(() => mocks.client),
}));

describe("api/jobs", () => {
  beforeEach(() => {
    vi.stubGlobal("fetch", vi.fn());
  });

  afterEach(() => {
    vi.clearAllMocks();
    vi.unstubAllGlobals();
  });

  it("exports the generated Connect client and forwards listJobs", async () => {
    const { jobsClient } = await import("./jobs");

    await jobsClient.listJobs({ limit: 10 });

    expect(mocks.client.listJobs).toHaveBeenCalledWith({ limit: 10 });
  });

  it("forwards cancelJob with the target job id", async () => {
    const { jobsClient } = await import("./jobs");

    await jobsClient.cancelJob({ id: "job-1" });

    expect(mocks.client.cancelJob).toHaveBeenCalledWith({ id: "job-1" });
  });

  it("re-exports the JobState and JobLane enums for callers", async () => {
    const { JobState, JobLane } = await import("./jobs");

    expect(JobState.SUCCEEDED).toBeDefined();
    expect(JobLane.GPU).toBeDefined();
  });
});
