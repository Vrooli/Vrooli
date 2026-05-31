/**
 * Settings Service - Data access layer for unified settings persistence
 */

import { UpdateSettingsRequestSchema } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import { DeleteConfirmLevel } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import type { IApiClient } from "../lib/api-client";
import type { DeleteConfirmLevel as DomainDeleteConfirmLevel } from "../types/settings";
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
  autoInitializeWorkshop: true,
  autoAdvanceWorkshop: true,
  autoCascadeWorkshop: true,
  maxAutoRounds: 10,
  autoAdvanceDelaySeconds: 10,
  agentMaxTurns: 600,
  agentTimeoutSeconds: 900,
  searchDebounceMs: 300,
  toastDurationMs: 5000,
  deleteConfirmation: { backlog: "simple", initiative: "strong", capture: "none" },
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
  fixBeforeFeatureDiscovery: false,
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

function normalizeSettings(input?: SettingsPatch): Settings {
  if (!input) return DEFAULT_SETTINGS;
  return {
    theme: input.theme ?? DEFAULT_SETTINGS.theme,
    defaultMode: input.defaultMode ?? DEFAULT_SETTINGS.defaultMode,
    autoFixup: input.autoFixup ?? DEFAULT_SETTINGS.autoFixup,
    maxFixupAttempts: input.maxFixupAttempts ?? DEFAULT_SETTINGS.maxFixupAttempts,
    reviewAgentEnabled: input.reviewAgentEnabled ?? DEFAULT_SETTINGS.reviewAgentEnabled,
    autoInitializeWorkshop: input.autoInitializeWorkshop ?? DEFAULT_SETTINGS.autoInitializeWorkshop,
    autoAdvanceWorkshop: input.autoAdvanceWorkshop ?? DEFAULT_SETTINGS.autoAdvanceWorkshop,
    autoCascadeWorkshop: input.autoCascadeWorkshop ?? DEFAULT_SETTINGS.autoCascadeWorkshop,
    maxAutoRounds: input.maxAutoRounds ?? DEFAULT_SETTINGS.maxAutoRounds,
    autoAdvanceDelaySeconds: input.autoAdvanceDelaySeconds ?? DEFAULT_SETTINGS.autoAdvanceDelaySeconds,
    agentMaxTurns: input.agentMaxTurns ?? DEFAULT_SETTINGS.agentMaxTurns,
    agentTimeoutSeconds: input.agentTimeoutSeconds ?? DEFAULT_SETTINGS.agentTimeoutSeconds,
    searchDebounceMs: input.searchDebounceMs ?? DEFAULT_SETTINGS.searchDebounceMs,
    toastDurationMs: input.toastDurationMs ?? DEFAULT_SETTINGS.toastDurationMs,
    deleteConfirmation: {
      backlog: input.deleteConfirmation?.backlog ?? DEFAULT_SETTINGS.deleteConfirmation.backlog,
      initiative: input.deleteConfirmation?.initiative ?? DEFAULT_SETTINGS.deleteConfirmation.initiative,
      capture: input.deleteConfirmation?.capture ?? DEFAULT_SETTINGS.deleteConfirmation.capture,
    },
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
    fixBeforeFeatureDiscovery: input.fixBeforeFeatureDiscovery ?? DEFAULT_SETTINGS.fixBeforeFeatureDiscovery,
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
        ...(patch.autoFixup !== undefined ? { autoFixup: patch.autoFixup } : {}),
        ...(patch.maxFixupAttempts !== undefined ? { maxFixupAttempts: patch.maxFixupAttempts } : {}),
        ...(patch.reviewAgentEnabled !== undefined ? { reviewAgentEnabled: patch.reviewAgentEnabled } : {}),
        ...(patch.autoInitializeWorkshop !== undefined ? { autoInitializeWorkshop: patch.autoInitializeWorkshop } : {}),
        ...(patch.autoAdvanceWorkshop !== undefined ? { autoAdvanceWorkshop: patch.autoAdvanceWorkshop } : {}),
        ...(patch.autoCascadeWorkshop !== undefined ? { autoCascadeWorkshop: patch.autoCascadeWorkshop } : {}),
        ...(patch.maxAutoRounds !== undefined ? { maxAutoRounds: patch.maxAutoRounds } : {}),
        ...(patch.agentMaxTurns !== undefined ? { agentMaxTurns: patch.agentMaxTurns } : {}),
        ...(patch.agentTimeoutSeconds !== undefined ? { agentTimeoutSeconds: patch.agentTimeoutSeconds } : {}),
        ...(patch.searchDebounceMs !== undefined ? { searchDebounceMs: patch.searchDebounceMs } : {}),
        ...(patch.toastDurationMs !== undefined ? { toastDurationMs: patch.toastDurationMs } : {}),
        ...(patch.deleteConfirmation !== undefined ? {
          deleteConfirmation: {
            backlog: domainToProtoDeleteConfirmLevel(patch.deleteConfirmation.backlog),
            initiative: domainToProtoDeleteConfirmLevel(patch.deleteConfirmation.initiative),
            capture: domainToProtoDeleteConfirmLevel(patch.deleteConfirmation.capture),
          },
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
        ...(patch.fixBeforeFeatureDiscovery !== undefined ? { fixBeforeFeatureDiscovery: patch.fixBeforeFeatureDiscovery } : {}),
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
