import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { ExecutionReviewTab } from "./execution-review-tab";
import type { ExecutionRecord, Finalization } from "../../types";
import type { ReviewRound } from "../../services/review-service";
import { selectors } from "../../consts/selectors";

// Mock child components to isolate review tab logic
vi.mock("./post-run-status-badge", () => ({
  PostRunStatusBadge: ({ execution }: { execution: ExecutionRecord }) => (
    <div data-testid="mock-post-run-badge">{execution.finalization?.aggregateClassification}</div>
  ),
}));

vi.mock("../backlog/scenario-review-results", () => ({
  ScenarioReviewResults: ({ targetScenarios }: { targetScenarios: string[] }) => (
    <div data-testid="mock-scenario-reviews">{targetScenarios.join(", ")}</div>
  ),
}));

vi.mock("../backlog/evidence-panel", () => ({
  EvidencePanel: ({ rounds }: { rounds: ReviewRound[] }) => (
    <div data-testid="mock-evidence-panel">{rounds.length} rounds</div>
  ),
}));

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
  onSelectScenario: vi.fn(),
  onVerifyEvidence: vi.fn(),
  onRequestMoreEvidence: vi.fn(),
  onRunPostRunChecks: vi.fn(),
};

describe("ExecutionReviewTab", () => {
  it("shows empty state when no review data", () => {
    render(
      <ExecutionReviewTab
        execution={makeExecution()}
        reviewRounds={[]}
        isGatheringEvidence={false}
        targetScenarios={[]}
        postRunBadgeExecution={null}
        isActive={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId(selectors.executionDetails.reviewEmpty)).toBeInTheDocument();
    expect(screen.getByText(/No review data available/)).toBeInTheDocument();
  });

  it("shows pending message when execution is active", () => {
    render(
      <ExecutionReviewTab
        execution={makeExecution({ status: "running" })}
        reviewRounds={[]}
        isGatheringEvidence={false}
        targetScenarios={[]}
        postRunBadgeExecution={null}
        isActive={true}
        {...noopHandlers}
      />,
    );

    expect(screen.getByText(/Review will be available after/)).toBeInTheDocument();
  });

  it("renders PostRunStatusBadge when postRunBadgeExecution exists", () => {
    const exec = makeExecution({ finalization: makeFinalization() });
    render(
      <ExecutionReviewTab
        execution={exec}
        reviewRounds={[]}
        isGatheringEvidence={false}
        targetScenarios={[]}
        postRunBadgeExecution={exec}
        isActive={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId("mock-post-run-badge")).toBeInTheDocument();
    expect(screen.getByText("ready")).toBeInTheDocument();
  });

  it("renders ScenarioReviewResults when scenarios exist", () => {
    const exec = makeExecution({ finalization: makeFinalization() });
    render(
      <ExecutionReviewTab
        execution={exec}
        reviewRounds={[]}
        isGatheringEvidence={false}
        targetScenarios={["app-a", "app-b"]}
        postRunBadgeExecution={exec}
        isActive={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId("mock-scenario-reviews")).toBeInTheDocument();
    expect(screen.getByText("app-a, app-b")).toBeInTheDocument();
  });

  it("renders EvidencePanel when review rounds exist", () => {
    const round: ReviewRound = {
      round: 1,
      generated_at: "2026-03-20T00:00:00Z",
      execution_id: "exec-1",
      status: "complete",
      evidence: [],
    };
    render(
      <ExecutionReviewTab
        execution={makeExecution()}
        reviewRounds={[round]}
        isGatheringEvidence={false}
        targetScenarios={[]}
        postRunBadgeExecution={null}
        isActive={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId("mock-evidence-panel")).toBeInTheDocument();
    expect(screen.getByText("1 rounds")).toBeInTheDocument();
  });

  it("renders EvidencePanel when gathering evidence", () => {
    render(
      <ExecutionReviewTab
        execution={makeExecution()}
        reviewRounds={[]}
        isGatheringEvidence={true}
        targetScenarios={[]}
        postRunBadgeExecution={null}
        isActive={false}
        {...noopHandlers}
      />,
    );

    expect(screen.getByTestId("mock-evidence-panel")).toBeInTheDocument();
  });
});
