/**
 * Mock builders for `api/jobs` — the UI ↔ API jobs boundary. Co-located
 * with the jobs feature; deleting `features/jobs/` takes these with it.
 *
 * See `test-utils/mocks/api.ts` for the full builder/hoisting rationale.
 * Canonical usage:
 *
 *   import { makeJobsMocks } from "./mocks/jobs";
 *
 *   vi.mock("../../api/jobs", async (importOriginal) => {
 *     const actual = await importOriginal<typeof import("../../api/jobs")>();
 *     return { ...actual, ...makeJobsMocks() };
 *   });
 *
 * The `...actual` spread keeps the re-exported proto types + enums intact —
 * only the network-touching client methods are substituted.
 *
 * Default behaviors:
 *   - `jobsClient.listJobs` resolves to an empty list
 *   - `jobsClient.cancelJob({ id })` echoes a canceled job back
 */
import { vi } from "vitest";

import { makeCancelJobResponse, makeJob, makeListJobsResponse } from "./factories";

export interface JobsMocks {
  jobsClient: {
    listJobs: ReturnType<typeof vi.fn>;
    cancelJob: ReturnType<typeof vi.fn>;
  };
}

export const makeJobsMocks = (): JobsMocks => ({
  jobsClient: {
    listJobs: vi.fn().mockResolvedValue(makeListJobsResponse()),
    cancelJob: vi
      .fn()
      .mockImplementation((input: { id: string }) =>
        Promise.resolve(makeCancelJobResponse({ job: makeJob({ id: input.id }) })),
      ),
  },
});
