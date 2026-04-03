import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LatestExecutionSummary } from "./latest-execution-summary";
import type { LatestExecutionSummaryProps } from "./latest-execution-summary";
import type { ExecutionRecord } from "../../types";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";

const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "execute",
  backlogName: "test-item",
  status: "completed",
  mode: "yolo",
  createdAt: "2026-03-20T12:00:00Z",
  ...overrides,
} as ExecutionRecord);

const makeActivity = (overrides?: Partial<AgentActivityRecord>): AgentActivityRecord => ({
  activityId: "act-1",
  runId: "run-1",
  ownerType: "backlog",
  ownerKind: "execute",
  ownerName: "test-item",
  purpose: "workshop",
  interactionType: "spawn",
  status: "running",
  requestedAt: "2026-03-20T12:00:00Z",
  updatedAt: "2026-03-20T12:00:00Z",
  isStopping: false,
  ...overrides,
} as AgentActivityRecord);

const defaultProps: LatestExecutionSummaryProps = {
  latestExecution: undefined,
  agentRunIsActive: false,
  latestAgentActivity: null,
  onStopRun: vi.fn(),
};

describe("LatestExecutionSummary", () => {
  it("renders empty state when no execution", () => {
    render(<LatestExecutionSummary {...defaultProps} />);
    expect(screen.getByText(/no executions yet/i)).toBeInTheDocument();
  });

  it("renders active run with pulse indicator", () => {
    const activity = makeActivity();
    render(
      <LatestExecutionSummary
        {...defaultProps}
        agentRunIsActive
        latestAgentActivity={activity}
      />,
    );
    expect(screen.getByText("running")).toBeInTheDocument();
    expect(screen.getByText("workshop")).toBeInTheDocument();
  });

  it("renders stop button that calls onStopRun", async () => {
    const onStopRun = vi.fn();
    const activity = makeActivity({ runId: "run-42" });
    render(
      <LatestExecutionSummary
        {...defaultProps}
        agentRunIsActive
        latestAgentActivity={activity}
        onStopRun={onStopRun}
      />,
    );
    await userEvent.click(screen.getByText("Stop"));
    expect(onStopRun).toHaveBeenCalledWith("run-42");
  });

  it("shows Stopping... when isStopping is true", () => {
    const activity = makeActivity({ isStopping: true });
    render(
      <LatestExecutionSummary
        {...defaultProps}
        agentRunIsActive
        latestAgentActivity={activity}
      />,
    );
    expect(screen.getByText("Stopping...")).toBeInTheDocument();
  });

  it("renders null for completed execution (status handled by ReviewStatusHeader)", () => {
    const { container } = render(
      <LatestExecutionSummary
        {...defaultProps}
        latestExecution={makeExecution({ status: "completed" })}
      />,
    );
    expect(container.firstChild).toBeNull();
  });

  it("renders null for failed execution (status handled by ReviewStatusHeader)", () => {
    const { container } = render(
      <LatestExecutionSummary
        {...defaultProps}
        latestExecution={makeExecution({ status: "failed" })}
      />,
    );
    expect(container.firstChild).toBeNull();
  });
});
