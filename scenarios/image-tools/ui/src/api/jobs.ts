import { createClient } from "@connectrpc/connect";
import {
  JobLane,
  JobsService,
  JobState,
  type Job,
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
export { JobLane, JobState };
export type { Job, ListJobsResponse, CancelJobResponse, ProgressEvent };
