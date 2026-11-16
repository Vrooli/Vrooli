import { describe, expect, it } from "vitest";
import { create } from "@bufbuild/protobuf";
import { StagedMessageMode, WorktreeSuggestionQueueEntrySchema, WorktreeSuggestionQueueState } from "@vrooli/proto-types/git-control-tower/v1/advisory/advisory_pb";
import { appendStageSelection, isStageSelectionComplete, queuedHeadIncludedInIndex, queuedSelectionAwaitingUnstage, reconcileStageQueueWithIndex, removeStageSelection, retireCommittedQueueHead, SuccessfulCommitQueueGate, toQueuedStageSelection, updateStageSelection } from "./stageQueue";

describe("queued stage selections", () => {
  it("keeps each path in one future commit group and removes only the selected group", () => {
    const queue = appendStageSelection(appendStageSelection([], ["a.go", "b.go"], "first"), ["b.go", "c.go"], "second");
    expect(queue.map(item => item.paths)).toEqual([["a.go", "b.go"], ["c.go"]]);
    expect(removeStageSelection(queue, "first")).toEqual([queue[1]]);
  });

  it("projects server-owned stale and commit-pending states explicitly", () => {
    const entry = create(WorktreeSuggestionQueueEntrySchema, {
      id: "entry",
      repositoryId: "repo",
      jobId: "job",
      snapshotDigest: "digest",
      paths: ["a.go"],
      mode: StagedMessageMode.DEEP,
      requestKey: "request",
      state: WorktreeSuggestionQueueState.STALE,
      failureReason: "selected file changed",
      createdAt: "now",
    });
    expect(toQueuedStageSelection(entry)).toMatchObject({ id: "entry", snapshotDigest: "digest", state: "stale", suggestionMode: "deep" });
    expect(toQueuedStageSelection({ ...entry, state: WorktreeSuggestionQueueState.COMMIT_PENDING })).toMatchObject({ state: "commit_pending" });
  });

  it("represents saved selections without implying a suggestion job or mode", () => {
    const entry = create(WorktreeSuggestionQueueEntrySchema, {
      id: "selection-1",
      repositoryId: "repo",
      paths: ["a.go"],
      selectionId: "selection-1",
      snapshotDigest: "digest",
      state: WorktreeSuggestionQueueState.AWAITING_UNSTAGE,
      createdAt: "now",
    });
    expect(toQueuedStageSelection(entry)).toMatchObject({ id: "selection-1", selectionId: "selection-1", state: "awaiting_unstage" });
    expect(toQueuedStageSelection(entry)).not.toHaveProperty("suggestionMode");
    expect(toQueuedStageSelection(entry)).not.toHaveProperty("jobId");
    expect(toQueuedStageSelection(entry)).not.toHaveProperty("requestKey");
  });

  it("retains an arbitrarily long ordered projection", () => {
    const queue = Array.from({ length: 512 }, (_, index) => index).reduce<ReturnType<typeof appendStageSelection>>(
      (current, index) => appendStageSelection(current, [`scope-${index}/file.go`], `group-${index}`), [],
    );
    expect(queue).toHaveLength(512);
    expect(queue[0]?.paths).toEqual(["scope-0/file.go"]);
    expect(queue[511]?.paths).toEqual(["scope-511/file.go"]);
  });

  it("keeps a successful commit pending until its first follow-up selection arrives", () => {
    const gate = new SuccessfulCommitQueueGate();
    gate.markSuccessful("repo");
    expect(gate.canDispatch("repo", false)).toBe(false);
    expect(gate.canDispatch("repo", appendStageSelection([], ["next.go"], "next").length > 0)).toBe(true);
    gate.consume("repo");
    expect(gate.canDispatch("repo", true)).toBe(false);
  });

  it("derives staging completion only when every requested path succeeded", () => {
    const [selection] = appendStageSelection([], ["a.go", "b.go"], "group");
    if (!selection) throw new Error("fixture requires a queue item");
    expect(isStageSelectionComplete(selection, { success: true, staged: ["a.go", "b.go"] })).toBe(true);
    expect(isStageSelectionComplete(selection, { success: true, staged: ["a.go"] })).toBe(false);
    expect(isStageSelectionComplete(selection, { success: false, staged: ["a.go", "b.go"] })).toBe(false);
  });

  it("blocks a commit while a captured selection still overlaps the staged index", () => {
    const queue = [{ ...appendStageSelection([], ["extra.go", "extra_test.go"], "selection")[0]!, state: "awaiting_unstage" as const }];
    expect(queuedSelectionAwaitingUnstage(queue, ["current.go", "extra.go"])?.id).toBe("selection");
    expect(queuedSelectionAwaitingUnstage(queue, ["current.go"])).toBeUndefined();
  });

  it("preserves an incomplete unstage handoff during index reconciliation", () => {
    const [selection] = appendStageSelection([], ["extra.go"], "selection");
    if (!selection) throw new Error("fixture requires a queue item");
    const awaiting = { ...selection, state: "awaiting_unstage" as const };
    expect(reconcileStageQueueWithIndex([awaiting], ["extra.go"])[0]).toEqual(awaiting);
  });

  it("demotes an entry if its files are later unstaged and retires only the included head", () => {
    const [selection] = appendStageSelection([], ["a.go", "b.go"], "head");
    if (!selection) throw new Error("fixture requires a queue item");
    const staged = updateStageSelection([{ ...selection, snapshotDigest: "digest" }], selection.id, { state: "staged" });
    expect(queuedHeadIncludedInIndex(staged, ["a.go", "b.go"])).toEqual({ entryId: "head", snapshotDigest: "digest" });
    const reconciled = reconcileStageQueueWithIndex(staged, ["a.go"]);
    expect(reconciled[0]).toMatchObject({ id: "head", state: "queued" });
    expect(retireCommittedQueueHead(reconciled, "different-head")).toHaveLength(1);
    expect(retireCommittedQueueHead(staged, "head")).toHaveLength(0);
  });
});
