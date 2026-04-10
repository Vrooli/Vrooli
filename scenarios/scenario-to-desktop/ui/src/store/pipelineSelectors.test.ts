/**
 * Tests for pipelineSelectors.
 * Tests all derived state selectors for the pipeline store.
 */

import { describe, it, expect } from "vitest";
import {
  selectIsRunning,
  selectCurrentStage,
  selectProgress,
  selectStageStatus,
  selectCanResume,
  selectStoppedAfterStage,
  selectIsSubmitting,
  selectIsBusy,
  selectPreflightValidationOk,
  selectPreflightReadinessOk,
  selectMissingSecrets,
  selectPreflightSecretsOk,
  selectPreflightOk,
  selectBundleResult,
  selectPreflightResult,
  selectGenerateResult,
  selectBuildResult,
  selectSmokeTestResult,
  selectDeployResult,
  selectStageLogs,
  selectError,
  selectErrorInfo,
  selectHasError,
  selectPipelineHistory,
  selectLatestPipelineId,
  selectPreflightSecrets,
  selectPreflightOverride,
} from "./pipelineSelectors";
import { initialPipelineState, type PipelineStore } from "./pipelineTypes";
import type { VerbosePipelineStatus } from "../lib/api";

// Helper to create test state with overrides
function createTestState(overrides: Partial<PipelineStore> = {}): PipelineStore {
  return {
    ...initialPipelineState,
    ...overrides,
    // Stub actions for type compatibility
    setScenario: () => {},
    runStage: async () => "",
    runFullPipeline: async () => "",
    cancelPipeline: async () => {},
    resumePipeline: async () => "",
    runBundleStage: async () => "",
    runPreflightStage: async () => "",
    runSmokeTestStage: async () => "",
    loadPipelineStatus: async () => {},
    startPolling: () => {},
    stopPolling: () => {},
    subscribeToStatus: () => () => {},
    reset: () => {},
    clearError: () => {},
    resetForRetry: () => {},
    setPreflightSecrets: () => {},
    setPreflightSecret: () => {},
    setPreflightOverride: () => {},
    resetPreflight: () => {},
    _setPipelineStatus: () => {},
    _extractStageResults: () => {},
    _notifySubscribers: () => {},
  } as PipelineStore;
}

describe("pipelineSelectors", () => {
  describe("selectIsRunning", () => {
    it("returns false for idle status", () => {
      const state = createTestState({ runStatus: "idle" });
      expect(selectIsRunning(state)).toBe(false);
    });

    it("returns true for running status", () => {
      const state = createTestState({ runStatus: "running" });
      expect(selectIsRunning(state)).toBe(true);
    });

    it("returns true for starting status", () => {
      const state = createTestState({ runStatus: "starting" });
      expect(selectIsRunning(state)).toBe(true);
    });

    it("returns false for completed status", () => {
      const state = createTestState({ runStatus: "completed" });
      expect(selectIsRunning(state)).toBe(false);
    });

    it("returns false for failed status", () => {
      const state = createTestState({ runStatus: "failed" });
      expect(selectIsRunning(state)).toBe(false);
    });

    it("returns false for cancelled status", () => {
      const state = createTestState({ runStatus: "cancelled" });
      expect(selectIsRunning(state)).toBe(false);
    });
  });

  describe("selectCurrentStage", () => {
    it("returns null when no pipeline status", () => {
      const state = createTestState({ pipelineStatus: null });
      expect(selectCurrentStage(state)).toBeNull();
    });

    it("returns current_stage from pipeline status", () => {
      const state = createTestState({
        pipelineStatus: {
          current_stage: "build",
        } as VerbosePipelineStatus,
      });
      expect(selectCurrentStage(state)).toBe("build");
    });
  });

  describe("selectProgress", () => {
    it("returns 0 when no pipeline status", () => {
      const state = createTestState({ pipelineStatus: null });
      expect(selectProgress(state)).toBe(0);
    });

    it("returns 0 when no stage_order", () => {
      const state = createTestState({
        pipelineStatus: {} as VerbosePipelineStatus,
      });
      expect(selectProgress(state)).toBe(0);
    });

    it("returns 0 when stage_order is empty", () => {
      const state = createTestState({
        pipelineStatus: {
          stage_order: [],
          stages: {},
        } as unknown as VerbosePipelineStatus,
      });
      expect(selectProgress(state)).toBe(0);
    });

    it("calculates progress based on completed stages", () => {
      const state = createTestState({
        pipelineStatus: {
          stage_order: ["bundle", "preflight", "generate", "build"],
          stages: {
            bundle: { status: "completed" },
            preflight: { status: "completed" },
            generate: { status: "running" },
            build: { status: "pending" },
          },
        } as unknown as VerbosePipelineStatus,
      });
      expect(selectProgress(state)).toBe(0.5); // 2 of 4 completed
    });

    it("counts skipped stages as completed", () => {
      const state = createTestState({
        pipelineStatus: {
          stage_order: ["bundle", "preflight", "generate"],
          stages: {
            bundle: { status: "completed" },
            preflight: { status: "skipped" },
            generate: { status: "completed" },
          },
        } as unknown as VerbosePipelineStatus,
      });
      expect(selectProgress(state)).toBe(1); // All done
    });

    it("returns 0 when all stages are pending", () => {
      const state = createTestState({
        pipelineStatus: {
          stage_order: ["bundle", "preflight"],
          stages: {
            bundle: { status: "pending" },
            preflight: { status: "pending" },
          },
        } as unknown as VerbosePipelineStatus,
      });
      expect(selectProgress(state)).toBe(0);
    });
  });

  describe("selectStageStatus", () => {
    it("returns pending when no pipeline status", () => {
      const state = createTestState({ pipelineStatus: null });
      const selector = selectStageStatus("build");
      expect(selector(state)).toBe("pending");
    });

    it("returns stage status from stages map", () => {
      const state = createTestState({
        pipelineStatus: {
          stages: {
            build: { status: "running" },
          },
        } as unknown as VerbosePipelineStatus,
      });
      const selector = selectStageStatus("build");
      expect(selector(state)).toBe("running");
    });

    it("returns pending for missing stage", () => {
      const state = createTestState({
        pipelineStatus: {
          stages: {
            bundle: { status: "completed" },
          },
        } as unknown as VerbosePipelineStatus,
      });
      const selector = selectStageStatus("build");
      expect(selector(state)).toBe("pending");
    });
  });

  describe("selectCanResume", () => {
    it("returns false when no pipeline status", () => {
      const state = createTestState({ pipelineStatus: null });
      expect(selectCanResume(state)).toBe(false);
    });

    it("returns false when pipeline not completed", () => {
      const state = createTestState({
        pipelineStatus: {
          status: "running",
          stopped_after_stage: "generate",
        } as VerbosePipelineStatus,
      });
      expect(selectCanResume(state)).toBe(false);
    });

    it("returns true when completed with stopped_after_stage", () => {
      const state = createTestState({
        pipelineStatus: {
          status: "completed",
          stopped_after_stage: "generate",
        } as VerbosePipelineStatus,
      });
      expect(selectCanResume(state)).toBe(true);
    });

    it("returns false when completed without stopped_after_stage", () => {
      const state = createTestState({
        pipelineStatus: {
          status: "completed",
        } as VerbosePipelineStatus,
      });
      expect(selectCanResume(state)).toBe(false);
    });
  });

  describe("selectStoppedAfterStage", () => {
    it("returns null when no pipeline status", () => {
      const state = createTestState({ pipelineStatus: null });
      expect(selectStoppedAfterStage(state)).toBeNull();
    });

    it("returns stopped_after_stage from status", () => {
      const state = createTestState({
        pipelineStatus: {
          stopped_after_stage: "generate",
        } as VerbosePipelineStatus,
      });
      expect(selectStoppedAfterStage(state)).toBe("generate");
    });
  });

  describe("selectIsSubmitting", () => {
    it("returns false when not submitting", () => {
      const state = createTestState({ isSubmitting: false });
      expect(selectIsSubmitting(state)).toBe(false);
    });

    it("returns true when submitting", () => {
      const state = createTestState({ isSubmitting: true });
      expect(selectIsSubmitting(state)).toBe(true);
    });
  });

  describe("selectIsBusy", () => {
    it("returns false when idle and not submitting", () => {
      const state = createTestState({ isSubmitting: false, runStatus: "idle" });
      expect(selectIsBusy(state)).toBe(false);
    });

    it("returns true when submitting", () => {
      const state = createTestState({ isSubmitting: true, runStatus: "idle" });
      expect(selectIsBusy(state)).toBe(true);
    });

    it("returns true when running", () => {
      const state = createTestState({ isSubmitting: false, runStatus: "running" });
      expect(selectIsBusy(state)).toBe(true);
    });

    it("returns true when starting", () => {
      const state = createTestState({ isSubmitting: false, runStatus: "starting" });
      expect(selectIsBusy(state)).toBe(true);
    });

    it("returns false when completed", () => {
      const state = createTestState({ isSubmitting: false, runStatus: "completed" });
      expect(selectIsBusy(state)).toBe(false);
    });
  });

  describe("preflight selectors", () => {
    describe("selectPreflightValidationOk", () => {
      it("returns false when no preflight result", () => {
        const state = createTestState({ preflightResult: null });
        expect(selectPreflightValidationOk(state)).toBe(false);
      });

      it("returns false when validation is invalid", () => {
        const state = createTestState({
          preflightResult: {
            validation: { valid: false },
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightValidationOk(state)).toBe(false);
      });

      it("returns true when validation is valid", () => {
        const state = createTestState({
          preflightResult: {
            validation: { valid: true },
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightValidationOk(state)).toBe(true);
      });
    });

    describe("selectPreflightReadinessOk", () => {
      it("returns false when no preflight result", () => {
        const state = createTestState({ preflightResult: null });
        expect(selectPreflightReadinessOk(state)).toBe(false);
      });

      it("returns false when not ready", () => {
        const state = createTestState({
          preflightResult: {
            ready: { ready: false },
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightReadinessOk(state)).toBe(false);
      });

      it("returns true when ready", () => {
        const state = createTestState({
          preflightResult: {
            ready: { ready: true },
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightReadinessOk(state)).toBe(true);
      });
    });

    describe("selectMissingSecrets", () => {
      it("returns empty array when no preflight result", () => {
        const state = createTestState({ preflightResult: null });
        expect(selectMissingSecrets(state)).toEqual([]);
      });

      it("returns empty array when no secrets in result", () => {
        const state = createTestState({
          preflightResult: {} as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectMissingSecrets(state)).toEqual([]);
      });

      it("returns only required secrets without values", () => {
        const state = createTestState({
          preflightResult: {
            secrets: [
              { id: "API_KEY", required: true, has_value: false },
              { id: "OPTIONAL", required: false, has_value: false },
              { id: "PROVIDED", required: true, has_value: true },
            ],
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        const missing = selectMissingSecrets(state);
        expect(missing).toHaveLength(1);
        expect(missing?.[0]?.id).toBe("API_KEY");
      });

      it("returns stable empty array reference", () => {
        const state = createTestState({ preflightResult: null });
        const result1 = selectMissingSecrets(state);
        const result2 = selectMissingSecrets(state);
        expect(result1).toBe(result2); // Same reference
      });
    });

    describe("selectPreflightSecretsOk", () => {
      it("returns true when no missing secrets", () => {
        const state = createTestState({ preflightResult: null });
        expect(selectPreflightSecretsOk(state)).toBe(true);
      });

      it("returns false when missing secrets", () => {
        const state = createTestState({
          preflightResult: {
            secrets: [{ id: "API_KEY", required: true, has_value: false }],
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightSecretsOk(state)).toBe(false);
      });
    });

    describe("selectPreflightOk", () => {
      it("returns false when no preflight result", () => {
        const state = createTestState({ preflightResult: null });
        expect(selectPreflightOk(state)).toBe(false);
      });

      it("returns false when validation fails", () => {
        const state = createTestState({
          preflightResult: {
            status: "completed",
            validation: { valid: false },
            ready: { ready: true, details: {} },
            secrets: [],
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightOk(state)).toBe(false);
      });

      it("returns false when readiness fails", () => {
        const state = createTestState({
          preflightResult: {
            status: "completed",
            validation: { valid: true },
            ready: { ready: false, details: {} },
            secrets: [],
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightOk(state)).toBe(false);
      });

      it("returns false when missing required secrets", () => {
        const state = createTestState({
          preflightResult: {
            validation: { valid: true },
            ready: { ready: true },
            secrets: [{ id: "API_KEY", required: true, has_value: false }],
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightOk(state)).toBe(false);
      });

      it("returns true when all checks pass", () => {
        const state = createTestState({
          preflightResult: {
            validation: { valid: true },
            ready: { ready: true },
            secrets: [{ id: "API_KEY", required: true, has_value: true }],
          } as ReturnType<typeof createTestState>["preflightResult"],
        });
        expect(selectPreflightOk(state)).toBe(true);
      });
    });

    describe("selectPreflightSecrets", () => {
      it("returns preflight secrets from state", () => {
        const state = createTestState({
          preflightSecrets: { API_KEY: "secret" },
        });
        expect(selectPreflightSecrets(state)).toEqual({ API_KEY: "secret" });
      });
    });

    describe("selectPreflightOverride", () => {
      it("returns preflight override from state", () => {
        const state = createTestState({ preflightOverride: true });
        expect(selectPreflightOverride(state)).toBe(true);
      });
    });
  });

  describe("stage result selectors", () => {
    it("selectBundleResult returns bundleResult", () => {
      const bundleResult = { bundle_path: "/path" } as ReturnType<typeof createTestState>["bundleResult"];
      const state = createTestState({ bundleResult });
      expect(selectBundleResult(state)).toBe(bundleResult);
    });

    it("selectPreflightResult returns preflightResult", () => {
      const preflightResult = { validation: {} } as ReturnType<typeof createTestState>["preflightResult"];
      const state = createTestState({ preflightResult });
      expect(selectPreflightResult(state)).toBe(preflightResult);
    });

    it("selectGenerateResult returns generateResult", () => {
      const generateResult = { output_path: "/path" } as ReturnType<typeof createTestState>["generateResult"];
      const state = createTestState({ generateResult });
      expect(selectGenerateResult(state)).toBe(generateResult);
    });

    it("selectBuildResult returns buildResult", () => {
      const buildResult = { artifacts: {} } as ReturnType<typeof createTestState>["buildResult"];
      const state = createTestState({ buildResult });
      expect(selectBuildResult(state)).toBe(buildResult);
    });

    it("selectSmokeTestResult returns smokeTestResult", () => {
      const smokeTestResult = { passed: true } as ReturnType<typeof createTestState>["smokeTestResult"];
      const state = createTestState({ smokeTestResult });
      expect(selectSmokeTestResult(state)).toBe(smokeTestResult);
    });

    it("selectDeployResult returns deployResult", () => {
      const deployResult = { update_url: "https://example.com" } as ReturnType<typeof createTestState>["deployResult"];
      const state = createTestState({ deployResult });
      expect(selectDeployResult(state)).toBe(deployResult);
    });
  });

  describe("selectStageLogs", () => {
    it("returns empty array for missing stage", () => {
      const state = createTestState({ stageLogs: {} });
      const selector = selectStageLogs("build");
      expect(selector(state)).toEqual([]);
    });

    it("returns logs for specific stage", () => {
      const state = createTestState({
        stageLogs: {
          build: ["Log 1", "Log 2"],
        },
      });
      const selector = selectStageLogs("build");
      expect(selector(state)).toEqual(["Log 1", "Log 2"]);
    });
  });

  describe("error selectors", () => {
    describe("selectError", () => {
      it("returns error message from errorInfo", () => {
        const state = createTestState({
          errorInfo: { message: "Something went wrong" },
        });
        expect(selectError(state)).toBe("Something went wrong");
      });

      it("returns null when no errorInfo", () => {
        const state = createTestState({ errorInfo: null });
        expect(selectError(state)).toBeNull();
      });
    });

    describe("selectErrorInfo", () => {
      it("returns errorInfo from state", () => {
        const errorInfo = { message: "Error", category: "network" as const };
        const state = createTestState({ errorInfo });
        expect(selectErrorInfo(state)).toEqual(errorInfo);
      });
    });

    describe("selectHasError", () => {
      it("returns false when no errorInfo", () => {
        const state = createTestState({ errorInfo: null });
        expect(selectHasError(state)).toBe(false);
      });

      it("returns true when errorInfo exists", () => {
        const state = createTestState({
          errorInfo: { message: "Error" },
        });
        expect(selectHasError(state)).toBe(true);
      });
    });
  });

  describe("history selectors", () => {
    describe("selectPipelineHistory", () => {
      it("returns pipeline history", () => {
        const state = createTestState({
          pipelineHistory: ["pipeline-1", "pipeline-2"],
        });
        expect(selectPipelineHistory(state)).toEqual(["pipeline-1", "pipeline-2"]);
      });
    });

    describe("selectLatestPipelineId", () => {
      it("returns null when history is empty", () => {
        const state = createTestState({ pipelineHistory: [] });
        expect(selectLatestPipelineId(state)).toBeNull();
      });

      it("returns latest pipeline ID from history", () => {
        const state = createTestState({
          pipelineHistory: ["pipeline-1", "pipeline-2", "pipeline-3"],
        });
        expect(selectLatestPipelineId(state)).toBe("pipeline-3");
      });
    });
  });
});
