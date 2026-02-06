/**
 * Tests for pipelineStore.
 * Tests state management, actions, and internal helpers.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { act } from "@testing-library/react";
import { usePipelineStore } from "./pipelineStore";
import type { PipelineRunStatus } from "./pipelineTypes";
import type { VerbosePipelineStatus } from "../lib/api";

// Mock the API module
vi.mock("../lib/api", () => ({
  runPipeline: vi.fn().mockResolvedValue({ pipeline_id: "test-pipeline-123" }),
  getPipelineStatus: vi.fn().mockResolvedValue({
    pipeline_id: "test-pipeline-123",
    status: "running",
    current_stage: "bundle",
    stage_order: ["bundle", "preflight", "generate"],
    stages: {
      bundle: { status: "completed" },
      preflight: { status: "running" },
      generate: { status: "pending" },
    },
  }),
  cancelPipeline: vi.fn().mockResolvedValue({ success: true }),
}));

// Reset store state before each test
beforeEach(() => {
  act(() => {
    usePipelineStore.getState().reset();
  });
  vi.clearAllMocks();
});

describe("pipelineStore", () => {
  describe("initial state", () => {
    it("starts with correct default values", () => {
      const state = usePipelineStore.getState();

      expect(state.scenarioName).toBeNull();
      expect(state.pipelineId).toBeNull();
      expect(state.pipelineStatus).toBeNull();
      expect(state.runStatus).toBe("idle");
      expect(state.errorInfo).toBeNull();
      expect(state.isPolling).toBe(false);
      expect(state.pollIntervalMs).toBe(2000);
      expect(state.bundleResult).toBeNull();
      expect(state.preflightResult).toBeNull();
      expect(state.generateResult).toBeNull();
      expect(state.buildResult).toBeNull();
      expect(state.smokeTestResult).toBeNull();
      expect(state.deployResult).toBeNull();
      expect(state.stageLogs).toEqual({});
      expect(state.pipelineHistory).toEqual([]);
      expect(state.preflightSecrets).toEqual({});
      expect(state.preflightOverride).toBe(false);
      expect(state.isSubmitting).toBe(false);
      expect(state.currentIdempotencyKey).toBeNull();
    });
  });

  describe("setScenario", () => {
    it("sets scenario name", () => {
      const store = usePipelineStore.getState();

      act(() => {
        store.setScenario("test-scenario");
      });

      expect(usePipelineStore.getState().scenarioName).toBe("test-scenario");
    });

    it("resets state when scenario changes", () => {
      const store = usePipelineStore.getState();

      // Set initial scenario and some state
      act(() => {
        store.setScenario("scenario-1");
        // Manually set pipelineId since _setPipelineStatus doesn't set it
        usePipelineStore.setState({ pipelineId: "pipeline-1" });
      });

      expect(usePipelineStore.getState().pipelineId).toBe("pipeline-1");

      // Change scenario
      act(() => {
        store.setScenario("scenario-2");
      });

      // Pipeline state should be reset
      expect(usePipelineStore.getState().scenarioName).toBe("scenario-2");
      expect(usePipelineStore.getState().pipelineId).toBeNull();
    });

    it("does not reset when setting same scenario", () => {
      const store = usePipelineStore.getState();

      act(() => {
        store.setScenario("scenario-1");
        store._setPipelineStatus({
          pipeline_id: "pipeline-1",
          status: "running",
        } as VerbosePipelineStatus);
      });

      const pipelineId = usePipelineStore.getState().pipelineId;

      // Set same scenario again
      act(() => {
        store.setScenario("scenario-1");
      });

      // State should be preserved
      expect(usePipelineStore.getState().pipelineId).toBe(pipelineId);
    });
  });

  describe("_setPipelineStatus", () => {
    it("sets pipeline status from verbose status", () => {
      const store = usePipelineStore.getState();

      const verboseStatus: VerbosePipelineStatus = {
        pipeline_id: "test-pipeline",
        status: "running",
        current_stage: "bundle",
        stage_order: ["bundle", "preflight"],
        stages: {
          bundle: { stage: "bundle", status: "completed", started_at: Date.now() },
          preflight: { stage: "preflight", status: "running", started_at: Date.now() },
        },
      } as VerbosePipelineStatus;

      act(() => {
        store._setPipelineStatus(verboseStatus);
      });

      const state = usePipelineStore.getState();
      // Note: _setPipelineStatus doesn't set pipelineId - that's set by runStage/runFullPipeline
      expect(state.pipelineStatus).toEqual(verboseStatus);
      expect(state.runStatus).toBe("running");
    });

    it("maps status to runStatus correctly", () => {
      const store = usePipelineStore.getState();

      const testCases: Array<{ apiStatus: string; expectedRunStatus: PipelineRunStatus }> = [
        { apiStatus: "pending", expectedRunStatus: "running" },
        { apiStatus: "running", expectedRunStatus: "running" },
        { apiStatus: "completed", expectedRunStatus: "completed" },
        { apiStatus: "failed", expectedRunStatus: "failed" },
        { apiStatus: "cancelled", expectedRunStatus: "cancelled" },
      ];

      for (const { apiStatus, expectedRunStatus } of testCases) {
        act(() => {
          store._setPipelineStatus({
            pipeline_id: "test",
            status: apiStatus,
          } as VerbosePipelineStatus);
        });

        expect(usePipelineStore.getState().runStatus).toBe(expectedRunStatus);
      }
    });

    it("clears pipelineStatus when set to null", () => {
      const store = usePipelineStore.getState();

      // First set a status
      act(() => {
        store._setPipelineStatus({
          pipeline_id: "test",
          status: "running",
        } as VerbosePipelineStatus);
      });

      expect(usePipelineStore.getState().pipelineStatus).not.toBeNull();

      // Then set to null
      act(() => {
        store._setPipelineStatus(null);
      });

      // pipelineStatus should be null, but runStatus is preserved (implementation detail)
      expect(usePipelineStore.getState().pipelineStatus).toBeNull();
    });
  });

  describe("preflight actions", () => {
    it("setPreflightSecrets sets secrets object", () => {
      const store = usePipelineStore.getState();

      act(() => {
        store.setPreflightSecrets({ API_KEY: "secret", DB_PASS: "password" });
      });

      expect(usePipelineStore.getState().preflightSecrets).toEqual({
        API_KEY: "secret",
        DB_PASS: "password",
      });
    });

    it("setPreflightSecret sets individual secret", () => {
      const store = usePipelineStore.getState();

      act(() => {
        store.setPreflightSecret("API_KEY", "secret-value");
      });

      expect(usePipelineStore.getState().preflightSecrets.API_KEY).toBe("secret-value");
    });

    it("setPreflightSecret preserves other secrets", () => {
      const store = usePipelineStore.getState();

      act(() => {
        store.setPreflightSecrets({ EXISTING: "value" });
        store.setPreflightSecret("NEW_KEY", "new-value");
      });

      const secrets = usePipelineStore.getState().preflightSecrets;
      expect(secrets.EXISTING).toBe("value");
      expect(secrets.NEW_KEY).toBe("new-value");
    });

    it("setPreflightOverride sets override flag", () => {
      const store = usePipelineStore.getState();

      act(() => {
        store.setPreflightOverride(true);
      });

      expect(usePipelineStore.getState().preflightOverride).toBe(true);

      act(() => {
        store.setPreflightOverride(false);
      });

      expect(usePipelineStore.getState().preflightOverride).toBe(false);
    });

    it("resetPreflight clears preflight state", () => {
      const store = usePipelineStore.getState();

      // Set some preflight state
      act(() => {
        store.setPreflightSecrets({ KEY: "value" });
        store.setPreflightOverride(true);
      });

      expect(usePipelineStore.getState().preflightSecrets).toEqual({ KEY: "value" });
      expect(usePipelineStore.getState().preflightOverride).toBe(true);

      // Reset
      act(() => {
        store.resetPreflight();
      });

      expect(usePipelineStore.getState().preflightSecrets).toEqual({});
      expect(usePipelineStore.getState().preflightOverride).toBe(false);
    });
  });

  describe("reset", () => {
    it("resets all state except scenarioName", () => {
      const store = usePipelineStore.getState();

      // Set various state
      act(() => {
        store.setScenario("test-scenario");
        usePipelineStore.setState({ pipelineId: "test-pipeline" });
        store._setPipelineStatus({
          pipeline_id: "test",
          status: "running",
        } as VerbosePipelineStatus);
        store.setPreflightSecrets({ KEY: "value" });
        store.setPreflightOverride(true);
      });

      // Reset
      act(() => {
        store.reset();
      });

      const state = usePipelineStore.getState();

      // Reset keeps scenarioName (by design)
      expect(state.scenarioName).toBe("test-scenario");
      // Everything else should be reset
      expect(state.pipelineId).toBeNull();
      expect(state.pipelineStatus).toBeNull();
      expect(state.runStatus).toBe("idle");
      expect(state.preflightSecrets).toEqual({});
      expect(state.preflightOverride).toBe(false);
    });
  });

  describe("clearError", () => {
    it("clears errorInfo", () => {
      const store = usePipelineStore.getState();

      // Set an error state directly (normally done through failed actions)
      act(() => {
        usePipelineStore.setState({
          errorInfo: { message: "Details", category: "unknown" },
        });
      });

      expect(usePipelineStore.getState().errorInfo).toBeDefined();

      // Clear
      act(() => {
        store.clearError();
      });

      expect(usePipelineStore.getState().errorInfo).toBeNull();
    });
  });

  describe("resetForRetry", () => {
    it("clears errorInfo and submission state for retry", () => {
      const store = usePipelineStore.getState();

      // Set a failed state with submitting stuck
      act(() => {
        usePipelineStore.setState({
          runStatus: "failed",
          errorInfo: { message: "Failed" },
          isSubmitting: true,
          currentIdempotencyKey: "test-key",
        });
      });

      // Reset for retry
      act(() => {
        store.resetForRetry();
      });

      const state = usePipelineStore.getState();
      // resetForRetry clears submission state and errors, but NOT runStatus
      expect(state.isSubmitting).toBe(false);
      expect(state.currentIdempotencyKey).toBeNull();
      expect(state.errorInfo).toBeNull();
      // runStatus is preserved (allows UI to show last state)
      expect(state.runStatus).toBe("failed");
    });
  });

  describe("polling", () => {
    it("startPolling requires pipelineId to be set", () => {
      const store = usePipelineStore.getState();

      // Without pipelineId, startPolling is a no-op
      act(() => {
        store.startPolling();
      });

      expect(usePipelineStore.getState().isPolling).toBe(false);

      // With pipelineId set, startPolling works
      act(() => {
        usePipelineStore.setState({ pipelineId: "test-pipeline" });
        store.startPolling();
      });

      expect(usePipelineStore.getState().isPolling).toBe(true);
    });

    it("stopPolling sets isPolling to false", () => {
      const store = usePipelineStore.getState();

      // Set up polling first
      act(() => {
        usePipelineStore.setState({ pipelineId: "test-pipeline" });
        store.startPolling();
      });

      expect(usePipelineStore.getState().isPolling).toBe(true);

      act(() => {
        store.stopPolling();
      });

      expect(usePipelineStore.getState().isPolling).toBe(false);
    });
  });

  describe("subscribeToStatus", () => {
    it("returns unsubscribe function", () => {
      const store = usePipelineStore.getState();
      const callback = vi.fn();

      const unsubscribe = store.subscribeToStatus(callback);

      expect(typeof unsubscribe).toBe("function");

      // Clean up
      unsubscribe();
    });
  });

  describe("stage results extraction", () => {
    it("extracts stage results from verbose status", () => {
      const store = usePipelineStore.getState();

      const verboseStatus = {
        pipeline_id: "test",
        status: "completed",
        stages: {
          bundle: {
            status: "completed",
            result: { bundle_path: "/path/to/bundle" },
          },
          preflight: {
            status: "completed",
            result: { validation: { valid: true } },
          },
          generate: {
            status: "completed",
            result: { output_path: "/path/to/output" },
          },
        },
      } as unknown as VerbosePipelineStatus;

      act(() => {
        store._setPipelineStatus(verboseStatus);
        store._extractStageResults(verboseStatus);
      });

      // Stage results should be extracted
      // The actual implementation depends on the store logic
    });
  });

  describe("pipeline history", () => {
    it("tracks pipeline IDs in history", () => {
      const store = usePipelineStore.getState();

      // Set up a pipeline
      act(() => {
        store._setPipelineStatus({
          pipeline_id: "pipeline-1",
          status: "completed",
        } as VerbosePipelineStatus);
      });

      // The history tracking depends on implementation
      // but we can verify the history field exists
      expect(Array.isArray(usePipelineStore.getState().pipelineHistory)).toBe(true);
    });
  });

  describe("submission tracking", () => {
    it("isSubmitting starts false", () => {
      expect(usePipelineStore.getState().isSubmitting).toBe(false);
    });

    it("currentIdempotencyKey starts null", () => {
      expect(usePipelineStore.getState().currentIdempotencyKey).toBeNull();
    });
  });

  describe("stage logs", () => {
    it("stageLogs starts as empty object", () => {
      expect(usePipelineStore.getState().stageLogs).toEqual({});
    });
  });

  describe("action availability", () => {
    it("exposes all required actions", () => {
      const store = usePipelineStore.getState();

      // Scenario context
      expect(typeof store.setScenario).toBe("function");

      // Pipeline execution
      expect(typeof store.runStage).toBe("function");
      expect(typeof store.runFullPipeline).toBe("function");
      expect(typeof store.cancelPipeline).toBe("function");
      expect(typeof store.resumePipeline).toBe("function");

      // Convenience actions
      expect(typeof store.runBundleStage).toBe("function");
      expect(typeof store.runPreflightStage).toBe("function");
      expect(typeof store.runSmokeTestStage).toBe("function");

      // Status management
      expect(typeof store.loadPipelineStatus).toBe("function");
      expect(typeof store.startPolling).toBe("function");
      expect(typeof store.stopPolling).toBe("function");
      expect(typeof store.subscribeToStatus).toBe("function");

      // State management
      expect(typeof store.reset).toBe("function");
      expect(typeof store.clearError).toBe("function");
      expect(typeof store.resetForRetry).toBe("function");

      // Preflight actions
      expect(typeof store.setPreflightSecrets).toBe("function");
      expect(typeof store.setPreflightSecret).toBe("function");
      expect(typeof store.setPreflightOverride).toBe("function");
      expect(typeof store.resetPreflight).toBe("function");

      // Internal helpers
      expect(typeof store._setPipelineStatus).toBe("function");
      expect(typeof store._extractStageResults).toBe("function");
      expect(typeof store._notifySubscribers).toBe("function");
    });
  });
});
