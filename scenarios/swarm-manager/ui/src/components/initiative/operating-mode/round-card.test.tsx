import { describe, expect, it, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RoundCard } from "./round-card";
import { selectors } from "../../../consts/selectors";
import type { OperatingModeCapabilities, OperatingModeRound } from "../../../types/operating-mode";

const capabilities: OperatingModeCapabilities = {
  supportsPhases: true,
  canStartPhases: true,
  canCompleteItems: true,
  canApplyBacklogSyncProposals: true,
  requiresAcceptanceCriteria: true,
  supportsArtifacts: true,
  supportsHandoffs: false,
  usesItemExecutionFlow: false,
};

function makeRound(overrides: Partial<OperatingModeRound> = {}): OperatingModeRound {
  return {
    mode: "holistic-loop",
    round: 1,
    scopeKind: "initiative",
    scopeId: "init-a",
    phase: "investigate",
    runStrategy: "operator_gated_loop",
    agentProfileKey: "swarm-manager/deep-work",
    generatedAt: "2026-04-30T00:00:00Z",
    runId: "run-1",
    status: "completed",
    ...overrides,
  };
}

describe("RoundCard", () => {
  it("renders the round number, status pill, and phase label", () => {
    render(
      <RoundCard
        round={makeRound()}
        capabilities={capabilities}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
      />,
    );
    expect(screen.getByText("Round 1")).toBeInTheDocument();
    // Phase label is title-cased by phaseLabel("investigate") = "Investigate".
    expect(screen.getByText("Investigate")).toBeInTheDocument();
  });

  it("does not render the View-details button when no callback is provided", () => {
    render(
      <RoundCard
        round={makeRound()}
        capabilities={capabilities}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
      />,
    );
    expect(screen.queryByTestId(selectors.initiativeDetails.roundDetailButton)).toBeNull();
  });

  it("renders the View-details button and invokes onViewDetails with the round when clicked", async () => {
    const onViewDetails = vi.fn();
    const round = makeRound();
    render(
      <RoundCard
        round={round}
        capabilities={capabilities}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
        onViewDetails={onViewDetails}
      />,
    );
    const button = screen.getByTestId(selectors.initiativeDetails.roundDetailButton);
    await userEvent.click(button);
    expect(onViewDetails).toHaveBeenCalledTimes(1);
    expect(onViewDetails).toHaveBeenCalledWith(round);
  });
});
