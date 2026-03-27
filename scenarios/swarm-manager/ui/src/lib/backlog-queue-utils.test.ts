import { describe, it, expect } from "vitest";
import {
  getBacklogNotQueueableReason,
  getBlockingDepKeys,
  getItemActions,
  hasBlockingDeps,
  isBacklogQueueable,
  LOCKED_STATUSES,
  TERMINAL_STATUSES,
  type ActionContext,
} from "./backlog-queue-utils";
import type { BacklogItem, BacklogKind, BacklogStatus } from "../types";

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

function makeCtx(overrides?: Partial<ActionContext>): ActionContext {
  return {
    item: makeItem(),
    allItems: [],
    readinessReady: null,
    agentRunning: false,
    hasPendingDecisions: false,
    hasExecutionHistory: false,
    ...overrides,
  };
}

// ---------------------------------------------------------------------------
// isBacklogQueueable
// ---------------------------------------------------------------------------

describe("isBacklogQueueable", () => {
  const queueableStatuses: BacklogStatus[] = ["backlog", "researching", "ready"];
  const nonQueueableStatuses: BacklogStatus[] = ["queued", "in_progress", "completed", "failed", "archived"];
  const nonResearchKinds: BacklogKind[] = ["idea", "fix", "execute", "chore"];

  it("returns true for non-research items in queueable statuses", () => {
    for (const kind of nonResearchKinds) {
      for (const status of queueableStatuses) {
        expect(isBacklogQueueable({ kind, status })).toBe(true);
      }
    }
  });

  it("returns false for non-research items in non-queueable statuses", () => {
    for (const kind of nonResearchKinds) {
      for (const status of nonQueueableStatuses) {
        // Exception: archived ideas ARE queueable
        if (kind === "idea" && status === "archived") continue;
        expect(isBacklogQueueable({ kind, status })).toBe(false);
      }
    }
  });

  it("returns true for research items in queueable statuses", () => {
    for (const status of queueableStatuses) {
      expect(isBacklogQueueable({ kind: "research", status })).toBe(true);
    }
  });

  it("returns true for archived ideas (special case)", () => {
    expect(isBacklogQueueable({ kind: "idea", status: "archived" })).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// hasBlockingDeps / getBlockingDepKeys
// ---------------------------------------------------------------------------

describe("hasBlockingDeps", () => {
  it("returns false when item has no dependencies", () => {
    const item = makeItem();
    expect(hasBlockingDeps(item, [])).toBe(false);
  });

  it("returns false when dependencies are in non-blocking statuses", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "ready" });
    const item = makeItem({ dependsOn: ["idea/dep"] });
    expect(hasBlockingDeps(item, [dep, item])).toBe(false);
  });

  it("returns true when a dependency is in backlog status", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "backlog" });
    const item = makeItem({ dependsOn: ["idea/dep"] });
    expect(hasBlockingDeps(item, [dep, item])).toBe(true);
  });

  it("returns true when a dependency is in researching status", () => {
    const dep = makeItem({ name: "dep", kind: "idea", status: "researching" });
    const item = makeItem({ dependsOn: ["idea/dep"] });
    expect(hasBlockingDeps(item, [dep, item])).toBe(true);
  });

  it("returns false when dependency reference doesn't exist in allItems", () => {
    const item = makeItem({ dependsOn: ["idea/nonexistent"] });
    expect(hasBlockingDeps(item, [item])).toBe(false);
  });

  it("returns false for completed/failed/queued/in_progress dependencies", () => {
    for (const status of ["completed", "failed", "queued", "in_progress", "archived"] as BacklogStatus[]) {
      const dep = makeItem({ name: "dep", kind: "idea", status });
      const item = makeItem({ dependsOn: ["idea/dep"] });
      expect(hasBlockingDeps(item, [dep, item])).toBe(false);
    }
  });
});

describe("getBlockingDepKeys", () => {
  it("returns empty array when no blocking deps", () => {
    const item = makeItem();
    expect(getBlockingDepKeys(item, [])).toEqual([]);
  });

  it("returns keys of blocking dependencies", () => {
    const dep1 = makeItem({ name: "dep1", kind: "idea", status: "backlog" });
    const dep2 = makeItem({ name: "dep2", kind: "fix", status: "ready" });
    const dep3 = makeItem({ name: "dep3", kind: "chore", status: "researching" });
    const item = makeItem({ dependsOn: ["idea/dep1", "fix/dep2", "chore/dep3"] });
    const keys = getBlockingDepKeys(item, [dep1, dep2, dep3, item]);
    expect(keys).toEqual(["idea/dep1", "chore/dep3"]);
  });
});

// ---------------------------------------------------------------------------
// getBacklogNotQueueableReason
// ---------------------------------------------------------------------------

describe("getBacklogNotQueueableReason", () => {
  it("returns null for queueable items", () => {
    expect(getBacklogNotQueueableReason({ kind: "idea", status: "backlog" })).toBeNull();
    expect(getBacklogNotQueueableReason({ kind: "idea", status: "ready" })).toBeNull();
    expect(getBacklogNotQueueableReason({ kind: "idea", status: "archived" })).toBeNull();
  });

  it("returns null for research items in queueable statuses", () => {
    expect(getBacklogNotQueueableReason({ kind: "research", status: "backlog" })).toBeNull();
  });

  it("returns status-specific reasons for non-queueable statuses", () => {
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "queued" })).toContain("Already queued");
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "in_progress" })).toContain("Already in progress");
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "completed" })).toContain("cannot be queued again");
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "failed" })).toContain("Reset status");
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "archived" })).toContain("Only archived ideas");
  });
});

// ---------------------------------------------------------------------------
// Status constants
// ---------------------------------------------------------------------------

describe("status constants", () => {
  it("LOCKED_STATUSES contains queued and in_progress", () => {
    expect(LOCKED_STATUSES.has("queued")).toBe(true);
    expect(LOCKED_STATUSES.has("in_progress")).toBe(true);
    expect(LOCKED_STATUSES.size).toBe(2);
  });

  it("TERMINAL_STATUSES contains completed and failed", () => {
    expect(TERMINAL_STATUSES.has("completed")).toBe(true);
    expect(TERMINAL_STATUSES.has("failed")).toBe(true);
    expect(TERMINAL_STATUSES.size).toBe(2);
  });
});

// ---------------------------------------------------------------------------
// getItemActions
// ---------------------------------------------------------------------------

describe("getItemActions", () => {
  // -------------------------------------------------------------------------
  // Step -1: Locked items
  // -------------------------------------------------------------------------

  describe("locked items (queued / in_progress)", () => {
    it("returns locked=true with no visible CTAs for queued status", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "queued" }),
      }));
      expect(result.locked).toBe(true);
      expect(result.primaryCta).toBeNull();
      expect(result.canRun).toBe(false);
      expect(result.canWorkshop).toBe(false);
      expect(result.canFollowUp).toBe(false);
      expect(result.canArchive).toBe(false);
      expect(result.showDecisionStepper).toBe(false);
    });

    it("returns locked=true with no visible CTAs for in_progress status", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "in_progress" }),
      }));
      expect(result.locked).toBe(true);
      expect(result.primaryCta).toBeNull();
    });

    it("ignores agentRunning and pendingDecisions when locked", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "queued" }),
        agentRunning: true,
        hasPendingDecisions: true,
      }));
      expect(result.locked).toBe(true);
      expect(result.showDecisionStepper).toBe(false);
    });
  });

  // -------------------------------------------------------------------------
  // Step 5: Terminal items
  // -------------------------------------------------------------------------

  describe("terminal items (completed / failed)", () => {
    it("completed: canArchive=true, canFollowUp depends on execution history", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "completed" }),
        hasExecutionHistory: true,
      }));
      expect(result.terminal).toBe(true);
      expect(result.canArchive).toBe(true);
      expect(result.canFollowUp).toBe(true);
      expect(result.primaryCta).toBe("followUp");
      expect(result.canRun).toBe(false);
      expect(result.canWorkshop).toBe(false);
    });

    it("completed without execution history: primaryCta is archive", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "completed" }),
        hasExecutionHistory: false,
      }));
      expect(result.canArchive).toBe(true);
      expect(result.canFollowUp).toBe(false);
      expect(result.primaryCta).toBe("archive");
    });

    it("failed: same behavior as completed", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "failed" }),
        hasExecutionHistory: true,
      }));
      expect(result.terminal).toBe(true);
      expect(result.canArchive).toBe(true);
      expect(result.canFollowUp).toBe(true);
      expect(result.primaryCta).toBe("followUp");
    });

    it("terminal items never show run/workshop even with readiness data", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "completed" }),
        readinessReady: false,
      }));
      expect(result.canRun).toBe(false);
      expect(result.canWorkshop).toBe(false);
    });
  });

  // -------------------------------------------------------------------------
  // Step 0: Blocked by deps
  // -------------------------------------------------------------------------

  describe("blocked by dependencies", () => {
    it("shows run as disabled when blocked and item is otherwise ready", () => {
      const dep = makeItem({ name: "dep", kind: "idea", status: "backlog" });
      const item = makeItem({ status: "ready", dependsOn: ["idea/dep"] });
      const result = getItemActions(makeCtx({
        item,
        allItems: [dep, item],
        readinessReady: true,
      }));
      expect(result.blocked).toBe(true);
      expect(result.runDisabled).toBe(true);
      expect(result.canRun).toBe(false);
      expect(result.primaryCta).toBe("run");
    });

    it("shows workshop as disabled when blocked and readiness not met", () => {
      const dep = makeItem({ name: "dep", kind: "idea", status: "backlog" });
      const item = makeItem({ status: "backlog", dependsOn: ["idea/dep"] });
      const result = getItemActions(makeCtx({
        item,
        allItems: [dep, item],
        readinessReady: false,
      }));
      expect(result.blocked).toBe(true);
      expect(result.workshopDisabled).toBe(true);
      expect(result.canWorkshop).toBe(false);
      expect(result.primaryCta).toBe("workshop");
    });

    it("populates blockingDepKeys", () => {
      const dep = makeItem({ name: "blocker", kind: "idea", status: "backlog" });
      const item = makeItem({ status: "backlog", dependsOn: ["idea/blocker"] });
      const result = getItemActions(makeCtx({
        item,
        allItems: [dep, item],
      }));
      expect(result.blockingDepKeys).toEqual(["idea/blocker"]);
    });

    it("still shows decision stepper when blocked with pending decisions", () => {
      const dep = makeItem({ name: "dep", kind: "idea", status: "backlog" });
      const item = makeItem({ status: "backlog", dependsOn: ["idea/dep"] });
      const result = getItemActions(makeCtx({
        item,
        allItems: [dep, item],
        hasPendingDecisions: true,
        readinessReady: false,
      }));
      expect(result.showDecisionStepper).toBe(true);
    });
  });

  // -------------------------------------------------------------------------
  // Step 1: Agent running (cross-cutting — tested with other steps)
  // -------------------------------------------------------------------------

  describe("agent running", () => {
    it("disables run when agent is running and item is ready", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "ready" }),
        readinessReady: true,
        agentRunning: true,
      }));
      expect(result.canRun).toBe(false);
      expect(result.runDisabled).toBe(true);
      expect(result.agentRunning).toBe(true);
    });

    it("disables workshop when agent is running and readiness not met", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        readinessReady: false,
        agentRunning: true,
      }));
      expect(result.canWorkshop).toBe(false);
      expect(result.workshopDisabled).toBe(true);
    });

    it("showDecisionStepper still true when agent running with pending decisions", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        readinessReady: false,
        agentRunning: true,
        hasPendingDecisions: true,
      }));
      expect(result.showDecisionStepper).toBe(true);
    });
  });

  // -------------------------------------------------------------------------
  // Step 2: Unanswered decisions
  // -------------------------------------------------------------------------

  describe("pending decisions (step 2)", () => {
    it("showDecisionStepper=true with pending decisions", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        hasPendingDecisions: true,
      }));
      expect(result.showDecisionStepper).toBe(true);
      expect(result.canRun).toBe(false);
    });

    it("workshop visible as secondary when not ready", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        hasPendingDecisions: true,
        readinessReady: false,
      }));
      expect(result.showDecisionStepper).toBe(true);
      expect(result.canWorkshop).toBe(true);
      expect(result.primaryCta).toBe("workshop");
    });

    it("no workshop when readiness is met", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "ready" }),
        hasPendingDecisions: true,
        readinessReady: true,
      }));
      expect(result.showDecisionStepper).toBe(true);
      expect(result.canWorkshop).toBe(false);
      expect(result.primaryCta).toBeNull();
    });

    it("workshop disabled when agent running with pending decisions", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        hasPendingDecisions: true,
        readinessReady: false,
        agentRunning: true,
      }));
      expect(result.showDecisionStepper).toBe(true);
      expect(result.canWorkshop).toBe(false);
      expect(result.workshopDisabled).toBe(true);
    });
  });

  // -------------------------------------------------------------------------
  // Step 3: Readiness not met
  // -------------------------------------------------------------------------

  describe("readiness not met (step 3)", () => {
    it("canWorkshop=true, canRun=false, primaryCta=workshop", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        readinessReady: false,
      }));
      expect(result.canWorkshop).toBe(true);
      expect(result.canRun).toBe(false);
      expect(result.primaryCta).toBe("workshop");
    });

    it("workshop disabled when agent running", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        readinessReady: false,
        agentRunning: true,
      }));
      expect(result.canWorkshop).toBe(false);
      expect(result.workshopDisabled).toBe(true);
      expect(result.primaryCta).toBe("workshop");
    });
  });

  // -------------------------------------------------------------------------
  // Step 4: Ready
  // -------------------------------------------------------------------------

  describe("ready (step 4)", () => {
    it("canRun=true, canWorkshop=false, primaryCta=run", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "ready" }),
        readinessReady: true,
      }));
      expect(result.canRun).toBe(true);
      expect(result.canWorkshop).toBe(false);
      expect(result.primaryCta).toBe("run");
    });

    it("run disabled when agent running", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "ready" }),
        readinessReady: true,
        agentRunning: true,
      }));
      expect(result.canRun).toBe(false);
      expect(result.runDisabled).toBe(true);
      expect(result.primaryCta).toBe("run");
    });

    it("items with null readiness (no data loaded) are treated as ready if queueable", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "ready" }),
        readinessReady: null,
      }));
      expect(result.canRun).toBe(true);
      expect(result.primaryCta).toBe("run");
    });
  });

  // -------------------------------------------------------------------------
  // Research items
  // -------------------------------------------------------------------------

  describe("research items", () => {
    it("research items follow normal run/workshop CTA funnel", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ kind: "research", status: "backlog" }),
        readinessReady: true,
      }));
      expect(result.canRun).toBe(true);
      expect(result.primaryCta).toBe("run");
    });

    it("research items show workshop when not ready", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ kind: "research", status: "backlog" }),
        readinessReady: false,
      }));
      expect(result.canWorkshop).toBe(true);
      expect(result.primaryCta).toBe("workshop");
    });
  });

  // -------------------------------------------------------------------------
  // Edge cases
  // -------------------------------------------------------------------------

  describe("edge cases", () => {
    it("archived idea is queueable (step 4)", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ kind: "idea", status: "archived" }),
        readinessReady: null,
      }));
      expect(result.canRun).toBe(true);
      expect(result.primaryCta).toBe("run");
    });

    it("item with empty allItems: blocking deps not checked", () => {
      const item = makeItem({ dependsOn: ["idea/something"] });
      const result = getItemActions(makeCtx({ item, allItems: [] }));
      expect(result.blocked).toBe(false);
    });

    it("notQueueableReason is null for queueable items", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
      }));
      expect(result.notQueueableReason).toBeNull();
    });
  });
});
