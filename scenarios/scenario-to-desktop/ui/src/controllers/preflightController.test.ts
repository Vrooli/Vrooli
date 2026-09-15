import { describe, it, expect } from "vitest";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  buildPreflightPipelineConfig,
  validatePreflightConfig,
  buildJobStepMap,
  resolveAllStepStatuses,
  buildPreflightSectionState,
  exportPreflightAsJson,
  createPreflightJsonBlob,
  isPreflightReadyForGeneration,
  getPreflightBlockingReason,
  type PreflightRunConfig,
} from "./preflightController";
import type { VerbosePipelineStatus } from "../lib/api";
import {
  StageName,
  StageStatus,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/common_pb";
import { createPipelineStatus } from "../test-utils/mocks";
import type { PreflightJobStep } from "../lib/preflight-status";
import {
  PreflightResponseSchema,
  PreflightSecretSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";

const preflight = (init: MessageInitShape<typeof PreflightResponseSchema>) =>
  create(PreflightResponseSchema, init);
const secret = (init: MessageInitShape<typeof PreflightSecretSchema>) =>
  create(PreflightSecretSchema, init);

describe("preflightController", () => {
  describe("buildPreflightPipelineConfig", () => {
    it("builds config with manifest path", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/path/to/manifest.json",
      };
      const result = buildPreflightPipelineConfig(config);

      expect(result.bundleManifestPath).toBe("/path/to/manifest.json");
    });

    it("filters empty secrets", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/manifest.json",
        secrets: {
          API_KEY: "secret123",
          EMPTY_KEY: "",
          WHITESPACE_KEY: "   ",
          VALID_KEY: "value",
        },
      };
      const result = buildPreflightPipelineConfig(config);

      expect(result.preflightSecrets).toEqual({
        API_KEY: "secret123",
        VALID_KEY: "value",
      });
    });

    it("omits secrets if all empty", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/manifest.json",
        secrets: { EMPTY: "", WHITESPACE: "  " },
      };
      const result = buildPreflightPipelineConfig(config);

      expect(result.preflightSecrets).toBeUndefined();
    });

    it("omits secrets if undefined", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/manifest.json",
      };
      const result = buildPreflightPipelineConfig(config);

      expect(result.preflightSecrets).toBeUndefined();
    });

    it("includes additional config", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/manifest.json",
        additionalConfig: {
          stopAfterStage: StageName.PREFLIGHT,
        },
      };
      const result = buildPreflightPipelineConfig(config);

      expect(result.stopAfterStage).toBe(StageName.PREFLIGHT);
    });
  });

  describe("validatePreflightConfig", () => {
    it("returns error for missing scenario name", () => {
      const config: PreflightRunConfig = {
        scenarioName: "",
        bundleManifestPath: "/manifest.json",
      };
      const error = validatePreflightConfig(config);
      expect(error).toBe("No scenario selected");
    });

    it("returns error for missing manifest path", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "",
      };
      const error = validatePreflightConfig(config);
      expect(error).toBe("Bundle manifest path is required");
    });

    it("returns error for whitespace-only manifest path", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "   ",
      };
      const error = validatePreflightConfig(config);
      expect(error).toBe("Bundle manifest path is required");
    });

    it("returns null for valid config", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/manifest.json",
      };
      const error = validatePreflightConfig(config);
      expect(error).toBeNull();
    });
  });

  describe("buildJobStepMap", () => {
    it("returns empty map for null status", () => {
      const map = buildJobStepMap(null);
      expect(map.size).toBe(0);
    });

    it("returns empty map for status without stages", () => {
      const status = {} as VerbosePipelineStatus;
      const map = buildJobStepMap(status);
      expect(map.size).toBe(0);
    });

    it("maps bundle stage to validation", () => {
      const status = createPipelineStatus({
        stages: {
          bundle: { stage: StageName.BUNDLE, status: StageStatus.COMPLETED },
        },
      });
      const map = buildJobStepMap(status);

      expect(map.has("validation")).toBe(true);
      expect(map.get("validation")?.state).toBe("pass");
    });

    it("maps preflight stage to multiple steps", () => {
      const status = createPipelineStatus({
        stages: {
          preflight: {
            stage: StageName.PREFLIGHT,
            status: StageStatus.RUNNING,
          },
        },
      });
      const map = buildJobStepMap(status);

      expect(map.has("secrets")).toBe(true);
      expect(map.has("runtime")).toBe(true);
      expect(map.has("services")).toBe(true);
      expect(map.has("diagnostics")).toBe(true);
      expect(map.get("secrets")?.state).toBe("running");
    });

    it("maps failed status correctly", () => {
      const status = createPipelineStatus({
        stages: {
          bundle: {
            stage: StageName.BUNDLE,
            status: StageStatus.FAILED,
            error: "Validation failed",
          },
        },
      });
      const map = buildJobStepMap(status);

      expect(map.get("validation")?.state).toBe("fail");
      expect(map.get("validation")?.detail).toBe("Validation failed");
    });

    it("maps cancelled status to fail", () => {
      const status = createPipelineStatus({
        stages: {
          preflight: {
            stage: StageName.PREFLIGHT,
            status: StageStatus.CANCELLED,
          },
        },
      });
      const map = buildJobStepMap(status);

      expect(map.get("secrets")?.state).toBe("fail");
    });

    it("maps skipped status correctly", () => {
      const status = createPipelineStatus({
        stages: {
          bundle: { stage: StageName.BUNDLE, status: StageStatus.SKIPPED },
        },
      });
      const map = buildJobStepMap(status);

      expect(map.get("validation")?.state).toBe("skipped");
    });
  });

  describe("resolveAllStepStatuses", () => {
    it("uses job step status when available", () => {
      const jobStepById = new Map<string, PreflightJobStep>();
      jobStepById.set("validation", {
        id: "validation",
        name: "validation",
        state: "pass",
      });

      const statuses = resolveAllStepStatuses(
        jobStepById,
        false,
        null,
        true,
        preflight({ validation: { valid: false } }),
        false,
        0,
      );

      expect(statuses.validation.state).toBe("pass");
    });

    it("falls back to computed status when no job step", () => {
      const jobStepById = new Map<string, PreflightJobStep>();

      const statuses = resolveAllStepStatuses(
        jobStepById,
        false,
        null,
        true,
        preflight({ validation: { valid: true } }),
        false,
        0,
      );

      expect(statuses.validation.state).toBe("pass");
    });

    it("shows testing status when pending", () => {
      const jobStepById = new Map<string, PreflightJobStep>();

      const statuses = resolveAllStepStatuses(
        jobStepById,
        true,
        null,
        false,
        null,
        false,
        0,
      );

      expect(statuses.validation.state).toBe("testing");
      expect(statuses.secrets.state).toBe("testing");
    });

    it("shows fail status on error", () => {
      const jobStepById = new Map<string, PreflightJobStep>();

      const statuses = resolveAllStepStatuses(
        jobStepById,
        false,
        "Something went wrong",
        true,
        null,
        false,
        0,
      );

      expect(statuses.validation.state).toBe("fail");
    });
  });

  describe("buildPreflightSectionState", () => {
    it("builds state with missing secrets", () => {
      const preflightResult = preflight({
        validation: { valid: true },
        ready: { ready: false },
        secrets: [
          secret({ id: "API_KEY", required: true, hasValue: false }),
          secret({ id: "OPTIONAL", required: false, hasValue: false }),
        ],
      });

      const state = buildPreflightSectionState(
        "/path/to/manifest.json",
        preflightResult,
        null,
        null,
        false,
      );

      expect(state.missingSecrets).toHaveLength(1);
      expect(state.missingSecrets?.[0]?.id).toBe("API_KEY");
    });

    it("extracts bundle root from manifest path", () => {
      const state = buildPreflightSectionState(
        "/app/bundles/my-app/manifest.json",
        null,
        null,
        null,
        false,
      );

      expect(state.bundleRootPreview).toContain("my-app");
    });

    it("includes export payload", () => {
      const preflightResult = preflight({
        validation: { valid: true },
      });

      const state = buildPreflightSectionState(
        "/manifest.json",
        preflightResult,
        null,
        null,
        false,
      );

      expect(state.exportPayload.bundleManifestPath).toBe("/manifest.json");
      expect(state.exportPayload.result).toBe(preflightResult);
    });
  });

  describe("exportPreflightAsJson", () => {
    it("exports valid JSON string", () => {
      const preflightResult = preflight({
        validation: { valid: true },
      });

      const json = exportPreflightAsJson(
        "/manifest.json",
        preflightResult,
        null,
        [],
      );

      const parsed = JSON.parse(json) as Record<string, unknown>;
      expect(parsed.bundleManifestPath).toBe("/manifest.json");
      const result = parsed.result as
        | Record<string, Record<string, unknown>>
        | undefined;
      expect(result?.validation?.valid).toBe(true);
    });

    it("includes error when present", () => {
      const json = exportPreflightAsJson(
        "/manifest.json",
        null,
        "Validation failed",
        [],
      );

      const parsed = JSON.parse(json) as Record<string, unknown>;
      expect(parsed.error).toBe("Validation failed");
    });

    it("includes missing secrets", () => {
      const json = exportPreflightAsJson("/manifest.json", null, null, [
        secret({ id: "API_KEY", required: true, hasValue: false }),
      ]);

      const parsed = JSON.parse(json) as Record<string, unknown>;
      expect(parsed.missingSecrets).toHaveLength(1);
    });
  });

  describe("createPreflightJsonBlob", () => {
    it("creates blob with correct type", () => {
      const blob = createPreflightJsonBlob('{"test": true}');
      expect(blob.type).toBe("application/json");
    });

    it("creates blob with content", () => {
      const json = '{"key": "value"}';
      const blob = createPreflightJsonBlob(json);
      expect(blob.size).toBe(json.length);
    });
  });

  describe("isPreflightReadyForGeneration", () => {
    it("returns true when override is enabled", () => {
      expect(isPreflightReadyForGeneration(null, true)).toBe(true);
    });

    it("returns true when preflight complete", () => {
      const result = preflight({
        validation: { valid: true },
        ready: { ready: true },
        secrets: [secret({ id: "KEY", required: true, hasValue: true })],
      });

      expect(isPreflightReadyForGeneration(result, false)).toBe(true);
    });

    it("returns false when preflight incomplete", () => {
      const result = preflight({
        validation: { valid: false },
      });

      expect(isPreflightReadyForGeneration(result, false)).toBe(false);
    });

    it("returns false for null result", () => {
      expect(isPreflightReadyForGeneration(null, false)).toBe(false);
    });
  });

  describe("getPreflightBlockingReason", () => {
    it("returns reason for null result", () => {
      const reason = getPreflightBlockingReason(null);
      expect(reason).toBe("Preflight validation has not been run");
    });

    it("returns reason for failed validation", () => {
      const result = preflight({
        validation: { valid: false },
      });

      const reason = getPreflightBlockingReason(result);
      expect(reason).toBe("Bundle validation failed");
    });

    it("returns reason for missing secrets", () => {
      const result = preflight({
        validation: { valid: true },
        secrets: [
          secret({ id: "KEY1", required: true, hasValue: false }),
          secret({ id: "KEY2", required: true, hasValue: false }),
        ],
      });

      const reason = getPreflightBlockingReason(result);
      expect(reason).toBe("Missing 2 required secret(s)");
    });

    it("returns reason for not ready services", () => {
      const result = preflight({
        validation: { valid: true },
        secrets: [],
        ready: { ready: false },
      });

      const reason = getPreflightBlockingReason(result);
      expect(reason).toBe("Services are not ready");
    });

    it("returns null when ready", () => {
      const result = preflight({
        validation: { valid: true },
        secrets: [],
        ready: { ready: true },
      });

      const reason = getPreflightBlockingReason(result);
      expect(reason).toBeNull();
    });
  });
});
