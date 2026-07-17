import { fireEvent, render, screen, waitFor, within } from "@testing-library/react";
import { MemoryRouter, Routes, Route } from "react-router-dom";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { describe, expect, it, vi, beforeEach, beforeAll } from "vitest";
import { OperatingModeDetailsPage } from "./OperatingModeDetailsPage";
import {
  installMatchMediaMock,
  installResizeObserverMock,
} from "../test-utils/browser";
import type {
  OperatingModeCatalogPhase,
  OperatingModeDetail,
} from "../types/operating-mode";

const getModeMock = vi.fn();
const updateModeMock = vi.fn();
const catalogMock = vi.fn();
const simulateModeMock = vi.fn();
const workspaceMock = vi.fn();
const refreshRoundMock = vi.fn();
const renderSimulationPromptMock = vi.fn();
const renderLivePromptMock = vi.fn();
const getSkillMock = vi.fn();
const navigateMock = vi.fn();

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => navigateMock };
});

vi.mock("../services", () => ({
  initiativeModeService: {
    catalog: () => catalogMock(),
    getMode: (mode: string) => getModeMock(mode),
    updateMode: (mode: string, args: unknown) => updateModeMock(mode, args),
    simulateMode: (mode: string, preset?: string) => simulateModeMock(mode, preset),
    workspace: (name: string) => workspaceMock(name),
    refreshRound: (name: string, mode: string, round: number) => refreshRoundMock(name, mode, round),
    renderSimulationPrompt: (mode: string, preset: string, step: number) =>
      renderSimulationPromptMock(mode, preset, step),
    renderLivePrompt: (name: string, phase: string, round?: number) =>
      renderLivePromptMock(name, phase, round),
  },
}));

vi.mock("../services/prompt-service", async (importOriginal) => {
  const actual = await importOriginal<Record<string, unknown>>();
  return { ...actual, promptService: { getSkill: (id: string) => getSkillMock(id) } };
});

vi.mock("../app/shell/AppShellContext", () => ({
  useAppShell: () => ({ openSidebar: () => {}, closeSidebar: () => {} }),
}));

beforeAll(() => {
  installResizeObserverMock();
  installMatchMediaMock();
  // jsdom doesn't implement smooth scrolling APIs used by the page helpers.
  window.scrollTo = vi.fn();
});

function renderPage(mode = "holistic-loop", search = "") {
  const queryClient = new QueryClient({ defaultOptions: { queries: { retry: false } } });
  const initial = `/operating-modes/${mode}${search}`;
  return render(
    <QueryClientProvider client={queryClient}>
      <MemoryRouter initialEntries={[initial]}>
        <Routes>
          <Route path="/operating-modes/:mode" element={<OperatingModeDetailsPage />} />
        </Routes>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

function makePhase(overrides: Partial<OperatingModeCatalogPhase> & { phase: string }): OperatingModeCatalogPhase {
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
    reads: { base: ["OPERATOR_NOTE"], target: ["MEMBER_ITEMS_JSON", "ACCEPTANCE_CRITERIA"] },
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

const SAMPLE_DETAIL: OperatingModeDetail = {
  entry: {
    mode: "holistic-loop",
    label: "Holistic Loop",
    description: "Investigate→plan→execute cycles",
    bestFor: ["Coupled work"],
    notFor: ["Independent items"],
    tradeoffs: ["One plan, not N"],
    whenInDoubtPickInstead: "item-level",
    usageCount: 2,
    targetKind: "initiative",
    runStrategy: "operator_gated_loop",
    workspaceTabId: "operating-mode",
    capabilities: {
      supportsPhases: true,
      canStartPhases: true,
      canCompleteItems: false,
      canApplyBacklogSyncProposals: false,
      requiresAcceptanceCriteria: false,
      supportsArtifacts: true,
      supportsHandoffs: false,
    },
    default: false,
    switchable: true,
    supportsPhases: true,
    phases: [
      makePhase({
        phase: "investigate",
        label: "Investigate",
        title: "Holistic Loop Investigate",
        purpose: "Investigate initiative-wide code, backlog, and system state.",
        trigger: "Operator starts holistic-loop investigate phase",
        profileKey: "swarm-manager/deep-work",
        catalogId: "swarm-manager-holistic-loop.investigate",
        skillId: "swarm-manager-holistic-loop-investigate",
        activityPurpose: "holistic_loop_investigate",
        lockPurpose: "holistic_loop_investigate",
        isStart: true,
        outputArtifacts: [
          { path: "modes/holistic-loop/findings.md", contentType: "text/markdown", required: true },
        ],
      }),
      makePhase({
        phase: "execute",
        label: "Execute",
        title: "Holistic Loop Execute",
        purpose: "Execute and report whether replanning is needed.",
        profileKey: "swarm-manager/deep-work",
        catalogId: "swarm-manager-holistic-loop.execute",
        skillId: "swarm-manager-holistic-loop-execute",
        activityPurpose: "holistic_loop_execute",
        lockPurpose: "holistic_loop_execute",
        writesRepo: true,
        samplesReplanRate: true,
      }),
      makePhase({
        phase: "review",
        label: "Review",
        title: "Holistic Loop Acceptance Review",
        purpose: "Evaluate against acceptance criteria.",
        profileKey: "swarm-manager/analysis",
        catalogId: "swarm-manager-holistic-loop.review",
        skillId: "swarm-manager-holistic-loop-review",
        activityPurpose: "holistic_loop_review",
        lockPurpose: "holistic_loop_review",
        isTerminal: true,
        requiresCriteria: true,
        samplesAcceptanceRate: true,
        outputContract: {
          requiresStructuredResult: true,
          requiresProgress: false,
          requiresVerdict: true,
          requiresHandoff: false,
          requiresBacklogSync: false,
          requiredArtifactCount: 0,
        },
      }),
    ],
    phaseGraph: {
      startPhase: "investigate",
      terminal: ["review"],
      transitions: [
        { from: "execute", to: "investigate", conditionKind: "eq", label: "on replan_needed = true", field: "replan_needed", value: "true" },
        { from: "execute", to: "review", conditionKind: "always", label: "always" },
        { from: "investigate", to: "execute", conditionKind: "always", label: "always" },
      ],
      acceptedVerdicts: ["accepted"],
    },
  },
  linkedInitiatives: [
    { name: "init-a", title: "Initiative A", status: "active", updated: "2026-04-30" },
    { name: "init-b", title: "Initiative B", status: "active", updated: "2026-04-29" },
  ],
};

describe("OperatingModeDetailsPage", () => {
  beforeEach(() => {
    getModeMock.mockReset();
    updateModeMock.mockReset();
    catalogMock.mockReset();
    simulateModeMock.mockReset();
    workspaceMock.mockReset();
    refreshRoundMock.mockReset();
    renderSimulationPromptMock.mockReset();
    renderLivePromptMock.mockReset();
    getSkillMock.mockReset();
    renderSimulationPromptMock.mockResolvedValue({
      mode: "holistic-loop",
      preset: "happy-path",
      stepIndex: 0,
      phase: "investigate",
      skillId: "swarm-manager-holistic-loop-investigate",
      profileKey: "swarm-manager/deep-work",
      variables: { INITIATIVE_TITLE: "Holistic Loop Simulation" },
      prompt: "Rendered simulation prompt for Holistic Loop Simulation.",
      degraded: false,
    });
    renderLivePromptMock.mockResolvedValue({
      mode: "holistic-loop",
      phase: "execute",
      skillId: "swarm-manager-holistic-loop-execute",
      profileKey: "swarm-manager/deep-work",
      variables: { INITIATIVE_TITLE: "Initiative A" },
      prompt: "Rendered live prompt for Initiative A.",
      degraded: false,
    });
    getSkillMock.mockResolvedValue({ id: "skill", name: "Skill", current_content: "Template {{INITIATIVE_TITLE}}." });
    catalogMock.mockResolvedValue({ modes: [SAMPLE_DETAIL.entry] });
    simulateModeMock.mockResolvedValue({
      mode: "holistic-loop",
      label: "Holistic Loop",
      activePreset: "happy-path",
      presets: [
        {
          id: "happy-path",
          label: "Clean pass",
          description: "Investigate → plan → execute → review → reconcile.",
          branch: "execute → review (replan not needed)",
          scenario: "A clean pass with no replanning.",
        },
        {
          id: "replan-after-execute",
          label: "Execute triggers replan",
          description: "Execution loops back to investigate before finishing.",
          branch: "execute → investigate (replan_needed)",
          scenario: "The first plan misses something material.",
        },
      ],
      initiative: {
        name: "simulation-sandbox",
        title: "Holistic Loop Simulation",
        mode: "holistic-loop",
        items: ["execute/item-a"],
        acceptanceCriteria: ["works"],
      },
      trace: [
        {
          index: 0,
          phase: "investigate",
          phaseKind: "investigate",
          inputs: {
            initiative: {
              name: "simulation-sandbox",
              title: "Holistic Loop Simulation",
              mode: "holistic-loop",
              items: ["execute/item-a"],
              acceptanceCriteria: ["works"],
            },
            items: [{ ref: "execute/item-a" }],
            artifacts: [],
            priorRounds: [],
            acceptanceCriteria: ["works"],
          },
          output: { handoff: { summary: "investigated" } },
          round: {
            round: 1,
            mode: "holistic-loop",
            scopeKind: "initiative",
            scopeId: "simulation-sandbox",
            initiativeName: "simulation-sandbox",
            phase: "investigate",
            runStrategy: "operator_gated_loop",
            agentProfileKey: "swarm-manager/deep-work",
            generatedAt: "2026-04-30T00:00:00Z",
            status: "completed",
          },
          transition: { from: "investigate", to: "execute", conditionKind: "always", label: "always" },
        },
        {
          index: 1,
          phase: "execute",
          phaseKind: "execute",
          inputs: {
            initiative: {
              name: "simulation-sandbox",
              title: "Holistic Loop Simulation",
              mode: "holistic-loop",
              items: ["execute/item-a"],
              acceptanceCriteria: ["works"],
            },
            items: [{ ref: "execute/item-a" }],
            artifacts: [],
            priorRounds: [],
            acceptanceCriteria: ["works"],
          },
          output: { handoff: { summary: "executed" } },
          round: {
            round: 2,
            mode: "holistic-loop",
            scopeKind: "initiative",
            scopeId: "simulation-sandbox",
            initiativeName: "simulation-sandbox",
            phase: "execute",
            runStrategy: "operator_gated_loop",
            agentProfileKey: "swarm-manager/deep-work",
            generatedAt: "2026-04-30T00:00:00Z",
            status: "completed",
          },
          terminal: true,
        },
      ],
    });
    workspaceMock.mockResolvedValue({
      initiativeName: "init-a",
      mode: "holistic-loop",
      definition: {
        mode: "holistic-loop",
        label: "Holistic Loop",
        scopeKind: "initiative",
        runStrategy: "operator_gated_loop",
        capabilities: SAMPLE_DETAIL.entry.capabilities,
        phases: [],
        terminal: ["review"],
        transitions: {},
      },
      artifacts: [{ path: "modes/holistic-loop/findings.md", content: "Findings" }],
      rounds: [
        {
          round: 1,
          mode: "holistic-loop",
          scopeKind: "initiative",
          scopeId: "init-a",
          initiativeName: "init-a",
          phase: "execute",
          runStrategy: "operator_gated_loop",
          agentProfileKey: "swarm-manager/deep-work",
          generatedAt: "2026-04-30T00:00:00Z",
          runId: "run-live-1",
          status: "completed",
          items: [{ ref: "execute/item-a", title: "Item A" }],
          payload: {
            agent_summary: "live completed",
            verdict: "accepted",
            progress: { decision: "continue" },
            backlog_sync: { completed_items: ["execute/item-a"] },
          },
        },
      ],
    });
    navigateMock.mockReset();
  });

  it("renders mode metadata and humanized overview enums", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

    expect(await screen.findByText("Holistic Loop")).toBeInTheDocument();
    expect(screen.getByTestId("operating-mode-details-tab-row")).toBeInTheDocument();
    expect(screen.getByTestId("operating-mode-details-tab-overview")).toHaveAttribute("aria-selected", "true");
    expect(screen.getByText("Investigate→plan→execute cycles")).toBeInTheDocument();
    expect(screen.getByText("Initiative")).toBeInTheDocument();
    expect(screen.getByText("Operator-gated loop")).toBeInTheDocument();
    expect(screen.getAllByText("Initiative A").length).toBeGreaterThan(0);
    expect(screen.getAllByText("Initiative B").length).toBeGreaterThan(0);
    expect(screen.queryByTestId("operating-mode-simulation-panel")).not.toBeInTheDocument();
  });

  it("renders phase cards with clean label, purpose, and chips", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=phases&view=list");

    expect(await screen.findByRole("heading", { name: "Investigate" })).toBeInTheDocument();
    expect(screen.queryByRole("heading", { name: "Holistic Loop Investigate" })).not.toBeInTheDocument();
    expect(screen.getByText("Investigate initiative-wide code, backlog, and system state.")).toBeInTheDocument();
    // start chip
    expect(screen.getByText("start")).toBeInTheDocument();
    // terminal chip on review
    expect(screen.getByText("terminal")).toBeInTheDocument();
    // writes-repo chip on execute
    expect(screen.getByText("writes repo")).toBeInTheDocument();
    // each phase card composes the shared viewer in contract source
    const viewers = screen.getAllByTestId("operating-mode-phase-viewer");
    expect(viewers.length).toBe(SAMPLE_DETAIL.entry.phases.length);
    expect(viewers[0]).toHaveAttribute("data-source", "contract");
  });

  it("toggles list/graph view via the action buttons", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=phases");

    // Default is graph view; the graph container should be present.
    await waitFor(() => {
      expect(screen.getByTestId("phase-graph")).toBeInTheDocument();
    });
    const toggle = screen.getByTestId("operating-mode-phases-view-toggle");
    const listButton = toggle.querySelector('button[aria-pressed="false"]');
    expect(listButton).not.toBeNull();
    if (listButton) fireEvent.click(listButton);
    await waitFor(() => {
      expect(screen.queryByTestId("phase-graph")).not.toBeInTheDocument();
    });
  });

  it("renders the Flow tab, not an Execution tab", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=flow");

    expect(await screen.findByTestId("operating-mode-details-tab-flow")).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(screen.queryByTestId("operating-mode-details-tab-execution")).not.toBeInTheDocument();
    // The Flow intro states the two data sources.
    expect(screen.getByText(/How this mode flows/i)).toBeInTheDocument();
  });

  it("opens the Flow guide dialog", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=flow");

    fireEvent.click(await screen.findByTestId("operating-mode-flow-guide-button"));
    const guide = await screen.findByTestId("operating-mode-flow-guide-dialog");
    expect(guide).toHaveAttribute("role", "dialog");
    expect(within(guide).getByText(/deterministic, in-memory walk/i)).toBeInTheDocument();
    expect(within(guide).getByText(/actual rounds recorded/i)).toBeInTheDocument();
  });

  it("renders one source-toggled viewer with step controls that advance the phase", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=flow");

    const viewer = await screen.findByTestId("operating-mode-phase-viewer");
    expect(simulateModeMock).toHaveBeenCalledWith("holistic-loop", undefined);
    // Default source is the simulation preset; the viewer shows the first phase.
    expect(viewer).toHaveAttribute("data-source", "simulation");
    expect(viewer).toHaveAttribute("data-phase", "investigate");
    expect(screen.getByText(/step 1 \/ 2/i)).toBeInTheDocument();
    // Both stacked panels are gone; there is exactly one flow viewer.
    expect(screen.getAllByTestId("operating-mode-phase-viewer")).toHaveLength(1);
    expect(screen.queryByTestId("operating-mode-simulation-panel")).not.toBeInTheDocument();
    expect(screen.queryByTestId("operating-mode-live-panel")).not.toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /step/i }));

    await waitFor(() =>
      expect(screen.getByTestId("operating-mode-phase-viewer")).toHaveAttribute("data-phase", "execute"),
    );
    // Semantic transition + emit rendering behind the concern tabs.
    fireEvent.click(screen.getByTestId("operating-mode-phase-viewer-tab-transition"));
    expect(screen.getByTestId("operating-mode-flow-trace-transition")).toHaveTextContent(/terminal phase/i);
    fireEvent.click(screen.getByTestId("operating-mode-phase-viewer-tab-emits"));
    expect(screen.getByTestId("operating-mode-flow-trace-emits")).toHaveTextContent(/executed/i);
  });

  it("toggles the flow source to Contract to reveal the unfilled template slots", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=flow");

    fireEvent.click(await screen.findByTestId("operating-mode-flow-source-contract"));
    await waitFor(() =>
      expect(screen.getByTestId("operating-mode-phase-viewer")).toHaveAttribute("data-source", "contract"),
    );
    await waitFor(() =>
      expect(screen.getByTestId("operating-mode-phase-viewer-prompt")).toHaveTextContent("{{INITIATIVE_TITLE}}"),
    );
  });

  it("selects a simulation preset and re-runs the simulation", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=flow");

    const select = await screen.findByTestId("operating-mode-flow-preset-select");
    // The active preset's demonstrated branch is shown.
    expect(screen.getByTestId("operating-mode-flow-preset-scenario")).toHaveTextContent(
      "execute → review",
    );

    fireEvent.change(select, { target: { value: "replan-after-execute" } });
    await waitFor(() => {
      expect(simulateModeMock).toHaveBeenCalledWith("holistic-loop", "replan-after-execute");
    });
  });

  it("renders live round payloads as semantic reads/emits/transition when the Live source is selected", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=flow");

    fireEvent.click(await screen.findByTestId("operating-mode-flow-source-live"));
    await waitFor(() => expect(workspaceMock).toHaveBeenCalledWith("init-a"));
    await waitFor(() =>
      expect(screen.getByTestId("operating-mode-phase-viewer")).toHaveAttribute("data-source", "live"),
    );
    const viewer = screen.getByTestId("operating-mode-phase-viewer");
    expect(viewer).toHaveAttribute("data-phase", "execute");

    // Reads render from the phase's declared, composed contract (looked up
    // from the catalog phase for the live round), grouped by provider.
    fireEvent.click(within(viewer).getByTestId("operating-mode-phase-viewer-tab-reads"));
    const reads = within(viewer).getByTestId("operating-mode-flow-trace-reads");
    expect(reads).toHaveTextContent("{{MEMBER_ITEMS_JSON}}");
    expect(within(reads).getByTestId("phase-reads-group-target")).toBeInTheDocument();

    fireEvent.click(within(viewer).getByTestId("operating-mode-phase-viewer-tab-emits"));
    const emits = within(viewer).getByTestId("operating-mode-flow-trace-emits");
    expect(emits).toHaveTextContent("verdict");
    expect(emits).toHaveTextContent("backlog_sync");
    // Raw payload stays available behind disclosure.
    expect(within(viewer).getByTestId("operating-mode-flow-trace-raw-toggle")).toHaveTextContent(
      "View raw payload",
    );

    fireEvent.click(within(viewer).getByTestId("operating-mode-phase-viewer-tab-transition"));
    expect(within(viewer).getByTestId("operating-mode-flow-trace-transition")).toHaveTextContent(
      "execute → review",
    );
  });

  it("switches the live viewer between linked initiative workspaces", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=flow");

    fireEvent.click(await screen.findByTestId("operating-mode-flow-source-live"));
    const select = await screen.findByLabelText("Live initiative");
    fireEvent.change(select, { target: { value: "init-b" } });

    await waitFor(() => {
      expect(workspaceMock).toHaveBeenCalledWith("init-b");
    });
  });

  it("opens phase internals disclosure when clicked", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=phases&view=list");

    const disclosure = await screen.findByTestId("phase-internals-investigate");
    expect(disclosure).not.toHaveAttribute("open");
    fireEvent.click(disclosure.querySelector("summary") as Element);
    expect(disclosure).toHaveAttribute("open");
    // catalog id surfaces in the open body
    expect(screen.getByText("swarm-manager-holistic-loop.investigate")).toBeInTheDocument();
  });

  it("navigates to the linked initiative when clicked", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

    const links = await screen.findAllByTestId("operating-mode-linked-initiative");
    const first = links[0];
    if (!first) throw new Error("expected at least one linked initiative");
    fireEvent.click(first);
    expect(navigateMock).toHaveBeenCalledWith("/initiatives/init-a");
  });

  it("saves edits via updateMode", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    updateModeMock.mockResolvedValue({
      ...SAMPLE_DETAIL,
      entry: { ...SAMPLE_DETAIL.entry, label: "Renamed Loop", description: "New text" },
    });
    renderPage();

    const editButton = await screen.findByRole("button", { name: /edit/i });
    fireEvent.click(editButton);

    const labelInput = screen.getByTestId("operating-mode-label-input");
    const descInput = screen.getByTestId("operating-mode-description-input");
    fireEvent.change(labelInput, { target: { value: "Renamed Loop" } });
    fireEvent.change(descInput, { target: { value: "New text" } });

    fireEvent.click(screen.getByRole("button", { name: /save/i }));

    await waitFor(() => {
      expect(updateModeMock).toHaveBeenCalledWith("holistic-loop", {
        label: "Renamed Loop",
        description: "New text",
      });
    });
  });

  it("disables save when label is blank", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

    fireEvent.click(await screen.findByRole("button", { name: /edit/i }));
    const labelInput = screen.getByTestId("operating-mode-label-input");
    fireEvent.change(labelInput, { target: { value: "   " } });

    expect(screen.getByRole("button", { name: /save/i })).toBeDisabled();
  });

  it("renders decision-support sections from the catalog metadata", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=guidance");

    await screen.findByText("Holistic Loop");
    expect(screen.getByTestId("operating-mode-details-best-for")).toHaveTextContent("Coupled work");
    expect(screen.getByTestId("operating-mode-details-not-for")).toHaveTextContent("Independent items");
    expect(screen.getByTestId("operating-mode-details-tradeoffs")).toHaveTextContent("One plan, not N");
    expect(screen.getByTestId("operating-mode-details-learn-more")).toBeInTheDocument();
  });

  it("opens the scope explainer dialog when the info icon is clicked", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

    await screen.findByText("Holistic Loop");
    expect(screen.queryByTestId("concept-explainer-dialog")).not.toBeInTheDocument();

    fireEvent.click(screen.getByTestId("operating-mode-details-scope-info"));

    const dialog = screen.getByTestId("concept-explainer-dialog");
    expect(dialog).toHaveAttribute("role", "dialog");
    expect(screen.getByRole("heading", { name: "Target" })).toBeInTheDocument();
  });

  it("renders the disabled docs fallback when the docs URL is unavailable", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=guidance");

    await screen.findByText("Holistic Loop");
    const learnMore = screen.getByTestId("operating-mode-details-learn-more");
    expect(learnMore).toHaveTextContent("Docs server unavailable.");
    // No anchor links since useDocsUrl returns null in test env.
    expect(learnMore.querySelector("a")).toBeNull();
  });
});
