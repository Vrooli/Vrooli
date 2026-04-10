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

function makeFinalization(classification: string, statusOverride?: string): Finalization {
  return {
    eligible: true,
    status: statusOverride ?? "completed",
    phase: statusOverride === "running" ? "reviewing" : "completed",
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
  isTriggeringEvidence: false,
  isCancelling: false,
  onOpenLaunchSheet: vi.fn(),
  onCancelReview: vi.fn(),
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

  it("shows Review for terminal execution without finalization", () => {
    render(
      <ReviewStatusHeader
        {...defaultProps}
        execution={makeExecution({ status: "completed" })}
      />
    );
    const btn = screen.getByTestId(selectors.review.primaryAction);
    expect(btn).toHaveTextContent("Review");
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

  it("shows Running... when triggering evidence only", () => {
    render(
      <ReviewStatusHeader
        {...defaultProps}
        execution={makeExecution({ status: "completed" })}
        isTriggeringEvidence
      />
    );
    const btn = screen.getByTestId(selectors.review.primaryAction);
    expect(btn).toHaveTextContent("Running...");
    expect(btn).toBeDisabled();
  });

  it("shows Rerun Checks when finalization is complete and ready", () => {
    const exec = makeExecution({
      status: "completed",
      finalization: makeFinalization("ready"),
    });
    render(<ReviewStatusHeader {...defaultProps} execution={exec} />);
    const btn = screen.getByTestId(selectors.review.primaryAction);
    expect(btn).toHaveTextContent("Rerun Checks");
  });

  it("shows Rerun Checks even when finalization needs_work", () => {
    const exec = makeExecution({
      status: "completed",
      finalization: makeFinalization("needs_work"),
    });
    render(<ReviewStatusHeader {...defaultProps} execution={exec} />);
    const btn = screen.getByTestId(selectors.review.primaryAction);
    expect(btn).toHaveTextContent("Rerun Checks");
  });

  it("shows Stop Review when finalization is running", () => {
    const onCancelReview = vi.fn();
    const exec = makeExecution({
      status: "validating",
      finalization: makeFinalization("", "running"),
    });
    render(
      <ReviewStatusHeader {...defaultProps} execution={exec} onCancelReview={onCancelReview} />
    );
    const btn = screen.getByTestId(selectors.review.stopAction);
    expect(btn).toHaveTextContent("Stop Review");
    fireEvent.click(btn);
    expect(onCancelReview).toHaveBeenCalled();
  });

  it("shows Stopping... when cancelling", () => {
    const exec = makeExecution({
      status: "validating",
      finalization: makeFinalization("", "running"),
    });
    render(
      <ReviewStatusHeader {...defaultProps} execution={exec} isCancelling />
    );
    const btn = screen.getByTestId(selectors.review.stopAction);
    expect(btn).toHaveTextContent("Stopping...");
    expect(btn).toBeDisabled();
  });

  it("shows failure reason when present", () => {
    const exec = makeExecution({ status: "failed", failureReason: "OOM killed" });
    render(<ReviewStatusHeader {...defaultProps} execution={exec} />);
    expect(screen.getByText("OOM killed")).toBeInTheDocument();
  });
});
