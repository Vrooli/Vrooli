import { describe, it, expect, vi, beforeEach } from "vitest";
import {
  findScenarioByName,
  getScenarioDefaults,
} from "./generatorController";
import type { ScenarioDesktopStatus } from "../components/scenario-inventory/types";

// Mock API module for async tests
vi.mock("../lib/api", () => ({
  fetchScenarioDesktopStatus: vi.fn(),
  fetchProxyHints: vi.fn(),
  fetchBundleManifest: vi.fn(),
  probeEndpoints: vi.fn(),
  runPipeline: vi.fn(),
}));

describe("generatorController", () => {
  describe("findScenarioByName", () => {
    const scenarios: ScenarioDesktopStatus[] = [
      { name: "scenario-a", service_display_name: "Scenario A" } as ScenarioDesktopStatus,
      { name: "scenario-b", service_display_name: "Scenario B" } as ScenarioDesktopStatus,
      { name: "scenario-c", service_display_name: "Scenario C" } as ScenarioDesktopStatus,
    ];

    it("finds scenario by name", () => {
      const result = findScenarioByName(scenarios, "scenario-b");
      expect(result).toBeDefined();
      expect(result?.name).toBe("scenario-b");
      expect(result?.service_display_name).toBe("Scenario B");
    });

    it("returns undefined for unknown name", () => {
      const result = findScenarioByName(scenarios, "unknown-scenario");
      expect(result).toBeUndefined();
    });

    it("returns undefined for empty array", () => {
      const result = findScenarioByName([], "scenario-a");
      expect(result).toBeUndefined();
    });

    it("returns undefined for empty name", () => {
      const result = findScenarioByName(scenarios, "");
      expect(result).toBeUndefined();
    });

    it("handles case-sensitive matching", () => {
      const result = findScenarioByName(scenarios, "Scenario-A");
      expect(result).toBeUndefined();
    });
  });

  describe("getScenarioDefaults", () => {
    it("returns empty values for undefined scenario", () => {
      const defaults = getScenarioDefaults(undefined);
      expect(defaults).toEqual({
        displayName: "",
        description: "",
        iconPath: "",
      });
    });

    it("extracts values from scenario", () => {
      const scenario: ScenarioDesktopStatus = {
        name: "test-scenario",
        service_display_name: "Test App",
        service_description: "Test description",
        service_icon_path: "/icons/test.png",
      } as ScenarioDesktopStatus;

      const defaults = getScenarioDefaults(scenario);
      expect(defaults).toEqual({
        displayName: "Test App",
        description: "Test description",
        iconPath: "/icons/test.png",
      });
    });

    it("handles missing optional fields", () => {
      const scenario: ScenarioDesktopStatus = {
        name: "minimal-scenario",
      } as ScenarioDesktopStatus;

      const defaults = getScenarioDefaults(scenario);
      expect(defaults.displayName).toBe("");
      expect(defaults.description).toBe("");
      expect(defaults.iconPath).toBe("");
    });

    it("handles partial fields", () => {
      const scenario: ScenarioDesktopStatus = {
        name: "partial-scenario",
        service_display_name: "Partial App",
      } as ScenarioDesktopStatus;

      const defaults = getScenarioDefaults(scenario);
      expect(defaults.displayName).toBe("Partial App");
      expect(defaults.description).toBe("");
      expect(defaults.iconPath).toBe("");
    });
  });

  describe("loadGeneratorPageData", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns scenarios from API", async () => {
      const { fetchScenarioDesktopStatus } = await import("../lib/api");
      vi.mocked(fetchScenarioDesktopStatus).mockResolvedValue({
        scenarios: [{ name: "test" } as ScenarioDesktopStatus],
      });

      const { loadGeneratorPageData } = await import("./generatorController");
      const result = await loadGeneratorPageData(null, null);

      expect(result.scenarios).toHaveLength(1);
      expect(result.scenarios[0].name).toBe("test");
      expect(result.error).toBeNull();
    });

    it("handles API error gracefully", async () => {
      const { fetchScenarioDesktopStatus } = await import("../lib/api");
      vi.mocked(fetchScenarioDesktopStatus).mockRejectedValue(new Error("Network error"));

      const { loadGeneratorPageData } = await import("./generatorController");
      const result = await loadGeneratorPageData(null, null);

      expect(result.scenarios).toEqual([]);
      expect(result.error).toBe("Network error");
    });

    it("fetches proxy hints when scenario provided", async () => {
      const { fetchScenarioDesktopStatus, fetchProxyHints } = await import("../lib/api");
      vi.mocked(fetchScenarioDesktopStatus).mockResolvedValue({ scenarios: [] });
      vi.mocked(fetchProxyHints).mockResolvedValue({ proxy_url: "https://api.example.com" });

      const { loadGeneratorPageData } = await import("./generatorController");
      const result = await loadGeneratorPageData("test-scenario", null);

      expect(fetchProxyHints).toHaveBeenCalledWith("test-scenario");
      expect(result.proxyHints).toEqual({ proxy_url: "https://api.example.com" });
    });

    it("fetches bundle manifest when path provided", async () => {
      const { fetchScenarioDesktopStatus, fetchBundleManifest } = await import("../lib/api");
      vi.mocked(fetchScenarioDesktopStatus).mockResolvedValue({ scenarios: [] });
      vi.mocked(fetchBundleManifest).mockResolvedValue({ manifest: {} });

      const { loadGeneratorPageData } = await import("./generatorController");
      const result = await loadGeneratorPageData(null, "/path/to/manifest.json");

      expect(fetchBundleManifest).toHaveBeenCalledWith({ bundle_manifest_path: "/path/to/manifest.json" });
      expect(result.bundleManifest).toBeDefined();
    });

    it("handles proxy hints error gracefully", async () => {
      const { fetchScenarioDesktopStatus, fetchProxyHints } = await import("../lib/api");
      vi.mocked(fetchScenarioDesktopStatus).mockResolvedValue({ scenarios: [] });
      vi.mocked(fetchProxyHints).mockRejectedValue(new Error("Proxy error"));

      const { loadGeneratorPageData } = await import("./generatorController");
      const result = await loadGeneratorPageData("test-scenario", null);

      expect(result.proxyHints).toBeNull();
      expect(result.error).toBeNull(); // Main request should still succeed
    });
  });

  describe("loadScenarios", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns scenarios on success", async () => {
      const { fetchScenarioDesktopStatus } = await import("../lib/api");
      vi.mocked(fetchScenarioDesktopStatus).mockResolvedValue({
        scenarios: [
          { name: "scenario-1" } as ScenarioDesktopStatus,
          { name: "scenario-2" } as ScenarioDesktopStatus,
        ],
      });

      const { loadScenarios } = await import("./generatorController");
      const result = await loadScenarios();

      expect(result.scenarios).toHaveLength(2);
      expect(result.error).toBeNull();
    });

    it("returns error on failure", async () => {
      const { fetchScenarioDesktopStatus } = await import("../lib/api");
      vi.mocked(fetchScenarioDesktopStatus).mockRejectedValue(new Error("Failed to fetch"));

      const { loadScenarios } = await import("./generatorController");
      const result = await loadScenarios();

      expect(result.scenarios).toEqual([]);
      expect(result.error).toBe("Failed to fetch");
    });
  });

  describe("loadProxyHints", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns null hints for empty scenario name", async () => {
      const { loadProxyHints } = await import("./generatorController");
      const result = await loadProxyHints("");

      expect(result.hints).toBeNull();
      expect(result.error).toBeNull();
    });

    it("fetches hints for valid scenario", async () => {
      const { fetchProxyHints } = await import("../lib/api");
      vi.mocked(fetchProxyHints).mockResolvedValue({
        proxy_url: "https://api.example.com",
        healthcheck_path: "/health",
      });

      const { loadProxyHints } = await import("./generatorController");
      const result = await loadProxyHints("my-scenario");

      expect(result.hints?.proxy_url).toBe("https://api.example.com");
      expect(result.error).toBeNull();
    });

    it("handles error gracefully", async () => {
      const { fetchProxyHints } = await import("../lib/api");
      vi.mocked(fetchProxyHints).mockRejectedValue(new Error("Not found"));

      const { loadProxyHints } = await import("./generatorController");
      const result = await loadProxyHints("my-scenario");

      expect(result.hints).toBeNull();
      expect(result.error).toBe("Not found");
    });
  });

  describe("loadBundleManifest", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns null manifest for empty path", async () => {
      const { loadBundleManifest } = await import("./generatorController");
      const result = await loadBundleManifest("");

      expect(result.manifest).toBeNull();
      expect(result.error).toBeNull();
    });

    it("returns null manifest for whitespace-only path", async () => {
      const { loadBundleManifest } = await import("./generatorController");
      const result = await loadBundleManifest("   ");

      expect(result.manifest).toBeNull();
      expect(result.error).toBeNull();
    });

    it("fetches manifest for valid path", async () => {
      const { fetchBundleManifest } = await import("../lib/api");
      vi.mocked(fetchBundleManifest).mockResolvedValue({
        bundle_root: "/app",
        version: "1.0.0",
      });

      const { loadBundleManifest } = await import("./generatorController");
      const result = await loadBundleManifest("/path/to/manifest.json");

      expect(result.manifest).toBeDefined();
      expect(result.error).toBeNull();
    });

    it("trims whitespace from path", async () => {
      const { fetchBundleManifest } = await import("../lib/api");
      vi.mocked(fetchBundleManifest).mockResolvedValue({ bundle_root: "/app" });

      const { loadBundleManifest } = await import("./generatorController");
      await loadBundleManifest("  /path/to/manifest.json  ");

      expect(fetchBundleManifest).toHaveBeenCalledWith({
        bundle_manifest_path: "/path/to/manifest.json",
      });
    });
  });

  describe("testProxyConnection", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns error for empty URL", async () => {
      const { testProxyConnection } = await import("./generatorController");
      const result = await testProxyConnection("");

      expect(result.result).toBeNull();
      expect(result.error).toBe("Enter the proxy URL before testing");
    });

    it("tests connection for valid URL", async () => {
      const { probeEndpoints } = await import("../lib/api");
      vi.mocked(probeEndpoints).mockResolvedValue({
        connected: true,
        version: "1.0.0",
      });

      const { testProxyConnection } = await import("./generatorController");
      const result = await testProxyConnection("https://api.example.com");

      expect(probeEndpoints).toHaveBeenCalledWith({ proxy_url: "https://api.example.com" });
      expect(result.result?.connected).toBe(true);
      expect(result.error).toBeNull();
    });

    it("handles connection error", async () => {
      const { probeEndpoints } = await import("../lib/api");
      vi.mocked(probeEndpoints).mockRejectedValue(new Error("Connection refused"));

      const { testProxyConnection } = await import("./generatorController");
      const result = await testProxyConnection("https://api.example.com");

      expect(result.result).toBeNull();
      expect(result.error).toBe("Connection refused");
    });
  });
});
