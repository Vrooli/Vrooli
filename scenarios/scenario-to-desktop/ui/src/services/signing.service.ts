/**
 * Signing service - pure functions for code signing configuration.
 * Extracted from SigningPage.tsx for testability and reuse.
 */

import type {
  SigningConfig,
  SigningReadinessResponse,
  SigningValidationResult,
  DiscoveredCertificate,
  WindowsSigningConfig,
  MacOSSigningConfig,
  LinuxSigningConfig,
} from "../lib/api";

// ============================================================================
// Types
// ============================================================================

export type SigningPlatform = "windows" | "macos" | "linux";

export interface ExpiryWarning {
  certificate: DiscoveredCertificate;
  daysToExpiry: number;
  isExpired: boolean;
}

export interface SigningPageState {
  selectedScenario: string;
  localConfig: SigningConfig;
  hasUnsavedChanges: boolean;
  discoverPlatform: SigningPlatform;
  discovered: DiscoveredCertificate[];
  keygenMessage?: string;
}

// ============================================================================
// Constants
// ============================================================================

/** Warning threshold for certificate expiration (days) */
export const EXPIRY_WARNING_THRESHOLD_DAYS = 30;

/** Critical threshold for certificate expiration (days) */
export const EXPIRY_CRITICAL_THRESHOLD_DAYS = 7;

/** Platform display order */
export const PLATFORM_ORDER: SigningPlatform[] = ["windows", "macos", "linux"];

/** Platform display names */
export const PLATFORM_DISPLAY_NAMES: Record<SigningPlatform, string> = {
  windows: "Windows",
  macos: "macOS",
  linux: "Linux",
};

// ============================================================================
// Certificate Application
// ============================================================================

/**
 * Apply a discovered certificate to the signing config.
 */
export function applyCertificateToConfig(
  platform: SigningPlatform,
  cert: DiscoveredCertificate,
  existingConfig: SigningConfig
): SigningConfig {
  const next: SigningConfig = { ...existingConfig, enabled: true };

  if (platform === "windows") {
    next.windows = {
      certificate_source: "store",
      certificate_thumbprint: cert.id || cert.name || "",
      timestamp_server: existingConfig.windows?.timestamp_server || "http://timestamp.digicert.com",
      sign_algorithm: existingConfig.windows?.sign_algorithm || "sha256",
      dual_sign: existingConfig.windows?.dual_sign,
    };
  } else if (platform === "macos") {
    next.macos = {
      identity: cert.name || cert.subject || "",
      team_id: existingConfig.macos?.team_id || "",
      hardened_runtime: existingConfig.macos?.hardened_runtime ?? true,
      notarize: existingConfig.macos?.notarize ?? false,
      gatekeeper_assess: existingConfig.macos?.gatekeeper_assess ?? true,
      entitlements_file: existingConfig.macos?.entitlements_file,
      apple_api_key_id: existingConfig.macos?.apple_api_key_id,
      apple_api_issuer_id: existingConfig.macos?.apple_api_issuer_id,
      apple_api_key_file: existingConfig.macos?.apple_api_key_file,
      apple_id_env: existingConfig.macos?.apple_id_env,
      apple_id_password_env: existingConfig.macos?.apple_id_password_env,
    };
  } else if (platform === "linux") {
    next.linux = {
      gpg_key_id: cert.id || cert.name || "",
      keyring_path: existingConfig.linux?.keyring_path,
      deb_keyring_path: existingConfig.linux?.deb_keyring_path,
      rpm_keyring_path: existingConfig.linux?.rpm_keyring_path,
    };
  }

  return next;
}

// ============================================================================
// Expiry Detection
// ============================================================================

/**
 * Detect certificates that are expiring soon or already expired.
 */
export function detectExpiryWarnings(
  certificates: DiscoveredCertificate[],
  thresholdDays: number = EXPIRY_WARNING_THRESHOLD_DAYS
): ExpiryWarning[] {
  return certificates
    .filter((cert) => {
      if (cert.is_expired) return true;
      if (typeof cert.days_to_expiry === "number" && cert.days_to_expiry <= thresholdDays) {
        return true;
      }
      return false;
    })
    .map((cert) => ({
      certificate: cert,
      daysToExpiry: cert.days_to_expiry ?? 0,
      isExpired: cert.is_expired ?? false,
    }))
    .sort((a, b) => a.daysToExpiry - b.daysToExpiry);
}

/**
 * Check if any certificates have imminent expiry (within critical threshold).
 */
export function hasImminentExpiry(certificates: DiscoveredCertificate[]): boolean {
  return certificates.some(
    (cert) =>
      !cert.is_expired &&
      typeof cert.days_to_expiry === "number" &&
      cert.days_to_expiry <= EXPIRY_CRITICAL_THRESHOLD_DAYS
  );
}

/**
 * Get the soonest expiring certificate.
 */
export function getSoonestExpiring(
  certificates: DiscoveredCertificate[]
): DiscoveredCertificate | null {
  const nonExpired = certificates.filter(
    (cert) => !cert.is_expired && typeof cert.days_to_expiry === "number"
  );
  if (nonExpired.length === 0) return null;

  return nonExpired.reduce((soonest, cert) =>
    (cert.days_to_expiry ?? Infinity) < (soonest.days_to_expiry ?? Infinity) ? cert : soonest
  );
}

// ============================================================================
// Config Comparison
// ============================================================================

/**
 * Check if local config has unsaved changes compared to server config.
 */
export function hasUnsavedConfigChanges(
  localConfig: SigningConfig,
  serverConfig: SigningConfig | null | undefined
): boolean {
  if (!serverConfig) {
    // If no server config, any local config with values is unsaved
    return localConfig.enabled || hasAnyPlatformConfig(localConfig);
  }

  return JSON.stringify(localConfig) !== JSON.stringify(serverConfig);
}

/**
 * Check if any platform has configuration.
 */
export function hasAnyPlatformConfig(config: SigningConfig): boolean {
  return Boolean(config.windows || config.macos || config.linux);
}

// ============================================================================
// Readiness Helpers
// ============================================================================

/**
 * Check if a specific platform is ready for signing.
 */
export function isPlatformReady(
  readiness: SigningReadinessResponse | null | undefined,
  platform: SigningPlatform
): boolean {
  if (!readiness) return false;
  return readiness.platforms[platform]?.ready ?? false;
}

/**
 * Get the reason why a platform is not ready.
 */
export function getPlatformNotReadyReason(
  readiness: SigningReadinessResponse | null | undefined,
  platform: SigningPlatform
): string | null {
  if (!readiness) return null;
  const status = readiness.platforms[platform];
  if (!status) return null;
  if (status.ready) return null;
  return status.reason || "Not configured";
}

/**
 * Count how many platforms are ready.
 */
export function countReadyPlatforms(
  readiness: SigningReadinessResponse | null | undefined
): number {
  if (!readiness) return 0;
  return PLATFORM_ORDER.filter((platform) => readiness.platforms[platform]?.ready).length;
}

// ============================================================================
// Validation Helpers
// ============================================================================

/**
 * Get validation error count.
 */
export function getValidationErrorCount(result: SigningValidationResult | null): number {
  if (!result) return 0;
  return result.errors.length;
}

/**
 * Get validation warning count.
 */
export function getValidationWarningCount(result: SigningValidationResult | null): number {
  if (!result) return 0;
  return result.warnings.length;
}

/**
 * Check if validation passed.
 */
export function isValidationPassed(result: SigningValidationResult | null): boolean {
  if (!result) return false;
  return result.valid;
}

// ============================================================================
// Default Configs
// ============================================================================

/**
 * Get default signing config.
 */
export function getDefaultSigningConfig(): SigningConfig {
  return {
    enabled: false,
  };
}

/**
 * Get default Windows signing config.
 */
export function getDefaultWindowsConfig(): WindowsSigningConfig {
  return {
    certificate_source: "file",
    timestamp_server: "http://timestamp.digicert.com",
    sign_algorithm: "sha256",
  };
}

/**
 * Get default macOS signing config.
 */
export function getDefaultMacOSConfig(): MacOSSigningConfig {
  return {
    identity: "",
    team_id: "",
    hardened_runtime: true,
    notarize: false,
    gatekeeper_assess: true,
  };
}

/**
 * Get default Linux signing config.
 */
export function getDefaultLinuxConfig(): LinuxSigningConfig {
  return {
    gpg_key_id: "",
  };
}

// ============================================================================
// Filter Helpers
// ============================================================================

/**
 * Filter discovered certificates by platform.
 */
export function filterCertificatesByPlatform(
  certificates: DiscoveredCertificate[],
  platform: SigningPlatform
): DiscoveredCertificate[] {
  return certificates.filter((cert) => cert.platform === platform);
}

/**
 * Filter to get only code signing certificates.
 */
export function filterCodeSigningCertificates(
  certificates: DiscoveredCertificate[]
): DiscoveredCertificate[] {
  return certificates.filter((cert) => cert.is_code_sign !== false);
}

/**
 * Filter to get only non-expired certificates.
 */
export function filterNonExpiredCertificates(
  certificates: DiscoveredCertificate[]
): DiscoveredCertificate[] {
  return certificates.filter((cert) => !cert.is_expired);
}

// ============================================================================
// Expiry Warning Message
// ============================================================================

/**
 * Build an expiry warning message for storage/display.
 */
export function buildExpiryWarningMessage(cert: DiscoveredCertificate): string | null {
  if (cert.is_expired) {
    return `Signing certificate expired on ${cert.expires_at || "unknown date"}.`;
  }
  if (typeof cert.days_to_expiry === "number" && cert.days_to_expiry <= EXPIRY_WARNING_THRESHOLD_DAYS) {
    return `Signing certificate expires in ${cert.days_to_expiry} days (${cert.expires_at || "date unknown"}).`;
  }
  return null;
}
