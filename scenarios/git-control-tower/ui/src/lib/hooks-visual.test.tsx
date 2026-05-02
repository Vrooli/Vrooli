import { act, waitFor } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import {
  useDeleteVisualCapture,
  useTestExecutions,
  useTidinessIssues,
  useTriggerTestExecution,
  useTriggerTidinessScan,
  useTriggerVisualCapture,
  useTriggerWorkflowCapture,
  useVisualCaptures,
} from "./hooks-visual";
import { queryKeys } from "./hooks-query-keys";
import { renderHookWithQueryClient } from "../test-utils";

const mockFetchVisualCaptures = vi.fn();
const mockFetchVisualCaptureDetail = vi.fn();
const mockTriggerVisualCapture = vi.fn();
const mockFetchCaptureStorageStats = vi.fn();
const mockDeleteVisualCapture = vi.fn();
const mockClearAllCaptureStorage = vi.fn();
const mockFetchWorkflowCaptures = vi.fn();
const mockTriggerWorkflowCapture = vi.fn();
const mockFetchTestExecutions = vi.fn();
const mockFetchTestExecution = vi.fn();
const mockTriggerTestExecution = vi.fn();
const mockFetchTidinessScore = vi.fn();
const mockFetchTidinessIssues = vi.fn();
const mockFetchTidinessStaleness = vi.fn();
const mockTriggerTidinessLightScan = vi.fn();
const mockFetchTidinessScenarioDetail = vi.fn();

vi.mock("./api", () => ({
  fetchVisualCaptures: (...args: unknown[]) => mockFetchVisualCaptures(...args),
  fetchVisualCaptureDetail: (...args: unknown[]) => mockFetchVisualCaptureDetail(...args),
  triggerVisualCapture: (...args: unknown[]) => mockTriggerVisualCapture(...args),
  fetchCaptureStorageStats: (...args: unknown[]) => mockFetchCaptureStorageStats(...args),
  deleteVisualCapture: (...args: unknown[]) => mockDeleteVisualCapture(...args),
  clearAllCaptureStorage: (...args: unknown[]) => mockClearAllCaptureStorage(...args),
  fetchWorkflowCaptures: (...args: unknown[]) => mockFetchWorkflowCaptures(...args),
  triggerWorkflowCapture: (...args: unknown[]) => mockTriggerWorkflowCapture(...args),
  fetchTestExecutions: (...args: unknown[]) => mockFetchTestExecutions(...args),
  fetchTestExecution: (...args: unknown[]) => mockFetchTestExecution(...args),
  triggerTestExecution: (...args: unknown[]) => mockTriggerTestExecution(...args),
  fetchTidinessScore: (...args: unknown[]) => mockFetchTidinessScore(...args),
  fetchTidinessIssues: (...args: unknown[]) => mockFetchTidinessIssues(...args),
  fetchTidinessStaleness: (...args: unknown[]) => mockFetchTidinessStaleness(...args),
  triggerTidinessLightScan: (...args: unknown[]) => mockTriggerTidinessLightScan(...args),
  fetchTidinessScenarioDetail: (...args: unknown[]) => mockFetchTidinessScenarioDetail(...args),
}));

describe("visual, test, and tidiness hooks", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mockFetchVisualCaptures.mockResolvedValue({ captures: [] });
    mockTriggerVisualCapture.mockResolvedValue({
      id: "cap-1",
      scenario_slug: "git-control-tower",
      mode: "capture",
      status: "completed",
      created_at: "2026-05-01T00:00:00Z",
      presets: [],
      snapshots: [],
    });
    mockDeleteVisualCapture.mockResolvedValue(undefined);
    mockTriggerWorkflowCapture.mockResolvedValue({
      id: "wf-1",
      scenario_slug: "git-control-tower",
      mode: "capture",
      status: "completed",
      created_at: "2026-05-01T00:00:00Z",
      executions: [],
    });
    mockFetchTestExecutions.mockResolvedValue({ executions: [] });
    mockTriggerTestExecution.mockResolvedValue({
      id: "test-1",
      scenario_name: "git-control-tower",
      status: "completed",
      created_at: "2026-05-01T00:00:00Z",
    });
    mockFetchTidinessIssues.mockResolvedValue([]);
    mockTriggerTidinessLightScan.mockResolvedValue({ success: true });
  });

  it("does not fetch visual captures until a scenario slug is available", () => {
    renderHookWithQueryClient(() => useVisualCaptures("", true, "repo-1"));

    expect(mockFetchVisualCaptures).not.toHaveBeenCalled();
  });

  it("fetches test executions with the fixed list limit and repo context", async () => {
    renderHookWithQueryClient(() => useTestExecutions("git-control-tower", true, "repo-1"));

    await waitFor(() => {
      expect(mockFetchTestExecutions).toHaveBeenCalledWith("git-control-tower", 10, "repo-1");
    });
  });

  it("triggers visual capture and invalidates that scenario capture list", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useTriggerVisualCapture("repo-2"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({
        scenarioSlug: "git-control-tower",
        mode: "capture",
        presets: [{ name: "Desktop Light", width: 1440, height: 900, theme: "light" }],
      });
    });

    await waitFor(() => {
      expect(mockTriggerVisualCapture).toHaveBeenCalledWith(
        "git-control-tower",
        "capture",
        "repo-2",
        [{ name: "Desktop Light", width: 1440, height: 900, theme: "light" }],
      );
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.visualCaptures("git-control-tower", "repo-2"),
      });
    });
  });

  it("surfaces failed visual captures as mutation errors", async () => {
    mockTriggerVisualCapture.mockResolvedValueOnce({
      id: "cap-failed",
      scenario_slug: "git-control-tower",
      mode: "capture",
      status: "failed",
      error: "Browser unavailable",
      created_at: "2026-05-01T00:00:00Z",
      presets: [],
      snapshots: [],
    });
    const { result } = renderHookWithQueryClient(() => useTriggerVisualCapture("repo-2"));

    await expect(result.current.mutateAsync({
      scenarioSlug: "git-control-tower",
      mode: "capture",
      presets: [],
    })).rejects.toThrow("Browser unavailable");
  });

  it("deleting a capture refreshes storage and scenario captures", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useDeleteVisualCapture("repo-3"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({
        id: "cap-1",
        scenarioSlug: "git-control-tower",
      });
    });

    await waitFor(() => {
      expect(mockDeleteVisualCapture).toHaveBeenCalledWith("cap-1", "git-control-tower", "repo-3");
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.captureStorage("repo-3"),
      });
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.visualCaptures("git-control-tower", "repo-3"),
      });
    });
  });

  it("triggers workflow capture and invalidates workflow history", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useTriggerWorkflowCapture("repo-4"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({
        scenarioSlug: "git-control-tower",
        mode: "baseline",
        executionModes: ["observer"],
      });
    });

    await waitFor(() => {
      expect(mockTriggerWorkflowCapture).toHaveBeenCalledWith(
        "git-control-tower",
        "baseline",
        ["observer"],
        "repo-4",
      );
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.workflowCaptures("git-control-tower", "repo-4"),
      });
    });
  });

  it("invalidates scenario test executions after triggering a test run", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useTriggerTestExecution("repo-5"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");
    const request = { scenarioName: "git-control-tower", suite: "unit" };

    await act(async () => {
      await result.current.mutateAsync(request);
    });

    await waitFor(() => {
      expect(mockTriggerTestExecution).toHaveBeenCalledWith(request, "repo-5");
      expect(invalidateSpy).toHaveBeenCalledWith({
        queryKey: queryKeys.testExecutions("git-control-tower", "repo-5"),
      });
    });
  });

  it("passes tidiness issue filters through the query seam", async () => {
    renderHookWithQueryClient(() => useTidinessIssues("git-control-tower", {
      file: "src/App.tsx",
      category: "coverage",
      severity: "warning",
      limit: 25,
      repoId: "repo-6",
    }));

    await waitFor(() => {
      expect(mockFetchTidinessIssues).toHaveBeenCalledWith(
        "git-control-tower",
        "src/App.tsx",
        "coverage",
        "warning",
        25,
        "repo-6",
      );
    });
  });

  it("refreshes all tidiness query families after a light scan", async () => {
    const { result, queryClient } = renderHookWithQueryClient(() => useTriggerTidinessScan("repo-7"));
    const invalidateSpy = vi.spyOn(queryClient, "invalidateQueries");

    await act(async () => {
      await result.current.mutateAsync({
        scenarioName: "git-control-tower",
        incremental: true,
      });
    });

    await waitFor(() => {
      expect(mockTriggerTidinessLightScan).toHaveBeenCalledWith({
        scenario_name: "git-control-tower",
        incremental: true,
      }, "repo-7");
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["repo", "tidiness-score"] });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["repo", "tidiness-issues"] });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["repo", "tidiness-staleness"] });
      expect(invalidateSpy).toHaveBeenCalledWith({ queryKey: ["repo", "tidiness-scenario"] });
    });
  });
});
