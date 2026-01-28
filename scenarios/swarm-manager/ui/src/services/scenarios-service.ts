/**
 * Scenarios Service - Data access layer for scenario operations
 *
 * This service encapsulates all scenario-related API operations behind a clean seam.
 * It accepts an API client as a dependency, making it easy to substitute for testing.
 *
 * Responsibilities:
 * - Scenario listing and details
 * - Scenario metadata updates
 * - Request/response transformation if needed
 *
 * NOT responsible for:
 * - HTTP implementation details (delegated to api client)
 * - UI state or caching (delegated to React Query)
 * - Scenario lifecycle operations (delegated to ecosystem-manager)
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Scenario, UpdateScenarioMetadataRequest, DeleteScenarioResponse } from "../types";

/**
 * Interface for the scenarios service.
 * This is the seam - implementations can be swapped for testing.
 * [REQ:REQ-P0-007] Includes updateMetadata for scenario metadata management
 * [REQ:REQ-P0-008] Includes delete for scenario deletion with archive option
 */
export interface IScenariosService {
  list(): Promise<Scenario[]>;
  get(name: string): Promise<Scenario>;
  updateMetadata(name: string, request: UpdateScenarioMetadataRequest): Promise<Scenario>;
  delete(name: string, archive?: boolean): Promise<DeleteScenarioResponse>;
}

/**
 * Creates a scenarios service with the given API client.
 *
 * @param apiClient - The API client to use for HTTP requests
 * @returns A scenarios service instance
 */
export function createScenariosService(
  apiClient: IApiClient = defaultApiClient
): IScenariosService {
  return {
    async list(): Promise<Scenario[]> {
      return apiClient.get<Scenario[]>(API_ENDPOINTS.scenarios);
    },

    async get(name: string): Promise<Scenario> {
      return apiClient.get<Scenario>(API_ENDPOINTS.scenarioByName(name));
    },

    /**
     * Updates scenario metadata (greenfield toggle, recommendations enable/disable)
     * [REQ:REQ-P0-007] PATCH endpoint for scenario metadata management
     */
    async updateMetadata(name: string, request: UpdateScenarioMetadataRequest): Promise<Scenario> {
      return apiClient.patch<Scenario>(API_ENDPOINTS.scenarioByName(name), request);
    },

    /**
     * Deletes a scenario with optional archive to ideas backlog
     * [REQ:REQ-P0-008] DELETE endpoint for scenario deletion with safeguards
     */
    async delete(name: string, archive = false): Promise<DeleteScenarioResponse> {
      const endpoint = archive
        ? `${API_ENDPOINTS.scenarioByName(name)}?archive=true`
        : API_ENDPOINTS.scenarioByName(name);
      return apiClient.delete<DeleteScenarioResponse>(endpoint);
    },
  };
}

/**
 * Default scenarios service instance.
 * Uses the default API client for production use.
 */
export const scenariosService = createScenariosService();
