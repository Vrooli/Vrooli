/**
 * Stats Service - Data access layer for stats operations
 *
 * Encapsulates the stats API call behind a clean seam.
 * Accepts an API client as a dependency for testability.
 *
 * DOC: docs/internal/SEAMS.md#ui-to-api-seam-improved-in-phase-3
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { StatsResponse } from "../types/stats";

export interface IStatsService {
  /** Fetches all stats categories from the API. */
  getStats(options?: { goal?: string }): Promise<StatsResponse>;
}

export function createStatsService(apiClient: IApiClient): IStatsService {
  return {
    async getStats(options?: { goal?: string }): Promise<StatsResponse> {
      const goal = options?.goal?.trim();
      const path = goal
        ? `${API_ENDPOINTS.stats}?${new URLSearchParams({ goal }).toString()}`
        : API_ENDPOINTS.stats;
      return apiClient.get<StatsResponse>(path);
    },
  };
}

export const statsService: IStatsService = createStatsService(defaultApiClient);
