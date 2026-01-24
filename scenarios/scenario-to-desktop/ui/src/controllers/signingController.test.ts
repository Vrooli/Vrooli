import { describe, it, expect, vi, beforeEach, afterEach } from "vitest";
import {
  applyDiscoveredCertificate,
  checkForUnsavedChanges,
  createFreshSigningConfig,
  mergeConfigWithServer,
  storeExpiryWarning,
  getStoredExpiryWarning,
  clearStoredExpiryWarning,
} from "./signingController";
import type { SigningConfig, DiscoveredCertificate } from "../lib/api";

// Mock API module
vi.mock("../lib/api", () => ({
  fetchSigningConfig: vi.fn(),
  saveSigningConfig: vi.fn(),
  validateSigningConfig: vi.fn(),
  checkSigningReadiness: vi.fn(),
  fetchSigningPrerequisites: vi.fn(),
  deleteSigningConfig: vi.fn(),
  discoverCertificates: vi.fn(),
  generateLinuxSigningKey: vi.fn(),
}));

describe("signingController", () => {
  describe("applyDiscoveredCertificate", () => {
    it("applies Windows certificate", () => {
      const cert: DiscoveredCertificate = {
        id: "THUMB123",
        name: "My Certificate",
      };
      const currentConfig: SigningConfig = { enabled: false };

      const result = applyDiscoveredCertificate("windows", cert, currentConfig);

      expect(result.enabled).toBe(true);
      expect(result.windows?.certificate_thumbprint).toBe("THUMB123");
    });

    it("applies macOS certificate", () => {
      const cert: DiscoveredCertificate = {
        name: "Developer ID Application: Test (ABC123)",
        subject: "CN=Developer ID",
      };
      const currentConfig: SigningConfig = { enabled: false };

      const result = applyDiscoveredCertificate("macos", cert, currentConfig);

      expect(result.enabled).toBe(true);
      expect(result.macos?.identity).toBe("Developer ID Application: Test (ABC123)");
    });

    it("applies Linux certificate", () => {
      const cert: DiscoveredCertificate = {
        id: "GPG_KEY_123",
        name: "GPG Key",
      };
      const currentConfig: SigningConfig = { enabled: false };

      const result = applyDiscoveredCertificate("linux", cert, currentConfig);

      expect(result.enabled).toBe(true);
      expect(result.linux?.gpg_key_id).toBe("GPG_KEY_123");
    });
  });

  describe("checkForUnsavedChanges", () => {
    it("returns false when configs match", () => {
      const config: SigningConfig = { enabled: true };
      expect(checkForUnsavedChanges(config, config)).toBe(false);
    });

    it("returns true when configs differ", () => {
      const local: SigningConfig = { enabled: true };
      const server: SigningConfig = { enabled: false };
      expect(checkForUnsavedChanges(local, server)).toBe(true);
    });

    it("returns true when server is null and local has values", () => {
      const local: SigningConfig = { enabled: true };
      expect(checkForUnsavedChanges(local, null)).toBe(true);
    });
  });

  describe("createFreshSigningConfig", () => {
    it("returns config with enabled false", () => {
      const config = createFreshSigningConfig();
      expect(config.enabled).toBe(false);
    });

    it("has no platform configs", () => {
      const config = createFreshSigningConfig();
      expect(config.windows).toBeUndefined();
      expect(config.macos).toBeUndefined();
      expect(config.linux).toBeUndefined();
    });
  });

  describe("mergeConfigWithServer", () => {
    it("uses server config as base when available", () => {
      const server: SigningConfig = {
        enabled: true,
        windows: { certificate_source: "store", certificate_thumbprint: "THUMB" },
      };
      const overrides: Partial<SigningConfig> = { enabled: false };

      const result = mergeConfigWithServer(server, overrides);

      expect(result.enabled).toBe(false);
      expect(result.windows?.certificate_thumbprint).toBe("THUMB");
    });

    it("uses defaults when server is null", () => {
      const overrides: Partial<SigningConfig> = { enabled: true };
      const result = mergeConfigWithServer(null, overrides);

      expect(result.enabled).toBe(true);
    });

    it("uses defaults when server is undefined", () => {
      const overrides: Partial<SigningConfig> = {};
      const result = mergeConfigWithServer(undefined, overrides);

      expect(result.enabled).toBe(false);
    });
  });

  describe("loadSigningPageData", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns empty data for empty scenario name", async () => {
      const { loadSigningPageData } = await import("./signingController");
      const result = await loadSigningPageData("");

      expect(result.config).toBeNull();
      expect(result.readiness).toBeNull();
      expect(result.prerequisites).toEqual([]);
      expect(result.error).toBeNull();
    });

    it("loads all data in parallel", async () => {
      const {
        fetchSigningConfig,
        checkSigningReadiness,
        fetchSigningPrerequisites,
      } = await import("../lib/api");

      vi.mocked(fetchSigningConfig).mockResolvedValue({
        config: { enabled: true },
      });
      vi.mocked(checkSigningReadiness).mockResolvedValue({
        ready: true,
        platforms: { windows: { ready: true }, macos: { ready: false }, linux: { ready: false } },
      });
      vi.mocked(fetchSigningPrerequisites).mockResolvedValue({
        tools: [{ name: "signtool", found: true }],
      });

      const { loadSigningPageData } = await import("./signingController");
      const result = await loadSigningPageData("test-scenario");

      expect(result.config?.enabled).toBe(true);
      expect(result.readiness?.ready).toBe(true);
      expect(result.prerequisites).toHaveLength(1);
      expect(result.error).toBeNull();
    });

    it("handles partial failures gracefully", async () => {
      const {
        fetchSigningConfig,
        checkSigningReadiness,
        fetchSigningPrerequisites,
      } = await import("../lib/api");

      vi.mocked(fetchSigningConfig).mockRejectedValue(new Error("Config error"));
      vi.mocked(checkSigningReadiness).mockResolvedValue({
        ready: true,
        platforms: { windows: { ready: true }, macos: { ready: false }, linux: { ready: false } },
      });
      vi.mocked(fetchSigningPrerequisites).mockResolvedValue({
        tools: [],
      });

      const { loadSigningPageData } = await import("./signingController");
      const result = await loadSigningPageData("test-scenario");

      expect(result.config).toBeNull();
      expect(result.readiness?.ready).toBe(true);
      expect(result.error).toBeNull();
    });
  });

  describe("loadSigningConfig", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns null for empty scenario", async () => {
      const { loadSigningConfig } = await import("./signingController");
      const result = await loadSigningConfig("");

      expect(result.config).toBeNull();
      expect(result.error).toBeNull();
    });

    it("fetches config for valid scenario", async () => {
      const { fetchSigningConfig } = await import("../lib/api");
      vi.mocked(fetchSigningConfig).mockResolvedValue({
        config: { enabled: true },
      });

      const { loadSigningConfig } = await import("./signingController");
      const result = await loadSigningConfig("test-scenario");

      expect(result.config?.enabled).toBe(true);
      expect(result.error).toBeNull();
    });

    it("handles error", async () => {
      const { fetchSigningConfig } = await import("../lib/api");
      vi.mocked(fetchSigningConfig).mockRejectedValue(new Error("Not found"));

      const { loadSigningConfig } = await import("./signingController");
      const result = await loadSigningConfig("test-scenario");

      expect(result.config).toBeNull();
      expect(result.error).toBe("Not found");
    });
  });

  describe("saveSigningConfigWithValidation", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns error for empty scenario", async () => {
      const { saveSigningConfigWithValidation } = await import("./signingController");
      const result = await saveSigningConfigWithValidation("", { enabled: true });

      expect(result.success).toBe(false);
      expect(result.error).toBe("No scenario selected");
    });

    it("saves and validates config", async () => {
      const { saveSigningConfig, validateSigningConfig } = await import("../lib/api");
      vi.mocked(saveSigningConfig).mockResolvedValue(undefined);
      vi.mocked(validateSigningConfig).mockResolvedValue({
        valid: true,
        errors: [],
        warnings: [],
      });

      const { saveSigningConfigWithValidation } = await import("./signingController");
      const result = await saveSigningConfigWithValidation("test-scenario", { enabled: true });

      expect(result.success).toBe(true);
      expect(result.validationResult?.valid).toBe(true);
      expect(result.error).toBeNull();
    });

    it("succeeds even if validation fails", async () => {
      const { saveSigningConfig, validateSigningConfig } = await import("../lib/api");
      vi.mocked(saveSigningConfig).mockResolvedValue(undefined);
      vi.mocked(validateSigningConfig).mockRejectedValue(new Error("Validation error"));

      const { saveSigningConfigWithValidation } = await import("./signingController");
      const result = await saveSigningConfigWithValidation("test-scenario", { enabled: true });

      expect(result.success).toBe(true);
      expect(result.validationResult).toBeUndefined();
      expect(result.error).toBeNull();
    });

    it("returns error on save failure", async () => {
      const { saveSigningConfig } = await import("../lib/api");
      vi.mocked(saveSigningConfig).mockRejectedValue(new Error("Save failed"));

      const { saveSigningConfigWithValidation } = await import("./signingController");
      const result = await saveSigningConfigWithValidation("test-scenario", { enabled: true });

      expect(result.success).toBe(false);
      expect(result.error).toBe("Save failed");
    });
  });

  describe("deleteSigningConfigForScenario", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns error for empty scenario", async () => {
      const { deleteSigningConfigForScenario } = await import("./signingController");
      const result = await deleteSigningConfigForScenario("");

      expect(result.success).toBe(false);
      expect(result.error).toBe("No scenario selected");
    });

    it("deletes config successfully", async () => {
      const { deleteSigningConfig } = await import("../lib/api");
      vi.mocked(deleteSigningConfig).mockResolvedValue(undefined);

      const { deleteSigningConfigForScenario } = await import("./signingController");
      const result = await deleteSigningConfigForScenario("test-scenario");

      expect(result.success).toBe(true);
      expect(result.error).toBeNull();
    });

    it("handles delete error", async () => {
      const { deleteSigningConfig } = await import("../lib/api");
      vi.mocked(deleteSigningConfig).mockRejectedValue(new Error("Delete failed"));

      const { deleteSigningConfigForScenario } = await import("./signingController");
      const result = await deleteSigningConfigForScenario("test-scenario");

      expect(result.success).toBe(false);
      expect(result.error).toBe("Delete failed");
    });
  });

  describe("discoverAndFilterCertificates", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("discovers certificates and detects warnings", async () => {
      const { discoverCertificates } = await import("../lib/api");
      vi.mocked(discoverCertificates).mockResolvedValue({
        certificates: [
          { name: "Valid Cert", is_expired: false, days_to_expiry: 100 },
          { name: "Expiring Cert", is_expired: false, days_to_expiry: 15 },
        ],
      });

      const { discoverAndFilterCertificates } = await import("./signingController");
      const result = await discoverAndFilterCertificates("windows");

      expect(result.certificates).toHaveLength(2);
      expect(result.warnings).toHaveLength(1);
      expect(result.warnings[0].daysToExpiry).toBe(15);
      expect(result.error).toBeNull();
    });

    it("handles discovery error", async () => {
      const { discoverCertificates } = await import("../lib/api");
      vi.mocked(discoverCertificates).mockRejectedValue(new Error("Discovery failed"));

      const { discoverAndFilterCertificates } = await import("./signingController");
      const result = await discoverAndFilterCertificates("windows");

      expect(result.certificates).toEqual([]);
      expect(result.warnings).toEqual([]);
      expect(result.error).toBe("Discovery failed");
    });
  });

  describe("generateLinuxKey", () => {
    beforeEach(() => {
      vi.clearAllMocks();
    });

    it("returns error for empty scenario", async () => {
      const { generateLinuxKey } = await import("./signingController");
      const result = await generateLinuxKey("", { name: "Test" });

      expect(result.fingerprint).toBeNull();
      expect(result.error).toBe("No scenario selected");
    });

    it("generates key successfully", async () => {
      const { generateLinuxSigningKey } = await import("../lib/api");
      vi.mocked(generateLinuxSigningKey).mockResolvedValue({
        fingerprint: "ABC123",
        homedir: "/home/user/.gnupg",
      });

      const { generateLinuxKey } = await import("./signingController");
      const result = await generateLinuxKey("test-scenario", { name: "Test Key" });

      expect(result.fingerprint).toBe("ABC123");
      expect(result.homedir).toBe("/home/user/.gnupg");
      expect(result.error).toBeNull();
    });

    it("handles generation error", async () => {
      const { generateLinuxSigningKey } = await import("../lib/api");
      vi.mocked(generateLinuxSigningKey).mockRejectedValue(new Error("Generation failed"));

      const { generateLinuxKey } = await import("./signingController");
      const result = await generateLinuxKey("test-scenario", { name: "Test" });

      expect(result.fingerprint).toBeNull();
      expect(result.error).toBe("Generation failed");
    });
  });

  describe("expiry warning storage", () => {
    const originalWindow = global.window;

    beforeEach(() => {
      // Mock localStorage
      const storage: Record<string, string> = {};
      global.window = {
        localStorage: {
          getItem: vi.fn((key: string) => storage[key] ?? null),
          setItem: vi.fn((key: string, value: string) => {
            storage[key] = value;
          }),
          removeItem: vi.fn((key: string) => {
            delete storage[key];
          }),
        },
      } as unknown as Window & typeof globalThis;
    });

    afterEach(() => {
      global.window = originalWindow;
    });

    it("stores expiry warning", () => {
      storeExpiryWarning("Certificate expires in 10 days");
      expect(window.localStorage.setItem).toHaveBeenCalledWith(
        "std_signing_expiry_warning",
        "Certificate expires in 10 days"
      );
    });

    it("retrieves stored warning", () => {
      (window.localStorage.getItem as ReturnType<typeof vi.fn>).mockReturnValue("Warning message");
      const warning = getStoredExpiryWarning();
      expect(warning).toBe("Warning message");
    });

    it("clears stored warning", () => {
      clearStoredExpiryWarning();
      expect(window.localStorage.removeItem).toHaveBeenCalledWith("std_signing_expiry_warning");
    });
  });
});
