import { describe, it, expect, vi } from "vitest";
import { render, screen, fireEvent } from "@testing-library/react";
import { ActivityTimeline } from "./ActivityTimeline";
import type { TimelineEntry } from "../../hooks/useActivityTimeline";
import type { ExecutionRecord, AgentActivity } from "../../types";

// Minimal factories
function makeExecEntry(overrides?: Partial<ExecutionRecord>): TimelineEntry {
  const exec: ExecutionRecord = {
    executionId: "e1",
    status: "completed",
    mode: "yolo",
    backlogKind: "execute",
    operation: "generator",
    createdAt: "2026-03-30T10:00:00Z",
    startedAt: "2026-03-30T10:00:05Z",
    finishedAt: "2026-03-30T10:15:00Z",
    ...overrides,
  } as ExecutionRecord;
  return {
    id: exec.executionId,
    type: "execution",
    timestamp: exec.createdAt,
    execution: exec,
  };
}

function makeActivityEntry(overrides?: Partial<AgentActivity>): TimelineEntry {
  const act: AgentActivity = {
    activityId: "a1",
    ownerType: "backlog",
    ownerKind: "execute",
    ownerName: "test",
    purpose: "workshop",
    interactionType: "spawn",
    status: "complete",
    requestedAt: "2026-03-30T10:01:00Z",
    ...overrides,
  } as AgentActivity;
  return {
    id: act.activityId,
    type: "activity",
    timestamp: act.requestedAt,
    activity: act,
  };
}

const defaultProps = {
  entries: [] as TimelineEntry[],
  isLoading: false,
  error: null,
  onViewExecution: vi.fn(),
  onStopRun: vi.fn(),
  onFollowUp: vi.fn(),
  latestAgentActivity: undefined,
  agentRunIsActive: false,
};

describe("ActivityTimeline", () => {
  it("shows empty state when no entries", () => {
    render(<ActivityTimeline {...defaultProps} />);
    expect(screen.getByText("No activity history yet.")).toBeInTheDocument();
  });

  it("shows loading state", () => {
    render(<ActivityTimeline {...defaultProps} isLoading={true} />);
    expect(screen.getByText(/Loading activity history/)).toBeInTheDocument();
  });

  it("shows error state", () => {
    render(
      <ActivityTimeline {...defaultProps} error={new Error("Network failure")} />,
    );
    expect(screen.getByText(/Network failure/)).toBeInTheDocument();
  });

  it("renders an execution entry with status and operation", () => {
    const entry = makeExecEntry({ status: "completed", operation: "generator" });
    render(<ActivityTimeline {...defaultProps} entries={[entry]} />);
    expect(screen.getByText("Completed")).toBeInTheDocument();
    expect(screen.getByText("generator")).toBeInTheDocument();
  });

  it("renders a standalone activity entry with purpose badge", () => {
    const entry = makeActivityEntry({ purpose: "workshop" });
    render(<ActivityTimeline {...defaultProps} entries={[entry]} />);
    expect(screen.getByText("Workshop")).toBeInTheDocument();
    expect(screen.getByText("spawn")).toBeInTheDocument();
  });

  it("expands executions by default and can collapse them", () => {
    const entry = makeExecEntry({
      executionId: "exec-123",
      failureReason: "Build failed",
    });
    render(<ActivityTimeline {...defaultProps} entries={[entry]} />);

    // Details visible by default (expanded)
    expect(screen.getByText("Build failed")).toBeInTheDocument();
    expect(screen.getByText(/ID: exec-123/)).toBeInTheDocument();

    // Click to collapse
    fireEvent.click(screen.getByText("Completed"));
    expect(screen.queryByText("Build failed")).not.toBeInTheDocument();
  });

  it("shows View and Follow Up buttons on expanded completed execution", () => {
    const entry = makeExecEntry({ status: "completed" });
    render(<ActivityTimeline {...defaultProps} entries={[entry]} />);

    expect(screen.getByText("View")).toBeInTheDocument();
    expect(screen.getByText("Follow Up")).toBeInTheDocument();
  });

  it("calls onViewExecution when View is clicked", () => {
    const onView = vi.fn();
    const entry = makeExecEntry();
    render(<ActivityTimeline {...defaultProps} entries={[entry]} onViewExecution={onView} />);

    fireEvent.click(screen.getByText("View"));
    expect(onView).toHaveBeenCalledWith(entry.execution);
  });

  it("calls onFollowUp when Follow Up is clicked", () => {
    const onFollowUp = vi.fn();
    const entry = makeExecEntry({ status: "completed" });
    render(<ActivityTimeline {...defaultProps} entries={[entry]} onFollowUp={onFollowUp} />);

    fireEvent.click(screen.getByText("Follow Up"));
    expect(onFollowUp).toHaveBeenCalledWith(entry.execution);
  });

  it("shows Run link on execution when agentManagerUiUrl and runId are present", () => {
    const entry = makeExecEntry({ runId: "run-abc" });
    render(
      <ActivityTimeline
        {...defaultProps}
        entries={[entry]}
        agentManagerUiUrl="https://agent.test"
      />,
    );
    const runLink = screen.getByRole("link", { name: /Run/ });
    expect(runLink).toHaveAttribute("href", "https://agent.test/runs/run-abc");
    expect(runLink).toHaveAttribute("target", "_blank");
  });

  it("does not show Run link on execution when agentManagerUiUrl is missing", () => {
    const entry = makeExecEntry({ runId: "run-abc" });
    render(<ActivityTimeline {...defaultProps} entries={[entry]} />);
    expect(screen.queryByRole("link", { name: /Run/ })).not.toBeInTheDocument();
  });

  it("does not show Run link on execution when runId is missing", () => {
    const entry = makeExecEntry({ runId: undefined });
    render(
      <ActivityTimeline
        {...defaultProps}
        entries={[entry]}
        agentManagerUiUrl="https://agent.test"
      />,
    );
    expect(screen.queryByRole("link", { name: /Run/ })).not.toBeInTheDocument();
  });

  it("shows Run link on expanded standalone activity when agentManagerUiUrl and runId are present", () => {
    const entry = makeActivityEntry({ runId: "run-xyz" });
    render(
      <ActivityTimeline
        {...defaultProps}
        entries={[entry]}
        agentManagerUiUrl="https://agent.test"
      />,
    );
    // Activity must be expanded to show the Run link
    fireEvent.click(screen.getByText("Workshop"));
    const runLink = screen.getByRole("link", { name: /Run/ });
    expect(runLink).toHaveAttribute("href", "https://agent.test/runs/run-xyz");
    expect(runLink).toHaveAttribute("target", "_blank");
  });

  it("does not show Run link on expanded activity when runId is missing", () => {
    const entry = makeActivityEntry({ runId: undefined });
    render(
      <ActivityTimeline
        {...defaultProps}
        entries={[entry]}
        agentManagerUiUrl="https://agent.test"
      />,
    );
    fireEvent.click(screen.getByText("Workshop"));
    expect(screen.queryByRole("link", { name: /Run/ })).not.toBeInTheDocument();
  });

  it("renders nested child activities when execution is expanded", () => {
    const childAct: AgentActivity = {
      activityId: "child-a1",
      ownerType: "backlog",
      ownerKind: "execute",
      ownerName: "test",
      purpose: "finalize",
      interactionType: "spawn",
      status: "complete",
      requestedAt: "2026-03-30T10:10:00Z",
    } as AgentActivity;
    const entry = makeExecEntry();
    entry.childActivities = [childAct];

    render(<ActivityTimeline {...defaultProps} entries={[entry]} />);

    // Expanded by default — child activities visible immediately
    expect(screen.getByText("Finalize")).toBeInTheDocument();
    expect(screen.getByText("Agent Activities")).toBeInTheDocument();
  });
});
