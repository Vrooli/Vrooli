import { describe, it, expect } from "vitest";
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
} from "./preflight.service";
import type { BundlePreflightResponse, BundlePreflightSecret } from "../lib/api";

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
      const result = { validation: { valid: true } } as BundlePreflightResponse;
      const secrets: BundlePreflightSecret[] = [{ id: "API_KEY", name: "API Key", required: true, has_value: false }];
      const payload = buildPreflightPayload("/path/to/manifest", result, "error", secrets);

      expect(payload.bundle_manifest_path).toBe("/path/to/manifest");
      expect(payload.start_services).toBe(true);
      expect(payload.result).toBe(result);
      expect(payload.error).toBe("error");
      expect(payload.missing_secrets).toBe(secrets);
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
      expect(checkDiagnosticsAvailable("8080: http", undefined, undefined)).toBe(true);
    });

    it("returns true when telemetry path exists", () => {
      expect(checkDiagnosticsAvailable(null, "/path/to/telemetry", undefined)).toBe(true);
    });

    it("returns true when log tails exist", () => {
      expect(checkDiagnosticsAvailable(null, undefined, [{ service: "api", lines: [] }])).toBe(true);
    });

    it("returns false when nothing available", () => {
      expect(checkDiagnosticsAvailable(null, undefined, undefined)).toBe(false);
      expect(checkDiagnosticsAvailable(null, undefined, [])).toBe(false);
    });
  });

  describe("getMissingSecrets", () => {
    it("returns empty array for undefined", () => {
      expect(getMissingSecrets(undefined)).toEqual([]);
    });

    it("filters to required secrets without values", () => {
      const secrets: BundlePreflightSecret[] = [
        { id: "1", name: "Secret 1", required: true, has_value: false },
        { id: "2", name: "Secret 2", required: false, has_value: false },
        { id: "3", name: "Secret 3", required: true, has_value: true },
      ];
      const missing = getMissingSecrets(secrets);
      expect(missing).toHaveLength(1);
      expect(missing[0].id).toBe("1");
    });
  });

  describe("areSecretsComplete", () => {
    it("returns true for undefined", () => {
      expect(areSecretsComplete(undefined)).toBe(true);
    });

    it("returns true when all required secrets have values", () => {
      const secrets: BundlePreflightSecret[] = [
        { id: "1", name: "Secret 1", required: true, has_value: true },
        { id: "2", name: "Secret 2", required: false, has_value: false },
      ];
      expect(areSecretsComplete(secrets)).toBe(true);
    });

    it("returns false when missing required secrets", () => {
      const secrets: BundlePreflightSecret[] = [
        { id: "1", name: "Secret 1", required: true, has_value: false },
      ];
      expect(areSecretsComplete(secrets)).toBe(false);
    });
  });

  describe("isValidationOk", () => {
    it("returns false for undefined", () => {
      expect(isValidationOk(undefined)).toBe(false);
    });

    it("returns validation.valid value", () => {
      expect(isValidationOk({ valid: true })).toBe(true);
      expect(isValidationOk({ valid: false })).toBe(false);
    });
  });

  describe("isReadinessOk", () => {
    it("returns false for undefined", () => {
      expect(isReadinessOk(undefined)).toBe(false);
    });

    it("returns readiness.ready value", () => {
      expect(isReadinessOk({ ready: true })).toBe(true);
      expect(isReadinessOk({ ready: false })).toBe(false);
    });
  });

  describe("isPreflightComplete", () => {
    it("returns false for null", () => {
      expect(isPreflightComplete(null)).toBe(false);
    });

    it("returns true when all checks pass", () => {
      const result: BundlePreflightResponse = {
        validation: { valid: true },
        ready: { ready: true },
        secrets: [{ id: "1", name: "Secret", required: true, has_value: true }],
      } as BundlePreflightResponse;
      expect(isPreflightComplete(result)).toBe(true);
    });

    it("returns false when validation fails", () => {
      const result: BundlePreflightResponse = {
        validation: { valid: false },
        ready: { ready: true },
        secrets: [],
      } as BundlePreflightResponse;
      expect(isPreflightComplete(result)).toBe(false);
    });

    it("returns false when readiness fails", () => {
      const result: BundlePreflightResponse = {
        validation: { valid: true },
        ready: { ready: false },
        secrets: [],
      } as BundlePreflightResponse;
      expect(isPreflightComplete(result)).toBe(false);
    });

    it("returns false when missing secrets", () => {
      const result: BundlePreflightResponse = {
        validation: { valid: true },
        ready: { ready: true },
        secrets: [{ id: "1", name: "Secret", required: true, has_value: false }],
      } as BundlePreflightResponse;
      expect(isPreflightComplete(result)).toBe(false);
    });
  });
});
