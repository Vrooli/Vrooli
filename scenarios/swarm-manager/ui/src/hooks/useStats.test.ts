import { describe, it, expect, vi, beforeEach } from "vitest";
import { renderHook, waitFor } from "@testing-library/react";
import type { StatsResponse } from "../types/stats";
import { createQueryWrapper } from "../test-utils";

const mockGetStats = vi.fn<(options?: { goal?: string }) => Promise<StatsResponse>>();

vi.mock("../services", () => ({
  statsService: { getStats: (options?: { goal?: string }) => mockGetStats(options) },
}));

import { useStats } from "./useStats";

const MOCK_STATS: StatsResponse = {
  generated_at: "2026-03-31T10:00:00Z",
  event_count: 42,
  history: {
    earliest_event_at: "2026-03-24T10:00:00Z",
    history_days: 7,
    has_history: true,
    min_sample_meaningful: 5,
  },
  throughput: {
    completed_last_7_days: 5,
    completed_last_30_days: 18,
    created_last_7_days: 8,
    created_last_30_days: 35,
    net_delta_7_days: 3,
    net_delta_30_days: 17,
    throughput_trend: [{ week_start: "2026-03-24", created: 8, completed: 5 }],
  },
  timing: {
    avg_lead_time_hours: 12.0,
    median_lead_time_hours: 10.0,
    lead_time_sample_size: 10,
    avg_execution_minutes: 5.0,
    median_execution_minutes: 4.0,
    execution_duration_samples: 10,
  },
  scope: { goals: [], max_dependency_depth: 0 },
  blocking: { currently_blocked: 0, blocked_ratio: 0, top_reasons: [], avg_block_hours: 0 },
  agent: {
    total_executions: 10,
    completed_count: 9,
    failed_count: 1,
    manually_accepted_count: 0,
    success_rate: 0.9,
    failure_rate: 0.1,
    manual_accept_rate: 0,
    follow_up_rate: 0.2,
    avg_execution_minutes: 5.0,
    avg_workshop_rounds: 1.5,
    success_rate_sample_size: 10,
    execution_duration_samples: 10,
    workshop_rounds_sample_size: 6,
    recommendation_acceptance_rate: 0,
    recommendation_acceptance_sample_size: 0,
    freeform_override_rate: 0,
    decision_items_total: 0,
    decision_items_answered: 0,
    recommendation_acceptance_by_kind: {},
  },
  dashboard: {
    total_backlog_size: 20,
    total_completed_all_time: 100,
    velocity_trend: [{ week_start: "2026-03-24", completed: 5 }],
    estimated_remaining: {
      p50_hours: 96,
      p80_hours: 144,
      p50_label: "~4 days",
      p80_label: "~6 days",
      basis: "default",
      basis_label: "priors only",
      confidence: "low",
      remaining_items: 20,
      lane_capacity: 1,
    },
    velocity_weeks_covered: 1,
  },
  mode: {
    usage_by_mode: { "item-level": 1 },
    mode_switch_count: 0,
    phase_runs_by_mode: {},
    completed_by_mode: {},
    failed_by_mode: {},
    canceled_by_mode: {},
    replan_rate_by_mode: {},
    acceptance_rate_by_mode: {},
    avg_phase_duration_seconds: {},
    avg_runs_per_completed_scope: {},
    backlog_sync_by_mode: {},
    usage_by_profile: {},
    phase_runs_by_profile: {},
  },
  session: {
    total_sessions: 0,
    active_sessions: 0,
    sessions_by_kind: {},
    sessions_by_status: {},
    proposal_created_by_kind: {},
    proposal_applied_by_kind: {},
    proposal_apply_rate_by_kind: {},
    artifacts_created_by_kind: {},
    artifacts_by_type: {},
    avg_messages_per_session: 0,
    avg_time_to_first_proposal_seconds: 0,
    first_proposal_sample_size: 0,
    failed_session_rate: 0,
    failed_session_sample_size: 0,
    session_created_backlog_items: 0,
    session_created_goals: 0,
  },
};

describe("useStats", () => {
  beforeEach(() => {
    vi.clearAllMocks();
  });

  it("fetches stats when enabled", async () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    const { result } = renderHook(() => useStats(true), { wrapper: createQueryWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockGetStats).toHaveBeenCalledTimes(1);
    expect(mockGetStats).toHaveBeenCalledWith({ goal: "" });
    expect(result.current.data).toEqual(MOCK_STATS);
  });

  it("passes the selected goal into the stats service", async () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    const { result } = renderHook(() => useStats(true, "goal-x"), { wrapper: createQueryWrapper() });

    await waitFor(() => expect(result.current.isSuccess).toBe(true));
    expect(mockGetStats).toHaveBeenCalledWith({ goal: "goal-x" });
  });

  it("does not fetch when disabled", () => {
    mockGetStats.mockResolvedValue(MOCK_STATS);
    const { result } = renderHook(() => useStats(false), { wrapper: createQueryWrapper() });

    expect(result.current.isFetching).toBe(false);
    expect(mockGetStats).not.toHaveBeenCalled();
  });

  it("exposes error when fetch fails", async () => {
    mockGetStats.mockRejectedValue(new Error("Server error"));
    const { result } = renderHook(() => useStats(true), { wrapper: createQueryWrapper() });

    await waitFor(() => expect(result.current.isError).toBe(true));
    expect(result.current.error?.message).toBe("Server error");
  });
});
