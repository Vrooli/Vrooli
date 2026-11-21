import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type {
	PromptCatalogEntry,
  PromptSkillSummary,
  PromptSkillVersions,
  PromptTrace,
} from "../types";

export interface PromptPreviewResponse {
  skill_id: string;
  with_scope: boolean;
  variables: Record<string, string>;
  prompt: string;
}

export interface ExperimentResultsData {
  experimentId: string;
  skillId?: string;
  status?: string;
  variants: {
    variantId: string;
    totalRuns: number;
    readyCount: number;
    needsWorkCount: number;
    fixupRate: number;
    avgDurationSecs?: number;
  }[];
  totalOutcomes: number;
  analyzedAt: string;
}

export interface IPromptService {
  listCatalog(): Promise<PromptCatalogEntry[]>;
  listSkills(): Promise<PromptSkillSummary[]>;
  getSkill(skillId: string): Promise<PromptSkillSummary>;
  updateSkill(
    skillId: string,
    patch: {
      content?: string;
      name?: string;
      description?: string;
      default_scope?: string;
      draft?: boolean;
    }
  ): Promise<PromptSkillSummary>;
  getSkillVersions(skillId: string): Promise<PromptSkillVersions>;
  revertSkillVersion(skillId: string, version: number): Promise<PromptSkillSummary>;
  preview(skillId: string, variables: Record<string, string>, withScope?: boolean): Promise<PromptPreviewResponse>;
  getExecutionPromptTrace(executionId: string): Promise<PromptTrace>;
  getExperimentResults(experimentId: string): Promise<ExperimentResultsData>;
}

export function createPromptService(apiClient: IApiClient = defaultApiClient): IPromptService {
  return {
    async listCatalog(): Promise<PromptCatalogEntry[]> {
      const data = await apiClient.get<{ items?: PromptCatalogEntry[] }>(API_ENDPOINTS.promptsCatalog);
      return data.items ?? [];
    },

    async listSkills(): Promise<PromptSkillSummary[]> {
      const data = await apiClient.get<{ items?: PromptSkillSummary[] }>(API_ENDPOINTS.promptSkills);
      return data.items ?? [];
    },

    async getSkill(skillId: string): Promise<PromptSkillSummary> {
      const data = await apiClient.get<{ item?: PromptSkillSummary }>(API_ENDPOINTS.promptSkillById(skillId));
      if (!data.item) {
        throw new Error("Prompt skill not found");
      }
      return data.item;
    },

    async updateSkill(
      skillId: string,
      patch: {
        content?: string;
        name?: string;
        description?: string;
        default_scope?: string;
        draft?: boolean;
      }
    ): Promise<PromptSkillSummary> {
      const data = await apiClient.put<{ item?: PromptSkillSummary }>(API_ENDPOINTS.promptSkillById(skillId), patch);
      if (!data.item) {
        throw new Error("Prompt skill update failed");
      }
      return data.item;
    },

    async getSkillVersions(skillId: string): Promise<PromptSkillVersions> {
      return apiClient.get<PromptSkillVersions>(API_ENDPOINTS.promptSkillVersions(skillId));
    },

    async revertSkillVersion(skillId: string, version: number): Promise<PromptSkillSummary> {
      const data = await apiClient.post<{ item?: PromptSkillSummary }>(
        API_ENDPOINTS.promptSkillRevert(skillId, version),
        {}
      );
      if (!data.item) {
        throw new Error("Prompt skill revert failed");
      }
      return data.item;
    },

    async preview(
      skillId: string,
      variables: Record<string, string>,
      withScope = true
    ): Promise<PromptPreviewResponse> {
      return apiClient.post<PromptPreviewResponse>(API_ENDPOINTS.promptsPreview, {
        skill_id: skillId,
        variables,
        with_scope: withScope,
      });
    },

    async getExecutionPromptTrace(executionId: string): Promise<PromptTrace> {
      const data = await apiClient.get<{ trace?: PromptTrace }>(API_ENDPOINTS.executionPromptTrace(executionId));
      if (!data.trace) {
        throw new Error("Prompt trace not found");
      }
      return data.trace;
    },

    async getExperimentResults(experimentId: string): Promise<ExperimentResultsData> {
      return apiClient.get<ExperimentResultsData>(API_ENDPOINTS.promptExperimentResults(experimentId));
    },

  };
}

export const promptService = createPromptService();
