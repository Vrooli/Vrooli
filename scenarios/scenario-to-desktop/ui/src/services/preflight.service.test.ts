import { describe, it, expect } from "vitest";
import { create, type MessageInitShape } from "@bufbuild/protobuf";
import {
  getValidationStatus,
  getSecretsStatus,
  getRuntimeStatus,
  getServicesStatus,
  getDiagnosticsStatus,
  buildPreflightPayload,
  filterValidSecrets,
  checkDiagnosticsAvailable,
  getMissingSecrets,
  areSecretsComplete,
  isValidationOk,
  isReadinessOk,
  isPreflightComplete,
  resolveJobStepStatus,
  buildPreflightDisplayState,
} from "./preflight.service";
import type { PreflightJobStep } from "../lib/preflight-status";
import {
  BundleValidationResultSchema,
  PreflightReadySchema,
  PreflightResponseSchema,
  PreflightSecretSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";

const preflight = (init: MessageInitShape<typeof PreflightResponseSchema>) =>
  create(PreflightResponseSchema, init);
const secret = (init: MessageInitShape<typeof PreflightSecretSchema>) =>
  create(PreflightSecretSchema, init);

describe("preflight.service", () => {
  describe("getValidationStatus", () => {
    it("returns testing when pending", () => {
      const status = getValidationStatus(true, null, false);
      expect(status.state).toBe("testing");
      expect(status.label).toBe("Testing");
    });

    it("returns fail when error", () => {
      const status = getValidationStatus(false, "some error", false);
      expect(status.state).toBe("fail");
      expect(status.label).toBe("Failed");
    });

    it("returns pending when not run", () => {
      const status = getValidationStatus(false, null, false);
      expect(status.state).toBe("pending");
      expect(status.label).toBe("Pending");
    });

    it("returns pass when validation valid", () => {
      const status = getValidationStatus(false, null, true, true);
      expect(status.state).toBe("pass");
      expect(status.label).toBe("Pass");
    });

    it("returns fail when validation invalid", () => {
      const status = getValidationStatus(false, null, true, false);
      expect(status.state).toBe("fail");
      expect(status.label).toBe("Fail");
    });

    it("returns warning when validation undefined", () => {
      const status = getValidationStatus(false, null, true, undefined);
      expect(status.state).toBe("warning");
      expect(status.label).toBe("Review");
    });
  });

  describe("getSecretsStatus", () => {
    it("returns testing when pending", () => {
      const status = getSecretsStatus(true, null, false, 0);
      expect(status.state).toBe("testing");
    });

    it("returns fail when error", () => {
      const status = getSecretsStatus(false, "error", false, 0);
      expect(status.state).toBe("fail");
    });

    it("returns pending when not run", () => {
      const status = getSecretsStatus(false, null, false, 0);
      expect(status.state).toBe("pending");
    });

    it("returns warning when missing secrets", () => {
      const status = getSecretsStatus(false, null, true, 2);
      expect(status.state).toBe("warning");
      expect(status.label).toBe("Missing");
    });

    it("returns pass when no missing secrets", () => {
      const status = getSecretsStatus(false, null, true, 0);
      expect(status.state).toBe("pass");
      expect(status.label).toBe("Ready");
    });
  });

  describe("getRuntimeStatus", () => {
    it("returns testing when pending", () => {
      const status = getRuntimeStatus(true, null, false, false);
      expect(status.state).toBe("testing");
    });

    it("returns fail when error", () => {
      const status = getRuntimeStatus(false, "error", false, false);
      expect(status.state).toBe("fail");
    });

    it("returns pass when has result", () => {
      const status = getRuntimeStatus(false, null, true, true);
      expect(status.state).toBe("pass");
      expect(status.label).toBe("Running");
    });

    it("returns pending when not run", () => {
      const status = getRuntimeStatus(false, null, false, false);
      expect(status.state).toBe("pending");
    });
  });

  describe("getServicesStatus", () => {
    it("returns testing when pending", () => {
      const status = getServicesStatus(true, null, false);
      expect(status.state).toBe("testing");
    });

    it("returns pass when ready", () => {
      const status = getServicesStatus(false, null, true, true);
      expect(status.state).toBe("pass");
      expect(status.label).toBe("Ready");
    });

    it("returns warning when not ready", () => {
      const status = getServicesStatus(false, null, true, false);
      expect(status.state).toBe("warning");
    });
  });

  describe("getDiagnosticsStatus", () => {
    it("returns testing when pending", () => {
      const status = getDiagnosticsStatus(true, null, false, false);
      expect(status.state).toBe("testing");
    });

    it("returns pass when diagnostics available", () => {
      const status = getDiagnosticsStatus(false, null, true, true);
      expect(status.state).toBe("pass");
      expect(status.label).toBe("Available");
    });

    it("returns warning when diagnostics empty", () => {
      const status = getDiagnosticsStatus(false, null, true, false);
      expect(status.state).toBe("warning");
      expect(status.label).toBe("Empty");
    });
  });

  describe("buildPreflightPayload", () => {
    it("builds payload with all fields", () => {
      const result = preflight({ validation: { valid: true } });
      const secrets = [
        secret({ id: "API_KEY", required: true, hasValue: false }),
      ];
      const payload = buildPreflightPayload(
        "/path/to/manifest",
        result,
        "error",
        secrets,
      );

      expect(payload.bundleManifestPath).toBe("/path/to/manifest");
      expect(payload.startServices).toBe(true);
      expect(payload.result).toBe(result);
      expect(payload.error).toBe("error");
      expect(payload.missingSecrets).toBe(secrets);
    });

    it("handles null error", () => {
      const payload = buildPreflightPayload("/path", null, null, []);
      expect(payload.error).toBeUndefined();
    });
  });

  describe("filterValidSecrets", () => {
    it("filters out empty values", () => {
      const secrets = {
        key1: "value1",
        key2: "",
        key3: "  ",
        key4: "value4",
      };
      const filtered = filterValidSecrets(secrets);

      expect(filtered).toEqual({
        key1: "value1",
        key4: "value4",
      });
    });

    it("returns empty object for all empty values", () => {
      const secrets = { key1: "", key2: "   " };
      const filtered = filterValidSecrets(secrets);
      expect(filtered).toEqual({});
    });
  });

  describe("checkDiagnosticsAvailable", () => {
    it("returns true when port summary exists", () => {
      expect(checkDiagnosticsAvailable(1, undefined, undefined)).toBe(true);
    });

    it("returns true when telemetry path exists", () => {
      expect(
        checkDiagnosticsAvailable(0, "/path/to/telemetry", undefined),
      ).toBe(true);
    });

    it("returns true when log tails exist", () => {
      expect(
        checkDiagnosticsAvailable(0, undefined, [
          { service: "api", lines: [] },
        ]),
      ).toBe(true);
    });

    it("returns false when nothing available", () => {
      expect(checkDiagnosticsAvailable(0, undefined, undefined)).toBe(false);
      expect(checkDiagnosticsAvailable(0, undefined, [])).toBe(false);
    });
  });

  describe("getMissingSecrets", () => {
    it("returns empty array for undefined", () => {
      expect(getMissingSecrets(undefined)).toEqual([]);
    });

    it("filters to required secrets without values", () => {
      const secrets = [
        secret({ id: "1", required: true, hasValue: false }),
        secret({ id: "2", required: false, hasValue: false }),
        secret({ id: "3", required: true, hasValue: true }),
      ];
      const missing = getMissingSecrets(secrets);
      expect(missing).toHaveLength(1);
      expect(missing?.[0]?.id).toBe("1");
    });
  });

  describe("areSecretsComplete", () => {
    it("returns true for undefined", () => {
      expect(areSecretsComplete(undefined)).toBe(true);
    });

    it("returns true when all required secrets have values", () => {
      const secrets = [
        secret({ id: "1", required: true, hasValue: true }),
        secret({ id: "2", required: false, hasValue: false }),
      ];
      expect(areSecretsComplete(secrets)).toBe(true);
    });

    it("returns false when missing required secrets", () => {
      const secrets = [secret({ id: "1", required: true, hasValue: false })];
      expect(areSecretsComplete(secrets)).toBe(false);
    });
  });

  describe("isValidationOk", () => {
    it("returns false for undefined", () => {
      expect(isValidationOk(undefined)).toBe(false);
    });

    it("returns validation.valid value", () => {
      expect(
        isValidationOk(create(BundleValidationResultSchema, { valid: true })),
      ).toBe(true);
      expect(
        isValidationOk(create(BundleValidationResultSchema, { valid: false })),
      ).toBe(false);
    });
  });

  describe("isReadinessOk", () => {
    it("returns false for undefined", () => {
      expect(isReadinessOk(undefined)).toBe(false);
    });

    it("returns readiness.ready value", () => {
      expect(isReadinessOk(create(PreflightReadySchema, { ready: true }))).toBe(
        true,
      );
      expect(
        isReadinessOk(create(PreflightReadySchema, { ready: false })),
      ).toBe(false);
    });
  });

  describe("isPreflightComplete", () => {
    it("returns false for null", () => {
      expect(isPreflightComplete(null)).toBe(false);
    });

    it("returns true when all checks pass", () => {
      const result = preflight({
        validation: { valid: true },
        ready: { ready: true },
        secrets: [secret({ id: "1", required: true, hasValue: true })],
      });
      expect(isPreflightComplete(result)).toBe(true);
    });

    it("returns false when validation fails", () => {
      const result = preflight({
        validation: { valid: false },
        ready: { ready: true },
        secrets: [],
      });
      expect(isPreflightComplete(result)).toBe(false);
    });

    it("returns false when readiness fails", () => {
      const result = preflight({
        validation: { valid: true },
        ready: { ready: false },
        secrets: [],
      });
      expect(isPreflightComplete(result)).toBe(false);
    });

    it("returns false when missing secrets", () => {
      const result = preflight({
        validation: { valid: true },
        ready: { ready: true },
        secrets: [secret({ id: "1", required: true, hasValue: false })],
      });
      expect(isPreflightComplete(result)).toBe(false);
    });
  });

  describe("resolveJobStepStatus", () => {
    it("returns null for missing step", () => {
      const jobMap = new Map<string, PreflightJobStep>();
      expect(resolveJobStepStatus(jobMap, "nonexistent")).toBe(null);
    });

    it("resolves pass state", () => {
      const jobMap = new Map<string, PreflightJobStep>();
      jobMap.set("validation", {
        id: "validation",
        name: "Validation",
        state: "pass",
      });
      const status = resolveJobStepStatus(jobMap, "validation");
      expect(status).toEqual({ state: "pass", label: "Pass" });
    });

    it("converts running to testing", () => {
      const jobMap = new Map<string, PreflightJobStep>();
      jobMap.set("secrets", {
        id: "secrets",
        name: "Secrets",
        state: "running",
      });
      const status = resolveJobStepStatus(jobMap, "secrets");
      expect(status?.state).toBe("testing");
      expect(status?.label).toBe("Testing");
    });

    it("resolves fail state", () => {
      const jobMap = new Map<string, PreflightJobStep>();
      jobMap.set("runtime", { id: "runtime", name: "Runtime", state: "fail" });
      const status = resolveJobStepStatus(jobMap, "runtime");
      expect(status).toEqual({ state: "fail", label: "Fail" });
    });

    it("resolves warning state", () => {
      const jobMap = new Map<string, PreflightJobStep>();
      jobMap.set("diag", { id: "diag", name: "Diagnostics", state: "warning" });
      const status = resolveJobStepStatus(jobMap, "diag");
      expect(status).toEqual({ state: "warning", label: "Review" });
    });
  });

  describe("buildPreflightDisplayState", () => {
    it("returns initial state when nothing has run", () => {
      const state = buildPreflightDisplayState(null, null, null, false, 0);
      expect(state.hasRun).toBe(false);
      expect(state.isRunning).toBe(false);
      expect(state.isComplete).toBe(false);
      expect(state.hasError).toBe(false);
    });

    it("marks hasRun true when result exists", () => {
      const result = preflight({
        validation: { valid: true },
        ready: { ready: true },
        secrets: [],
      });
      const state = buildPreflightDisplayState(null, result, null, false, 0);
      expect(state.hasRun).toBe(true);
      expect(state.isComplete).toBe(true);
    });

    it("detects error state", () => {
      const state = buildPreflightDisplayState(
        null,
        null,
        "Connection failed",
        false,
        0,
      );
      expect(state.hasRun).toBe(true);
      expect(state.hasError).toBe(true);
    });

    it("computes validation status correctly", () => {
      const result = preflight({
        validation: { valid: true },
        ready: { ready: false },
        secrets: [],
      });
      const state = buildPreflightDisplayState(null, result, null, false, 0);
      expect(state.validationOk).toBe(true);
      expect(state.readinessOk).toBe(false);
      expect(state.overallOk).toBe(false);
    });

    it("computes secrets status correctly", () => {
      const result = preflight({
        validation: { valid: true },
        ready: { ready: true },
        secrets: [],
      });
      const state = buildPreflightDisplayState(null, result, null, false, 2);
      expect(state.secretsOk).toBe(false);
      expect(state.overallOk).toBe(false);
    });

    it("returns overallOk true when all checks pass", () => {
      const result = preflight({
        validation: { valid: true },
        ready: { ready: true },
        secrets: [],
      });
      const state = buildPreflightDisplayState(null, result, null, false, 0);
      expect(state.validationOk).toBe(true);
      expect(state.secretsOk).toBe(true);
      expect(state.readinessOk).toBe(true);
      expect(state.overallOk).toBe(true);
    });

    it("marks isRunning when in progress", () => {
      const state = buildPreflightDisplayState(null, null, null, true, 0);
      expect(state.isRunning).toBe(true);
      expect(state.isComplete).toBe(false);
    });
  });
});
