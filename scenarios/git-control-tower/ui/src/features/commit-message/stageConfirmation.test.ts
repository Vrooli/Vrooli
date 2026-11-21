import { describe, expect, it } from "vitest";
import { checkStageConfirmation, isSameQueuedSelection, removeQueueSelectionIfUnchanged } from "./stageConfirmation";
import type { QueuedStageSelection } from "./stageQueue";

const queued = (overrides: Partial<QueuedStageSelection> = {}): QueuedStageSelection => ({
  id: "entry-1", snapshotDigest: "snapshot-a", paths: ["src/a.go"], queuedAt: "now", state: "queued", ...overrides,
});

describe("staged-file confirmation contract", () => {
  it("rejects a confirmation captured for a repository that is no longer active", () => {
    expect(checkStageConfirmation("repo-a", "repo-b", [], [])).toBe("stale-repository");
  });

  it("requires a new confirmation when an overlapping queue entry is added or replaced", () => {
    const confirmed = [queued()];
    expect(checkStageConfirmation("repo-a", "repo-a", confirmed, [queued(), queued({ id: "entry-2" })])).toBe("queue-changed");
    expect(checkStageConfirmation("repo-a", "repo-a", confirmed, [queued({ snapshotDigest: "snapshot-b" })])).toBe("queue-changed");
    expect(isSameQueuedSelection(queued(), queued({ snapshotDigest: "snapshot-b" }))).toBe(false);
    expect(isSameQueuedSelection(queued(), queued({ requestKey: "replacement-request" }))).toBe(false);
    expect(isSameQueuedSelection(queued(), queued({ state: "staged" }))).toBe(false);
    expect(checkStageConfirmation("repo-a", "repo-a", confirmed, [queued()])).toBe("confirmed");
  });

  it("does not remove a same-id replacement after a revision conflict", async () => {
    let reads = 0;
    const remove = async () => {
      throw new Error("revision changed");
    };
    const removed = await removeQueueSelectionIfUnchanged(queued(), async () => {
      reads++;
      return { revision: BigInt(reads), entries: [queued(reads === 1 ? {} : { requestKey: "new-request", snapshotDigest: "snapshot-b" })] };
    }, remove);
    expect(removed).toBe(false);
    expect(reads).toBe(2);
  });
});
