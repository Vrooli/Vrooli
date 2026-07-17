/**
 * Settings domain types.
 */

import type { Settings as ProtoSettings } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import type { BacklogResearchResponse as ProtoBacklogResearchResponse } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
import type { DeletableEntityType } from "../lib/deletable-entities";
import type { ExecutionMode } from "./execution";
import type { ProtoMessage } from "./shared";

/**
 * Theme preference for the UI.
 */
export type ThemePreference = "dark" | "light" | "system";

/**
 * Confirmation level for delete operations.
 */
export type DeleteConfirmLevel = "none" | "simple" | "strong";

/**
 * Per-entity-type delete confirmation settings. Keyed by the registry's
 * DeletableEntityType so adding an entity type does not change this shape.
 */
export type DeleteConfirmationSettings = Record<DeletableEntityType, DeleteConfirmLevel>;

/**
 * User preferences and configuration (unified settings including execution defaults).
 */
export type Settings = Omit<
  ProtoMessage<ProtoSettings>,
  "theme" | "defaultDelaySeconds" | "maxFixupAttempts" | "maxAutoRounds" | "autoAdvanceDelaySeconds" | "agentMaxTurns" | "agentTimeoutSeconds" | "searchDebounceMs" | "toastDurationMs" | "reviewMaxBlockingViolations" | "reviewMaxWarnings" | "laneConcurrencyLimits" | "maxQueueDepth" | "circuitBreakerThreshold" | "circuitBreakerCooldownMinutes" | "executionCostCapPerRun" | "costPerTurnEstimate" | "deleteConfirmationLevels" | "fixBeforeFeature" | "autoFiler"
> & {
  /** UI theme preference */
  theme: ThemePreference;
  /** Execution defaults */
  defaultMode: ExecutionMode;
  autoFixup: boolean;
  maxFixupAttempts: number;
  reviewAgentEnabled: boolean;
  /** Workshop */
  autoInitializeWorkshop: boolean;
  autoAdvanceWorkshop: boolean;
  autoCascadeWorkshop: boolean;
  maxAutoRounds: number;
  autoAdvanceDelaySeconds: number;
  /** Agent behavior */
  agentMaxTurns: number;
  agentTimeoutSeconds: number;
  /** UI preferences */
  searchDebounceMs: number;
  toastDurationMs: number;
  deleteConfirmation: DeleteConfirmationSettings;
  /** Review thresholds */
  reviewCodeQualityMinScore: number;
  reviewTestMinPassRate: number;
  reviewMaxBlockingViolations: number;
  reviewMaxWarnings: number;
  reviewRequireScreenshots: boolean;
  reviewRequireTests: boolean;
  /**
   * Concurrency and governance.
   *
   * laneConcurrencyLimits caps simultaneous tracked agent activity by
   * phase-kind lane. Keys are lane names matching the API's
   * `agentactivity.Lane`: `investigate`, `execute`, `review`, `reconcile`.
   */
  laneConcurrencyLimits: Record<string, number>;
  maxQueueDepth: number;
  circuitBreakerThreshold: number;
  circuitBreakerCooldownMinutes: number;
  executionCostCapPerRun: number;
  costPerTurnEstimate: number;
  /**
   * Fix-before-feature gate. "off" | "suggest" (default) | "block": when a
   * feature item is queued onto a scenario with open fix/chore work, advise
   * or block.
   */
  fixBeforeFeature: FixBeforeFeatureMode;
  /** Governed automatic backlog filing for maintenance findings. */
  autoFiler: AutoFilerSettings;
};

/**
 * Fix-before-feature gate modes.
 */
export type FixBeforeFeatureMode = "off" | "suggest" | "block";

export type AutoFilerMode = "suggest" | "auto_add";

export type AutoFilerStrategy = "feature_pending" | "importance";

export interface AutoFilerSettings {
  enabled: boolean;
  mode: AutoFilerMode;
  strategy: AutoFilerStrategy;
  maxOpenAutoFiled: number;
  velocityWindowDays: number;
  minVelocityTransitions: number;
  intervalMinutes: number;
  goalName: string;
}

/**
 * Research agent spawn response
 */
export type ResearchResponse = ProtoMessage<ProtoBacklogResearchResponse>;

/**
 * Role of a persisted settings field in the declarative agent-operations
 * model (mirrors proto SettingsFieldRole).
 */
export type SettingsFieldRole =
  | "unspecified"
  | "user_preference"
  | "policy_control"
  | "governance"
  | "dormant";

/**
 * Classification of one settings field: its role plus (for policy controls)
 * the destination path inside the PolicyControls projection.
 */
export interface SettingsFieldClassification {
  field: string;
  role: SettingsFieldRole;
  control: string;
  note: string;
}

/**
 * Effective policy controls derived from current settings — the values the
 * operation runner's transition-policy consumers read.
 */
export interface PolicyControlsView {
  defaultMode: ExecutionMode;
  autoInitialize: boolean;
  autoAdvanceEnabled: boolean;
  cascadeEnabled: boolean;
  autoAdvanceDelaySeconds: number;
  maxAutoRounds: number;
  autoFixup: boolean;
  maxFixupAttempts: number;
  reviewAgentEnabled: boolean;
  reviewCodeQualityMinScore: number;
  reviewTestMinPassRate: number;
  reviewMaxBlockingViolations: number;
  reviewMaxWarnings: number;
  reviewRequireScreenshots: boolean;
  reviewRequireTests: boolean;
  agentMaxTurns: number;
  agentTimeoutSeconds: number;
}

/**
 * Public settings → policy-controls projection served alongside Settings so
 * the UI can label which controls are policy-level vs user preference.
 */
export interface SettingsPolicyProjection {
  effectiveControls: PolicyControlsView;
  classifications: SettingsFieldClassification[];
}
