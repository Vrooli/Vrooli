import { describe, it, expect } from "vitest";
import {
  buildIconPreviewUrl,
  extractScenarioDefaults,
  applyScenarioDefaults,
  transformConnectionConfigToFormState,
  buildPipelineConfigFromForm,
  buildValidationParams,
  getAllowedServerTypes,
  adjustServerTypeForMode,
  serializeFormStateForServer,
  deserializeFormStateFromServer,
  type GeneratorFormState,
  type SerializedFormState,
} from "./generator.service";
import type { DesktopConnectionConfig } from "../components/scenario-inventory/types";

describe("generator.service", () => {
  describe("buildIconPreviewUrl", () => {
    it("returns empty string for empty path", () => {
      expect(buildIconPreviewUrl("")).toBe("");
    });

    it("builds URL with encoded path", () => {
      const url = buildIconPreviewUrl("/path/to/icon.png");
      expect(url).toBe("/api/icon-preview?path=%2Fpath%2Fto%2Ficon.png");
    });

    it("uses custom API base URL", () => {
      const url = buildIconPreviewUrl("/icon.png", "http://localhost:3000");
      expect(url).toBe("http://localhost:3000/api/icon-preview?path=%2Ficon.png");
    });

    it("encodes special characters", () => {
      const url = buildIconPreviewUrl("/path with spaces/icon.png");
      expect(url).toContain("%20");
    });
  });

  describe("extractScenarioDefaults", () => {
    it("returns empty values for null scenario", () => {
      const defaults = extractScenarioDefaults(null);
      expect(defaults).toEqual({
        displayName: "",
        description: "",
        iconPath: "",
      });
    });

    it("extracts values from scenario", () => {
      const scenario = {
        service_display_name: "My App",
        service_description: "App description",
        service_icon_path: "/icons/app.png",
      };
      const defaults = extractScenarioDefaults(scenario);
      expect(defaults).toEqual({
        displayName: "My App",
        description: "App description",
        iconPath: "/icons/app.png",
      });
    });

    it("handles missing optional fields", () => {
      const scenario = {
        service_display_name: "My App",
      };
      const defaults = extractScenarioDefaults(scenario);
      expect(defaults.displayName).toBe("My App");
      expect(defaults.description).toBe("");
      expect(defaults.iconPath).toBe("");
    });
  });

  describe("applyScenarioDefaults", () => {
    it("applies all defaults when nothing edited", () => {
      const scenario = {
        displayName: "Default Name",
        description: "Default Desc",
        iconPath: "/default.png",
      };
      const currentState = {
        displayNameEdited: false,
        descriptionEdited: false,
        iconPathEdited: false,
      };
      const updates = applyScenarioDefaults(scenario, currentState);
      expect(updates).toEqual({
        displayName: "Default Name",
        description: "Default Desc",
        iconPath: "/default.png",
      });
    });

    it("skips edited fields", () => {
      const scenario = {
        displayName: "Default Name",
        description: "Default Desc",
        iconPath: "/default.png",
      };
      const currentState = {
        displayNameEdited: true,
        descriptionEdited: false,
        iconPathEdited: true,
      };
      const updates = applyScenarioDefaults(scenario, currentState);
      expect(updates.displayName).toBeUndefined();
      expect(updates.description).toBe("Default Desc");
      expect(updates.iconPath).toBeUndefined();
    });

    it("returns empty object when all edited", () => {
      const scenario = {
        displayName: "Default Name",
        description: "Default Desc",
        iconPath: "/default.png",
      };
      const currentState = {
        displayNameEdited: true,
        descriptionEdited: true,
        iconPathEdited: true,
      };
      const updates = applyScenarioDefaults(scenario, currentState);
      expect(updates).toEqual({});
    });
  });

  describe("transformConnectionConfigToFormState", () => {
    it("returns empty object for null config", () => {
      expect(transformConnectionConfigToFormState(null)).toEqual({});
    });

    it("returns empty object for undefined config", () => {
      expect(transformConnectionConfigToFormState(undefined)).toEqual({});
    });

    it("transforms config to form state", () => {
      const config: DesktopConnectionConfig = {
        deployment_mode: "bundled",
        server_type: "external",
        proxy_url: "https://api.example.com",
        bundle_manifest_path: "/path/to/manifest.json",
        app_display_name: "Test App",
        app_description: "Test description",
        icon: "/icons/test.png",
        auto_manage_vrooli: true,
        vrooli_binary_path: "/usr/local/bin/vrooli",
      };
      const result = transformConnectionConfigToFormState(config);

      expect(result.deployment?.mode).toBe("bundled");
      expect(result.deployment?.serverType).toBe("external");
      expect(result.connection?.proxyUrl).toBe("https://api.example.com");
      expect(result.connection?.bundleManifestPath).toBe("/path/to/manifest.json");
      expect(result.appMetadata?.displayName).toBe("Test App");
      expect(result.appMetadata?.description).toBe("Test description");
      expect(result.appMetadata?.iconPath).toBe("/icons/test.png");
      expect(result.connection?.autoManageTier1).toBe(true);
    });

    it("uses server_url as fallback for proxy_url", () => {
      const config: DesktopConnectionConfig = {
        server_url: "https://fallback.example.com",
      };
      const result = transformConnectionConfigToFormState(config);
      expect(result.connection?.proxyUrl).toBe("https://fallback.example.com");
    });
  });

  describe("buildPipelineConfigFromForm", () => {
    const mockFormState: GeneratorFormState = {
      appMetadata: {
        displayName: "Test App",
        description: "Test",
        iconPath: "",
        displayNameEdited: true,
        descriptionEdited: true,
        iconPathEdited: false,
        iconPreviewError: false,
      },
      deployment: {
        mode: "bundled",
        serverType: "external",
        framework: "electron",
      },
      output: {
        locationMode: "proper",
        outputPath: "",
      },
      platforms: { win: true, mac: true, linux: false },
      connection: {
        proxyUrl: "https://api.example.com",
        bundleManifestPath: "/manifest.json",
        serverPort: 3000,
        localServerPath: "ui/server.js",
        localApiEndpoint: "http://localhost:3001/api",
        autoManageTier1: false,
        vrooliBinaryPath: "vrooli",
        connectionResult: null,
        connectionError: null,
      },
      selectedTemplate: "electron-react",
      signingEnabledForBuild: false,
    };

    it("builds pipeline config from form state", () => {
      const config = buildPipelineConfigFromForm(mockFormState, "test-scenario");
      expect(config.scenario_name).toBe("test-scenario");
      expect(config.template_type).toBe("electron-react");
      expect(config.deployment_mode).toBe("bundled");
      expect(config.stop_after_stage).toBe("generate");
    });

    it("sets platforms from form state", () => {
      const config = buildPipelineConfigFromForm(mockFormState, "test-scenario");
      expect(config.platforms).toContain("win");
      expect(config.platforms).toContain("mac");
      expect(config.platforms).not.toContain("linux");
    });

    it("includes proxy_url when provided", () => {
      const config = buildPipelineConfigFromForm(mockFormState, "test-scenario");
      expect(config.proxy_url).toBe("https://api.example.com");
    });

    it("omits proxy_url when empty", () => {
      const stateWithoutProxy = {
        ...mockFormState,
        connection: { ...mockFormState.connection, proxyUrl: "" },
      };
      const config = buildPipelineConfigFromForm(stateWithoutProxy, "test-scenario");
      expect(config.proxy_url).toBeUndefined();
    });
  });

  describe("buildValidationParams", () => {
    const mockFormState: GeneratorFormState = {
      appMetadata: {
        displayName: "Test App",
        description: "Test",
        iconPath: "",
        displayNameEdited: true,
        descriptionEdited: true,
        iconPathEdited: false,
        iconPreviewError: false,
      },
      deployment: {
        mode: "bundled",
        serverType: "external",
        framework: "electron",
      },
      output: {
        locationMode: "proper",
        outputPath: "",
      },
      platforms: { win: true, mac: false, linux: false },
      connection: {
        proxyUrl: "",
        bundleManifestPath: "/manifest.json",
        serverPort: 3000,
        localServerPath: "ui/server.js",
        localApiEndpoint: "http://localhost:3001/api",
        autoManageTier1: false,
        vrooliBinaryPath: "vrooli",
        connectionResult: null,
        connectionError: null,
      },
      selectedTemplate: "electron-react",
      signingEnabledForBuild: true,
    };

    it("builds validation params", () => {
      const params = buildValidationParams(
        mockFormState,
        "test-scenario",
        null,
        false,
        null,
        undefined
      );

      expect(params.scenarioName).toBe("test-scenario");
      expect(params.appDisplayName).toBe("Test App");
      expect(params.bundleManifestPath).toBe("/manifest.json");
      expect(params.signingEnabledForBuild).toBe(true);
    });

    it("determines bundled mode correctly", () => {
      const params = buildValidationParams(
        mockFormState,
        "test-scenario",
        null,
        false,
        null,
        undefined
      );
      expect(params.isBundled).toBe(true);
    });
  });

  describe("getAllowedServerTypes", () => {
    it("returns only external for bundled mode", () => {
      const types = getAllowedServerTypes("bundled");
      expect(types).toEqual(["external"]);
    });

    it("returns only external for cloud-api mode", () => {
      const types = getAllowedServerTypes("cloud-api");
      expect(types).toEqual(["external"]);
    });

    it("returns all types for proxy mode", () => {
      const types = getAllowedServerTypes("proxy");
      expect(types.length).toBeGreaterThan(1);
    });
  });

  describe("adjustServerTypeForMode", () => {
    it("returns external for bundled mode", () => {
      const adjusted = adjustServerTypeForMode("local", "bundled");
      expect(adjusted).toBe("external");
    });

    it("keeps current type if allowed", () => {
      const adjusted = adjustServerTypeForMode("external", "bundled");
      expect(adjusted).toBe("external");
    });
  });

  describe("serializeFormStateForServer", () => {
    const mockFormState: GeneratorFormState = {
      appMetadata: {
        displayName: "My App",
        description: "My Description",
        iconPath: "/icon.png",
        displayNameEdited: true,
        descriptionEdited: false,
        iconPathEdited: true,
        iconPreviewError: false,
      },
      deployment: {
        mode: "bundled",
        serverType: "external",
        framework: "electron",
      },
      output: {
        locationMode: "custom",
        outputPath: "/output/path",
      },
      platforms: { win: true, mac: true, linux: false },
      connection: {
        proxyUrl: "https://api.example.com",
        bundleManifestPath: "/manifest.json",
        serverPort: 3000,
        localServerPath: "ui/server.js",
        localApiEndpoint: "http://localhost:3001/api",
        autoManageTier1: true,
        vrooliBinaryPath: "/usr/local/bin/vrooli",
        connectionResult: null,
        connectionError: null,
      },
      selectedTemplate: "electron-react",
      signingEnabledForBuild: true,
    };

    it("serializes form state to snake_case", () => {
      const serialized = serializeFormStateForServer(mockFormState);

      expect(serialized.selected_template).toBe("electron-react");
      expect(serialized.app_display_name).toBe("My App");
      expect(serialized.app_description).toBe("My Description");
      expect(serialized.icon_path).toBe("/icon.png");
      expect(serialized.display_name_edited).toBe(true);
      expect(serialized.description_edited).toBe(false);
      expect(serialized.deployment_mode).toBe("bundled");
      expect(serialized.server_type).toBe("external");
      expect(serialized.location_mode).toBe("custom");
      expect(serialized.output_path).toBe("/output/path");
      expect(serialized.signing_enabled_for_build).toBe(true);
    });

    it("preserves platform selection", () => {
      const serialized = serializeFormStateForServer(mockFormState);
      expect(serialized.platforms).toEqual({ win: true, mac: true, linux: false });
    });
  });

  describe("deserializeFormStateFromServer", () => {
    it("deserializes server data to form state", () => {
      const serverData: Partial<SerializedFormState> = {
        selected_template: "electron-vue",
        app_display_name: "Server App",
        app_description: "Server Desc",
        icon_path: "/server-icon.png",
        display_name_edited: true,
        description_edited: true,
        deployment_mode: "proxy",
        server_type: "local",
        location_mode: "proper",
        platforms: { win: false, mac: true, linux: true },
        signing_enabled_for_build: false,
      };

      const formState = deserializeFormStateFromServer(serverData);

      expect(formState.selectedTemplate).toBe("electron-vue");
      expect(formState.appMetadata?.displayName).toBe("Server App");
      expect(formState.appMetadata?.description).toBe("Server Desc");
      expect(formState.appMetadata?.iconPath).toBe("/server-icon.png");
      expect(formState.deployment?.mode).toBe("proxy");
      expect(formState.deployment?.serverType).toBe("local");
      expect(formState.platforms).toEqual({ win: false, mac: true, linux: true });
      expect(formState.signingEnabledForBuild).toBe(false);
    });

    it("uses defaults for missing fields", () => {
      const formState = deserializeFormStateFromServer({});

      expect(formState.appMetadata?.displayName).toBe("");
      expect(formState.appMetadata?.displayNameEdited).toBe(false);
      expect(formState.deployment?.framework).toBe("electron");
      expect(formState.platforms).toEqual({ win: true, mac: true, linux: true });
      expect(formState.connection?.serverPort).toBe(3000);
    });
  });
});
