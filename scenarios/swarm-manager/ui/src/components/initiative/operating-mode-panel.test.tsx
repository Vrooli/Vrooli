import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { OperatingModePanel } from "./operating-mode-panel";
import { initiativeModeService, initiativeService } from "../../services";
import type { Initiative } from "../../types";
import type {
  OperatingModeCapabilities,
  OperatingModeCatalogPhase,
} from "../../types/operating-mode";

function catalogPhase(overrides: Partial<OperatingModeCatalogPhase> & { phase: string }): OperatingModeCatalogPhase {
  return {
    title: overrides.phase,
    purpose: "",
    trigger: "",
    profileKey: "swarm-manager/deep-work",
    writesRepo: false,
    catalogId: "",
    skillId: "",
    activityPurpose: "",
    lockPurpose: "",
    outputContract: {
      requiresStructuredResult: true,
      requiresProgress: false,
      requiresVerdict: false,
      requiresHandoff: false,
      requiredArtifactCount: 0,
    },
    ...overrides,
  };
}

vi.mock("../../services", () => ({
  initiativeModeService: {
    catalog: vi.fn(),
    workspace: vi.fn(),
    switchMode: vi.fn(),
    startPhase: vi.fn(),
    refreshRound: vi.fn(),
    cancelRound: vi.fn(),
    completeItems: vi.fn(),
    applyBacklogSync: vi.fn(),
  },
  initiativeService: {
    updateMetadata: vi.fn(),
  },
}));

describe("OperatingModePanel", () => {
  let queryClient: QueryClient;
  const itemExecutionCapabilities: OperatingModeCapabilities = {
    supportsPhases: false,
    canStartPhases: false,
    canCompleteItems: false,
    canApplyBacklogSyncProposals: false,
    requiresAcceptanceCriteria: false,
    supportsArtifacts: false,
    supportsHandoffs: false,
    usesItemExecutionFlow: true,
  };
  const initiativeModeCapabilities: OperatingModeCapabilities = {
    supportsPhases: true,
    canStartPhases: true,
    canCompleteItems: true,
    canApplyBacklogSyncProposals: true,
    requiresAcceptanceCriteria: true,
    supportsArtifacts: true,
    supportsHandoffs: false,
    usesItemExecutionFlow: false,
  };
  const initiative: Initiative = {
    name: "mode-initiative",
    title: "Mode Initiative",
    description: "",
    status: "active",
    mode: "holistic-loop",
    acceptanceCriteria: ["Pass initiative review"],
    priority: 0,
    dependsOn: [],
    items: [],
    created: "2026-04-30T00:00:00Z",
    updated: "2026-04-30T00:00:00Z",
  };

  beforeEach(() => {
    queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    vi.clearAllMocks();
    vi.mocked(initiativeModeService.catalog).mockResolvedValue({
      modes: [
        {
          mode: "item-level",
          label: "Item Level",
          scopeKind: "backlog_item",
          runStrategy: "existing_item_flow",
          workspaceTabId: "info",
          capabilities: itemExecutionCapabilities,
          default: true,
          switchable: true,
          usageCount: 0,
          supportsPhases: false,
          phases: [],
        },
        {
          mode: "holistic-loop",
          label: "Holistic Loop",
          scopeKind: "initiative",
          runStrategy: "operator_gated_loop",
          workspaceTabId: "operating-mode",
          capabilities: initiativeModeCapabilities,
          default: false,
          switchable: true,
          usageCount: 0,
          supportsPhases: true,
          phases: [catalogPhase({ phase: "investigate", profileKey: "swarm-manager/deep-work", writesRepo: false })],
        },
        {
          mode: "phased-plan-drain",
          label: "Phased Plan Drain",
          scopeKind: "initiative",
          runStrategy: "sequential_handoff",
          workspaceTabId: "operating-mode",
          capabilities: { ...initiativeModeCapabilities, supportsHandoffs: true },
          default: false,
          switchable: true,
          usageCount: 0,
          supportsPhases: true,
          phases: [catalogPhase({ phase: "execute_next", profileKey: "swarm-manager/deep-work", writesRepo: true })],
        },
      ],
    });
    vi.mocked(initiativeModeService.workspace).mockResolvedValue({
      initiativeName: "mode-initiative",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        label: "Holistic Loop",
        scopeKind: "initiative",
        capabilities: initiativeModeCapabilities,
        runStrategy: "operator_gated_loop",
        terminal: ["review"],
        transitions: { investigate: ["plan"] },
        phases: [{
          phase: "investigate",
          activityPurpose: "holistic_loop_investigate",
          profileKey: "swarm-manager/deep-work",
          writesRepo: false,
          startable: true,
          next: true,
          outputArtifacts: [{
            path: "modes/holistic-loop/findings.md",
            contentType: "text/markdown",
            required: true,
          }],
        }],
      },
      artifacts: [{
        path: "modes/holistic-loop/findings.md",
        contentType: "text/markdown",
        required: true,
        content: "# Findings",
      }],
      rounds: [{
        round: 1,
        mode: "holistic-loop",
        scopeKind: "initiative",
        scopeId: "mode-initiative",
        phase: "investigate",
        runStrategy: "operator_gated_loop",
        agentProfileKey: "swarm-manager/deep-work",
        generatedAt: "2026-04-30T00:00:00Z",
        runId: "run-1",
        status: "completed",
        payload: {
          agent_summary: "Investigation complete",
          backlog_sync_plan: { completed_items: ["execute/item-1"] },
        },
      }],
    });
  });

  function renderPanel(onInitiativeUpdated = vi.fn()) {
    return render(
      <QueryClientProvider client={queryClient}>
        <OperatingModePanel initiative={initiative} onInitiativeUpdated={onInitiativeUpdated} />
      </QueryClientProvider>,
    );
  }

  it("renders workspace artifacts, rounds, and starts a phase", async () => {
    vi.mocked(initiativeModeService.startPhase).mockResolvedValue({
      round: 2,
      mode: "holistic-loop",
      scopeKind: "initiative",
      scopeId: "mode-initiative",
      phase: "investigate",
      runStrategy: "operator_gated_loop",
      agentProfileKey: "swarm-manager/deep-work",
      generatedAt: "2026-04-30T01:00:00Z",
      status: "agent_running",
    });

    renderPanel();

    await waitFor(() => {
      expect(screen.getAllByText("Holistic Loop").length).toBeGreaterThan(0);
    });
    expect(await screen.findByText("modes/holistic-loop/findings.md")).toBeInTheDocument();
    expect(screen.getByText("# Findings")).toBeInTheDocument();
    expect(screen.getByText("Investigation complete")).toBeInTheDocument();

    const user = userEvent.setup();
    await user.click(screen.getByRole("button", { name: /investigate/i }));

    expect(initiativeModeService.startPhase).toHaveBeenCalledWith(
      "mode-initiative",
      "investigate",
      { note: "" },
    );
  });

  it("completes items from a round backlog sync plan", async () => {
    vi.mocked(initiativeModeService.completeItems).mockResolvedValue({
      initiativeName: "mode-initiative",
      mode: "holistic-loop",
      phase: "investigate",
      round: 1,
      runId: "run-1",
      completedItems: [{
        itemRef: "execute/item-1",
        fromStatus: "ready",
        toStatus: "completed",
      }],
    });
    const onInitiativeUpdated = vi.fn();

    renderPanel(onInitiativeUpdated);
    await screen.findByText("execute/item-1");

    const user = userEvent.setup();
    await user.click(screen.getByTestId("initiative-mode-complete-items"));

    await waitFor(() => {
      expect(initiativeModeService.completeItems).toHaveBeenCalledWith("mode-initiative", {
        mode: "holistic-loop",
        round: 1,
        runId: "run-1",
        itemRefs: ["execute/item-1"],
      });
    });
    expect(onInitiativeUpdated).toHaveBeenCalled();
  });

  it("disables sync actions and explains when a completed round is missing its run ID", async () => {
    vi.mocked(initiativeModeService.workspace).mockResolvedValue({
      initiativeName: "mode-initiative",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        label: "Holistic Loop",
        scopeKind: "initiative",
        capabilities: initiativeModeCapabilities,
        runStrategy: "operator_gated_loop",
        terminal: ["review"],
        transitions: {},
        phases: [],
      },
      artifacts: [],
      rounds: [{
        round: 2,
        mode: "holistic-loop",
        scopeKind: "initiative",
        scopeId: "mode-initiative",
        phase: "execute",
        runStrategy: "operator_gated_loop",
        agentProfileKey: "swarm-manager/deep-work",
        generatedAt: "2026-04-30T00:00:00Z",
        status: "completed",
        payload: {
          backlog_sync_plan: { completed_items: ["execute/item-1"] },
        },
      }],
    });

    renderPanel();

    expect(await screen.findByText(/missing an AgentManager run ID/i)).toBeInTheDocument();
    expect(screen.queryByTestId("initiative-mode-complete-items")).not.toBeInTheDocument();
  });

  it("applies selected proposal mutations from a round backlog sync plan", async () => {
    vi.mocked(initiativeModeService.workspace).mockResolvedValue({
      initiativeName: "mode-initiative",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        label: "Holistic Loop",
        scopeKind: "initiative",
        capabilities: initiativeModeCapabilities,
        runStrategy: "operator_gated_loop",
        terminal: ["review"],
        transitions: {},
        phases: [],
      },
      artifacts: [],
      rounds: [{
        round: 2,
        mode: "holistic-loop",
        scopeKind: "initiative",
        scopeId: "mode-initiative",
        phase: "execute",
        runStrategy: "operator_gated_loop",
        agentProfileKey: "swarm-manager/deep-work",
        generatedAt: "2026-04-30T00:00:00Z",
        runId: "run-2",
        status: "completed",
        payload: {
          backlog_sync_plan: {
            proposal: {
              form: "mutation_list",
              rationale: "Follow-up cleanup",
              mutations: [
                { id: "m1", op: "add_item", rationale: "Add follow-up", item: { kind: "fix", name: "follow-up", title: "Follow up" } },
                { id: "m2", op: "change_status", target: "execute/item-1", status: "blocked" },
              ],
            },
          },
        },
      }],
    });
    vi.mocked(initiativeModeService.applyBacklogSync).mockResolvedValue({
      initiativeName: "mode-initiative",
      mode: "holistic-loop",
      phase: "execute",
      round: 2,
      runId: "run-2",
      completedItems: [],
      proposalResult: { applied: 1, failed: 0, skipped: 1 },
    });
    const onInitiativeUpdated = vi.fn();

    renderPanel(onInitiativeUpdated);
    await screen.findByTestId("initiative-mode-backlog-proposal");

    const toggles = screen.getAllByTestId("initiative-mode-backlog-proposal-mutation-toggle");
    const user = userEvent.setup();
    await user.click(toggles[1]!);
    await user.click(screen.getByTestId("initiative-mode-apply-backlog-sync"));

    await waitFor(() => {
      expect(initiativeModeService.applyBacklogSync).toHaveBeenCalledWith("mode-initiative", {
        mode: "holistic-loop",
        round: 2,
        runId: "run-2",
        acceptedMutationIds: ["m1"],
      });
    });
    expect(onInitiativeUpdated).toHaveBeenCalled();
  });

  it("saves mode and acceptance criteria through initiative metadata", async () => {
    const onInitiativeUpdated = vi.fn();
    vi.mocked(initiativeModeService.switchMode).mockResolvedValue({
      initiativeName: "mode-initiative",
      fromMode: "holistic-loop",
      toMode: "phased-plan-drain",
    });
    vi.mocked(initiativeService.updateMetadata).mockResolvedValue({
      initiative: { ...initiative, mode: "phased-plan-drain" },
      rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0 },
    });

    renderPanel(onInitiativeUpdated);
    await screen.findByRole("option", { name: "Phased Plan Drain" });

    const user = userEvent.setup();
    await user.selectOptions(screen.getByTestId("initiative-mode-select"), "phased-plan-drain");
    await user.click(screen.getByTestId("initiative-mode-save"));
    await user.clear(screen.getByTestId("initiative-mode-criteria"));
    await user.type(screen.getByTestId("initiative-mode-criteria"), "Criterion A\nCriterion B");
    await user.click(screen.getByTestId("initiative-mode-criteria-save"));

    await waitFor(() => {
      expect(initiativeModeService.switchMode).toHaveBeenCalledWith("mode-initiative", {
        mode: "phased-plan-drain",
        cancelActiveItemExecutions: false,
      });
    });
    expect(initiativeService.updateMetadata).toHaveBeenCalledWith("mode-initiative", {
      acceptanceCriteria: ["Criterion A", "Criterion B"],
    });
    expect(onInitiativeUpdated).toHaveBeenCalled();
  });

  it("renders selectable modes from the backend catalog", async () => {
    vi.mocked(initiativeModeService.catalog).mockResolvedValue({
      modes: [
        {
          mode: "item-level",
          label: "Item Level",
          scopeKind: "backlog_item",
          runStrategy: "existing_item_flow",
          workspaceTabId: "info",
          capabilities: itemExecutionCapabilities,
          default: true,
          switchable: true,
          usageCount: 0,
          supportsPhases: false,
          phases: [],
        },
        {
          mode: "custom-audit-loop",
          label: "Custom Audit Loop",
          scopeKind: "initiative",
          runStrategy: "operator_gated_loop",
          workspaceTabId: "operating-mode",
          capabilities: initiativeModeCapabilities,
          default: false,
          switchable: true,
          usageCount: 0,
          supportsPhases: true,
          phases: [],
        },
      ],
    });
    vi.mocked(initiativeModeService.switchMode).mockResolvedValue({
      initiativeName: "mode-initiative",
      fromMode: "holistic-loop",
      toMode: "custom-audit-loop",
    });

    renderPanel();
    const select = await screen.findByTestId("initiative-mode-select");

    expect(await screen.findByRole("option", { name: "Custom Audit Loop" })).toBeInTheDocument();

    const user = userEvent.setup();
    await user.selectOptions(select, "custom-audit-loop");
    await user.click(screen.getByTestId("initiative-mode-save"));

    await waitFor(() => {
      expect(initiativeModeService.switchMode).toHaveBeenCalledWith("mode-initiative", {
        mode: "custom-audit-loop",
        cancelActiveItemExecutions: false,
      });
    });
  });

  it("disables mode switching when the backend catalog fails", async () => {
    vi.mocked(initiativeModeService.catalog).mockRejectedValue(new Error("catalog unavailable"));

    renderPanel();

    const save = await screen.findByTestId("initiative-mode-save");
    expect(await screen.findByText("catalog unavailable")).toBeInTheDocument();
    expect(save).toBeDisabled();
  });

  it("requires a second click when switching away from item-level mode", async () => {
    vi.mocked(initiativeModeService.switchMode).mockResolvedValue({
      initiativeName: "mode-initiative",
      fromMode: "item-level",
      toMode: "holistic-loop",
    });

    render(
      <QueryClientProvider client={queryClient}>
        <OperatingModePanel
          initiative={{ ...initiative, mode: "item-level" }}
          onInitiativeUpdated={vi.fn()}
        />
      </QueryClientProvider>,
    );
    await screen.findByRole("option", { name: "Holistic Loop" });

    const user = userEvent.setup();
    await user.selectOptions(screen.getByTestId("initiative-mode-select"), "holistic-loop");
    await user.click(screen.getByTestId("initiative-mode-save"));

    expect(initiativeModeService.switchMode).not.toHaveBeenCalled();
    expect(screen.getByText(/can cancel active member item executions/i)).toBeInTheDocument();

    await user.click(screen.getByTestId("initiative-mode-save"));

    await waitFor(() => {
      expect(initiativeModeService.switchMode).toHaveBeenCalledWith("mode-initiative", {
        mode: "holistic-loop",
        cancelActiveItemExecutions: true,
      });
    });
  });
});
