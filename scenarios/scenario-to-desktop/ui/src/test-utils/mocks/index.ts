/**
 * Mock factories for testing.
 * Provides consistent test data creation with sensible defaults and override support.
 */

import type { VerbosePipelineStatus, BundlePreflightResponse, BundlePreflightSecret } from "../../lib/api";
import { initialPipelineState, type PipelineStoreState } from "../../store/pipelineTypes";
import { initialFormState, type FormStoreState } from "../../store/formTypes";

// ============================================================================
// Pipeline Store Mocks
// ============================================================================

/**
 * Creates a mock pipeline store state with sensible defaults.
 * All values can be overridden.
 *
 * @example
 * ```ts
 * // Default state
 * const state = createPipelineState();
 *
 * // Override specific values
 * const state = createPipelineState({
 *   scenarioName: "my-scenario",
 *   runStatus: "running",
 * });
 * ```
 */
export function createPipelineState(
  overrides?: Partial<PipelineStoreState>
): PipelineStoreState {
  return {
    ...initialPipelineState,
    ...overrides,
  };
}

/**
 * Creates a mock pipeline state for a running pipeline.
 */
export function createRunningPipelineState(
  scenarioName: string,
  pipelineId: string = "test-pipeline-123"
): PipelineStoreState {
  return createPipelineState({
    scenarioName,
    pipelineId,
    runStatus: "running",
    isPolling: true,
  });
}

/**
 * Creates a mock pipeline state for a completed pipeline.
 */
export function createCompletedPipelineState(
  scenarioName: string,
  pipelineId: string = "test-pipeline-123"
): PipelineStoreState {
  return createPipelineState({
    scenarioName,
    pipelineId,
    runStatus: "completed",
    isPolling: false,
    pipelineHistory: [pipelineId],
  });
}

/**
 * Creates a mock pipeline state for a failed pipeline.
 */
export function createFailedPipelineState(
  scenarioName: string,
  errorMessage: string = "Pipeline failed",
  pipelineId: string = "test-pipeline-123"
): PipelineStoreState {
  return createPipelineState({
    scenarioName,
    pipelineId,
    runStatus: "failed",
    errorInfo: {
      message: errorMessage,
      category: "unknown",
    },
    isPolling: false,
  });
}

// ============================================================================
// Form Store Mocks
// ============================================================================

/**
 * Creates a mock form store state with sensible defaults.
 * All values can be overridden.
 *
 * @example
 * ```ts
 * // Default state
 * const state = createFormState();
 *
 * // Override specific values
 * const state = createFormState({
 *   appMetadata: { ...defaultAppMetadata, displayName: "My App" },
 * });
 * ```
 */
export function createFormState(
  overrides?: Partial<FormStoreState>
): FormStoreState {
  return {
    ...initialFormState,
    ...overrides,
  };
}

// ============================================================================
// API Response Mocks
// ============================================================================

/**
 * Creates a mock verbose pipeline status.
 *
 * @example
 * ```ts
 * const status = createPipelineStatus({
 *   status: "running",
 *   current_stage: "build",
 * });
 * ```
 */
export function createPipelineStatus(
  overrides?: Partial<VerbosePipelineStatus>
): VerbosePipelineStatus {
  return {
    pipeline_id: "test-pipeline-123",
    status: "running",
    current_stage: "bundle",
    stage_order: ["bundle", "preflight", "generate", "build", "smoketest", "deploy"],
    stages: {
      bundle: { status: "pending" },
      preflight: { status: "pending" },
      generate: { status: "pending" },
      build: { status: "pending" },
      smoketest: { status: "pending" },
      deploy: { status: "pending" },
    },
    ...overrides,
  } as VerbosePipelineStatus;
}

/**
 * Creates a mock preflight response with validation passed.
 */
export function createPreflightResponse(
  overrides?: Partial<BundlePreflightResponse>
): BundlePreflightResponse {
  return {
    validation: {
      valid: true,
      missing_assets: [],
      missing_binaries: [],
    },
    ready: {
      ready: true,
      details: {},
    },
    secrets: [],
    ...overrides,
  } as BundlePreflightResponse;
}

/**
 * Creates a mock preflight response with missing secrets.
 */
export function createPreflightWithMissingSecrets(
  secrets: Array<{ id: string; label?: string; class?: string }>
): BundlePreflightResponse {
  const secretsList: BundlePreflightSecret[] = secrets.map((s) => ({
    id: s.id,
    label: s.label ?? s.id,
    class: s.class ?? "env",
    required: true,
    has_value: false,
    description: `Secret for ${s.id}`,
  }));

  return createPreflightResponse({
    secrets: secretsList,
  });
}

/**
 * Creates a mock preflight response with validation errors.
 * @param missingAssets - Array of asset paths that are missing
 * @param missingBinaries - Array of binary paths that are missing
 */
export function createPreflightWithValidationErrors(
  missingAssets: string[] = [],
  missingBinaries: string[] = []
): BundlePreflightResponse {
  return createPreflightResponse({
    validation: {
      valid: false,
      missing_assets: missingAssets.map((path) => ({
        service_id: "default",
        path,
      })),
      missing_binaries: missingBinaries.map((path) => ({
        service_id: "default",
        platform: "linux",
        path,
      })),
    },
  });
}

// ============================================================================
// Verbose Stage Result Mocks
// ============================================================================

/**
 * Creates a mock VerboseStageResult for use in VerbosePipelineStatus.
 * This provides all required fields for type-safe test mocks.
 *
 * @example
 * ```ts
 * const stages = {
 *   build: createVerboseStageResult("running"),
 *   bundle: createVerboseStageResult("completed"),
 * };
 * ```
 */
export function createVerboseStageResult(
  status: string = "pending",
  overrides?: {
    stage?: string;
    started_at?: number;
    completed_at?: number;
    error?: string;
    details?: unknown;
    logs?: string[];
  }
): {
  stage: string;
  status: string;
  started_at: number;
  completed_at?: number;
  error?: string;
  details?: unknown;
  logs?: string[];
} {
  return {
    stage: overrides?.stage ?? "unknown",
    status,
    started_at: overrides?.started_at ?? (status === "pending" ? 0 : Date.now()),
    ...(overrides?.completed_at !== undefined && { completed_at: overrides.completed_at }),
    ...(overrides?.error !== undefined && { error: overrides.error }),
    ...(overrides?.details !== undefined && { details: overrides.details }),
    ...(overrides?.logs !== undefined && { logs: overrides.logs }),
  };
}

// ============================================================================
// Stage Result Mocks
// ============================================================================

/**
 * Creates a mock bundle stage result.
 */
export function createBundleResult(overrides?: Record<string, unknown>) {
  return {
    bundle_dir: "/path/to/bundle",
    manifest_path: "/path/to/manifest.json",
    total_size_human: "150 MB",
    copied_artifacts: ["app.js", "index.html"],
    runtime_binaries: { linux: "/path/to/binary" },
    ...overrides,
  };
}

/**
 * Creates a mock generate stage result.
 */
export function createGenerateResult(overrides?: Record<string, unknown>) {
  return {
    desktop_path: "/path/to/desktop",
    build_id: "build-abc123",
    ...overrides,
  };
}

/**
 * Creates a mock build stage result.
 */
export function createBuildResult(
  platforms: string[] = ["win", "mac", "linux"],
  overrides?: Record<string, unknown>
) {
  const artifacts: Record<string, string> = {};
  platforms.forEach((p) => {
    artifacts[p] = `/path/to/output/${p}/installer`;
  });

  return {
    output_path: "/path/to/output",
    platforms,
    artifacts,
    ...overrides,
  };
}

/**
 * Creates a mock smoke test result.
 */
export function createSmokeTestResult(
  status: "passed" | "failed" | "completed" = "passed",
  overrides?: Record<string, unknown>
) {
  return {
    status,
    platform: "linux",
    artifact_path: "/path/to/artifact",
    logs: ["Test started", "Test completed"],
    telemetry_uploaded: true,
    error: status === "failed" ? "Test failed" : undefined,
    ...overrides,
  };
}

/**
 * Creates a mock deploy result.
 */
export function createDeployResult(overrides?: Record<string, unknown>) {
  return {
    artifacts: [
      { artifact_id: 1, platform: "win" },
      { artifact_id: 2, platform: "mac" },
    ],
    update_url: "https://example.com/api/v1/updates/my-app",
    ...overrides,
  };
}
