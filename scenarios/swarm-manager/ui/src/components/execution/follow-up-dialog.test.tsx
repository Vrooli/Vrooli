import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor, within } from "@testing-library/react";
import { FollowUpDialog } from "./follow-up-dialog";
import type { ExecutionRecord, Finalization, ReviewResult } from "../../types";
import { selectors } from "../../consts/selectors";

vi.mock("../../services", () => ({
  executionService: {
    followUp: vi.fn(),
  },
}));

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

function makeFinalization(overrides?: Partial<Finalization>): Finalization {
  return {
    eligible: true,
    status: "completed",
    phase: "completed",
    scopeSource: "sandbox_diff",
    warnings: [],
    affectedScenarios: ["swarm-manager"],
    aggregateClassification: "needs_work",
    aggregateSummary: "Work needs fixes",
    scenarios: [
      {
        scenarioName: "swarm-manager",
        changedPaths: ["scenarios/swarm-manager/ui/src/components/execution/follow-up-dialog.tsx"],
        restart: {
          status: "completed",
          attempts: 1,
          startedAt: "2026-03-24T00:30:00Z",
          finishedAt: "2026-03-24T00:31:00Z",
        },
        health: {
          status: "completed",
          scenarioStatus: "running",
          healthStatus: "healthy",
          schemaValid: true,
          checkedAt: "2026-03-24T00:32:00Z",
        },
        review: {
          status: "completed",
          result: makeReviewResult(),
        },
      },
    ],
    ...overrides,
  };
}

describe("FollowUpDialog", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders dialog with type options; fixup type hidden when execution has no finalization issues", () => {
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

    // "Fix Review Issues" should NOT be visible without actionable post-run findings.
    expect(screen.queryByText("Fix Review Issues")).not.toBeInTheDocument();
  });

  it("fixup type shown and pre-selected when execution has actionable finalization findings", () => {
    render(
      <FollowUpDialog
        isOpen={true}
        onClose={vi.fn()}
        execution={makeExecution({ finalization: makeFinalization() })}
      />,
    );

    // "Fix Review Issues" should be visible
    expect(screen.getByText("Fix Review Issues")).toBeInTheDocument();

    // The fixup button should be selected (cyan border indicates selection)
    const fixupButton = screen.getByTestId(selectors.followUp.typeFixup);
    expect(fixupButton).toHaveClass("border-cyan-500");

    // Review summary panel should be visible
    const reviewSummary = screen.getByTestId(selectors.followUp.reviewSummary);
    expect(reviewSummary).toBeInTheDocument();
    expect(within(reviewSummary).getByText(/swarm-manager Tests \(red\): No tests found/)).toBeInTheDocument();
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
