import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import type { ReactNode } from "react";
import { createElement } from "react";
import type { StatsResponse } from "../types/stats";

const mockGetStats = vi.fn<() => Promise<StatsResponse>>();

vi.mock("../services", () => ({
  statsService: { getStats: (...args: unknown[]) => mockGetStats(...(args as [])) },
}));

import { useStats } from "./useStats";

function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: ReactNode }) {
    return createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

const MOCK_STATS: StatsResponse = {
  generated_at: "2026-03-31T10:00:00Z",
  event_count: 42,
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
  scope: { initiatives: [], max_dependency_depth: 0 },
  blocking: { currently_blocked: 0, blocked_ratio: 0, top_reasons: [], avg_block_hours: 0 },
  agent: {
    total_executions: 10,
    success_rate: 0.9,
    failure_rate: 0.1,
    follow_up_rate: 0.2,
    avg_execution_minutes: 5.0,
    avg_workshop_rounds: 1.5,
  },
  dashboard: {
    total_backlog_size: 20,
    total_completed_all_time: 100,
    velocity_trend: [{ week_start: "2026-03-24", completed: 5 }],
    estimated_weeks_remaining: 4.0,
  },
};

describe("useStats", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches stats when enabled", async () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    const { result } = renderHook(() => useStats(true), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockGetStats).toHaveBeenCalledTimes(1);
    expect(result.current.data).toEqual(MOCK_STATS);
  });

  it("does not fetch when disabled", () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    const { result } = renderHook(() => useStats(false), { wrapper: createWrapper() });

    expect(result.current.isFetching).toBe(false);
    expect(mockGetStats).not.toHaveBeenCalled();
  });

  it("exposes error when fetch fails", async () => {
    mockGetStats.mockRejectedValue(new Error("Server error"));
    const { result } = renderHook(() => useStats(true), { wrapper: createWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("Server error");
  });
});
