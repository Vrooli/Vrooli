import { describe, it, expect } from "vitest";
import {
  applyCertificateToConfig,
  detectExpiryWarnings,
  hasImminentExpiry,
  getSoonestExpiring,
  hasUnsavedConfigChanges,
  hasAnyPlatformConfig,
  isPlatformReady,
  getPlatformNotReadyReason,
  countReadyPlatforms,
  getValidationErrorCount,
  getValidationWarningCount,
  isValidationPassed,
  getDefaultSigningConfig,
  getDefaultWindowsConfig,
  getDefaultMacOSConfig,
  getDefaultLinuxConfig,
  filterCertificatesByPlatform,
  filterCodeSigningCertificates,
  filterNonExpiredCertificates,
  buildExpiryWarningMessage,
  EXPIRY_WARNING_THRESHOLD_DAYS,
  EXPIRY_CRITICAL_THRESHOLD_DAYS,
  PLATFORM_ORDER,
  PLATFORM_DISPLAY_NAMES,
} from "./signing.service";
import type {
  SigningConfig,
  DiscoveredCertificate,
  SigningReadinessResponse,
  SigningValidationResult,
} from "../domain/signing";

describe("signing.service", () => {
  describe("constants", () => {
    it("defines expiry thresholds", () => {
      expect(EXPIRY_WARNING_THRESHOLD_DAYS).toBe(30);
      expect(EXPIRY_CRITICAL_THRESHOLD_DAYS).toBe(7);
    });

    it("defines platform order", () => {
      expect(PLATFORM_ORDER).toContain("windows");
      expect(PLATFORM_ORDER).toContain("macos");
      expect(PLATFORM_ORDER).toContain("linux");
    });

    it("defines platform display names", () => {
      expect(PLATFORM_DISPLAY_NAMES.windows).toBe("Windows");
      expect(PLATFORM_DISPLAY_NAMES.macos).toBe("macOS");
      expect(PLATFORM_DISPLAY_NAMES.linux).toBe("Linux");
    });
  });

  describe("applyCertificateToConfig", () => {
    const baseConfig: SigningConfig = { enabled: false };

    it("applies Windows certificate", () => {
      const cert: DiscoveredCertificate = {
        id: "THUMBPRINT123",
        name: "Test Certificate",
        subject: "CN=Test",
      };
      const result = applyCertificateToConfig("windows", cert, baseConfig);

      expect(result.enabled).toBe(true);
      expect(result.windows?.certificate_source).toBe("store");
      expect(result.windows?.certificate_thumbprint).toBe("THUMBPRINT123");
      expect(result.windows?.timestamp_server).toBe(
        "http://timestamp.digicert.com",
      );
    });

    it("applies macOS certificate", () => {
      const cert: DiscoveredCertificate = {
        name: "Developer ID Application: Test (ABC123)",
        subject: "CN=Developer ID Application: Test",
      };
      const result = applyCertificateToConfig("macos", cert, baseConfig);

      expect(result.enabled).toBe(true);
      expect(result.macos?.identity).toBe(
        "Developer ID Application: Test (ABC123)",
      );
      expect(result.macos?.hardened_runtime).toBe(true);
    });

    it("applies Linux certificate", () => {
      const cert: DiscoveredCertificate = {
        id: "GPG_KEY_ID",
        name: "Test GPG Key",
      };
      const result = applyCertificateToConfig("linux", cert, baseConfig);

      expect(result.enabled).toBe(true);
      expect(result.linux?.gpg_key_id).toBe("GPG_KEY_ID");
    });

    it("preserves existing config values", () => {
      const existingConfig: SigningConfig = {
        enabled: true,
        windows: {
          certificate_source: "file",
          timestamp_server: "http://custom.timestamp.com",
          sign_algorithm: "sha384",
          dual_sign: true,
        },
      };
      const cert: DiscoveredCertificate = { id: "NEW_THUMB", name: "New Cert" };
      const result = applyCertificateToConfig("windows", cert, existingConfig);

      expect(result.windows?.timestamp_server).toBe(
        "http://custom.timestamp.com",
      );
      expect(result.windows?.sign_algorithm).toBe("sha384");
      expect(result.windows?.dual_sign).toBe(true);
    });
  });

  describe("detectExpiryWarnings", () => {
    it("returns empty array for no certificates", () => {
      expect(detectExpiryWarnings([])).toEqual([]);
    });

    it("detects expired certificates", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Expired Cert", is_expired: true, days_to_expiry: -5 },
      ];
      const warnings = detectExpiryWarnings(certs);
      expect(warnings).toHaveLength(1);
      expect(warnings?.[0]?.isExpired).toBe(true);
    });

    it("detects certificates expiring soon", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Expiring Soon", is_expired: false, days_to_expiry: 15 },
      ];
      const warnings = detectExpiryWarnings(certs);
      expect(warnings).toHaveLength(1);
      expect(warnings?.[0]?.daysToExpiry).toBe(15);
    });

    it("ignores certificates with plenty of time", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Good Cert", is_expired: false, days_to_expiry: 365 },
      ];
      const warnings = detectExpiryWarnings(certs);
      expect(warnings).toHaveLength(0);
    });

    it("uses custom threshold", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Cert", is_expired: false, days_to_expiry: 50 },
      ];
      const warnings = detectExpiryWarnings(certs, 60);
      expect(warnings).toHaveLength(1);
    });

    it("sorts by days to expiry ascending", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Cert A", is_expired: false, days_to_expiry: 20 },
        { name: "Cert B", is_expired: false, days_to_expiry: 5 },
        { name: "Cert C", is_expired: false, days_to_expiry: 10 },
      ];
      const warnings = detectExpiryWarnings(certs);
      expect(warnings?.[0]?.daysToExpiry).toBe(5);
      expect(warnings?.[1]?.daysToExpiry).toBe(10);
      expect(warnings?.[2]?.daysToExpiry).toBe(20);
    });
  });

  describe("hasImminentExpiry", () => {
    it("returns false for empty array", () => {
      expect(hasImminentExpiry([])).toBe(false);
    });

    it("returns true for certificate within critical threshold", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Critical", is_expired: false, days_to_expiry: 3 },
      ];
      expect(hasImminentExpiry(certs)).toBe(true);
    });

    it("returns false for certificate beyond critical threshold", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "OK", is_expired: false, days_to_expiry: 15 },
      ];
      expect(hasImminentExpiry(certs)).toBe(false);
    });

    it("returns false for already expired certificates", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Expired", is_expired: true, days_to_expiry: 0 },
      ];
      expect(hasImminentExpiry(certs)).toBe(false);
    });
  });

  describe("getSoonestExpiring", () => {
    it("returns null for empty array", () => {
      expect(getSoonestExpiring([])).toBe(null);
    });

    it("returns null when all expired", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Expired", is_expired: true },
      ];
      expect(getSoonestExpiring(certs)).toBe(null);
    });

    it("returns certificate with soonest expiry", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Later", is_expired: false, days_to_expiry: 100 },
        { name: "Soonest", is_expired: false, days_to_expiry: 10 },
        { name: "Middle", is_expired: false, days_to_expiry: 50 },
      ];
      const soonest = getSoonestExpiring(certs);
      expect(soonest?.name).toBe("Soonest");
    });

    it("ignores expired certificates", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Expired", is_expired: true, days_to_expiry: -5 },
        { name: "Valid", is_expired: false, days_to_expiry: 30 },
      ];
      const soonest = getSoonestExpiring(certs);
      expect(soonest?.name).toBe("Valid");
    });
  });

  describe("hasUnsavedConfigChanges", () => {
    it("returns false when local matches server", () => {
      const config: SigningConfig = { enabled: true };
      expect(hasUnsavedConfigChanges(config, config)).toBe(false);
    });

    it("returns true when local differs from server", () => {
      const local: SigningConfig = { enabled: true };
      const server: SigningConfig = { enabled: false };
      expect(hasUnsavedConfigChanges(local, server)).toBe(true);
    });

    it("returns true when no server config and local has values", () => {
      const local: SigningConfig = { enabled: true };
      expect(hasUnsavedConfigChanges(local, null)).toBe(true);
    });

    it("returns false when no server config and local is empty", () => {
      const local: SigningConfig = { enabled: false };
      expect(hasUnsavedConfigChanges(local, null)).toBe(false);
    });

    it("returns true when local has platform config and no server", () => {
      const local: SigningConfig = {
        enabled: false,
        windows: { certificate_source: "file" },
      };
      expect(hasUnsavedConfigChanges(local, null)).toBe(true);
    });
  });

  describe("hasAnyPlatformConfig", () => {
    it("returns false for empty config", () => {
      expect(hasAnyPlatformConfig({ enabled: false })).toBe(false);
    });

    it("returns true for Windows config", () => {
      expect(
        hasAnyPlatformConfig({
          enabled: false,
          windows: { certificate_source: "file" },
        }),
      ).toBe(true);
    });

    it("returns true for macOS config", () => {
      expect(
        hasAnyPlatformConfig({
          enabled: false,
          macos: {
            identity: "test",
            team_id: "",
            hardened_runtime: false,
            notarize: false,
          },
        }),
      ).toBe(true);
    });

    it("returns true for Linux config", () => {
      expect(
        hasAnyPlatformConfig({ enabled: false, linux: { gpg_key_id: "test" } }),
      ).toBe(true);
    });
  });

  describe("isPlatformReady", () => {
    it("returns false for null readiness", () => {
      expect(isPlatformReady(null, "windows")).toBe(false);
    });

    it("returns false for undefined readiness", () => {
      expect(isPlatformReady(undefined, "windows")).toBe(false);
    });

    it("returns true when platform is ready", () => {
      const readiness: SigningReadinessResponse = {
        ready: true,
        platforms: {
          windows: { ready: true },
          macos: { ready: false },
          linux: { ready: false },
        },
      };
      expect(isPlatformReady(readiness, "windows")).toBe(true);
    });

    it("returns false when platform is not ready", () => {
      const readiness: SigningReadinessResponse = {
        ready: false,
        platforms: {
          windows: { ready: false },
          macos: { ready: false },
          linux: { ready: false },
        },
      };
      expect(isPlatformReady(readiness, "windows")).toBe(false);
    });
  });

  describe("getPlatformNotReadyReason", () => {
    it("returns null for null readiness", () => {
      expect(getPlatformNotReadyReason(null, "windows")).toBe(null);
    });

    it("returns null when platform is ready", () => {
      const readiness: SigningReadinessResponse = {
        ready: true,
        platforms: {
          windows: { ready: true },
          macos: { ready: false },
          linux: { ready: false },
        },
      };
      expect(getPlatformNotReadyReason(readiness, "windows")).toBe(null);
    });

    it("returns reason when platform not ready", () => {
      const readiness: SigningReadinessResponse = {
        ready: false,
        platforms: {
          windows: { ready: false, reason: "Certificate not found" },
          macos: { ready: false },
          linux: { ready: false },
        },
      };
      expect(getPlatformNotReadyReason(readiness, "windows")).toBe(
        "Certificate not found",
      );
    });

    it("returns default reason when no reason provided", () => {
      const readiness: SigningReadinessResponse = {
        ready: false,
        platforms: {
          windows: { ready: false },
          macos: { ready: false },
          linux: { ready: false },
        },
      };
      expect(getPlatformNotReadyReason(readiness, "windows")).toBe(
        "Not configured",
      );
    });
  });

  describe("countReadyPlatforms", () => {
    it("returns 0 for null readiness", () => {
      expect(countReadyPlatforms(null)).toBe(0);
    });

    it("counts ready platforms", () => {
      const readiness: SigningReadinessResponse = {
        ready: true,
        platforms: {
          windows: { ready: true },
          macos: { ready: true },
          linux: { ready: false },
        },
      };
      expect(countReadyPlatforms(readiness)).toBe(2);
    });

    it("returns 0 when none ready", () => {
      const readiness: SigningReadinessResponse = {
        ready: false,
        platforms: {
          windows: { ready: false },
          macos: { ready: false },
          linux: { ready: false },
        },
      };
      expect(countReadyPlatforms(readiness)).toBe(0);
    });
  });

  describe("getValidationErrorCount", () => {
    it("returns 0 for null result", () => {
      expect(getValidationErrorCount(null)).toBe(0);
    });

    it("returns error count", () => {
      const result: SigningValidationResult = {
        valid: false,
        errors: [
          { code: "E001", message: "Error 1" },
          { code: "E002", message: "Error 2" },
        ],
        warnings: [],
      };
      expect(getValidationErrorCount(result)).toBe(2);
    });
  });

  describe("getValidationWarningCount", () => {
    it("returns 0 for null result", () => {
      expect(getValidationWarningCount(null)).toBe(0);
    });

    it("returns warning count", () => {
      const result: SigningValidationResult = {
        valid: true,
        errors: [],
        warnings: [{ code: "W001", message: "Warning 1" }],
      };
      expect(getValidationWarningCount(result)).toBe(1);
    });
  });

  describe("isValidationPassed", () => {
    it("returns false for null result", () => {
      expect(isValidationPassed(null)).toBe(false);
    });

    it("returns validation valid flag", () => {
      expect(
        isValidationPassed({ valid: true, errors: [], warnings: [] }),
      ).toBe(true);
      expect(
        isValidationPassed({
          valid: false,
          errors: [{ code: "E001", message: "Error" }],
          warnings: [],
        }),
      ).toBe(false);
    });
  });

  describe("getDefaultSigningConfig", () => {
    it("returns config with enabled false", () => {
      const config = getDefaultSigningConfig();
      expect(config.enabled).toBe(false);
    });
  });

  describe("getDefaultWindowsConfig", () => {
    it("returns Windows defaults", () => {
      const config = getDefaultWindowsConfig();
      expect(config.certificate_source).toBe("file");
      expect(config.timestamp_server).toBe("http://timestamp.digicert.com");
      expect(config.sign_algorithm).toBe("sha256");
    });
  });

  describe("getDefaultMacOSConfig", () => {
    it("returns macOS defaults", () => {
      const config = getDefaultMacOSConfig();
      expect(config.identity).toBe("");
      expect(config.team_id).toBe("");
      expect(config.hardened_runtime).toBe(true);
      expect(config.notarize).toBe(false);
      expect(config.gatekeeper_assess).toBe(true);
    });
  });

  describe("getDefaultLinuxConfig", () => {
    it("returns Linux defaults", () => {
      const config = getDefaultLinuxConfig();
      expect(config.gpg_key_id).toBe("");
    });
  });

  describe("filterCertificatesByPlatform", () => {
    const certs: DiscoveredCertificate[] = [
      { name: "Win Cert", platform: "windows" },
      { name: "Mac Cert", platform: "macos" },
      { name: "Linux Cert", platform: "linux" },
    ];

    it("filters by Windows platform", () => {
      const filtered = filterCertificatesByPlatform(certs, "windows");
      expect(filtered).toHaveLength(1);
      expect(filtered?.[0]?.name).toBe("Win Cert");
    });

    it("filters by macOS platform", () => {
      const filtered = filterCertificatesByPlatform(certs, "macos");
      expect(filtered).toHaveLength(1);
      expect(filtered?.[0]?.name).toBe("Mac Cert");
    });

    it("returns empty array when no match", () => {
      const filtered = filterCertificatesByPlatform(
        [{ name: "Win Cert", platform: "windows" }],
        "linux",
      );
      expect(filtered).toHaveLength(0);
    });
  });

  describe("filterCodeSigningCertificates", () => {
    it("includes certificates with is_code_sign true", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Code Sign", is_code_sign: true },
      ];
      expect(filterCodeSigningCertificates(certs)).toHaveLength(1);
    });

    it("includes certificates with is_code_sign undefined", () => {
      const certs: DiscoveredCertificate[] = [{ name: "Unknown" }];
      expect(filterCodeSigningCertificates(certs)).toHaveLength(1);
    });

    it("excludes certificates with is_code_sign false", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Not Code Sign", is_code_sign: false },
      ];
      expect(filterCodeSigningCertificates(certs)).toHaveLength(0);
    });
  });

  describe("filterNonExpiredCertificates", () => {
    it("includes non-expired certificates", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Valid", is_expired: false },
        { name: "Unknown" },
      ];
      expect(filterNonExpiredCertificates(certs)).toHaveLength(2);
    });

    it("excludes expired certificates", () => {
      const certs: DiscoveredCertificate[] = [
        { name: "Expired", is_expired: true },
      ];
      expect(filterNonExpiredCertificates(certs)).toHaveLength(0);
    });
  });

  describe("buildExpiryWarningMessage", () => {
    it("returns null for certificate with plenty of time", () => {
      const cert: DiscoveredCertificate = {
        name: "Valid",
        is_expired: false,
        days_to_expiry: 100,
      };
      expect(buildExpiryWarningMessage(cert)).toBe(null);
    });

    it("returns expiration message for expired certificate", () => {
      const cert: DiscoveredCertificate = {
        name: "Expired",
        is_expired: true,
        expires_at: "2024-01-01",
      };
      const message = buildExpiryWarningMessage(cert);
      expect(message).toContain("expired");
      expect(message).toContain("2024-01-01");
    });

    it("returns warning message for certificate expiring soon", () => {
      const cert: DiscoveredCertificate = {
        name: "Expiring",
        is_expired: false,
        days_to_expiry: 15,
        expires_at: "2024-02-15",
      };
      const message = buildExpiryWarningMessage(cert);
      expect(message).toContain("15 days");
      expect(message).toContain("2024-02-15");
    });

    it("handles unknown expiration date", () => {
      const cert: DiscoveredCertificate = {
        name: "Expired",
        is_expired: true,
      };
      const message = buildExpiryWarningMessage(cert);
      expect(message).toContain("unknown date");
    });
  });
});
