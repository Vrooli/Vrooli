import type { Settings } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import { DeleteConfirmLevel } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import { SettingsResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import type {
  Settings as SettingsDomain,
  DeleteConfirmLevel as DomainDeleteConfirmLevel,
  ExecutionMode,
  ThemePreference,
} from "../../types";
import { EXECUTION_MODES } from "../../types";
import { createProtoSchema } from "./shared";

const executionModeSet = new Set<string>(EXECUTION_MODES);

function isExecutionMode(value: unknown): value is ExecutionMode {
  return typeof value === "string" && executionModeSet.has(value);
}

function isThemePreference(value: unknown): value is ThemePreference {
  return value === "dark" || value === "light" || value === "system";
}

function normalizeThemePreference(value?: string): ThemePreference {
  return isThemePreference(value) ? value : "dark";
}

// Canonical lane defaults — kept in sync with API DefaultSettings and
// settings-service.DEFAULT_SETTINGS. Mirrored here so the proto-mapping
// layer can fill missing keys when the API has not yet written the
// settings file with lane caps.
const DEFAULT_LANE_LIMITS: Record<string, number> = {
  investigate: 6,
  execute: 3,
  review: 8,
  reconcile: 2,
};

function mapLaneConcurrencyLimits(input?: Record<string, number>): Record<string, number> {
  const out: Record<string, number> = { ...DEFAULT_LANE_LIMITS };
  if (!input) return out;
  for (const lane of Object.keys(DEFAULT_LANE_LIMITS)) {
    const val = input[lane];
    if (typeof val === "number" && val > 0) {
      out[lane] = val;
    }
  }
  return out;
}

function mapDeleteConfirmLevel(proto: DeleteConfirmLevel): DomainDeleteConfirmLevel {
  switch (proto) {
    case DeleteConfirmLevel.SIMPLE:
      return "simple";
    case DeleteConfirmLevel.NONE:
      return "none";
    case DeleteConfirmLevel.STRONG:
      return "strong";
    default:
      return "simple";
  }
}

export const settingsResponseSchema = createProtoSchema(
  SettingsResponseSchema,
  "settings"
);

export function mapProtoSettings(protoSettings: Settings): SettingsDomain {
  const mode = isExecutionMode(protoSettings.defaultMode) ? protoSettings.defaultMode : "manual";
  return {
    theme: normalizeThemePreference(protoSettings.theme),
    defaultMode: mode,
    autoFixup: protoSettings.autoFixup ?? false,
    maxFixupAttempts: protoSettings.maxFixupAttempts ?? 0,
    reviewAgentEnabled: protoSettings.reviewAgentEnabled ?? true,
    maxAutoRounds: protoSettings.maxAutoRounds ?? 10,
    autoInitializeWorkshop: protoSettings.autoInitializeWorkshop ?? true,
    autoAdvanceWorkshop: protoSettings.autoAdvanceWorkshop ?? true,
    autoCascadeWorkshop: protoSettings.autoCascadeWorkshop ?? true,
    autoAdvanceDelaySeconds: protoSettings.autoAdvanceDelaySeconds ?? 10,
    agentMaxTurns: protoSettings.agentMaxTurns ?? 600,
    agentTimeoutSeconds: protoSettings.agentTimeoutSeconds ?? 900,
    searchDebounceMs: protoSettings.searchDebounceMs ?? 300,
    toastDurationMs: protoSettings.toastDurationMs ?? 5000,
    deleteConfirmation: {
      backlog: mapDeleteConfirmLevel(protoSettings.deleteConfirmation?.backlog ?? DeleteConfirmLevel.SIMPLE),
      initiative: mapDeleteConfirmLevel(protoSettings.deleteConfirmation?.initiative ?? DeleteConfirmLevel.SIMPLE),
      capture: mapDeleteConfirmLevel(protoSettings.deleteConfirmation?.capture ?? DeleteConfirmLevel.SIMPLE),
    },
    reviewCodeQualityMinScore: protoSettings.reviewCodeQualityMinScore ?? 60,
    reviewTestMinPassRate: protoSettings.reviewTestMinPassRate ?? 1.0,
    reviewMaxBlockingViolations: protoSettings.reviewMaxBlockingViolations ?? 0,
    reviewMaxWarnings: protoSettings.reviewMaxWarnings ?? -1,
    reviewRequireScreenshots: protoSettings.reviewRequireScreenshots ?? true,
    reviewRequireTests: protoSettings.reviewRequireTests ?? true,
    laneConcurrencyLimits: mapLaneConcurrencyLimits(protoSettings.laneConcurrencyLimits),
    maxQueueDepth: protoSettings.maxQueueDepth ?? 50,
    circuitBreakerThreshold: protoSettings.circuitBreakerThreshold ?? 3,
    circuitBreakerCooldownMinutes: protoSettings.circuitBreakerCooldownMinutes ?? 60,
    executionCostCapPerRun: protoSettings.executionCostCapPerRun ?? 0,
    costPerTurnEstimate: protoSettings.costPerTurnEstimate ?? 0.10,
  };
}
