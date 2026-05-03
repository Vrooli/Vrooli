import { describe, it, expect, vi, beforeEach } from "vitest";
import type { IApiClient } from "../lib/api-client";
import { ApiError } from "../lib/api-client";
import {
  createInitiativeModeService,
  parseActiveItemExecutionsConflict,
  type IInitiativeModeService,
} from "./initiative-mode-service";

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
        capabilities: {
          supports_phases: true,
          can_start_phases: true,
          can_complete_items: true,
          can_apply_backlog_sync_proposals: true,
          requires_acceptance_criteria: true,
          supports_artifacts: true,
          supports_handoffs: false,
          uses_item_execution_flow: false,
        },
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
    expect(workspace.definition.capabilities.canStartPhases).toBe(true);
    expect(workspace.definition.capabilities.canApplyBacklogSyncProposals).toBe(true);
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
        capabilities: {
          supports_phases: true,
          can_start_phases: true,
          can_complete_items: true,
          can_apply_backlog_sync_proposals: true,
          requires_acceptance_criteria: true,
          supports_artifacts: true,
          supports_handoffs: true,
          uses_item_execution_flow: false,
        },
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
    expect(catalog.modes[0]?.capabilities.supportsHandoffs).toBe(true);
    expect(catalog.modes[0]?.capabilities.canStartPhases).toBe(true);
    expect(catalog.modes[0]?.phases[0]?.profileKey).toBe("swarm-manager/analysis");
    expect(catalog.modes[0]?.phases[0]?.requiresCriteria).toBe(true);
  });

  it("normalizes catalog decision-support metadata", async () => {
    vi.mocked(api.get).mockResolvedValue({
      modes: [
        {
          mode: "item-level",
          label: "Item Level",
          description: "Default mode",
          best_for: ["Right-sized items", "Loose coupling"],
          not_for: ["Coupled work"],
          tradeoffs: ["Highest parallelism"],
          // No when_in_doubt_pick_instead — item-level is the safe default.
          scope_kind: "backlog_item",
          run_strategy: "existing_item_flow",
          workspace_tab_id: "info",
          capabilities: {},
          default: true,
          switchable: true,
          supports_phases: false,
          phases: [],
        },
        {
          mode: "holistic-loop",
          label: "Holistic Loop",
          description: "investigate→plan→execute",
          best_for: ["Coupled work"],
          not_for: ["Independent items"],
          tradeoffs: ["One plan, not N"],
          when_in_doubt_pick_instead: "item-level",
          scope_kind: "initiative",
          run_strategy: "operator_gated_loop",
          workspace_tab_id: "operating-mode",
          capabilities: {},
          default: false,
          switchable: true,
          supports_phases: true,
          phases: [],
        },
      ],
    });

    const catalog = await service.catalog();
    expect(catalog.modes[0]?.bestFor).toEqual(["Right-sized items", "Loose coupling"]);
    expect(catalog.modes[0]?.notFor).toEqual(["Coupled work"]);
    expect(catalog.modes[0]?.tradeoffs).toEqual(["Highest parallelism"]);
    expect(catalog.modes[0]?.whenInDoubtPickInstead).toBeUndefined();
    expect(catalog.modes[1]?.bestFor).toEqual(["Coupled work"]);
    expect(catalog.modes[1]?.whenInDoubtPickInstead).toBe("item-level");
  });

  it("returns empty arrays for missing decision metadata fields", async () => {
    vi.mocked(api.get).mockResolvedValue({
      modes: [{
        mode: "holistic-loop",
        label: "Holistic Loop",
        scope_kind: "initiative",
        run_strategy: "operator_gated_loop",
        workspace_tab_id: "operating-mode",
        capabilities: {},
        default: false,
        switchable: true,
        supports_phases: true,
        phases: [],
      }],
    });

    const catalog = await service.catalog();
    expect(catalog.modes[0]?.bestFor).toEqual([]);
    expect(catalog.modes[0]?.notFor).toEqual([]);
    expect(catalog.modes[0]?.tradeoffs).toEqual([]);
    expect(catalog.modes[0]?.whenInDoubtPickInstead).toBeUndefined();
  });

  it("normalizes catalog usage_count and description", async () => {
    vi.mocked(api.get).mockResolvedValue({
      modes: [{
        mode: "holistic-loop",
        label: "Holistic Loop",
        description: "Investigate→plan→execute cycles",
        usage_count: 3,
        scope_kind: "initiative",
        run_strategy: "operator_gated_loop",
        workspace_tab_id: "operating-mode",
        capabilities: {},
        default: false,
        switchable: true,
        supports_phases: true,
        phases: [],
      }],
    });

    const catalog = await service.catalog();
    expect(catalog.modes[0]?.description).toBe("Investigate→plan→execute cycles");
    expect(catalog.modes[0]?.usageCount).toBe(3);
  });

  it("fetches mode detail with linked initiatives", async () => {
    vi.mocked(api.get).mockResolvedValue({
      entry: {
        mode: "holistic-loop",
        label: "Holistic Loop",
        description: "desc",
        usage_count: 2,
        scope_kind: "initiative",
        run_strategy: "operator_gated_loop",
        workspace_tab_id: "operating-mode",
        capabilities: {},
        default: false,
        switchable: true,
        supports_phases: true,
        phases: [],
      },
      linked_initiatives: [
        { name: "init-a", title: "Init A", status: "active", updated: "2026-04-30" },
        { name: "init-b", title: "Init B", status: "active", updated: "2026-04-29" },
      ],
    });

    const detail = await service.getMode("holistic-loop");
    expect(api.get).toHaveBeenCalledWith("/operating-modes/holistic-loop");
    expect(detail.entry.label).toBe("Holistic Loop");
    expect(detail.linkedInitiatives).toHaveLength(2);
    expect(detail.linkedInitiatives[0]?.name).toBe("init-a");
  });

  it("updates mode via PATCH and normalizes the response", async () => {
    vi.mocked(api.patch).mockResolvedValue({
      entry: {
        mode: "holistic-loop",
        label: "Renamed",
        description: "New text",
        usage_count: 0,
        scope_kind: "initiative",
        run_strategy: "operator_gated_loop",
        workspace_tab_id: "operating-mode",
        capabilities: {},
        default: false,
        switchable: true,
        supports_phases: true,
        phases: [],
      },
      linked_initiatives: [],
    });

    const detail = await service.updateMode("holistic-loop", { label: "Renamed", description: "New text" });
    expect(api.patch).toHaveBeenCalledWith(
      "/operating-modes/holistic-loop",
      { label: "Renamed", description: "New text" },
    );
    expect(detail.entry.label).toBe("Renamed");
    expect(detail.entry.description).toBe("New text");
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

describe("parseActiveItemExecutionsConflict", () => {
  it("returns the parsed payload for a 409 active-item-executions error", () => {
    const error = new ApiError("http", "active executions", {
      status: 409,
      code: "active_item_executions",
      details: {
        initiative_name: "initiative-a",
        from_mode: "item-level",
        to_mode: "holistic-loop",
        active_item_executions: [
          { item_ref: "fix:auth-cookie", execution_id: "exec-1", run_id: "run-aaaa-bbbb", status: "running" },
          { item_ref: "feat:onboarding", run_id: "run-cccc-dddd", status: "running" },
        ],
      },
    });

    const conflict = parseActiveItemExecutionsConflict(error);
    expect(conflict).not.toBeNull();
    expect(conflict?.initiativeName).toBe("initiative-a");
    expect(conflict?.fromMode).toBe("item-level");
    expect(conflict?.toMode).toBe("holistic-loop");
    expect(conflict?.executions).toHaveLength(2);
    expect(conflict?.executions[0]).toMatchObject({
      itemRef: "fix:auth-cookie",
      executionId: "exec-1",
      runId: "run-aaaa-bbbb",
      status: "running",
    });
    expect(conflict?.executions[1]).toMatchObject({
      itemRef: "feat:onboarding",
      runId: "run-cccc-dddd",
    });
  });

  it("returns null for non-ApiError values", () => {
    expect(parseActiveItemExecutionsConflict(new Error("boom"))).toBeNull();
    expect(parseActiveItemExecutionsConflict("string error")).toBeNull();
    expect(parseActiveItemExecutionsConflict(null)).toBeNull();
  });

  it("returns null for ApiErrors that aren't 409 active-item-executions", () => {
    const wrongStatus = new ApiError("http", "lock held", {
      status: 409,
      code: "active_operating_mode_round",
      details: { initiative_name: "initiative-a" },
    });
    expect(parseActiveItemExecutionsConflict(wrongStatus)).toBeNull();

    const wrong404 = new ApiError("http", "not found", { status: 404 });
    expect(parseActiveItemExecutionsConflict(wrong404)).toBeNull();
  });

  it("returns null when details lacks the executions array", () => {
    const error = new ApiError("http", "conflict", {
      status: 409,
      code: "active_item_executions",
      details: { initiative_name: "initiative-a" },
    });
    expect(parseActiveItemExecutionsConflict(error)).toBeNull();
  });
});

describe("Initiative Mode Service phase_kind normalization", () => {
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

  it("normalizes phase_kind on workspace and catalog phases", async () => {
    vi.mocked(api.get).mockResolvedValueOnce({
      initiative_name: "initiative-a",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        scope_kind: "initiative",
        run_strategy: "operator_gated_loop",
        capabilities: {},
        phases: [
          {
            phase: "investigate",
            phase_kind: "investigate",
            activity_purpose: "holistic_loop_investigate",
            profile_key: "swarm-manager/deep-work",
            writes_repo: false,
          },
          {
            phase: "execute",
            phase_kind: "execute",
            activity_purpose: "holistic_loop_execute",
            profile_key: "swarm-manager/deep-work",
            writes_repo: true,
            auto_start_after: ["plan"],
          },
        ],
        terminal: ["review"],
        transitions: {},
      },
      artifacts: [],
      rounds: [],
    });

    const workspace = await service.workspace("initiative-a");
    expect(workspace.definition.phases[0]?.phaseKind).toBe("investigate");
    expect(workspace.definition.phases[1]?.phaseKind).toBe("execute");
    expect(workspace.definition.phases[1]?.autoStartAfter).toEqual(["plan"]);

    vi.mocked(api.get).mockResolvedValueOnce({
      modes: [{
        mode: "holistic-loop",
        label: "Holistic Loop",
        scope_kind: "initiative",
        run_strategy: "operator_gated_loop",
        workspace_tab_id: "operating-mode",
        capabilities: {},
        default: false,
        switchable: true,
        supports_phases: true,
        phases: [
          { phase: "investigate", phase_kind: "investigate" },
          { phase: "review", phase_kind: "review", auto_start_after: ["execute"] },
          { phase: "weird", phase_kind: "fabricated-kind" },
        ],
      }],
    });
    const catalog = await service.catalog();
    expect(catalog.modes[0]?.phases[0]?.phaseKind).toBe("investigate");
    expect(catalog.modes[0]?.phases[1]?.phaseKind).toBe("review");
    expect(catalog.modes[0]?.phases[1]?.autoStartAfter).toEqual(["execute"]);
    // Unknown kinds collapse to empty rather than silently passing through
    // a malformed value to the lane-aware UI surfaces.
    expect(catalog.modes[0]?.phases[2]?.phaseKind).toBe("");
  });
});
