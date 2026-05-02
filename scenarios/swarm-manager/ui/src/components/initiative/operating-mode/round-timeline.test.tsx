import { describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { RoundTimeline } from "./round-timeline";
import { selectors } from "../../../consts/selectors";
import type {
  OperatingModeCapabilities,
  OperatingModeRound,
} from "../../../types/operating-mode";

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

function round(overrides: Partial<OperatingModeRound> & { round: number; phase: string }): OperatingModeRound {
  return {
    mode: "holistic-loop",
    scopeKind: "initiative",
    scopeId: "init-a",
    runStrategy: "operator_gated_loop",
    agentProfileKey: "swarm-manager/deep-work",
    generatedAt: "2026-04-30T00:00:00Z",
    runId: `run-${overrides.round}`,
    status: "completed",
    ...overrides,
  };
}

describe("RoundTimeline", () => {
  it("renders the supportsPhases empty state when there are no rounds", () => {
    render(
      <RoundTimeline
        rounds={[]}
        capabilities={capabilities}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
      />,
    );
    const empty = screen.getByTestId(selectors.initiativeDetails.roundTimelineEmpty);
    expect(empty).toHaveTextContent(/Start a phase from the composer above/);
    expect(empty).toHaveAttribute("data-supports-phases", "true");
  });

  it("renders the item-level empty state when supportsPhases is false", () => {
    render(
      <RoundTimeline
        rounds={[]}
        capabilities={{ ...capabilities, supportsPhases: false, usesItemExecutionFlow: true }}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
      />,
    );
    const empty = screen.getByTestId(selectors.initiativeDetails.roundTimelineEmpty);
    expect(empty).toHaveTextContent(/does not run rounds/);
    expect(empty).toHaveAttribute("data-supports-phases", "false");
  });

  it("groups rounds by phase, newest phase first", () => {
    const rounds: OperatingModeRound[] = [
      round({ round: 1, phase: "investigate" }),
      round({ round: 2, phase: "investigate" }),
      round({ round: 3, phase: "plan" }),
      round({ round: 4, phase: "investigate" }),
    ];
    render(
      <RoundTimeline
        rounds={rounds}
        capabilities={capabilities}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
      />,
    );
    const groups = screen.getAllByTestId(selectors.initiativeDetails.roundsPhaseGroup);
    expect(groups).toHaveLength(2);
    // The most-recent round (round 4, phase=investigate) means investigate appears first.
    expect(groups[0]!).toHaveAttribute("data-phase", "investigate");
    expect(groups[1]!).toHaveAttribute("data-phase", "plan");
    expect(within(groups[0]!).getByText(/3 rounds/)).toBeInTheDocument();
    expect(within(groups[1]!).getByText(/1 round/)).toBeInTheDocument();
  });

  it("collapses rounds beyond the first 10 with a Show-more affordance", async () => {
    const rounds: OperatingModeRound[] = Array.from({ length: 12 }, (_, i) =>
      round({ round: i + 1, phase: "investigate" }),
    );
    render(
      <RoundTimeline
        rounds={rounds}
        capabilities={capabilities}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
      />,
    );
    const cardsBeforeExpand = screen.getAllByTestId(selectors.initiativeDetails.roundCard);
    expect(cardsBeforeExpand.length).toBe(10);
    const showMore = screen.getByTestId(selectors.initiativeDetails.roundTimelineShowMore);
    expect(showMore).toHaveTextContent(/Show 2 more rounds/);

    await userEvent.click(showMore);

    const cardsAfterExpand = screen.getAllByTestId(selectors.initiativeDetails.roundCard);
    expect(cardsAfterExpand.length).toBe(12);
    expect(screen.queryByTestId(selectors.initiativeDetails.roundTimelineShowMore)).toBeNull();
  });

  it("does not render the Show-more button when bucket has 10 or fewer rounds", () => {
    const rounds: OperatingModeRound[] = Array.from({ length: 10 }, (_, i) =>
      round({ round: i + 1, phase: "investigate" }),
    );
    render(
      <RoundTimeline
        rounds={rounds}
        capabilities={capabilities}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
      />,
    );
    expect(screen.queryByTestId(selectors.initiativeDetails.roundTimelineShowMore)).toBeNull();
    expect(screen.getAllByTestId(selectors.initiativeDetails.roundCard).length).toBe(10);
  });

  it("shows the last-status pill in the bucket summary", () => {
    const rounds: OperatingModeRound[] = [
      round({ round: 1, phase: "investigate", status: "completed" }),
      round({ round: 2, phase: "investigate", status: "failed" }),
    ];
    render(
      <RoundTimeline
        rounds={rounds}
        capabilities={capabilities}
        busy={false}
        onRefresh={vi.fn()}
        onCancel={vi.fn()}
        onCompleteItems={vi.fn()}
        onApplyBacklogSync={vi.fn()}
      />,
    );
    const groups = screen.getAllByTestId(selectors.initiativeDetails.roundsPhaseGroup);
    expect(within(groups[0]!).getByText(/last: Failed/i)).toBeInTheDocument();
  });
});
