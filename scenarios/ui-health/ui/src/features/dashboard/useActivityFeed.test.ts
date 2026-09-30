import { describe, expect, it } from "vitest";

import { mergeActivity } from "./useActivityFeed";
import type { RecentRun } from "../validation/useValidation";
import type { TrackedJob } from "../reindex/useReindexJobs";

const run = (overrides: Partial<RecentRun> = {}): RecentRun => ({
  scenario: "ui-health",
  passed: true,
  errors: 0,
  warnings: 0,
  infos: 0,
  ranAt: "2026-05-20T10:00:00.000Z",
  ...overrides,
});

const job = (overrides: Partial<TrackedJob> = {}): TrackedJob => ({
  jobId: "job-1",
  scenario: "ui-health",
  dryRun: false,
  triggeredAt: "2026-05-20T11:00:00.000Z",
  plannedUpserts: 0,
  plannedDeletes: 0,
  ...overrides,
});

describe("mergeActivity", () => {
  it("interleaves runs and jobs sorted newest-first", () => {
    const out = mergeActivity(
      [run({ ranAt: "2026-05-20T09:00:00.000Z", scenario: "a" })],
      [job({ jobId: "j-late", triggeredAt: "2026-05-20T12:00:00.000Z" })],
    );
    expect(out.map((x) => x.kind)).toEqual(["reindex", "validation"]);
  });

  it("respects the limit", () => {
    const runs = Array.from({ length: 5 }, (_, i) =>
      run({ ranAt: `2026-05-${10 + i}T10:00:00.000Z`, scenario: `s${i}` }),
    );
    const jobs = Array.from({ length: 5 }, (_, i) =>
      job({ jobId: `j${i}`, triggeredAt: `2026-05-${15 + i}T10:00:00.000Z` }),
    );
    expect(mergeActivity(runs, jobs, 3)).toHaveLength(3);
  });

  it("returns an empty array when both inputs are empty", () => {
    expect(mergeActivity([], [])).toEqual([]);
  });
});
