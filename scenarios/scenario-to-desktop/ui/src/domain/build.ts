/**
 * Pure domain functions for pipeline stage result extraction.
 * These functions have no side effects and can be tested in isolation.
 */

import type {
  VerbosePipelineStatus,
  BuildStageDetails,
  GenerateStageDetails,
  BundleStageDetails,
  BundlePreflightResponse,
  SmokeTestStageDetails,
  DeployStageDetails,
} from "../lib/api";

// ============================================================================
// Pipeline Status Transformation
// ============================================================================

/**
 * Extracted stage results from a verbose pipeline status.
 * Separates the data transformation logic from store state management.
 */
export interface ExtractedStageResults {
  bundleResult: BundleStageDetails | null;
  preflightResult: BundlePreflightResponse | null;
  generateResult: GenerateStageDetails | null;
  buildResult: BuildStageDetails | null;
  smokeTestResult: SmokeTestStageDetails | null;
  deployResult: DeployStageDetails | null;
  stageLogs: Record<string, string[]>;
}

/**
 * Extract stage results from a verbose pipeline status.
 * This is pure data transformation - no side effects.
 *
 * ASSUMPTION: Stage details types match the expected TypeScript interfaces.
 * Backend schema changes would require updating these casts.
 */
export function extractStageResults(status: VerbosePipelineStatus): ExtractedStageResults {
  const stages = status.stages ?? {};
  const logs: Record<string, string[]> = {};

  const results: ExtractedStageResults = {
    bundleResult: null,
    preflightResult: null,
    generateResult: null,
    buildResult: null,
    smokeTestResult: null,
    deployResult: null,
    stageLogs: {},
  };

  // Extract stage details
  if (stages.bundle?.details) {
    results.bundleResult = stages.bundle.details as BundleStageDetails;
  }
  if (stages.preflight?.details) {
    results.preflightResult = stages.preflight.details as BundlePreflightResponse;
  }
  if (stages.generate?.details) {
    results.generateResult = stages.generate.details as GenerateStageDetails;
  }
  if (stages.build?.details) {
    results.buildResult = stages.build.details as BuildStageDetails;
  }
  if (stages.smoketest?.details) {
    results.smokeTestResult = stages.smoketest.details as SmokeTestStageDetails;
  }
  if (stages.deploy?.details) {
    results.deployResult = stages.deploy.details as DeployStageDetails;
  }

  // Extract logs from all stages
  for (const [stageName, stage] of Object.entries(stages)) {
    if (stage?.logs?.length) {
      logs[stageName] = stage.logs;
    }
  }
  results.stageLogs = logs;

  return results;
}
