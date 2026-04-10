/**
 * Tests for usePipelineInvestigation composite hook.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { usePipelineInvestigation } from "../useInvestigation";
import type {
  Investigation,
  CreateTaskRequest,
} from "../../types/investigation";
import {
  mockGetAgentManagerStatus,
  mockCreateTask,
  mockListTasks,
  mockGetTask,
  mockStopTask,
  createWrapper,
  createMockInvestigation,
  createMockInvestigationSummary,
} from "./useInvestigation.testUtils";

beforeEach(() => {
  vi.clearAllMocks();
  // Default mocks for the composite hook
  mockGetAgentManagerStatus.mockResolvedValue({ available: true });
  mockListTasks.mockResolvedValue([]);
  mockGetTask.mockResolvedValue(null);
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("usePipelineInvestigation", () => {
  describe("agent status", () => {
    it("returns agent availability status", async () => {
      mockGetAgentManagerStatus.mockResolvedValue({
        available: true,
        url: "http://localhost:8080",
      });

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isAgentLoading).toBe(false);
      });

      expect(result.current.isAgentAvailable).toBe(true);
      expect(result.current.agentStatus?.url).toBe("http://localhost:8080");
    });

    it("returns unavailable when agent is down", async () => {
      mockGetAgentManagerStatus.mockResolvedValue({
        available: false,
        reason: "Service not running",
      });

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isAgentLoading).toBe(false);
      });

      expect(result.current.isAgentAvailable).toBe(false);
    });
  });

  describe("tasks list", () => {
    it("returns tasks for pipeline", async () => {
      const mockTasks = [
        createMockInvestigationSummary({ id: "task-1" }),
        createMockInvestigationSummary({ id: "task-2" }),
      ];
      mockListTasks.mockResolvedValue(mockTasks);

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoadingTasks).toBe(false);
      });

      expect(result.current.tasks).toEqual(mockTasks);
    });

    it("returns empty array when pipelineId is null", async () => {
      const { result } = renderHook(() => usePipelineInvestigation(null), {
        wrapper: createWrapper(),
      });

      expect(result.current.tasks).toEqual([]);
    });
  });

  describe("active task tracking", () => {
    it("auto-tracks running task", async () => {
      const runningTask = createMockInvestigationSummary({
        id: "running-task",
        status: "running",
      });
      const completedTask = createMockInvestigationSummary({
        id: "completed-task",
        status: "completed",
      });
      mockListTasks.mockResolvedValue([completedTask, runningTask]);
      mockGetTask.mockResolvedValue(
        createMockInvestigation({ id: "running-task", status: "running" })
      );

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("running-task");
      });
    });

    it("tracks pending task", async () => {
      const pendingTask = createMockInvestigationSummary({
        id: "pending-task",
        status: "pending",
      });
      mockListTasks.mockResolvedValue([pendingTask]);
      mockGetTask.mockResolvedValue(
        createMockInvestigation({ id: "pending-task", status: "pending" })
      );

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("pending-task");
      });
    });

    it("tracks most recent task when no running task", async () => {
      const task1 = createMockInvestigationSummary({
        id: "task-1",
        status: "completed",
        created_at: "2024-01-01T00:00:00Z",
      });
      const task2 = createMockInvestigationSummary({
        id: "task-2",
        status: "completed",
        created_at: "2024-01-02T00:00:00Z",
      });
      // Tasks are sorted by created_at desc, so task-2 is first
      mockListTasks.mockResolvedValue([task2, task1]);
      mockGetTask.mockResolvedValue(createMockInvestigation({ id: "task-2" }));

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("task-2");
      });
    });

    it("resets active task when pipeline changes", async () => {
      const task = createMockInvestigationSummary({ id: "task-1" });
      mockListTasks.mockResolvedValue([task]);
      mockGetTask.mockResolvedValue(createMockInvestigation({ id: "task-1" }));

      const { result, rerender } = renderHook(
        (pipelineId: string | null) => usePipelineInvestigation(pipelineId),
        {
          wrapper: createWrapper(),
          initialProps: "pipeline-1",
        }
      );

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("task-1");
      });

      // Change pipeline
      mockListTasks.mockResolvedValue([]);
      rerender("pipeline-2");

      await waitFor(() => {
        expect(result.current.activeTaskId).toBeNull();
      });
    });
  });

  describe("isRunning", () => {
    it("returns true when active task is running", async () => {
      const runningTask = createMockInvestigationSummary({
        id: "task-1",
        status: "running",
      });
      mockListTasks.mockResolvedValue([runningTask]);
      mockGetTask.mockResolvedValue(
        createMockInvestigation({ id: "task-1", status: "running" })
      );

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isRunning).toBe(true);
      });
    });

    it("returns true when active task is pending", async () => {
      const pendingTask = createMockInvestigationSummary({
        id: "task-1",
        status: "pending",
      });
      mockListTasks.mockResolvedValue([pendingTask]);
      mockGetTask.mockResolvedValue(
        createMockInvestigation({ id: "task-1", status: "pending" })
      );

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isRunning).toBe(true);
      });
    });

    it("returns false when active task is completed", async () => {
      const completedTask = createMockInvestigationSummary({
        id: "task-1",
        status: "completed",
      });
      mockListTasks.mockResolvedValue([completedTask]);
      mockGetTask.mockResolvedValue(
        createMockInvestigation({ id: "task-1", status: "completed" })
      );

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("task-1");
      });

      expect(result.current.isRunning).toBe(false);
    });
  });

  describe("trigger", () => {
    it("creates a new task and tracks it", async () => {
      const newTask = createMockInvestigation({
        id: "new-task",
        status: "running",
      });
      const newTaskSummary = createMockInvestigationSummary({
        id: "new-task",
        status: "running",
      });

      // After creation, the task list will be refetched and include the new task
      mockListTasks
        .mockResolvedValueOnce([]) // Initial fetch
        .mockResolvedValue([newTaskSummary]); // After creation
      mockCreateTask.mockResolvedValue(newTask);
      mockGetTask.mockResolvedValue(newTask);

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isAgentLoading).toBe(false);
        expect(result.current.isLoadingTasks).toBe(false);
      });

      const request: CreateTaskRequest = {
        task_type: "investigate",
        focus: { harness: true, subject: false },
      };

      await act(async () => {
        const task = await result.current.trigger(request);
        expect(task).toEqual(newTask);
      });

      expect(mockCreateTask).toHaveBeenCalledWith("pipeline-456", request);

      // Wait for state update - the auto-track effect will pick up the running task
      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("new-task");
      });
    });

    it("does nothing when pipelineId is null", async () => {
      const { result } = renderHook(() => usePipelineInvestigation(null), {
        wrapper: createWrapper(),
      });

      await act(async () => {
        const task = await result.current.trigger({
          task_type: "investigate",
          focus: { harness: true, subject: false },
        });
        expect(task).toBeUndefined();
      });

      expect(mockCreateTask).not.toHaveBeenCalled();
    });

    it("sets isTriggering while creating", async () => {
      let resolveCreate: (value: Investigation) => void = () => {};
      mockCreateTask.mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveCreate = resolve;
          })
      );

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isAgentLoading).toBe(false);
      });

      expect(result.current.isTriggering).toBe(false);

      let triggerPromise: Promise<Investigation | undefined>;
      act(() => {
        triggerPromise = result.current.trigger({
          task_type: "investigate",
          focus: { harness: true, subject: false },
        });
      });

      await waitFor(() => {
        expect(result.current.isTriggering).toBe(true);
      });

      await act(async () => {
        resolveCreate(createMockInvestigation({ id: "new-task" }));
        await triggerPromise;
      });

      await waitFor(() => {
        expect(result.current.isTriggering).toBe(false);
      });
    });
  });

  describe("triggerFix", () => {
    it("creates a fix task from investigation", async () => {
      const newTask = createMockInvestigation({
        id: "fix-task",
        status: "running",
        details: { task_type: "fix" },
      });
      const newTaskSummary = createMockInvestigationSummary({
        id: "fix-task",
        status: "running",
        task_type: "fix",
      });

      // After creation, the task list will be refetched and include the new task
      mockListTasks
        .mockResolvedValueOnce([]) // Initial fetch
        .mockResolvedValue([newTaskSummary]); // After creation
      mockCreateTask.mockResolvedValue(newTask);
      mockGetTask.mockResolvedValue(newTask);

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isAgentLoading).toBe(false);
        expect(result.current.isLoadingTasks).toBe(false);
      });

      await act(async () => {
        const task = await result.current.triggerFix("source-investigation-123", {
          focus: { harness: true, subject: false },
          permissions: { immediate: true, permanent: false, prevention: false },
        });
        expect(task).toEqual(newTask);
      });

      expect(mockCreateTask).toHaveBeenCalledWith("pipeline-456", {
        task_type: "fix",
        source_investigation_id: "source-investigation-123",
        focus: { harness: true, subject: false },
        permissions: { immediate: true, permanent: false, prevention: false },
      });

      // Wait for state update - the auto-track effect will pick up the running task
      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("fix-task");
      });
    });
  });

  describe("stop", () => {
    it("stops the active task", async () => {
      const runningTask = createMockInvestigationSummary({
        id: "task-1",
        status: "running",
      });
      mockListTasks.mockResolvedValue([runningTask]);
      mockGetTask.mockResolvedValue(
        createMockInvestigation({ id: "task-1", status: "running" })
      );
      mockStopTask.mockResolvedValue(undefined);

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("task-1");
      });

      await act(async () => {
        await result.current.stop();
      });

      expect(mockStopTask).toHaveBeenCalledWith("pipeline-456", "task-1");
    });

    it("does nothing when no active task", async () => {
      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoadingTasks).toBe(false);
      });

      await act(async () => {
        await result.current.stop();
      });

      expect(mockStopTask).not.toHaveBeenCalled();
    });

    it("sets isStopping while stopping", async () => {
      const runningTask = createMockInvestigationSummary({
        id: "task-1",
        status: "running",
      });
      mockListTasks.mockResolvedValue([runningTask]);
      mockGetTask.mockResolvedValue(
        createMockInvestigation({ id: "task-1", status: "running" })
      );

      let resolveStop: () => void = () => {};
      mockStopTask.mockImplementation(
        () =>
          new Promise((resolve) => {
            resolveStop = () => resolve(undefined);
          })
      );

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("task-1");
      });

      expect(result.current.isStopping).toBe(false);

      let stopPromise: Promise<void>;
      act(() => {
        stopPromise = result.current.stop();
      });

      await waitFor(() => {
        expect(result.current.isStopping).toBe(true);
      });

      await act(async () => {
        resolveStop();
        await stopPromise;
      });

      await waitFor(() => {
        expect(result.current.isStopping).toBe(false);
      });
    });
  });

  describe("report modal", () => {
    it("opens report for specific task", async () => {
      const task1 = createMockInvestigationSummary({ id: "task-1" });
      const task2 = createMockInvestigationSummary({ id: "task-2" });
      mockListTasks.mockResolvedValue([task1, task2]);
      mockGetTask.mockResolvedValue(createMockInvestigation({ id: "task-1" }));

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoadingTasks).toBe(false);
      });

      expect(result.current.showReport).toBe(false);

      act(() => {
        result.current.viewReport("task-2");
      });

      expect(result.current.showReport).toBe(true);
      // activeTaskId is set to the viewed task
      expect(result.current.activeTaskId).toBe("task-2");
    });

    it("closes report modal", async () => {
      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isLoadingTasks).toBe(false);
      });

      act(() => {
        result.current.viewReport("task-1");
      });

      expect(result.current.showReport).toBe(true);

      act(() => {
        result.current.closeReport();
      });

      expect(result.current.showReport).toBe(false);
    });

    it("resets showReport when pipeline changes", async () => {
      const { result, rerender } = renderHook(
        (pipelineId: string | null) => usePipelineInvestigation(pipelineId),
        {
          wrapper: createWrapper(),
          initialProps: "pipeline-1",
        }
      );

      await waitFor(() => {
        expect(result.current.isLoadingTasks).toBe(false);
      });

      act(() => {
        result.current.viewReport("task-1");
      });

      expect(result.current.showReport).toBe(true);

      rerender("pipeline-2");

      expect(result.current.showReport).toBe(false);
    });
  });

  describe("refresh", () => {
    it("invalidates task queries", async () => {
      const task = createMockInvestigationSummary({ id: "task-1" });
      mockListTasks.mockResolvedValue([task]);
      mockGetTask.mockResolvedValue(createMockInvestigation({ id: "task-1" }));

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("task-1");
      });

      // Clear mock to track refresh calls
      mockListTasks.mockClear();
      mockGetTask.mockClear();

      act(() => {
        result.current.refresh();
      });

      // The queries should be invalidated and refetched
      await waitFor(() => {
        expect(mockListTasks).toHaveBeenCalled();
      });
    });
  });

  describe("error handling", () => {
    it("exposes trigger error", async () => {
      mockCreateTask.mockRejectedValue(new Error("Creation failed"));

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.isAgentLoading).toBe(false);
      });

      await act(async () => {
        try {
          await result.current.trigger({
            task_type: "investigate",
            focus: { harness: true, subject: false },
          });
        } catch {
          // Expected
        }
      });

      // Wait for error to be captured
      await waitFor(() => {
        expect(result.current.triggerError).toEqual(new Error("Creation failed"));
      });
    });

    it("exposes stop error", async () => {
      const task = createMockInvestigationSummary({ id: "task-1", status: "running" });
      mockListTasks.mockResolvedValue([task]);
      mockGetTask.mockResolvedValue(
        createMockInvestigation({ id: "task-1", status: "running" })
      );
      mockStopTask.mockRejectedValue(new Error("Stop failed"));

      const { result } = renderHook(() => usePipelineInvestigation("pipeline-456"), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.activeTaskId).toBe("task-1");
      });

      await act(async () => {
        try {
          await result.current.stop();
        } catch {
          // Expected
        }
      });

      // Wait for error to be captured
      await waitFor(() => {
        expect(result.current.stopError).toEqual(new Error("Stop failed"));
      });
    });
  });
});
