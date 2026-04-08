/**
 * Settings domain types.
 */

import type { Settings as ProtoSettings } from "@vrooli/proto-types/swarm-manager/v1/domain/settings_pb";
import type { BacklogResearchResponse as ProtoBacklogResearchResponse } from "@vrooli/proto-types/swarm-manager/v1/api/backlog_pb";
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
 * Per-entity-type delete confirmation settings.
 */
export type DeleteConfirmationSettings = {
  backlog: DeleteConfirmLevel;
  initiative: DeleteConfirmLevel;
  capture: DeleteConfirmLevel;
};

/**
 * User preferences and configuration (unified settings including execution defaults).
 */
export type Settings = Omit<
  ProtoMessage<ProtoSettings>,
  "theme" | "defaultDelaySeconds" | "maxFixupAttempts" | "maxAutoRounds" | "autoAdvanceDelaySeconds" | "agentMaxTurns" | "agentTimeoutSeconds" | "searchDebounceMs" | "toastDurationMs" | "reviewMaxBlockingViolations" | "reviewMaxWarnings" | "maxConcurrentExecutions" | "maxQueueDepth" | "circuitBreakerThreshold" | "circuitBreakerCooldownMinutes" | "executionCostCapPerRun" | "costPerTurnEstimate" | "deleteConfirmation"
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
  agentRequiresApproval: boolean;
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
  /** Concurrency and governance */
  maxConcurrentExecutions: number;
  maxQueueDepth: number;
  circuitBreakerThreshold: number;
  circuitBreakerCooldownMinutes: number;
  executionCostCapPerRun: number;
  costPerTurnEstimate: number;
};

/**
 * Research agent spawn response
 */
export type ResearchResponse = ProtoMessage<ProtoBacklogResearchResponse>;
