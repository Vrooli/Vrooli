import { describe, it, expect } from "vitest";
import { mergeTimeline } from "./useActivityTimeline";
import type { ExecutionRecord, AgentActivity } from "../types";

// Minimal factories — only fields needed by mergeTimeline
function makeExecution(overrides: Partial<ExecutionRecord> & { executionId: string; createdAt: string }): ExecutionRecord {
  return {
    status: "completed",
    mode: "yolo",
    backlogKind: "execute",
    ...overrides,
  } as ExecutionRecord;
}

function makeActivity(overrides: Partial<AgentActivity> & { activityId: string; requestedAt: string }): AgentActivity {
  return {
    ownerType: "backlog",
    ownerKind: "execute",
    ownerName: "test-item",
    purpose: "process",
    interactionType: "spawn",
    status: "complete",
    ...overrides,
  } as AgentActivity;
}

describe("mergeTimeline", () => {
  it("returns empty array when both inputs are empty", () => {
    expect(mergeTimeline([], [])).toEqual([]);
    expect(mergeTimeline(undefined, undefined)).toEqual([]);
  });

  it("returns execution entries when no activities exist", () => {
    const execs = [
      makeExecution({ executionId: "e1", createdAt: "2026-03-30T10:00:00Z" }),
      makeExecution({ executionId: "e2", createdAt: "2026-03-30T09:00:00Z" }),
    ];
    const result = mergeTimeline(execs, []);
    expect(result).toHaveLength(2);
    expect(result[0]!.id).toBe("e1");
    expect(result[0]!.type).toBe("execution");
    expect(result[1]!.id).toBe("e2");
  });

  it("groups activities under their parent execution", () => {
    const execs = [
      makeExecution({ executionId: "e1", createdAt: "2026-03-30T10:00:00Z" }),
    ];
    const acts = [
      makeActivity({ activityId: "a1", executionId: "e1", requestedAt: "2026-03-30T10:01:00Z", purpose: "workshop" }),
      makeActivity({ activityId: "a2", executionId: "e1", requestedAt: "2026-03-30T10:05:00Z", purpose: "finalize" }),
    ];
    const result = mergeTimeline(execs, acts);
    expect(result).toHaveLength(1);
    expect(result[0]!.type).toBe("execution");
    expect(result[0]!.childActivities).toHaveLength(2);
    // Sorted newest-first
    expect(result[0]!.childActivities![0]!.activityId).toBe("a2");
    expect(result[0]!.childActivities![1]!.activityId).toBe("a1");
  });

  it("places orphan activities as standalone top-level entries", () => {
    const acts = [
      makeActivity({ activityId: "a1", requestedAt: "2026-03-30T10:00:00Z" }),
      makeActivity({ activityId: "a2", executionId: undefined, requestedAt: "2026-03-30T09:00:00Z" }),
    ];
    const result = mergeTimeline([], acts);
    expect(result).toHaveLength(2);
    expect(result[0]!.type).toBe("activity");
    expect(result[0]!.id).toBe("a1");
    expect(result[1]!.id).toBe("a2");
  });

  it("sorts top-level entries newest-first", () => {
    const execs = [
      makeExecution({ executionId: "e1", createdAt: "2026-03-30T08:00:00Z" }),
      makeExecution({ executionId: "e2", createdAt: "2026-03-30T12:00:00Z" }),
    ];
    const acts = [
      makeActivity({ activityId: "a1", requestedAt: "2026-03-30T10:00:00Z" }),
    ];
    const result = mergeTimeline(execs, acts);
    expect(result.map((e) => e.id)).toEqual(["e2", "a1", "e1"]);
  });

  it("handles mixed grouped and orphan activities", () => {
    const execs = [
      makeExecution({ executionId: "e1", createdAt: "2026-03-30T10:00:00Z" }),
    ];
    const acts = [
      makeActivity({ activityId: "a1", executionId: "e1", requestedAt: "2026-03-30T10:01:00Z" }),
      makeActivity({ activityId: "a2", requestedAt: "2026-03-30T11:00:00Z" }),
    ];
    const result = mergeTimeline(execs, acts);
    expect(result).toHaveLength(2);
    // a2 (orphan at 11:00) is newest, e1 (10:00) is second
    expect(result[0]!.id).toBe("a2");
    expect(result[0]!.type).toBe("activity");
    expect(result[1]!.id).toBe("e1");
    expect(result[1]!.childActivities).toHaveLength(1);
  });
});
