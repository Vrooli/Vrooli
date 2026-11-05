/**
 * Bundle Validator Tests
 *
 * DOC: docs/internal/SEAMS.md#bundle-validator-tests
 *
 * Tests for bundle manifest validation.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import type {
    IBundleFileSystem,
    IBundlePathUtils,
    IPlatformInfo,
    BundleManifest,
} from "../types";
import {
    getBundlePlatformKey,
    getPlatformKeyAliases,
    loadBundleManifest,
    validateBundlePreFlight,
} from "../validator";

// ===== Mock Factories =====

interface MockBundleFileSystem extends IBundleFileSystem {
    _files: Map<string, string>;
}

function createMockFileSystem(): MockBundleFileSystem {
    const files = new Map<string, string>();

    return {
        _files: files,
        readFile: vi.fn(async (path: string) => {
            const content = files.get(path);
            if (content === undefined) {
                const error = new Error(`ENOENT: no such file: ${path}`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
            return content;
        }),
        access: vi.fn(async (path: string) => {
            if (!files.has(path)) {
                const error = new Error(`ENOENT: no such file: ${path}`);
                (error as NodeJS.ErrnoException).code = "ENOENT";
                throw error;
            }
        }),
    };
}

function createMockPathUtils(): IBundlePathUtils {
    return {
        join: (...segments) => segments.join("/"),
        sep: "/",
    };
}

function createMockPlatformInfo(platform = "linux", arch = "x64"): IPlatformInfo {
    return { platform, arch };
}

function createValidManifest(overrides?: Partial<BundleManifest>): BundleManifest {
    return {
        schema_version: "1.0",
        target: "desktop",
        app: { name: "TestApp", version: "1.0.0" },
        ipc: { host: "127.0.0.1", port: 39200 },
        services: [],
        ...overrides,
    };
}

// ===== Tests =====

describe("getBundlePlatformKey", () => {
    it("returns linux-x64 for linux x64", () => {
        const key = getBundlePlatformKey({ platform: "linux", arch: "x64" });
        expect(key).toBe("linux-x64");
    });

    it("returns darwin-arm64 for macOS ARM", () => {
        const key = getBundlePlatformKey({ platform: "darwin", arch: "arm64" });
        expect(key).toBe("darwin-arm64");
    });

    it("returns darwin-x64 for macOS Intel", () => {
        const key = getBundlePlatformKey({ platform: "darwin", arch: "x64" });
        expect(key).toBe("darwin-x64");
    });

    it("returns win-x64 for Windows", () => {
        const key = getBundlePlatformKey({ platform: "win32", arch: "x64" });
        expect(key).toBe("win-x64");
    });

    it("preserves other architectures", () => {
        const key = getBundlePlatformKey({ platform: "linux", arch: "arm64" });
        expect(key).toBe("linux-arm64");
    });
});

describe("getPlatformKeyAliases", () => {
    it("includes darwin alias for mac", () => {
        const aliases = getPlatformKeyAliases("mac-x64");
        expect(aliases).toContain("mac-x64");
        expect(aliases).toContain("darwin-x64");
    });

    it("includes mac alias for darwin", () => {
        const aliases = getPlatformKeyAliases("darwin-arm64");
        expect(aliases).toContain("darwin-arm64");
        expect(aliases).toContain("mac-arm64");
    });

    it("includes windows alias for win", () => {
        const aliases = getPlatformKeyAliases("win-x64");
        expect(aliases).toContain("win-x64");
        expect(aliases).toContain("windows-x64");
    });

    it("includes win alias for windows", () => {
        const aliases = getPlatformKeyAliases("windows-x64");
        expect(aliases).toContain("windows-x64");
        expect(aliases).toContain("win-x64");
    });

    it("returns single key for linux", () => {
        const aliases = getPlatformKeyAliases("linux-x64");
        expect(aliases).toEqual(["linux-x64"]);
    });
});

describe("loadBundleManifest", () => {
    let fs: MockBundleFileSystem;

    beforeEach(() => {
        fs = createMockFileSystem();
    });

    it("loads and parses valid manifest", async () => {
        const manifest = createValidManifest({ app: { name: "MyApp", version: "2.0.0" } });
        fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

        const result = await loadBundleManifest("/bundle/manifest.json", fs);

        expect(result).not.toBeNull();
        expect(result?.app.name).toBe("MyApp");
        expect(result?.app.version).toBe("2.0.0");
    });

    it("returns null for missing file", async () => {
        const result = await loadBundleManifest("/missing.json", fs);
        expect(result).toBeNull();
    });

    it("returns null for invalid JSON", async () => {
        fs._files.set("/bundle/manifest.json", "not valid json {");

        const result = await loadBundleManifest("/bundle/manifest.json", fs);

        expect(result).toBeNull();
    });
});

describe("validateBundlePreFlight", () => {
    let fs: MockBundleFileSystem;
    let path: IBundlePathUtils;
    let platform: IPlatformInfo;

    beforeEach(() => {
        fs = createMockFileSystem();
        path = createMockPathUtils();
        platform = createMockPlatformInfo();
    });

    describe("manifest structure validation", () => {
        it("passes for valid manifest", async () => {
            const manifest = createValidManifest();
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(true);
            expect(result.errors).toHaveLength(0);
        });

        it("fails for missing manifest", async () => {
            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.errors).toContain("Failed to load or parse bundle manifest");
        });

        it("fails for missing schema_version", async () => {
            const manifest = createValidManifest();
            delete (manifest as Partial<BundleManifest>).schema_version;
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.errors).toContain("Manifest missing schema_version");
        });

        it("fails for invalid target", async () => {
            const manifest = createValidManifest({ target: "mobile" });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => e.includes("Invalid manifest target"))).toBe(true);
        });

        it("fails for missing app.name", async () => {
            const manifest = createValidManifest();
            manifest.app.name = "";
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => e.includes("app.name"))).toBe(true);
        });

        it("fails for missing ipc.host", async () => {
            const manifest = createValidManifest();
            manifest.ipc.host = "";
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => e.includes("ipc.host"))).toBe(true);
        });

        it("accepts ipc.port 0 as an allocator request", async () => {
            const manifest = createValidManifest();
            manifest.ipc.port = 0;
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.errors.some(e => e.includes("ipc.port"))).toBe(false);
            expect(result.valid).toBe(true);
        });

        it("fails for a missing ipc.port", async () => {
            const manifest = createValidManifest();
            delete (manifest.ipc as Partial<typeof manifest.ipc>).port;
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => e.includes("ipc.port"))).toBe(true);
        });

        it("fails for an out-of-range ipc.port", async () => {
            const manifest = createValidManifest();
            manifest.ipc.port = 70000;
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => e.includes("ipc.port"))).toBe(true);
        });
    });

    describe("service binary validation", () => {
        it("passes when binary exists for platform", async () => {
            const manifest = createValidManifest({
                services: [{
                    id: "api",
                    binaries: { "linux-x64": { path: "bin/api" } },
                    assets: [],
                    health: { type: "http" },
                    readiness: { type: "http" },
                }],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));
            fs._files.set("/bundle/bin/api", "binary content");

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(true);
            expect(result.missingBinaries).toHaveLength(0);
        });

        it("passes when binary exists for platform alias", async () => {
            platform = createMockPlatformInfo("darwin", "x64");
            const manifest = createValidManifest({
                services: [{
                    id: "api",
                    binaries: { "mac-x64": { path: "bin/api" } }, // mac- alias
                    assets: [],
                    health: { type: "http" },
                    readiness: { type: "http" },
                }],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));
            fs._files.set("/bundle/bin/api", "binary content");

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(true);
        });

        it("fails when binary missing for platform", async () => {
            const manifest = createValidManifest({
                services: [{
                    id: "api",
                    binaries: { "linux-x64": { path: "bin/api" } },
                    assets: [],
                    health: { type: "http" },
                    readiness: { type: "http" },
                }],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));
            // Binary file NOT added

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.missingBinaries).toHaveLength(1);
            expect(result.missingBinaries[0]?.serviceId).toBe("api");
            expect(result.missingBinaries[0]?.platform).toBe("linux-x64");
        });

        it("fails when no binary defined for current platform", async () => {
            const manifest = createValidManifest({
                services: [{
                    id: "api",
                    binaries: { "darwin-x64": { path: "bin/api-mac" } }, // Only mac binary
                    assets: [],
                    health: { type: "http" },
                    readiness: { type: "http" },
                }],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));
            fs._files.set("/bundle/bin/api-mac", "binary content");

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.errors.some(e => e.includes("Binary missing"))).toBe(true);
        });

        it("passes for services with empty binaries object", async () => {
            const manifest = createValidManifest({
                services: [{
                    id: "sidecar",
                    binaries: {}, // No binaries (maybe a script-based service)
                    assets: [],
                    health: { type: "http" },
                    readiness: { type: "http" },
                }],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(true);
        });
    });

    describe("service asset validation", () => {
        it("passes when all assets exist", async () => {
            const manifest = createValidManifest({
                services: [{
                    id: "ui",
                    binaries: {},
                    assets: [
                        { path: "assets/index.html" },
                        { path: "assets/style.css" },
                    ],
                    health: { type: "http" },
                    readiness: { type: "http" },
                }],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));
            fs._files.set("/bundle/assets/index.html", "<html>");
            fs._files.set("/bundle/assets/style.css", "body {}");

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(true);
            expect(result.missingAssets).toHaveLength(0);
        });

        it("fails when asset is missing", async () => {
            const manifest = createValidManifest({
                services: [{
                    id: "ui",
                    binaries: {},
                    assets: [{ path: "assets/missing.html" }],
                    health: { type: "http" },
                    readiness: { type: "http" },
                }],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.missingAssets).toHaveLength(1);
            expect(result.missingAssets[0]?.serviceId).toBe("ui");
            expect(result.missingAssets[0]?.path).toBe("assets/missing.html");
        });

        it("reports all missing assets", async () => {
            const manifest = createValidManifest({
                services: [{
                    id: "ui",
                    binaries: {},
                    assets: [
                        { path: "assets/missing1.html" },
                        { path: "assets/missing2.css" },
                    ],
                    health: { type: "http" },
                    readiness: { type: "http" },
                }],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.missingAssets).toHaveLength(2);
            expect(result.errors).toHaveLength(2);
        });
    });

    describe("multiple services", () => {
        it("validates all services", async () => {
            const manifest = createValidManifest({
                services: [
                    {
                        id: "api",
                        binaries: { "linux-x64": { path: "bin/api" } },
                        assets: [{ path: "config/api.json" }],
                        health: { type: "http" },
                        readiness: { type: "http" },
                    },
                    {
                        id: "ui",
                        binaries: {},
                        assets: [{ path: "public/index.html" }],
                        health: { type: "http" },
                        readiness: { type: "http" },
                    },
                ],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));
            fs._files.set("/bundle/bin/api", "binary");
            fs._files.set("/bundle/config/api.json", "{}");
            fs._files.set("/bundle/public/index.html", "<html>");

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(true);
        });

        it("collects errors from multiple services", async () => {
            const manifest = createValidManifest({
                services: [
                    {
                        id: "api",
                        binaries: { "linux-x64": { path: "bin/api" } },
                        assets: [],
                        health: { type: "http" },
                        readiness: { type: "http" },
                    },
                    {
                        id: "worker",
                        binaries: { "linux-x64": { path: "bin/worker" } },
                        assets: [],
                        health: { type: "http" },
                        readiness: { type: "http" },
                    },
                ],
            });
            fs._files.set("/bundle/manifest.json", JSON.stringify(manifest));
            // Neither binary exists

            const result = await validateBundlePreFlight("/bundle", "/bundle/manifest.json", fs, path, platform);

            expect(result.valid).toBe(false);
            expect(result.missingBinaries).toHaveLength(2);
            expect(result.missingBinaries.map(b => b.serviceId).sort()).toEqual(["api", "worker"]);
        });
    });
});
