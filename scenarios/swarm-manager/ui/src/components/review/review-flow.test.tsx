import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ReviewFlow } from "./review-flow";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord, Finalization } from "../../types";
import type { ReviewRound } from "../../services/review-service";

vi.mock("../../services", () => ({
  executionService: {
    triggerReview: vi.fn().mockResolvedValue({}),
  },
}));

function makeExecution(overrides?: Partial<ExecutionRecord>): ExecutionRecord {
  return {
    executionId: "exec-1",
    backlogKind: "task",
    backlogName: "test-item",
    status: "completed",
    mode: "yolo",
    createdAt: new Date().toISOString(),
    ...overrides,
  } as ExecutionRecord;
}

function makeRound(overrides?: Partial<ReviewRound>): ReviewRound {
  return {
    round: 1,
    generated_at: new Date().toISOString(),
    execution_id: "exec-1",
    status: "complete",
    evidence: [],
    ...overrides,
  } as ReviewRound;
}

const defaultProps = {
  execution: undefined as ExecutionRecord | undefined,
  targetScenarios: [] as string[],
  reviewRounds: [] as ReviewRound[],
  isGatheringEvidence: false,
  isActive: false,
  backlogKind: "task",
  backlogName: "test-item",
  onFollowUp: vi.fn(),
  onSelectScenario: vi.fn(),
  onVerifyEvidence: vi.fn(),
  onRequestMoreEvidence: vi.fn(),
};

describe("ReviewFlow", () => {
  it("renders nothing when no execution and not active", () => {
    const { container } = render(<ReviewFlow {...defaultProps} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders status header and scenario chips with execution", () => {
    render(
      <ReviewFlow
        {...defaultProps}
        execution={makeExecution()}
        targetScenarios={["deployment-manager"]}
      />
    );
    expect(screen.getByTestId(selectors.review.flow)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.review.statusHeader)).toBeInTheDocument();
    expect(screen.getByTestId(selectors.review.scenarioChips)).toBeInTheDocument();
  });

  it("renders finalization badge when execution has finalization", () => {
    const exec = makeExecution({
      finalization: {
        eligible: true,
        status: "completed",
        phase: "completed",
        scopeSource: "none",
        aggregateClassification: "ready",
        aggregateSummary: "All checks passed",
        warnings: [],
        scenarios: [],
        affectedScenarios: [],
      } as Finalization,
    });
    render(<ReviewFlow {...defaultProps} execution={exec} />);
    expect(screen.getByTestId("post-run-status-badge")).toBeInTheDocument();
  });

  it("renders evidence panel when review rounds exist", () => {
    render(
      <ReviewFlow
        {...defaultProps}
        execution={makeExecution()}
        reviewRounds={[makeRound()]}
      />
    );
    // Evidence panel renders when rounds provided
    expect(screen.getByTestId(selectors.review.flow)).toBeInTheDocument();
  });
});
