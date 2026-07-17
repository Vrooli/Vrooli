import type { Settings } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import { DeleteConfirmLevel } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import { SettingsFieldRole as ProtoSettingsFieldRole, SettingsResponseSchema } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import type { SettingsPolicyProjection as ProtoSettingsPolicyProjection } from "@vrooli/proto-types/swarm-manager/v1/api/settings_pb";
import type {
  Settings as SettingsDomain,
  DeleteConfirmLevel as DomainDeleteConfirmLevel,
  DeleteConfirmationSettings,
  AutoFilerMode,
  AutoFilerSettings,
  AutoFilerStrategy,
  ExecutionMode,
  FixBeforeFeatureMode,
  SettingsFieldRole as DomainSettingsFieldRole,
  SettingsPolicyProjection,
  ThemePreference,
} from "../../types";
import { EXECUTION_MODES } from "../../types";
import { defaultDeleteConfirmationLevels } from "../../lib/deletable-entities";
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

// Map the proto delete-confirmation map (entity-type → proto enum) to the
// domain record, filling every known registry key from defaults and
// preserving any unknown keys a newer API may send.
function mapDeleteConfirmationLevels(
  input?: Record<string, DeleteConfirmLevel>,
): DeleteConfirmationSettings {
  const out = defaultDeleteConfirmationLevels() as Record<string, DomainDeleteConfirmLevel>;
  if (input) {
    for (const [key, value] of Object.entries(input)) {
      out[key] = mapDeleteConfirmLevel(value);
    }
  }
  return out as DeleteConfirmationSettings;
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
    deleteConfirmation: mapDeleteConfirmationLevels(protoSettings.deleteConfirmationLevels),
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
    fixBeforeFeature: normalizeFixBeforeFeature(protoSettings.fixBeforeFeature),
    autoFiler: mapAutoFilerSettings(protoSettings.autoFiler),
  };
}

function mapProtoSettingsFieldRole(role: ProtoSettingsFieldRole | undefined): DomainSettingsFieldRole {
  switch (role) {
    case ProtoSettingsFieldRole.USER_PREFERENCE:
      return "user_preference";
    case ProtoSettingsFieldRole.POLICY_CONTROL:
      return "policy_control";
    case ProtoSettingsFieldRole.GOVERNANCE:
      return "governance";
    case ProtoSettingsFieldRole.DORMANT:
      return "dormant";
    default:
      return "unspecified";
  }
}

/**
 * Map the proto settings → policy-controls projection. Returns null when the
 * API predates the projection (older server); callers treat null as "no
 * policy metadata available" and fall back to static labeling.
 */
export function mapProtoPolicyProjection(
  proto: ProtoSettingsPolicyProjection | undefined,
): SettingsPolicyProjection | null {
  if (!proto?.effectiveControls) return null;
  const c = proto.effectiveControls;
  return {
    effectiveControls: {
      defaultMode: isExecutionMode(c.defaultMode) ? c.defaultMode : "manual",
      autoInitialize: c.autoInitialize ?? false,
      autoAdvanceEnabled: c.autoAdvanceEnabled ?? false,
      cascadeEnabled: c.cascadeEnabled ?? false,
      autoAdvanceDelaySeconds: c.autoAdvanceDelaySeconds ?? 0,
      maxAutoRounds: c.maxAutoRounds ?? 0,
      autoFixup: c.autoFixup ?? false,
      maxFixupAttempts: c.maxFixupAttempts ?? 0,
      reviewAgentEnabled: c.reviewAgentEnabled ?? false,
      reviewCodeQualityMinScore: c.reviewCodeQualityMinScore ?? 0,
      reviewTestMinPassRate: c.reviewTestMinPassRate ?? 0,
      reviewMaxBlockingViolations: c.reviewMaxBlockingViolations ?? 0,
      reviewMaxWarnings: c.reviewMaxWarnings ?? -1,
      reviewRequireScreenshots: c.reviewRequireScreenshots ?? false,
      reviewRequireTests: c.reviewRequireTests ?? false,
      agentMaxTurns: c.agentMaxTurns ?? 0,
      agentTimeoutSeconds: c.agentTimeoutSeconds ?? 0,
    },
    classifications: (proto.classifications ?? []).map((entry) => ({
      field: entry.field ?? "",
      role: mapProtoSettingsFieldRole(entry.role),
      control: entry.control ?? "",
      note: entry.note ?? "",
    })),
  };
}

function normalizeFixBeforeFeature(value: string | undefined): FixBeforeFeatureMode {
  return value === "off" || value === "block" ? value : "suggest";
}

function normalizeAutoFilerMode(value: string | undefined): AutoFilerMode {
  return value === "auto_add" ? value : "suggest";
}

function normalizeAutoFilerStrategy(value: string | undefined): AutoFilerStrategy {
  return value === "importance" ? value : "feature_pending";
}

type ProtoAutoFilerSettingsLike = {
  enabled?: boolean;
  mode?: string;
  strategy?: string;
  maxOpenAutoFiled?: number;
  velocityWindowDays?: number;
  minVelocityTransitions?: number;
  intervalMinutes?: number;
  goalName?: string;
};

function mapAutoFilerSettings(input?: ProtoAutoFilerSettingsLike): AutoFilerSettings {
  return {
    enabled: input?.enabled ?? false,
    mode: normalizeAutoFilerMode(input?.mode),
    strategy: normalizeAutoFilerStrategy(input?.strategy),
    maxOpenAutoFiled: input?.maxOpenAutoFiled ?? 10,
    velocityWindowDays: input?.velocityWindowDays ?? 7,
    minVelocityTransitions: input?.minVelocityTransitions ?? 1,
    intervalMinutes: input?.intervalMinutes ?? 30,
    goalName: input?.goalName?.trim() || "automated-maintenance",
  };
}
