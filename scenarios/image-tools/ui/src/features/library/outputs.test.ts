/**
 * outputs tests — `imageOutputs` derives the library gallery from a ListJobs
 * snapshot: keep only succeeded jobs that produced a result blob, map to the
 * lean OutputItem shape, and sort newest-first by createdAt. The missing-
 * createdAt branch (defaults to 0) is exercised alongside the happy path.
 */
import { describe, expect, it } from "vitest";

import { JobState } from "../../api/jobs";
import { makeJob } from "../jobs/mocks/factories";
import { imageOutputs } from "./outputs";

describe("imageOutputs", () => {
  it("keeps only succeeded jobs that produced a result blob", () => {
    const jobs = [
      makeJob({ id: "ok", state: JobState.SUCCEEDED, resultRef: "out/ok.png" }),
      makeJob({ id: "no-blob", state: JobState.SUCCEEDED, resultRef: "" }),
      makeJob({ id: "running", state: JobState.RUNNING, resultRef: "out/x.png" }),
      makeJob({ id: "failed", state: JobState.FAILED, resultRef: "out/y.png" }),
    ];
    const outputs = imageOutputs(jobs);
    expect(outputs.map((o) => o.jobId)).toEqual(["ok"]);
    expect(outputs[0]?.resultRef).toBe("out/ok.png");
  });

  it("maps to the lean OutputItem shape including a millisecond timestamp", () => {
    const jobs = [makeJob({ id: "j1", state: JobState.SUCCEEDED, resultRef: "out/j1.png", operation: "crop" })];
    const [item] = imageOutputs(jobs);
    expect(item).toEqual({
      jobId: "j1",
      operation: "crop",
      resultRef: "out/j1.png",
      createdAtMs: new Date("2026-01-01T00:00:00.000Z").getTime(),
    });
  });

  it("defaults createdAtMs to 0 when a job has no createdAt timestamp", () => {
    const jobs = [
      makeJob({ id: "no-ts", state: JobState.SUCCEEDED, resultRef: "out/n.png", createdAt: undefined }),
    ];
    const [item] = imageOutputs(jobs);
    expect(item?.createdAtMs).toBe(0);
  });

  it("sorts newest-first by createdAt", () => {
    const old = makeJob({
      id: "old",
      state: JobState.SUCCEEDED,
      resultRef: "out/old.png",
      createdAt: { seconds: 1000n, nanos: 0 },
    });
    const recent = makeJob({
      id: "recent",
      state: JobState.SUCCEEDED,
      resultRef: "out/recent.png",
      createdAt: { seconds: 5000n, nanos: 0 },
    });
    const outputs = imageOutputs([old, recent]);
    expect(outputs.map((o) => o.jobId)).toEqual(["recent", "old"]);
  });
});
