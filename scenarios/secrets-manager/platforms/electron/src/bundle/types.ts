/**
 * Bundle Module Types
 *
 * DOC: docs/internal/SEAMS.md#bundle-types
 *
 * Type definitions for bundle manifest parsing and validation.
 */

// ===== Bundle Manifest Types =====

/**
 * App information in the bundle manifest.
 */
export interface BundleApp {
    name: string;
    version: string;
}

/**
 * IPC configuration in the bundle manifest.
 */
export interface BundleIpc {
    host: string;
    port: number;
}

/**
 * Binary configuration for a service.
 */
export interface BundleBinary {
    path: string;
}

/**
 * Asset configuration for a service.
 */
export interface BundleAsset {
    path: string;
    sha256?: string;
}

/**
 * Health check configuration.
 */
export interface BundleHealthCheck {
    type: string;
}

/**
 * Service definition in the bundle manifest.
 */
export interface BundleService {
    id: string;
    binaries: Record<string, BundleBinary>;
    assets?: BundleAsset[];
    health: BundleHealthCheck;
    readiness: BundleHealthCheck;
    env?: Record<string, string>;
}

/**
 * Complete bundle manifest structure.
 */
export interface BundleManifest {
    schema_version: string;
    target: string;
    app: BundleApp;
    ipc: BundleIpc;
    services: BundleService[];
}

// ===== Validation Result Types =====

/**
 * A missing binary detected during validation.
 */
export interface MissingBinary {
    serviceId: string;
    platform: string;
    path: string;
}

/**
 * A missing asset detected during validation.
 */
export interface MissingAsset {
    serviceId: string;
    path: string;
}

/**
 * Result of pre-flight bundle validation.
 */
export interface BundleValidationResult {
    valid: boolean;
    errors: string[];
    warnings: string[];
    missingBinaries: MissingBinary[];
    missingAssets: MissingAsset[];
}

/**
 * An error or warning from runtime validation.
 */
export interface RuntimeValidationIssue {
    code: string;
    service?: string;
    path?: string;
    message: string;
}

/**
 * A missing binary reported by runtime.
 */
export interface RuntimeMissingBinary {
    service_id: string;
    platform: string;
    path: string;
}

/**
 * A missing asset reported by runtime.
 */
export interface RuntimeMissingAsset {
    service_id: string;
    path: string;
}

/**
 * An invalid checksum reported by runtime.
 */
export interface RuntimeInvalidChecksum {
    service_id: string;
    path: string;
    expected: string;
    actual: string;
}

/**
 * Response from the runtime's /validate endpoint.
 */
export interface RuntimeValidationResponse {
    valid: boolean;
    errors?: RuntimeValidationIssue[];
    warnings?: RuntimeValidationIssue[];
    missing_binaries?: RuntimeMissingBinary[];
    missing_assets?: RuntimeMissingAsset[];
    invalid_checksums?: RuntimeInvalidChecksum[];
}

// ===== Seam Interfaces =====

/**
 * Filesystem operations needed by the bundle module.
 */
export interface IBundleFileSystem {
    readFile(path: string, encoding: "utf-8"): Promise<string>;
    access(path: string): Promise<void>;
}

/**
 * Path utilities needed by the bundle module.
 */
export interface IBundlePathUtils {
    join(...segments: string[]): string;
    sep: string;
}

/**
 * Platform information seam.
 */
export interface IPlatformInfo {
    arch: string;
    platform: string;
}
