import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, fireEvent, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import type { StatsResponse } from "../../../types/stats";

// ---------------------------------------------------------------------------
// Mocks
// ---------------------------------------------------------------------------

const mockGetStats = vi.fn<() => Promise<StatsResponse>>();

vi.mock("../../../services", () => ({
  statsService: { getStats: (...args: unknown[]) => mockGetStats(...(args as [])) },
}));

vi.mock("../../../components/ui/floating-panel", () => ({
  FloatingPanel: ({
    children,
    isOpen,
    testId,
  }: {
    children: ReactNode;
    isOpen: boolean;
    title: string;
    onClose: () => void;
    testId?: string;
    className?: string;
    initialPosition?: { x: number; y: number };
  }) => (isOpen ? <div data-testid={testId}>{children}</div> : null),
}));

import { StatsPanel } from "./StatsPanel";

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

const MOCK_STATS: StatsResponse = {
  generated_at: "2026-03-31T10:00:00Z",
  event_count: 1234,
  throughput: {
    completed_last_7_days: 5,
    completed_last_30_days: 18,
    created_last_7_days: 8,
    created_last_30_days: 35,
    net_delta_7_days: 3,
    net_delta_30_days: 17,
  },
  timing: {
    avg_cycle_time_hours: 2.5,
    avg_lead_time_hours: 12.0,
    avg_queue_wait_hours: 0.5,
    median_cycle_time_hours: 2.0,
    median_lead_time_hours: 10.0,
  },
  scope: {
    initiatives: [
      { name: "auth-rework", total: 10, completed: 8, in_progress: 1, blocked: 1, scope_creep: 0.15 },
    ],
    max_dependency_depth: 2,
  },
  blocking: {
    currently_blocked: 3,
    blocked_ratio: 0.064,
    top_reasons: [
      { reason: "waiting on upstream PR", count: 7 },
      { reason: "missing API spec", count: 4 },
    ],
    avg_block_hours: 4.1,
  },
  agent: {
    total_executions: 87,
    success_rate: 0.912,
    failure_rate: 0.088,
    follow_up_rate: 0.143,
    avg_execution_minutes: 4.2,
    avg_workshop_rounds: 1.8,
  },
  dashboard: {
    total_backlog_size: 47,
    total_completed_all_time: 123,
    velocity_trend: [
      { week_start: "2026-03-17", completed: 3 },
      { week_start: "2026-03-24", completed: 5 },
    ],
    estimated_weeks_remaining: 8.6,
  },
};

function renderWithProviders(ui: React.ReactElement) {
  const queryClient = new QueryClient({
    defaultOptions: { queries: { retry: false } },
  });
  return render(<QueryClientProvider client={queryClient}>{ui}</QueryClientProvider>);
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

describe("StatsPanel", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("renders nothing when closed", () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    renderWithProviders(<StatsPanel isOpen={false} onClose={vi.fn()} />);
    expect(screen.queryByTestId("stats-panel")).not.toBeInTheDocument();
  });

  it("shows loading state while fetching", () => {
    // Never resolve — stays loading
    mockGetStats.mockReturnValue(new Promise(() => {}));
    renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);
    expect(screen.getByTestId("stats-loading")).toBeInTheDocument();
  });

  it("shows error state when fetch fails", async () => {
    mockGetStats.mockRejectedValue(new Error("Server down"));
    renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId("stats-error")).toBeInTheDocument());
    expect(screen.getByText(/Server down/)).toBeInTheDocument();
  });

  it("renders all 6 tab buttons", async () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId("stats-content-dashboard")).toBeInTheDocument());

    expect(screen.getByTestId("stats-tab-dashboard")).toBeInTheDocument();
    expect(screen.getByTestId("stats-tab-throughput")).toBeInTheDocument();
    expect(screen.getByTestId("stats-tab-agent")).toBeInTheDocument();
    expect(screen.getByTestId("stats-tab-timing")).toBeInTheDocument();
    expect(screen.getByTestId("stats-tab-blocking")).toBeInTheDocument();
    expect(screen.getByTestId("stats-tab-scope")).toBeInTheDocument();
  });

  it("defaults to the dashboard tab", async () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId("stats-content-dashboard")).toBeInTheDocument());
    expect(screen.getByTestId("stats-tab-dashboard")).toHaveAttribute("aria-selected", "true");
  });

  it("switches tabs when clicked", async () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

    await waitFor(() => expect(screen.getByTestId("stats-content-dashboard")).toBeInTheDocument());

    fireEvent.click(screen.getByTestId("stats-tab-agent"));
    expect(screen.getByTestId("stats-content-agent")).toBeInTheDocument();
    expect(screen.queryByTestId("stats-content-dashboard")).not.toBeInTheDocument();
  });

  describe("Dashboard tab", () => {
    it("displays backlog size, completed count, and estimated weeks", async () => {
      mockGetStats.mockResolvedValue(MOCK_STATS);
      renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

      await waitFor(() => expect(screen.getByTestId("stat-backlog-size")).toBeInTheDocument());
      expect(screen.getByTestId("stat-backlog-size")).toHaveTextContent("47");
      expect(screen.getByTestId("stat-completed-all-time")).toHaveTextContent("123");
      expect(screen.getByTestId("stat-weeks-remaining")).toHaveTextContent("~8.6 weeks");
    });

    it("shows event count", async () => {
      mockGetStats.mockResolvedValue(MOCK_STATS);
      renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

      await waitFor(() => expect(screen.getByText(/1,234 events processed/)).toBeInTheDocument());
    });

    it("shows empty state for velocity trend", async () => {
      const emptyStats = {
        ...MOCK_STATS,
        dashboard: { ...MOCK_STATS.dashboard, velocity_trend: [] },
      };
      mockGetStats.mockResolvedValue(emptyStats);
      renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

      await waitFor(() => expect(screen.getByText("No velocity data yet")).toBeInTheDocument());
    });
  });

  describe("Throughput tab", () => {
    it("displays created and completed counts", async () => {
      mockGetStats.mockResolvedValue(MOCK_STATS);
      renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

      await waitFor(() => expect(screen.getByTestId("stats-content-dashboard")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("stats-tab-throughput"));

      expect(screen.getByTestId("stats-content-throughput")).toBeInTheDocument();
      expect(screen.getByText("8")).toBeInTheDocument(); // created 7d
      expect(screen.getByText("35")).toBeInTheDocument(); // created 30d
    });
  });

  describe("Blocking tab", () => {
    it("displays blocking reasons", async () => {
      mockGetStats.mockResolvedValue(MOCK_STATS);
      renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

      await waitFor(() => expect(screen.getByTestId("stats-content-dashboard")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("stats-tab-blocking"));

      expect(screen.getByText("waiting on upstream PR")).toBeInTheDocument();
      expect(screen.getByText("missing API spec")).toBeInTheDocument();
    });

    it("shows empty state when no reasons", async () => {
      const noBlockStats = {
        ...MOCK_STATS,
        blocking: { ...MOCK_STATS.blocking, top_reasons: [] },
      };
      mockGetStats.mockResolvedValue(noBlockStats);
      renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

      await waitFor(() => expect(screen.getByTestId("stats-content-dashboard")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("stats-tab-blocking"));

      expect(screen.getByText("No blocking reasons recorded")).toBeInTheDocument();
    });
  });

  describe("Scope tab", () => {
    it("displays initiative progress", async () => {
      mockGetStats.mockResolvedValue(MOCK_STATS);
      renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

      await waitFor(() => expect(screen.getByTestId("stats-content-dashboard")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("stats-tab-scope"));

      expect(screen.getByText("auth-rework")).toBeInTheDocument();
      expect(screen.getByText("80%")).toBeInTheDocument(); // 8/10
      expect(screen.getByText("1 blocked")).toBeInTheDocument();
    });

    it("shows empty state when no initiatives", async () => {
      const noInitStats = {
        ...MOCK_STATS,
        scope: { initiatives: [], max_dependency_depth: 0 },
      };
      mockGetStats.mockResolvedValue(noInitStats);
      renderWithProviders(<StatsPanel isOpen={true} onClose={vi.fn()} />);

      await waitFor(() => expect(screen.getByTestId("stats-content-dashboard")).toBeInTheDocument());
      fireEvent.click(screen.getByTestId("stats-tab-scope"));

      expect(screen.getByText("No initiatives yet")).toBeInTheDocument();
    });
  });
});
