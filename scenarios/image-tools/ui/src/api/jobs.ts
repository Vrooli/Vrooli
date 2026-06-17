import { createClient } from "@connectrpc/connect";
import {
  JobLane,
  JobsService,
  JobState,
  type Job,
  type ListJobsResponse,
  type CancelJobResponse,
} from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";

import { transport } from "./client";

export const jobsClient = createClient(JobsService, transport);

// NOTE: WatchJob is a server-streaming RPC. The streaming/SSE UI is
// deliberately out of scope for now — JobsCard polls ListJobs instead.
// Wire WatchJob through `jobsClient.watchJob(...)` (an async iterable)
// when live progress is added.

export { JobLane, JobState };
export type { Job, ListJobsResponse, CancelJobResponse };
