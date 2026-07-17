import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { LatestExecutionSummary } from "./latest-execution-summary";
import type { LatestExecutionSummaryProps } from "./latest-execution-summary";
import type { ExecutionRecord } from "../../types";
import type { WorkflowExecutionSummary } from "../../types/agent-operations";
import type { OperationProvenanceData } from "../../lib";
import type { AgentActivityRecord } from "../../stores/agent-activities-store";
import { installMatchMediaMock } from "../../test-utils/browser";

installMatchMediaMock();

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
  agentRunIsBusy: false,
  latestAgentActivity: null,
  agentManagerUiUrl: null,
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
        agentRunIsBusy
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
        agentRunIsBusy
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
        agentRunIsBusy
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

  it("shows a View Run link for active runs when the agent-manager URL is available", () => {
    const activity = makeActivity({ runId: "run-99" });
    render(
      <LatestExecutionSummary
        {...defaultProps}
        agentRunIsBusy
        latestAgentActivity={activity}
        agentManagerUiUrl="https://agent.test"
      />,
    );
    const runLink = screen.getByRole("link", { name: /view run/i });
    expect(runLink).toHaveAttribute("href", "https://agent.test/runs/run-99");
    expect(runLink).toHaveAttribute("target", "_blank");
  });

  describe("mixed canonical/legacy honesty (partial migration)", () => {
    const makeSummary = (
      overrides?: Partial<WorkflowExecutionSummary>,
    ): WorkflowExecutionSummary => ({
      executionId: "exec-c1",
      operation: "backlog.execute",
      operationVersion: "1.0.0",
      mode: "backlog-fixup",
      modeRevision: "rev-3",
      bindingLayer: "system-default",
      compiledModeDigest: "sha256:aaaa",
      promptCatalogDigest: "sha256:bbbb",
      callerInputDigest: "sha256:cccc",
      outcome: "success",
      reproducible: true,
      recordedAt: "2026-07-01T00:00:00Z",
      legacyImport: false,
      ...overrides,
    });

    const runProvenance: OperationProvenanceData = {
      source: "canonical",
      operation: "backlog.execute",
      operationVersion: "1.0.0",
      executionId: "exec-c2",
      runId: "run-1",
      mode: "backlog-fixup",
      modeRevision: "rev-3",
      bindingLayer: "system-default",
      bindingOwnerKind: "system",
      bindingOwnerId: "",
      recordedAt: "2026-07-01T00:00:00Z",
    };

    it("labels a canonical-covered active run with the provenance badge, not the legacy badge", () => {
      render(
        <LatestExecutionSummary
          {...defaultProps}
          agentRunIsBusy
          latestAgentActivity={makeActivity()}
          runProvenance={runProvenance}
        />,
      );
      expect(
        screen.getByRole("button", { name: "Operation provenance" }),
      ).toBeInTheDocument();
      expect(screen.queryByText("legacy record")).not.toBeInTheDocument();
    });

    it("marks an uncovered active run as a legacy record while canonical history coexists", () => {
      // Workflow document exists but has no operation matching this run
      // (e.g. found=true with zero operations): both sources render, each
      // honestly labeled.
      render(
        <LatestExecutionSummary
          {...defaultProps}
          agentRunIsBusy
          latestAgentActivity={makeActivity()}
          runProvenance={null}
          canonicalHistory={[makeSummary()]}
        />,
      );
      expect(screen.getByText("legacy record")).toBeInTheDocument();
      const entries = screen.getAllByTestId("canonical-operation-history-entry");
      expect(entries).toHaveLength(1);
      expect(entries[0]).toHaveTextContent("backlog.execute@1.0.0");
      expect(entries[0]).toHaveTextContent("verified");
    });

    it("flags digest drift on canonical history entries", () => {
      render(
        <LatestExecutionSummary
          {...defaultProps}
          latestExecution={makeExecution({ status: "completed" })}
          canonicalHistory={[makeSummary({ reproducible: false })]}
        />,
      );
      const entry = screen.getByTestId("canonical-operation-history-entry");
      expect(entry).toHaveTextContent("drift");
      expect(entry).not.toHaveTextContent("verified");
    });

    it("renders the canonical history for completed items (inspectability after the run)", () => {
      render(
        <LatestExecutionSummary
          {...defaultProps}
          latestExecution={makeExecution({ status: "completed" })}
          canonicalHistory={[makeSummary(), makeSummary({ executionId: "exec-c0" })]}
        />,
      );
      expect(screen.getAllByTestId("canonical-operation-history-entry")).toHaveLength(2);
    });

    it("keeps the plain empty state when the canonical history is empty", () => {
      render(<LatestExecutionSummary {...defaultProps} canonicalHistory={[]} />);
      expect(screen.getByText(/no executions yet/i)).toBeInTheDocument();
      expect(screen.queryByTestId("canonical-operation-history")).not.toBeInTheDocument();
    });
  });

  it("does not render the live-run summary for needs_review", () => {
    const { container } = render(
      <LatestExecutionSummary
        {...defaultProps}
        latestExecution={makeExecution({ status: "needs_review", runId: "run-review" })}
        latestAgentActivity={makeActivity({ status: "needs_review", runId: "run-review" })}
      />,
    );
    expect(container.firstChild).toBeNull();
    expect(screen.queryByText("Stop")).not.toBeInTheDocument();
  });
});
