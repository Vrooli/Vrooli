/**
 * Recommendations Service - Data access layer for recommendations
 */

import { z } from "zod";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Recommendation, RecommendationStatus } from "../types";

const recommendationSchema = z.object({
  id: z.string(),
  scenarioName: z.string(),
  type: z.enum(["test", "feature", "refactor", "docs"]),
  description: z.string(),
  status: z.enum(["pending", "approved", "rejected"]),
  priority: z.number(),
  created: z.string(),
  taskId: z.string().optional(),
  runId: z.string().optional(),
  startedAt: z.string().optional(),
  startedBy: z.string().optional(),
  autoApproved: z.boolean().optional(),
});

const listSchema = z.object({
  recommendations: z.array(recommendationSchema),
});

const updateSchema = z.object({
  recommendation: recommendationSchema,
});

export interface IRecommendationsService {
  list(): Promise<Recommendation[]>;
  refresh(): Promise<Recommendation[]>;
  updateStatus(id: string, status: RecommendationStatus): Promise<Recommendation>;
  start(id: string): Promise<Recommendation>;
}

export function createRecommendationsService(apiClient: IApiClient = defaultApiClient): IRecommendationsService {
  return {
    async list(): Promise<Recommendation[]> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.recommendations);
      const parsed = listSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error("Invalid recommendations response");
      }
      return parsed.data.recommendations;
    },

    async refresh(): Promise<Recommendation[]> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.recommendationsRefresh, {});
      const parsed = listSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error("Invalid recommendations response");
      }
      return parsed.data.recommendations;
    },

    async updateStatus(id: string, status: RecommendationStatus): Promise<Recommendation> {
      const data = await apiClient.patch<unknown>(`${API_ENDPOINTS.recommendations}/${id}`, { status });
      const parsed = updateSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error("Invalid recommendation response");
      }
      return parsed.data.recommendation;
    },

    async start(id: string): Promise<Recommendation> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.recommendationsStart(id), {});
      const parsed = updateSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error("Invalid recommendation response");
      }
      return parsed.data.recommendation;
    },
  };
}

export const recommendationsService = createRecommendationsService();
