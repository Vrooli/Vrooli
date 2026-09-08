/**
 * Bundle Module
 *
 * DOC: docs/internal/SEAMS.md#bundle-module
 *
 * Barrel exports for bundle manifest parsing and validation.
 */

// Types
export type {
    BundleApp,
    BundleIpc,
    BundleBinary,
    BundleAsset,
    BundleHealthCheck,
    BundleService,
    BundleManifest,
    MissingBinary,
    MissingAsset,
    BundleValidationResult,
    RuntimeValidationIssue,
    RuntimeMissingBinary,
    RuntimeMissingAsset,
    RuntimeInvalidChecksum,
    RuntimeValidationResponse,
    IBundleFileSystem,
    IBundlePathUtils,
    IPlatformInfo,
} from "./types";

// Validator
export {
    getBundlePlatformKey,
    getPlatformKeyAliases,
    loadBundleManifest,
    validateBundlePreFlight,
    createNodeBundleFileSystem,
    createNodeBundlePathUtils,
    createRealPlatformInfo,
} from "./validator";
