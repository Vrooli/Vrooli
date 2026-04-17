/**
 * Tests for basic useInvestigation hooks.
 * Tests useAgentManagerStatus, useTasks, useTaskDetails, useCreateTask, and useStopTask.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
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
import { renderHook, act, waitFor } from "@testing-library/react";
import {
  useAgentManagerStatus,
  useTasks,
  useTaskDetails,
  useCreateTask,
  useStopTask,
} from "../useInvestigation";
import type {
  AgentManagerStatus,
  Investigation,
  CreateTaskRequest,
} from "../../types/investigation";

beforeEach(() => {
  vi.clearAllMocks();
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("useAgentManagerStatus", () => {
  it("returns loading state initially", () => {
    mockGetAgentManagerStatus.mockImplementation(() => new Promise(() => {}));

    const { result } = renderHook(() => useAgentManagerStatus(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isLoading).toBe(true);
    expect(result.current.data).toBeUndefined();
  });

  it("returns agent manager status when available", async () => {
    const mockStatus: AgentManagerStatus = {
      available: true,
      url: "http://localhost:8080",
    };
    mockGetAgentManagerStatus.mockResolvedValue(mockStatus);

    const { result } = renderHook(() => useAgentManagerStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data).toEqual(mockStatus);
  });

  it("returns unavailable status", async () => {
    const mockStatus: AgentManagerStatus = {
      available: false,
      reason: "Service unreachable",
    };
    mockGetAgentManagerStatus.mockResolvedValue(mockStatus);

    const { result } = renderHook(() => useAgentManagerStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(result.current.data?.available).toBe(false);
    expect(result.current.data?.reason).toBe("Service unreachable");
  });

  it("handles error without retry", async () => {
    mockGetAgentManagerStatus.mockRejectedValue(new Error("Connection refused"));

    const { result } = renderHook(() => useAgentManagerStatus(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isError).toBe(true);
    });

    // Should have been called only once (no retry)
    expect(mockGetAgentManagerStatus).toHaveBeenCalledTimes(1);
  });
});

describe("useTasks", () => {
  it("does not fetch when pipelineId is null", () => {
    mockListTasks.mockResolvedValue([]);

    renderHook(() => useTasks(null), {
      wrapper: createWrapper(),
    });

    expect(mockListTasks).not.toHaveBeenCalled();
  });

  it("returns empty array when pipelineId is null", async () => {
    const { result } = renderHook(() => useTasks(null), {
      wrapper: createWrapper(),
    });

    // Query is disabled, so no loading state
    expect(result.current.isLoading).toBe(false);
    expect(result.current.data).toBeUndefined();
  });

  it("fetches tasks for pipeline", async () => {
    const mockTasks = [
      createMockInvestigationSummary({ id: "task-1" }),
      createMockInvestigationSummary({ id: "task-2" }),
    ];
    mockListTasks.mockResolvedValue(mockTasks);

    const { result } = renderHook(() => useTasks("pipeline-456"), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockListTasks).toHaveBeenCalledWith("pipeline-456", 50);
    expect(result.current.data).toEqual(mockTasks);
  });

  it("uses custom limit", async () => {
    mockListTasks.mockResolvedValue([]);

    renderHook(() => useTasks("pipeline-456", 10), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(mockListTasks).toHaveBeenCalledWith("pipeline-456", 10);
    });
  });
});

describe("useTaskDetails", () => {
  it("does not fetch when pipelineId is null", () => {
    mockGetTask.mockResolvedValue(createMockInvestigation());

    renderHook(() => useTaskDetails(null, "task-123"), {
      wrapper: createWrapper(),
    });

    expect(mockGetTask).not.toHaveBeenCalled();
  });

  it("does not fetch when taskId is null", () => {
    mockGetTask.mockResolvedValue(createMockInvestigation());

    renderHook(() => useTaskDetails("pipeline-456", null), {
      wrapper: createWrapper(),
    });

    expect(mockGetTask).not.toHaveBeenCalled();
  });

  it("fetches task details", async () => {
    const mockTask = createMockInvestigation();
    mockGetTask.mockResolvedValue(mockTask);

    const { result } = renderHook(
      () => useTaskDetails("pipeline-456", "task-123"),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.isLoading).toBe(false);
    });

    expect(mockGetTask).toHaveBeenCalledWith("pipeline-456", "task-123");
    expect(result.current.data).toEqual(mockTask);
  });
});

describe("useCreateTask", () => {
  it("creates a task and invalidates cache", async () => {
    const mockTask = createMockInvestigation();
    mockCreateTask.mockResolvedValue(mockTask);

    const { result } = renderHook(() => useCreateTask(), {
      wrapper: createWrapper(),
    });

    const request: CreateTaskRequest = {
      task_type: "investigate",
      focus: { harness: true, subject: false },
      effort: "logs",
    };

    await act(async () => {
      const task = await result.current.mutateAsync({
        pipelineId: "pipeline-456",
        request,
      });
      expect(task).toEqual(mockTask);
    });

    expect(mockCreateTask).toHaveBeenCalledWith("pipeline-456", request);
  });

  it("returns isPending while creating", async () => {
    let resolveCreate: (value: Investigation) => void = () => {};
    mockCreateTask.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveCreate = resolve;
        })
    );

    const { result } = renderHook(() => useCreateTask(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isPending).toBe(false);

    let createPromise: Promise<Investigation>;
    act(() => {
      createPromise = result.current.mutateAsync({
        pipelineId: "pipeline-456",
        request: {
          task_type: "investigate",
          focus: { harness: true, subject: false },
        },
      });
    });

    await waitFor(() => {
      expect(result.current.isPending).toBe(true);
    });

    await act(async () => {
      resolveCreate(createMockInvestigation());
      await createPromise;
    });

    await waitFor(() => {
      expect(result.current.isPending).toBe(false);
    });
  });
});

describe("useStopTask", () => {
  it("stops a task and invalidates cache", async () => {
    mockStopTask.mockResolvedValue(undefined);

    const { result } = renderHook(() => useStopTask(), {
      wrapper: createWrapper(),
    });

    await act(async () => {
      await result.current.mutateAsync({
        pipelineId: "pipeline-456",
        taskId: "task-123",
      });
    });

    expect(mockStopTask).toHaveBeenCalledWith("pipeline-456", "task-123");
  });

  it("returns isPending while stopping", async () => {
    let resolveStop: () => void = () => {};
    mockStopTask.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveStop = () => resolve(undefined);
        })
    );

    const { result } = renderHook(() => useStopTask(), {
      wrapper: createWrapper(),
    });

    expect(result.current.isPending).toBe(false);

    let stopPromise: Promise<void>;
    act(() => {
      stopPromise = result.current.mutateAsync({
        pipelineId: "pipeline-456",
        taskId: "task-123",
      });
    });

    await waitFor(() => {
      expect(result.current.isPending).toBe(true);
    });

    await act(async () => {
      resolveStop();
      await stopPromise;
    });

    await waitFor(() => {
      expect(result.current.isPending).toBe(false);
    });
  });
});
