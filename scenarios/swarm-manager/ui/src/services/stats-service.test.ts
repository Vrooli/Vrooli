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
        blocking: {
          currently_blocked: 0,
          blocked_ratio: 0,
          top_reasons: [],
          avg_block_hours: 0,
        },
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
