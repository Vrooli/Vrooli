import { describe, it, expect } from "vitest";
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
import type { VerbosePipelineStatus, BundlePreflightResponse } from "../lib/api";

describe("preflightController", () => {
  describe("buildPreflightPipelineConfig", () => {
    it("builds config with manifest path", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/path/to/manifest.json",
      };
      const result = buildPreflightPipelineConfig(config);

      expect(result.bundle_manifest_path).toBe("/path/to/manifest.json");
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

      expect(result.preflight_secrets).toEqual({
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

      expect(result.preflight_secrets).toBeUndefined();
    });

    it("omits secrets if undefined", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/manifest.json",
      };
      const result = buildPreflightPipelineConfig(config);

      expect(result.preflight_secrets).toBeUndefined();
    });

    it("includes additional config", () => {
      const config: PreflightRunConfig = {
        scenarioName: "test-scenario",
        bundleManifestPath: "/manifest.json",
        additionalConfig: {
          stop_after_stage: "preflight",
        },
      };
      const result = buildPreflightPipelineConfig(config);

      expect(result.stop_after_stage).toBe("preflight");
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
      const status = {
        stages: {
          bundle: { status: "completed" },
        },
      } as unknown as VerbosePipelineStatus;
      const map = buildJobStepMap(status);

      expect(map.has("validation")).toBe(true);
      expect(map.get("validation")?.state).toBe("pass");
    });

    it("maps preflight stage to multiple steps", () => {
      const status = {
        stages: {
          preflight: { status: "running" },
        },
      } as unknown as VerbosePipelineStatus;
      const map = buildJobStepMap(status);

      expect(map.has("secrets")).toBe(true);
      expect(map.has("runtime")).toBe(true);
      expect(map.has("services")).toBe(true);
      expect(map.has("diagnostics")).toBe(true);
      expect(map.get("secrets")?.state).toBe("running");
    });

    it("maps failed status correctly", () => {
      const status = {
        stages: {
          bundle: { status: "failed", error: "Validation failed" },
        },
      } as unknown as VerbosePipelineStatus;
      const map = buildJobStepMap(status);

      expect(map.get("validation")?.state).toBe("fail");
      expect(map.get("validation")?.detail).toBe("Validation failed");
    });

    it("maps cancelled status to fail", () => {
      const status = {
        stages: {
          preflight: { status: "cancelled" },
        },
      } as unknown as VerbosePipelineStatus;
      const map = buildJobStepMap(status);

      expect(map.get("secrets")?.state).toBe("fail");
    });

    it("maps skipped status correctly", () => {
      const status = {
        stages: {
          bundle: { status: "skipped" },
        },
      } as unknown as VerbosePipelineStatus;
      const map = buildJobStepMap(status);

      expect(map.get("validation")?.state).toBe("skipped");
    });
  });

  describe("resolveAllStepStatuses", () => {
    it("uses job step status when available", () => {
      const jobStepById = new Map();
      jobStepById.set("validation", { id: "validation", state: "pass" });

      const statuses = resolveAllStepStatuses(
        jobStepById,
        false,
        null,
        true,
        { validation: { valid: false } } as BundlePreflightResponse,
        false,
        0
      );

      expect(statuses.validation.state).toBe("pass");
    });

    it("falls back to computed status when no job step", () => {
      const jobStepById = new Map();

      const statuses = resolveAllStepStatuses(
        jobStepById,
        false,
        null,
        true,
        { validation: { valid: true } } as BundlePreflightResponse,
        false,
        0
      );

      expect(statuses.validation.state).toBe("pass");
    });

    it("shows testing status when pending", () => {
      const jobStepById = new Map();

      const statuses = resolveAllStepStatuses(
        jobStepById,
        true,
        null,
        false,
        null,
        false,
        0
      );

      expect(statuses.validation.state).toBe("testing");
      expect(statuses.secrets.state).toBe("testing");
    });

    it("shows fail status on error", () => {
      const jobStepById = new Map();

      const statuses = resolveAllStepStatuses(
        jobStepById,
        false,
        "Something went wrong",
        true,
        null,
        false,
        0
      );

      expect(statuses.validation.state).toBe("fail");
    });
  });

  describe("buildPreflightSectionState", () => {
    it("builds state with missing secrets", () => {
      const preflightResult: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: true },
        ready: { ready: false, details: {} },
        secrets: [
          { id: "API_KEY", name: "API Key", required: true, has_value: false },
          { id: "OPTIONAL", name: "Optional", required: false, has_value: false },
        ],
      } as BundlePreflightResponse;

      const state = buildPreflightSectionState(
        "/path/to/manifest.json",
        preflightResult,
        null,
        null,
        false
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
        false
      );

      expect(state.bundleRootPreview).toContain("my-app");
    });

    it("includes export payload", () => {
      const preflightResult: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: true },
      } as BundlePreflightResponse;

      const state = buildPreflightSectionState(
        "/manifest.json",
        preflightResult,
        null,
        null,
        false
      );

      expect(state.exportPayload.bundle_manifest_path).toBe("/manifest.json");
      expect(state.exportPayload.result).toBe(preflightResult);
    });
  });

  describe("exportPreflightAsJson", () => {
    it("exports valid JSON string", () => {
      const preflightResult: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: true },
      } as BundlePreflightResponse;

      const json = exportPreflightAsJson(
        "/manifest.json",
        preflightResult,
        null,
        []
      );

      const parsed = JSON.parse(json);
      expect(parsed.bundle_manifest_path).toBe("/manifest.json");
      expect(parsed.result.validation.valid).toBe(true);
    });

    it("includes error when present", () => {
      const json = exportPreflightAsJson(
        "/manifest.json",
        null,
        "Validation failed",
        []
      );

      const parsed = JSON.parse(json);
      expect(parsed.error).toBe("Validation failed");
    });

    it("includes missing secrets", () => {
      const json = exportPreflightAsJson(
        "/manifest.json",
        null,
        null,
        [{ id: "API_KEY", name: "API Key", required: true, has_value: false }]
      );

      const parsed = JSON.parse(json);
      expect(parsed.missing_secrets).toHaveLength(1);
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
      const result: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: true },
        ready: { ready: true, details: {} },
        secrets: [{ id: "KEY", name: "Key", required: true, has_value: true }],
      } as BundlePreflightResponse;

      expect(isPreflightReadyForGeneration(result, false)).toBe(true);
    });

    it("returns false when preflight incomplete", () => {
      const result: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: false },
      } as BundlePreflightResponse;

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
      const result: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: false },
      } as BundlePreflightResponse;

      const reason = getPreflightBlockingReason(result);
      expect(reason).toBe("Bundle validation failed");
    });

    it("returns reason for missing secrets", () => {
      const result: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: true },
        secrets: [
          { id: "KEY1", name: "Key 1", required: true, has_value: false },
          { id: "KEY2", name: "Key 2", required: true, has_value: false },
        ],
      } as BundlePreflightResponse;

      const reason = getPreflightBlockingReason(result);
      expect(reason).toBe("Missing 2 required secret(s)");
    });

    it("returns reason for not ready services", () => {
      const result: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: true },
        secrets: [],
        ready: { ready: false, details: {} },
      } as BundlePreflightResponse;

      const reason = getPreflightBlockingReason(result);
      expect(reason).toBe("Services are not ready");
    });

    it("returns null when ready", () => {
      const result: BundlePreflightResponse = {
        status: "completed",
        validation: { valid: true },
        secrets: [],
        ready: { ready: true, details: {} },
      } as BundlePreflightResponse;

      const reason = getPreflightBlockingReason(result);
      expect(reason).toBeNull();
    });
  });
});
