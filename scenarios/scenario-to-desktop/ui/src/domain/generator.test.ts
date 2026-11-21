import { describe, expect, it } from "vitest";
import {
  buildDesktopConfig,
  computeStandardOutputPath,
  computeStagingPreviewPath,
  getSelectedPlatforms,
  resolveEndpoints,
  validateFormInputs,
} from "./generator";
import { decideConnection } from "./deployment";

describe("generator domain", () => {
  it("returns selected platforms from the selection map", () => {
    const platforms = getSelectedPlatforms({
      win: true,
      mac: false,
      linux: true,
    });

    expect(platforms).toEqual(["win", "linux"]);
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
        localApiEndpoint: "http://localhost:3001/api",
      }),
    ).toEqual({
      serverPath: "http://127.0.0.1",
      apiEndpoint: "http://127.0.0.1",
    });

    expect(
      resolveEndpoints({
        decision: remoteDecision,
        proxyUrl: "https://example.com/proxy/",
        localServerPath: "ui/server.js",
        localApiEndpoint: "http://localhost:3001/api",
      }),
    ).toEqual({
      serverPath: "https://example.com/proxy/",
      apiEndpoint: "https://example.com/proxy/",
    });

    expect(
      resolveEndpoints({
        decision: localDecision,
        proxyUrl: "https://example.com/proxy/",
        localServerPath: "ui/server.js",
        localApiEndpoint: "http://localhost:3001/api",
      }),
    ).toEqual({
      serverPath: "ui/server.js",
      apiEndpoint: "http://localhost:3001/api",
    });
  });

  it("builds a desktop config with expected derived fields", () => {
    const decision = decideConnection("external-server", "external");
    const endpoints = resolveEndpoints({
      decision,
      proxyUrl: "https://example.com/proxy/",
      localServerPath: "ui/server.js",
      localApiEndpoint: "http://localhost:3001/api",
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
      codeSigning: { enabled: true },
    });

    expect(config.app_id).toBe("com.vrooli.picker.wheel");
    expect(config.proxy_url).toBe("https://example.com/proxy/");
    expect(config.external_server_url).toBe("https://example.com/proxy/");
    expect(config.external_api_url).toBeUndefined();
    expect(config.bundle_manifest_path).toBeUndefined();
    expect(config.code_signing?.enabled).toBe(true);
  });

  it("builds bundled config fields and validates required remote and signing inputs", () => {
    const bundled = buildDesktopConfig({
      scenarioName: "picker-wheel",
      appDisplayName: "Picker Wheel",
      appDescription: "Test app",
      iconPath: "",
      selectedTemplate: "basic",
      framework: "electron",
      serverType: "static",
      serverPort: 3000,
      outputPath: "scenarios/picker-wheel/platforms/electron",
      selectedPlatforms: ["linux"],
      deploymentMode: "bundled",
      autoManageTier1: false,
      vrooliBinaryPath: "vrooli",
      proxyUrl: "",
      bundleManifestPath: "/tmp/bundle.json",
      isBundled: true,
      requiresRemoteConfig: false,
      resolvedEndpoints: {
        serverPath: "http://127.0.0.1",
        apiEndpoint: "http://127.0.0.1",
      },
      locationMode: "proper",
      includeSigning: false,
    });

    expect(bundled.icon).toBeUndefined();
    expect(bundled.bundle_manifest_path).toBe("/tmp/bundle.json");
    expect(bundled.external_api_url).toBeUndefined();
    expect(bundled.code_signing).toEqual({ enabled: false });

    const errors = validateFormInputs({
      scenarioName: "",
      selectedPlatforms: [],
      isBundled: false,
      requiresProxyUrl: true,
      bundleManifestPath: "",
      proxyUrl: " ",
      appDisplayName: " ",
      appDescription: " ",
      locationMode: "custom",
      outputPath: " ",
      preflightResult: null,
      preflightOk: false,
      preflightOverride: false,
      signingEnabledForBuild: true,
      signingConfig: null,
      signingReadiness: undefined,
    });

    expect(errors.map(({ id }) => id)).toEqual([
      "no-scenario",
      "no-platforms",
      "no-proxy-url",
      "no-display-name",
      "no-description",
      "no-output-path",
      "no-signing-config",
    ]);
  });

  it("reports failed preflight and signing readiness issues", () => {
    const errors = validateFormInputs({
      scenarioName: "picker-wheel",
      selectedPlatforms: ["linux"],
      isBundled: true,
      requiresProxyUrl: false,
      bundleManifestPath: "/tmp/bundle.json",
      proxyUrl: "",
      appDisplayName: "Picker Wheel",
      appDescription: "Test app",
      locationMode: "proper",
      outputPath: "",
      preflightResult: { ready: { ready: false } },
      preflightOk: false,
      preflightOverride: false,
      signingEnabledForBuild: true,
      signingConfig: { enabled: true },
      signingReadiness: { ready: false, issues: ["missing certificate"] },
    });

    expect(errors.map(({ id }) => id)).toEqual([
      "preflight-failed",
      "signing-not-ready",
    ]);
  });

  it("computes standard output and staging preview paths", () => {
    expect(computeStandardOutputPath("demo-scenario")).toBe(
      "scenarios/demo-scenario/platforms/electron",
    );
    expect(computeStagingPreviewPath("demo-scenario")).toBe(
      "<cache-root>/vrooli/scenario-to-desktop/staging/demo-scenario/<build-id>",
    );
  });
});
