import { describe, it, expect } from "vitest";
import { buildFeed, countActionableItems, type FeedbackItem, type FeedItem, type MaturityItem } from "./feed";
import type { BacklogItem, BacklogKind, BacklogStatus, Capture } from "../types";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

let seq = 0;
function makeItem(overrides?: Partial<BacklogItem>): BacklogItem {
  seq++;
  return {
    name: `item-${seq}`,
    title: `Item ${seq}`,
    description: "",
    kind: "idea" as BacklogKind,
    status: "backlog" as BacklogStatus,
    priority: 2,
    tags: [],
    created: "2026-03-20T00:00:00Z",
    updated: "2026-03-20T00:00:00Z",
    ...overrides,
  };
}

function makeCapture(overrides?: Partial<Capture>): Capture {
  seq++;
  return {
    id: `cap-${seq}`,
    raw_input: "test",
    status: "classifying",
    created: "2026-03-20T00:00:00Z",
    ...overrides,
  } as Capture;
}

function itemNames(feed: FeedItem[]): string[] {
  return feed.map((f) => {
    if (f.type === "capture") return `capture:${f.capture.id}`;
    return f.item.name;
  });
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("buildFeed", () => {
  it("returns empty feed when no items", () => {
    expect(buildFeed([], [], [], [])).toEqual([]);
  });

  it("captures appear before backlog items", () => {
    const cap = makeCapture();
    const item = makeItem({ priority: 1 });
    const feed = buildFeed([cap], [item], [], []);
    const first = feed[0];
    const second = feed[1];
    expect(first).toBeDefined();
    expect(second).toBeDefined();
    if (!first || !second) {
      throw new Error("Expected two feed items");
    }
    expect(first.type).toBe("capture");
    expect(second.type).toBe("backlog");
  });

  it("attention items are boosted above normal backlog items with the same priority", () => {
    const normal = makeItem({ name: "normal", priority: 2 });
    const attention = makeItem({ name: "attention", priority: 2 });
    const feedback: FeedbackItem[] = [{ kind: "idea", name: "attention", pendingDecisions: 3 }];
    const feed = buildFeed([], [normal, attention], feedback, []);
    const first = feed[0];
    expect(itemNames(feed)).toEqual(["attention", "normal"]);
    expect(first).toBeDefined();
    if (!first) {
      throw new Error("Expected one feed item");
    }
    expect(first.type).toBe("attention");
  });

  it("excludes archived items by default", () => {
    const archived = makeItem({ name: "old", status: "archived" });
    const active = makeItem({ name: "active", status: "backlog" });
    const feed = buildFeed([], [archived, active], [], []);
    expect(itemNames(feed)).toEqual(["active"]);
  });

  it("includes completed and failed items by default", () => {
    const completed = makeItem({ name: "done", status: "completed" });
    const failed = makeItem({ name: "broken", status: "failed" });
    const active = makeItem({ name: "active", status: "backlog" });
    const feed = buildFeed([], [completed, failed, active], [], []);
    expect(itemNames(feed)).toContain("done");
    expect(itemNames(feed)).toContain("broken");
    expect(itemNames(feed)).toContain("active");
  });

  it("includes archived items when showFinished is true", () => {
    const archived = makeItem({ name: "old", status: "archived" });
    const feed = buildFeed([], [archived], [], [], { showFinished: true });
    expect(feed.length).toBe(1);
  });

  it("sorts by priority ascending", () => {
    const p1 = makeItem({ name: "p1", priority: 1 });
    const p3 = makeItem({ name: "p3", priority: 3 });
    const p2 = makeItem({ name: "p2", priority: 2 });
    const feed = buildFeed([], [p3, p1, p2], [], []);
    expect(itemNames(feed)).toEqual(["p1", "p2", "p3"]);
  });

  it("uses recency as tiebreaker for equal priority", () => {
    const older = makeItem({ name: "older", priority: 2, updated: "2026-03-01T00:00:00Z" });
    const newer = makeItem({ name: "newer", priority: 2, updated: "2026-03-20T00:00:00Z" });
    const feed = buildFeed([], [older, newer], [], []);
    expect(itemNames(feed)).toEqual(["newer", "older"]);
  });

  // -------------------------------------------------------------------------
  // Dependency blocking
  // -------------------------------------------------------------------------

  it("blocked items sort below unblocked items of the same priority", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "backlog", priority: 1 });
    const blocked = makeItem({ name: "blocked", kind: "fix", priority: 1, dependsOn: ["idea/dep"] });
    const unblocked = makeItem({ name: "unblocked", kind: "fix", priority: 1 });
    const feed = buildFeed([], [dep, blocked, unblocked], [], []);
    // unblocked and dep should appear before blocked
    const names = itemNames(feed);
    expect(names.indexOf("unblocked")).toBeLessThan(names.indexOf("blocked"));
  });

  it("blocked items sort below even lower-priority unblocked items", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "researching", priority: 5 });
    const blockedP1 = makeItem({ name: "blocked-p1", kind: "fix", priority: 1, dependsOn: ["idea/dep"] });
    const unblockedP3 = makeItem({ name: "unblocked-p3", kind: "fix", priority: 3 });
    const feed = buildFeed([], [dep, blockedP1, unblockedP3], [], []);
    const names = itemNames(feed);
    expect(names.indexOf("unblocked-p3")).toBeLessThan(names.indexOf("blocked-p1"));
  });

  it("completed dependencies do not affect sort depth", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "completed", priority: 5 });
    const item = makeItem({ name: "downstream", kind: "fix", priority: 2, dependsOn: ["idea/dep"] });
    const other = makeItem({ name: "other", kind: "fix", priority: 2 });
    // Include finished so dep shows up
    const feed = buildFeed([], [dep, item, other], [], [], { showFinished: true });
    const names = itemNames(feed);
    // Both downstream and other at depth 0 (dep is resolved), so sort by priority
    expect(names.indexOf("downstream")).toBeLessThan(names.indexOf("dep"));
    expect(names.indexOf("other")).toBeLessThan(names.indexOf("dep"));
  });

  it("items with ready dependencies sort below them", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "ready", priority: 5 });
    const item = makeItem({ name: "downstream", kind: "fix", priority: 1, dependsOn: ["idea/dep"] });
    const feed = buildFeed([], [dep, item], [], []);
    // dep is incomplete (ready ≠ completed/archived), so downstream sorts below it
    expect(itemNames(feed)).toEqual(["dep", "downstream"]);
  });

  it("items with in_progress dependencies sort below them", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "in_progress", priority: 3 });
    const item = makeItem({ name: "downstream", kind: "fix", priority: 1, dependsOn: ["idea/dep"] });
    const feed = buildFeed([], [dep, item], [], []);
    expect(itemNames(feed)).toEqual(["dep", "downstream"]);
  });

  it("items with failed dependencies sort below them", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "failed", priority: 3 });
    const item = makeItem({ name: "downstream", kind: "fix", priority: 1, dependsOn: ["idea/dep"] });
    const feed = buildFeed([], [dep, item], [], []);
    expect(itemNames(feed)).toEqual(["dep", "downstream"]);
  });

  it("multi-level chain sorts in dependency order", () => {
    const c = makeItem({ name: "c", kind: "fix", status: "backlog", priority: 3 });
    const b = makeItem({ name: "b", kind: "fix", status: "backlog", priority: 2, dependsOn: ["fix/c"] });
    const a = makeItem({ name: "a", kind: "fix", status: "backlog", priority: 1, dependsOn: ["fix/b"] });
    const feed = buildFeed([], [a, b, c], [], []);
    expect(itemNames(feed)).toEqual(["c", "b", "a"]);
  });

  it("completed dep does not push dependent down", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "completed", priority: 5 });
    const item = makeItem({ name: "downstream", kind: "fix", priority: 1, dependsOn: ["idea/dep"] });
    const feed = buildFeed([], [dep, item], [], [], { showFinished: true });
    // Both at depth 0 — downstream(p1) sorts before dep(p5)
    expect(itemNames(feed)).toEqual(["downstream", "dep"]);
  });

  it("blocked attention items are demoted below unblocked normal items", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "backlog", priority: 5 });
    const blockedAttention = makeItem({ name: "blocked-att", kind: "fix", priority: 1, dependsOn: ["idea/dep"] });
    const unblockedNormal = makeItem({ name: "unblocked", kind: "fix", priority: 2 });
    // blockedAttention has pending decisions → classified as attention
    const feedback: FeedbackItem[] = [{ kind: "fix", name: "blocked-att", pendingDecisions: 1 }];
    const feed = buildFeed([], [dep, blockedAttention, unblockedNormal], feedback, []);
    const names = itemNames(feed);
    expect(names.indexOf("unblocked")).toBeLessThan(names.indexOf("blocked-att"));
  });

  it("among blocked items, sort by priority then recency", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "backlog", priority: 5 });
    const blockedP1 = makeItem({ name: "blocked-p1", kind: "fix", priority: 1, dependsOn: ["idea/dep"], updated: "2026-03-01T00:00:00Z" });
    const blockedP2 = makeItem({ name: "blocked-p2", kind: "fix", priority: 2, dependsOn: ["idea/dep"], updated: "2026-03-20T00:00:00Z" });
    const feed = buildFeed([], [dep, blockedP2, blockedP1], [], []);
    const names = itemNames(feed);
    // dep at depth 0, then blocked-p1 and blocked-p2 at depth 1 — sorted by priority within depth
    expect(names.indexOf("blocked-p1")).toBeLessThan(names.indexOf("blocked-p2"));
  });

  it("item with missing dependency reference is not considered blocked", () => {
    // dependsOn references a non-existent item
    const item = makeItem({ name: "orphan", kind: "fix", priority: 1, dependsOn: ["idea/nonexistent"] });
    const other = makeItem({ name: "other", kind: "fix", priority: 1, updated: "2026-03-01T00:00:00Z" });
    const feed = buildFeed([], [item, other], [], []);
    // Both at same priority — orphan should not be penalized
    const names = itemNames(feed);
    expect(names.indexOf("orphan")).toBeLessThan(names.indexOf("other"));
  });

  // -------------------------------------------------------------------------
  // Attention reasons
  // -------------------------------------------------------------------------

  it("marks items with pending decisions as attention", () => {
    const item = makeItem({ name: "has-decisions" });
    const feedback: FeedbackItem[] = [{ kind: "idea", name: "has-decisions", pendingDecisions: 2 }];
    const feed = buildFeed([], [item], feedback, []);
    const first = feed[0];
    expect(first).toBeDefined();
    if (!first) {
      throw new Error("Expected one feed item");
    }
    expect(first.type).toBe("attention");
    if (first.type === "attention") {
      expect(first.reasons).toEqual([{ kind: "pending-decisions", count: 2 }]);
    }
  });

  it("marks ready items as plan-ready attention", () => {
    const item = makeItem({ name: "ready-item" });
    const maturity: MaturityItem[] = [{ kind: "idea", name: "ready-item", ready: true, pendingItems: 0 }];
    const feed = buildFeed([], [item], [], maturity);
    const first = feed[0];
    expect(first).toBeDefined();
    if (!first) {
      throw new Error("Expected one feed item");
    }
    expect(first.type).toBe("attention");
    if (first.type === "attention") {
      expect(first.reasons).toContainEqual({ kind: "plan-ready" });
    }
  });

  it("marks researching items as research-complete attention", () => {
    const item = makeItem({ name: "researching-item", status: "researching" });
    const feed = buildFeed([], [item], [], []);
    const first = feed[0];
    expect(first).toBeDefined();
    if (!first) {
      throw new Error("Expected one feed item");
    }
    expect(first.type).toBe("attention");
    if (first.type === "attention") {
      expect(first.reasons).toContainEqual({ kind: "research-complete" });
    }
  });
});

describe("countActionableItems", () => {
  it("counts captures and attention items", () => {
    const cap = makeCapture();
    const attentionItem = makeItem({ name: "att", status: "researching" });
    const normalItem = makeItem({ name: "norm" });
    const feed = buildFeed([cap], [attentionItem, normalItem], [], []);
    expect(countActionableItems(feed)).toBe(2); // 1 capture + 1 attention
  });

  it("does not count normal backlog items", () => {
    const item = makeItem();
    const feed = buildFeed([], [item], [], []);
    expect(countActionableItems(feed)).toBe(0);
  });
});
