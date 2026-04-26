import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { EvidencePanel } from "./evidence-panel";
import type { ReviewRound } from "../../services/review-service";

function makeRound(overrides?: Partial<ReviewRound>): ReviewRound {
  return {
    round: 1,
    generated_at: "2026-04-26T12:00:00Z",
    execution_id: "exec-1",
    status: "gathering",
    evidence: [],
    ...overrides,
  };
}

const defaultProps = {
  rounds: [] as ReviewRound[],
  backlogKind: "execute",
  backlogName: "test-item",
  isGathering: false,
  isAwaitingManualReview: false,
  onVerify: vi.fn(),
  onRequestMore: vi.fn(),
};

describe("EvidencePanel", () => {
  it("shows gathering copy for actively gathering rounds", () => {
    render(
      <EvidencePanel
        {...defaultProps}
        rounds={[makeRound()]}
        isGathering
      />,
    );
    expect(screen.getByText("Gathering evidence...")).toBeInTheDocument();
    expect(screen.getByText("Gathering")).toBeInTheDocument();
  });

  it("shows awaiting manual review copy when the review run is parked in needs_review", () => {
    render(
      <EvidencePanel
        {...defaultProps}
        rounds={[makeRound({ current_run_status: "needs_review" })]}
        isAwaitingManualReview
      />,
    );
    expect(screen.getByText("Awaiting manual review...")).toBeInTheDocument();
    expect(screen.getByText("Awaiting review")).toBeInTheDocument();
    expect(screen.queryByText("Gathering evidence...")).not.toBeInTheDocument();
  });
});
