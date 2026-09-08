/**
 * Bundle Validator Implementation
 *
 * DOC: docs/internal/SEAMS.md#bundle-validator
 *
 * Validates bundle manifests before spawning the runtime.
 * Performs fast, Electron-side checks that catch obvious issues early.
 */

import type {
    BundleManifest,
    BundleValidationResult,
    IBundleFileSystem,
    IBundlePathUtils,
    IPlatformInfo,
} from "./types";

/**
 * Get the platform key for the current OS/arch in the manifest format.
 * E.g., "linux-x64", "darwin-arm64", "win-x64"
 */
export function getBundlePlatformKey(platformInfo: IPlatformInfo): string {
    const arch = platformInfo.arch === "x64" ? "x64" : platformInfo.arch;
    const os = platformInfo.platform === "darwin"
        ? "darwin"
        : platformInfo.platform === "win32"
            ? "win"
            : "linux";
    return `${os}-${arch}`;
}

/**
 * Get platform key aliases for the manifest.
 * E.g., "mac-x64" is an alias for "darwin-x64"
 */
export function getPlatformKeyAliases(key: string): string[] {
    const keys = [key];
    if (key.startsWith("darwin-")) {
        keys.push("mac-" + key.slice(7));
    } else if (key.startsWith("mac-")) {
        keys.push("darwin-" + key.slice(4));
    } else if (key.startsWith("win-")) {
        keys.push("windows-" + key.slice(4));
    } else if (key.startsWith("windows-")) {
        keys.push("win-" + key.slice(8));
    }
    return keys;
}

/**
 * Load and parse a bundle manifest from disk.
 */
export async function loadBundleManifest(
    manifestPath: string,
    fs: IBundleFileSystem
): Promise<BundleManifest | null> {
    try {
        const raw = await fs.readFile(manifestPath, "utf-8");
        return JSON.parse(raw) as BundleManifest;
    } catch (error) {
        console.error("[Bundle] Failed to load bundle manifest:", error);
        return null;
    }
}

/**
 * Check if a path exists.
 */
async function pathExists(filePath: string, fs: IBundleFileSystem): Promise<boolean> {
    try {
        await fs.access(filePath);
        return true;
    } catch {
        return false;
    }
}

/**
 * Create an initial validation result.
 */
function createInitialResult(): BundleValidationResult {
    return {
        valid: true,
        errors: [],
        warnings: [],
        missingBinaries: [],
        missingAssets: [],
    };
}

/**
 * Perform pre-flight validation of a bundle manifest.
 * Checks that all required binaries and assets exist before spawning the runtime.
 * This is a fast, Electron-side check that catches obvious issues early.
 */
export async function validateBundlePreFlight(
    bundleRoot: string,
    manifestPath: string,
    fs: IBundleFileSystem,
    path: IBundlePathUtils,
    platformInfo: IPlatformInfo
): Promise<BundleValidationResult> {
    const result = createInitialResult();

    // Load manifest
    const manifest = await loadBundleManifest(manifestPath, fs);
    if (!manifest) {
        result.valid = false;
        result.errors.push("Failed to load or parse bundle manifest");
        return result;
    }

    // Validate schema version and target
    if (!manifest.schema_version) {
        result.valid = false;
        result.errors.push("Manifest missing schema_version");
    }
    if (manifest.target !== "desktop") {
        result.valid = false;
        result.errors.push(`Invalid manifest target: ${manifest.target} (expected 'desktop')`);
    }
    if (!manifest.app?.name || !manifest.app?.version) {
        result.valid = false;
        result.errors.push("Manifest missing app.name or app.version");
    }
    if (!manifest.ipc?.host || !manifest.ipc?.port) {
        result.valid = false;
        result.errors.push("Manifest missing ipc.host or ipc.port");
    }

    // Early exit if manifest is fundamentally broken
    if (!result.valid) {
        return result;
    }

    // Get platform keys to check
    const platformKey = getBundlePlatformKey(platformInfo);
    const platformKeys = getPlatformKeyAliases(platformKey);

    // Check each service
    for (const service of manifest.services ?? []) {
        // Check for service binary
        let binaryFound = false;
        let binaryPath = "";

        for (const pk of platformKeys) {
            const bin = service.binaries?.[pk];
            if (bin?.path) {
                const normalizedPath = bin.path.replace(/\\/g, path.sep);
                const fullPath = path.join(bundleRoot, normalizedPath);
                if (await pathExists(fullPath, fs)) {
                    binaryFound = true;
                    break;
                }
                binaryPath = bin.path;
            }
        }

        if (!binaryFound && Object.keys(service.binaries ?? {}).length > 0) {
            // Only flag as missing if binaries are defined but none match platform
            const anyBinary = Object.values(service.binaries ?? {})[0];
            result.missingBinaries.push({
                serviceId: service.id,
                platform: platformKey,
                path: binaryPath || anyBinary?.path || "<undefined>",
            });
            result.errors.push(`Binary missing for service ${service.id} on platform ${platformKey}`);
            result.valid = false;
        }

        // Check for service assets (existence only, checksums verified by runtime)
        for (const asset of service.assets ?? []) {
            const normalizedPath = asset.path.replace(/\\/g, path.sep);
            const assetPath = path.join(bundleRoot, normalizedPath);
            if (!(await pathExists(assetPath, fs))) {
                result.missingAssets.push({
                    serviceId: service.id,
                    path: asset.path,
                });
                result.errors.push(`Asset missing for service ${service.id}: ${asset.path}`);
                result.valid = false;
            }
        }
    }

    return result;
}

/**
 * Create a Node.js fs-based bundle filesystem.
 */
export function createNodeBundleFileSystem(
    fsPromises: typeof import("node:fs").promises
): IBundleFileSystem {
    return {
        readFile: (path, encoding) => fsPromises.readFile(path, encoding),
        access: (path) => fsPromises.access(path),
    };
}

/**
 * Create a Node.js path module wrapper.
 */
export function createNodeBundlePathUtils(
    pathModule: typeof import("node:path")
): IBundlePathUtils {
    return {
        join: (...segments) => pathModule.join(...segments),
        sep: pathModule.sep,
    };
}

/**
 * Create a real platform info object.
 */
export function createRealPlatformInfo(): IPlatformInfo {
    return {
        arch: process.arch,
        platform: process.platform,
    };
}
