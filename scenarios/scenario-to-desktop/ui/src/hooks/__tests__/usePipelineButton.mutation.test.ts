/**
 * Tests for usePipelineMutation and usePipelineStatus hooks.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import {
  usePipelineMutation,
  usePipelineStatus,
} from "../usePipelineButton";
import type {
  PipelineRunResponse,
  PipelineConfig,
  PipelineStatus,
} from "../../lib/api";
import {
  mockRunPipeline,
  mockGetPipelineStatus,
  createWrapper,
  localStorageMock,
} from "./usePipelineButton.testUtils";

beforeEach(() => {
  vi.clearAllMocks();
  localStorageMock.clear();
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("usePipelineMutation", () => {
  it("starts with null buildId and error", () => {
    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    expect(result.current.state.buildId).toBeNull();
    expect(result.current.state.error).toBeNull();
  });

  it("sets buildId on successful run", async () => {
    const mockResponse: PipelineRunResponse = {
      pipeline_id: "build-123",
      message: "Pipeline started",
    };
    mockRunPipeline.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    const config: PipelineConfig = {
      scenario_name: "test-scenario",
      platforms: ["win"],
    };

    act(() => {
      result.current.runPipelineWithConfig(config);
    });

    await waitFor(() => {
      expect(result.current.state.buildId).toBe("build-123");
    });

    expect(result.current.state.error).toBeNull();
    // First argument to mutationFn is the config
    expect(mockRunPipeline.mock.calls?.[0]?.[0]).toEqual(config);
  });

  it("sets error on failed run", async () => {
    mockRunPipeline.mockRejectedValue(new Error("Pipeline failed"));

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.runPipelineWithConfig({
        scenario_name: "test-scenario",
        platforms: ["win"],
      });
    });

    await waitFor(() => {
      expect(result.current.state.error).toBe("Pipeline failed");
    });

    expect(result.current.state.buildId).toBeNull();
  });

  it("calls onSuccess callback", async () => {
    const mockResponse: PipelineRunResponse = {
      pipeline_id: "build-123",
      message: "Pipeline started",
    };
    mockRunPipeline.mockResolvedValue(mockResponse);

    const onSuccess = vi.fn();

    const { result } = renderHook(
      () => usePipelineMutation({ onSuccess }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.runPipelineWithConfig({
        scenario_name: "test-scenario",
        platforms: ["win"],
      });
    });

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith(mockResponse);
    });
  });

  it("calls onError callback", async () => {
    mockRunPipeline.mockRejectedValue(new Error("Pipeline failed"));

    const onError = vi.fn();

    const { result } = renderHook(
      () => usePipelineMutation({ onError }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.runPipelineWithConfig({
        scenario_name: "test-scenario",
        platforms: ["win"],
      });
    });

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith(new Error("Pipeline failed"));
    });
  });

  it("reset clears state", async () => {
    const mockResponse: PipelineRunResponse = {
      pipeline_id: "build-123",
      message: "Pipeline started",
    };
    mockRunPipeline.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.runPipelineWithConfig({
        scenario_name: "test-scenario",
        platforms: ["win"],
      });
    });

    await waitFor(() => {
      expect(result.current.state.buildId).toBe("build-123");
    });

    act(() => {
      result.current.reset();
    });

    expect(result.current.state.buildId).toBeNull();
    expect(result.current.state.error).toBeNull();
  });

  it("clearBuildId only clears buildId", async () => {
    const mockResponse: PipelineRunResponse = {
      pipeline_id: "build-123",
      message: "Pipeline started",
    };
    mockRunPipeline.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.runPipelineWithConfig({
        scenario_name: "test-scenario",
        platforms: ["win"],
      });
    });

    await waitFor(() => {
      expect(result.current.state.buildId).toBe("build-123");
    });

    act(() => {
      result.current.clearBuildId();
    });

    expect(result.current.state.buildId).toBeNull();
    // Mutation state should still be available
    expect(result.current.mutation.isSuccess).toBe(true);
  });

  it("mutation exposes isPending state", async () => {
    let resolveRun: (value: PipelineRunResponse) => void = () => {};
    mockRunPipeline.mockImplementation(
      () =>
        new Promise((resolve) => {
          resolveRun = resolve;
        })
    );

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    expect(result.current.mutation.isPending).toBe(false);

    act(() => {
      result.current.runPipelineWithConfig({
        scenario_name: "test-scenario",
        platforms: ["win"],
      });
    });

    await waitFor(() => {
      expect(result.current.mutation.isPending).toBe(true);
    });

    await act(async () => {
      resolveRun({ pipeline_id: "build-123", message: "Started" });
    });

    await waitFor(() => {
      expect(result.current.mutation.isPending).toBe(false);
    });
  });
});

describe("usePipelineStatus", () => {
  it("returns null when no buildId", () => {
    mockGetPipelineStatus.mockResolvedValue(null);

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: null }),
      { wrapper: createWrapper() }
    );

    expect(result.current.pipelineStatus).toBeUndefined();
    expect(result.current.mappedStatus).toBeNull();
    expect(result.current.isBuilding).toBe(false);
    expect(result.current.isComplete).toBe(false);
    expect(result.current.isFailed).toBe(false);
  });

  it("fetches status when buildId is provided", async () => {
    const mockStatus: PipelineStatus = {
      pipeline_id: "build-123",
      status: "running",
      progress_message: "Building...",
      started_at: Date.now(),
    };
    mockGetPipelineStatus.mockResolvedValue(mockStatus);

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.pipelineStatus).toEqual(mockStatus);
    });
  });

  it("maps running status to building", async () => {
    mockGetPipelineStatus.mockResolvedValue({
      pipeline_id: "build-123",
      status: "running",
      message: "Building...",
      started_at: new Date().toISOString(),
    });

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("building");
    });

    expect(result.current.isBuilding).toBe(true);
    expect(result.current.isComplete).toBe(false);
    expect(result.current.isFailed).toBe(false);
  });

  it("maps pending status to building", async () => {
    mockGetPipelineStatus.mockResolvedValue({
      pipeline_id: "build-123",
      status: "pending",
      message: "Queued",
      started_at: new Date().toISOString(),
    });

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("building");
    });

    expect(result.current.isBuilding).toBe(true);
  });

  it("maps completed status to ready", async () => {
    mockGetPipelineStatus.mockResolvedValue({
      pipeline_id: "build-123",
      status: "completed",
      message: "Done",
      started_at: new Date().toISOString(),
      completed_at: new Date().toISOString(),
    });

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("ready");
    });

    expect(result.current.isBuilding).toBe(false);
    expect(result.current.isComplete).toBe(true);
    expect(result.current.isFailed).toBe(false);
  });

  it("maps failed status to failed", async () => {
    mockGetPipelineStatus.mockResolvedValue({
      pipeline_id: "build-123",
      status: "failed",
      message: "Build error",
      started_at: new Date().toISOString(),
      error: "Compilation failed",
    });

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("failed");
    });

    expect(result.current.isBuilding).toBe(false);
    expect(result.current.isComplete).toBe(false);
    expect(result.current.isFailed).toBe(true);
  });

  it("maps cancelled status to failed", async () => {
    mockGetPipelineStatus.mockResolvedValue({
      pipeline_id: "build-123",
      status: "cancelled",
      message: "Cancelled by user",
      started_at: new Date().toISOString(),
    });

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("failed");
    });

    expect(result.current.isFailed).toBe(true);
  });

  it("uses custom query key prefix", async () => {
    mockGetPipelineStatus.mockResolvedValue({
      pipeline_id: "build-123",
      status: "running",
      message: "Building...",
      started_at: new Date().toISOString(),
    });

    const { result } = renderHook(
      () =>
        usePipelineStatus({
          buildId: "build-123",
          queryKeyPrefix: "custom-status",
        }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.pipelineStatus).toBeDefined();
    });
  });

  it("fetches verbose status when requested", async () => {
    mockGetPipelineStatus.mockResolvedValue({
      pipeline_id: "build-123",
      status: "running",
      message: "Building...",
      started_at: new Date().toISOString(),
      stages: [{ name: "bundle", status: "completed" }],
    });

    const { result } = renderHook(
      () =>
        usePipelineStatus({
          buildId: "build-123",
          verbose: true,
        }),
      { wrapper: createWrapper() }
    );

    await waitFor(() => {
      expect(result.current.pipelineStatus).toBeDefined();
    });

    expect(mockGetPipelineStatus).toHaveBeenCalledWith("build-123", { verbose: true });
  });
});
