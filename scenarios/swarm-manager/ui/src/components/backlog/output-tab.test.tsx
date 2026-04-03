import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { OutputTab } from "./output-tab";
import type { OutputTabProps } from "./output-tab";
import type { ExecutionRecord } from "../../types";

vi.mock("../../services", () => ({
  executionService: {
    triggerReview: vi.fn().mockResolvedValue({}),
    cancel: vi.fn().mockResolvedValue({}),
  },
}));

vi.mock("../../services/review-service", () => ({
  reviewService: {
    triggerReviewAgent: vi.fn().mockResolvedValue(undefined),
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
  agentRunIsActive: false,
  latestAgentActivity: null,
  reviewRounds: [],
  isGatheringEvidence: false,
  backlogKind: "execute",
  backlogName: "test-item",
  onStopRun: vi.fn(),
  onFollowUp: vi.fn(),
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
});
