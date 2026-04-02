import { describe, it, expect } from "vitest";
import {
  aggregateCrossItemQuestions,
  computeBadgeCount,
  groupActionItems,
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
    expect(getGroup(groups, "pending-decisions").count).toBe(1);
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

  it("skips locked items (queued/in_progress)", () => {
    const queued = makeBacklogItem({ status: "queued" });
    const inProgress = makeBacklogItem({ status: "in_progress" });
    const groups = groupActionItems([queued, inProgress], [], [], EMPTY_FEEDBACK, EMPTY_MATURITY, NO_SNOOZED);
    const totalCount = groups.reduce((sum, g) => sum + g.count, 0);
    expect(totalCount).toBe(0);
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
