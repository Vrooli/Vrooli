/**
 * Tests for usePreflightSection hook.
 * Tests step status computation, view mode management, and copy/download actions.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import { renderHook, act } from "@testing-library/react";
import { create } from "@bufbuild/protobuf";
import {
  CheckStatus,
  PreflightCheckStep,
  PreflightResponseSchema,
} from "@vrooli/proto-types/scenario-to-desktop/v1/shared/preflight_results_pb";
import {
  usePreflightSection,
  type UsePreflightSectionProps,
} from "./usePreflightSection";
import { usePipelineStore } from "../store";

// Mock browser utilities
vi.mock("../lib/browser", () => ({
  writeToClipboard: vi.fn().mockResolvedValue({ success: true }),
  triggerBlobDownload: vi.fn(),
}));

// Import mocked functions for assertions
import { writeToClipboard, triggerBlobDownload } from "../lib/browser";

// Reset store before each test
beforeEach(() => {
  usePipelineStore.getState().reset();
  vi.clearAllMocks();
});

const defaultProps: UsePreflightSectionProps = {
  scenarioName: "test-scenario",
  bundleManifestPath: "/path/to/bundle.json",
  isBundled: true,
};

describe("usePreflightSection", () => {
  describe("view mode management", () => {
    it("starts with summary view mode", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.viewMode).toBe("summary");
    });

    it("allows switching to json view mode", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      act(() => {
        result.current.setViewMode("json");
      });

      expect(result.current.viewMode).toBe("json");
    });

    it("allows switching back to summary view mode", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      act(() => {
        result.current.setViewMode("json");
        result.current.setViewMode("summary");
      });

      expect(result.current.viewMode).toBe("summary");
    });
  });

  describe("copy status", () => {
    it("starts with idle copy status", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.copyStatus).toBe("idle");
    });
  });

  describe("pipeline store state", () => {
    it("reads preflightResult from store", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.preflightResult).toBeNull();
    });

    it("reads preflightSecrets from store", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.preflightSecrets).toEqual({});
    });

    it("reads preflightOverride from store", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.preflightOverride).toBe(false);
    });

    it("reads preflightError from store", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.preflightError).toBeNull();
    });

    it("reads preflightPending from store (via selector)", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.preflightPending).toBe(false);
    });

    it("reads preflightOk from store (via selector)", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.preflightOk).toBe(false);
    });

    it("reads missingSecrets from store (via selector)", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.missingSecrets).toEqual([]);
    });
  });

  describe("step statuses", () => {
    it("provides step statuses object", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.stepStatuses).toBeDefined();
      expect(result.current.stepStatuses.validation).toBeDefined();
      expect(result.current.stepStatuses.secrets).toBeDefined();
      expect(result.current.stepStatuses.runtime).toBeDefined();
      expect(result.current.stepStatuses.services).toBeDefined();
      expect(result.current.stepStatuses.diagnostics).toBeDefined();
    });

    it("all steps start as pending when no preflight has run", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      // PreflightStepStatus has 'state' property, not 'status'
      expect(result.current.stepStatuses.validation.state).toBe("pending");
      expect(result.current.stepStatuses.secrets.state).toBe("pending");
      expect(result.current.stepStatuses.runtime.state).toBe("pending");
      expect(result.current.stepStatuses.services.state).toBe("pending");
      expect(result.current.stepStatuses.diagnostics.state).toBe("pending");
    });
  });

  describe("computed values", () => {
    it("computes bundleRootPreview from bundleManifestPath", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      // bundleRootPreview should be derived from the manifest path
      expect(typeof result.current.bundleRootPreview).toBe("string");
    });

    it("computes hasRun based on preflight state", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      // hasRun should be false initially
      expect(result.current.hasRun).toBe(false);
    });

    it("computes diagnosticsAvailable based on preflight result", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      // diagnosticsAvailable should be false when no preflight result
      expect(result.current.diagnosticsAvailable).toBe(false);
    });

    it("provides empty arrays for validation/readiness data when no result", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.validation).toBeUndefined();
      expect(result.current.readiness).toBeUndefined();
      expect(result.current.fingerprints).toEqual([]);
      expect(result.current.checks).toEqual([]);
      expect(result.current.preflightErrors).toEqual([]);
    });

    it("provides empty arrays for filtered checks", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.validationChecks).toEqual([]);
      expect(result.current.secretChecks).toEqual([]);
      expect(result.current.runtimeChecks).toEqual([]);
      expect(result.current.serviceChecks).toEqual([]);
      expect(result.current.diagnosticsChecks).toEqual([]);
    });
  });

  describe("preflight payload", () => {
    it("builds preflight payload for export", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      const payload = result.current.preflightPayload;

      expect(payload).toHaveProperty("bundle_manifest_path");
      expect(payload).toHaveProperty("start_services");
      expect(payload).toHaveProperty("result");
      expect(payload).toHaveProperty("missing_secrets");
    });

    it("includes bundle manifest path in payload", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      const payload = result.current.preflightPayload as {
        bundle_manifest_path: string;
      };

      expect(payload.bundle_manifest_path).toBe("/path/to/bundle.json");
    });
  });

  describe("actions", () => {
    it("provides setPreflightSecret action", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.setPreflightSecret).toBe("function");

      act(() => {
        result.current.setPreflightSecret("API_KEY", "test-value");
      });

      expect(usePipelineStore.getState().preflightSecrets.API_KEY).toBe(
        "test-value",
      );
    });

    it("provides setPreflightOverride action", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.setPreflightOverride).toBe("function");

      act(() => {
        result.current.setPreflightOverride(true);
      });

      expect(usePipelineStore.getState().preflightOverride).toBe(true);
    });

    it("provides cancelPipeline action", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.cancelPipeline).toBe("function");
    });

    it("provides runPreflight action", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.runPreflight).toBe("function");
    });
  });

  describe("copyJson", () => {
    it("copies preflight JSON to clipboard when result exists", async () => {
      // Set up a preflight result in the store
      act(() => {
        usePipelineStore.setState({
          preflightResult: create(PreflightResponseSchema, {
            validation: { valid: true },
            ready: { ready: true },
          }),
        });
      });

      const { result } = renderHook(() => usePreflightSection(defaultProps));

      await act(async () => {
        await result.current.copyJson();
      });

      expect(writeToClipboard).toHaveBeenCalled();
    });

    it("sets copyStatus to copied on success", async () => {
      // Set up a preflight result
      act(() => {
        usePipelineStore.setState({
          preflightResult: create(PreflightResponseSchema, {
            validation: { valid: true },
          }),
        });
      });

      const { result } = renderHook(() => usePreflightSection(defaultProps));

      await act(async () => {
        await result.current.copyJson();
      });

      expect(result.current.copyStatus).toBe("copied");
    });

    it("sets copyStatus to error on failure", async () => {
      vi.mocked(writeToClipboard).mockResolvedValueOnce({
        success: false,
        error: "Failed",
      });

      // Set up a preflight result
      act(() => {
        usePipelineStore.setState({
          preflightResult: create(PreflightResponseSchema, {
            validation: { valid: true },
          }),
        });
      });

      const { result } = renderHook(() => usePreflightSection(defaultProps));

      await act(async () => {
        await result.current.copyJson();
      });

      expect(result.current.copyStatus).toBe("error");
    });

    it("does nothing when no result or error exists", async () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      await act(async () => {
        await result.current.copyJson();
      });

      expect(writeToClipboard).not.toHaveBeenCalled();
    });
  });

  describe("downloadJson", () => {
    it("triggers blob download with preflight JSON", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      act(() => {
        result.current.downloadJson();
      });

      expect(triggerBlobDownload).toHaveBeenCalledWith(
        expect.any(Blob),
        "preflight.json",
      );
    });
  });

  describe("resolveJobStepDetail", () => {
    it("provides function to resolve job step details", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.resolveJobStepDetail).toBe("function");

      // When no job steps exist, should return undefined
      const detail = result.current.resolveJobStepDetail("validation");
      expect(detail).toBeUndefined();
    });
  });

  describe("snapshot timing", () => {
    it("provides snapshotTs (timestamp)", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.snapshotTs).toBe("number");
    });

    it("provides snapshotLabel", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.snapshotLabel).toBe("string");
    });

    it("provides snapshotAge", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.snapshotAge).toBe("string");
    });

    it("provides tick for re-renders", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.tick).toBe("number");
    });
  });

  describe("likelyRootMismatch detection", () => {
    it("provides likelyRootMismatch flag", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(typeof result.current.likelyRootMismatch).toBe("boolean");
    });

    it("returns false when no validation result", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.likelyRootMismatch).toBe(false);
    });
  });

  describe("readinessDetails", () => {
    it("provides readinessDetails as array of tuples", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(Array.isArray(result.current.readinessDetails)).toBe(true);
    });

    it("returns empty array when no readiness data", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.readinessDetails).toEqual([]);
    });
  });

  describe("port summary", () => {
    it("provides portSummary as empty string when no ports", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      // formatPortSummary returns "" when no ports data
      expect(result.current.portSummary).toBe("");
    });
  });

  describe("with preflight result in store", () => {
    beforeEach(() => {
      act(() => {
        usePipelineStore.setState({
          preflightResult: create(PreflightResponseSchema, {
            validation: {
              valid: true,
            },
            ready: {
              ready: true,
              details: [{ serviceId: "api", ready: true }],
            },
            ports: [{ serviceId: "api", name: "http", port: 3000 }],
            telemetry: { path: "/telemetry" },
            checks: [
              {
                step: PreflightCheckStep.VALIDATION,
                status: CheckStatus.PASSED,
              },
              { step: PreflightCheckStep.SECRETS, status: CheckStatus.PASSED },
            ],
            errors: [],
            secrets: [],
          }),
        });
      });
    });

    it("extracts validation from result", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.validation).toBeDefined();
      expect(result.current.validation?.valid).toBe(true);
    });

    it("extracts readiness from result", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.readiness).toBeDefined();
      expect(result.current.readiness?.ready).toBe(true);
    });

    it("extracts ports from result", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.ports).toBeDefined();
    });

    it("extracts checks from result", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.checks.length).toBe(2);
    });

    it("filters checks by step", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.validationChecks.length).toBe(1);
      expect(result.current.secretChecks.length).toBe(1);
    });

    it("hasRun is true when result exists", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.hasRun).toBe(true);
    });

    it("diagnosticsAvailable is true when ports or telemetry exist", () => {
      const { result } = renderHook(() => usePreflightSection(defaultProps));

      expect(result.current.diagnosticsAvailable).toBe(true);
    });
  });
});
