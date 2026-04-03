import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { OutputTab } from "./output-tab";
import type { OutputTabProps } from "./output-tab";
import type { ExecutionRecord } from "../../types";

vi.mock("../../services", () => ({
  executionService: {
    triggerReview: vi.fn().mockResolvedValue({}),
  },
}));

const makeExecution = (overrides?: Partial<ExecutionRecord>): ExecutionRecord => ({
  executionId: "exec-1",
  backlogKind: "execute",
  backlogName: "test-item",
  status: "completed",
  mode: "yolo",
  createdAt: "2026-03-20T12:00:00Z",
  ...overrides,
} as ExecutionRecord);

const createTestQueryClient = () =>
  new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });

const defaultProps: OutputTabProps = {
  executionHistory: undefined,
  timeline: { entries: [], isLoading: false, error: null },
  targetScenarios: [],
  agentRunIsActive: false,
  latestAgentActivity: null,
  agentManagerUiUrl: null,
  reviewRounds: [],
  isGatheringEvidence: false,
  backlogKind: "execute",
  backlogName: "test-item",
  onStopRun: vi.fn(),
  onFollowUp: vi.fn(),
  onViewExecution: vi.fn(),
  onSelectScenario: vi.fn(),
  onVerifyEvidence: vi.fn(),
  onRequestMoreEvidence: vi.fn(),
};

function renderWithProviders(props: OutputTabProps) {
  const queryClient = createTestQueryClient();
  return render(
    <QueryClientProvider client={queryClient}>
      <OutputTab {...props} />
    </QueryClientProvider>,
  );
}

describe("OutputTab", () => {
  it("renders empty state when no data", () => {
    renderWithProviders(defaultProps);
    expect(screen.getByText(/no executions yet/i)).toBeInTheDocument();
    expect(screen.getByTestId("backlog-details-output-tab")).toBeInTheDocument();
  });

  it("renders LatestExecutionSummary with execution data", () => {
    renderWithProviders({
      ...defaultProps,
      executionHistory: [makeExecution()],
    });
    expect(screen.getByText("Completed")).toBeInTheDocument();
  });

  it("hides scenario chips when no target scenarios", () => {
    renderWithProviders({
      ...defaultProps,
      executionHistory: [makeExecution()],
      targetScenarios: [],
    });
    expect(screen.queryByTestId("review-scenario-chips")).not.toBeInTheDocument();
  });

  it("shows scenario chips when target scenarios exist", () => {
    renderWithProviders({
      ...defaultProps,
      executionHistory: [makeExecution()],
      targetScenarios: ["my-scenario"],
    });
    expect(screen.getByText("my-scenario")).toBeInTheDocument();
  });

  it("renders ActivityTimeline section", () => {
    renderWithProviders({
      ...defaultProps,
      timeline: {
        entries: [],
        isLoading: false,
        error: null,
      },
    });
    // ActivityTimeline renders with empty entries - look for the output tab container
    expect(screen.getByTestId("backlog-details-output-tab")).toBeInTheDocument();
  });
});
