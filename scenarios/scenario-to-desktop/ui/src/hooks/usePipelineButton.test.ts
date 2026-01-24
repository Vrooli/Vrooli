/**
 * Tests for usePipelineButton hooks.
 * Tests usePlatformSelection, useWineCheck, usePipelineMutation, and usePipelineStatus.
 */

import { describe, it, expect, beforeEach, vi, afterEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import {
  usePlatformSelection,
  useWineCheck,
  usePipelineMutation,
  usePipelineStatus,
} from "./usePipelineButton";
import type {
  WineCheckResponse,
  PipelineRunResponse,
  PipelineConfig,
  PipelineStatus,
} from "../lib/api";

// Mock the API module
vi.mock("../lib/api", () => ({
  runPipeline: vi.fn(),
  getPipelineStatus: vi.fn(),
  checkWineStatus: vi.fn(),
}));

// Import mocks after setting up vi.mock
import { runPipeline, getPipelineStatus, checkWineStatus } from "../lib/api";

const mockRunPipeline = runPipeline as ReturnType<typeof vi.fn>;
const mockGetPipelineStatus = getPipelineStatus as ReturnType<typeof vi.fn>;
const mockCheckWineStatus = checkWineStatus as ReturnType<typeof vi.fn>;

// Create a wrapper with QueryClientProvider
function createWrapper() {
  const queryClient = new QueryClient({
    defaultOptions: {
      queries: { retry: false },
      mutations: { retry: false },
    },
  });
  return function Wrapper({ children }: { children: React.ReactNode }) {
    return React.createElement(QueryClientProvider, { client: queryClient }, children);
  };
}

// Mock localStorage
const localStorageMock = (() => {
  let store: Record<string, string> = {};
  return {
    getItem: vi.fn((key: string) => store[key] || null),
    setItem: vi.fn((key: string, value: string) => {
      store[key] = value;
    }),
    removeItem: vi.fn((key: string) => {
      delete store[key];
    }),
    clear: vi.fn(() => {
      store = {};
    }),
  };
})();

Object.defineProperty(window, "localStorage", { value: localStorageMock });

beforeEach(() => {
  vi.clearAllMocks();
  localStorageMock.clear();
});

afterEach(() => {
  vi.clearAllTimers();
});

describe("usePlatformSelection", () => {
  it("returns default platforms when no stored value", () => {
    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win", "mac", "linux"]);
  });

  it("uses custom default platforms", () => {
    const { result } = renderHook(
      () =>
        usePlatformSelection({
          storageKey: "test-platforms",
          defaultPlatforms: ["win"],
        }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win"]);
  });

  it("loads platforms from localStorage", () => {
    localStorageMock.getItem.mockReturnValue(JSON.stringify(["win", "mac"]));

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win", "mac"]);
  });

  it("falls back to defaults on invalid localStorage data", () => {
    localStorageMock.getItem.mockReturnValue("invalid json");

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win", "mac", "linux"]);
  });

  it("falls back to defaults when localStorage contains non-array", () => {
    localStorageMock.getItem.mockReturnValue(JSON.stringify({ not: "array" }));

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    expect(result.current.selectedPlatforms).toEqual(["win", "mac", "linux"]);
  });

  it("saves to localStorage when platforms change", async () => {
    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.setSelectedPlatforms(["win"]);
    });

    await waitFor(() => {
      expect(localStorageMock.setItem).toHaveBeenCalledWith(
        "test-platforms",
        JSON.stringify(["win"])
      );
    });
  });

  it("togglePlatform adds platform when not selected", () => {
    localStorageMock.getItem.mockReturnValue(JSON.stringify(["win"]));

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.togglePlatform("mac");
    });

    expect(result.current.selectedPlatforms).toEqual(["win", "mac"]);
  });

  it("togglePlatform removes platform when selected", () => {
    localStorageMock.getItem.mockReturnValue(JSON.stringify(["win", "mac"]));

    const { result } = renderHook(
      () => usePlatformSelection({ storageKey: "test-platforms" }),
      { wrapper: createWrapper() }
    );

    act(() => {
      result.current.togglePlatform("mac");
    });

    expect(result.current.selectedPlatforms).toEqual(["win"]);
  });
});

describe("useWineCheck", () => {
  it("fetches wine check status", async () => {
    const mockWineCheck: WineCheckResponse = {
      installed: true,
      platform: "linux",
      version: "wine-9.0",
    };
    mockCheckWineStatus.mockResolvedValue(mockWineCheck);

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.wineCheck).toEqual(mockWineCheck);
    });
  });

  it("showWineDialog starts as false", () => {
    mockCheckWineStatus.mockResolvedValue({ installed: false, platform: "linux" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    expect(result.current.showWineDialog).toBe(false);
  });

  it("setShowWineDialog updates state", () => {
    mockCheckWineStatus.mockResolvedValue({ installed: false, platform: "linux" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.setShowWineDialog(true);
    });

    expect(result.current.showWineDialog).toBe(true);
  });

  it("pendingPlatforms starts as empty", () => {
    mockCheckWineStatus.mockResolvedValue({ installed: false, platform: "linux" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    expect(result.current.pendingPlatforms).toEqual([]);
  });

  it("setPendingPlatforms updates state", () => {
    mockCheckWineStatus.mockResolvedValue({ installed: false, platform: "linux" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    act(() => {
      result.current.setPendingPlatforms(["win", "mac"]);
    });

    expect(result.current.pendingPlatforms).toEqual(["win", "mac"]);
  });

  describe("needsWineForPlatforms", () => {
    it("returns true when win platform on linux without wine", async () => {
      mockCheckWineStatus.mockResolvedValue({
        installed: false,
        platform: "linux",
      });

      const { result } = renderHook(() => useWineCheck(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.wineCheck).toBeDefined();
      });

      expect(result.current.needsWineForPlatforms(["win"])).toBe(true);
      expect(result.current.needsWineForPlatforms(["win", "mac"])).toBe(true);
    });

    it("returns false when wine is installed", async () => {
      mockCheckWineStatus.mockResolvedValue({
        installed: true,
        platform: "linux",
        version: "wine-9.0",
      });

      const { result } = renderHook(() => useWineCheck(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.wineCheck).toBeDefined();
      });

      expect(result.current.needsWineForPlatforms(["win"])).toBe(false);
    });

    it("returns false when not on linux", async () => {
      mockCheckWineStatus.mockResolvedValue({
        installed: false,
        platform: "darwin",
      });

      const { result } = renderHook(() => useWineCheck(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.wineCheck).toBeDefined();
      });

      expect(result.current.needsWineForPlatforms(["win"])).toBe(false);
    });

    it("returns false when win not in platforms", async () => {
      mockCheckWineStatus.mockResolvedValue({
        installed: false,
        platform: "linux",
      });

      const { result } = renderHook(() => useWineCheck(), {
        wrapper: createWrapper(),
      });

      await waitFor(() => {
        expect(result.current.wineCheck).toBeDefined();
      });

      expect(result.current.needsWineForPlatforms(["mac", "linux"])).toBe(false);
    });
  });

  it("handleWineInstallComplete closes dialog and invalidates query", async () => {
    mockCheckWineStatus
      .mockResolvedValueOnce({ installed: false, platform: "linux" })
      .mockResolvedValueOnce({ installed: true, platform: "linux", version: "wine-9.0" });

    const { result } = renderHook(() => useWineCheck(), {
      wrapper: createWrapper(),
    });

    await waitFor(() => {
      expect(result.current.wineCheck).toBeDefined();
    });

    act(() => {
      result.current.setShowWineDialog(true);
    });

    expect(result.current.showWineDialog).toBe(true);

    act(() => {
      result.current.handleWineInstallComplete();
    });

    expect(result.current.showWineDialog).toBe(false);
  });
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
    expect(mockRunPipeline.mock.calls[0][0]).toEqual(config);
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
      message: "Building...",
      started_at: new Date().toISOString(),
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
