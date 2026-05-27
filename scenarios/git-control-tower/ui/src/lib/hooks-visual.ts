// ============================================================================
// Visual Capture, Workflow, Test & Tidiness Hooks
// ============================================================================

import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { queryKeys } from "./hooks-query-keys";
import {
  fetchVisualCaptures,
  fetchVisualCaptureDetail,
  triggerVisualCapture,
  fetchCaptureStorageStats,
  deleteVisualCapture,
  clearAllCaptureStorage,
  fetchTestExecutions,
  fetchTestExecution,
  triggerTestExecution,
  fetchTidinessScore,
  fetchTidinessIssues,
  fetchTidinessStaleness,
  triggerTidinessLightScan,
  fetchTidinessScenarioDetail,
} from "./api";
import type {
  CapturePreset,
  VisualCaptureListResponse,
  SnapshotSetMeta,
  SnapshotSetDetail,
  CaptureStorageStats,
  TestExecutionRequest,
  TestExecutionResult,
  TestExecutionListResponse,
  TidinessScoreResponse,
  TidinessIssue,
  TidinessStalenessInfo,
  TidinessLightScanResult,
  TidinessScenarioDetail,
} from "./api";

// ── Visual Capture ─────────────────────────────────────────────────────

export function useVisualCaptures(slug: string, enabled = true, repoId?: string | null) {
  return useQuery<VisualCaptureListResponse, Error>({
    queryKey: queryKeys.visualCaptures(slug, repoId),
    queryFn: () => fetchVisualCaptures(slug, repoId ?? undefined),
    enabled: enabled && Boolean(slug),
    refetchInterval: 10_000,
  });
}

export function useVisualCaptureDetail(id: string, slug: string, enabled = true, repoId?: string | null) {
  return useQuery<SnapshotSetDetail, Error>({
    queryKey: queryKeys.visualCaptureDetail(id, slug, repoId),
    queryFn: () => fetchVisualCaptureDetail(id, slug, repoId ?? undefined),
    enabled: enabled && Boolean(id) && Boolean(slug),
    staleTime: 60_000,
  });
}

export function useTriggerVisualCapture(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<SnapshotSetMeta, Error, { scenarioSlug: string; mode: "baseline" | "capture"; presets: CapturePreset[] }>({
    mutationFn: async ({ scenarioSlug, mode, presets }) => {
      const meta = await triggerVisualCapture(scenarioSlug, mode, repoId ?? undefined, presets);
      if (meta.status === "failed") {
        throw new Error(meta.error || "Capture failed — no screenshots were captured");
      }
      return meta;
    },
    onSuccess: (_data, { scenarioSlug }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.visualCaptures(scenarioSlug, repoId) });
    },
  });
}

export function useCaptureStorageStats(repoId?: string | null) {
  return useQuery<CaptureStorageStats, Error>({
    queryKey: queryKeys.captureStorage(repoId),
    queryFn: () => fetchCaptureStorageStats(repoId ?? undefined),
    staleTime: 30_000,
  });
}

export function useDeleteVisualCapture(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<void, Error, { id: string; scenarioSlug: string }>({
    mutationFn: ({ id, scenarioSlug }) => deleteVisualCapture(id, scenarioSlug, repoId ?? undefined),
    onSuccess: (_data, { scenarioSlug }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.captureStorage(repoId) });
      queryClient.invalidateQueries({ queryKey: queryKeys.visualCaptures(scenarioSlug, repoId) });
    },
  });
}

export function useClearAllCaptureStorage(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<void, Error, void>({
    mutationFn: () => clearAllCaptureStorage(repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.captureStorage(repoId) });
    },
  });
}

// ── Test Execution ─────────────────────────────────────────────────────

export function useTestExecutions(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<TestExecutionListResponse, Error>({
    queryKey: queryKeys.testExecutions(scenarioName, repoId),
    queryFn: () => fetchTestExecutions(scenarioName, 10, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    refetchInterval: 15_000,
  });
}

export function useTestExecution(id: string, enabled = true, repoId?: string | null) {
  return useQuery<TestExecutionResult, Error>({
    queryKey: queryKeys.testExecution(id, repoId),
    queryFn: () => fetchTestExecution(id, repoId ?? undefined),
    enabled: enabled && Boolean(id),
    staleTime: 30_000,
  });
}

export function useTriggerTestExecution(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<TestExecutionResult, Error, TestExecutionRequest>({
    mutationFn: (request: TestExecutionRequest) =>
      triggerTestExecution(request, repoId ?? undefined),
    onSuccess: (_data, request) => {
      queryClient.invalidateQueries({
        queryKey: queryKeys.testExecutions(request.scenarioName, repoId),
      });
    },
  });
}

// ── Tidiness Manager ───────────────────────────────────────────────────

export function useTidinessScore(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<TidinessScoreResponse, Error>({
    queryKey: queryKeys.tidinessScore(scenarioName, repoId),
    queryFn: () => fetchTidinessScore(scenarioName, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    refetchInterval: 30_000,
  });
}

export interface TidinessIssuesOptions {
  file?: string;
  category?: string;
  severity?: string;
  limit?: number;
  enabled?: boolean;
  repoId?: string | null;
}

export function useTidinessIssues(
  scenarioName: string,
  options: TidinessIssuesOptions = {},
) {
  const { file, category, severity, limit, enabled = true, repoId } = options;
  return useQuery<TidinessIssue[], Error>({
    queryKey: queryKeys.tidinessIssues(scenarioName, file, repoId, category, severity, limit),
    queryFn: () => fetchTidinessIssues(scenarioName, file, category, severity, limit, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    staleTime: 30_000,
  });
}

export function useTidinessStaleness(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<TidinessStalenessInfo, Error>({
    queryKey: queryKeys.tidinessStaleness(scenarioName, repoId),
    queryFn: () => fetchTidinessStaleness(scenarioName, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    staleTime: 60_000,
  });
}

export function useTriggerTidinessScan(repoId?: string | null) {
  const queryClient = useQueryClient();
  return useMutation<TidinessLightScanResult, Error, { scenarioName: string; incremental?: boolean }>({
    mutationFn: ({ scenarioName, incremental }) =>
      triggerTidinessLightScan({ scenario_name: scenarioName, incremental }, repoId ?? undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ["repo", "tidiness-score"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "tidiness-issues"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "tidiness-staleness"] });
      queryClient.invalidateQueries({ queryKey: ["repo", "tidiness-scenario"] });
    },
  });
}

export function useTidinessScenarioDetail(scenarioName: string, enabled = true, repoId?: string | null) {
  return useQuery<TidinessScenarioDetail, Error>({
    queryKey: queryKeys.tidinessScenarioDetail(scenarioName, repoId),
    queryFn: () => fetchTidinessScenarioDetail(scenarioName, repoId ?? undefined),
    enabled: enabled && Boolean(scenarioName),
    refetchInterval: 30_000,
  });
}
