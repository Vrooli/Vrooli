import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ExecutionCard } from "./execution-card";
import type { ExecutionRecord, Finalization, ReviewResult } from "../../types";

const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "idea",
  backlogName: "test-feature",
  status: "completed",
  mode: "manual",
  createdAt: "2026-03-20T00:00:00Z",
  updatedAt: "2026-03-20T01:00:00Z",
  ...overrides,
});

const makeReviewResult = (overrides?: Partial<ReviewResult>): ReviewResult => ({
  jobId: "job-1",
  classification: "needs_work",
  dimensions: [],
  summary: "Fix the tests",
  reviewedAt: "2026-03-20T03:00:00Z",
  ...overrides,
});

const makeFinalization = (overrides?: Partial<Finalization>): Finalization => ({
  eligible: true,
  status: "completed",
  phase: "completed",
  scopeSource: "sandbox_diff",
  warnings: [],
  affectedScenarios: ["swarm-manager"],
  aggregateClassification: "needs_work",
  aggregateSummary: "Fix the tests",
  scenarios: [
    {
      scenarioName: "swarm-manager",
      changedPaths: ["scenarios/swarm-manager/ui/src/components/execution/execution-card.tsx"],
      restart: {
        status: "completed",
        attempts: 1,
        startedAt: "2026-03-20T02:30:00Z",
        finishedAt: "2026-03-20T02:31:00Z",
      },
      health: {
        status: "completed",
        scenarioStatus: "running",
        healthStatus: "healthy",
        schemaValid: true,
        checkedAt: "2026-03-20T02:32:00Z",
      },
      review: {
        status: "completed",
        result: makeReviewResult(),
      },
    },
  ],
  ...overrides,
});

const noopHandlers = {
  onStart: vi.fn(),
  onCancel: vi.fn(),
  onRetry: vi.fn(),
  onViewTrace: vi.fn(),
  onViewBacklog: vi.fn(),
};

describe("ExecutionCard", () => {
  it("renders status, title, and mode", () => {
    render(
      <ExecutionCard
        item={makeExecution()}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText("Completed")).toBeInTheDocument();
    expect(screen.getByText(/Idea: test-feature/)).toBeInTheDocument();
    expect(screen.getByText("Manual")).toBeInTheDocument();
  });

  it("title is clickable and fires onViewBacklog", () => {
    const onViewBacklog = vi.fn();
    render(
      <ExecutionCard
        item={makeExecution()}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        onViewBacklog={onViewBacklog}
      />,
    );

    fireEvent.click(screen.getByTestId("execution-backlog-link"));
    expect(onViewBacklog).toHaveBeenCalledWith("idea", "test-feature");
  });

  it("shows operation badge when operation is present", () => {
    render(
      <ExecutionCard
        item={makeExecution({ operation: "generator" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText("generator")).toBeInTheDocument();
  });

  it("shows startedBy in metadata", () => {
    render(
      <ExecutionCard
        item={makeExecution({ startedBy: "swarm-manager-ui" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText("swarm-manager-ui")).toBeInTheDocument();
  });

  it("shows timestamps", () => {
    render(
      <ExecutionCard
        item={makeExecution()}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText(/Updated/)).toBeInTheDocument();
    expect(screen.getByText(/Created/)).toBeInTheDocument();
  });

  it("shows failure reason when present", () => {
    render(
      <ExecutionCard
        item={makeExecution({ status: "failed", failureReason: "Agent crashed" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={true}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText("Agent crashed")).toBeInTheDocument();
  });

  it("shows Start button when canStart is true", () => {
    const onStart = vi.fn();
    render(
      <ExecutionCard
        item={makeExecution({ status: "pending" })}
        isBusy={false}
        canStart={true}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        onStart={onStart}
      />,
    );

    fireEvent.click(screen.getByText("Start"));
    expect(onStart).toHaveBeenCalledWith("exec-1");
  });

  it("shows Cancel button when canCancel is true", () => {
    const onCancel = vi.fn();
    render(
      <ExecutionCard
        item={makeExecution({ status: "running" })}
        isBusy={false}
        canStart={false}
        canCancel={true}
        canRetry={false}
        {...noopHandlers}
        onCancel={onCancel}
      />,
    );

    fireEvent.click(screen.getByText("Cancel"));
    expect(onCancel).toHaveBeenCalledWith("exec-1");
  });

  it("shows Retry button when canRetry is true", () => {
    const onRetry = vi.fn();
    render(
      <ExecutionCard
        item={makeExecution({ status: "failed" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={true}
        {...noopHandlers}
        onRetry={onRetry}
      />,
    );

    fireEvent.click(screen.getByText("Retry"));
    expect(onRetry).toHaveBeenCalledWith("exec-1");
  });

  it("shows Follow Up button for completed executions when handler provided", () => {
    const onFollowUp = vi.fn();
    render(
      <ExecutionCard
        item={makeExecution({ status: "completed" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        onFollowUp={onFollowUp}
      />,
    );

    fireEvent.click(screen.getByText("Follow Up"));
    expect(onFollowUp).toHaveBeenCalledWith("exec-1");
  });

  it("disables action buttons when isBusy is true", () => {
    render(
      <ExecutionCard
        item={makeExecution({ status: "pending" })}
        isBusy={true}
        canStart={true}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    const btn = screen.getByText("Start").closest("button");
    expect(btn).toBeDisabled();
  });

  it("shows Trace button and fires onViewTrace", () => {
    const onViewTrace = vi.fn();
    render(
      <ExecutionCard
        item={makeExecution()}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        onViewTrace={onViewTrace}
      />,
    );

    fireEvent.click(screen.getByText("Trace"));
    expect(onViewTrace).toHaveBeenCalledWith("exec-1");
  });

  it("shows Run link when runId and agentManagerUiUrl are provided", () => {
    render(
      <ExecutionCard
        item={makeExecution({ runId: "run-123" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        agentManagerUiUrl="https://agent.test"
      />,
    );

    const link = screen.getByText("Run").closest("a");
    expect(link).toHaveAttribute("href", "https://agent.test/runs/run-123");
  });

  it("hides Run link when runId is absent", () => {
    render(
      <ExecutionCard
        item={makeExecution({ runId: undefined })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        agentManagerUiUrl="https://agent.test"
      />,
    );

    // The only "Run" link should not be present as an anchor
    const links = screen.queryAllByText("Run");
    links.forEach((el) => {
      expect(el.closest("a")).toBeNull();
    });
  });

  it("toggles ID details panel", () => {
    render(
      <ExecutionCard
        item={makeExecution({ runId: "run-456", taskId: "task-789" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    expect(screen.queryByText("exec exec-1")).not.toBeInTheDocument();

    fireEvent.click(screen.getByText("IDs"));
    expect(screen.getByText("exec exec-1")).toBeInTheDocument();
    expect(screen.getByText("run run-456")).toBeInTheDocument();
    expect(screen.getByText("task task-789")).toBeInTheDocument();

    fireEvent.click(screen.getByText("IDs"));
    expect(screen.queryByText("exec exec-1")).not.toBeInTheDocument();
  });

  it("shows prompt trace when provided", () => {
    render(
      <ExecutionCard
        item={makeExecution()}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        trace={{ purpose: "Generate code", prompt: "Write tests", used_fallback: false, captured_at: "2026-03-20T02:00:00Z" }}
      />,
    );

    expect(screen.getByText("Generate code")).toBeInTheDocument();
    expect(screen.getByText("Write tests")).toBeInTheDocument();
  });

  it("shows post-run status badge when finalization is present", () => {
    render(
      <ExecutionCard
        item={makeExecution({
          finalization: makeFinalization(),
        })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText("Needs fixup")).toBeInTheDocument();
  });

  it("shows validating indicator when status is validating", () => {
    render(
      <ExecutionCard
        item={makeExecution({ status: "validating" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId("post-run-validating-indicator")).toBeInTheDocument();
  });

  it("shows Run Post-Run Checks button for completed executions without finalization", () => {
    const onTriggerReview = vi.fn();
    render(
      <ExecutionCard
        item={makeExecution({ status: "completed" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        onTriggerReview={onTriggerReview}
      />,
    );

    fireEvent.click(screen.getByTestId("review-trigger-button"));
    expect(onTriggerReview).toHaveBeenCalledWith("exec-1");
  });

  it("does not show the standalone post-run checks button when finalization already exists", () => {
    render(
      <ExecutionCard
        item={makeExecution({
          status: "completed",
          finalization: makeFinalization({
            aggregateClassification: "ready",
            aggregateSummary: "All good",
            scenarios: [
              {
                scenarioName: "swarm-manager",
                changedPaths: [],
                restart: { status: "completed", attempts: 1 },
                health: { status: "completed", schemaValid: true },
                review: {
                  status: "completed",
                  result: makeReviewResult({
                    classification: "ready",
                    summary: "All good",
                  }),
                },
              },
            ],
          }),
        })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
        onTriggerReview={vi.fn()}
      />,
    );

    expect(screen.queryByTestId("review-trigger-button")).not.toBeInTheDocument();
    expect(screen.getByTestId("post-run-status-badge")).toBeInTheDocument();
  });

  it("maps backlog kinds to readable labels", () => {
    render(
      <ExecutionCard
        item={makeExecution({ backlogKind: "research" })}
        isBusy={false}
        canStart={false}
        canCancel={false}
        canRetry={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText(/Research: test-feature/)).toBeInTheDocument();
  });
});
