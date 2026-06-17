/**
 * Test data factories for the jobs domain. Co-located with the feature
 * so deleting `features/jobs/` takes the factories with it (no central
 * residue).
 *
 * Each `make<Domain>(overrides?)` returns a stable default instance that
 * tests selectively override via `MessageInitShape<Schema>`.
 *
 * # Wire shape lives in proto, not here
 *
 * The Job / ListJobsResponse types are GENERATED proto messages at
 * `packages/proto/gen/typescript/image-tools/v1/jobs/...`. Factories use
 * `create(<Schema>, overrides)` so the runtime instance includes proto's
 * internal `$typeName` / reflection state, field defaults match proto3
 * semantics, and adding a field to the schema makes it instantly
 * available without editing this file.
 */
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import { timestampFromDate } from "@bufbuild/protobuf/wkt";
import {
  JobLane,
  JobSchema,
  JobState,
  ListJobsResponseSchema,
  CancelJobResponseSchema,
  ProgressEventSchema,
  type Job,
  type ListJobsResponse,
  type CancelJobResponse,
  type ProgressEvent,
} from "@vrooli/proto-types/image-tools/v1/jobs/jobs_pb";

export type { Job, ListJobsResponse, CancelJobResponse, ProgressEvent };

export const makeProgressEvent = (
  overrides: MessageInitShape<typeof ProgressEventSchema> = {},
): ProgressEvent =>
  create(ProgressEventSchema, {
    jobId: "job-1",
    state: JobState.RUNNING,
    progress: 60,
    message: "Working…",
    ...overrides,
  });

/** Wrap a list of progress events as the async iterable WatchJob returns. */
export async function* asProgressStream(
  events: ProgressEvent[],
): AsyncGenerator<ProgressEvent> {
  for (const ev of events) {
    await Promise.resolve();
    yield ev;
  }
}

export const makeJob = (overrides: MessageInitShape<typeof JobSchema> = {}): Job =>
  create(JobSchema, {
    id: "job-1",
    operation: "upscale",
    lane: JobLane.GPU,
    state: JobState.RUNNING,
    progress: 42,
    message: "Working…",
    error: "",
    resultRef: "",
    estimatedSeconds: 30,
    createdAt: timestampFromDate(new Date("2026-01-01T00:00:00.000Z")),
    startedAt: timestampFromDate(new Date("2026-01-01T00:00:05.000Z")),
    ...overrides,
  });

export const makeListJobsResponse = (
  overrides: MessageInitShape<typeof ListJobsResponseSchema> = {},
): ListJobsResponse =>
  create(ListJobsResponseSchema, {
    jobs: [],
    ...overrides,
  });

export const makeCancelJobResponse = (
  overrides: MessageInitShape<typeof CancelJobResponseSchema> = {},
): CancelJobResponse =>
  create(CancelJobResponseSchema, {
    job: makeJob({ state: JobState.CANCELED }),
    ...overrides,
  });
