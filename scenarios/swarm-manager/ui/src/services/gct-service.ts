import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";

export interface GCTStatusResponse {
  available: boolean;
}

export interface IGCTService {
  getStatus(): Promise<GCTStatusResponse>;
}

export function createGCTService(apiClient: IApiClient = defaultApiClient): IGCTService {
  return {
    async getStatus(): Promise<GCTStatusResponse> {
      try {
        return await apiClient.get<GCTStatusResponse>(API_ENDPOINTS.gctStatus);
      } catch {
        return { available: false };
      }
    },
  };
}

export const gctService = createGCTService();
