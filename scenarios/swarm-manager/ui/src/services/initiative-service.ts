/**
 * Initiative Service - Data access layer for initiative operations
 *
 * Encapsulates initiative-related API operations behind a clean seam.
 * Accepts an API client as a dependency for test injection.
 *
 * The backend returns plain JSON (not proto), so no proto mapping is needed.
 * File management responses reuse backlog file proto types (structurally generic).
 */

import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { InitiativeWithRollup } from "../types";
import type { TreeFile } from "../components/ui/file-tree";

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
      return apiClient.get<InitiativeWithRollup[]>(API_ENDPOINTS.initiatives);
    },

    async get(name: string): Promise<InitiativeWithRollup> {
      return apiClient.get<InitiativeWithRollup>(API_ENDPOINTS.initiativeByName(name));
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
