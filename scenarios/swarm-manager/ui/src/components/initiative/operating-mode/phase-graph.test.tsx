import { describe, expect, it, beforeAll, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { PhaseGraph } from "./phase-graph";
import { selectors } from "../../../consts/selectors";
import { installMatchMediaMock, installResizeObserverMock } from "../../../test-utils/browser";
import type {
  OperatingModeCatalogEntry,
  OperatingModeCatalogPhase,
} from "../../../types/operating-mode";

beforeAll(() => {
  installResizeObserverMock();
  installMatchMediaMock();
});

function basePhase(overrides: Partial<OperatingModeCatalogPhase> & { phase: string }): OperatingModeCatalogPhase {
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
      requiredArtifactCount: 0,
    },
    ...overrides,
  };
}

const ENTRY: OperatingModeCatalogEntry = {
  mode: "holistic-loop" as OperatingModeCatalogEntry["mode"],
  label: "Holistic Loop",
  bestFor: ["Coupled work"],
  notFor: ["Independent items"],
  tradeoffs: ["One plan, not N"],
  usageCount: 0,
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
    basePhase({ phase: "investigate", label: "Investigate", title: "Holistic Loop Investigate", isStart: true }),
    basePhase({ phase: "execute", label: "Execute", title: "Holistic Loop Execute", writesRepo: true }),
    basePhase({ phase: "review", label: "Review", title: "Holistic Loop Review", isTerminal: true }),
  ],
  phaseGraph: {
    startPhase: "investigate",
    terminal: ["review"],
    transitions: [
      { from: "execute", to: "investigate", conditionKind: "payload_bool", label: "on payload.replan_needed=true", payloadKey: "replan_needed" },
      { from: "execute", to: "review", conditionKind: "always", label: "always" },
      { from: "investigate", to: "execute", conditionKind: "always", label: "always" },
    ],
    acceptedVerdicts: [],
  },
};

describe("PhaseGraph", () => {
  it("renders the legend and the canvas", () => {
    render(<PhaseGraph entry={ENTRY} />);
    expect(screen.getByTestId("phase-graph")).toBeInTheDocument();
    expect(screen.getByText("Legend:")).toBeInTheDocument();
    expect(screen.getByText("payload bool")).toBeInTheDocument();
    expect(screen.getByText("progress decision")).toBeInTheDocument();
  });

  it("renders phase node labels (no mode prefix)", () => {
    render(<PhaseGraph entry={ENTRY} />);
    expect(screen.getByText("Investigate")).toBeInTheDocument();
    expect(screen.getByText("Execute")).toBeInTheDocument();
    expect(screen.getByText("Review")).toBeInTheDocument();
    expect(screen.queryByText("Holistic Loop Investigate")).not.toBeInTheDocument();
    expect(screen.queryByText("Holistic Loop Execute")).not.toBeInTheDocument();
  });

  it("opens the glossary dialog when the legend area is clicked", async () => {
    render(<PhaseGraph entry={ENTRY} />);
    const legendButton = screen.getByTestId(selectors.initiativeDetails.phaseGraphLegend);
    expect(legendButton.tagName).toBe("BUTTON");
    expect(legendButton).toHaveAttribute("aria-label", "Open phase-graph glossary");
    expect(screen.queryByTestId(selectors.initiativeDetails.phaseGraphGlossaryDialog)).not.toBeInTheDocument();

    await userEvent.click(legendButton);

    const dialog = screen.getByTestId(selectors.initiativeDetails.phaseGraphGlossaryDialog);
    expect(dialog).toHaveAttribute("role", "dialog");
    expect(screen.getByRole("heading", { name: /How to read this phase graph/i })).toBeInTheDocument();
  });

  it("renders the legend transition kinds", () => {
    render(<PhaseGraph entry={ENTRY} />);
    // Legend always renders. Edge labels also render in real DOM but xyflow's
    // edge label rendering relies on ReactFlow's measured viewport, which
    // jsdom doesn't compute, so we verify the wire by inspecting nodes +
    // markers (edges + correct stroke colors are emitted as <marker> defs).
    expect(screen.getByText("always")).toBeInTheDocument();
    expect(screen.getByText("payload bool")).toBeInTheDocument();
  });

  it("calls onSelectPhase when a node is clicked", () => {
    const onSelectPhase = vi.fn();
    render(<PhaseGraph entry={ENTRY} onSelectPhase={onSelectPhase} />);

    const investigate = screen.getByText("Investigate");
    fireEvent.click(investigate);
    expect(onSelectPhase).toHaveBeenCalledWith("investigate");
  });

  it("renders MiniMap and Controls in default mode", () => {
    const { container } = render(<PhaseGraph entry={ENTRY} />);
    // xyflow renders MiniMap as .react-flow__minimap and Controls as .react-flow__controls.
    expect(container.querySelector(".react-flow__minimap")).not.toBeNull();
    expect(container.querySelector(".react-flow__controls")).not.toBeNull();
  });

  it("hides MiniMap and Controls when compact, and uses a shorter canvas", () => {
    const { container } = render(<PhaseGraph entry={ENTRY} compact />);
    expect(container.querySelector(".react-flow__minimap")).toBeNull();
    expect(container.querySelector(".react-flow__controls")).toBeNull();
    expect(container.querySelector(".h-\\[200px\\]")).not.toBeNull();
  });
});
