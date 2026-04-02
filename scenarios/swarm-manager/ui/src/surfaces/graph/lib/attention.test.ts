import { describe, expect, it } from "vitest";
import {
  computeNodeAttention,
  formatAttentionSummary,
  type NodeEnrichment,
} from "./attention";
import type { GraphNodeData } from "../types";

function makeBacklog(status: string, kind = "execute", name = "test"): GraphNodeData {
  return {
    label: name,
    entityType: "backlog",
    rawType: "BacklogItem",
    kind,
    name,
    title: name,
    status,
    priority: 1,
  } as GraphNodeData;
}

function makeExecution(status: string): GraphNodeData {
  return {
    label: "exec-1",
    entityType: "execution",
    rawType: "ExecutionRecord",
    executionId: "exec-1",
    backlogKind: "execute",
    backlogName: "test",
    status,
    mode: "manual",
  } as GraphNodeData;
}

function makeCapture(status: string): GraphNodeData {
  return {
    label: "cap-1",
    entityType: "capture",
    rawType: "Capture",
    id: "cap-1",
    text: "test capture",
    status,
  } as GraphNodeData;
}

describe("computeNodeAttention", () => {
  it("flags ready backlog as ready-to-run", () => {
    const result = computeNodeAttention(makeBacklog("ready"));
    expect(result.needsAttention).toBe(true);
    expect(result.reasons).toContain("ready-to-run");
  });

  it("flags failed backlog as failed", () => {
    const result = computeNodeAttention(makeBacklog("failed"));
    expect(result.needsAttention).toBe(true);
    expect(result.reasons).toContain("failed");
  });

  it("flags pending decisions from enrichment", () => {
    const enrichment: NodeEnrichment = { pendingDecisions: 2 };
    const result = computeNodeAttention(makeBacklog("in_progress"), enrichment);
    expect(result.needsAttention).toBe(true);
    expect(result.reasons).toContain("pending-decisions");
  });

  it("does not flag queued backlog", () => {
    const result = computeNodeAttention(makeBacklog("queued"));
    expect(result.needsAttention).toBe(false);
  });

  it("flags needs_review execution", () => {
    const result = computeNodeAttention(makeExecution("needs_review"));
    expect(result.needsAttention).toBe(true);
    expect(result.reasons).toContain("needs-review");
  });

  it("flags needs_fixup execution", () => {
    const result = computeNodeAttention(makeExecution("needs_fixup"));
    expect(result.needsAttention).toBe(true);
    expect(result.reasons).toContain("needs-review");
  });

  it("flags failed execution", () => {
    const result = computeNodeAttention(makeExecution("failed"));
    expect(result.needsAttention).toBe(true);
    expect(result.reasons).toContain("failed");
  });

  it("does not flag running execution", () => {
    const result = computeNodeAttention(makeExecution("running"));
    expect(result.needsAttention).toBe(false);
  });

  it("flags classifying capture", () => {
    const result = computeNodeAttention(makeCapture("classifying"));
    expect(result.needsAttention).toBe(true);
    expect(result.reasons).toContain("needs-classification");
  });

  it("does not flag classified capture", () => {
    const result = computeNodeAttention(makeCapture("classified"));
    expect(result.needsAttention).toBe(false);
  });

  it("respects snoozed keys", () => {
    const snoozed = new Set(["backlog:execute/test"]);
    const result = computeNodeAttention(makeBacklog("ready"), undefined, snoozed);
    expect(result.needsAttention).toBe(false);
  });
});

describe("formatAttentionSummary", () => {
  it("returns empty string for no reasons", () => {
    expect(formatAttentionSummary([])).toBe("");
  });

  it("formats single reason", () => {
    expect(formatAttentionSummary(["ready-to-run"])).toBe("Ready to run");
  });

  it("formats multiple reasons", () => {
    const summary = formatAttentionSummary(["failed", "pending-decisions"]);
    expect(summary).toBe("Failed, Pending decisions");
  });
});
