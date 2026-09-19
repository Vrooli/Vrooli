import { describe, it, expect } from "vitest";
import {
  mapPipelineToRunStatus,
  isTerminalState,
  shouldContinuePolling,
  generateRequestIdempotencyKey,
  categorizeError,
  getRecoverySuggestions,
  createPipelineErrorInfo,
  calculatePipelineProgress,
  getCurrentStage,
  getStageStatus,
  canResumePipeline,
  getStoppedAfterStage,
  TERMINAL_STATES,
  POLL_INTERVAL_MS,
} from "./pipeline.service";
import { create } from "@bufbuild/protobuf";
import { PipelineStatusSchema } from "@vrooli/proto-types/scenario-to-desktop/v1/pipeline/types_pb";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";

describe("pipeline.service", () => {
  describe("constants", () => {
    it("should define poll interval", () => {
      expect(POLL_INTERVAL_MS).toBe(2000);
    });

    it("should define terminal states", () => {
      expect(TERMINAL_STATES).toContain(StageStatus.COMPLETED);
      expect(TERMINAL_STATES).toContain(StageStatus.FAILED);
      expect(TERMINAL_STATES).toContain(StageStatus.CANCELLED);
    });
  });

  describe("mapPipelineToRunStatus", () => {
    it("returns idle for null/undefined", () => {
      expect(mapPipelineToRunStatus(undefined)).toBe("idle");
    });

    it("maps pending to running", () => {
      expect(mapPipelineToRunStatus(StageStatus.PENDING)).toBe("running");
    });

    it("maps running to running", () => {
      expect(mapPipelineToRunStatus(StageStatus.RUNNING)).toBe("running");
    });

    it("maps completed to completed", () => {
      expect(mapPipelineToRunStatus(StageStatus.COMPLETED)).toBe("completed");
    });

    it("maps failed to failed", () => {
      expect(mapPipelineToRunStatus(StageStatus.FAILED)).toBe("failed");
    });

    it("maps cancelled to cancelled", () => {
      expect(mapPipelineToRunStatus(StageStatus.CANCELLED)).toBe("cancelled");
    });

    it("maps unknown status to idle", () => {
      expect(mapPipelineToRunStatus(StageStatus.UNSPECIFIED)).toBe("idle");
    });
  });

  describe("isTerminalState", () => {
    it("returns false for null/undefined", () => {
      expect(isTerminalState(undefined)).toBe(false);
    });

    it("returns true for completed", () => {
      expect(isTerminalState(StageStatus.COMPLETED)).toBe(true);
    });

    it("returns true for failed", () => {
      expect(isTerminalState(StageStatus.FAILED)).toBe(true);
    });

    it("returns true for cancelled", () => {
      expect(isTerminalState(StageStatus.CANCELLED)).toBe(true);
    });

    it("returns false for pending", () => {
      expect(isTerminalState(StageStatus.PENDING)).toBe(false);
    });

    it("returns false for running", () => {
      expect(isTerminalState(StageStatus.RUNNING)).toBe(false);
    });
  });

  describe("shouldContinuePolling", () => {
    it("returns false for null status", () => {
      expect(shouldContinuePolling(null)).toBe(false);
    });

    it("returns false for completed status", () => {
      expect(
        shouldContinuePolling(
          create(PipelineStatusSchema, { status: StageStatus.COMPLETED }),
        ),
      ).toBe(false);
    });

    it("returns false for failed status", () => {
      expect(
        shouldContinuePolling(
          create(PipelineStatusSchema, { status: StageStatus.FAILED }),
        ),
      ).toBe(false);
    });

    it("returns true for running status", () => {
      expect(
        shouldContinuePolling(
          create(PipelineStatusSchema, { status: StageStatus.RUNNING }),
        ),
      ).toBe(true);
    });

    it("returns true for pending status", () => {
      expect(
        shouldContinuePolling(
          create(PipelineStatusSchema, { status: StageStatus.PENDING }),
        ),
      ).toBe(true);
    });
  });

  describe("categorizeError", () => {
    it("returns unknown for null/undefined", () => {
      expect(categorizeError(null)).toBe("unknown");
      expect(categorizeError(undefined)).toBe("unknown");
    });

    it("categorizes network errors", () => {
      expect(categorizeError(new Error("network error"))).toBe("network");
      expect(categorizeError(new Error("fetch failed"))).toBe("network");
      expect(categorizeError(new Error("failed to connect"))).toBe("network");
    });

    it("categorizes validation errors", () => {
      expect(categorizeError(new Error("validation failed"))).toBe(
        "validation",
      );
      expect(categorizeError(new Error("invalid input"))).toBe("validation");
      expect(categorizeError(new Error("field required"))).toBe("validation");
    });

    it("categorizes permission errors", () => {
      expect(categorizeError(new Error("permission denied"))).toBe(
        "permission",
      );
      expect(categorizeError(new Error("unauthorized access"))).toBe(
        "permission",
      );
    });

    it("categorizes timeout errors", () => {
      expect(categorizeError(new Error("request timeout"))).toBe("timeout");
      expect(categorizeError(new Error("operation timed out"))).toBe("timeout");
    });

    it("categorizes resource errors", () => {
      expect(categorizeError(new Error("out of memory"))).toBe("resource");
      expect(categorizeError(new Error("disk full"))).toBe("resource");
    });

    it("returns unknown for unrecognized errors", () => {
      expect(categorizeError(new Error("something went wrong"))).toBe(
        "unknown",
      );
    });
  });

  describe("getRecoverySuggestions", () => {
    it("provides suggestions for network errors", () => {
      const suggestions = getRecoverySuggestions("network");
      expect(suggestions).toContain("Check your internet connection");
    });

    it("provides suggestions for validation errors", () => {
      const suggestions = getRecoverySuggestions("validation");
      expect(suggestions).toContain("Review your form inputs");
    });

    it("provides suggestions for permission errors", () => {
      const suggestions = getRecoverySuggestions("permission");
      expect(suggestions).toContain("Check file and directory permissions");
    });

    it("provides suggestions for unknown errors", () => {
      const suggestions = getRecoverySuggestions("unknown");
      expect(suggestions.length).toBeGreaterThan(0);
    });
  });

  describe("createPipelineErrorInfo", () => {
    it("creates error info from Error object", () => {
      const error = new Error("Network connection failed");
      const info = createPipelineErrorInfo(error);
      expect(info.message).toBe("Network connection failed");
      expect(info.category).toBe("network");
      expect(info.suggestions).toBeDefined();
      expect(info.raw).toBe(error);
    });

    it("creates error info from string", () => {
      const info = createPipelineErrorInfo("Some error");
      expect(info.message).toBe("Some error");
    });
  });

  describe("calculatePipelineProgress", () => {
    it("returns 0 for null status", () => {
      expect(calculatePipelineProgress(null)).toBe(0);
    });

    it("returns 0 for status without stage_order", () => {
      expect(calculatePipelineProgress(create(PipelineStatusSchema))).toBe(0);
    });

    it("calculates progress based on completed stages", () => {
      const status = create(PipelineStatusSchema, {
        stageOrder: [
          StageName.BUNDLE,
          StageName.PREFLIGHT,
          StageName.GENERATE,
          StageName.BUILD,
        ],
        stages: {
          bundle: { stage: StageName.BUNDLE, status: StageStatus.COMPLETED },
          preflight: {
            stage: StageName.PREFLIGHT,
            status: StageStatus.COMPLETED,
          },
          generate: { stage: StageName.GENERATE, status: StageStatus.RUNNING },
          build: { stage: StageName.BUILD, status: StageStatus.PENDING },
        },
      });
      expect(calculatePipelineProgress(status)).toBe(0.5); // 2 of 4 completed
    });

    it("counts skipped stages as completed", () => {
      const status = create(PipelineStatusSchema, {
        stageOrder: [StageName.BUNDLE, StageName.PREFLIGHT],
        stages: {
          bundle: { stage: StageName.BUNDLE, status: StageStatus.COMPLETED },
          preflight: {
            stage: StageName.PREFLIGHT,
            status: StageStatus.SKIPPED,
          },
        },
      });
      expect(calculatePipelineProgress(status)).toBe(1); // Both done
    });
  });

  describe("getCurrentStage", () => {
    it("returns null for null status", () => {
      expect(getCurrentStage(null)).toBe(null);
    });

    it("returns current stage from status", () => {
      const status = create(PipelineStatusSchema, {
        currentStage: StageName.BUILD,
      });
      expect(getCurrentStage(status)).toBe(StageName.BUILD);
    });
  });

  describe("getStageStatus", () => {
    it("returns pending for null status", () => {
      expect(getStageStatus(null, "build")).toBe(StageStatus.PENDING);
    });

    it("returns stage status from stages map", () => {
      const status = create(PipelineStatusSchema, {
        stages: {
          build: { stage: StageName.BUILD, status: StageStatus.RUNNING },
        },
      });
      expect(getStageStatus(status, "build")).toBe(StageStatus.RUNNING);
    });

    it("returns pending for missing stage", () => {
      const status = create(PipelineStatusSchema);
      expect(getStageStatus(status, "build")).toBe(StageStatus.PENDING);
    });
  });

  describe("canResumePipeline", () => {
    it("returns false for null status", () => {
      expect(canResumePipeline(null)).toBe(false);
    });

    it("returns false for incomplete pipeline", () => {
      const status = create(PipelineStatusSchema, {
        status: StageStatus.RUNNING,
      });
      expect(canResumePipeline(status)).toBe(false);
    });

    it("returns true for completed pipeline with stopped_after_stage", () => {
      const status = create(PipelineStatusSchema, {
        status: StageStatus.COMPLETED,
        stoppedAfterStage: StageName.GENERATE,
      });
      expect(canResumePipeline(status)).toBe(true);
    });

    it("returns false for completed pipeline without stopped_after_stage", () => {
      const status = create(PipelineStatusSchema, {
        status: StageStatus.COMPLETED,
      });
      expect(canResumePipeline(status)).toBe(false);
    });
  });

  describe("getStoppedAfterStage", () => {
    it("returns null for null status", () => {
      expect(getStoppedAfterStage(null)).toBe(null);
    });

    it("returns stopped-after stage from status", () => {
      const status = create(PipelineStatusSchema, {
        stoppedAfterStage: StageName.GENERATE,
      });
      expect(getStoppedAfterStage(status)).toBe(StageName.GENERATE);
    });
  });

  describe("generateRequestIdempotencyKey", () => {
    it("generates key with scenario, stage, and session", () => {
      const key = generateRequestIdempotencyKey(
        "my-scenario",
        "generate",
        "session-123",
      );
      expect(key).toMatch(/^my-scenario:generate:session-123:\d+$/);
    });

    it("includes timestamp component", () => {
      const key = generateRequestIdempotencyKey("test", "build", "abc");
      const parts = key.split(":");
      expect(parts.length).toBe(4);
      const timestamp = parseInt(parts[3] ?? "", 10);
      expect(timestamp).toBeGreaterThan(Date.now() - 1000);
      expect(timestamp).toBeLessThanOrEqual(Date.now());
    });

    it("handles empty strings", () => {
      const key = generateRequestIdempotencyKey("", "", "");
      expect(key).toMatch(/^:::\d+$/);
    });

    it("handles special characters in scenario name", () => {
      const key = generateRequestIdempotencyKey(
        "my-test:scenario",
        "generate",
        "session",
      );
      expect(key).toContain("my-test:scenario");
    });
  });
});
