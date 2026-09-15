/**
 * Pipeline store type definitions.
 * Extracted from pipelineStore.ts for modularity.
 */

import type {
  VerbosePipelineStatus,
  PreflightStageDetails,
  GenerateStageDetails,
  BuildStageDetails,
  SmokeTestStageDetails,
  DeployStageDetails,
  PipelineConfig,
} from "../lib/api";
import type { StageName } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import type { BundleStageDetails } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";

// ============================================================================
// Stage Types
// ============================================================================

/** Pipeline stages use the generated Proto enum end-to-end. */
export type PipelineStage = StageName;

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
  /** Connect/HTTP error detail retained for copyable operator diagnostics. */
  details?: Record<string, unknown>;
}

// ============================================================================
// Pipeline Cache Types (for scenario switching)
// ============================================================================

/** Cache entry for a scenario's pipeline state */
export interface ScenarioPipelineCacheEntry {
  pipelineId: string;
  status: PipelineRunStatus;
  lastAccessed: number;
}

/** Maximum number of cached scenarios (LRU limit) */
export const PIPELINE_CACHE_MAX_SIZE = 10;

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
  preflightResult: PreflightStageDetails | null;
  generateResult: GenerateStageDetails | null;
  buildResult: BuildStageDetails | null;
  smokeTestResult: SmokeTestStageDetails | null;
  deployResult: DeployStageDetails | null;

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

  // Scenario pipeline cache (LRU, protects running scenarios from eviction)
  scenarioPipelineCache: Map<string, ScenarioPipelineCacheEntry>;
  /** Scenarios with running pipelines (protected from cache eviction) */
  runningScenarios: Set<string>;
  /** Whether we're loading the active pipeline from server */
  isLoadingActivePipeline: boolean;
}

/** Status subscriber callback type */
export type StatusSubscriber = (status: VerbosePipelineStatus | null) => void;

export interface PipelineStoreActions {
  // Scenario context
  setScenario: (name: string | null) => void;

  // Pipeline execution
  runStage: (
    stage: PipelineStage,
    config?: Partial<PipelineConfig>,
  ) => Promise<string>;
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

  // Scenario-based pipeline management (server-side persistence)
  /**
   * Load the active pipeline for the current scenario from the server.
   * If autoCreate is true (default), creates a new pipeline if none exists.
   */
  loadActivePipeline: (autoCreate?: boolean) => Promise<void>;
  /**
   * Create a new pipeline for the current scenario.
   * Archives the current active pipeline if one exists.
   */
  createNewPipelineForScenario: (
    config?: Partial<PipelineConfig>,
  ) => Promise<string>;
  /**
   * Reset the current pipeline (archive and clear active).
   */
  resetCurrentPipeline: () => Promise<void>;
  /**
   * Load the pipeline history for the current scenario.
   */
  loadPipelineHistory: (limit?: number) => Promise<VerbosePipelineStatus[]>;

  // Internal helpers (prefixed with _ to indicate private)
  _setPipelineStatus: (status: VerbosePipelineStatus | null) => void;
  _extractStageResults: (status: VerbosePipelineStatus) => void;
  _notifySubscribers: () => void;
  _updateCache: () => void;
  _pruneCache: () => void;
  _startPipeline: (
    config: Partial<import("../lib/api").PipelineConfig>,
    label: string,
  ) => Promise<string>;
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
  deployResult: null,
  stageLogs: {},
  pipelineHistory: [],
  preflightSecrets: {},
  preflightOverride: false,
  isSubmitting: false,
  currentIdempotencyKey: null,
  scenarioPipelineCache: new Map(),
  runningScenarios: new Set(),
  isLoadingActivePipeline: false,
};
