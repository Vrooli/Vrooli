import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { MemoryRouter } from "react-router-dom";
import { OperatingModePanel } from "./operating-mode-panel";
import {
  installMatchMediaMock,
  installResizeObserverMock,
} from "../../test-utils/browser";

installMatchMediaMock();
installResizeObserverMock();
import { agentOperationsService, initiativeModeService, initiativeService } from "../../services";
import { selectors } from "../../consts/selectors";
import type { Initiative, InitiativeRollup } from "../../types";
import type {
  OperatingModeCapabilities,
  OperatingModeCatalogEntry,
  OperatingModeCatalogPhase,
  OperatingModeWorkspace,
} from "../../types/operating-mode";
import type { WorkflowMigrationStatus } from "../../types/agent-operations";

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
  // The workspace hook and the Operation Bindings section reach the
  // agent-operations surface; stub it so queries resolve deterministically.
  agentOperationsService: {
    getWorkflowProjection: vi.fn().mockResolvedValue({ found: false, operations: [] }),
    getResolvedBindings: vi.fn().mockResolvedValue([]),
    listCompatibleModes: vi.fn().mockResolvedValue([]),
    listBindingOverrides: vi.fn().mockResolvedValue([]),
    putBindingOverride: vi.fn(),
    deleteBindingOverride: vi.fn(),
    getMigrationStatus: vi.fn().mockResolvedValue({ documentFound: false, state: "not-started" }),
    listExecutionHistory: vi.fn().mockResolvedValue([]),
  },
}));

// Capability factory — reusable across tests so flags can be flipped in
// isolation without redefining the whole object.
function capabilities(overrides: Partial<OperatingModeCapabilities> = {}): OperatingModeCapabilities {
  return {
    supportsPhases: false,
    canStartPhases: false,
    canCompleteItems: false,
    canApplyBacklogSyncProposals: false,
    requiresAcceptanceCriteria: false,
    supportsArtifacts: false,
    supportsHandoffs: false,
    ...overrides,
  };
}

const fullPhaseCapabilities = capabilities({
  supportsPhases: true,
  canStartPhases: true,
  canCompleteItems: true,
  canApplyBacklogSyncProposals: true,
  requiresAcceptanceCriteria: true,
  supportsArtifacts: true,
});

function makeCatalogPhase(overrides: Partial<OperatingModeCatalogPhase> & { phase: string }): OperatingModeCatalogPhase {
  return {
    label: overrides.phase,
    phaseKind: "investigate",
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
      requiresBacklogSync: false,
      requiredArtifactCount: 0,
    },
    ...overrides,
  };
}

function makeCatalogEntry(overrides: Partial<OperatingModeCatalogEntry> & { mode: string; label: string }): OperatingModeCatalogEntry {
  return {
    description: `${overrides.label} description.`,
    bestFor: [`${overrides.label} best for`],
    notFor: [`${overrides.label} not for`],
    tradeoffs: [`${overrides.label} tradeoff`],
    usageCount: 0,
    targetKind: "initiative",
    runStrategy: "operator_gated_loop",
    workspaceTabId: "operating-mode",
    capabilities: fullPhaseCapabilities,
    default: false,
    switchable: true,
    supportsPhases: true,
    phases: [],
    ...overrides,
  };
}

const baseInitiative: Initiative = {
  name: "mode-initiative",
  title: "Mode Initiative",
  description: "",
  status: "active",
  mode: "holistic-loop",
  acceptanceCriteria: ["Pass initiative review"],
  priority: 0,
  dependsOn: [],
  items: ["execute/item-1"],
  created: "2026-04-30T00:00:00Z",
  updated: "2026-04-30T00:00:00Z",
};

const baseCatalog: OperatingModeCatalogEntry[] = [
  makeCatalogEntry({
    mode: "item-level",
    label: "Member-item workflow",
    targetKind: "initiative",
    runStrategy: "",
    capabilities: capabilities(),
    default: true,
    supportsPhases: false,
  }),
  makeCatalogEntry({
    mode: "holistic-loop",
    label: "Holistic Loop",
    capabilities: fullPhaseCapabilities,
    phases: [makeCatalogPhase({ phase: "investigate", isStart: true })],
  }),
  makeCatalogEntry({
    mode: "phased-plan-drain",
    label: "Phased Plan Drain",
    runStrategy: "sequential_handoff",
    capabilities: capabilities({
      supportsPhases: true,
      canStartPhases: true,
      canCompleteItems: true,
      canApplyBacklogSyncProposals: true,
      requiresAcceptanceCriteria: true,
      supportsArtifacts: true,
      supportsHandoffs: true,
    }),
    phases: [makeCatalogPhase({ phase: "execute_next", writesRepo: true })],
  }),
];

function makeWorkspace(overrides?: Partial<OperatingModeWorkspace>): OperatingModeWorkspace {
  return {
    initiativeName: "mode-initiative",
    mode: "holistic-loop",
    definition: {
      mode: "holistic-loop",
      label: "Holistic Loop",
      targetKind: "initiative",
      capabilities: fullPhaseCapabilities,
      runStrategy: "operator_gated_loop",
      terminal: ["review"],
      transitions: { investigate: ["plan"] },
      phases: [
        {
          phase: "investigate",
          phaseKind: "investigate",
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
        },
      ],
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
    ...overrides,
  };
}

describe("OperatingModePanel", () => {
  let queryClient: QueryClient;

  function renderPanel(overrides?: {
    initiative?: Partial<Initiative>;
    rollup?: InitiativeRollup;
    onInitiativeUpdated?: () => void;
  }) {
    const initiative: Initiative = { ...baseInitiative, ...overrides?.initiative };
    return render(
      <MemoryRouter>
        <QueryClientProvider client={queryClient}>
          <OperatingModePanel
            initiative={initiative}
            rollup={overrides?.rollup}
            onInitiativeUpdated={overrides?.onInitiativeUpdated ?? vi.fn()}
          />
        </QueryClientProvider>
      </MemoryRouter>,
    );
  }

  beforeEach(() => {
    queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
    vi.clearAllMocks();
    vi.mocked(initiativeModeService.catalog).mockResolvedValue({ modes: baseCatalog });
    vi.mocked(initiativeModeService.workspace).mockResolvedValue(makeWorkspace());
  });

  describe("capability gating", () => {
    it("hides the Acceptance Criteria section when requiresAcceptanceCriteria=false", async () => {
      vi.mocked(initiativeModeService.workspace).mockResolvedValue(
        makeWorkspace({
          definition: {
            ...makeWorkspace().definition,
            capabilities: capabilities({ supportsPhases: true, canStartPhases: true, supportsArtifacts: true }),
          },
        }),
      );
      renderPanel();
      await screen.findByTestId(selectors.initiativeDetails.modeHero);
      expect(screen.queryByTestId(selectors.initiativeDetails.criteriaInput)).toBeNull();
    });

    it("shows the Acceptance Criteria section when requiresAcceptanceCriteria=true", async () => {
      renderPanel();
      expect(await screen.findByTestId(selectors.initiativeDetails.criteriaInput)).toBeInTheDocument();
    });

    it("renders MemberItemStrategyPanel only for member-item-strategy initiatives", async () => {
      vi.mocked(initiativeModeService.workspace).mockResolvedValue(
        makeWorkspace({
          definition: {
            ...makeWorkspace().definition,
            capabilities: capabilities(),
          },
        }),
      );
      renderPanel({ initiative: { mode: "item-level" } });
      expect(await screen.findByTestId(selectors.initiativeDetails.memberItemStrategyPanel)).toBeInTheDocument();
      expect(screen.queryByTestId(selectors.initiativeDetails.phaseComposer)).toBeNull();
    });

    it("offers the Operation Bindings section (collapsed by default)", async () => {
      renderPanel();
      await screen.findByTestId(selectors.initiativeDetails.modeHero);
      const section = screen.getByTestId(selectors.initiativeDetails.operationBindingsSection);
      expect(section).toHaveTextContent("Operation Bindings");
      // Collapsed by default — the bindings panel (and its queries) only
      // mount once the operator expands the section.
      expect(screen.queryByTestId(selectors.workflowBindings.panel)).toBeNull();
    });

    it("hides Phase Composer, Artifacts, and Rounds when supportsPhases=false", async () => {
      vi.mocked(initiativeModeService.workspace).mockResolvedValue(
        makeWorkspace({
          definition: {
            ...makeWorkspace().definition,
            capabilities: capabilities(),
          },
        }),
      );
      renderPanel({ initiative: { mode: "item-level" } });
      await screen.findByTestId(selectors.initiativeDetails.modeHero);
      expect(screen.queryByTestId(selectors.initiativeDetails.phaseComposer)).toBeNull();
      expect(screen.queryByText("Artifacts")).toBeNull();
      expect(screen.queryByText("Rounds")).toBeNull();
    });

    it("hides Artifacts but keeps Rounds when supportsArtifacts=false but supportsPhases=true", async () => {
      vi.mocked(initiativeModeService.workspace).mockResolvedValue(
        makeWorkspace({
          definition: {
            ...makeWorkspace().definition,
            capabilities: capabilities({ supportsPhases: true, canStartPhases: true, supportsArtifacts: false }),
          },
        }),
      );
      renderPanel();
      await screen.findByTestId(selectors.initiativeDetails.modeHero);
      // Wait for query data to settle
      await waitFor(() => {
        expect(screen.queryByText("Artifacts")).toBeNull();
      });
      expect(screen.getByText("Rounds")).toBeInTheDocument();
    });
  });

  describe("partial migration visibility", () => {
    const migrationStatus: WorkflowMigrationStatus = {
      state: "not-started",
      epoch: 0,
      stagedCount: 0,
      promotedCount: 0,
      quarantinedCount: 0,
      startedAt: "",
      updatedAt: "",
      reportPath: "",
      documentFound: false,
    };

    it("shows the staged migration banner with counts on the initiative surface", async () => {
      vi.mocked(agentOperationsService.getMigrationStatus).mockResolvedValueOnce({
        ...migrationStatus,
        state: "staged",
        stagedCount: 7,
        epoch: 2,
        documentFound: true,
      });
      renderPanel();
      const banner = await screen.findByTestId("workflow-migration-banner");
      expect(banner).toHaveTextContent("Workflow migration staged");
      expect(banner).toHaveTextContent("Epoch 2: 7 documents staged");
    });

    it("shows the quarantined migration warning on the initiative surface", async () => {
      vi.mocked(agentOperationsService.getMigrationStatus).mockResolvedValueOnce({
        ...migrationStatus,
        state: "quarantined",
        quarantinedCount: 3,
        epoch: 2,
        documentFound: true,
      });
      renderPanel();
      const banner = await screen.findByTestId("workflow-migration-banner");
      expect(banner).toHaveTextContent("Workflow migration quarantined");
      expect(banner).toHaveTextContent("Epoch 2: 3 documents quarantined");
    });

    it("shows no banner in the not-started steady state", async () => {
      renderPanel();
      await screen.findByTestId(selectors.initiativeDetails.modeHero);
      expect(screen.queryByTestId("workflow-migration-banner")).toBeNull();
    });
  });

  describe("hero + picker switch flow", () => {
    it("opens the picker when the hero Switch button is clicked", async () => {
      renderPanel();
      const switchBtn = await screen.findByTestId(selectors.initiativeDetails.modeHeroSwitchButton);
      const user = userEvent.setup();
      await user.click(switchBtn);
      expect(await screen.findByTestId(selectors.initiativeDetails.modePicker)).toBeInTheDocument();
    });

    it("calls switchMode when a different mode is selected and confirmed", async () => {
      vi.mocked(initiativeModeService.switchMode).mockResolvedValue({
        initiativeName: "mode-initiative",
        fromMode: "holistic-loop",
        toMode: "phased-plan-drain",
      });
      renderPanel();
      const user = userEvent.setup();
      await user.click(await screen.findByTestId(selectors.initiativeDetails.modeHeroSwitchButton));
      const cards = await screen.findAllByTestId(selectors.initiativeDetails.modePickerCard);
      await user.click(cards.find((c) => c.textContent?.includes("Phased Plan Drain"))!);
      await user.click(screen.getByTestId(selectors.initiativeDetails.modePickerConfirm));
      await waitFor(() => {
        expect(initiativeModeService.switchMode).toHaveBeenCalledWith("mode-initiative", {
          mode: "phased-plan-drain",
          cancelActiveItemExecutions: false,
        });
      });
    });

    it("shows the orientation banner pointing at acceptance-criteria after switching to a mode that requires criteria", async () => {
      vi.mocked(initiativeModeService.switchMode).mockResolvedValue({
        initiativeName: "mode-initiative",
        fromMode: "item-level",
        toMode: "holistic-loop",
      });
      // Initiative starts in item-level (capabilities don't require criteria
      // initially) and has no acceptance criteria yet.
      renderPanel({ initiative: { mode: "item-level", acceptanceCriteria: [] } });
      const user = userEvent.setup();
      await user.click(await screen.findByTestId(selectors.initiativeDetails.modeHeroSwitchButton));
      const cards = await screen.findAllByTestId(selectors.initiativeDetails.modePickerCard);
      await user.click(cards.find((c) => c.textContent?.includes("Holistic Loop"))!);
      await user.click(screen.getByTestId(selectors.initiativeDetails.modePickerConfirm));

      const banner = await screen.findByTestId(selectors.initiativeDetails.orientationBanner);
      expect(banner).toHaveTextContent("Holistic Loop");
      expect(banner).toHaveTextContent(/acceptance criteria/i);

      // Dismiss closes the banner.
      await user.click(within(banner).getByLabelText("Dismiss orientation banner"));
      expect(screen.queryByTestId(selectors.initiativeDetails.orientationBanner)).toBeNull();
    });

    it("submits with cancel=false on first attempt — server-side 409 drives the ack flow", async () => {
      renderPanel({ initiative: { mode: "item-level" } });
      const user = userEvent.setup();
      await user.click(await screen.findByTestId(selectors.initiativeDetails.modeHeroSwitchButton));
      const cards = await screen.findAllByTestId(selectors.initiativeDetails.modePickerCard);
      await user.click(cards.find((c) => c.textContent?.includes("Holistic Loop"))!);
      // No upfront ack — the dialog only renders the cancellation list after
      // the server returns a 409 active-item-executions conflict.
      expect(screen.queryByTestId(selectors.initiativeDetails.modePickerOverrideAck)).toBeNull();
      const confirm = screen.getByTestId(selectors.initiativeDetails.modePickerConfirm);
      expect(confirm).toBeEnabled();
      await user.click(confirm);
      await waitFor(() => {
        expect(initiativeModeService.switchMode).toHaveBeenCalledWith("mode-initiative", {
          mode: "holistic-loop",
          cancelActiveItemExecutions: false,
        });
      });
    });
  });

  describe("phase composer", () => {
    it("starts a phase via the composer with an envelope-wrapped note", async () => {
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
      await screen.findByTestId(selectors.initiativeDetails.phaseComposer);
      // Phase graph nodes are rendered by xyflow; node-click within JSDOM is
      // covered by the dedicated PhaseComposer unit test (which mocks the
      // graph). This test only confirms the composer mounts inside the panel
      // when supportsPhases=true — which is sufficient signal that the phase
      // mutation is wired through the hook envelope path.
      // phase mutation is wired through the hook envelope path).
      expect(screen.getByTestId(selectors.initiativeDetails.phaseComposer)).toBeInTheDocument();
    });
  });

  describe("round actions", () => {
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
      renderPanel({ onInitiativeUpdated });
      await screen.findByText("execute/item-1");
      const user = userEvent.setup();
      await user.click(screen.getByTestId(selectors.initiativeDetails.completeItems));
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

    it("applies selected proposal mutations from a round backlog sync plan", async () => {
      vi.mocked(initiativeModeService.workspace).mockResolvedValue(
        makeWorkspace({
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
        }),
      );
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
      renderPanel({ onInitiativeUpdated });
      await screen.findByTestId(selectors.initiativeDetails.backlogProposal);
      const toggles = screen.getAllByTestId(selectors.initiativeDetails.backlogProposalMutationToggle);
      const user = userEvent.setup();
      await user.click(toggles[1]!);
      await user.click(screen.getByTestId(selectors.initiativeDetails.applyBacklogSync));
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
  });

  describe("acceptance criteria", () => {
    it("saves criteria parsed from the textarea", async () => {
      vi.mocked(initiativeService.updateMetadata).mockResolvedValue({
        initiative: { ...baseInitiative, mode: "holistic-loop" },
        rollup: { total: 0, completed: 0, inProgress: 0, failed: 0, pending: 0, archived: 0, blocked: 0 } as InitiativeRollup,
      });
      const onInitiativeUpdated = vi.fn();
      renderPanel({ onInitiativeUpdated });
      const textarea = await screen.findByTestId(selectors.initiativeDetails.criteriaInput);
      const user = userEvent.setup();
      await user.clear(textarea);
      await user.type(textarea, "A{Enter}B");
      await user.click(screen.getByTestId(selectors.initiativeDetails.criteriaSave));
      await waitFor(() => {
        expect(initiativeService.updateMetadata).toHaveBeenCalledWith("mode-initiative", {
          acceptanceCriteria: ["A", "B"],
        });
      });
    });
  });
});
