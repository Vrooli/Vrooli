/**
 * Self-tests for the jobs-domain proto-typed test factories. Co-located
 * with the feature so deleting `features/jobs/` takes them along.
 *
 * Same contract as the central `test-utils/factories.test.ts`:
 *
 *   - sane defaults make the most common test path `makeX()` no-args
 *   - overrides merge field-level (no all-or-nothing replacement)
 *   - the returned instance round-trips through proto's
 *     `toJson` / `fromJson` byte-identically
 */
import { fromJson, toJson } from "@bufbuild/protobuf";
import { timestampDate } from "@bufbuild/protobuf/wkt";
import { describe, expect, it } from "vitest";

import {
  JobLane,
  JobSchema,
  JobState,
  type Job,
} from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";

import { makeJob, makeListJobsResponse } from "./factories";

describe("makeJob", () => {
  it("returns a job with non-empty id/operation and a parseable created_at", () => {
    const j = makeJob();
    expect(j.id).not.toBe("");
    expect(j.operation).not.toBe("");
    expect(j.lane).toBe(JobLane.GPU);
    expect(j.state).toBe(JobState.RUNNING);
    if (!j.createdAt) {
      throw new Error("factory did not populate created_at");
    }
    expect(Number.isNaN(timestampDate(j.createdAt).getTime())).toBe(false);
  });

  it("merges overrides without dropping defaults", () => {
    const j = makeJob({ id: "custom-1", state: JobState.SUCCEEDED });
    expect(j.id).toBe("custom-1");
    expect(j.state).toBe(JobState.SUCCEEDED);
    expect(j.operation).not.toBe("");
  });

  it("round-trips through JobSchema JSON encode + decode", () => {
    const original = makeJob({ id: "rt-1", progress: 77 });
    const decoded = fromJson(JobSchema, toJson(JobSchema, original));
    expect(decoded.id).toBe("rt-1");
    expect(decoded.progress).toBe(77);
    expect(decoded.createdAt).toEqual(original.createdAt);
  });
});

describe("makeListJobsResponse", () => {
  it("defaults to an empty jobs array (proto3: arrays default to [], not undefined)", () => {
    const r = makeListJobsResponse();
    expect(r.jobs).toEqual([]);
  });

  it("accepts job overrides", () => {
    const r = makeListJobsResponse({ jobs: [makeJob({ id: "a" }), makeJob({ id: "b" })] });
    expect(r.jobs.map((j: Job) => j.id)).toEqual(["a", "b"]);
  });
});
