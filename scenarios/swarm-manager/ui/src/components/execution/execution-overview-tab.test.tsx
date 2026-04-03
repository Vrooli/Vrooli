import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ExecutionOverviewTab } from "./execution-overview-tab";
import { useDetailSelectionStore } from "../../stores/detail-selection-store";
import type { ExecutionRecord, Finalization } from "../../types";
import { selectors } from "../../consts/selectors";

const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "fix",
  backlogName: "test-bug",
  status: "completed",
  mode: "manual",
  createdAt: "2026-03-20T00:00:00Z",
  updatedAt: "2026-03-20T01:00:00Z",
  ...overrides,
});

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
});

const noopHandlers = {
  agentManagerUiUrl: null as string | null,
  onFollowUp: vi.fn(),
  onCancel: vi.fn(),
  onRetry: vi.fn(),
  onRunPostRunChecks: vi.fn(),
};

describe("ExecutionOverviewTab", () => {
  beforeEach(() => {
    useDetailSelectionStore.setState({ selection: null });
  });

  it("renders metadata grid with backlog link", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution()}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText("fix/test-bug")).toBeInTheDocument();
    expect(screen.getByTestId(selectors.executionDetails.overviewMetadata)).toBeInTheDocument();
  });

  it("navigates to backlog when backlog link is clicked", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution()}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    fireEvent.click(screen.getByText("fix/test-bug"));
    const selection = useDetailSelectionStore.getState().selection;
    expect(selection).toMatchObject({ entityType: "backlog", kind: "fix", name: "test-bug" });
  });

  it("shows failure reason when present", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution({ failureReason: "agent-manager run failed" })}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText("agent-manager run failed")).toBeInTheDocument();
  });

  it("hides failure reason when absent", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution()}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    expect(screen.queryByText("Failure Reason")).not.toBeInTheDocument();
  });

  it("shows Follow-up button for terminal executions", () => {
    const handler = vi.fn();
    render(
      <ExecutionOverviewTab
        execution={makeExecution()}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
        onFollowUp={handler}
      />,
    );

    const btn = screen.getByTestId(selectors.executionDetails.followUpButton);
    expect(btn).toBeInTheDocument();
    fireEvent.click(btn);
    expect(handler).toHaveBeenCalled();
  });

  it("shows Cancel button for active executions", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution({ status: "running" })}
        isActive={true}
        isTerminal={false}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId(selectors.executionDetails.cancelButton)).toBeInTheDocument();
  });

  it("shows Retry button only for failed executions", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution({ status: "failed" })}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId(selectors.executionDetails.retryButton)).toBeInTheDocument();
  });

  it("shows Run Post-Run Checks button for terminal without finalization", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution({ status: "completed" })}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId(selectors.executionDetails.runChecksButton)).toBeInTheDocument();
  });

  it("hides Run Post-Run Checks when finalization exists", () => {
    const exec = makeExecution({ finalization: makeFinalization() });
    render(
      <ExecutionOverviewTab
        execution={exec}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={exec}
        {...noopHandlers}
      />,
    );

    expect(screen.queryByTestId(selectors.executionDetails.runChecksButton)).not.toBeInTheDocument();
  });

  it("navigates to parent execution when link is clicked", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution({ parentExecutionId: "parent-exec-1" })}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    const link = screen.getByText("parent-exec-1");
    expect(link).toBeInTheDocument();
    fireEvent.click(link);
    const selection = useDetailSelectionStore.getState().selection;
    expect(selection).toMatchObject({ entityType: "execution", identifier: "parent-exec-1" });
  });

  it("disables buttons when actionBusy", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution({ status: "failed" })}
        isActive={false}
        isTerminal={true}
        actionBusy={true}
        postRunBadgeExecution={null}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId(selectors.executionDetails.followUpButton)).toBeDisabled();
    expect(screen.getByTestId(selectors.executionDetails.retryButton)).toBeDisabled();
  });

  it("shows View Run button when runId and agentManagerUiUrl are present", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution({ runId: "run-abc" })}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
        agentManagerUiUrl="https://agent.example.com"
      />,
    );

    const link = screen.getByTestId(selectors.executionDetails.viewRunButton);
    expect(link).toBeInTheDocument();
    expect(link).toHaveAttribute("href", "https://agent.example.com/runs/run-abc");
    expect(link).toHaveAttribute("target", "_blank");
  });

  it("hides View Run button when agentManagerUiUrl is null", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution({ runId: "run-abc" })}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
        agentManagerUiUrl={null}
      />,
    );

    expect(screen.queryByTestId(selectors.executionDetails.viewRunButton)).not.toBeInTheDocument();
  });

  it("hides View Run button when runId is absent", () => {
    render(
      <ExecutionOverviewTab
        execution={makeExecution()}
        isActive={false}
        isTerminal={true}
        actionBusy={false}
        postRunBadgeExecution={null}
        {...noopHandlers}
        agentManagerUiUrl="https://agent.example.com"
      />,
    );

    expect(screen.queryByTestId(selectors.executionDetails.viewRunButton)).not.toBeInTheDocument();
  });
});
