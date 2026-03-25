import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { FollowUpDialog } from "./follow-up-dialog";
import type { ExecutionRecord, ReviewResult } from "../../types";
import { selectors } from "../../consts/selectors";

vi.mock("../../services", () => ({
  executionService: {
    followUp: vi.fn(),
  },
}));

// eslint-disable-next-line @typescript-eslint/consistent-type-imports
const { executionService } = await import("../../services") as unknown as {
  executionService: { followUp: ReturnType<typeof vi.fn> };
};

function makeExecution(overrides?: Partial<ExecutionRecord>): ExecutionRecord {
  return {
    executionId: "exec-1",
    backlogKind: "fix",
    backlogName: "test-item",
    status: "completed",
    mode: "yolo",
    createdAt: "2026-03-24T00:00:00Z",
    updatedAt: "2026-03-24T00:00:00Z",
    ...overrides,
  };
}

function makeReviewResult(overrides?: Partial<ReviewResult>): ReviewResult {
  return {
    jobId: "job-1",
    classification: "needs_work",
    dimensions: [
      { name: "Tests", status: "red", details: "No tests found" },
      { name: "Linting", status: "green" },
    ],
    summary: "Work needs fixes",
    reviewedAt: "2026-03-24T01:00:00Z",
    ...overrides,
  };
}

describe("FollowUpDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders dialog with type options; fixup type hidden when execution has no reviewResult", () => {
    render(
      <FollowUpDialog
        isOpen={true}
        onClose={vi.fn()}
        execution={makeExecution()}
      />,
    );

    // Dialog should be open
    expect(screen.getByTestId(selectors.followUp.dialog)).toBeInTheDocument();

    // "General Follow-up" and "Custom" should be visible
    expect(screen.getByText("General Follow-up")).toBeInTheDocument();
    expect(screen.getByText("Custom")).toBeInTheDocument();

    // "Fix Review Issues" should NOT be visible (no reviewResult)
    expect(screen.queryByText("Fix Review Issues")).not.toBeInTheDocument();
  });

  it("fixup type shown and pre-selected when execution has reviewResult with classification needs_work", () => {
    render(
      <FollowUpDialog
        isOpen={true}
        onClose={vi.fn()}
        execution={makeExecution({ reviewResult: makeReviewResult() })}
      />,
    );

    // "Fix Review Issues" should be visible
    expect(screen.getByText("Fix Review Issues")).toBeInTheDocument();

    // The fixup button should be selected (cyan border indicates selection)
    const fixupButton = screen.getByTestId(selectors.followUp.typeFixup);
    expect(fixupButton).toHaveClass("border-cyan-500");

    // Review summary panel should be visible
    expect(screen.getByTestId(selectors.followUp.reviewSummary)).toBeInTheDocument();
    expect(screen.getByText("Tests")).toBeInTheDocument();
  });

  it("run mode toggle: Continue Run disabled when execution has no runId", () => {
    render(
      <FollowUpDialog
        isOpen={true}
        onClose={vi.fn()}
        execution={makeExecution({ runId: undefined })}
      />,
    );

    const continueButton = screen.getByTestId(selectors.followUp.runModeContinue);
    expect(continueButton).toBeDisabled();

    // "New Run" should be selected by default when no runId
    const newButton = screen.getByTestId(selectors.followUp.runModeNew);
    expect(newButton).toHaveClass("border-cyan-500");
  });

  it("Continue Run enabled when execution has runId", () => {
    render(
      <FollowUpDialog
        isOpen={true}
        onClose={vi.fn()}
        execution={makeExecution({ runId: "run-123" })}
      />,
    );

    const continueButton = screen.getByTestId(selectors.followUp.runModeContinue);
    expect(continueButton).not.toBeDisabled();
    // Continue should be selected by default when runId is present
    expect(continueButton).toHaveClass("border-cyan-500");
  });

  it("submit calls executionService.followUp with correct parameters", async () => {
    const newExec = makeExecution({ executionId: "exec-2", status: "pending" });
    executionService.followUp.mockResolvedValue(newExec);

    const onSuccess = vi.fn();
    render(
      <FollowUpDialog
        isOpen={true}
        onClose={vi.fn()}
        execution={makeExecution({ runId: "run-123" })}
        onSuccess={onSuccess}
      />,
    );

    // Click submit
    fireEvent.click(screen.getByTestId(selectors.followUp.submitButton));

    await waitFor(() => {
      expect(executionService.followUp).toHaveBeenCalledWith("exec-1", {
        followUpType: "followup",
        context: undefined,
        runMode: "continue",
      });
    });
  });

  it("dialog closes on success and calls onSuccess", async () => {
    const newExec = makeExecution({ executionId: "exec-2", status: "pending" });
    executionService.followUp.mockResolvedValue(newExec);

    const onClose = vi.fn();
    const onSuccess = vi.fn();
    render(
      <FollowUpDialog
        isOpen={true}
        onClose={onClose}
        execution={makeExecution()}
        onSuccess={onSuccess}
      />,
    );

    fireEvent.click(screen.getByTestId(selectors.followUp.submitButton));

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith(newExec);
      expect(onClose).toHaveBeenCalledOnce();
    });
  });

  it("error state renders on API failure", async () => {
    executionService.followUp.mockRejectedValue(new Error("Server error"));

    render(
      <FollowUpDialog
        isOpen={true}
        onClose={vi.fn()}
        execution={makeExecution()}
      />,
    );

    fireEvent.click(screen.getByTestId(selectors.followUp.submitButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.followUp.error)).toBeInTheDocument();
      expect(screen.getByText("Server error")).toBeInTheDocument();
    });
  });
});
