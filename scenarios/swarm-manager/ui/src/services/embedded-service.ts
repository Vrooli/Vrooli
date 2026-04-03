/**
 * Embedded Service - Data access layer for embedded service proxy endpoints
 *
 * These endpoints are served at the origin root (not under /api/v1),
 * so they use a dedicated API client with the base origin URL.
 */

import type { IApiClient } from "../lib/api-client";
import { createApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";

interface ExternalUrlResponse {
  url?: string;
}

export interface IEmbeddedService {
  getExternalUrl(serviceName: string): Promise<string | null>;
}

/**
 * Creates an embedded service using a client that targets the origin root.
 * The default client appends /api/v1; embedded endpoints live at /embedded/...
 */
export function createEmbeddedService(apiClient?: IApiClient): IEmbeddedService {
  // Use origin root as the base — embedded routes are not under /api/v1
  const client = apiClient ?? createApiClient(window.location.origin);

  return {
    async getExternalUrl(serviceName: string): Promise<string | null> {
      const data = await client.get<ExternalUrlResponse>(
        API_ENDPOINTS.embeddedExternalUrl(serviceName),
      );
      return data?.url ?? null;
    },
  };
}

export const embeddedService = createEmbeddedService();
