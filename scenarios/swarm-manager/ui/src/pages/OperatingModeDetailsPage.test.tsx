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
    simulateMode: (mode: string) => simulateModeMock(mode),
    workspace: (name: string) => workspaceMock(name),
    refreshRound: (name: string, mode: string, round: number) => refreshRoundMock(name, mode, round),
  },
}));

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
    scopeKind: "initiative",
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
      usesItemExecutionFlow: false,
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
        { from: "execute", to: "investigate", conditionKind: "payload_bool", label: "on payload.replan_needed=true", payloadKey: "replan_needed" },
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
    catalogMock.mockResolvedValue({ modes: [SAMPLE_DETAIL.entry] });
    simulateModeMock.mockResolvedValue({
      mode: "holistic-loop",
      label: "Holistic Loop",
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
    // output schema is shown on phase cards
    expect(screen.getAllByText("verdict").length).toBeGreaterThan(0);
    // writes-repo chip on execute
    expect(screen.getByText("writes repo")).toBeInTheDocument();
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

  it("renders simulation trace controls and advances the highlighted phase", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=execution");

    const panel = await screen.findByTestId("operating-mode-simulation-panel");
    expect(panel).toBeInTheDocument();
    expect(simulateModeMock).toHaveBeenCalledWith("holistic-loop");
    expect(screen.getByText("1 / 2 · investigate")).toBeInTheDocument();

    fireEvent.click(screen.getByRole("button", { name: /step/i }));

    expect(screen.getByText("2 / 2 · execute")).toBeInTheDocument();
    expect(within(panel).getByText("terminal")).toBeInTheDocument();
    expect(within(panel).getByText(/executed/i)).toBeInTheDocument();
  });

  it("renders live round payloads on the shared phase trace substrate", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=execution");

    const panel = await screen.findByTestId("operating-mode-live-panel");
    expect(workspaceMock).toHaveBeenCalledWith("init-a");
    await waitFor(() => {
      expect(panel).toHaveTextContent("Round 1 · execute");
    });
    expect(panel).toHaveTextContent("init-a");
    expect(panel).toHaveTextContent("Items");
    expect(panel).toHaveTextContent("Artifacts");
    expect(panel).toHaveTextContent("verdict");
    expect(panel).toHaveTextContent("backlog_sync");
    expect(panel).toHaveTextContent("execute -> review (always)");
  });

  it("switches the live viewer between linked initiative workspaces", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?tab=execution");

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
    expect(screen.getByRole("heading", { name: "Scope" })).toBeInTheDocument();
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
