/**
 * Tests for pipelineController.
 */

import { describe, it, expect } from "vitest";
import {
  buildPipelineConfig,
  buildGenerateConfig,
  validateBeforeRun,
  canProceedToGeneration,
  getEffectivePreflightResult,
  getEffectivePreflightOk,
  shouldAutoStartPolling,
  isInTerminalState,
  filterNonEmptySecrets,
  mapValidationErrorsToFormErrors,
} from "./pipelineController";
import type { BundlePreflightResponse } from "../lib/api";

// ============================================================================
// buildPipelineConfig tests
// ============================================================================

describe("buildPipelineConfig", () => {
  it("builds basic config with required fields", () => {
    const config = buildPipelineConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "bundled",
      platforms: ["linux-x64"],
    });

    expect(config).toEqual({
      scenario_name: "my-scenario",
      template_type: "basic",
      deployment_mode: "bundled",
      platforms: ["linux-x64"],
    });
  });

  it("includes optional stop_after_stage", () => {
    const config = buildPipelineConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "bundled",
      platforms: ["linux-x64"],
      stopAfterStage: "generate",
    });

    expect(config.stop_after_stage).toBe("generate");
  });

  it("includes proxy_url when provided", () => {
    const config = buildPipelineConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "proxy",
      proxyUrl: "http://localhost:3000",
      platforms: ["linux-x64"],
    });

    expect(config.proxy_url).toBe("http://localhost:3000");
  });

  it("trims whitespace from proxy_url", () => {
    const config = buildPipelineConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "proxy",
      proxyUrl: "  http://localhost:3000  ",
      platforms: ["linux-x64"],
    });

    expect(config.proxy_url).toBe("http://localhost:3000");
  });

  it("excludes proxy_url when empty", () => {
    const config = buildPipelineConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "proxy",
      proxyUrl: "   ",
      platforms: ["linux-x64"],
    });

    expect(config.proxy_url).toBeUndefined();
  });

  it("includes bundle_manifest_path when provided", () => {
    const config = buildPipelineConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "bundled",
      bundleManifestPath: "/path/to/manifest.json",
      platforms: ["linux-x64"],
    });

    expect(config.bundle_manifest_path).toBe("/path/to/manifest.json");
  });

  it("filters empty secrets", () => {
    const config = buildPipelineConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "bundled",
      platforms: ["linux-x64"],
      preflightSecrets: {
        API_KEY: "secret123",
        EMPTY_KEY: "",
        WHITESPACE: "   ",
        VALID: "value",
      },
    });

    expect(config.preflight_secrets).toEqual({
      API_KEY: "secret123",
      VALID: "value",
    });
  });

  it("omits preflight_secrets when all are empty", () => {
    const config = buildPipelineConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "bundled",
      platforms: ["linux-x64"],
      preflightSecrets: {
        EMPTY: "",
        WHITESPACE: "   ",
      },
    });

    expect(config.preflight_secrets).toBeUndefined();
  });
});

describe("buildGenerateConfig", () => {
  it("sets stop_after_stage to generate", () => {
    const config = buildGenerateConfig({
      scenarioName: "my-scenario",
      templateType: "basic",
      deploymentMode: "bundled",
      proxyUrl: "",
      platforms: ["linux-x64"],
      bundleManifestPath: "",
    });

    expect(config.stop_after_stage).toBe("generate");
  });
});

// ============================================================================
// validateBeforeRun tests
// ============================================================================

describe("validateBeforeRun", () => {
  it("returns valid when all conditions are met", () => {
    const result = validateBeforeRun({
      scenarioName: "my-scenario",
      isSubmitting: false,
      isBundled: true,
      bundleManifestPath: "/path/to/manifest.json",
    });

    expect(result).toEqual({ valid: true, error: null });
  });

  it("returns error when no scenario selected", () => {
    const result = validateBeforeRun({
      scenarioName: null,
      isSubmitting: false,
      isBundled: true,
      bundleManifestPath: "/path/to/manifest.json",
    });

    expect(result.valid).toBe(false);
    expect(result.error).toBe("No scenario selected");
  });

  it("returns error when already submitting", () => {
    const result = validateBeforeRun({
      scenarioName: "my-scenario",
      isSubmitting: true,
      isBundled: true,
      bundleManifestPath: "/path/to/manifest.json",
    });

    expect(result.valid).toBe(false);
    expect(result.error).toBe("A pipeline request is already in progress");
  });

  it("returns error when bundled mode without manifest path", () => {
    const result = validateBeforeRun({
      scenarioName: "my-scenario",
      isSubmitting: false,
      isBundled: true,
      bundleManifestPath: "",
    });

    expect(result.valid).toBe(false);
    expect(result.error).toBe("Bundle manifest path is required for bundled mode");
  });

  it("allows empty manifest path in proxy mode", () => {
    const result = validateBeforeRun({
      scenarioName: "my-scenario",
      isSubmitting: false,
      isBundled: false,
      bundleManifestPath: "",
    });

    expect(result).toEqual({ valid: true, error: null });
  });
});

// ============================================================================
// canProceedToGeneration tests
// ============================================================================

describe("canProceedToGeneration", () => {
  const validPreflightResult: BundlePreflightResponse = {
    status: "completed",
    validation: { valid: true },
    ready: { ready: true },
  } as BundlePreflightResponse;

  it("allows proceed when override is enabled", () => {
    const result = canProceedToGeneration(null, true, 0);
    expect(result).toEqual({ canProceed: true, reason: null });
  });

  it("blocks when no preflight result", () => {
    const result = canProceedToGeneration(null, false, 0);
    expect(result.canProceed).toBe(false);
    expect(result.reason).toBe("Preflight validation has not been run");
  });

  it("blocks when validation failed", () => {
    const result = canProceedToGeneration(
      { ...validPreflightResult, validation: { valid: false } } as BundlePreflightResponse,
      false,
      0
    );
    expect(result.canProceed).toBe(false);
    expect(result.reason).toBe("Bundle validation failed");
  });

  it("blocks when missing secrets", () => {
    const result = canProceedToGeneration(validPreflightResult, false, 2);
    expect(result.canProceed).toBe(false);
    expect(result.reason).toBe("Missing 2 required secret(s)");
  });

  it("blocks when services not ready", () => {
    const result = canProceedToGeneration(
      { ...validPreflightResult, ready: { ready: false } } as BundlePreflightResponse,
      false,
      0
    );
    expect(result.canProceed).toBe(false);
    expect(result.reason).toBe("Services are not ready");
  });

  it("allows proceed when all checks pass", () => {
    const result = canProceedToGeneration(validPreflightResult, false, 0);
    expect(result).toEqual({ canProceed: true, reason: null });
  });
});

// ============================================================================
// getEffectivePreflightResult tests
// ============================================================================

describe("getEffectivePreflightResult", () => {
  const storeResult = { status: "store" } as unknown as BundlePreflightResponse;
  const serverResult = { status: "server" } as unknown as BundlePreflightResponse;

  it("prefers store result when available", () => {
    const result = getEffectivePreflightResult({
      storeResult,
      serverResult,
    });
    expect(result).toBe(storeResult);
  });

  it("falls back to server result when store is null", () => {
    const result = getEffectivePreflightResult({
      storeResult: null,
      serverResult,
    });
    expect(result).toBe(serverResult);
  });

  it("returns null when both are null", () => {
    const result = getEffectivePreflightResult({
      storeResult: null,
      serverResult: null,
    });
    expect(result).toBeNull();
  });
});

// ============================================================================
// getEffectivePreflightOk tests
// ============================================================================

describe("getEffectivePreflightOk", () => {
  it("returns false when no preflight result", () => {
    expect(getEffectivePreflightOk(null, 0)).toBe(false);
  });

  it("returns false when validation is invalid", () => {
    const result = { validation: { valid: false }, ready: { ready: true } } as BundlePreflightResponse;
    expect(getEffectivePreflightOk(result, 0)).toBe(false);
  });

  it("returns false when services not ready", () => {
    const result = { validation: { valid: true }, ready: { ready: false } } as BundlePreflightResponse;
    expect(getEffectivePreflightOk(result, 0)).toBe(false);
  });

  it("returns false when missing secrets", () => {
    const result = { validation: { valid: true }, ready: { ready: true } } as BundlePreflightResponse;
    expect(getEffectivePreflightOk(result, 2)).toBe(false);
  });

  it("returns true when all conditions pass", () => {
    const result = { validation: { valid: true }, ready: { ready: true } } as BundlePreflightResponse;
    expect(getEffectivePreflightOk(result, 0)).toBe(true);
  });
});

// ============================================================================
// shouldAutoStartPolling tests
// ============================================================================

describe("shouldAutoStartPolling", () => {
  it("returns false for null status", () => {
    expect(shouldAutoStartPolling(null)).toBe(false);
  });

  it("returns true for running status", () => {
    expect(shouldAutoStartPolling("running")).toBe(true);
  });

  it("returns true for starting status", () => {
    expect(shouldAutoStartPolling("starting")).toBe(true);
  });

  it("returns false for completed status", () => {
    expect(shouldAutoStartPolling("completed")).toBe(false);
  });

  it("returns false for idle status", () => {
    expect(shouldAutoStartPolling("idle")).toBe(false);
  });
});

// ============================================================================
// isInTerminalState tests
// ============================================================================

describe("isInTerminalState", () => {
  it("returns false for null status", () => {
    expect(isInTerminalState(null)).toBe(false);
  });

  it("returns true for completed", () => {
    expect(isInTerminalState("completed")).toBe(true);
  });

  it("returns true for failed", () => {
    expect(isInTerminalState("failed")).toBe(true);
  });

  it("returns true for cancelled", () => {
    expect(isInTerminalState("cancelled")).toBe(true);
  });

  it("returns false for running", () => {
    expect(isInTerminalState("running")).toBe(false);
  });

  it("returns false for idle", () => {
    expect(isInTerminalState("idle")).toBe(false);
  });
});

// ============================================================================
// filterNonEmptySecrets tests
// ============================================================================

describe("filterNonEmptySecrets", () => {
  it("returns empty object for undefined", () => {
    expect(filterNonEmptySecrets(undefined)).toEqual({});
  });

  it("filters out empty values", () => {
    const result = filterNonEmptySecrets({
      API_KEY: "secret",
      EMPTY: "",
      WHITESPACE: "   ",
      VALID: "value",
    });
    expect(result).toEqual({ API_KEY: "secret", VALID: "value" });
  });

  it("returns empty object when all values empty", () => {
    const result = filterNonEmptySecrets({
      EMPTY: "",
      WHITESPACE: "   ",
    });
    expect(result).toEqual({});
  });
});

// ============================================================================
// mapValidationErrorsToFormErrors tests
// ============================================================================

describe("mapValidationErrorsToFormErrors", () => {
  it("maps errors with field property", () => {
    const errors = [
      { id: "required_scenario", message: "Scenario required", field: "scenario" },
    ];
    const result = mapValidationErrorsToFormErrors(errors);
    expect(result).toEqual([
      { field: "scenario", message: "Scenario required", code: "required_scenario" },
    ]);
  });

  it("uses id as field when field not provided", () => {
    const errors = [{ id: "required_scenario", message: "Scenario required" }];
    const result = mapValidationErrorsToFormErrors(errors);
    expect(result).toEqual([
      { field: "required_scenario", message: "Scenario required", code: "required_scenario" },
    ]);
  });

  it("maps multiple errors", () => {
    const errors = [
      { id: "error1", message: "Error 1", field: "field1" },
      { id: "error2", message: "Error 2" },
    ];
    const result = mapValidationErrorsToFormErrors(errors);
    expect(result).toHaveLength(2);
  });
});
