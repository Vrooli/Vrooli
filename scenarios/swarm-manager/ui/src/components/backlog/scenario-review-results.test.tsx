import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { ScenarioReviewResults } from "./scenario-review-results";
import type { ScenarioReviewResultsProps } from "./scenario-review-results";
import type { ExecutionRecord, Finalization } from "../../types";

vi.mock("../../services", () => ({
  executionService: {
    triggerReview: vi.fn().mockResolvedValue({}),
  },
}));

const makeFinalization = (overrides?: Partial<Finalization>): Finalization => ({
  eligible: true,
  status: "completed",
  phase: "completed",
  scopeSource: "sandbox_diff",
  warnings: [],
  affectedScenarios: ["test-scenario"],
  aggregateClassification: "ready",
  aggregateSummary: "All scenarios are ready.",
  scenarios: [],
  ...overrides,
});

const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "execute",
  backlogName: "test-item",
  status: "completed",
  mode: "yolo",
  createdAt: "2026-03-20T12:00:00Z",
  ...overrides,
} as ExecutionRecord);

const defaultProps: ScenarioReviewResultsProps = {
  latestExecution: undefined,
  targetScenarios: ["scenario-a", "scenario-b"],
  onSelectScenario: vi.fn(),
};

describe("ScenarioReviewResults", () => {
  it("returns null when no scenarios", () => {
    const { container } = render(
      <ScenarioReviewResults {...defaultProps} targetScenarios={[]} />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders scenario chips for each target scenario", () => {
    render(<ScenarioReviewResults {...defaultProps} />);
    expect(screen.getByText("scenario-a")).toBeInTheDocument();
    expect(screen.getByText("scenario-b")).toBeInTheDocument();
  });

  it("clicking scenario chip calls onSelectScenario", async () => {
    const onSelectScenario = vi.fn();
    render(
      <ScenarioReviewResults {...defaultProps} onSelectScenario={onSelectScenario} />,
    );
    await userEvent.click(screen.getByText("scenario-a"));
    expect(onSelectScenario).toHaveBeenCalledWith("scenario-a");
  });

  it("shows empty state when no execution", () => {
    render(<ScenarioReviewResults {...defaultProps} />);
    expect(screen.getByText("Post-run checks will run after execution")).toBeInTheDocument();
  });

  it("shows run checks button when execution has no finalization", () => {
    render(
      <ScenarioReviewResults
        {...defaultProps}
        latestExecution={makeExecution()}
      />,
    );
    expect(screen.getByText("No post-run checks yet")).toBeInTheDocument();
    expect(screen.getByText("Run Post-Run Checks")).toBeInTheDocument();
  });

  it("renders PostRunStatusBadge when finalization exists", () => {
    render(
      <ScenarioReviewResults
        {...defaultProps}
        latestExecution={makeExecution({ finalization: makeFinalization() })}
      />,
    );
    // PostRunStatusBadge renders the classification label
    expect(screen.getByText("Post-run checks passed")).toBeInTheDocument();
  });

  it("renders validating state", () => {
    render(
      <ScenarioReviewResults
        {...defaultProps}
        latestExecution={makeExecution({ status: "validating" })}
      />,
    );
    // PostRunStatusBadge shows phase label for scope_detection
    expect(screen.getByText("Resolving affected scenarios")).toBeInTheDocument();
  });
});
