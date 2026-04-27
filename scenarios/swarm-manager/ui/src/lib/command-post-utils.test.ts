import { describe, it, expect } from "vitest";
import {
  aggregateCrossItemQuestions,
  computeBadgeCount,
  groupActionItems,
  sortedGroupActionItems,
  type ActionGroup,
} from "./command-post-utils";
import type {
  BacklogItem,
  BacklogKind,
  BacklogStatus,
  Capture,
  ExecutionRecord,
  PendingQuestionsItem,
} from "../types";
import type { FeedbackItem, MaturityItem } from "./feed";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

let seq = 0;
function makeBacklogItem(overrides?: Partial<BacklogItem>): BacklogItem {
  seq++;
  return {
    name: `item-${seq}`,
    title: `Item ${seq}`,
    description: "",
    kind: "idea" as BacklogKind,
    status: "ready" as BacklogStatus,
    priority: 2,
    tags: [],
    suggestedSkills: [],
    created: "2026-04-01T00:00:00Z",
    updated: "2026-04-01T00:00:00Z",
    ...overrides,
  };
}

function makeExecution(overrides?: Partial<ExecutionRecord>): ExecutionRecord {
  seq++;
  return {
    executionId: `exec-${seq}`,
    backlogKind: "idea",
    backlogName: `exec-item-${seq}`,
    status: "completed",
    mode: "manual",
    startedAt: "2026-04-01T00:00:00Z",
    ...overrides,
  } as ExecutionRecord;
}

function makeCapture(overrides?: Partial<Capture>): Capture {
  seq++;
  return {
    id: `cap-${seq}`,
    text: `Capture ${seq}`,
    attachments: [],
    created: "2026-04-01T00:00:00Z",
    status: "classifying",
    classification: null,
    ...overrides,
  };
}

const EMPTY_FEEDBACK = new Map<string, FeedbackItem>();
const EMPTY_MATURITY = new Map<string, MaturityItem>();
const NO_SNOOZED = new Set<string>();

function getGroup(groups: ActionGroup[], id: string): ActionGroup {
  const group = groups.find((g) => g.id === id);
  if (!group) throw new Error(`ActionGroup "${id}" not found`);
  return group;
}

// ---------------------------------------------------------------------------
// groupActionItems
// ---------------------------------------------------------------------------

describe("groupActionItems", () => {
  it("returns all group types even when empty", () => {
    const groups = groupActionItems([], [], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(groups).toHaveLength(5);
    expect(groups.every((g) => g.count === 0)).toBe(true);
  });

  it("classifies ready backlog items into ready-to-run", () => {
    const item = makeBacklogItem({ status: "ready" });
    const maturityMap = new Map([
      [`${item.kind}/${item.name}`, { kind: item.kind, name: item.name, ready: true, pendingItems: 0 }],
    ]);
    const groups = groupActionItems([item], [], [], EMPTY_FEEDBACK, maturityMap, NO_SNOOZED);
    expect(getGroup(groups, "ready-to-run").count).toBe(1);
    expect(getGroup(groups, "ready-to-run").items[0]?.name).toBe(item.name);
  });

  it("classifies items with pending decisions into pending-decisions", () => {
    const item = makeBacklogItem({ status: "ready" });
    const feedbackMap = new Map([
      [`${item.kind}/${item.name}`, { kind: item.kind, name: item.name, pendingDecisions: 3 }],
    ]);
    const groups = groupActionItems([item], [], [], feedbackMap, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "pending-decisions").count).toBe(3);
    expect(getGroup(groups, "pending-decisions").items).toHaveLength(1);
  });

  it("classifies failed backlog items into needs-review group", () => {
    const item = makeBacklogItem({ status: "failed" });
    const groups = groupActionItems([item], [], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-review").count).toBe(1);
  });

  it("classifies completed backlog items into needs-review group", () => {
    const item = makeBacklogItem({ status: "completed" });
    const groups = groupActionItems([item], [], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-review").count).toBe(1);
  });

  it("classifies workshop-needing items into needs-workshop group", () => {
    // An item with readinessReady=false → primaryCta = "workshop"
    const item = makeBacklogItem({ status: "backlog" });
    const maturityMap = new Map([
      [`${item.kind}/${item.name}`, { kind: item.kind, name: item.name, ready: false, pendingItems: 1 }],
    ]);
    const groups = groupActionItems([item], [], [], EMPTY_FEEDBACK, maturityMap, NO_SNOOZED);
    expect(getGroup(groups, "needs-workshop").count).toBe(1);
  });

  it("classifies needs_review executions into needs-review group", () => {
    const exec = makeExecution({ status: "needs_review" });
    const groups = groupActionItems([], [exec], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-review").count).toBe(1);
  });

  it("classifies needs_fixup executions into needs-review group", () => {
    const exec = makeExecution({ status: "needs_fixup" });
    const groups = groupActionItems([], [exec], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-review").count).toBe(1);
  });

  it("classifies failed executions into needs-review group", () => {
    const exec = makeExecution({ status: "failed" });
    const groups = groupActionItems([], [exec], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-review").count).toBe(1);
  });

  it("classifies completed executions into needs-review group", () => {
    const exec = makeExecution({ status: "completed" });
    const groups = groupActionItems([], [exec], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-review").count).toBe(1);
  });

  it("classifies classifying captures into needs-classification", () => {
    const cap = makeCapture({ status: "classifying" });
    const groups = groupActionItems([], [], [cap], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-classification").count).toBe(1);
  });

  it("classifies classified captures with items into needs-classification", () => {
    const cap = makeCapture({
      status: "classified",
      classification: {
        items: [{ kind: "fix", title: "Fix bug", description: "", priority: 5, tags: [], confidence: 0.9 }],
        classifiedAt: "2026-04-01T00:00:00Z",
      },
    });
    const groups = groupActionItems([], [], [cap], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    const group = getGroup(groups, "needs-classification");
    expect(group.count).toBe(1);
    expect(group.items[0]?.primaryCta).toBe("review");
  });

  it("excludes classified captures with no items from needs-classification", () => {
    const cap = makeCapture({
      status: "classified",
      classification: { items: [], classifiedAt: "2026-04-01T00:00:00Z" },
    });
    const groups = groupActionItems([], [], [cap], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-classification").count).toBe(0);
  });

  it("excludes failed captures from needs-classification (retry via Captures tab)", () => {
    const cap = makeCapture({ status: "failed" });
    const groups = groupActionItems([], [], [cap], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    expect(getGroup(groups, "needs-classification").count).toBe(0);
  });

  it("excludes snoozed items", () => {
    const item = makeBacklogItem({ status: "ready" });
    const maturityMap = new Map([
      [`${item.kind}/${item.name}`, { kind: item.kind, name: item.name, ready: true, pendingItems: 0 }],
    ]);
    const snoozed = new Set([`backlog:${item.kind}/${item.name}`]);
    const groups = groupActionItems([item], [], [], EMPTY_FEEDBACK, maturityMap, snoozed);
    expect(getGroup(groups, "ready-to-run").count).toBe(0);
  });

  it("excludes snoozed executions", () => {
    const exec = makeExecution({ status: "needs_review" });
    const snoozed = new Set([`execution:${exec.executionId}`]);
    const groups = groupActionItems([], [exec], [], EMPTY_FEEDBACK, EMPTY_MATURITY, snoozed);
    expect(getGroup(groups, "needs-review").count).toBe(0);
  });

  it("excludes snoozed captures", () => {
    const cap = makeCapture({ status: "classifying" });
    const snoozed = new Set([`capture:${cap.id}`]);
    const groups = groupActionItems([], [], [cap], EMPTY_FEEDBACK, EMPTY_MATURITY, snoozed);
    expect(getGroup(groups, "needs-classification").count).toBe(0);
  });

  it("excludes archived backlog items from all groups", () => {
    // An archived completed item should NOT appear in needs-review
    const archivedCompleted = makeBacklogItem({ status: "completed", archivedAt: "2026-04-01T00:00:00Z" });
    // An archived ready item should NOT appear in ready-to-run
    const archivedReady = makeBacklogItem({ status: "ready", archivedAt: "2026-04-01T00:00:00Z" });
    const maturityMap = new Map([
      [`${archivedReady.kind}/${archivedReady.name}`, { kind: archivedReady.kind, name: archivedReady.name, ready: true, pendingItems: 0 }],
    ]);
    const groups = groupActionItems([archivedCompleted, archivedReady], [], [], EMPTY_FEEDBACK, maturityMap, NO_SNOOZED);
    const totalCount = groups.reduce((sum, g) => sum + g.count, 0);
    expect(totalCount).toBe(0);
  });

  it("skips locked items (queued/in_progress)", () => {
    const queued = makeBacklogItem({ status: "queued" });
    const inProgress = makeBacklogItem({ status: "in_progress" });
    const groups = groupActionItems([queued, inProgress], [], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    const totalCount = groups.reduce((sum, g) => sum + g.count, 0);
    expect(totalCount).toBe(0);
  });
});

// ---------------------------------------------------------------------------
// sortedGroupActionItems
// ---------------------------------------------------------------------------

describe("sortedGroupActionItems", () => {
  it("sorts backlog items within groups by dependency order", () => {
    const parent = makeBacklogItem({
      name: "parent",
      kind: "execute",
      status: "ready",
      priority: 5,
    });
    const child = makeBacklogItem({
      name: "child",
      kind: "execute",
      status: "ready",
      priority: 1,
      dependsOn: ["execute/parent"],
    });
    const maturityMap = new Map([
      [`execute/parent`, { kind: "execute", name: "parent", ready: true, pendingItems: 0 }],
      [`execute/child`, { kind: "execute", name: "child", ready: true, pendingItems: 0 }],
    ]);
    // Pass child before parent to verify sorting reorders them
    const groups = sortedGroupActionItems(
      [child, parent], [], [], EMPTY_FEEDBACK, maturityMap, NO_SNOOZED,
    );
    const readyGroup = getGroup(groups, "ready-to-run");
    expect(readyGroup.count).toBe(2);
    // Parent (depth 0) should come before child (depth 1) despite child having lower priority number
    expect(readyGroup.items[0]?.name).toBe("parent");
    expect(readyGroup.items[1]?.name).toBe("child");
  });

  it("uses priority as tiebreaker within same depth", () => {
    const lowPri = makeBacklogItem({
      name: "low-pri",
      kind: "execute",
      status: "ready",
      priority: 1,
    });
    const highPri = makeBacklogItem({
      name: "high-pri",
      kind: "execute",
      status: "ready",
      priority: 5,
    });
    const maturityMap = new Map([
      [`execute/low-pri`, { kind: "execute", name: "low-pri", ready: true, pendingItems: 0 }],
      [`execute/high-pri`, { kind: "execute", name: "high-pri", ready: true, pendingItems: 0 }],
    ]);
    const groups = sortedGroupActionItems(
      [highPri, lowPri], [], [], EMPTY_FEEDBACK, maturityMap, NO_SNOOZED,
    );
    const readyGroup = getGroup(groups, "ready-to-run");
    // Same depth (0), priority ascending: low-pri (1) before high-pri (5)
    expect(readyGroup.items[0]?.name).toBe("low-pri");
    expect(readyGroup.items[1]?.name).toBe("high-pri");
  });

  it("does not affect execution and capture ordering", () => {
    const exec1 = makeExecution({ status: "needs_review", backlogName: "exec-first" });
    const exec2 = makeExecution({ status: "needs_review", backlogName: "exec-second" });
    const groups = sortedGroupActionItems(
      [], [exec1, exec2], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED,
    );
    const reviewGroup = getGroup(groups, "needs-review");
    expect(reviewGroup.count).toBe(2);
    // Executions appear in original order (no dependency sorting applies)
    expect(reviewGroup.items[0]?.executionId).toBe(exec1.executionId);
    expect(reviewGroup.items[1]?.executionId).toBe(exec2.executionId);
  });

  it("populates backlogItem reference on backlog actionable items", () => {
    const item = makeBacklogItem({ status: "ready", kind: "execute" });
    const maturityMap = new Map([
      [`${item.kind}/${item.name}`, { kind: item.kind, name: item.name, ready: true, pendingItems: 0 }],
    ]);
    const groups = sortedGroupActionItems(
      [item], [], [], EMPTY_FEEDBACK, maturityMap, NO_SNOOZED,
    );
    const readyGroup = getGroup(groups, "ready-to-run");
    expect(readyGroup.items[0]?.backlogItem).toBe(item);
  });

  it("items with higher fan-out sort before same-priority peers within same group", () => {
    // Both items are ready-to-run, same priority, but blocker has dependents
    const blocker = makeBacklogItem({ name: "blocker", status: "ready", priority: 3, kind: "fix" });
    const standalone = makeBacklogItem({ name: "standalone", status: "ready", priority: 3, kind: "idea" });
    const dep = makeBacklogItem({ name: "dep", status: "backlog", priority: 5, dependsOn: ["fix/blocker"] });
    const maturityMap = new Map([
      [`fix/blocker`, { kind: "fix", name: "blocker", ready: true, pendingItems: 0 }],
      [`idea/standalone`, { kind: "idea", name: "standalone", ready: true, pendingItems: 0 }],
    ]);
    const groups = sortedGroupActionItems(
      [standalone, blocker, dep], [], [], EMPTY_FEEDBACK, maturityMap, NO_SNOOZED,
    );
    const readyGroup = getGroup(groups, "ready-to-run");
    const readyNames = readyGroup.items.map((i) => i.name);
    expect(readyNames.indexOf("blocker")).toBeLessThan(readyNames.indexOf("standalone"));
  });
});

// ---------------------------------------------------------------------------
// aggregateCrossItemQuestions
// ---------------------------------------------------------------------------

describe("aggregateCrossItemQuestions", () => {
  it("returns empty array when no items", () => {
    expect(aggregateCrossItemQuestions([], NO_SNOOZED)).toEqual([]);
  });

  it("flattens questions from multiple items", () => {
    const items: PendingQuestionsItem[] = [
      {
        kind: "idea",
        name: "item-a",
        questions: [
          { id: "q1", source: "workshop", item_kind: "idea", item_name: "item-a", topic: "T1" },
          { id: "q2", source: "workshop", item_kind: "idea", item_name: "item-a", topic: "T2" },
        ],
      },
      {
        kind: "fix",
        name: "item-b",
        questions: [
          { id: "q3", source: "review", item_kind: "fix", item_name: "item-b", title: "Review X" },
        ],
      },
    ];

    const result = aggregateCrossItemQuestions(items, NO_SNOOZED);
    expect(result).toHaveLength(3);
    expect(result[0]?.parentKind).toBe("idea");
    expect(result[0]?.parentName).toBe("item-a");
    expect(result[2]?.parentKind).toBe("fix");
  });

  it("filters out questions from snoozed items", () => {
    const items: PendingQuestionsItem[] = [
      {
        kind: "idea",
        name: "snoozed-item",
        questions: [
          { id: "q1", source: "workshop", item_kind: "idea", item_name: "snoozed-item" },
        ],
      },
      {
        kind: "fix",
        name: "active-item",
        questions: [
          { id: "q2", source: "workshop", item_kind: "fix", item_name: "active-item" },
        ],
      },
    ];

    const snoozed = new Set(["backlog:idea/snoozed-item"]);
    const result = aggregateCrossItemQuestions(items, snoozed);
    expect(result).toHaveLength(1);
    expect(result[0]?.parentName).toBe("active-item");
  });

  it("filters out questions from items not in active backlog", () => {
    const items: PendingQuestionsItem[] = [
      {
        kind: "idea",
        name: "archived-item",
        questions: [
          { id: "q1", source: "workshop", item_kind: "idea", item_name: "archived-item" },
        ],
      },
      {
        kind: "fix",
        name: "active-item",
        questions: [
          { id: "q2", source: "workshop", item_kind: "fix", item_name: "active-item" },
        ],
      },
    ];

    const activeKeys = new Set(["fix/active-item"]);
    const result = aggregateCrossItemQuestions(items, NO_SNOOZED, activeKeys);
    expect(result).toHaveLength(1);
    expect(result[0]?.parentName).toBe("active-item");
  });

  it("filters out questions that are already answered in stale payloads", () => {
    const items: PendingQuestionsItem[] = [
      {
        kind: "idea",
        name: "item-a",
        questions: [
          { id: "q1", source: "workshop", item_kind: "idea", item_name: "item-a", selected: "yes" },
          { id: "q2", source: "workshop", item_kind: "idea", item_name: "item-a" },
          { id: "q3", source: "review", item_kind: "idea", item_name: "item-a", review_status: "approved" },
          { id: "q4", source: "review", item_kind: "idea", item_name: "item-a", review_status: "unreviewed" },
        ],
      },
    ];

    const result = aggregateCrossItemQuestions(items, NO_SNOOZED);
    expect(result.map((ciq) => ciq.question.id)).toEqual(["q2", "q4"]);
  });

  it("includes all items when activeItemKeys is undefined", () => {
    const items: PendingQuestionsItem[] = [
      {
        kind: "idea",
        name: "any-item",
        questions: [
          { id: "q1", source: "workshop", item_kind: "idea", item_name: "any-item" },
        ],
      },
    ];

    const result = aggregateCrossItemQuestions(items, NO_SNOOZED, undefined);
    expect(result).toHaveLength(1);
  });
});

// ---------------------------------------------------------------------------
// computeBadgeCount
// ---------------------------------------------------------------------------

describe("computeBadgeCount", () => {
  it("returns 0 for empty groups", () => {
    expect(computeBadgeCount([])).toBe(0);
  });

  it("sums counts across all groups", () => {
    const groups: ActionGroup[] = [
      { id: "ready-to-run", label: "Ready", count: 3, items: [] },
      { id: "pending-decisions", label: "Pending", count: 2, items: [] },
      { id: "needs-review", label: "Review", count: 1, items: [] },
      { id: "needs-workshop", label: "Workshop", count: 0, items: [] },
      { id: "needs-classification", label: "Classify", count: 4, items: [] },
    ];
    expect(computeBadgeCount(groups)).toBe(10);
  });
});
