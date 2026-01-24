/**
 * Tests for formStore.
 * Tests all actions and state mutations.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { act } from "@testing-library/react";
import { useFormStore } from "./formStore";
import {
  defaultAppMetadata,
  defaultDeployment,
  defaultOutput,
  defaultPlatforms,
  defaultConnection,
} from "./formTypes";

// Reset store state before each test
beforeEach(() => {
  act(() => {
    useFormStore.getState().resetFormState();
  });
});

describe("formStore", () => {
  describe("initial state", () => {
    it("starts with correct default values", () => {
      const state = useFormStore.getState();

      expect(state.appMetadata).toEqual(defaultAppMetadata);
      expect(state.deployment).toEqual(defaultDeployment);
      expect(state.output).toEqual(defaultOutput);
      expect(state.platforms).toEqual(defaultPlatforms);
      expect(state.connection).toEqual(defaultConnection);
      expect(state.signingEnabledForBuild).toBe(false);
      expect(state.selectedTemplate).toBe("basic");
      expect(state.validationErrors).toEqual([]);
      expect(state.scenarioLocked).toBe(false);
    });
  });

  describe("app metadata setters", () => {
    it("setAppDisplayName updates display name and marks as edited", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setAppDisplayName("My App");
      });

      const state = useFormStore.getState();
      expect(state.appMetadata.displayName).toBe("My App");
      expect(state.appMetadata.displayNameEdited).toBe(true);
    });

    it("setAppDescription updates description and marks as edited", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setAppDescription("My app description");
      });

      const state = useFormStore.getState();
      expect(state.appMetadata.description).toBe("My app description");
      expect(state.appMetadata.descriptionEdited).toBe(true);
    });

    it("setIconPath updates icon path and marks as edited", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setIconPath("/path/to/icon.png");
      });

      const state = useFormStore.getState();
      expect(state.appMetadata.iconPath).toBe("/path/to/icon.png");
      expect(state.appMetadata.iconPathEdited).toBe(true);
    });

    it("setIconPreviewError updates icon preview error state", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setIconPreviewError(true);
      });

      expect(useFormStore.getState().appMetadata.iconPreviewError).toBe(true);

      act(() => {
        store.setIconPreviewError(false);
      });

      expect(useFormStore.getState().appMetadata.iconPreviewError).toBe(false);
    });
  });

  describe("deployment setters", () => {
    it("setDeploymentMode updates deployment mode", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setDeploymentMode("external-server");
      });

      expect(useFormStore.getState().deployment.mode).toBe("external-server");
    });

    it("setServerType updates server type", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setServerType("node");
      });

      expect(useFormStore.getState().deployment.serverType).toBe("node");
    });

    it("setFramework updates framework", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setFramework("tauri");
      });

      expect(useFormStore.getState().deployment.framework).toBe("tauri");
    });
  });

  describe("output setters", () => {
    it("setLocationMode updates location mode", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setLocationMode("custom");
      });

      expect(useFormStore.getState().output.locationMode).toBe("custom");
    });

    it("setOutputPath updates output path", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setOutputPath("/custom/output/path");
      });

      expect(useFormStore.getState().output.outputPath).toBe("/custom/output/path");
    });
  });

  describe("platform setters", () => {
    it("setPlatforms replaces entire platforms object", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setPlatforms({ win: false, mac: true, linux: false });
      });

      const state = useFormStore.getState();
      expect(state.platforms.win).toBe(false);
      expect(state.platforms.mac).toBe(true);
      expect(state.platforms.linux).toBe(false);
    });

    it("handlePlatformChange updates individual platform", () => {
      const store = useFormStore.getState();

      act(() => {
        store.handlePlatformChange("win", false);
      });

      expect(useFormStore.getState().platforms.win).toBe(false);
      expect(useFormStore.getState().platforms.mac).toBe(true);
      expect(useFormStore.getState().platforms.linux).toBe(true);

      act(() => {
        store.handlePlatformChange("mac", false);
      });

      expect(useFormStore.getState().platforms.mac).toBe(false);
    });
  });

  describe("connection setters", () => {
    it("setProxyUrl updates proxy URL", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setProxyUrl("https://proxy.example.com");
      });

      expect(useFormStore.getState().connection.proxyUrl).toBe("https://proxy.example.com");
    });

    it("setBundleManifestPath updates bundle manifest path", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setBundleManifestPath("/path/to/bundle.json");
      });

      expect(useFormStore.getState().connection.bundleManifestPath).toBe("/path/to/bundle.json");
    });

    it("setServerPort updates server port", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setServerPort(4000);
      });

      expect(useFormStore.getState().connection.serverPort).toBe(4000);
    });

    it("setLocalServerPath updates local server path", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setLocalServerPath("dist/server.js");
      });

      expect(useFormStore.getState().connection.localServerPath).toBe("dist/server.js");
    });

    it("setLocalApiEndpoint updates local API endpoint", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setLocalApiEndpoint("http://localhost:4000/api");
      });

      expect(useFormStore.getState().connection.localApiEndpoint).toBe("http://localhost:4000/api");
    });

    it("setAutoManageTier1 updates auto manage tier 1 flag", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setAutoManageTier1(true);
      });

      expect(useFormStore.getState().connection.autoManageTier1).toBe(true);
    });

    it("setVrooliBinaryPath updates vrooli binary path", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setVrooliBinaryPath("/usr/local/bin/vrooli");
      });

      expect(useFormStore.getState().connection.vrooliBinaryPath).toBe("/usr/local/bin/vrooli");
    });

    it("setConnectionResult updates connection result", () => {
      const store = useFormStore.getState();
      const mockResult = {
        proxy_url: "https://example.com",
        healthy: true,
        api_version: "1.0.0",
      };

      act(() => {
        store.setConnectionResult(mockResult);
      });

      expect(useFormStore.getState().connection.connectionResult).toEqual(mockResult);
    });

    it("setConnectionResult clears result when null", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setConnectionResult({ proxy_url: "https://example.com", healthy: true });
      });

      expect(useFormStore.getState().connection.connectionResult).not.toBeNull();

      act(() => {
        store.setConnectionResult(null);
      });

      expect(useFormStore.getState().connection.connectionResult).toBeNull();
    });

    it("setConnectionError updates connection error", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setConnectionError("Connection failed");
      });

      expect(useFormStore.getState().connection.connectionError).toBe("Connection failed");
    });

    it("setConnectionError clears error when null", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setConnectionError("Some error");
      });

      expect(useFormStore.getState().connection.connectionError).toBe("Some error");

      act(() => {
        store.setConnectionError(null);
      });

      expect(useFormStore.getState().connection.connectionError).toBeNull();
    });
  });

  describe("signing setters", () => {
    it("setSigningEnabledForBuild updates signing enabled state", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setSigningEnabledForBuild(true);
      });

      expect(useFormStore.getState().signingEnabledForBuild).toBe(true);
    });
  });

  describe("template setters", () => {
    it("setSelectedTemplate updates selected template", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setSelectedTemplate("advanced");
      });

      expect(useFormStore.getState().selectedTemplate).toBe("advanced");
    });
  });

  describe("validation setters", () => {
    it("setValidationErrors updates validation errors", () => {
      const store = useFormStore.getState();
      const errors = [
        { field: "scenarioName", message: "Required" },
        { field: "platforms", message: "Select at least one" },
      ];

      act(() => {
        store.setValidationErrors(errors);
      });

      expect(useFormStore.getState().validationErrors).toEqual(errors);
    });

    it("clearValidationErrors clears all validation errors", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setValidationErrors([{ field: "test", message: "error" }]);
      });

      expect(useFormStore.getState().validationErrors).toHaveLength(1);

      act(() => {
        store.clearValidationErrors();
      });

      expect(useFormStore.getState().validationErrors).toEqual([]);
    });
  });

  describe("UI state setters", () => {
    it("setScenarioLocked updates scenario locked state", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setScenarioLocked(true);
      });

      expect(useFormStore.getState().scenarioLocked).toBe(true);
    });
  });

  describe("resetFormState", () => {
    it("resets all state to defaults", () => {
      const store = useFormStore.getState();

      // Make various changes
      act(() => {
        store.setAppDisplayName("Custom Name");
        store.setAppDescription("Custom Description");
        store.setIconPath("/custom/icon.png");
        store.setDeploymentMode("external-server");
        store.setServerType("node");
        store.setFramework("tauri");
        store.setLocationMode("custom");
        store.setOutputPath("/custom/output");
        store.setPlatforms({ win: false, mac: false, linux: true });
        store.setProxyUrl("https://custom.com");
        store.setBundleManifestPath("/bundle.json");
        store.setServerPort(5000);
        store.setAutoManageTier1(true);
        store.setSigningEnabledForBuild(true);
        store.setSelectedTemplate("advanced");
        store.setValidationErrors([{ field: "test", message: "error" }]);
        store.setScenarioLocked(true);
      });

      // Reset
      act(() => {
        store.resetFormState();
      });

      const state = useFormStore.getState();

      // Verify all state is reset to defaults
      expect(state.appMetadata).toEqual(defaultAppMetadata);
      expect(state.deployment).toEqual(defaultDeployment);
      expect(state.output).toEqual(defaultOutput);
      expect(state.platforms).toEqual(defaultPlatforms);
      expect(state.connection).toEqual(defaultConnection);
      expect(state.signingEnabledForBuild).toBe(false);
      expect(state.selectedTemplate).toBe("basic");
      expect(state.validationErrors).toEqual([]);
      expect(state.scenarioLocked).toBe(false);
    });
  });

  describe("hydrateFromServer", () => {
    it("merges partial appMetadata", () => {
      const store = useFormStore.getState();

      act(() => {
        store.hydrateFromServer({
          appMetadata: {
            displayName: "Server App",
            displayNameEdited: true,
          },
        });
      });

      const state = useFormStore.getState();
      expect(state.appMetadata.displayName).toBe("Server App");
      expect(state.appMetadata.displayNameEdited).toBe(true);
      // Other fields should remain at defaults
      expect(state.appMetadata.description).toBe("");
      expect(state.appMetadata.descriptionEdited).toBe(false);
    });

    it("merges partial deployment", () => {
      const store = useFormStore.getState();

      act(() => {
        store.hydrateFromServer({
          deployment: {
            framework: "tauri",
          },
        });
      });

      const state = useFormStore.getState();
      expect(state.deployment.framework).toBe("tauri");
      // Other fields should remain at defaults
      expect(state.deployment.mode).toBe("bundled");
      expect(state.deployment.serverType).toBe("external");
    });

    it("merges partial output", () => {
      const store = useFormStore.getState();

      act(() => {
        store.hydrateFromServer({
          output: {
            locationMode: "custom",
            outputPath: "/custom/path",
          },
        });
      });

      const state = useFormStore.getState();
      expect(state.output.locationMode).toBe("custom");
      expect(state.output.outputPath).toBe("/custom/path");
    });

    it("replaces platforms completely", () => {
      const store = useFormStore.getState();

      act(() => {
        store.hydrateFromServer({
          platforms: { win: false, mac: true, linux: false },
        });
      });

      const state = useFormStore.getState();
      expect(state.platforms).toEqual({ win: false, mac: true, linux: false });
    });

    it("merges partial connection", () => {
      const store = useFormStore.getState();

      act(() => {
        store.hydrateFromServer({
          connection: {
            proxyUrl: "https://api.example.com",
            serverPort: 5000,
          },
        });
      });

      const state = useFormStore.getState();
      expect(state.connection.proxyUrl).toBe("https://api.example.com");
      expect(state.connection.serverPort).toBe(5000);
      // Other fields should remain at defaults
      expect(state.connection.localServerPath).toBe("ui/server.js");
    });

    it("updates signingEnabledForBuild when provided", () => {
      const store = useFormStore.getState();

      act(() => {
        store.hydrateFromServer({
          signingEnabledForBuild: true,
        });
      });

      expect(useFormStore.getState().signingEnabledForBuild).toBe(true);
    });

    it("updates selectedTemplate when provided", () => {
      const store = useFormStore.getState();

      act(() => {
        store.hydrateFromServer({
          selectedTemplate: "advanced",
        });
      });

      expect(useFormStore.getState().selectedTemplate).toBe("advanced");
    });

    it("does not modify state when data is empty", () => {
      const store = useFormStore.getState();

      // Set some custom state first
      act(() => {
        store.setAppDisplayName("Custom Name");
      });

      // Hydrate with empty object
      act(() => {
        store.hydrateFromServer({});
      });

      // State should be unchanged
      expect(useFormStore.getState().appMetadata.displayName).toBe("Custom Name");
    });

    it("handles all fields in one call", () => {
      const store = useFormStore.getState();

      act(() => {
        store.hydrateFromServer({
          appMetadata: {
            displayName: "Full App",
            description: "Full description",
            iconPath: "/icon.png",
            displayNameEdited: true,
            descriptionEdited: true,
            iconPathEdited: true,
            iconPreviewError: false,
          },
          deployment: {
            mode: "external-server",
            serverType: "node",
            framework: "electron",
          },
          output: {
            locationMode: "temp",
            outputPath: "/temp/path",
          },
          platforms: {
            win: true,
            mac: false,
            linux: true,
          },
          connection: {
            proxyUrl: "https://api.example.com",
            bundleManifestPath: "/bundle.json",
            serverPort: 4000,
            localServerPath: "server.js",
            localApiEndpoint: "http://localhost:4000",
            autoManageTier1: true,
            vrooliBinaryPath: "/bin/vrooli",
            connectionResult: null,
            connectionError: null,
          },
          signingEnabledForBuild: true,
          selectedTemplate: "multi_window",
        });
      });

      const state = useFormStore.getState();
      expect(state.appMetadata.displayName).toBe("Full App");
      expect(state.deployment.mode).toBe("external-server");
      expect(state.output.locationMode).toBe("temp");
      expect(state.platforms.mac).toBe(false);
      expect(state.connection.proxyUrl).toBe("https://api.example.com");
      expect(state.signingEnabledForBuild).toBe(true);
      expect(state.selectedTemplate).toBe("multi_window");
    });
  });

  describe("state isolation", () => {
    it("setter for one field does not affect other fields", () => {
      const store = useFormStore.getState();

      act(() => {
        store.setAppDisplayName("Changed");
      });

      const state = useFormStore.getState();
      // Only displayName should change
      expect(state.appMetadata.displayName).toBe("Changed");
      expect(state.appMetadata.description).toBe("");
      expect(state.appMetadata.iconPath).toBe("");
      expect(state.deployment.mode).toBe("bundled");
      expect(state.connection.proxyUrl).toBe("");
    });

    it("nested state updates preserve other nested fields", () => {
      const store = useFormStore.getState();

      // First update
      act(() => {
        store.setAppDisplayName("Name 1");
      });

      // Second update to different field in same nested object
      act(() => {
        store.setAppDescription("Description 1");
      });

      const state = useFormStore.getState();
      // Both should be preserved
      expect(state.appMetadata.displayName).toBe("Name 1");
      expect(state.appMetadata.description).toBe("Description 1");
    });
  });
});
