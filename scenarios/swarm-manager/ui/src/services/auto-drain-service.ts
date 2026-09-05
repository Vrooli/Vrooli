/**
 * Auto-drain service — the continuous goal-directed auto-enqueue toggle (D4,
 * default OFF). Backed by a swarm-manager-local endpoint (not proto settings),
 * so goal-directed execution stays inside the scenario change boundary.
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";

export interface AutoDrainState {
  enabled: boolean;
}

export interface IAutoDrainService {
  get(): Promise<AutoDrainState>;
  set(enabled: boolean): Promise<AutoDrainState>;
}

function normalize(raw: { enabled?: boolean } | null | undefined): AutoDrainState {
  return { enabled: Boolean(raw?.enabled) };
}

export function createAutoDrainService(apiClient: IApiClient = defaultApiClient): IAutoDrainService {
  return {
    async get(): Promise<AutoDrainState> {
      const raw = await apiClient.get<{ enabled?: boolean }>(API_ENDPOINTS.executionAutoDrain);
      return normalize(raw);
    },
    async set(enabled: boolean): Promise<AutoDrainState> {
      const raw = await apiClient.put<{ enabled?: boolean }>(API_ENDPOINTS.executionAutoDrain, { enabled });
      return normalize(raw);
    },
  };
}

export const autoDrainService = createAutoDrainService();
