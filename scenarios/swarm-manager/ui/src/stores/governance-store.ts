/**
 * Governance Store — Tracks circuit-breaker state from the overview endpoint.
 *
 * Polls /api/v1/overview and extracts the governance.circuit_broken_items list.
 * Components use `isCircuitBroken(kind, name)` for item-level checks.
 */

import { create } from "zustand";
import { defaultApiClient, type IApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";

interface GovernanceState {
  /** Set of "kind/name" strings for items whose circuit breaker is tripped. */
  circuitBrokenItems: Set<string>;
  isRefreshing: boolean;
  refreshGovernance: () => Promise<void>;
}

interface OverviewGovernance {
  circuit_broken_items?: string[];
}

interface OverviewResponse {
  governance?: OverviewGovernance;
}

let apiClient: IApiClient = defaultApiClient;

/** Override the API client (for testing). */
export function setGovernanceApiClient(client: IApiClient): void {
  apiClient = client;
}

export const useGovernanceStore = create<GovernanceState>((set, get) => ({
  circuitBrokenItems: new Set<string>(),
  isRefreshing: false,

  refreshGovernance: async (): Promise<void> => {
    if (get().isRefreshing) return;
    set({ isRefreshing: true });
    try {
      const resp = await apiClient.get<OverviewResponse>(API_ENDPOINTS.overview);
      const items = resp.governance?.circuit_broken_items ?? [];
      set({ circuitBrokenItems: new Set(items) });
    } catch {
      // Governance polling is supplemental. Preserve last known state on failure.
    } finally {
      set({ isRefreshing: false });
    }
  },
}));

/** Selector: check whether a specific backlog item's circuit breaker is tripped. */
export function isCircuitBroken(state: GovernanceState, kind: string, name: string): boolean {
  return state.circuitBrokenItems.has(`${kind}/${name}`);
}
