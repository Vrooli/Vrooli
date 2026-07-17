import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router-dom";
import { OperatingModeHero } from "./operating-mode-hero";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCapabilities,
  OperatingModeCatalogEntry,
} from "../../../types/operating-mode";

const itemExecutionCapabilities: OperatingModeCapabilities = {
  supportsPhases: false,
  canStartPhases: false,
  canCompleteItems: false,
  canApplyBacklogSyncProposals: false,
  requiresAcceptanceCriteria: false,
  supportsArtifacts: false,
  supportsHandoffs: false,
};

function makeEntry(overrides: Partial<OperatingModeCatalogEntry> = {}): OperatingModeCatalogEntry {
  return {
    mode: "holistic-loop",
    label: "Holistic Loop",
    description: "Iterative initiative loop.",
    bestFor: ["Coupled work"],
    notFor: ["Independent items"],
    tradeoffs: ["One plan, not N"],
    usageCount: 1,
    targetKind: "initiative",
    runStrategy: "operator_gated_loop",
    workspaceTabId: "operating-mode",
    capabilities: itemExecutionCapabilities,
    default: false,
    switchable: true,
    supportsPhases: true,
    phases: [],
    ...overrides,
  };
}

function renderHero(props: Partial<Parameters<typeof OperatingModeHero>[0]> = {}) {
  return render(
    <MemoryRouter>
      <OperatingModeHero
        currentMode="holistic-loop"
        catalogEntry={makeEntry()}
        onSwitchClick={() => {}}
        {...props}
      />
    </MemoryRouter>,
  );
}

describe("OperatingModeHero", () => {
  it("renders the mode label and currentMode chip", () => {
    renderHero();
    expect(screen.getByText("Holistic Loop")).toBeInTheDocument();
    expect(screen.getByText("holistic-loop")).toBeInTheDocument();
  });

  it("wraps the label and chip in a link to the operating-mode details page", () => {
    renderHero();
    const link = screen.getByTestId(selectors.initiativeDetails.modeHeroLink);
    expect(link.tagName).toBe("A");
    expect(link).toHaveAttribute("href", "/operating-modes/holistic-loop");
    expect(link).toHaveTextContent("Holistic Loop");
    expect(link).toHaveTextContent("holistic-loop");
  });

  it("does not invoke onSwitchClick when the label link is clicked", async () => {
    const onSwitchClick = vi.fn();
    renderHero({ onSwitchClick });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.modeHeroLink));
    expect(onSwitchClick).not.toHaveBeenCalled();
  });

  it("invokes onSwitchClick when the Switch Mode button is clicked", async () => {
    const onSwitchClick = vi.fn();
    renderHero({ onSwitchClick });
    await userEvent.click(screen.getByTestId(selectors.initiativeDetails.modeHeroSwitchButton));
    expect(onSwitchClick).toHaveBeenCalledTimes(1);
  });

  it("uses the raw mode key as the link target when no catalog entry is provided", () => {
    renderHero({ catalogEntry: undefined, currentMode: "phased-plan-drain" });
    const link = screen.getByTestId(selectors.initiativeDetails.modeHeroLink);
    expect(link).toHaveAttribute("href", "/operating-modes/phased-plan-drain");
  });

  it("renders the running-round badge when a round is running", () => {
    renderHero({
      runningRound: {
        mode: "holistic-loop",
        round: 7,
        phase: "investigate",
        status: "agent_running",
        scopeKind: "initiative",
        scopeId: "demo",
        runStrategy: "operator_gated_loop",
        agentProfileKey: "swarm-manager/deep-work",
        generatedAt: "2026-05-02T00:00:00Z",
      },
    });
    expect(screen.getByText(/Round 7 running/)).toBeInTheDocument();
  });
});
