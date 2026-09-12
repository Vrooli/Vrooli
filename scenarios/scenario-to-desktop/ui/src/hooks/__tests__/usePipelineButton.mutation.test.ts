/**
 * Tests for usePipelineMutation and usePipelineStatus hooks.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import {
  mockRunPipeline,
  mockGetPipelineStatus,
  createWrapper,
  localStorageMock,
} from "./usePipelineButton.testUtils";
import { renderHook, act, waitFor } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  PipelineRunResponseSchema,
  PipelineStatusSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import { usePipelineMutation, usePipelineStatus } from "../usePipelineButton";
import type {
  PipelineRunResponse,
  PipelineConfig,
  PipelineStatus,
} from "../../lib/api";
import {
  Platform,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

const pipelineRunResponse = (
  pipelineId = "build-123",
  message = "Pipeline started",
) => create(PipelineRunResponseSchema, { pipelineId, message });

const pipelineStatus = (status: StageStatus) =>
  create(PipelineStatusSchema, { pipelineId: "build-123", status });

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
    const mockResponse: PipelineRunResponse = pipelineRunResponse();
    mockRunPipeline.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    const config: PipelineConfig = {
      scenarioName: "test-scenario",
      platforms: [Platform.WIN],
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
        scenarioName: "test-scenario",
        platforms: [Platform.WIN],
      });
    });

    await waitFor(() => {
      expect(result.current.state.error).toBe("Pipeline failed");
    });

    expect(result.current.state.buildId).toBeNull();
  });

  it("calls onSuccess callback", async () => {
    const mockResponse: PipelineRunResponse = pipelineRunResponse();
    mockRunPipeline.mockResolvedValue(mockResponse);

    const onSuccess = vi.fn();

    const { result } = renderHook(() => usePipelineMutation({ onSuccess }), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.runPipelineWithConfig({
        scenarioName: "test-scenario",
        platforms: [Platform.WIN],
      });
    });

    await waitFor(() => {
      expect(onSuccess).toHaveBeenCalledWith(mockResponse);
    });
  });

  it("calls onError callback", async () => {
    mockRunPipeline.mockRejectedValue(new Error("Pipeline failed"));

    const onError = vi.fn();

    const { result } = renderHook(() => usePipelineMutation({ onError }), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.runPipelineWithConfig({
        scenarioName: "test-scenario",
        platforms: [Platform.WIN],
      });
    });

    await waitFor(() => {
      expect(onError).toHaveBeenCalledWith(new Error("Pipeline failed"));
    });
  });

  it("reset clears state", async () => {
    const mockResponse: PipelineRunResponse = pipelineRunResponse();
    mockRunPipeline.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.runPipelineWithConfig({
        scenarioName: "test-scenario",
        platforms: [Platform.WIN],
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
    const mockResponse: PipelineRunResponse = pipelineRunResponse();
    mockRunPipeline.mockResolvedValue(mockResponse);

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.runPipelineWithConfig({
        scenarioName: "test-scenario",
        platforms: [Platform.WIN],
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
        }),
    );

    const { result } = renderHook(() => usePipelineMutation(), {
      wrapper: createWrapper(),
    });

    expect(result.current.mutation.isPending).toBe(false);

    act(() => {
      result.current.runPipelineWithConfig({
        scenarioName: "test-scenario",
        platforms: [Platform.WIN],
      });
    });

    await waitFor(() => {
      expect(result.current.mutation.isPending).toBe(true);
    });

    await act(async () => {
      resolveRun(pipelineRunResponse("build-123", "Started"));
    });

    await waitFor(() => {
      expect(result.current.mutation.isPending).toBe(false);
    });
  });
});

describe("usePipelineStatus", () => {
  it("returns null when no buildId", () => {
    mockGetPipelineStatus.mockResolvedValue(null);

    const { result } = renderHook(() => usePipelineStatus({ buildId: null }), {
      wrapper: createWrapper(),
    });

    expect(result.current.pipelineStatus).toBeUndefined();
    expect(result.current.mappedStatus).toBeNull();
    expect(result.current.isBuilding).toBe(false);
    expect(result.current.isComplete).toBe(false);
    expect(result.current.isFailed).toBe(false);
  });

  it("fetches status when buildId is provided", async () => {
    const mockStatus: PipelineStatus = create(PipelineStatusSchema, {
      pipelineId: "build-123",
      status: StageStatus.RUNNING,
      progressMessage: "Building...",
    });
    mockGetPipelineStatus.mockResolvedValue(mockStatus);

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.pipelineStatus).toEqual(mockStatus);
    });
  });

  it("maps running status to building", async () => {
    mockGetPipelineStatus.mockResolvedValue(
      pipelineStatus(StageStatus.RUNNING),
    );

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("building");
    });

    expect(result.current.isBuilding).toBe(true);
    expect(result.current.isComplete).toBe(false);
    expect(result.current.isFailed).toBe(false);
  });

  it("maps pending status to building", async () => {
    mockGetPipelineStatus.mockResolvedValue(
      pipelineStatus(StageStatus.PENDING),
    );

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("building");
    });

    expect(result.current.isBuilding).toBe(true);
  });

  it("maps completed status to ready", async () => {
    mockGetPipelineStatus.mockResolvedValue(
      pipelineStatus(StageStatus.COMPLETED),
    );

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("ready");
    });

    expect(result.current.isBuilding).toBe(false);
    expect(result.current.isComplete).toBe(true);
    expect(result.current.isFailed).toBe(false);
  });

  it("maps failed status to failed", async () => {
    mockGetPipelineStatus.mockResolvedValue(pipelineStatus(StageStatus.FAILED));

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("failed");
    });

    expect(result.current.isBuilding).toBe(false);
    expect(result.current.isComplete).toBe(false);
    expect(result.current.isFailed).toBe(true);
  });

  it("maps cancelled status to failed", async () => {
    mockGetPipelineStatus.mockResolvedValue(
      pipelineStatus(StageStatus.CANCELLED),
    );

    const { result } = renderHook(
      () => usePipelineStatus({ buildId: "build-123" }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.mappedStatus).toBe("failed");
    });

    expect(result.current.isFailed).toBe(true);
  });

  it("uses custom query key prefix", async () => {
    mockGetPipelineStatus.mockResolvedValue(
      pipelineStatus(StageStatus.RUNNING),
    );

    const { result } = renderHook(
      () =>
        usePipelineStatus({
          buildId: "build-123",
          queryKeyPrefix: "custom-status",
        }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.pipelineStatus).toBeDefined();
    });
  });

  it("fetches verbose status when requested", async () => {
    mockGetPipelineStatus.mockResolvedValue(
      create(PipelineStatusSchema, {
        pipelineId: "build-123",
        status: StageStatus.RUNNING,
        stages: { bundle: { stage: 1, status: StageStatus.COMPLETED } },
      }),
    );

    const { result } = renderHook(
      () =>
        usePipelineStatus({
          buildId: "build-123",
          verbose: true,
        }),
      { wrapper: createWrapper() },
    );

    await waitFor(() => {
      expect(result.current.pipelineStatus).toBeDefined();
    });

    expect(mockGetPipelineStatus).toHaveBeenCalledWith("build-123", {
      verbose: true,
    });
  });
});
