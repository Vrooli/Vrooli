import { describe, expect, it } from "vitest";
import { getAttentionReasons, type FeedbackItem, type MaturityItem } from "./attention";
import type { BacklogItem } from "../types";

function item(overrides: Partial<BacklogItem>): BacklogItem {
  return {
    kind: "idea",
    name: "test-item",
    title: "Test Item",
    status: "backlog",
    priority: 5,
    created: "2026-07-01T00:00:00Z",
    updated: "2026-07-01T00:00:00Z",
    ...overrides,
  } as BacklogItem;
}

describe("getAttentionReasons", () => {
  it("returns nothing for a plain backlog item", () => {
    expect(getAttentionReasons(item({}), new Map(), new Map())).toEqual([]);
  });

  it("flags pending decisions, plan readiness, and research completion", () => {
    const feedback = new Map<string, FeedbackItem>([
      ["idea/test-item", { kind: "idea", name: "test-item", pendingDecisions: 2 }],
    ]);
    const maturity = new Map<string, MaturityItem>([
      ["idea/test-item", { kind: "idea", name: "test-item", ready: true, pendingItems: 0 }],
    ]);

    expect(getAttentionReasons(item({ status: "researching" }), feedback, maturity)).toEqual([
      { kind: "pending-decisions", count: 2 },
      { kind: "plan-ready" },
      { kind: "research-complete" },
    ]);
  });

  it("ignores feedback with zero pending decisions", () => {
    const feedback = new Map<string, FeedbackItem>([
      ["idea/test-item", { kind: "idea", name: "test-item", pendingDecisions: 0 }],
    ]);
    expect(getAttentionReasons(item({}), feedback, new Map())).toEqual([]);
  });
});
