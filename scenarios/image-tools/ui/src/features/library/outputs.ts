import { timestampDate } from "@bufbuild/protobuf/wkt";

import { JobState, type Job } from "../../api/jobs";

/**
 * A library output — a succeeded job that produced a result blob. Derived from
 * `JobsService.ListJobs` (no separate outputs backend, per the Stage-4 contract
 * decision); the result image is fetched via the blob-serve edge.
 */
export interface OutputItem {
  jobId: string;
  operation: string;
  resultRef: string;
  createdAtMs: number;
}

/**
 * Succeeded jobs that produced a result blob, newest first. A non-image
 * resultRef (e.g. an analysis job) simply fails to render as a thumbnail and is
 * hidden on the `<img>` error path — so no output-kind metadata is needed.
 */
export function imageOutputs(jobs: readonly Job[]): OutputItem[] {
  return jobs
    .filter((job) => job.state === JobState.SUCCEEDED && job.resultRef !== "")
    .map((job) => ({
      jobId: job.id,
      operation: job.operation,
      resultRef: job.resultRef,
      createdAtMs: job.createdAt ? timestampDate(job.createdAt).getTime() : 0,
    }))
    .sort((a, b) => b.createdAtMs - a.createdAtMs);
}
