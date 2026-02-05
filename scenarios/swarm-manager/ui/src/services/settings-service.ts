/**
 * Settings Service - Data access layer for settings persistence
 */

import { UpdateSettingsRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Settings, RecommendationSources, RecommendationAutoSync } from "../types";
import {
  buildMessage,
  mapProtoSettings,
  parseProtoResponse,
  requireProtoField,
  settingsResponseSchema,
  toProtoJson,
} from "./proto-contracts";

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
      const parsed = parseProtoResponse(settingsResponseSchema, data, "settings");
      return normalizeSettings(mapProtoSettings(requireProtoField(parsed.settings, "settings")));
    },

    async update(patch: SettingsPatch): Promise<Settings> {
      const message = buildMessage(UpdateSettingsRequestSchema, {
        theme: patch.theme,
        recommendationMode: patch.recommendationMode,
        customFocus: patch.customFocus,
        insightsEnabled: patch.insightsEnabled,
        insightsAutoAnalyze: patch.insightsAutoAnalyze,
        recommendationSources: patch.recommendationSources
          ? {
              problems: patch.recommendationSources.problems,
              completeness: patch.recommendationSources.completeness,
              tests: patch.recommendationSources.tests,
              coverage: patch.recommendationSources.coverage,
              customFocus: patch.recommendationSources.customFocus,
              scenarioNotes: patch.recommendationSources.scenarioNotes,
            }
          : undefined,
        recommendationAutoSync: patch.recommendationAutoSync
          ? {
              enabled: patch.recommendationAutoSync.enabled,
              interval: patch.recommendationAutoSync.interval,
              lastRefresh: patch.recommendationAutoSync.lastRefresh,
              nextRefresh: patch.recommendationAutoSync.nextRefresh,
              refreshScope: patch.recommendationAutoSync.refreshScope,
            }
          : undefined,
      });
      const data = await apiClient.put<unknown>(
        API_ENDPOINTS.settings,
        toProtoJson(UpdateSettingsRequestSchema, message)
      );
      const parsed = parseProtoResponse(settingsResponseSchema, data, "settings");
      return normalizeSettings(mapProtoSettings(requireProtoField(parsed.settings, "settings")));
    },
  };
}

export const settingsService = createSettingsService();
