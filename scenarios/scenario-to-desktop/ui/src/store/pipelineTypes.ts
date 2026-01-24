/**
 * Pipeline store type definitions.
 * Extracted from pipelineStore.ts for modularity.
 */

import type {
  VerbosePipelineStatus,
  BundleStageDetails,
  BundlePreflightResponse,
  GenerateStageDetails,
  BuildStageDetails,
  SmokeTestStageDetails,
  DistributionStageDetails,
  PipelineConfig,
} from "../lib/api";

// ============================================================================
// Stage Types
// ============================================================================

/** Pipeline stages available for stop_after_stage */
export type PipelineStage =
  | "bundle"
  | "preflight"
  | "generate"
  | "build"
  | "smoketest"
  | "distribution";

/** Pipeline run status (simplified for UI consumption) */
export type PipelineRunStatus =
  | "idle"
  | "starting"
  | "running"
  | "completed"
  | "failed"
  | "cancelled";

// ============================================================================
// Error Types
// ============================================================================

/** Error categories for targeted recovery UI */
export type PipelineErrorCategory =
  | "network"
  | "validation"
  | "permission"
  | "timeout"
  | "resource"
  | "unknown";

/** Structured error information for UI consumption */
export interface PipelineErrorInfo {
  message: string;
  category?: PipelineErrorCategory;
  suggestions?: string[];
  raw?: unknown;
}

// ============================================================================
// Store State Types
// ============================================================================

export interface PipelineStoreState {
  // Current scenario context
  scenarioName: string | null;

  // Active pipeline tracking
  pipelineId: string | null;
  pipelineStatus: VerbosePipelineStatus | null;
  runStatus: PipelineRunStatus;
  /** Structured error information with recovery guidance */
  errorInfo: PipelineErrorInfo | null;

  // Polling configuration
  isPolling: boolean;
  pollIntervalMs: number;

  // Stage-specific results (extracted from verbose pipeline status)
  bundleResult: BundleStageDetails | null;
  preflightResult: BundlePreflightResponse | null;
  generateResult: GenerateStageDetails | null;
  buildResult: BuildStageDetails | null;
  smokeTestResult: SmokeTestStageDetails | null;
  distributionResult: DistributionStageDetails | null;

  // Stage logs (from verbose pipeline status)
  stageLogs: Record<string, string[]>;

  // Historical pipeline IDs for this scenario (for resume functionality)
  pipelineHistory: string[];

  // Preflight-specific state (for GeneratorForm integration)
  /** User-provided secret values for preflight validation (keyed by secret ID) */
  preflightSecrets: Record<string, string>;
  /**
   * When true, allows generation to proceed even if preflight fails.
   * Used for development/debugging when users want to bypass validation.
   * UI shows "Override preflight" checkbox when preflight is not OK.
   * Note: This does NOT skip preflight checks, it just allows proceeding despite failures.
   */
  preflightOverride: boolean;

  // Request deduplication: tracks whether a request is currently in-flight
  isSubmitting: boolean;
  /** The idempotency key for the current/most recent request */
  currentIdempotencyKey: string | null;
}

/** Status subscriber callback type */
export type StatusSubscriber = (status: VerbosePipelineStatus | null) => void;

export interface PipelineStoreActions {
  // Scenario context
  setScenario: (name: string | null) => void;

  // Pipeline execution
  runStage: (stage: PipelineStage, config?: Partial<PipelineConfig>) => Promise<string>;
  runFullPipeline: (config?: Partial<PipelineConfig>) => Promise<string>;
  cancelPipeline: () => Promise<void>;
  resumePipeline: (pipelineId: string) => Promise<string>;

  // Convenience actions for specific stages
  runBundleStage: (config?: Partial<PipelineConfig>) => Promise<string>;
  runPreflightStage: (config?: Partial<PipelineConfig>) => Promise<string>;
  runSmokeTestStage: (config?: Partial<PipelineConfig>) => Promise<string>;

  // Status management
  loadPipelineStatus: (pipelineId: string) => Promise<void>;
  startPolling: () => void;
  stopPolling: () => void;

  /**
   * Subscribe to status updates. Returns unsubscribe function.
   * This is the primary way for components to receive pipeline status.
   * Replaces React Query polling in components.
   */
  subscribeToStatus: (callback: StatusSubscriber) => () => void;

  // State management
  reset: () => void;
  clearError: () => void;
  resetForRetry: () => void;

  // Preflight-specific actions
  setPreflightSecrets: (secrets: Record<string, string>) => void;
  setPreflightSecret: (id: string, value: string) => void;
  setPreflightOverride: (override: boolean) => void;
  resetPreflight: () => void;

  // Internal helpers (prefixed with _ to indicate private)
  _setPipelineStatus: (status: VerbosePipelineStatus | null) => void;
  _extractStageResults: (status: VerbosePipelineStatus) => void;
  _notifySubscribers: () => void;
}

export type PipelineStore = PipelineStoreState & PipelineStoreActions;

// ============================================================================
// Initial State
// ============================================================================

export const initialPipelineState: PipelineStoreState = {
  scenarioName: null,
  pipelineId: null,
  pipelineStatus: null,
  runStatus: "idle",
  errorInfo: null,
  isPolling: false,
  pollIntervalMs: 2000,
  bundleResult: null,
  preflightResult: null,
  generateResult: null,
  buildResult: null,
  smokeTestResult: null,
  distributionResult: null,
  stageLogs: {},
  pipelineHistory: [],
  preflightSecrets: {},
  preflightOverride: false,
  isSubmitting: false,
  currentIdempotencyKey: null,
};
