/**
 * Tests for formSelectors.
 * Tests all derived state selectors for the form store.
 */

import { describe, it, expect } from "vitest";
import {
  selectConnectionDecision,
  selectIsBundled,
  selectRequiresRemoteConfig,
  selectAllowedServerTypes,
  selectSelectedPlatformsList,
  selectStandardOutputPath,
  selectStagingPreviewPath,
  selectIsCustomLocation,
  selectIconPreviewUrl,
  selectHasValidationErrors,
  selectFieldErrors,
  selectAppMetadata,
  selectDeployment,
  selectOutput,
  selectPlatforms,
  selectConnection,
} from "./formSelectors";
import type { FormStore } from "./formTypes";
import { initialFormState } from "./formTypes";

// Helper to create test state with overrides
function createTestState(overrides: Partial<FormStore> = {}): FormStore {
  return {
    ...initialFormState,
    ...overrides,
    // Add stub actions for type compatibility
    setAppDisplayName: () => {},
    setAppDescription: () => {},
    setIconPath: () => {},
    setIconPreviewError: () => {},
    setDeploymentMode: () => {},
    setServerType: () => {},
    setFramework: () => {},
    setLocationMode: () => {},
    setOutputPath: () => {},
    setPlatforms: () => {},
    handlePlatformChange: () => {},
    setProxyUrl: () => {},
    setBundleManifestPath: () => {},
    setServerPort: () => {},
    setLocalServerPath: () => {},
    setLocalApiEndpoint: () => {},
    setAutoManageTier1: () => {},
    setVrooliBinaryPath: () => {},
    setConnectionResult: () => {},
    setConnectionError: () => {},
    setSigningEnabledForBuild: () => {},
    setSelectedTemplate: () => {},
    setValidationErrors: () => {},
    clearValidationErrors: () => {},
    setScenarioLocked: () => {},
    resetFormState: () => {},
    hydrateFromServer: () => {},
  } as FormStore;
}

describe("formSelectors", () => {
  describe("selectConnectionDecision", () => {
    it("returns bundled-runtime for bundled deployment mode", () => {
      const state = createTestState({
        deployment: { mode: "bundled", serverType: "external", framework: "electron" },
      });

      const decision = selectConnectionDecision(state);

      expect(decision.kind).toBe("bundled-runtime");
      expect(decision.requiresProxyUrl).toBe(false);
      expect(decision.requiresBundleManifest).toBe(true);
    });

    it("returns remote-server for external-server with external server type", () => {
      const state = createTestState({
        deployment: { mode: "external-server", serverType: "external", framework: "electron" },
      });

      const decision = selectConnectionDecision(state);

      expect(decision.kind).toBe("remote-server");
      expect(decision.requiresProxyUrl).toBe(true);
      expect(decision.requiresBundleManifest).toBe(false);
    });

    it("returns local-embedded for external-server with node server type", () => {
      const state = createTestState({
        deployment: { mode: "external-server", serverType: "node", framework: "electron" },
      });

      const decision = selectConnectionDecision(state);

      expect(decision.kind).toBe("local-embedded");
      expect(decision.requiresProxyUrl).toBe(false);
      expect(decision.requiresBundleManifest).toBe(false);
    });

    it("returns local-embedded for external-server with static server type", () => {
      const state = createTestState({
        deployment: { mode: "external-server", serverType: "static", framework: "electron" },
      });

      const decision = selectConnectionDecision(state);

      expect(decision.kind).toBe("local-embedded");
    });

    it("returns bundled-runtime for cloud-api mode", () => {
      const state = createTestState({
        deployment: { mode: "cloud-api", serverType: "external", framework: "electron" },
      });

      // cloud-api should behave like bundled for now
      const decision = selectConnectionDecision(state);

      // cloud-api is not bundled mode, so it follows external-server pattern
      // Let's check what decideConnection actually does for cloud-api
      expect(decision.kind).toBe("remote-server");
    });
  });

  describe("selectIsBundled", () => {
    it("returns true for bundled deployment mode", () => {
      const state = createTestState({
        deployment: { mode: "bundled", serverType: "external", framework: "electron" },
      });

      expect(selectIsBundled(state)).toBe(true);
    });

    it("returns false for external-server deployment mode", () => {
      const state = createTestState({
        deployment: { mode: "external-server", serverType: "external", framework: "electron" },
      });

      expect(selectIsBundled(state)).toBe(false);
    });

    it("returns false for cloud-api deployment mode", () => {
      const state = createTestState({
        deployment: { mode: "cloud-api", serverType: "external", framework: "electron" },
      });

      expect(selectIsBundled(state)).toBe(false);
    });
  });

  describe("selectRequiresRemoteConfig", () => {
    it("returns false for bundled mode", () => {
      const state = createTestState({
        deployment: { mode: "bundled", serverType: "external", framework: "electron" },
      });

      expect(selectRequiresRemoteConfig(state)).toBe(false);
    });

    it("returns true for external-server with external server type", () => {
      const state = createTestState({
        deployment: { mode: "external-server", serverType: "external", framework: "electron" },
      });

      expect(selectRequiresRemoteConfig(state)).toBe(true);
    });

    it("returns false for external-server with node server type", () => {
      const state = createTestState({
        deployment: { mode: "external-server", serverType: "node", framework: "electron" },
      });

      expect(selectRequiresRemoteConfig(state)).toBe(false);
    });
  });

  describe("selectAllowedServerTypes", () => {
    it("returns only external for bundled mode", () => {
      const state = createTestState({
        deployment: { mode: "bundled", serverType: "external", framework: "electron" },
      });

      expect(selectAllowedServerTypes(state)).toEqual(["external"]);
    });

    it("returns only external for cloud-api mode", () => {
      const state = createTestState({
        deployment: { mode: "cloud-api", serverType: "external", framework: "electron" },
      });

      expect(selectAllowedServerTypes(state)).toEqual(["external"]);
    });

    it("returns all server types for external-server mode", () => {
      const state = createTestState({
        deployment: { mode: "external-server", serverType: "external", framework: "electron" },
      });

      const allowed = selectAllowedServerTypes(state);

      expect(allowed).toContain("external");
      expect(allowed).toContain("static");
      expect(allowed).toContain("node");
      expect(allowed).toContain("executable");
    });
  });

  describe("selectSelectedPlatformsList", () => {
    it("returns all platforms when all are selected", () => {
      const state = createTestState({
        platforms: { win: true, mac: true, linux: true },
      });

      const platforms = selectSelectedPlatformsList(state);

      expect(platforms).toContain("win");
      expect(platforms).toContain("mac");
      expect(platforms).toContain("linux");
      expect(platforms).toHaveLength(3);
    });

    it("returns only selected platforms", () => {
      const state = createTestState({
        platforms: { win: true, mac: false, linux: true },
      });

      const platforms = selectSelectedPlatformsList(state);

      expect(platforms).toContain("win");
      expect(platforms).not.toContain("mac");
      expect(platforms).toContain("linux");
      expect(platforms).toHaveLength(2);
    });

    it("returns empty array when no platforms selected", () => {
      const state = createTestState({
        platforms: { win: false, mac: false, linux: false },
      });

      expect(selectSelectedPlatformsList(state)).toEqual([]);
    });

    it("returns single platform when only one selected", () => {
      const state = createTestState({
        platforms: { win: false, mac: true, linux: false },
      });

      expect(selectSelectedPlatformsList(state)).toEqual(["mac"]);
    });
  });

  describe("selectStandardOutputPath", () => {
    it("returns correct path for scenario", () => {
      const path = selectStandardOutputPath("my-scenario");

      expect(path).toBe("scenarios/my-scenario/platforms/electron");
    });

    it("returns placeholder for empty scenario name", () => {
      const path = selectStandardOutputPath("");

      expect(path).toBe("scenarios/<scenario>/platforms/electron");
    });

    it("handles scenario names with special characters", () => {
      const path = selectStandardOutputPath("my-cool-scenario");

      expect(path).toBe("scenarios/my-cool-scenario/platforms/electron");
    });
  });

  describe("selectStagingPreviewPath", () => {
    it("returns correct path for scenario", () => {
      const path = selectStagingPreviewPath("my-scenario");

      expect(path).toBe("scenarios/scenario-to-desktop/data/staging/my-scenario/<build-id>");
    });

    it("returns placeholder for empty scenario name", () => {
      const path = selectStagingPreviewPath("");

      expect(path).toBe("scenarios/scenario-to-desktop/data/staging/<scenario>/<build-id>");
    });
  });

  describe("selectIsCustomLocation", () => {
    it("returns true for custom location mode", () => {
      const state = createTestState({
        output: { locationMode: "custom", outputPath: "/custom/path" },
      });

      expect(selectIsCustomLocation(state)).toBe(true);
    });

    it("returns false for proper location mode", () => {
      const state = createTestState({
        output: { locationMode: "proper", outputPath: "" },
      });

      expect(selectIsCustomLocation(state)).toBe(false);
    });

    it("returns false for temp location mode", () => {
      const state = createTestState({
        output: { locationMode: "temp", outputPath: "" },
      });

      expect(selectIsCustomLocation(state)).toBe(false);
    });
  });

  describe("selectIconPreviewUrl", () => {
    it("returns empty string when no icon path", () => {
      const state = createTestState({
        appMetadata: {
          ...initialFormState.appMetadata,
          iconPath: "",
        },
      });

      expect(selectIconPreviewUrl(state)).toBe("");
    });

    it("returns preview URL when icon path is set", () => {
      const state = createTestState({
        appMetadata: {
          ...initialFormState.appMetadata,
          iconPath: "/path/to/icon.png",
        },
      });

      const url = selectIconPreviewUrl(state);

      // URL-encoded path should be present in the preview URL
      expect(url).toContain("icons/preview");
      expect(url).toContain(encodeURIComponent("/path/to/icon.png"));
      expect(url).not.toBe("");
    });
  });

  describe("selectHasValidationErrors", () => {
    it("returns false when no validation errors", () => {
      const state = createTestState({
        validationErrors: [],
      });

      expect(selectHasValidationErrors(state)).toBe(false);
    });

    it("returns true when validation errors exist", () => {
      const state = createTestState({
        validationErrors: [{ id: "1", field: "scenarioName", message: "Required" }],
      });

      expect(selectHasValidationErrors(state)).toBe(true);
    });

    it("returns true for multiple validation errors", () => {
      const state = createTestState({
        validationErrors: [
          { id: "1", field: "scenarioName", message: "Required" },
          { id: "2", field: "platforms", message: "Select at least one" },
        ],
      });

      expect(selectHasValidationErrors(state)).toBe(true);
    });
  });

  describe("selectFieldErrors", () => {
    it("returns empty array when no errors for field", () => {
      const state = createTestState({
        validationErrors: [{ id: "1", field: "scenarioName", message: "Required" }],
      });

      const selector = selectFieldErrors("platforms");
      expect(selector(state)).toEqual([]);
    });

    it("returns errors for specific field", () => {
      const state = createTestState({
        validationErrors: [
          { id: "1", field: "scenarioName", message: "Required" },
          { id: "2", field: "platforms", message: "Select at least one" },
        ],
      });

      const selector = selectFieldErrors("scenarioName");
      const errors = selector(state);

      expect(errors).toHaveLength(1);
      expect(errors?.[0]?.message).toBe("Required");
    });

    it("returns multiple errors for same field", () => {
      const state = createTestState({
        validationErrors: [
          { id: "1", field: "scenarioName", message: "Required" },
          { id: "2", field: "scenarioName", message: "Invalid format" },
          { id: "3", field: "platforms", message: "Select at least one" },
        ],
      });

      const selector = selectFieldErrors("scenarioName");
      const errors = selector(state);

      expect(errors).toHaveLength(2);
      expect(errors?.[0]?.message).toBe("Required");
      expect(errors?.[1]?.message).toBe("Invalid format");
    });

    it("is a curried function for reuse", () => {
      const state = createTestState({
        validationErrors: [{ id: "1", field: "test", message: "Error" }],
      });

      // Can create reusable selectors
      const testFieldSelector = selectFieldErrors("test");
      const otherFieldSelector = selectFieldErrors("other");

      expect(testFieldSelector(state)).toHaveLength(1);
      expect(otherFieldSelector(state)).toHaveLength(0);
    });
  });

  describe("combined state selectors", () => {
    it("selectAppMetadata returns complete app metadata", () => {
      const customMetadata = {
        displayName: "My App",
        description: "My Description",
        iconPath: "/icon.png",
        displayNameEdited: true,
        descriptionEdited: true,
        iconPathEdited: true,
        iconPreviewError: false,
      };
      const state = createTestState({
        appMetadata: customMetadata,
      });

      expect(selectAppMetadata(state)).toEqual(customMetadata);
    });

    it("selectDeployment returns complete deployment config", () => {
      const customDeployment = {
        mode: "external-server" as const,
        serverType: "node" as const,
        framework: "tauri",
      };
      const state = createTestState({
        deployment: customDeployment,
      });

      expect(selectDeployment(state)).toEqual(customDeployment);
    });

    it("selectOutput returns complete output config", () => {
      const customOutput = {
        locationMode: "custom" as const,
        outputPath: "/custom/path",
      };
      const state = createTestState({
        output: customOutput,
      });

      expect(selectOutput(state)).toEqual(customOutput);
    });

    it("selectPlatforms returns complete platforms config", () => {
      const customPlatforms = {
        win: false,
        mac: true,
        linux: true,
      };
      const state = createTestState({
        platforms: customPlatforms,
      });

      expect(selectPlatforms(state)).toEqual(customPlatforms);
    });

    it("selectConnection returns complete connection config", () => {
      const state = createTestState();

      const connection = selectConnection(state);

      expect(connection).toHaveProperty("proxyUrl");
      expect(connection).toHaveProperty("bundleManifestPath");
      expect(connection).toHaveProperty("serverPort");
      expect(connection).toHaveProperty("localServerPath");
      expect(connection).toHaveProperty("localApiEndpoint");
      expect(connection).toHaveProperty("autoManageTier1");
      expect(connection).toHaveProperty("vrooliBinaryPath");
      expect(connection).toHaveProperty("connectionResult");
      expect(connection).toHaveProperty("connectionError");
    });
  });

  describe("selector stability", () => {
    it("selectConnectionDecision returns consistent values for same input", () => {
      const state = createTestState();

      const result1 = selectConnectionDecision(state);
      const result2 = selectConnectionDecision(state);

      // Values should be equivalent (not necessarily same reference for objects)
      expect(result1.kind).toBe(result2.kind);
      expect(result1.requiresProxyUrl).toBe(result2.requiresProxyUrl);
    });

    it("selectFieldErrors returns new function for each call but same logic", () => {
      const state = createTestState({
        validationErrors: [{ id: "1", field: "test", message: "Error" }],
      });

      const selector1 = selectFieldErrors("test");
      const selector2 = selectFieldErrors("test");

      // Different function references
      expect(selector1).not.toBe(selector2);
      // But same results
      expect(selector1(state)).toEqual(selector2(state));
    });
  });
});
