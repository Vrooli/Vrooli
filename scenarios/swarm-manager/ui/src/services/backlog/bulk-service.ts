/**
 * Backlog Bulk Service — export, import, and summary operations
 */

import type { IApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import type {
  BacklogSummaryResponse,
  MaturitySummaryResponse,
  PendingQuestionsResponse,
} from "../../types";
import type { ImportBacklogResponse } from "./types";

export function createBulkMethods(apiClient: IApiClient) {
  return {
    async exportItems(params?: {
      kinds?: string[];
      statuses?: string[];
      names?: string[];
      priorityMax?: number;
      tags?: string[];
      includePrd?: boolean;
      includeRequirements?: boolean;
      includeClarifyQuestions?: boolean;
      includeSuggestions?: boolean;
      includeNotes?: boolean;
      includeTemplate?: boolean;
    }): Promise<Blob> {
      const response = await apiClient.post<Blob>(API_ENDPOINTS.backlogExport, params ?? {}, {
        responseType: "blob",
      });
      return response;
    },

    async importItems(file: File, apply = false): Promise<ImportBacklogResponse> {
      const formData = new FormData();
      formData.append("file", file);
      if (apply) {
        formData.append("apply", "true");
      }
      return apiClient.post<ImportBacklogResponse>(API_ENDPOINTS.backlogImport, formData, {
        headers: {},
      });
    },

    async getBacklogSummary(): Promise<BacklogSummaryResponse> {
      return apiClient.get<BacklogSummaryResponse>(API_ENDPOINTS.backlogSummary);
    },

    async getMaturitySummary(): Promise<MaturitySummaryResponse> {
      return apiClient.get<MaturitySummaryResponse>(API_ENDPOINTS.backlogMaturitySummary);
    },

    async getPendingQuestions(): Promise<PendingQuestionsResponse> {
      return apiClient.get<PendingQuestionsResponse>(API_ENDPOINTS.backlogPendingQuestions);
    },
  };
}
