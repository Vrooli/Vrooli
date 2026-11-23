import { createClient } from "@connectrpc/connect";
import {
  JobLane,
  JobQueue,
  JobsService,
  JobState,
  type Job,
  type JobRating,
  type ListJobsResponse,
  type CancelJobResponse,
  type ProgressEvent,
} from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";

import { transport } from "./client";

export const jobsClient = createClient(JobsService, transport);

/**
 * WatchJob is a server-streaming RPC backing SSE-style live progress. The
 * Connect-Web transport exposes it as an async iterable of ProgressEvent;
 * the server replays the latest known event first, then streams updates
 * until the job reaches a terminal state. `useJobProgress` subscribes to it.
 */
export { JobLane, JobQueue, JobState };
export type { Job, JobRating, ListJobsResponse, CancelJobResponse, ProgressEvent };

/** Highest rating an operator can give; the lowest is 1. Mirrors the API contract. */
export const MAX_RATING = 5;

/** The longest note the API accepts. */
export const MAX_RATING_NOTE = 1000;

/**
 * Who the UI records as the rater. The API trusts the local caller, so this is
 * a label for the record (the CLI sends the agent or user name instead).
 */
export const RATING_ACTOR = "operator";

/** rateJob stores (or replaces) the operator's rating of a succeeded job. */
export async function rateJob(jobId: string, score: number, note: string): Promise<JobRating | undefined> {
  const resp = await jobsClient.rateJob({ jobId, score, note: note.trim(), actor: RATING_ACTOR });
  return resp.rating;
}

/** clearJobRating removes a job's rating. */
export async function clearJobRating(jobId: string): Promise<void> {
  await jobsClient.clearJobRating({ jobId });
}
