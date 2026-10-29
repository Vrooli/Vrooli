import assert from "node:assert/strict";
import { test } from "vitest";
import { runStatusCountsFromRows } from "./statsClient.js";

test("status rows fold into the fixed per-status counts, with unknown statuses still in the total", () => {
  const counts = runStatusCountsFromRows([{ status: "running", count: 2 }, { status: "needs_review", count: 1 }, { status: "complete", count: 4 }, { status: "odd", count: 3 }]);
  assert.deepEqual(counts, { pending: 0, running: 2, complete: 4, failed: 0, cancelled: 0, needsReview: 1, total: 10 });
});
