import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ExecutionReviewTab } from "./execution-review-tab";
import type { ExecutionRecord, Finalization } from "../../types";
import { selectors } from "../../consts/selectors";

// Mock ReviewFlow to isolate tab logic
vi.mock("../review/review-flow", () => ({
  ReviewFlow: () => <div data-testid="mock-review-flow">ReviewFlow</div>,
}));

vi.mock("../../services", () => ({
  executionService: {
    triggerReview: vi.fn().mockResolvedValue({}),
  },
}));

const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "fix",
  backlogName: "test-bug",
  status: "completed",
  mode: "manual",
  createdAt: "2026-03-20T00:00:00Z",
  updatedAt: "2026-03-20T01:00:00Z",
  ...overrides,
} as ExecutionRecord);

const makeFinalization = (overrides?: Partial<Finalization>): Finalization => ({
  eligible: true,
  status: "completed",
  phase: "completed",
  scopeSource: "sandbox_diff",
  warnings: [],
  affectedScenarios: ["swarm-manager"],
  aggregateClassification: "ready",
  scenarios: [],
  ...overrides,
} as Finalization);

const noopHandlers = {
  onFollowUp: vi.fn(),
  onSelectScenario: vi.fn(),
  onVerifyEvidence: vi.fn(),
  onRequestMoreEvidence: vi.fn(),
};

describe("ExecutionReviewTab", () => {
  it("shows empty state when no review data", () => {
    render(
      <ExecutionReviewTab
        execution={makeExecution()}
        reviewRounds={[]}
        isGatheringEvidence={false}
        targetScenarios={[]}
        isActive={false}
        {...noopHandlers}
      />,
    );
    expect(screen.getByTestId(selectors.executionDetails.reviewEmpty)).toBeInTheDocument();
    expect(screen.getByText(/No review data available/)).toBeInTheDocument();
  });

  it("shows pending message when execution is active", () => {
    render(
      <ExecutionReviewTab
        execution={makeExecution({ status: "running" })}
        reviewRounds={[]}
        isGatheringEvidence={false}
        targetScenarios={[]}
        isActive={true}
        {...noopHandlers}
      />,
    );
    expect(screen.getByText(/Review will be available after/)).toBeInTheDocument();
  });

  it("renders ReviewFlow when execution has finalization", () => {
    const exec = makeExecution({ finalization: makeFinalization() });
    render(
      <ExecutionReviewTab
        execution={exec}
        reviewRounds={[]}
        isGatheringEvidence={false}
        targetScenarios={[]}
        isActive={false}
        {...noopHandlers}
      />,
    );
    expect(screen.getByTestId("mock-review-flow")).toBeInTheDocument();
  });

  it("renders ReviewFlow when scenarios exist", () => {
    render(
      <ExecutionReviewTab
        execution={makeExecution()}
        reviewRounds={[]}
        isGatheringEvidence={false}
        targetScenarios={["app-a"]}
        isActive={false}
        {...noopHandlers}
      />,
    );
    expect(screen.getByTestId("mock-review-flow")).toBeInTheDocument();
  });

  it("renders ReviewFlow when gathering evidence", () => {
    render(
      <ExecutionReviewTab
        execution={makeExecution()}
        reviewRounds={[]}
        isGatheringEvidence={true}
        targetScenarios={[]}
        isActive={false}
        {...noopHandlers}
      />,
    );
    expect(screen.getByTestId("mock-review-flow")).toBeInTheDocument();
  });
});
