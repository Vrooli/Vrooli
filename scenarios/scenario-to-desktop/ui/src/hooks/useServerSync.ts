/**
 * Hook that persists pipeline build and smoke test status to the server.
 * Extracted from App.tsx to reduce root component complexity.
 */

import { useEffect, useRef } from "react";
import { usePipelineStore } from "../store";
import { useScenarioState, type UseScenarioStateResult } from "./useScenarioState";
import type { FormState, SmokeTestStageDetails } from "../lib/api";
import type { ViewMode } from "./useUrlState";

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
export function useServerSync({ scenarioName, viewMode }: UseServerSyncOptions): UseServerSyncResult {
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
      wrapper_output_path: generateResult?.desktop_path ?? null,
    });
  }, [scenarioName, serverStateLoaded, pipelineId, uiBuildStatus, generateResult, updateServerFormState]);

  // Persist smoke test status changes to server
  const prevSmokeRef = useRef<string | null>(null);
  useEffect(() => {
    if (!scenarioName || !serverStateLoaded) return;
    if (!smokeTestResult) return;

    const testId = smokeTestResult.smoke_test_id;
    if (!testId) return;

    const statusKey = `${testId}:${smokeTestResult.status}`;
    if (statusKey === prevSmokeRef.current) return;
    prevSmokeRef.current = statusKey;

    const smokeTestFormState: Partial<FormState> = {
      smoke_test_id: testId,
      smoke_test_platform: smokeTestResult.platform as "win" | "mac" | "linux" | null,
      smoke_test_status: smokeTestResult.status as "running" | "passed" | "failed" | null,
      smoke_test_started_at: smokeTestResult.started_at,
      smoke_test_completed_at: smokeTestResult.completed_at ?? null,
      smoke_test_logs: smokeTestResult.logs ?? null,
      smoke_test_error: smokeTestResult.error ?? null,
      smoke_test_telemetry_uploaded: smokeTestResult.telemetry_uploaded ?? false,
    };

    if (smokeTestResult.status === "passed" || smokeTestResult.status === "failed") {
      void saveStageResult("smoke_test", smokeTestResult, smokeTestFormState);
    } else {
      updateServerFormState(smokeTestFormState);
    }
  }, [scenarioName, serverStateLoaded, smokeTestResult, saveStageResult, updateServerFormState]);

  return { serverStateLoaded };
}
