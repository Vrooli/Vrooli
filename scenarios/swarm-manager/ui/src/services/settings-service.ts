/**
 * Settings Service - Data access layer for unified settings persistence
 */

import { UpdateSettingsRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import { DeleteConfirmLevel } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import type { IApiClient } from "../lib/api-client";
import type {
  AutoFilerSettings,
  DeleteConfirmLevel as DomainDeleteConfirmLevel,
  DeleteConfirmationSettings,
} from "../types/settings";
import { defaultDeleteConfirmationLevels } from "../lib/deletable-entities";
import { defaultApiClient } from "../lib/api-client";
import { API_ENDPOINTS } from "../lib/api-endpoints";
import type { Settings } from "../types";
import type { SettingsPolicyProjection } from "../types/settings";
import {
  buildMessage,
  mapProtoPolicyProjection,
  mapProtoSettings,
  parseProtoResponse,
  requireProtoField,
  settingsResponseSchema,
  toProtoJson,
} from "./proto-contracts";

function domainToProtoDeleteConfirmLevel(level: DomainDeleteConfirmLevel): DeleteConfirmLevel {
  switch (level) {
    case "none":
      return DeleteConfirmLevel.NONE;
    case "strong":
      return DeleteConfirmLevel.STRONG;
    default:
      return DeleteConfirmLevel.SIMPLE;
  }
}

export const DEFAULT_SETTINGS: Settings = {
  theme: "dark",
  defaultMode: "manual",
  autoFixup: false,
  maxFixupAttempts: 2,
  reviewAgentEnabled: true,
  agentMaxTurns: 600,
  agentTimeoutSeconds: 900,
  searchDebounceMs: 300,
  toastDurationMs: 5000,
  deleteConfirmation: defaultDeleteConfirmationLevels(),
  reviewCodeQualityMinScore: 60,
  reviewTestMinPassRate: 1.0,
  reviewMaxBlockingViolations: 0,
  reviewMaxWarnings: -1,
  reviewRequireScreenshots: true,
  reviewRequireTests: true,
  laneConcurrencyLimits: {
    investigate: 6,
    execute: 3,
    review: 8,
    reconcile: 2,
  },
  maxQueueDepth: 50,
  circuitBreakerThreshold: 3,
  circuitBreakerCooldownMinutes: 60,
  executionCostCapPerRun: 0,
  costPerTurnEstimate: 0.10,
  fixBeforeFeature: "suggest",
  autoFiler: {
    enabled: false,
    mode: "suggest",
    strategy: "feature_pending",
    maxOpenAutoFiled: 10,
    velocityWindowDays: 7,
    minVelocityTransitions: 1,
    intervalMinutes: 30,
    goalName: "automated-maintenance",
  },
};

type SettingsPatch = Partial<Settings>;

/**
 * Fill any missing canonical lane keys from DEFAULT_SETTINGS so the four
 * lanes are always present in the rendered Settings shape. The API does
 * the same on its side; this is the UI guard.
 */
function normalizeLaneLimits(input?: Record<string, number>): Record<string, number> {
  const defaults = DEFAULT_SETTINGS.laneConcurrencyLimits;
  const out: Record<string, number> = { ...defaults };
  if (!input) return out;
  for (const lane of Object.keys(defaults)) {
    const val = input[lane];
    if (typeof val === "number" && val > 0) {
      out[lane] = val;
    }
  }
  return out;
}

/**
 * Fill every known deletable entity key from registry defaults, overridden by
 * any provided value. Unknown keys (e.g. from a newer API) are preserved so an
 * older UI does not silently drop them on the next save.
 */
function normalizeDeleteConfirmation(
  input?: Partial<Record<string, DomainDeleteConfirmLevel>>,
): DeleteConfirmationSettings {
  const out = defaultDeleteConfirmationLevels() as Record<string, DomainDeleteConfirmLevel>;
  if (input) {
    for (const [key, value] of Object.entries(input)) {
      if (value) out[key] = value;
    }
  }
  return out as DeleteConfirmationSettings;
}

function normalizeAutoFiler(input?: Partial<AutoFilerSettings>): AutoFilerSettings {
  const defaults = DEFAULT_SETTINGS.autoFiler;
  if (!input) return { ...defaults };
  return {
    enabled: input.enabled ?? defaults.enabled,
    mode: input.mode === "auto_add" ? "auto_add" : defaults.mode,
    strategy: input.strategy === "importance" ? "importance" : defaults.strategy,
    maxOpenAutoFiled: input.maxOpenAutoFiled ?? defaults.maxOpenAutoFiled,
    velocityWindowDays: input.velocityWindowDays ?? defaults.velocityWindowDays,
    minVelocityTransitions: input.minVelocityTransitions ?? defaults.minVelocityTransitions,
    intervalMinutes: input.intervalMinutes ?? defaults.intervalMinutes,
    goalName: input.goalName?.trim() || defaults.goalName,
  };
}

function normalizeSettings(input?: SettingsPatch): Settings {
  if (!input) return DEFAULT_SETTINGS;
  return {
    theme: input.theme ?? DEFAULT_SETTINGS.theme,
    defaultMode: input.defaultMode ?? DEFAULT_SETTINGS.defaultMode,
    autoFixup: input.autoFixup ?? DEFAULT_SETTINGS.autoFixup,
    maxFixupAttempts: input.maxFixupAttempts ?? DEFAULT_SETTINGS.maxFixupAttempts,
    reviewAgentEnabled: input.reviewAgentEnabled ?? DEFAULT_SETTINGS.reviewAgentEnabled,
    agentMaxTurns: input.agentMaxTurns ?? DEFAULT_SETTINGS.agentMaxTurns,
    agentTimeoutSeconds: input.agentTimeoutSeconds ?? DEFAULT_SETTINGS.agentTimeoutSeconds,
    searchDebounceMs: input.searchDebounceMs ?? DEFAULT_SETTINGS.searchDebounceMs,
    toastDurationMs: input.toastDurationMs ?? DEFAULT_SETTINGS.toastDurationMs,
    deleteConfirmation: normalizeDeleteConfirmation(input.deleteConfirmation),
    reviewCodeQualityMinScore: input.reviewCodeQualityMinScore ?? DEFAULT_SETTINGS.reviewCodeQualityMinScore,
    reviewTestMinPassRate: input.reviewTestMinPassRate ?? DEFAULT_SETTINGS.reviewTestMinPassRate,
    reviewMaxBlockingViolations: input.reviewMaxBlockingViolations ?? DEFAULT_SETTINGS.reviewMaxBlockingViolations,
    reviewMaxWarnings: input.reviewMaxWarnings ?? DEFAULT_SETTINGS.reviewMaxWarnings,
    reviewRequireScreenshots: input.reviewRequireScreenshots ?? DEFAULT_SETTINGS.reviewRequireScreenshots,
    reviewRequireTests: input.reviewRequireTests ?? DEFAULT_SETTINGS.reviewRequireTests,
    laneConcurrencyLimits: normalizeLaneLimits(input.laneConcurrencyLimits),
    maxQueueDepth: input.maxQueueDepth ?? DEFAULT_SETTINGS.maxQueueDepth,
    circuitBreakerThreshold: input.circuitBreakerThreshold ?? DEFAULT_SETTINGS.circuitBreakerThreshold,
    circuitBreakerCooldownMinutes: input.circuitBreakerCooldownMinutes ?? DEFAULT_SETTINGS.circuitBreakerCooldownMinutes,
    executionCostCapPerRun: input.executionCostCapPerRun ?? DEFAULT_SETTINGS.executionCostCapPerRun,
    costPerTurnEstimate: input.costPerTurnEstimate ?? DEFAULT_SETTINGS.costPerTurnEstimate,
    fixBeforeFeature: input.fixBeforeFeature ?? DEFAULT_SETTINGS.fixBeforeFeature,
    autoFiler: normalizeAutoFiler(input.autoFiler),
  };
}

export interface ISettingsService {
  get(): Promise<Settings>;
  update(patch: SettingsPatch): Promise<Settings>;
  /**
   * Fetch the settings → policy-controls projection: which settings are
   * policy-level (govern the operation runner's transition policies) vs pure
   * user preference, plus the effective control values. Returns null when the
   * API does not serve the projection yet.
   */
  getPolicyProjection(): Promise<SettingsPolicyProjection | null>;
}

export function createSettingsService(apiClient: IApiClient = defaultApiClient): ISettingsService {
  return {
    async get(): Promise<Settings> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.settings);
      const parsed = parseProtoResponse(settingsResponseSchema, data, "settings");
      return normalizeSettings(mapProtoSettings(requireProtoField(parsed.settings, "settings")));
    },

    async getPolicyProjection(): Promise<SettingsPolicyProjection | null> {
      const data = await apiClient.get<unknown>(API_ENDPOINTS.settings);
      const parsed = parseProtoResponse(settingsResponseSchema, data, "settings");
      return mapProtoPolicyProjection(parsed.policyProjection);
    },

    async update(patch: SettingsPatch): Promise<Settings> {
      const message = buildMessage(UpdateSettingsRequestSchema, {
        ...(patch.theme !== undefined ? { theme: patch.theme } : {}),
        ...(patch.defaultMode !== undefined ? { defaultMode: patch.defaultMode } : {}),
        ...(patch.autoFixup !== undefined ? { autoFixup: patch.autoFixup } : {}),
        ...(patch.maxFixupAttempts !== undefined ? { maxFixupAttempts: patch.maxFixupAttempts } : {}),
        ...(patch.reviewAgentEnabled !== undefined ? { reviewAgentEnabled: patch.reviewAgentEnabled } : {}),
        ...(patch.agentMaxTurns !== undefined ? { agentMaxTurns: patch.agentMaxTurns } : {}),
        ...(patch.agentTimeoutSeconds !== undefined ? { agentTimeoutSeconds: patch.agentTimeoutSeconds } : {}),
        ...(patch.searchDebounceMs !== undefined ? { searchDebounceMs: patch.searchDebounceMs } : {}),
        ...(patch.toastDurationMs !== undefined ? { toastDurationMs: patch.toastDurationMs } : {}),
        ...(patch.deleteConfirmation !== undefined ? {
          deleteConfirmationLevels: Object.fromEntries(
            Object.entries(patch.deleteConfirmation).map(([key, level]) => [
              key,
              domainToProtoDeleteConfirmLevel(level),
            ]),
          ),
        } : {}),
        ...(patch.reviewCodeQualityMinScore !== undefined ? { reviewCodeQualityMinScore: patch.reviewCodeQualityMinScore } : {}),
        ...(patch.reviewTestMinPassRate !== undefined ? { reviewTestMinPassRate: patch.reviewTestMinPassRate } : {}),
        ...(patch.reviewMaxBlockingViolations !== undefined ? { reviewMaxBlockingViolations: patch.reviewMaxBlockingViolations } : {}),
        ...(patch.reviewMaxWarnings !== undefined ? { reviewMaxWarnings: patch.reviewMaxWarnings } : {}),
        ...(patch.reviewRequireScreenshots !== undefined ? { reviewRequireScreenshots: patch.reviewRequireScreenshots } : {}),
        ...(patch.reviewRequireTests !== undefined ? { reviewRequireTests: patch.reviewRequireTests } : {}),
        ...(patch.laneConcurrencyLimits !== undefined ? { laneConcurrencyLimits: patch.laneConcurrencyLimits } : {}),
        ...(patch.maxQueueDepth !== undefined ? { maxQueueDepth: patch.maxQueueDepth } : {}),
        ...(patch.circuitBreakerThreshold !== undefined ? { circuitBreakerThreshold: patch.circuitBreakerThreshold } : {}),
        ...(patch.circuitBreakerCooldownMinutes !== undefined ? { circuitBreakerCooldownMinutes: patch.circuitBreakerCooldownMinutes } : {}),
        ...(patch.executionCostCapPerRun !== undefined ? { executionCostCapPerRun: patch.executionCostCapPerRun } : {}),
        ...(patch.costPerTurnEstimate !== undefined ? { costPerTurnEstimate: patch.costPerTurnEstimate } : {}),
        ...(patch.fixBeforeFeature !== undefined ? { fixBeforeFeature: patch.fixBeforeFeature } : {}),
        ...(patch.autoFiler !== undefined ? { autoFiler: patch.autoFiler } : {}),
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
