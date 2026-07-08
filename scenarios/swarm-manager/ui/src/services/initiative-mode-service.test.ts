import { describe, it, expect, vi, beforeEach } from "vitest";
import { ConnectError, Code } from "@connectrpc/connect";
import { OperatingModeActiveItemExecutionsConflictSchema } from "@vrooli/proto-types/swarm-manager/v1/api/operating_mode_pb";
import {
  createInitiativeModeService,
  parseActiveItemExecutionsConflict,
  type IInitiativeModeService,
  type OperatingModeClient,
} from "./initiative-mode-service";

// The service consumes the generated OperatingModeService Connect client and
// projects its (camelCase) proto messages onto the domain types. These tests
// drive it through a mock client: each RPC is a vi.fn() returning a
// proto-shaped response, so we assert both the request the service builds and
// the domain shape it maps back.
const RPCS = [
  "catalog",
  "getMode",
  "updateMode",
  "simulateMode",
  "renderSimulationPrompt",
  "renderLivePrompt",
  "getWorkspace",
  "switchMode",
  "startPhase",
  "refreshRound",
  "cancelRound",
  "completeItems",
  "applyBacklogSync",
  "scaffoldMode",
  "validateMode",
] as const;

type MockClient = Record<(typeof RPCS)[number], ReturnType<typeof vi.fn>>;

function makeClient(): MockClient {
  const client = {} as MockClient;
  for (const rpc of RPCS) client[rpc] = vi.fn();
  return client;
}

describe("Initiative Mode Service", () => {
  let client: MockClient;
  let service: IInitiativeModeService;

  beforeEach(() => {
    client = makeClient();
    service = createInitiativeModeService(client as unknown as OperatingModeClient);
  });

  it("maps a workspace projection", async () => {
    client.getWorkspace.mockResolvedValue({
      initiativeName: "initiative-a",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        label: "Holistic Loop",
        scopeKind: "initiative",
        runStrategy: "operator_gated_loop",
        capabilities: {
          supportsPhases: true,
          canStartPhases: true,
          canCompleteItems: true,
          canApplyBacklogSyncProposals: true,
          requiresAcceptanceCriteria: true,
          supportsArtifacts: true,
        },
        phases: [{
          phase: "investigate",
          activityPurpose: "holistic_loop_investigate",
          profileKey: "swarm-manager/deep-work",
          writesRepo: false,
          outputArtifacts: [{
            path: "modes/holistic-loop/findings.md",
            contentType: "text/markdown",
            required: true,
          }],
        }],
        terminal: ["review"],
        transitions: { investigate: { values: ["plan"] } },
      },
      artifacts: [{
        path: "modes/holistic-loop/findings.md",
        contentType: "text/markdown",
        required: true,
        updatedAt: "2026-04-30T00:00:00Z",
        sizeBytes: 42n,
        content: "# Findings",
      }],
      rounds: [{
        round: 1,
        mode: "holistic-loop",
        scopeKind: "initiative",
        scopeId: "initiative-a",
        initiativeName: "initiative-a",
        phase: "investigate",
        runStrategy: "operator_gated_loop",
        agentProfileKey: "swarm-manager/deep-work",
        generatedAt: "2026-04-30T00:00:00Z",
        runId: "run-1",
        status: "completed",
        items: [{ ref: "execute/item-1", title: "Item 1", priority: 1 }],
        payload: { agent_summary: "done" },
      }],
    });

    const workspace = await service.workspace("initiative-a");

    expect(client.getWorkspace).toHaveBeenCalledWith({ initiativeName: "initiative-a" });
    expect(workspace.initiativeName).toBe("initiative-a");
    expect(workspace.definition.scopeKind).toBe("initiative");
    expect(workspace.definition.runStrategy).toBe("operator_gated_loop");
    expect(workspace.definition.capabilities.canStartPhases).toBe(true);
    expect(workspace.definition.capabilities.canApplyBacklogSyncProposals).toBe(true);
    expect(workspace.definition.transitions.investigate).toEqual(["plan"]);
    expect(workspace.definition.phases[0]?.profileKey).toBe("swarm-manager/deep-work");
    expect(workspace.artifacts[0]?.updatedAt).toBe("2026-04-30T00:00:00Z");
    // int64 size_bytes arrives as bigint and is coerced to a number.
    expect(workspace.artifacts[0]?.sizeBytes).toBe(42);
    expect(workspace.rounds[0]?.agentProfileKey).toBe("swarm-manager/deep-work");
    expect(workspace.rounds[0]?.items?.[0]?.ref).toBe("execute/item-1");
    expect(workspace.rounds[0]?.payload).toEqual({ agent_summary: "done" });
  });

  it("maps catalog entries and capabilities", async () => {
    client.catalog.mockResolvedValue({
      modes: [{
        mode: "custom-audit-loop",
        label: "Custom Audit Loop",
        scopeKind: "initiative",
        runStrategy: "operator_gated_loop",
        workspaceTabId: "operating-mode",
        capabilities: {
          supportsPhases: true,
          canStartPhases: true,
          supportsHandoffs: true,
        },
        switchable: true,
        supportsPhases: true,
        phases: [{
          phase: "audit",
          profileKey: "swarm-manager/analysis",
          writesRepo: false,
          requiresCriteria: true,
        }],
      }],
    });

    const catalog = await service.catalog();

    expect(client.catalog).toHaveBeenCalledWith({});
    expect(catalog.modes[0]?.mode).toBe("custom-audit-loop");
    expect(catalog.modes[0]?.scopeKind).toBe("initiative");
    expect(catalog.modes[0]?.workspaceTabId).toBe("operating-mode");
    expect(catalog.modes[0]?.supportsPhases).toBe(true);
    expect(catalog.modes[0]?.capabilities.supportsHandoffs).toBe(true);
    expect(catalog.modes[0]?.capabilities.canStartPhases).toBe(true);
    expect(catalog.modes[0]?.phases[0]?.profileKey).toBe("swarm-manager/analysis");
    expect(catalog.modes[0]?.phases[0]?.requiresCriteria).toBe(true);
  });

  it("maps a simulation trace with a generic guard transition", async () => {
    client.simulateMode.mockResolvedValue({
      mode: "phased-plan-drain",
      label: "Phased Plan Drain",
      activePreset: "happy-path",
      presets: [
        {
          id: "happy-path",
          label: "Drains in one slice",
          description: "Prepare → execute → classify → review → reconcile.",
          branch: "classify_progress → review (complete)",
          scenario: "A stable plan a single slice can drain.",
        },
        {
          id: "blocked",
          label: "Work is blocked",
          description: "Classify reports blocked; the cycle ends before review.",
          branch: "classify_progress → (blocked, terminal)",
          scenario: "A drain that stalls on an external blocker.",
        },
      ],
      initiative: {
        name: "simulation-sandbox",
        title: "Phased Plan Drain Simulation",
        mode: "phased-plan-drain",
        items: ["execute/item-1"],
        acceptanceCriteria: ["review output"],
      },
      trace: [{
        index: 0,
        phase: "classify_progress",
        phaseKind: "review",
        inputs: {
          initiative: {
            name: "simulation-sandbox",
            title: "Phased Plan Drain Simulation",
            mode: "phased-plan-drain",
            items: ["execute/item-1"],
            acceptanceCriteria: ["review output"],
          },
          items: [{ ref: "execute/item-1", title: "Item 1" }],
          artifacts: [{ path: "modes/phased-plan-drain/progress.json", required: true }],
          priorRounds: [],
          acceptanceCriteria: ["review output"],
        },
        output: {
          progress: { decision: "complete", currentPhase: "classify_progress", rationale: "ready" },
        },
        round: {
          round: 1,
          mode: "phased-plan-drain",
          scopeKind: "initiative",
          scopeId: "simulation-sandbox",
          initiativeName: "simulation-sandbox",
          phase: "classify_progress",
          runStrategy: "sequential_handoff",
          agentProfileKey: "swarm-manager/analysis",
          generatedAt: "2026-04-30T00:00:00Z",
          status: "completed",
        },
        transition: {
          from: "classify_progress",
          to: "review",
          conditionKind: "eq",
          label: "on progress.decision = complete",
          field: "progress.decision",
          value: "complete",
        },
      }],
    });

    const simulation = await service.simulateMode("phased-plan-drain");

    expect(client.simulateMode).toHaveBeenCalledWith({ mode: "phased-plan-drain", preset: "" });
    expect(simulation.mode).toBe("phased-plan-drain");
    expect(simulation.activePreset).toBe("happy-path");
    expect(simulation.presets.map((preset) => preset.id)).toEqual(["happy-path", "blocked"]);
    expect(simulation.presets[1]?.branch).toBe("classify_progress → (blocked, terminal)");
    expect(simulation.initiative.acceptanceCriteria).toEqual(["review output"]);
    expect(simulation.trace[0]?.phaseKind).toBe("review");
    expect(simulation.trace[0]?.inputs.items[0]?.ref).toBe("execute/item-1");
    expect(simulation.trace[0]?.output.progress?.decision).toBe("complete");
    expect(simulation.trace[0]?.transition?.conditionKind).toBe("eq");
    expect(simulation.trace[0]?.transition?.field).toBe("progress.decision");
    expect(simulation.trace[0]?.transition?.value).toBe("complete");
  });

  it("passes the selected preset id in the request", async () => {
    client.simulateMode.mockResolvedValue({
      mode: "phased-plan-drain",
      activePreset: "blocked",
      presets: [],
      initiative: { name: "simulation-sandbox", mode: "phased-plan-drain", items: [], acceptanceCriteria: [] },
      trace: [],
    });

    const simulation = await service.simulateMode("phased-plan-drain", "blocked");

    expect(client.simulateMode).toHaveBeenCalledWith({ mode: "phased-plan-drain", preset: "blocked" });
    expect(simulation.activePreset).toBe("blocked");
  });

  it("maps decision-support metadata and coerces empty strings to undefined", async () => {
    client.catalog.mockResolvedValue({
      modes: [
        {
          mode: "item-level",
          label: "Item Level",
          description: "Default mode",
          bestFor: ["Right-sized items", "Loose coupling"],
          notFor: ["Coupled work"],
          tradeoffs: ["Highest parallelism"],
          whenInDoubtPickInstead: "", // item-level is the safe default
          scopeKind: "backlog_item",
          runStrategy: "existing_item_flow",
          workspaceTabId: "info",
          capabilities: {},
          default: true,
          switchable: true,
          supportsPhases: false,
          phases: [],
        },
        {
          mode: "holistic-loop",
          label: "Holistic Loop",
          description: "investigate→plan→execute",
          bestFor: ["Coupled work"],
          notFor: ["Independent items"],
          tradeoffs: ["One plan, not N"],
          whenInDoubtPickInstead: "item-level",
          scopeKind: "initiative",
          runStrategy: "operator_gated_loop",
          workspaceTabId: "operating-mode",
          capabilities: {},
          usageCount: 3,
          switchable: true,
          supportsPhases: true,
          phases: [],
        },
      ],
    });

    const catalog = await service.catalog();
    expect(catalog.modes[0]?.bestFor).toEqual(["Right-sized items", "Loose coupling"]);
    expect(catalog.modes[0]?.tradeoffs).toEqual(["Highest parallelism"]);
    expect(catalog.modes[0]?.whenInDoubtPickInstead).toBeUndefined();
    expect(catalog.modes[1]?.whenInDoubtPickInstead).toBe("item-level");
    expect(catalog.modes[1]?.usageCount).toBe(3);
    expect(catalog.modes[1]?.description).toBe("investigate→plan→execute");
  });

  it("returns empty arrays for missing decision metadata", async () => {
    client.catalog.mockResolvedValue({
      modes: [{ mode: "holistic-loop", label: "Holistic Loop", capabilities: {}, phases: [] }],
    });

    const catalog = await service.catalog();
    expect(catalog.modes[0]?.bestFor).toEqual([]);
    expect(catalog.modes[0]?.notFor).toEqual([]);
    expect(catalog.modes[0]?.tradeoffs).toEqual([]);
    expect(catalog.modes[0]?.whenInDoubtPickInstead).toBeUndefined();
  });

  it("fetches mode detail with linked initiatives", async () => {
    client.getMode.mockResolvedValue({
      entry: { mode: "holistic-loop", label: "Holistic Loop", description: "desc", capabilities: {}, phases: [] },
      linkedInitiatives: [
        { name: "init-a", title: "Init A", status: "active", updated: "2026-04-30" },
        { name: "init-b", title: "Init B", status: "active", updated: "2026-04-29" },
      ],
    });

    const detail = await service.getMode("holistic-loop");
    expect(client.getMode).toHaveBeenCalledWith({ mode: "holistic-loop" });
    expect(detail.entry.label).toBe("Holistic Loop");
    expect(detail.linkedInitiatives).toHaveLength(2);
    expect(detail.linkedInitiatives[0]?.name).toBe("init-a");
  });

  it("updates a mode's overlay fields", async () => {
    client.updateMode.mockResolvedValue({
      entry: { mode: "holistic-loop", label: "Renamed", description: "New text", capabilities: {}, phases: [] },
      linkedInitiatives: [],
    });

    const detail = await service.updateMode("holistic-loop", { label: "Renamed", description: "New text" });
    expect(client.updateMode).toHaveBeenCalledWith({
      mode: "holistic-loop",
      label: "Renamed",
      description: "New text",
    });
    expect(detail.entry.label).toBe("Renamed");
    expect(detail.entry.description).toBe("New text");
  });

  it("starts, refreshes, and cancels rounds", async () => {
    const round = {
      round: 2,
      mode: "phased-plan-drain",
      phase: "execute_next",
      scopeKind: "initiative",
      scopeId: "initiative-a",
      runStrategy: "sequential_handoff",
      agentProfileKey: "swarm-manager/deep-work",
      generatedAt: "2026-04-30T00:00:00Z",
      status: "agent_running",
    };
    client.startPhase.mockResolvedValue(round);
    client.refreshRound.mockResolvedValue(round);
    client.cancelRound.mockResolvedValue(round);

    await service.startPhase("initiative-a", "execute_next", { note: "continue", override: true });
    expect(client.startPhase).toHaveBeenCalledWith({
      initiativeName: "initiative-a",
      phase: "execute_next",
      note: "continue",
      override: true,
      requestedBy: "swarm-manager-ui",
    });

    await service.refreshRound("initiative-a", "phased-plan-drain", 2);
    expect(client.refreshRound).toHaveBeenCalledWith({
      initiativeName: "initiative-a",
      mode: "phased-plan-drain",
      round: 2,
    });

    await service.cancelRound("initiative-a", "phased-plan-drain", 2);
    expect(client.cancelRound).toHaveBeenCalledWith({
      initiativeName: "initiative-a",
      mode: "phased-plan-drain",
      round: 2,
    });
  });

  it("switches modes and maps canceled executions", async () => {
    client.switchMode.mockResolvedValue({
      initiativeName: "initiative-a",
      fromMode: "item-level",
      toMode: "holistic-loop",
      canceledItemExecutions: [{
        itemRef: "execute/item-1",
        executionId: "exec-1",
        runId: "run-1",
        status: "canceled",
      }],
    });

    const result = await service.switchMode("initiative-a", {
      mode: "holistic-loop",
      cancelActiveItemExecutions: true,
    });

    expect(client.switchMode).toHaveBeenCalledWith({
      initiativeName: "initiative-a",
      mode: "holistic-loop",
      cancelActiveItemExecutions: true,
      requestedBy: "swarm-manager-ui",
    });
    expect(result.fromMode).toBe("item-level");
    expect(result.toMode).toBe("holistic-loop");
    expect(result.canceledItemExecutions?.[0]?.executionId).toBe("exec-1");
  });

  it("completes round items", async () => {
    client.completeItems.mockResolvedValue({
      initiativeName: "initiative-a",
      mode: "holistic-loop",
      phase: "execute",
      round: 3,
      runId: "run-3",
      completedItems: [{ itemRef: "execute/item-1", fromStatus: "ready", toStatus: "completed" }],
    });

    const result = await service.completeItems("initiative-a", {
      mode: "holistic-loop",
      round: 3,
      runId: "run-3",
      itemRefs: ["execute/item-1"],
    });

    expect(client.completeItems).toHaveBeenCalledWith({
      initiativeName: "initiative-a",
      mode: "holistic-loop",
      round: 3,
      runId: "run-3",
      itemRefs: ["execute/item-1"],
      requestedBy: "swarm-manager-ui",
    });
    expect(result.completedItems[0]?.itemRef).toBe("execute/item-1");
  });

  it("applies backlog proposal mutations", async () => {
    client.applyBacklogSync.mockResolvedValue({
      initiativeName: "initiative-a",
      mode: "phased-plan-drain",
      phase: "classify_progress",
      round: 4,
      runId: "run-4",
      proposalResult: {
        applied: 1,
        failed: 0,
        skipped: 1,
        created: 1,
        updated: 0,
        outcomes: [{ mutationId: "m1", op: "add_item", target: "fix/follow-up", applied: true }],
      },
    });

    const result = await service.applyBacklogSync("initiative-a", {
      mode: "phased-plan-drain",
      round: 4,
      runId: "run-4",
      acceptedMutationIds: ["m1"],
    });

    expect(client.applyBacklogSync).toHaveBeenCalledWith({
      initiativeName: "initiative-a",
      mode: "phased-plan-drain",
      round: 4,
      runId: "run-4",
      acceptedMutationIds: ["m1"],
      requestedBy: "swarm-manager-ui",
    });
    expect(result.proposalResult?.applied).toBe(1);
    expect(result.proposalResult?.outcomes?.[0]?.mutationId).toBe("m1");
  });

  it("renders a live prompt, omitting a zero round", async () => {
    client.renderLivePrompt.mockResolvedValue({
      mode: "holistic-loop",
      phase: "execute",
      skillId: "swarm-manager/execute",
      profileKey: "swarm-manager/deep-work",
      variables: { INITIATIVE: "initiative-a" },
      prompt: "Do the work.",
    });

    const rendered = await service.renderLivePrompt("initiative-a", "execute");
    expect(client.renderLivePrompt).toHaveBeenCalledWith({
      initiativeName: "initiative-a",
      phase: "execute",
      round: 0,
      note: "",
    });
    expect(rendered.prompt).toBe("Do the work.");
    expect(rendered.variables.INITIATIVE).toBe("initiative-a");
  });

  it("normalizes phase_kind on workspace and catalog phases", async () => {
    client.getWorkspace.mockResolvedValue({
      initiativeName: "initiative-a",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        scopeKind: "initiative",
        runStrategy: "operator_gated_loop",
        capabilities: {},
        phases: [
          {
            phase: "investigate",
            phaseKind: "investigate",
            activityPurpose: "holistic_loop_investigate",
            profileKey: "swarm-manager/deep-work",
            writesRepo: false,
          },
          {
            phase: "execute",
            phaseKind: "execute",
            activityPurpose: "holistic_loop_execute",
            profileKey: "swarm-manager/deep-work",
            writesRepo: true,
            autoStartAfter: ["plan"],
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

    client.catalog.mockResolvedValue({
      modes: [{
        mode: "holistic-loop",
        label: "Holistic Loop",
        capabilities: {},
        supportsPhases: true,
        phases: [
          { phase: "investigate", phaseKind: "investigate" },
          { phase: "review", phaseKind: "review", autoStartAfter: ["execute"] },
          { phase: "weird", phaseKind: "fabricated-kind" },
        ],
      }],
    });
    const catalog = await service.catalog();
    expect(catalog.modes[0]?.phases[0]?.phaseKind).toBe("investigate");
    expect(catalog.modes[0]?.phases[1]?.phaseKind).toBe("review");
    expect(catalog.modes[0]?.phases[1]?.autoStartAfter).toEqual(["execute"]);
    // Unknown kinds collapse to empty rather than silently passing through a
    // malformed value to the lane-aware UI surfaces.
    expect(catalog.modes[0]?.phases[2]?.phaseKind).toBe("");
  });
});

describe("parseActiveItemExecutionsConflict", () => {
  function conflictError(): ConnectError {
    return new ConnectError(
      "initiative has active item-level executions",
      Code.FailedPrecondition,
      undefined,
      [{
        desc: OperatingModeActiveItemExecutionsConflictSchema,
        value: {
          initiativeName: "initiative-a",
          fromMode: "item-level",
          toMode: "holistic-loop",
          executions: [
            { itemRef: "fix:auth-cookie", executionId: "exec-1", runId: "run-aaaa-bbbb", status: "running" },
            { itemRef: "feat:onboarding", runId: "run-cccc-dddd", status: "running" },
          ],
        },
      }],
    );
  }

  it("decodes the structured detail from a FailedPrecondition Connect error", () => {
    const conflict = parseActiveItemExecutionsConflict(conflictError());
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

  it("returns null for non-ConnectError values", () => {
    expect(parseActiveItemExecutionsConflict(new Error("boom"))).toBeNull();
    expect(parseActiveItemExecutionsConflict("string error")).toBeNull();
    expect(parseActiveItemExecutionsConflict(null)).toBeNull();
  });

  it("returns null for a different Connect code", () => {
    const notFound = new ConnectError("not found", Code.NotFound);
    expect(parseActiveItemExecutionsConflict(notFound)).toBeNull();
  });

  it("returns null when the conflict detail is absent", () => {
    const bare = new ConnectError("lock held", Code.FailedPrecondition);
    expect(parseActiveItemExecutionsConflict(bare)).toBeNull();
  });
});
