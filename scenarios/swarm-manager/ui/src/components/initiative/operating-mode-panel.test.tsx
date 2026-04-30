import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { OperatingModePanel } from "./operating-mode-panel";
import { initiativeModeService, initiativeService } from "../../services";
import type { Initiative } from "../../types";

vi.mock("../../services", () => ({
  initiativeModeService: {
    workspace: vi.fn(),
    switchMode: vi.fn(),
    startPhase: vi.fn(),
    refreshRound: vi.fn(),
    cancelRound: vi.fn(),
  },
  initiativeService: {
    updateMetadata: vi.fn(),
  },
}));

describe("OperatingModePanel", () => {
  let queryClient: QueryClient;
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
    vi.mocked(initiativeModeService.workspace).mockResolvedValue({
      initiativeName: "mode-initiative",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        label: "Holistic Loop",
        scopeKind: "initiative",
        runStrategy: "operator_gated_loop",
        terminal: ["review"],
        transitions: { investigate: ["plan"] },
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
        status: "completed",
        payload: { agent_summary: "Investigation complete" },
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
    await waitFor(() => {
      expect(screen.getByTestId("initiative-mode-select")).toBeInTheDocument();
    });

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
    await waitFor(() => {
      expect(screen.getByTestId("initiative-mode-select")).toBeInTheDocument();
    });

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
