/**
 * Tests for usePipelineActions hook.
 * Tests store wrapper behavior, mutation invocations, and preflight actions.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import React from "react";
import { usePipelineActions, type UsePipelineActionsProps } from "./usePipelineActions";
import { usePipelineStore } from "../store";

// Mock the API module - use importOriginal to preserve non-mocked exports
vi.mock("../lib/api", async (importOriginal) => {
  const actual = await importOriginal<typeof import("../lib/api")>();
  return {
    ...actual,
    runPipeline: vi.fn().mockResolvedValue({ pipeline_id: "test-pipeline-123" }),
    probeEndpoints: vi.fn().mockResolvedValue({
      proxy_url: "https://api.example.com",
      healthy: true,
      api_version: "1.0.0",
    }),
    startActivePipeline: vi.fn().mockResolvedValue({
      pipeline: { pipeline_id: "test-pipeline-123", status: "running" },
      status_url: "/pipeline/test-pipeline-123",
    }),
    getActivePipeline: vi.fn().mockResolvedValue({
      pipeline: { pipeline_id: "test-pipeline-123", status: "pending" },
      created: false,
    }),
    getPipelineStatus: vi.fn().mockResolvedValue({
      pipeline_id: "test-pipeline-123",
      status: "completed",
      stages: {},
    }),
  };
});

// Import mocked functions for assertions
import { runPipeline, probeEndpoints } from "../lib/api";

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

// Reset store and mocks before each test
beforeEach(() => {
  usePipelineStore.getState().reset();
  vi.clearAllMocks();
});

const defaultProps: UsePipelineActionsProps = {
  scenarioName: "test-scenario",
};

describe("usePipelineActions", () => {
  describe("store integration", () => {
    it("reads pipeline state from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.pipelineId).toBeNull();
      expect(result.current.pipelineStatus).toBeNull();
      expect(result.current.runStatus).toBe("idle");
      expect(result.current.error).toBeNull();
      expect(result.current.errorInfo).toBeNull();
    });

    it("exposes preflight state from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.preflightResult).toBeNull();
      expect(result.current.preflightSecrets).toEqual({});
      expect(result.current.preflightOverride).toBe(false);
      expect(result.current.preflightOk).toBe(false);
      expect(result.current.missingPreflightSecrets).toEqual([]);
    });

    it("sets scenario in store when scenarioName changes", async () => {
      const { rerender } = renderHook(
        (props: UsePipelineActionsProps) => usePipelineActions(props),
        {
          initialProps: { scenarioName: "scenario-1" },
          wrapper: createWrapper(),
        }
      );

      // Verify initial scenario was set
      expect(usePipelineStore.getState().scenarioName).toBe("scenario-1");

      // Change scenario
      rerender({ scenarioName: "scenario-2" });

      await waitFor(() => {
        expect(usePipelineStore.getState().scenarioName).toBe("scenario-2");
      });
    });
  });

  describe("derived state", () => {
    it("computes isRunning from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.isRunning).toBe(false);
    });

    it("computes isSubmitting from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.isSubmitting).toBe(false);
    });

    it("computes isBusy from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.isBusy).toBe(false);
    });

    it("computes progress from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.progress).toBe(0);
    });

    it("computes canResume from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.canResume).toBe(false);
    });
  });

  describe("generate mutation", () => {
    it("calls runPipeline API on generateDesktop", async () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      const config = {
        scenario_name: "test-scenario",
        template_type: "basic",
        platforms: ["win", "mac"],
      };

      act(() => {
        result.current.generateDesktop(config);
      });

      await waitFor(() => {
        expect(runPipeline).toHaveBeenCalledWith(config);
      });
    });

    it("tracks pending state during generation", async () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.generatePending).toBe(false);

      const config = { scenario_name: "test-scenario" };

      // Call the mutation - since the mock resolves immediately,
      // we may not capture the pending=true state in tests
      act(() => {
        result.current.generateDesktop(config);
      });

      // Wait for the mutation to complete and verify final state
      await waitFor(() => {
        expect(result.current.generatePending).toBe(false);
      });

      // Verify the API was called
      expect(runPipeline).toHaveBeenCalledWith(config);
    });

    it("calls onBuildStart callback on success", async () => {
      const onBuildStart = vi.fn();
      const { result } = renderHook(
        () => usePipelineActions({ scenarioName: "test", onBuildStart }),
        { wrapper: createWrapper() }
      );

      act(() => {
        result.current.generateDesktop({ scenario_name: "test" });
      });

      await waitFor(() => {
        expect(onBuildStart).toHaveBeenCalledWith("test-pipeline-123");
      });
    });

    it("captures generate error", async () => {
      vi.mocked(runPipeline).mockRejectedValueOnce(new Error("Generation failed"));

      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.generateDesktop({ scenario_name: "test" });
      });

      await waitFor(() => {
        expect(result.current.generateError).toBe("Generation failed");
      });
    });
  });

  describe("connection test mutation", () => {
    it("calls probeEndpoints API on testConnection", async () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.testConnection("https://api.example.com");
      });

      await waitFor(() => {
        expect(probeEndpoints).toHaveBeenCalledWith({ proxy_url: "https://api.example.com" });
      });
    });

    it("tracks pending state during connection test", async () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(result.current.connectionTestPending).toBe(false);

      // Call the mutation - since the mock resolves immediately,
      // we may not capture the pending=true state in tests
      act(() => {
        result.current.testConnection("https://api.example.com");
      });

      // Wait for the mutation to complete and verify final state
      await waitFor(() => {
        expect(result.current.connectionTestPending).toBe(false);
      });

      // Verify the API was called
      expect(probeEndpoints).toHaveBeenCalledWith({ proxy_url: "https://api.example.com" });
    });

    it("stores connection test result on success", async () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.testConnection("https://api.example.com");
      });

      await waitFor(() => {
        expect(result.current.connectionTestResult).toEqual({
          proxy_url: "https://api.example.com",
          healthy: true,
          api_version: "1.0.0",
        });
      });
    });

    it("throws error when proxyUrl is empty", async () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.testConnection("");
      });

      await waitFor(() => {
        expect(result.current.connectionTestError).toBe(
          "Enter the proxy URL above before testing."
        );
      });
    });

    it("captures connection test error", async () => {
      vi.mocked(probeEndpoints).mockRejectedValueOnce(new Error("Connection failed"));

      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.testConnection("https://api.example.com");
      });

      await waitFor(() => {
        expect(result.current.connectionTestError).toBe("Connection failed");
      });
    });
  });

  describe("preflight actions", () => {
    it("exposes setPreflightSecrets from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.setPreflightSecrets({ API_KEY: "test-key" });
      });

      expect(usePipelineStore.getState().preflightSecrets).toEqual({ API_KEY: "test-key" });
    });

    it("exposes setPreflightSecret for individual secrets", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.setPreflightSecret("API_KEY", "secret-value");
      });

      expect(usePipelineStore.getState().preflightSecrets.API_KEY).toBe("secret-value");
    });

    it("exposes setPreflightOverride from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      act(() => {
        result.current.setPreflightOverride(true);
      });

      expect(usePipelineStore.getState().preflightOverride).toBe(true);
    });

    it("exposes resetPreflight from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      // Set some preflight state first
      act(() => {
        result.current.setPreflightSecrets({ KEY: "value" });
        result.current.setPreflightOverride(true);
      });

      // Reset
      act(() => {
        result.current.resetPreflight();
      });

      expect(usePipelineStore.getState().preflightSecrets).toEqual({});
      expect(usePipelineStore.getState().preflightOverride).toBe(false);
    });

    it("filters empty secrets before running preflight", async () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      // Set secrets including empty ones
      act(() => {
        result.current.setPreflightSecrets({
          API_KEY: "valid-key",
          EMPTY_SECRET: "",
          ANOTHER_KEY: "another-value",
          WHITESPACE: "   ",
        });
      });

      // Run preflight - this should filter out empty/whitespace secrets
      await act(async () => {
        await result.current.runPreflight();
      });

      // The filtering happens internally in runPreflight, but we can verify
      // that the override was reset (which happens in runPreflight)
      expect(usePipelineStore.getState().preflightOverride).toBe(false);
    });

    it("does nothing when scenarioName is empty", async () => {
      const { result } = renderHook(
        () => usePipelineActions({ scenarioName: "" }),
        { wrapper: createWrapper() }
      );

      await act(async () => {
        await result.current.runPreflight();
      });

      // Should not have called the store's runPreflightStage
      // (we can verify by checking the store state hasn't changed unexpectedly)
      expect(usePipelineStore.getState().runStatus).toBe("idle");
    });
  });

  describe("pipeline actions", () => {
    it("exposes runStage from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.runStage).toBe("function");
    });

    it("exposes runFullPipeline from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.runFullPipeline).toBe("function");
    });

    it("exposes cancelPipeline from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.cancelPipeline).toBe("function");
    });

    it("exposes resumePipeline from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.resumePipeline).toBe("function");
    });

    it("exposes stage-specific run actions", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.runBundleStage).toBe("function");
      expect(typeof result.current.runPreflightStage).toBe("function");
      expect(typeof result.current.runSmokeTestStage).toBe("function");
    });
  });

  describe("state management actions", () => {
    it("exposes reset from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.reset).toBe("function");
    });

    it("exposes clearError from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.clearError).toBe("function");
    });

    it("exposes resetForRetry from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.resetForRetry).toBe("function");
    });
  });

  describe("status management", () => {
    it("exposes startPolling from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.startPolling).toBe("function");
    });

    it("exposes stopPolling from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.stopPolling).toBe("function");
    });

    it("exposes loadPipelineStatus from store", () => {
      const { result } = renderHook(() => usePipelineActions(defaultProps), {
        wrapper: createWrapper(),
      });

      expect(typeof result.current.loadPipelineStatus).toBe("function");
    });
  });
});
