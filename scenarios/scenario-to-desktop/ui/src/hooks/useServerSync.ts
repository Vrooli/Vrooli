/**
 * Hook that persists pipeline build and smoke test status to the server.
 * Extracted from App.tsx to reduce root component complexity.
 */

import { useEffect, useRef } from "react";
import { usePipelineStore } from "../store";
import { useScenarioState } from "./useScenarioState";
import type { FormState } from "../lib/api";
import type { ViewMode } from "./useUrlState";
import { Platform } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { SmokeTestStatus } from "@vrooli/proto-types/scenario-to-desktop/v1/shared/operation_results_pb";

function formPlatform(platform: Platform): "win" | "mac" | "linux" | null {
  switch (platform) {
    case Platform.WIN:
      return "win";
    case Platform.MAC:
      return "mac";
    case Platform.LINUX:
      return "linux";
    default:
      return null;
  }
}

function formSmokeTestStatus(
  status: SmokeTestStatus,
): "running" | "passed" | "failed" | null {
  switch (status) {
    case SmokeTestStatus.RUNNING:
      return "running";
    case SmokeTestStatus.PASSED:
      return "passed";
    case SmokeTestStatus.FAILED:
      return "failed";
    default:
      return null;
  }
}

function timestampISO(
  value: { seconds: bigint; nanos: number } | undefined,
): string | null {
  if (!value) return null;
  return new Date(
    Number(value.seconds) * 1_000 + value.nanos / 1_000_000,
  ).toISOString();
}

interface UseServerSyncOptions {
  scenarioName: string;
  viewMode: ViewMode;
}

interface UseServerSyncResult {
  serverStateLoaded: boolean;
}

/**
 * Syncs pipeline build and smoke test status changes to the server.
 * Subscribes to the pipeline store and persists status transitions
 * via useScenarioState's updateFormState and saveStageResult.
 */
export function useServerSync({
  scenarioName,
  viewMode,
}: UseServerSyncOptions): UseServerSyncResult {
  // Pipeline store selectors
  const pipelineId = usePipelineStore((s) => s.pipelineId);
  const runStatus = usePipelineStore((s) => s.runStatus);
  const generateResult = usePipelineStore((s) => s.generateResult);
  const smokeTestResult = usePipelineStore((s) => s.smokeTestResult);
  const clearError = usePipelineStore((s) => s.clearError);

  // Server-side state persistence
  const {
    hasInitiallyLoaded: serverStateLoaded,
    updateFormState: updateServerFormState,
    saveStageResult,
  } = useScenarioState({
    scenarioName,
    enabled: Boolean(scenarioName),
    checkStaleness: viewMode === "generator",
  });

  // Reset errors when scenario changes
  useEffect(() => {
    clearError();
  }, [scenarioName, clearError]);

  // Map store run status to server-friendly build status
  const uiBuildStatus = (() => {
    switch (runStatus) {
      case "running":
      case "starting":
        return "building" as const;
      case "completed":
        return "ready" as const;
      case "failed":
      case "cancelled":
        return "failed" as const;
      default:
        return null;
    }
  })();

  // Persist wrapper build status changes to server
  const prevBuildRef = useRef<string | null>(null);
  useEffect(() => {
    if (!scenarioName || !serverStateLoaded) return;
    if (!pipelineId || !uiBuildStatus) return;

    const statusKey = `${pipelineId}:${uiBuildStatus}`;
    if (statusKey === prevBuildRef.current) return;
    prevBuildRef.current = statusKey;

    updateServerFormState({
      wrapper_build_id: pipelineId,
      wrapper_build_status: uiBuildStatus,
      wrapper_output_path: generateResult?.desktopPath ?? null,
    });
  }, [
    scenarioName,
    serverStateLoaded,
    pipelineId,
    uiBuildStatus,
    generateResult,
    updateServerFormState,
  ]);

  // Persist smoke test status changes to server
  const prevSmokeRef = useRef<string | null>(null);
  useEffect(() => {
    if (!scenarioName || !serverStateLoaded) return;
    if (!smokeTestResult) return;

    const testId = smokeTestResult.smokeTestId;
    if (!testId) return;

    const statusKey = `${testId}:${String(smokeTestResult.status)}`;
    if (statusKey === prevSmokeRef.current) return;
    prevSmokeRef.current = statusKey;

    const smokeTestFormState: Partial<FormState> = {
      smoke_test_id: testId,
      smoke_test_platform: formPlatform(smokeTestResult.platform),
      smoke_test_status: formSmokeTestStatus(smokeTestResult.status),
      smoke_test_started_at: timestampISO(smokeTestResult.startedAt),
      smoke_test_completed_at: timestampISO(smokeTestResult.completedAt),
      smoke_test_logs: smokeTestResult.logs,
      smoke_test_error: smokeTestResult.error ?? null,
      smoke_test_telemetry_uploaded: smokeTestResult.telemetryUploaded,
    };

    if (
      smokeTestResult.status === SmokeTestStatus.PASSED ||
      smokeTestResult.status === SmokeTestStatus.FAILED
    ) {
      void saveStageResult("smoke_test", smokeTestResult, smokeTestFormState);
    } else {
      updateServerFormState(smokeTestFormState);
    }
  }, [
    scenarioName,
    serverStateLoaded,
    smokeTestResult,
    saveStageResult,
    updateServerFormState,
  ]);

  return { serverStateLoaded };
}
