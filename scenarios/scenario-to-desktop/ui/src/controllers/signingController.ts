/**
 * Signing controller - orchestrates signing configuration operations.
 * Combines API calls and services for the Signing page.
 */

import {
  fetchSigningConfig,
  saveSigningConfig,
  validateSigningConfig,
  checkSigningReadiness,
  fetchSigningPrerequisites,
  deleteSigningConfig,
  discoverCertificates,
  generateLinuxSigningKey,
  type SigningConfig,
  type SigningReadinessResponse,
  type SigningValidationResult,
  type ToolDetectionResult,
  type DiscoveredCertificate,
} from "../lib/api";
import {
  applyCertificateToConfig,
  detectExpiryWarnings,
  hasUnsavedConfigChanges,
  getDefaultSigningConfig,
  type SigningPlatform,
  type ExpiryWarning,
} from "../services/signing.service";

// ============================================================================
// Types
// ============================================================================

export interface SigningPageData {
  config: SigningConfig | null;
  readiness: SigningReadinessResponse | null;
  prerequisites: ToolDetectionResult[];
  error: string | null;
}

export interface SaveSigningConfigResult {
  success: boolean;
  error: string | null;
}

export interface DiscoverCertificatesResult {
  certificates: DiscoveredCertificate[];
  warnings: ExpiryWarning[];
  error: string | null;
}

export interface GenerateKeyResult {
  fingerprint: string | null;
  homedir: string | null;
  error: string | null;
}

// ============================================================================
// Page Data Loading
// ============================================================================

/**
 * Load all data needed for the Signing page.
 */
export async function loadSigningPageData(
  scenarioName: string
): Promise<SigningPageData> {
  if (!scenarioName) {
    return {
      config: null,
      readiness: null,
      prerequisites: [],
      error: null,
    };
  }

  try {
    // Load all data in parallel
    const [configResponse, readinessResponse, prerequisitesResponse] = await Promise.all([
      fetchSigningConfig(scenarioName).catch(() => ({ config: null })),
      checkSigningReadiness(scenarioName).catch(() => null),
      fetchSigningPrerequisites().catch(() => ({ tools: [] })),
    ]);

    return {
      config: configResponse?.config ?? null,
      readiness: readinessResponse,
      prerequisites: prerequisitesResponse?.tools ?? [],
      error: null,
    };
  } catch (error) {
    return {
      config: null,
      readiness: null,
      prerequisites: [],
      error: error instanceof Error ? error.message : "Failed to load signing data",
    };
  }
}

/**
 * Load just the signing config.
 */
export async function loadSigningConfig(
  scenarioName: string
): Promise<{ config: SigningConfig | null; error: string | null }> {
  if (!scenarioName) {
    return { config: null, error: null };
  }

  try {
    const response = await fetchSigningConfig(scenarioName);
    return { config: response?.config ?? null, error: null };
  } catch (error) {
    return {
      config: null,
      error: error instanceof Error ? error.message : "Failed to load signing config",
    };
  }
}

/**
 * Load signing readiness status.
 */
export async function loadSigningReadiness(
  scenarioName: string
): Promise<{ readiness: SigningReadinessResponse | null; error: string | null }> {
  if (!scenarioName) {
    return { readiness: null, error: null };
  }

  try {
    const readiness = await checkSigningReadiness(scenarioName);
    return { readiness, error: null };
  } catch (error) {
    return {
      readiness: null,
      error: error instanceof Error ? error.message : "Failed to load signing readiness",
    };
  }
}

/**
 * Load signing prerequisites (tools).
 */
export async function loadSigningPrerequisites(): Promise<{
  tools: ToolDetectionResult[];
  error: string | null;
}> {
  try {
    const response = await fetchSigningPrerequisites();
    return { tools: response?.tools ?? [], error: null };
  } catch (error) {
    return {
      tools: [],
      error: error instanceof Error ? error.message : "Failed to load prerequisites",
    };
  }
}

// ============================================================================
// Config Operations
// ============================================================================

/**
 * Save signing configuration with validation.
 */
export async function saveSigningConfigWithValidation(
  scenarioName: string,
  config: SigningConfig
): Promise<{
  success: boolean;
  validationResult?: SigningValidationResult;
  error: string | null;
}> {
  if (!scenarioName) {
    return { success: false, error: "No scenario selected" };
  }

  try {
    await saveSigningConfig(scenarioName, config);

    // Optionally validate after save
    let validationResult: SigningValidationResult | undefined;
    try {
      validationResult = await validateSigningConfig(scenarioName);
    } catch {
      // Validation is optional, don't fail save if it fails
    }

    return { success: true, validationResult, error: null };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to save config",
    };
  }
}

/**
 * Delete signing configuration.
 */
export async function deleteSigningConfigForScenario(
  scenarioName: string
): Promise<SaveSigningConfigResult> {
  if (!scenarioName) {
    return { success: false, error: "No scenario selected" };
  }

  try {
    await deleteSigningConfig(scenarioName);
    return { success: true, error: null };
  } catch (error) {
    return {
      success: false,
      error: error instanceof Error ? error.message : "Failed to delete config",
    };
  }
}

/**
 * Validate signing configuration.
 */
export async function validateSigningConfigForScenario(
  scenarioName: string
): Promise<{ result: SigningValidationResult | null; error: string | null }> {
  if (!scenarioName) {
    return { result: null, error: "No scenario selected" };
  }

  try {
    const result = await validateSigningConfig(scenarioName);
    return { result, error: null };
  } catch (error) {
    return {
      result: null,
      error: error instanceof Error ? error.message : "Validation failed",
    };
  }
}

// ============================================================================
// Certificate Discovery
// ============================================================================

/**
 * Discover certificates for a platform and process expiry warnings.
 */
export async function discoverAndFilterCertificates(
  platform: SigningPlatform
): Promise<DiscoverCertificatesResult> {
  try {
    const response = await discoverCertificates(platform);
    const certificates = response.certificates || [];
    const warnings = detectExpiryWarnings(certificates);

    return { certificates, warnings, error: null };
  } catch (error) {
    return {
      certificates: [],
      warnings: [],
      error: error instanceof Error ? error.message : "Certificate discovery failed",
    };
  }
}

/**
 * Apply a discovered certificate to the current config.
 */
export function applyDiscoveredCertificate(
  platform: SigningPlatform,
  certificate: DiscoveredCertificate,
  currentConfig: SigningConfig
): SigningConfig {
  return applyCertificateToConfig(platform, certificate, currentConfig);
}

// ============================================================================
// Linux Key Generation
// ============================================================================

export interface LinuxKeyGenPayload {
  name?: string;
  email?: string;
  passphrase?: string;
  passphrase_env?: string;
  expiry?: string;
}

/**
 * Generate a Linux GPG signing key.
 */
export async function generateLinuxKey(
  scenarioName: string,
  payload: LinuxKeyGenPayload
): Promise<GenerateKeyResult> {
  if (!scenarioName) {
    return { fingerprint: null, homedir: null, error: "No scenario selected" };
  }

  try {
    const response = await generateLinuxSigningKey(scenarioName, payload);
    return {
      fingerprint: response.fingerprint ?? null,
      homedir: response.homedir ?? null,
      error: null,
    };
  } catch (error) {
    return {
      fingerprint: null,
      homedir: null,
      error: error instanceof Error ? error.message : "Key generation failed",
    };
  }
}

// ============================================================================
// State Helpers
// ============================================================================

/**
 * Check if local config has changes compared to server.
 */
export function checkForUnsavedChanges(
  localConfig: SigningConfig,
  serverConfig: SigningConfig | null | undefined
): boolean {
  return hasUnsavedConfigChanges(localConfig, serverConfig);
}

/**
 * Create a fresh signing config from defaults.
 */
export function createFreshSigningConfig(): SigningConfig {
  return getDefaultSigningConfig();
}

/**
 * Merge server config with local overrides.
 */
export function mergeConfigWithServer(
  serverConfig: SigningConfig | null | undefined,
  localOverrides: Partial<SigningConfig>
): SigningConfig {
  const base = serverConfig ?? getDefaultSigningConfig();
  return { ...base, ...localOverrides };
}

// ============================================================================
// Expiry Warning Storage
// ============================================================================

const EXPIRY_WARNING_KEY = "std_signing_expiry_warning";

/**
 * Store an expiry warning in local storage.
 */
export function storeExpiryWarning(message: string): void {
  if (typeof window === "undefined") return;
  window.localStorage.setItem(EXPIRY_WARNING_KEY, message);
}

/**
 * Get stored expiry warning from local storage.
 */
export function getStoredExpiryWarning(): string | null {
  if (typeof window === "undefined") return null;
  return window.localStorage.getItem(EXPIRY_WARNING_KEY);
}

/**
 * Clear stored expiry warning.
 */
export function clearStoredExpiryWarning(): void {
  if (typeof window === "undefined") return;
  window.localStorage.removeItem(EXPIRY_WARNING_KEY);
}
