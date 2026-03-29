/**
 * Initiative Service - Data access layer for initiative operations
 *
 * Encapsulates initiative-related API operations behind a clean seam.
 * Accepts an API client as a dependency for test injection.
 *
 * The backend returns JSON with proto field names (snake_case). This service
 * normalizes responses to match the generated TS types (camelCase).
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { InitiativeWithRollup } from "../types";
import type { TreeFile } from "../components/ui/file-tree";

/** Raw API rollup shape (snake_case from proto JSON). */
interface RawRollup {
  total?: number;
  completed?: number;
  in_progress?: number;
  inProgress?: number;
  failed?: number;
  pending?: number;
}

/** Normalize a single initiative-with-rollup from the API's snake_case to camelCase. */
function normalizeItem(raw: { initiative?: Record<string, unknown>; rollup?: RawRollup }): InitiativeWithRollup {
  const rollup = raw.rollup ?? {};
  return {
    ...raw,
    rollup: {
      ...rollup,
      // Accept both snake_case (API) and camelCase (already normalized)
      inProgress: rollup.inProgress ?? rollup.in_progress ?? 0,
    },
  } as InitiativeWithRollup;
}

function normalizeItems(raw: unknown[]): InitiativeWithRollup[] {
  return raw.map((item) => normalizeItem(item as { initiative?: Record<string, unknown>; rollup?: RawRollup }));
}

export interface IInitiativeService {
  list(): Promise<InitiativeWithRollup[]>;
  get(name: string): Promise<InitiativeWithRollup>;
  listFiles(name: string): Promise<TreeFile[]>;
  getFileContent(name: string, path: string): Promise<string>;
}

export function createInitiativeService(
  apiClient: IApiClient = defaultApiClient,
): IInitiativeService {
  return {
    async list(): Promise<InitiativeWithRollup[]> {
      const resp = await apiClient.get<{ items?: unknown[] } | unknown[]>(API_ENDPOINTS.initiatives);
      const raw = Array.isArray(resp) ? resp : (resp.items ?? []);
      return normalizeItems(raw);
    },

    async get(name: string): Promise<InitiativeWithRollup> {
      const raw = await apiClient.get<Record<string, unknown>>(API_ENDPOINTS.initiativeByName(name));
      return normalizeItem(raw as { initiative?: Record<string, unknown>; rollup?: RawRollup });
    },

    async listFiles(name: string): Promise<TreeFile[]> {
      const resp = await apiClient.get<{ files: TreeFile[] }>(API_ENDPOINTS.initiativeFiles(name));
      return resp.files ?? [];
    },

    async getFileContent(name: string, path: string): Promise<string> {
      return apiClient.get<string>(API_ENDPOINTS.initiativeFileContent(name, path), { responseType: "text" });
    },
  };
}

export const initiativeService = createInitiativeService();
