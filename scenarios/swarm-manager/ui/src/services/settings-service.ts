/**
 * Settings Service - Data access layer for settings persistence
 */

import { z } from "zod";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Settings, RecommendationSources, RecommendationAutoSync } from "../types";

const recommendationSourcesSchema = z.object({
  problems: z.boolean().optional(),
  completeness: z.boolean().optional(),
  tests: z.boolean().optional(),
  coverage: z.boolean().optional(),
  customFocus: z.boolean().optional(),
  scenarioNotes: z.boolean().optional(),
});

const recommendationAutoSyncSchema = z.object({
  enabled: z.boolean().optional(),
  interval: z.string().optional(),
  lastRefresh: z.string().optional(),
  nextRefresh: z.string().optional(),
  refreshScope: z.string().optional(),
});

const settingsSchema = z.object({
  settings: z.object({
    theme: z.enum(["dark", "light", "system"]).optional(),
    recommendationMode: z.enum(["off", "suggestions", "yolo"]).optional(),
    customFocus: z.string().optional(),
    insightsEnabled: z.boolean().optional(),
    insightsAutoAnalyze: z.boolean().optional(),
    recommendationSources: recommendationSourcesSchema.optional(),
    recommendationAutoSync: recommendationAutoSyncSchema.optional(),
  }),
});

const DEFAULT_SOURCES: RecommendationSources = {
  problems: true,
  completeness: true,
  tests: true,
  coverage: true,
  customFocus: true,
  scenarioNotes: true,
};

const DEFAULT_AUTOSYNC: RecommendationAutoSync = {
  enabled: false,
  interval: "1h",
  lastRefresh: "",
  nextRefresh: "",
  refreshScope: "manual",
};

const DEFAULT_SETTINGS: Settings = {
  theme: "dark",
  recommendationMode: "off",
  customFocus: "",
  insightsEnabled: false,
  insightsAutoAnalyze: false,
  recommendationSources: DEFAULT_SOURCES,
  recommendationAutoSync: DEFAULT_AUTOSYNC,
};

type SettingsPatch = Omit<Partial<Settings>, "recommendationSources" | "recommendationAutoSync"> & {
  recommendationSources?: Partial<RecommendationSources>;
  recommendationAutoSync?: Partial<RecommendationAutoSync>;
};

function normalizeSettings(input?: SettingsPatch): Settings {
  if (!input) return DEFAULT_SETTINGS;
  return {
    theme: input.theme ?? DEFAULT_SETTINGS.theme,
    recommendationMode: input.recommendationMode ?? DEFAULT_SETTINGS.recommendationMode,
    customFocus: input.customFocus ?? "",
    insightsEnabled: input.insightsEnabled ?? DEFAULT_SETTINGS.insightsEnabled,
    insightsAutoAnalyze: input.insightsAutoAnalyze ?? DEFAULT_SETTINGS.insightsAutoAnalyze,
    recommendationSources: {
      ...DEFAULT_SOURCES,
      ...(input.recommendationSources ?? {}),
    },
    recommendationAutoSync: {
      ...DEFAULT_AUTOSYNC,
      ...(input.recommendationAutoSync ?? {}),
    },
  };
}

export interface ISettingsService {
  get(): Promise<Settings>;
  update(patch: SettingsPatch): Promise<Settings>;
}

export function createSettingsService(apiClient: IApiClient = defaultApiClient): ISettingsService {
  return {
    async get(): Promise<Settings> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.settings);
      const parsed = settingsSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error("Invalid settings response");
      }
      return normalizeSettings(parsed.data.settings);
    },

    async update(patch: SettingsPatch): Promise<Settings> {
      const payload = {
        theme: patch.theme,
        recommendationMode: patch.recommendationMode,
        customFocus: patch.customFocus,
        insightsEnabled: patch.insightsEnabled,
        insightsAutoAnalyze: patch.insightsAutoAnalyze,
        recommendationSources: patch.recommendationSources,
        recommendationAutoSync: patch.recommendationAutoSync,
      };
      const data = await apiClient.put<unknown>(API_ENDPOINTS.settings, payload);
      const parsed = settingsSchema.safeParse(data);
      if (!parsed.success) {
        throw new Error("Invalid settings response");
      }
      return normalizeSettings(parsed.data.settings);
    },
  };
}

export const settingsService = createSettingsService();
