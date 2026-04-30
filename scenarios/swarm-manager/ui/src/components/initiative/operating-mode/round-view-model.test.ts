import { describe, expect, it } from "vitest";
import type { OperatingModeRound } from "../../../types/operating-mode";
import {
  backlogSyncActionUnavailableReason,
  buildRoundViewModel,
  canApplyBacklogProposal,
  hasAppliedBacklogSync,
  mutationSummary,
  pendingBacklogProposal,
  pendingCompletedItems,
} from "./round-view-model";

function round(overrides: Partial<OperatingModeRound> = {}): OperatingModeRound {
  return {
    round: 1,
    mode: "holistic-loop",
    scopeKind: "initiative",
    scopeId: "initiative-a",
    phase: "execute",
    runStrategy: "operator_gated_loop",
    agentProfileKey: "swarm-manager/deep-work",
    generatedAt: "2026-04-30T00:00:00Z",
    runId: "run-1",
    status: "completed",
    ...overrides,
  };
}

describe("round view model", () => {
  it("extracts pending completed item refs from snake_case and camelCase plans", () => {
    expect(pendingCompletedItems(round({
      payload: { backlog_sync_plan: { completed_items: ["execute/a", 7, "fix/b"] } },
    }))).toEqual(["execute/a", "fix/b"]);
    expect(pendingCompletedItems(round({
      payload: { backlog_sync_plan: { completedItems: ["execute/c"] } },
    }))).toEqual(["execute/c"]);
  });

  it("hides pending sync actions after sync has been applied", () => {
    const syncedRound = round({
      payload: {
        backlog_sync_plan: {
          completed_items: ["execute/a"],
          proposal: {
            form: "mutation_list",
            mutations: [{ id: "m1", op: "add_item" }],
          },
        },
        backlog_sync: {
          initiativeName: "initiative-a",
          mode: "holistic-loop",
          phase: "execute",
          round: 1,
          completedItems: [],
        },
      },
    });

    expect(hasAppliedBacklogSync(syncedRound)).toBe(true);
    expect(pendingCompletedItems(syncedRound)).toEqual([]);
    expect(pendingBacklogProposal(syncedRound)).toBeUndefined();
  });

  it("normalizes mutation-list proposals and ignores malformed mutations", () => {
    const proposal = pendingBacklogProposal(round({
      payload: {
        backlog_sync_plan: {
          proposal: {
            form: "mutation_list",
            rationale: "Clean up backlog",
            mutations: [
              { id: "m1", op: "add_item", item: { kind: "fix", name: "follow-up", title: "Follow up" } },
              { id: 2, op: "change_status" },
              { id: "m3" },
            ],
          },
        },
      },
    }));

    expect(proposal?.rationale).toBe("Clean up backlog");
    expect(proposal?.mutations.map((mutation) => mutation.id)).toEqual(["m1"]);
  });

  it("derives sync action availability from backend state and run ownership", () => {
    const missingRun = round({
      runId: undefined,
      payload: { backlog_sync_plan: { completed_items: ["execute/a"] } },
    });

    expect(buildRoundViewModel(missingRun).canCompleteItems).toBe(false);
    expect(backlogSyncActionUnavailableReason(missingRun)).toMatch(/missing an AgentManager run ID/i);
    expect(canApplyBacklogProposal(missingRun, new Set(["m1"]))).toBe(false);
  });

  it("builds default proposal selection and action flags for completed rounds", () => {
    const view = buildRoundViewModel(round({
      payload: {
        agent_summary: "Execution complete",
        backlog_sync_plan: {
          completed_items: ["execute/a"],
          proposal: {
            form: "mutation_list",
            mutations: [
              { id: "m1", op: "add_item" },
              { id: "m2", op: "change_status", target: "execute/a", status: "completed" },
            ],
          },
        },
      },
    }));

    expect(view.summary).toBe("Execution complete");
    expect(view.canCompleteItems).toBe(true);
    expect(view.defaultSelectedMutationIds).toEqual(["m1", "m2"]);
  });

  it("summarizes mutation details without requiring React rendering", () => {
    expect(mutationSummary({
      id: "m1",
      op: "change_status",
      target: "execute/a",
      patch: { priority: 2 },
      status: "completed",
      priority: 2,
    })).toContain("execute/a");
  });
});
