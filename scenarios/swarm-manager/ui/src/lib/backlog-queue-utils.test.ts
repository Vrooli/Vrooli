import { describe, it, expect } from "vitest";
import {
  computeDependencyRelations,
  getBacklogNotQueueableReason,
  getItemActions,
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
    blockingInfo: null,
    readinessReady: null,
    pendingSynthesis: false,
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
  const nonQueueableStatuses: BacklogStatus[] = ["queued", "in_progress", "completed", "failed"];
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
        expect(isBacklogQueueable({ kind, status })).toBe(false);
      }
    }
  });

  it("returns true for research items in queueable statuses", () => {
    for (const status of queueableStatuses) {
      expect(isBacklogQueueable({ kind: "research", status })).toBe(true);
    }
  });

  it("returns true for archived ideas (special case via archivedAt)", () => {
    expect(isBacklogQueueable({ kind: "idea", status: "completed", archivedAt: "2026-01-01T00:00:00Z" })).toBe(true);
  });
});

// ---------------------------------------------------------------------------
// getBacklogNotQueueableReason
// ---------------------------------------------------------------------------

describe("getBacklogNotQueueableReason", () => {
  it("returns null for queueable items", () => {
    expect(getBacklogNotQueueableReason({ kind: "idea", status: "backlog" })).toBeNull();
    expect(getBacklogNotQueueableReason({ kind: "idea", status: "ready" })).toBeNull();
    expect(getBacklogNotQueueableReason({ kind: "idea", status: "completed", archivedAt: "2026-01-01T00:00:00Z" })).toBeNull();
  });

  it("returns null for research items in queueable statuses", () => {
    expect(getBacklogNotQueueableReason({ kind: "research", status: "backlog" })).toBeNull();
  });

  it("returns status-specific reasons for non-queueable statuses", () => {
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "queued" })).toContain("Already queued");
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "in_progress" })).toContain("Already in progress");
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "completed" })).toContain("cannot be queued again");
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "failed" })).toContain("Reset status");
    expect(getBacklogNotQueueableReason({ kind: "fix", status: "completed" })).toContain("cannot be queued again");
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
    it("sets blocked=true but CTAs remain available when blocked and item is ready", () => {
      const item = makeItem({ status: "ready", dependsOn: ["idea/dep"] });
      const result = getItemActions(makeCtx({
        item,
        blockingInfo: { blocked: true, blockingDepKeys: ["idea/dep"], allForceable: true },
        readinessReady: true,
      }));
      expect(result.blocked).toBe(true);
      // CTAs are available (not hard-disabled) so user can override via modal
      expect(result.canRun).toBe(true);
      expect(result.primaryCta).toBe("run");
    });

    it("sets blocked=true but workshop still available when readiness not met", () => {
      const item = makeItem({ status: "backlog", dependsOn: ["idea/dep"] });
      const result = getItemActions(makeCtx({
        item,
        blockingInfo: { blocked: true, blockingDepKeys: ["idea/dep"], allForceable: true },
        readinessReady: false,
      }));
      expect(result.blocked).toBe(true);
      expect(result.canWorkshop).toBe(true);
      expect(result.primaryCta).toBe("workshop");
    });

    it("populates blockingDepKeys from server-provided blockingInfo", () => {
      const item = makeItem({ status: "backlog", dependsOn: ["idea/blocker"] });
      const result = getItemActions(makeCtx({
        item,
        blockingInfo: { blocked: true, blockingDepKeys: ["idea/blocker"], allForceable: false },
      }));
      expect(result.blockingDepKeys).toEqual(["idea/blocker"]);
    });

    it("still shows decision stepper when blocked with pending decisions", () => {
      const item = makeItem({ status: "backlog", dependsOn: ["idea/dep"] });
      const result = getItemActions(makeCtx({
        item,
        blockingInfo: { blocked: true, blockingDepKeys: ["idea/dep"], allForceable: true },
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

    it("workshop blocked when pending decisions exist, even if not ready", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        hasPendingDecisions: true,
        readinessReady: false,
      }));
      expect(result.showDecisionStepper).toBe(true);
      expect(result.canWorkshop).toBe(false);
      expect(result.primaryCta).toBeNull();
    });

    it("workshop blocked when readiness is met with pending decisions", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "ready" }),
        hasPendingDecisions: true,
        readinessReady: true,
      }));
      expect(result.showDecisionStepper).toBe(true);
      expect(result.canWorkshop).toBe(false);
      expect(result.primaryCta).toBeNull();
    });

    it("workshop blocked when agent running with pending decisions", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        hasPendingDecisions: true,
        readinessReady: false,
        agentRunning: true,
      }));
      expect(result.showDecisionStepper).toBe(true);
      expect(result.canWorkshop).toBe(false);
      expect(result.workshopDisabled).toBe(false);
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
    it("canRun=true, canWorkshop=true, primaryCta=run", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "ready" }),
        readinessReady: true,
      }));
      expect(result.canRun).toBe(true);
      expect(result.canWorkshop).toBe(true);
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
      expect(result.canWorkshop).toBe(true);
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
    it("archived items have no actions — must unarchive first", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ kind: "idea", status: "completed", archivedAt: "2026-01-01T00:00:00Z" }),
        readinessReady: null,
      }));
      expect(result.canRun).toBe(false);
      expect(result.canArchive).toBe(false);
      expect(result.canFollowUp).toBe(false);
      expect(result.canWorkshop).toBe(false);
      expect(result.canFinalize).toBe(false);
      expect(result.primaryCta).toBeNull();
    });

    it("item with null blockingInfo: not blocked", () => {
      const item = makeItem({ dependsOn: ["idea/something"] });
      const result = getItemActions(makeCtx({ item, blockingInfo: null }));
      expect(result.blocked).toBe(false);
    });

    it("notQueueableReason is null for queueable items", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
      }));
      expect(result.notQueueableReason).toBeNull();
    });
  });

  describe("pending synthesis", () => {
    it("shows finalize as primary when latest answers are ready but unsynthesized", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "ready" }),
        readinessReady: true,
        pendingSynthesis: true,
      }));
      expect(result.canFinalize).toBe(true);
      expect(result.canRun).toBe(false);
      expect(result.canWorkshop).toBe(true);
      expect(result.primaryCta).toBe("finalize");
    });

    it("shows workshop as primary when latest answers are unsynthesized and still not ready", () => {
      const result = getItemActions(makeCtx({
        item: makeItem({ status: "backlog" }),
        readinessReady: false,
        pendingSynthesis: true,
      }));
      expect(result.canFinalize).toBe(false);
      expect(result.canWorkshop).toBe(true);
      expect(result.primaryCta).toBe("workshop");
    });
  });
});

// ---------------------------------------------------------------------------
// computeDependencyRelations
// ---------------------------------------------------------------------------

describe("computeDependencyRelations", () => {
  it("returns empty arrays when item has no deps and nothing depends on it", () => {
    const item = makeItem({ name: "solo", kind: "idea" });
    const result = computeDependencyRelations(item, [item]);
    expect(result.parents).toEqual([]);
    expect(result.children).toEqual([]);
  });

  it("resolves parents from dependsOn", () => {
    const parent1 = makeItem({ name: "p1", kind: "fix", title: "Parent One", status: "ready" });
    const parent2 = makeItem({ name: "p2", kind: "chore", title: "Parent Two", status: "completed" });
    const item = makeItem({ name: "child", kind: "idea", dependsOn: ["fix/p1", "chore/p2"] });
    const result = computeDependencyRelations(item, [parent1, parent2, item]);
    expect(result.parents).toEqual([
      { kind: "fix", name: "p1", title: "Parent One", status: "ready" },
      { kind: "chore", name: "p2", title: "Parent Two", status: "completed" },
    ]);
  });

  it("returns dangling refs with completed status", () => {
    const item = makeItem({ name: "orphan", kind: "idea", dependsOn: ["fix/gone"] });
    const result = computeDependencyRelations(item, [item]);
    expect(result.parents).toEqual([
      { kind: "fix", name: "gone", title: "fix/gone", status: "completed" },
    ]);
  });

  it("computes children (reverse lookup)", () => {
    const parent = makeItem({ name: "parent", kind: "fix", title: "The Parent", status: "ready" });
    const child1 = makeItem({ name: "c1", kind: "idea", title: "Child 1", status: "backlog", dependsOn: ["fix/parent"] });
    const child2 = makeItem({ name: "c2", kind: "chore", title: "Child 2", status: "researching", dependsOn: ["fix/parent"] });
    const unrelated = makeItem({ name: "other", kind: "idea" });
    const result = computeDependencyRelations(parent, [parent, child1, child2, unrelated]);
    expect(result.children).toEqual([
      { kind: "idea", name: "c1", title: "Child 1", status: "backlog" },
      { kind: "chore", name: "c2", title: "Child 2", status: "researching" },
    ]);
  });

  it("filters out self-references", () => {
    const item = makeItem({ name: "self", kind: "idea", dependsOn: ["idea/self"] });
    const result = computeDependencyRelations(item, [item]);
    expect(result.parents).toEqual([]);
    expect(result.children).toEqual([]);
  });

  it("skips malformed dependsOn entries (no slash)", () => {
    const item = makeItem({ name: "bad", kind: "idea", dependsOn: ["noslash"] });
    const result = computeDependencyRelations(item, [item]);
    expect(result.parents).toEqual([]);
  });

  it("uses name as title fallback when title is empty", () => {
    const parent = makeItem({ name: "notitle", kind: "fix", title: "", status: "backlog" });
    const item = makeItem({ name: "child", kind: "idea", dependsOn: ["fix/notitle"] });
    const result = computeDependencyRelations(item, [parent, item]);
    expect(result.parents[0]?.title).toBe("notitle");
  });
});
