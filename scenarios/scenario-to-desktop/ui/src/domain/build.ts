/**
 * Pure domain functions for pipeline stage result extraction.
 * These functions have no side effects and can be tested in isolation.
 */

import type {
  VerbosePipelineStatus,
  BuildStageDetails,
  GenerateStageDetails,
  PreflightStageDetails,
  SmokeTestStageDetails,
  DeployStageDetails,
} from "../lib/api";
import type { BundleStageDetails } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";

// ============================================================================
// Pipeline Status Transformation
// ============================================================================

/**
 * Extracted stage results from a verbose pipeline status.
 * Separates the data transformation logic from store state management.
 */
export interface ExtractedStageResults {
  bundleResult: BundleStageDetails | null;
  preflightResult: PreflightStageDetails | null;
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
export function extractStageResults(
  status: VerbosePipelineStatus,
): ExtractedStageResults {
  const stages = status.stages;
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
  if (stages.bundle?.details?.kind.case === "bundle") {
    results.bundleResult = stages.bundle.details.kind.value;
  }
  if (stages.preflight?.details?.kind.case === "preflight") {
    results.preflightResult = stages.preflight.details.kind.value;
  }
  if (stages.generate?.details?.kind.case === "generate") {
    results.generateResult = stages.generate.details.kind.value;
  }
  if (stages.build?.details?.kind.case === "build") {
    results.buildResult = stages.build.details.kind.value;
  }
  if (stages.smoketest?.details?.kind.case === "smokeTest") {
    results.smokeTestResult = stages.smoketest.details.kind.value;
  }
  if (stages.deploy?.details?.kind.case === "deploy") {
    results.deployResult = stages.deploy.details.kind.value;
  }

  // Extract logs from all stages
  for (const [stageName, stage] of Object.entries(stages)) {
    const stageLogs = (stage as { logs?: string[] }).logs;
    if (stageLogs?.length) {
      logs[stageName] = stageLogs;
    }
  }
  results.stageLogs = logs;

  return results;
}
