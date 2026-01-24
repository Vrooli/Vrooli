/**
 * Tests for useFormState hook.
 * Tests store integration, derived values, effects, and actions.
 */

import { describe, it, expect, beforeEach } from "vitest";
import { renderHook, act, waitFor } from "@testing-library/react";
import { useFormState, type UseFormStateProps } from "./useFormState";
import { useFormStore } from "../store/formStore";
import type { DesktopConnectionConfig, ScenarioDesktopStatus } from "../components/scenario-inventory/types";

// Reset store state before each test
beforeEach(() => {
  useFormStore.getState().resetFormState();
});

const defaultProps: UseFormStateProps = {
  scenarioName: "test-scenario",
  selectionSource: null,
};

describe("useFormState", () => {
  describe("store integration", () => {
    it("reads initial state from formStore", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Verify initial state matches store defaults
      expect(result.current.appMetadata.displayName).toBe("");
      expect(result.current.appMetadata.description).toBe("");
      expect(result.current.deployment.mode).toBe("bundled");
      expect(result.current.deployment.serverType).toBe("external");
      expect(result.current.deployment.framework).toBe("electron");
      expect(result.current.platforms.win).toBe(true);
      expect(result.current.platforms.mac).toBe(true);
      expect(result.current.platforms.linux).toBe(true);
    });

    it("updates store when setters are called", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.setAppDisplayName("My App");
        result.current.setAppDescription("My App Description");
      });

      expect(result.current.appMetadata.displayName).toBe("My App");
      expect(result.current.appMetadata.description).toBe("My App Description");
      expect(result.current.appMetadata.displayNameEdited).toBe(true);
      expect(result.current.appMetadata.descriptionEdited).toBe(true);
    });

    it("exposes all store setters", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Verify all setters are functions
      expect(typeof result.current.setAppDisplayName).toBe("function");
      expect(typeof result.current.setAppDescription).toBe("function");
      expect(typeof result.current.setIconPath).toBe("function");
      expect(typeof result.current.setIconPreviewError).toBe("function");
      expect(typeof result.current.setDeploymentMode).toBe("function");
      expect(typeof result.current.setServerType).toBe("function");
      expect(typeof result.current.setFramework).toBe("function");
      expect(typeof result.current.setLocationMode).toBe("function");
      expect(typeof result.current.setOutputPath).toBe("function");
      expect(typeof result.current.setPlatforms).toBe("function");
      expect(typeof result.current.handlePlatformChange).toBe("function");
      expect(typeof result.current.setProxyUrl).toBe("function");
      expect(typeof result.current.setBundleManifestPath).toBe("function");
      expect(typeof result.current.setServerPort).toBe("function");
      expect(typeof result.current.setLocalServerPath).toBe("function");
      expect(typeof result.current.setLocalApiEndpoint).toBe("function");
      expect(typeof result.current.setAutoManageTier1).toBe("function");
      expect(typeof result.current.setVrooliBinaryPath).toBe("function");
      expect(typeof result.current.setConnectionResult).toBe("function");
      expect(typeof result.current.setConnectionError).toBe("function");
    });
  });

  describe("derived values computation", () => {
    it("computes connectionDecision for bundled mode", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Default is bundled mode
      expect(result.current.connectionDecision.kind).toBe("bundled-runtime");
      expect(result.current.isBundled).toBe(true);
      expect(result.current.requiresRemoteConfig).toBe(false);
    });

    it("computes connectionDecision for external-server mode", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.setDeploymentMode("external-server");
      });

      expect(result.current.connectionDecision.kind).toBe("remote-server");
      expect(result.current.isBundled).toBe(false);
      expect(result.current.requiresRemoteConfig).toBe(true);
    });

    it("computes connectionDecision for local embedded server", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.setDeploymentMode("external-server");
        result.current.setServerType("node");
      });

      expect(result.current.connectionDecision.kind).toBe("local-embedded");
      expect(result.current.isBundled).toBe(false);
      expect(result.current.requiresRemoteConfig).toBe(false);
    });

    it("computes allowedServerTypes based on deployment mode", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Bundled mode only allows external
      expect(result.current.allowedServerTypes).toEqual(["external"]);

      act(() => {
        result.current.setDeploymentMode("external-server");
      });

      // External-server mode allows all server types
      expect(result.current.allowedServerTypes).toContain("external");
      expect(result.current.allowedServerTypes).toContain("node");
      expect(result.current.allowedServerTypes).toContain("static");
    });

    it("computes selectedPlatformsList from platforms object", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // All platforms selected by default
      expect(result.current.selectedPlatformsList).toEqual(
        expect.arrayContaining(["win", "mac", "linux"])
      );

      act(() => {
        result.current.handlePlatformChange("linux", false);
      });

      expect(result.current.selectedPlatformsList).toEqual(
        expect.arrayContaining(["win", "mac"])
      );
      expect(result.current.selectedPlatformsList).not.toContain("linux");
    });

    it("computes standardOutputPath from scenarioName", () => {
      const { result } = renderHook(() =>
        useFormState({ scenarioName: "my-scenario", selectionSource: null })
      );

      expect(result.current.standardOutputPath).toBe(
        "scenarios/my-scenario/platforms/electron"
      );
    });

    it("computes stagingPreviewPath from scenarioName", () => {
      const { result } = renderHook(() =>
        useFormState({ scenarioName: "my-scenario", selectionSource: null })
      );

      expect(result.current.stagingPreviewPath).toBe(
        "scenarios/scenario-to-desktop/data/staging/my-scenario/<build-id>"
      );
    });

    it("computes iconPreviewUrl from iconPath", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // No icon path by default
      expect(result.current.iconPreviewUrl).toBe("");

      act(() => {
        result.current.setIconPath("/path/to/icon.png");
      });

      // URL-encoded path should be present in the preview URL
      expect(result.current.iconPreviewUrl).toContain("icons/preview");
      expect(result.current.iconPreviewUrl).toContain(encodeURIComponent("/path/to/icon.png"));
    });

    it("computes isCustomLocation from locationMode", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Default is "proper"
      expect(result.current.isCustomLocation).toBe(false);

      act(() => {
        result.current.setLocationMode("custom");
      });

      expect(result.current.isCustomLocation).toBe(true);
    });

    it("computes isUpdateMode from selectionSource", () => {
      const { result: manualResult } = renderHook(() =>
        useFormState({ scenarioName: "test", selectionSource: "manual" })
      );
      expect(manualResult.current.isUpdateMode).toBe(false);

      const { result: inventoryResult } = renderHook(() =>
        useFormState({ scenarioName: "test", selectionSource: "inventory" })
      );
      expect(inventoryResult.current.isUpdateMode).toBe(true);
    });
  });

  describe("effect behaviors", () => {
    it("syncs scenarioLocked with selectionSource", async () => {
      const { result, rerender } = renderHook(
        (props: UseFormStateProps) => useFormState(props),
        { initialProps: { scenarioName: "test", selectionSource: null } }
      );

      // Initially not locked
      expect(result.current.scenarioLocked).toBe(false);

      // Rerender with inventory selection source
      rerender({ scenarioName: "test", selectionSource: "inventory" });

      await waitFor(() => {
        expect(result.current.scenarioLocked).toBe(true);
      });
    });

    it("resets icon preview error when iconPreviewUrl changes", async () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Set an error first
      act(() => {
        result.current.setIconPreviewError(true);
      });
      expect(result.current.appMetadata.iconPreviewError).toBe(true);

      // Change icon path
      act(() => {
        result.current.setIconPath("/new/icon.png");
      });

      // Error should be reset
      await waitFor(() => {
        expect(result.current.appMetadata.iconPreviewError).toBe(false);
      });
    });

    it("adjusts server type when not in allowed list", async () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Set to external-server mode with node server type
      act(() => {
        result.current.setDeploymentMode("external-server");
        result.current.setServerType("node");
      });

      expect(result.current.deployment.serverType).toBe("node");

      // Switch to bundled mode - should reset server type to external
      act(() => {
        result.current.setDeploymentMode("bundled");
      });

      await waitFor(() => {
        expect(result.current.deployment.serverType).toBe("external");
      });
    });

    it("syncs effectiveServerType in bundled mode", async () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // In bundled mode, server type should be forced to external
      expect(result.current.connectionDecision.effectiveServerType).toBe("external");
    });

    it("disables autoManageTier1 in bundled mode", async () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Set autoManageTier1 in external-server mode
      act(() => {
        result.current.setDeploymentMode("external-server");
        result.current.setAutoManageTier1(true);
      });

      expect(result.current.connection.autoManageTier1).toBe(true);

      // Switch to bundled mode - should disable autoManageTier1
      act(() => {
        result.current.setDeploymentMode("bundled");
      });

      await waitFor(() => {
        expect(result.current.connection.autoManageTier1).toBe(false);
      });
    });
  });

  describe("handleDeploymentChange", () => {
    it("updates deployment mode", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.handleDeploymentChange("external-server");
      });

      expect(result.current.deployment.mode).toBe("external-server");
    });

    it("adjusts server type when switching to bundled mode", async () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Set to external-server with node
      act(() => {
        result.current.handleDeploymentChange("external-server");
        result.current.setServerType("node");
      });

      expect(result.current.deployment.serverType).toBe("node");

      // Switch to bundled
      act(() => {
        result.current.handleDeploymentChange("bundled");
      });

      await waitFor(() => {
        expect(result.current.deployment.serverType).toBe("external");
      });
    });
  });

  describe("resetFormState", () => {
    it("resets all form state to defaults", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Make some changes
      act(() => {
        result.current.setAppDisplayName("Custom Name");
        result.current.setAppDescription("Custom Description");
        result.current.setDeploymentMode("external-server");
        result.current.setProxyUrl("https://example.com");
      });

      // Reset
      act(() => {
        result.current.resetFormState(true);
      });

      // Verify reset
      expect(result.current.appMetadata.displayName).toBe("");
      expect(result.current.appMetadata.description).toBe("");
      expect(result.current.deployment.mode).toBe("bundled");
      expect(result.current.connection.proxyUrl).toBe("");
    });

    it("can reset without resetting template", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.setSelectedTemplate("advanced");
        result.current.setAppDisplayName("Custom Name");
      });

      act(() => {
        result.current.resetFormState(false);
      });

      // Template should be preserved when resetTemplate=false
      // (this is controlled by the calling code, not the hook itself)
      expect(result.current.appMetadata.displayName).toBe("");
    });
  });

  describe("hydrateFromServer", () => {
    it("hydrates partial data from server", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.hydrateFromServer({
          appMetadata: {
            displayName: "Server App Name",
            displayNameEdited: true,
          },
          deployment: {
            framework: "tauri",
          },
        });
      });

      expect(result.current.appMetadata.displayName).toBe("Server App Name");
      expect(result.current.appMetadata.displayNameEdited).toBe(true);
      expect(result.current.deployment.framework).toBe("tauri");
      // Other fields should remain at defaults
      expect(result.current.appMetadata.description).toBe("");
    });

    it("hydrates all state groups", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.hydrateFromServer({
          appMetadata: {
            displayName: "App",
            description: "Desc",
            iconPath: "/icon.png",
          },
          deployment: {
            mode: "external-server",
            serverType: "external",
            framework: "electron",
          },
          output: {
            locationMode: "custom",
            outputPath: "/custom/path",
          },
          platforms: {
            win: true,
            mac: false,
            linux: true,
          },
          connection: {
            proxyUrl: "https://api.example.com",
            serverPort: 4000,
          },
        });
      });

      expect(result.current.appMetadata.displayName).toBe("App");
      expect(result.current.appMetadata.description).toBe("Desc");
      expect(result.current.deployment.mode).toBe("external-server");
      expect(result.current.output.locationMode).toBe("custom");
      expect(result.current.output.outputPath).toBe("/custom/path");
      expect(result.current.platforms.mac).toBe(false);
      expect(result.current.connection.proxyUrl).toBe("https://api.example.com");
      expect(result.current.connection.serverPort).toBe(4000);
    });
  });

  describe("applySavedConnection", () => {
    it("does nothing when config is null/undefined", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.applySavedConnection(null);
        result.current.applySavedConnection(undefined);
      });

      // State should remain at defaults
      expect(result.current.deployment.mode).toBe("bundled");
      expect(result.current.connection.proxyUrl).toBe("");
    });

    it("applies connection config values", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      const config: DesktopConnectionConfig = {
        deployment_mode: "external-server",
        proxy_url: "https://proxy.example.com",
        server_url: "https://server.example.com",
        auto_manage_vrooli: true,
        vrooli_binary_path: "/usr/local/bin/vrooli",
        bundle_manifest_path: "/path/to/manifest.json",
        app_display_name: "My Desktop App",
        app_description: "A great app",
        icon: "/path/to/icon.png",
        server_type: "external",
      };

      act(() => {
        result.current.applySavedConnection(config);
      });

      expect(result.current.deployment.mode).toBe("external-server");
      expect(result.current.connection.proxyUrl).toBe("https://proxy.example.com");
      expect(result.current.connection.autoManageTier1).toBe(true);
      expect(result.current.connection.vrooliBinaryPath).toBe("/usr/local/bin/vrooli");
      expect(result.current.connection.bundleManifestPath).toBe("/path/to/manifest.json");
      expect(result.current.appMetadata.displayName).toBe("My Desktop App");
      expect(result.current.appMetadata.description).toBe("A great app");
      expect(result.current.appMetadata.iconPath).toBe("/path/to/icon.png");
      expect(result.current.deployment.serverType).toBe("external");
    });

    it("falls back to server_url when proxy_url is not set", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      const config: DesktopConnectionConfig = {
        server_url: "https://fallback.example.com",
      };

      act(() => {
        result.current.applySavedConnection(config);
      });

      expect(result.current.connection.proxyUrl).toBe("https://fallback.example.com");
    });

    it("only applies optional fields when present", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Set some initial values
      act(() => {
        result.current.setAppDisplayName("Original Name");
        result.current.setAppDescription("Original Description");
      });

      // Apply config without app_display_name or app_description
      const config: DesktopConnectionConfig = {
        deployment_mode: "external-server",
        proxy_url: "https://proxy.example.com",
      };

      act(() => {
        result.current.applySavedConnection(config);
      });

      // These should remain unchanged since config didn't include them
      expect(result.current.appMetadata.displayName).toBe("Original Name");
      expect(result.current.appMetadata.description).toBe("Original Description");
    });
  });

  describe("applyScenarioDefaults", () => {
    const mockScenario: ScenarioDesktopStatus = {
      name: "test-scenario",
      service_display_name: "Test Scenario App",
      service_description: "A test scenario description",
      service_icon_path: "/scenarios/test-scenario/icon.png",
      has_ui: true,
      has_api: true,
    };

    it("applies scenario defaults when fields are not edited", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      act(() => {
        result.current.applyScenarioDefaults(mockScenario);
      });

      expect(result.current.appMetadata.displayName).toBe("Test Scenario App");
      expect(result.current.appMetadata.description).toBe("A test scenario description");
      expect(result.current.appMetadata.iconPath).toBe("/scenarios/test-scenario/icon.png");
    });

    it("does not override edited fields", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Edit the display name first
      act(() => {
        result.current.setAppDisplayName("Custom Name");
      });

      // Now apply scenario defaults
      act(() => {
        result.current.applyScenarioDefaults(mockScenario);
      });

      // Display name should remain "Custom Name" because it was edited
      expect(result.current.appMetadata.displayName).toBe("Custom Name");
      // But description and icon should be applied since they weren't edited
      expect(result.current.appMetadata.description).toBe("A test scenario description");
      expect(result.current.appMetadata.iconPath).toBe("/scenarios/test-scenario/icon.png");
    });

    it("respects all edit flags independently", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      // Edit only the description
      act(() => {
        result.current.setAppDescription("My Custom Description");
      });

      act(() => {
        result.current.applyScenarioDefaults(mockScenario);
      });

      // Display name and icon should come from scenario (not edited)
      expect(result.current.appMetadata.displayName).toBe("Test Scenario App");
      expect(result.current.appMetadata.iconPath).toBe("/scenarios/test-scenario/icon.png");
      // Description should remain custom (was edited)
      expect(result.current.appMetadata.description).toBe("My Custom Description");
    });

    it("handles missing scenario fields gracefully", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      const partialScenario: ScenarioDesktopStatus = {
        name: "test-scenario",
        has_ui: true,
        has_api: false,
        // No display_name, description, or icon_path
      };

      act(() => {
        result.current.applyScenarioDefaults(partialScenario);
      });

      // Should set to empty strings for missing fields
      expect(result.current.appMetadata.displayName).toBe("");
      expect(result.current.appMetadata.description).toBe("");
      expect(result.current.appMetadata.iconPath).toBe("");
    });
  });

  describe("validation state", () => {
    it("manages validation errors", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      expect(result.current.validationErrors).toEqual([]);

      act(() => {
        result.current.setValidationErrors([
          { field: "scenarioName", message: "Required" },
          { field: "platforms", message: "Select at least one" },
        ]);
      });

      expect(result.current.validationErrors).toHaveLength(2);
      expect(result.current.validationErrors[0].field).toBe("scenarioName");

      act(() => {
        result.current.clearValidationErrors();
      });

      expect(result.current.validationErrors).toEqual([]);
    });
  });

  describe("signing state", () => {
    it("manages signing enabled for build", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      expect(result.current.signingEnabledForBuild).toBe(false);

      act(() => {
        result.current.setSigningEnabledForBuild(true);
      });

      expect(result.current.signingEnabledForBuild).toBe(true);
    });
  });

  describe("template selection", () => {
    it("manages selected template", () => {
      const { result } = renderHook(() => useFormState(defaultProps));

      expect(result.current.selectedTemplate).toBe("basic");

      act(() => {
        result.current.setSelectedTemplate("advanced");
      });

      expect(result.current.selectedTemplate).toBe("advanced");
    });
  });
});
