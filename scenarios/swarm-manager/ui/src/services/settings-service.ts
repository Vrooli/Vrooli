/**
 * Settings Service - Data access layer for unified settings persistence
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

export const DEFAULT_SETTINGS: Settings = {
  theme: "dark",
  defaultMode: "manual",
  defaultDelaySeconds: 300,
  autoFixup: false,
  maxFixupAttempts: 2,
  maxAutoRounds: 10,
  agentMaxTurns: 60,
  agentTimeoutSeconds: 900,
  agentRequiresApproval: true,
  searchDebounceMs: 300,
  toastDurationMs: 5000,
  confirmDestructiveActions: true,
};

type SettingsPatch = Partial<Settings>;

function normalizeSettings(input?: SettingsPatch): Settings {
  if (!input) return DEFAULT_SETTINGS;
  return {
    theme: input.theme ?? DEFAULT_SETTINGS.theme,
    defaultMode: input.defaultMode ?? DEFAULT_SETTINGS.defaultMode,
    defaultDelaySeconds: input.defaultDelaySeconds ?? DEFAULT_SETTINGS.defaultDelaySeconds,
    autoFixup: input.autoFixup ?? DEFAULT_SETTINGS.autoFixup,
    maxFixupAttempts: input.maxFixupAttempts ?? DEFAULT_SETTINGS.maxFixupAttempts,
    maxAutoRounds: input.maxAutoRounds ?? DEFAULT_SETTINGS.maxAutoRounds,
    agentMaxTurns: input.agentMaxTurns ?? DEFAULT_SETTINGS.agentMaxTurns,
    agentTimeoutSeconds: input.agentTimeoutSeconds ?? DEFAULT_SETTINGS.agentTimeoutSeconds,
    agentRequiresApproval: input.agentRequiresApproval ?? DEFAULT_SETTINGS.agentRequiresApproval,
    searchDebounceMs: input.searchDebounceMs ?? DEFAULT_SETTINGS.searchDebounceMs,
    toastDurationMs: input.toastDurationMs ?? DEFAULT_SETTINGS.toastDurationMs,
    confirmDestructiveActions: input.confirmDestructiveActions ?? DEFAULT_SETTINGS.confirmDestructiveActions,
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
        ...(patch.theme !== undefined ? { theme: patch.theme } : {}),
        ...(patch.defaultMode !== undefined ? { defaultMode: patch.defaultMode } : {}),
        ...(patch.defaultDelaySeconds !== undefined ? { defaultDelaySeconds: BigInt(patch.defaultDelaySeconds) } : {}),
        ...(patch.autoFixup !== undefined ? { autoFixup: patch.autoFixup } : {}),
        ...(patch.maxFixupAttempts !== undefined ? { maxFixupAttempts: patch.maxFixupAttempts } : {}),
        ...(patch.maxAutoRounds !== undefined ? { maxAutoRounds: patch.maxAutoRounds } : {}),
        ...(patch.agentMaxTurns !== undefined ? { agentMaxTurns: patch.agentMaxTurns } : {}),
        ...(patch.agentTimeoutSeconds !== undefined ? { agentTimeoutSeconds: patch.agentTimeoutSeconds } : {}),
        ...(patch.agentRequiresApproval !== undefined ? { agentRequiresApproval: patch.agentRequiresApproval } : {}),
        ...(patch.searchDebounceMs !== undefined ? { searchDebounceMs: patch.searchDebounceMs } : {}),
        ...(patch.toastDurationMs !== undefined ? { toastDurationMs: patch.toastDurationMs } : {}),
        ...(patch.confirmDestructiveActions !== undefined ? { confirmDestructiveActions: patch.confirmDestructiveActions } : {}),
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
