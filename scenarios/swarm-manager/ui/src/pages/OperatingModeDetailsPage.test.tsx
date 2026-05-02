import { fireEvent, render, screen, waitFor } from "@testing-library/react";
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
const navigateMock = vi.fn();

vi.mock("react-router-dom", async () => {
  const actual = await vi.importActual<typeof import("react-router-dom")>("react-router-dom");
  return { ...actual, useNavigate: () => navigateMock };
});

vi.mock("../services", () => ({
  initiativeModeService: {
    getMode: (mode: string) => getModeMock(mode),
    updateMode: (mode: string, args: unknown) => updateModeMock(mode, args),
  },
}));

vi.mock("../app/shell/AppShellContext", () => ({
  useAppShell: () => ({ openSidebar: () => {}, closeSidebar: () => {} }),
}));

beforeAll(() => {
  installResizeObserverMock();
  installMatchMediaMock();
  // jsdom doesn't implement scrollIntoView; the page calls it on graph clicks.
  Element.prototype.scrollIntoView = vi.fn();
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

const SAMPLE_DETAIL: OperatingModeDetail = {
  entry: {
    mode: "holistic-loop",
    label: "Holistic Loop",
    description: "Investigate→plan→execute cycles",
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
    navigateMock.mockReset();
  });

  it("renders mode metadata and humanized overview enums", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

    expect(await screen.findByText("Holistic Loop")).toBeInTheDocument();
    expect(screen.getByText("Investigate→plan→execute cycles")).toBeInTheDocument();
    expect(screen.getByText("Initiative")).toBeInTheDocument();
    expect(screen.getByText("Operator-gated loop")).toBeInTheDocument();
    expect(screen.getByText("Initiative A")).toBeInTheDocument();
    expect(screen.getByText("Initiative B")).toBeInTheDocument();
  });

  it("renders phase cards with title, purpose, and chips", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?view=list");

    expect(await screen.findByText("Holistic Loop Investigate")).toBeInTheDocument();
    expect(screen.getByText("Investigate initiative-wide code, backlog, and system state.")).toBeInTheDocument();
    // start chip
    expect(screen.getByText("start")).toBeInTheDocument();
    // terminal chip on review
    expect(screen.getByText("terminal")).toBeInTheDocument();
    // verdict chip on review
    expect(screen.getByText("verdict")).toBeInTheDocument();
    // writes-repo chip on execute
    expect(screen.getByText("writes repo")).toBeInTheDocument();
  });

  it("toggles list/graph view via the action buttons", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage();

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

  it("opens phase internals disclosure when clicked", async () => {
    getModeMock.mockResolvedValue(SAMPLE_DETAIL);
    renderPage("holistic-loop", "?view=list");

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
});
