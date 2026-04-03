import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { FollowUpSheet } from "./follow-up-sheet";
import { selectors } from "../../consts/selectors";
import type { ExecutionRecord, Finalization } from "../../types";
import type { ReviewRound } from "../../services/review-service";

vi.mock("../../services", () => ({
  executionService: {
    followUp: vi.fn(),
  },
}));

vi.mock("../ui/drawer", () => ({
  Drawer: ({ isOpen, children, footer, title, testId }: { isOpen: boolean; children: React.ReactNode; footer?: React.ReactNode; title: string; testId?: string }) =>
    isOpen ? (
      <div data-testid={testId}>
        <div data-testid="drawer-title">{title}</div>
        <div>{children}</div>
        <div data-testid="drawer-footer">{footer}</div>
      </div>
    ) : null,
}));

const { executionService } = await import("../../services");
const mockFollowUp = vi.mocked(executionService.followUp);

function makeExecution(overrides?: Partial<ExecutionRecord>): ExecutionRecord {
  return {
    executionId: "exec-1",
    backlogKind: "task",
    backlogName: "test-item",
    status: "completed",
    mode: "yolo",
    runId: "run-1",
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

function makeRound(overrides?: Partial<ReviewRound>): ReviewRound {
  return {
    round: 1,
    generated_at: new Date().toISOString(),
    execution_id: "exec-1",
    status: "complete",
    classification: "needs_work",
    agent_assessment: "Tests failed",
    evidence: [{ id: "ev-1", type: "screenshot", title: "Screenshot", description: "", capture_path: "", verified: false }],
    ...overrides,
  } as ReviewRound;
}

const defaultProps = {
  isOpen: true,
  onClose: vi.fn(),
  execution: makeExecution(),
  reviewRounds: [] as ReviewRound[],
  onSuccess: vi.fn(),
};

beforeEach(() => {
  vi.clearAllMocks();
});

describe("FollowUpSheet", () => {
  it("renders when open", () => {
    render(<FollowUpSheet {...defaultProps} />);
    expect(screen.getByTestId(selectors.review.followUpSheet)).toBeInTheDocument();
  });

  it("does not render when closed", () => {
    render(<FollowUpSheet {...defaultProps} isOpen={false} />);
    expect(screen.queryByTestId(selectors.review.followUpSheet)).toBeNull();
  });

  it("shows fixup type when execution has finalization issues", () => {
    const exec = makeExecution({ finalization: makeFinalization("needs_work") });
    render(<FollowUpSheet {...defaultProps} execution={exec} />);
    expect(screen.getByTestId(selectors.followUp.typeFixup)).toBeInTheDocument();
  });

  it("hides fixup type when no finalization issues", () => {
    render(<FollowUpSheet {...defaultProps} />);
    expect(screen.queryByTestId(selectors.followUp.typeFixup)).toBeNull();
  });

  it("shows evidence context when fixup selected and rounds exist", () => {
    const exec = makeExecution({ finalization: makeFinalization("needs_work") });
    render(<FollowUpSheet {...defaultProps} execution={exec} reviewRounds={[makeRound()]} />);
    expect(screen.getByTestId(selectors.review.evidenceContextSummary)).toBeInTheDocument();
  });

  it("shows evidence context for general follow-up when rounds exist", () => {
    render(<FollowUpSheet {...defaultProps} reviewRounds={[makeRound()]} />);
    // Evidence context is always shown when rounds exist
    expect(screen.getByTestId(selectors.review.evidenceContextSummary)).toBeInTheDocument();
  });

  it("submits follow-up and calls onSuccess", async () => {
    const newExec = makeExecution({ executionId: "exec-2" });
    mockFollowUp.mockResolvedValue(newExec);
    const onSuccess = vi.fn();
    const onClose = vi.fn();

    render(<FollowUpSheet {...defaultProps} onClose={onClose} onSuccess={onSuccess} />);
    fireEvent.click(screen.getByTestId(selectors.followUp.submitButton));

    await waitFor(() => {
      expect(mockFollowUp).toHaveBeenCalledWith("exec-1", expect.objectContaining({
        followUpType: "followup",
        runMode: "continue",
      }));
      expect(onSuccess).toHaveBeenCalledWith(newExec);
      expect(onClose).toHaveBeenCalled();
    });
  });

  it("shows error on service failure", async () => {
    mockFollowUp.mockRejectedValue(new Error("Server error"));
    render(<FollowUpSheet {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.followUp.submitButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.followUp.error)).toHaveTextContent("Server error");
    });
  });

  it("shows session expired message on 409", async () => {
    mockFollowUp.mockRejectedValue(new Error("409 Conflict"));
    render(<FollowUpSheet {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.followUp.submitButton));

    await waitFor(() => {
      expect(screen.getByTestId(selectors.followUp.error)).toHaveTextContent("Agent session has expired");
    });
  });

  it("disables submit when custom type has no context", () => {
    render(<FollowUpSheet {...defaultProps} />);
    fireEvent.click(screen.getByTestId(selectors.followUp.typeCustom));
    expect(screen.getByTestId(selectors.followUp.submitButton)).toBeDisabled();
  });

  it("disables Continue Run when no runId", () => {
    const exec = makeExecution({ runId: undefined });
    render(<FollowUpSheet {...defaultProps} execution={exec} />);
    expect(screen.getByTestId(selectors.followUp.runModeContinue)).toBeDisabled();
  });
});
