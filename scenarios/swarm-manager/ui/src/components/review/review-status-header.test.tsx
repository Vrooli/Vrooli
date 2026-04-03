import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ReviewStatusHeader } from "./review-status-header";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord, Finalization } from "../../types";

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

function makeFinalization(classification: string): Finalization {
  return {
    eligible: true,
    status: "completed",
    phase: "completed",
    scopeSource: "none",
    aggregateClassification: classification,
    aggregateSummary: "Test summary",
    warnings: [],
    scenarios: [],
    affectedScenarios: [],
  } as Finalization;
}

const defaultProps = {
  isActive: false,
  isTriggering: false,
  onTriggerReview: vi.fn(),
  onFollowUp: vi.fn(),
};

describe("ReviewStatusHeader", () => {
  it("renders nothing when no execution", () => {
    const { container } = render(<ReviewStatusHeader {...defaultProps} execution={undefined} />);
    expect(container.firstChild).toBeNull();
  });

  it("renders nothing when active", () => {
    const { container } = render(
      <ReviewStatusHeader {...defaultProps} execution={makeExecution()} isActive />
    );
    expect(container.firstChild).toBeNull();
  });

  it("shows Run Review for terminal execution without finalization", () => {
    render(
      <ReviewStatusHeader
        {...defaultProps}
        execution={makeExecution({ status: "completed" })}
      />
    );
    const btn = screen.getByTestId(selectors.review.primaryAction);
    expect(btn).toHaveTextContent("Run Review");
  });

  it("shows Running... when triggering", () => {
    render(
      <ReviewStatusHeader
        {...defaultProps}
        execution={makeExecution({ status: "completed" })}
        isTriggering
      />
    );
    const btn = screen.getByTestId(selectors.review.primaryAction);
    expect(btn).toHaveTextContent("Running...");
    expect(btn).toBeDisabled();
  });

  it("shows Fix Issues when finalization needs_work", () => {
    const onFollowUp = vi.fn();
    const exec = makeExecution({
      status: "completed",
      finalization: makeFinalization("needs_work"),
    });
    render(
      <ReviewStatusHeader {...defaultProps} execution={exec} onFollowUp={onFollowUp} />
    );
    const btn = screen.getByTestId(selectors.review.primaryAction);
    expect(btn).toHaveTextContent("Fix Issues");
    fireEvent.click(btn);
    expect(onFollowUp).toHaveBeenCalledWith(exec);
  });

  it("shows Follow Up when finalization is ready", () => {
    const onFollowUp = vi.fn();
    const exec = makeExecution({
      status: "completed",
      finalization: makeFinalization("ready"),
    });
    render(
      <ReviewStatusHeader {...defaultProps} execution={exec} onFollowUp={onFollowUp} />
    );
    const btn = screen.getByTestId(selectors.review.primaryAction);
    expect(btn).toHaveTextContent("Follow Up");
    fireEvent.click(btn);
    expect(onFollowUp).toHaveBeenCalledWith(exec);
  });

  it("shows Re-run link when finalization exists", () => {
    const exec = makeExecution({
      status: "completed",
      finalization: makeFinalization("ready"),
    });
    render(<ReviewStatusHeader {...defaultProps} execution={exec} />);
    expect(screen.getByTestId(selectors.review.rerunAction)).toBeInTheDocument();
  });

  it("hides Re-run link when no finalization", () => {
    render(<ReviewStatusHeader {...defaultProps} execution={makeExecution()} />);
    expect(screen.queryByTestId(selectors.review.rerunAction)).toBeNull();
  });

  it("shows failure reason when present", () => {
    const exec = makeExecution({ status: "failed", failureReason: "OOM killed" });
    render(<ReviewStatusHeader {...defaultProps} execution={exec} />);
    expect(screen.getByText("OOM killed")).toBeInTheDocument();
  });
});
