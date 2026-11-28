import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  client: {
    getJob: vi.fn(),
    waitJob: vi.fn(),
    listJobs: vi.fn(),
    cancelJob: vi.fn(),
    rateJob: vi.fn(),
    clearJobRating: vi.fn(),
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

  it("re-exports the job lifecycle, legacy lane, and scheduling queue enums", async () => {
    const { JobState, JobLane, JobQueue } = await import("./jobs");

    expect(JobState.SUCCEEDED).toBeDefined();
    expect(JobLane.GPU).toBeDefined();
    expect(JobQueue.LOCAL_SERIAL).toBeDefined();
  });

  it("rateJob records the operator as the rater, trims the note and returns the stored rating", async () => {
    mocks.client.rateJob.mockResolvedValue({ rating: { jobId: "job-1", score: 4 } });
    const { rateJob } = await import("./jobs");

    const rating = await rateJob("job-1", 4, "  kept the face  ");

    expect(mocks.client.rateJob).toHaveBeenCalledWith({ jobId: "job-1", score: 4, note: "kept the face", actor: "operator" });
    expect(rating?.score).toBe(4);
  });

  it("clearJobRating removes the rating of the given job", async () => {
    mocks.client.clearJobRating.mockResolvedValue({});
    const { clearJobRating } = await import("./jobs");

    await clearJobRating("job-1");

    expect(mocks.client.clearJobRating).toHaveBeenCalledWith({ jobId: "job-1" });
  });
});
