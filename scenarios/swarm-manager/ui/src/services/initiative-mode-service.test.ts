import { describe, it, expect, vi, beforeEach } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { createInitiativeModeService, type IInitiativeModeService } from "./initiative-mode-service";

describe("Initiative Mode Service", () => {
  let api: IApiClient;
  let service: IInitiativeModeService;

  beforeEach(() => {
    api = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };
    service = createInitiativeModeService(api);
  });

  it("normalizes workspace snake_case fields", async () => {
    vi.mocked(api.get).mockResolvedValue({
      initiative_name: "initiative-a",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        label: "Holistic Loop",
        scope_kind: "initiative",
        run_strategy: "operator_gated_loop",
        phases: [{
          phase: "investigate",
          activity_purpose: "holistic_loop_investigate",
          profile_key: "swarm-manager/deep-work",
          writes_repo: false,
          output_artifacts: [{
            path: "modes/holistic-loop/findings.md",
            content_type: "text/markdown",
            required: true,
          }],
        }],
        terminal: ["review"],
        transitions: { investigate: ["plan"] },
      },
      artifacts: [{
        path: "modes/holistic-loop/findings.md",
        content_type: "text/markdown",
        required: true,
        updated_at: "2026-04-30T00:00:00Z",
        size_bytes: 42,
        content: "# Findings",
      }],
      rounds: [{
        round: 1,
        mode: "holistic-loop",
        scope_kind: "initiative",
        scope_id: "initiative-a",
        initiative_name: "initiative-a",
        phase: "investigate",
        run_strategy: "operator_gated_loop",
        agent_profile_key: "swarm-manager/deep-work",
        generated_at: "2026-04-30T00:00:00Z",
        run_id: "run-1",
        status: "completed",
        items: [{ ref: "execute/item-1", title: "Item 1", priority: 1 }],
        payload: { agent_summary: "done" },
      }],
    });

    const workspace = await service.workspace("initiative-a");

    expect(api.get).toHaveBeenCalledWith("/initiatives/initiative-a/operating-mode/workspace");
    expect(workspace.initiativeName).toBe("initiative-a");
    expect(workspace.definition.scopeKind).toBe("initiative");
    expect(workspace.definition.runStrategy).toBe("operator_gated_loop");
    expect(workspace.definition.phases[0]?.profileKey).toBe("swarm-manager/deep-work");
    expect(workspace.artifacts[0]?.updatedAt).toBe("2026-04-30T00:00:00Z");
    expect(workspace.artifacts[0]?.sizeBytes).toBe(42);
    expect(workspace.rounds[0]?.agentProfileKey).toBe("swarm-manager/deep-work");
    expect(workspace.rounds[0]?.items?.[0]?.ref).toBe("execute/item-1");
  });

  it("normalizes operating-mode catalog entries", async () => {
    vi.mocked(api.get).mockResolvedValue({
      modes: [{
        mode: "custom-audit-loop",
        label: "Custom Audit Loop",
        scope_kind: "initiative",
        run_strategy: "operator_gated_loop",
        workspace_tab_id: "operating-mode",
        default: false,
        switchable: true,
        supports_phases: true,
        phases: [{
          phase: "audit",
          profile_key: "swarm-manager/analysis",
          writes_repo: false,
          requires_criteria: true,
        }],
      }],
    });

    const catalog = await service.catalog();

    expect(api.get).toHaveBeenCalledWith("/operating-modes");
    expect(catalog.modes[0]?.mode).toBe("custom-audit-loop");
    expect(catalog.modes[0]?.scopeKind).toBe("initiative");
    expect(catalog.modes[0]?.runStrategy).toBe("operator_gated_loop");
    expect(catalog.modes[0]?.workspaceTabId).toBe("operating-mode");
    expect(catalog.modes[0]?.supportsPhases).toBe(true);
    expect(catalog.modes[0]?.phases[0]?.profileKey).toBe("swarm-manager/analysis");
    expect(catalog.modes[0]?.phases[0]?.requiresCriteria).toBe(true);
  });

  it("starts, refreshes, and cancels rounds through canonical endpoints", async () => {
    vi.mocked(api.post).mockResolvedValue({
      round: 2,
      mode: "phased-plan-drain",
      phase: "execute_next",
      scope_kind: "initiative",
      scope_id: "initiative-a",
      run_strategy: "sequential_handoff",
      agent_profile_key: "swarm-manager/deep-work",
      generated_at: "2026-04-30T00:00:00Z",
      status: "agent_running",
    });

    await service.startPhase("initiative-a", "execute_next", { note: "continue", override: true });
    expect(api.post).toHaveBeenNthCalledWith(
      1,
      "/initiatives/initiative-a/operating-mode/phases/execute_next/start",
      expect.objectContaining({ note: "continue", override: true, requested_by: "swarm-manager-ui" }),
    );

    await service.refreshRound("initiative-a", "phased-plan-drain", 2);
    expect(api.post).toHaveBeenNthCalledWith(
      2,
      "/initiatives/initiative-a/operating-mode/rounds/2/refresh?mode=phased-plan-drain",
      {},
    );

    await service.cancelRound("initiative-a", "phased-plan-drain", 2);
    expect(api.post).toHaveBeenNthCalledWith(
      3,
      "/initiatives/initiative-a/operating-mode/rounds/2/cancel?mode=phased-plan-drain",
      {},
    );
  });

  it("switches modes through the operating-mode lifecycle endpoint", async () => {
    vi.mocked(api.post).mockResolvedValue({
      initiative_name: "initiative-a",
      from_mode: "item-level",
      to_mode: "holistic-loop",
      canceled_item_executions: [{
        item_ref: "execute/item-1",
        execution_id: "exec-1",
        run_id: "run-1",
        status: "canceled",
      }],
    });

    const result = await service.switchMode("initiative-a", {
      mode: "holistic-loop",
      cancelActiveItemExecutions: true,
    });

    expect(api.post).toHaveBeenCalledWith(
      "/initiatives/initiative-a/operating-mode/switch",
      {
        mode: "holistic-loop",
        cancel_active_item_executions: true,
        requested_by: "swarm-manager-ui",
      },
    );
    expect(result.fromMode).toBe("item-level");
    expect(result.toMode).toBe("holistic-loop");
    expect(result.canceledItemExecutions?.[0]?.executionId).toBe("exec-1");
  });

  it("marks operating-mode round items complete through the reconciliation endpoint", async () => {
    vi.mocked(api.post).mockResolvedValue({
      initiative_name: "initiative-a",
      mode: "holistic-loop",
      phase: "execute",
      round: 3,
      run_id: "run-3",
      completed_items: [{
        item_ref: "execute/item-1",
        from_status: "ready",
        to_status: "completed",
      }],
    });

    const result = await service.completeItems("initiative-a", {
      mode: "holistic-loop",
      round: 3,
      runId: "run-3",
      itemRefs: ["execute/item-1"],
    });

    expect(api.post).toHaveBeenCalledWith(
      "/initiatives/initiative-a/operating-mode/rounds/3/complete-items?mode=holistic-loop",
      {
        mode: "holistic-loop",
        run_id: "run-3",
        item_refs: ["execute/item-1"],
        requested_by: "swarm-manager-ui",
      },
    );
    expect(result.completedItems[0]?.itemRef).toBe("execute/item-1");
  });

  it("applies operating-mode backlog proposal mutations through the reconciliation endpoint", async () => {
    vi.mocked(api.post).mockResolvedValue({
      initiative_name: "initiative-a",
      mode: "phased-plan-drain",
      phase: "classify_progress",
      round: 4,
      run_id: "run-4",
      proposal_result: {
        applied: 1,
        failed: 0,
        skipped: 1,
        created: 1,
        updated: 0,
        outcomes: [{
          mutation_id: "m1",
          op: "add_item",
          target: "fix/follow-up",
          applied: true,
        }],
      },
    });

    const result = await service.applyBacklogSync("initiative-a", {
      mode: "phased-plan-drain",
      round: 4,
      runId: "run-4",
      acceptedMutationIds: ["m1"],
    });

    expect(api.post).toHaveBeenCalledWith(
      "/initiatives/initiative-a/operating-mode/rounds/4/apply-backlog-sync?mode=phased-plan-drain",
      {
        mode: "phased-plan-drain",
        run_id: "run-4",
        accepted_mutation_ids: ["m1"],
        requested_by: "swarm-manager-ui",
      },
    );
    expect(result.proposalResult?.applied).toBe(1);
    expect(result.proposalResult?.outcomes?.[0]?.mutationId).toBe("m1");
  });
});
