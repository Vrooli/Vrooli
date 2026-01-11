import { describe, expect, it } from "vitest";
import {
  buildDesktopConfig,
  computeStandardOutputPath,
  computeStagingPreviewPath,
  getSelectedPlatforms,
  resolveEndpoints,
  validateGeneratorInputs
} from "./generator";
import { decideConnection } from "./deployment";

describe("generator domain", () => {
  it("returns selected platforms from the selection map", () => {
    const platforms = getSelectedPlatforms({ win: true, mac: false, linux: true });

    expect(platforms).toEqual(["win", "linux"]);
  });

  it("validates required inputs based on deployment decision", () => {
    const remoteDecision = decideConnection("external-server", "external");
    const bundledDecision = decideConnection("bundled", "external");

    expect(
      validateGeneratorInputs({
        selectedPlatforms: [],
        decision: remoteDecision,
        bundleManifestPath: "",
        proxyUrl: "",
        appDisplayName: "App",
        appDescription: "Desc",
        locationMode: "proper",
        outputPath: ""
      })
    ).toBe("Please select at least one target platform");

    expect(
      validateGeneratorInputs({
        selectedPlatforms: ["win"],
        decision: remoteDecision,
        bundleManifestPath: "",
        proxyUrl: "",
        appDisplayName: "App",
        appDescription: "Desc",
        locationMode: "proper",
        outputPath: ""
      })
    ).toBe("Provide the proxy URL you use in the browser (for example https://app-monitor.example.com/apps/<scenario>/proxy/).");

    expect(
      validateGeneratorInputs({
        selectedPlatforms: ["win"],
        decision: bundledDecision,
        bundleManifestPath: "",
        proxyUrl: "https://example.com/proxy/",
        appDisplayName: "App",
        appDescription: "Desc",
        locationMode: "proper",
        outputPath: ""
      })
    ).toBe("Provide bundle_manifest_path from deployment-manager before generating a bundled build.");

    expect(
      validateGeneratorInputs({
        selectedPlatforms: ["win"],
        decision: remoteDecision,
        bundleManifestPath: "",
        proxyUrl: "https://example.com/proxy/",
        appDisplayName: "App",
        appDescription: "Desc",
        locationMode: "custom",
        outputPath: ""
      })
    ).toBe("Provide an output path when choosing a custom location.");

    expect(
      validateGeneratorInputs({
        selectedPlatforms: ["win"],
        decision: remoteDecision,
        bundleManifestPath: "",
        proxyUrl: "https://example.com/proxy/",
        appDisplayName: "App",
        appDescription: "Desc",
        locationMode: "proper",
        outputPath: ""
      })
    ).toBeNull();
  });

  it("resolves endpoints based on deployment mode", () => {
    const bundledDecision = decideConnection("bundled", "external");
    const remoteDecision = decideConnection("external-server", "external");
    const localDecision = decideConnection("external-server", "node");

    expect(
      resolveEndpoints({
        decision: bundledDecision,
        proxyUrl: "https://example.com/proxy/",
        localServerPath: "ui/server.js",
        localApiEndpoint: "http://localhost:3001/api"
      })
    ).toEqual({ serverPath: "http://127.0.0.1", apiEndpoint: "http://127.0.0.1" });

    expect(
      resolveEndpoints({
        decision: remoteDecision,
        proxyUrl: "https://example.com/proxy/",
        localServerPath: "ui/server.js",
        localApiEndpoint: "http://localhost:3001/api"
      })
    ).toEqual({ serverPath: "https://example.com/proxy/", apiEndpoint: "https://example.com/proxy/" });

    expect(
      resolveEndpoints({
        decision: localDecision,
        proxyUrl: "https://example.com/proxy/",
        localServerPath: "ui/server.js",
        localApiEndpoint: "http://localhost:3001/api"
      })
    ).toEqual({ serverPath: "ui/server.js", apiEndpoint: "http://localhost:3001/api" });
  });

  it("builds a desktop config with expected derived fields", () => {
    const decision = decideConnection("external-server", "external");
    const endpoints = resolveEndpoints({
      decision,
      proxyUrl: "https://example.com/proxy/",
      localServerPath: "ui/server.js",
      localApiEndpoint: "http://localhost:3001/api"
    });

    const config = buildDesktopConfig({
      scenarioName: "picker-wheel",
      appDisplayName: "Picker Wheel",
      appDescription: "Test app",
      iconPath: "/tmp/icon.png",
      selectedTemplate: "basic",
      framework: "electron",
      serverType: decision.effectiveServerType,
      serverPort: 3000,
      outputPath: "scenarios/picker-wheel/platforms/electron",
      selectedPlatforms: ["win"],
      deploymentMode: "external-server",
      autoManageTier1: true,
      vrooliBinaryPath: "vrooli",
      proxyUrl: "https://example.com/proxy/",
      bundleManifestPath: "/tmp/bundle.json",
      isBundled: false,
      requiresRemoteConfig: true,
      resolvedEndpoints: endpoints,
      locationMode: "proper",
      includeSigning: true,
      codeSigning: { enabled: true }
    });

    expect(config.app_id).toBe("com.vrooli.picker.wheel");
    expect(config.proxy_url).toBe("https://example.com/proxy/");
    expect(config.external_server_url).toBe("https://example.com/proxy/");
    expect(config.external_api_url).toBeUndefined();
    expect(config.bundle_manifest_path).toBeUndefined();
    expect(config.code_signing?.enabled).toBe(true);
  });

  it("computes standard output and staging preview paths", () => {
    expect(computeStandardOutputPath("demo-scenario"))
      .toBe("scenarios/demo-scenario/platforms/electron");
    expect(computeStagingPreviewPath("demo-scenario"))
      .toBe("scenarios/scenario-to-desktop/data/staging/demo-scenario/<build-id>");
  });
});
