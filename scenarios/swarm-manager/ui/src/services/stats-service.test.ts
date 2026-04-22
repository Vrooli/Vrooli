import { describe, it, expect, vi, beforeEach } from "vitest";
import { createStatsService, type IStatsService } from "./stats-service";
import type { IApiClient } from "../lib/api-client";
import type { StatsResponse } from "../types/stats";

describe("Stats Service", () => {
  let mockApiClient: IApiClient;
  let service: IStatsService;

  beforeEach(() => {
    mockApiClient = {
      get: vi.fn(),
      post: vi.fn(),
      put: vi.fn(),
      patch: vi.fn(),
      delete: vi.fn(),
    };

    service = createStatsService(mockApiClient);
  });

  describe("getStats", () => {
    it("fetches stats from the correct endpoint", async () => {
      const mockStats: StatsResponse = {
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
        },
        timing: {
          avg_lead_time_hours: 12.0,
          median_lead_time_hours: 10.0,
          lead_time_sample_size: 10,
          avg_execution_minutes: 5.0,
          median_execution_minutes: 4.0,
          execution_duration_samples: 10,
        },
        scope: { initiatives: [], max_dependency_depth: 0 },
        blocking: {
          currently_blocked: 0,
          blocked_ratio: 0,
          top_reasons: [],
          avg_block_hours: 0,
        },
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
        },
        dashboard: {
          total_backlog_size: 20,
          total_completed_all_time: 100,
          velocity_trend: [{ week_start: "2026-03-24", completed: 5 }],
          estimated_weeks_remaining: 4.0,
          velocity_weeks_covered: 1,
        },
      };
      vi.mocked(mockApiClient.get).mockResolvedValue(mockStats);

      const result = await service.getStats();

      expect(mockApiClient.get).toHaveBeenCalledWith("/stats");
      expect(result).toEqual(mockStats);
    });

    it("propagates API errors", async () => {
      const error = new Error("Network failure");
      vi.mocked(mockApiClient.get).mockRejectedValue(error);

      await expect(service.getStats()).rejects.toThrow("Network failure");
    });
  });
});
