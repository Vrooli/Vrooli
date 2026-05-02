import { describe, expect, it, vi } from "vitest";
import { render, screen, within } from "@testing-library/react";
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
  it("renders an empty state when there are no rounds", () => {
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
    expect(screen.getByText("No operating-mode rounds have run yet.")).toBeInTheDocument();
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
