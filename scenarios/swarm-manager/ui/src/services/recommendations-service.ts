/**
 * Recommendations Service - Data access layer for recommendations
 */

import {
  StartRecommendationRequestSchema,
  UpdateRecommendationRequestSchema,
} from "@vrooli/proto-types/swarm-manager/v1/api/recommendations_pb";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Recommendation, RecommendationStatus } from "../types";
import {
  buildMessage,
  listRecommendationsResponseSchema,
  mapProtoRecommendation,
  parseProtoResponse,
  recommendationResponseSchema,
  requireProtoField,
  toProtoJson,
} from "./proto-contracts";

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
      const parsed = parseProtoResponse(listRecommendationsResponseSchema, data, "recommendations list");
      return (parsed.recommendations ?? []).map(mapProtoRecommendation);
    },

    async refresh(): Promise<Recommendation[]> {
      const data = await apiClient.post<unknown>(API_ENDPOINTS.recommendationsRefresh, {});
      const parsed = parseProtoResponse(listRecommendationsResponseSchema, data, "recommendations list");
      return (parsed.recommendations ?? []).map(mapProtoRecommendation);
    },

    async updateStatus(id: string, status: RecommendationStatus): Promise<Recommendation> {
      const message = buildMessage(UpdateRecommendationRequestSchema, { status });
      const payload = toProtoJson(UpdateRecommendationRequestSchema, message);
      const data = await apiClient.patch<unknown>(`${API_ENDPOINTS.recommendations}/${id}`, payload);
      const parsed = parseProtoResponse(recommendationResponseSchema, data, "recommendation");
      return mapProtoRecommendation(requireProtoField(parsed.recommendation, "recommendation"));
    },

    async start(id: string): Promise<Recommendation> {
      const message = buildMessage(StartRecommendationRequestSchema, {});
      const payload = toProtoJson(StartRecommendationRequestSchema, message);
      const data = await apiClient.post<unknown>(API_ENDPOINTS.recommendationsStart(id), payload);
      const parsed = parseProtoResponse(recommendationResponseSchema, data, "recommendation");
      return mapProtoRecommendation(requireProtoField(parsed.recommendation, "recommendation"));
    },
  };
}

export const recommendationsService = createRecommendationsService();
