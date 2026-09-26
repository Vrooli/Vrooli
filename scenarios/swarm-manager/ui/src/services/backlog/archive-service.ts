/**
 * Backlog Archive Service — archive targets, modules, and review operations
 */

import type { IApiClient } from "../../lib/api-client";
import { API_ENDPOINTS } from "../../lib/api-endpoints";
import type {
  ArchiveRequirementRecord,
  ArchiveTargetFormValues,
  ArchiveTargetsResponse,
  BacklogKind,
  ModuleFormValues,
  ReviewUpdate,
} from "../../types";

export function createArchiveMethods(apiClient: IApiClient) {
  return {
    async getArchiveTargets(kind: BacklogKind, name: string): Promise<ArchiveTargetsResponse> {
      return apiClient.get<ArchiveTargetsResponse>(API_ENDPOINTS.backlogArchiveTargets(kind, name));
    },

    async createArchiveTarget(kind: string, name: string, target: ArchiveTargetFormValues): Promise<void> {
      await apiClient.post(API_ENDPOINTS.backlogArchiveTargets(kind, name), target);
    },

    async updateArchiveTarget(kind: string, name: string, targetId: string, target: ArchiveTargetFormValues): Promise<void> {
      await apiClient.put(API_ENDPOINTS.backlogArchiveTarget(kind, name, targetId), target);
    },

    async deleteArchiveTarget(kind: string, name: string, targetId: string): Promise<void> {
      await apiClient.delete(API_ENDPOINTS.backlogArchiveTarget(kind, name, targetId));
    },

    async updateModuleRequirements(kind: string, name: string, moduleId: string, requirements: ArchiveRequirementRecord[]): Promise<void> {
      await apiClient.put(API_ENDPOINTS.backlogArchiveRequirementsModule(kind, name, moduleId), { requirements });
    },

    async createModule(kind: string, name: string, payload: ModuleFormValues & { position?: number }): Promise<void> {
      await apiClient.post(API_ENDPOINTS.backlogArchiveRequirements(kind, name), payload);
    },

    async updateModuleMeta(kind: string, name: string, moduleId: string, payload: { title: string; description: string }): Promise<void> {
      await apiClient.put(API_ENDPOINTS.backlogArchiveRequirementsModuleMeta(kind, name, moduleId), payload);
    },

    async deleteModule(kind: string, name: string, moduleId: string): Promise<void> {
      await apiClient.delete(API_ENDPOINTS.backlogArchiveRequirementsModule(kind, name, moduleId));
    },

    async batchReview(kind: string, name: string, items: ReviewUpdate[]): Promise<void> {
      await apiClient.put(API_ENDPOINTS.backlogArchiveReview(kind, name), { items });
    },
  };
}
