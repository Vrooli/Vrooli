/**
 * Initiative Service - Data access layer for initiative operations
 *
 * Encapsulates initiative-related API operations behind a clean seam.
 * Accepts an API client as a dependency for test injection.
 *
 * The backend returns plain JSON (not proto), so no proto mapping is needed.
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { InitiativeWithRollup } from "../types";

export interface IInitiativeService {
  list(): Promise<InitiativeWithRollup[]>;
  get(name: string): Promise<InitiativeWithRollup>;
}

export function createInitiativeService(
  apiClient: IApiClient = defaultApiClient,
): IInitiativeService {
  return {
    async list(): Promise<InitiativeWithRollup[]> {
      return apiClient.get<InitiativeWithRollup[]>(API_ENDPOINTS.initiatives);
    },

    async get(name: string): Promise<InitiativeWithRollup> {
      return apiClient.get<InitiativeWithRollup>(API_ENDPOINTS.initiativeByName(name));
    },
  };
}

export const initiativeService = createInitiativeService();
