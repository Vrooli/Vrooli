/**
 * Settings Service - Data access layer for settings persistence
 */

import { UpdateSettingsRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import type { IApiClient } from "../lib/api-client";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Settings } from "../types";
import {
  buildMessage,
  mapProtoSettings,
  parseProtoResponse,
  requireProtoField,
  settingsResponseSchema,
  toProtoJson,
} from "./proto-contracts";

const DEFAULT_SETTINGS: Settings = {
  theme: "dark",
  customFocus: "",
  insightsEnabled: false,
  insightsAutoAnalyze: false,
};

type SettingsPatch = Partial<Settings>;

function normalizeSettings(input?: SettingsPatch): Settings {
  if (!input) return DEFAULT_SETTINGS;
  return {
    theme: input.theme ?? DEFAULT_SETTINGS.theme,
    customFocus: input.customFocus ?? "",
    insightsEnabled: input.insightsEnabled ?? DEFAULT_SETTINGS.insightsEnabled,
    insightsAutoAnalyze: input.insightsAutoAnalyze ?? DEFAULT_SETTINGS.insightsAutoAnalyze,
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
        customFocus: patch.customFocus,
        insightsEnabled: patch.insightsEnabled,
        insightsAutoAnalyze: patch.insightsAutoAnalyze,
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
