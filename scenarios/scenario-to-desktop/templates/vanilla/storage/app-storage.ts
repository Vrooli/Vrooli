/**
 * App Storage Implementation
 *
 * DOC: docs/internal/SEAMS.md#app-storage
 *
 * Provides sandboxed filesystem access within the app's userData directory.
 * All operations validate paths to prevent directory traversal attacks.
 */

import type {
    IAppStorage,
    IStorageFileSystem,
    IStoragePathUtils,
    StorageConfig,
    StorageEntry,
    StorageStats,
    StorageInfo,
} from "./types";

/**
 * Create an app storage instance with injected dependencies.
 *
 * @param fs - Filesystem operations seam
 * @param path - Path utilities seam
 * @param config - Storage configuration
 */
export function createAppStorage(
    fs: IStorageFileSystem,
    path: IStoragePathUtils,
    config: StorageConfig
): IAppStorage {
    const storageDirName = config.storageDirName ?? "app-storage";
    let storageRoot: string | null = null;

    /**
     * Get the storage root, initializing it if needed.
     */
    async function getRoot(): Promise<string> {
        if (storageRoot) return storageRoot;

        storageRoot = path.join(config.userDataPath, storageDirName);
        await fs.mkdir(storageRoot, { recursive: true });
        return storageRoot;
    }

    /**
     * Resolve and validate a relative path to ensure it stays within storage root.
     * Prevents directory traversal attacks (e.g., "../../../etc/passwd").
     * @returns Absolute path within storage root, or null if invalid
     */
    async function resolvePath(relativePath: string): Promise<string | null> {
        const root = await getRoot();

        // Normalize the path to resolve any ".." or "." segments
        const normalizedRelative = path.normalize(relativePath);

        // Prevent absolute paths or paths that escape the storage root
        if (path.isAbsolute(normalizedRelative)) {
            console.warn(`[Storage] Rejected absolute path: ${relativePath}`);
            return null;
        }

        const resolved = path.resolve(root, normalizedRelative);

        // Ensure the resolved path is still within the storage root
        if (!resolved.startsWith(root + path.sep) && resolved !== root) {
            console.warn(`[Storage] Path traversal attempt blocked: ${relativePath} -> ${resolved}`);
            return null;
        }

        return resolved;
    }

    /**
     * Calculate total size of a directory recursively.
     */
    async function calculateDirectorySize(dirPath: string): Promise<{ size: number; count: number }> {
        let totalSize = 0;
        let fileCount = 0;

        async function walk(currentPath: string): Promise<void> {
            try {
                const entries = await fs.readdir(currentPath, { withFileTypes: true });
                for (const entry of entries) {
                    const entryPath = path.join(currentPath, entry.name);
                    if (entry.isDirectory()) {
                        await walk(entryPath);
                    } else if (entry.isFile()) {
                        const stats = await fs.stat(entryPath);
                        totalSize += stats.size;
                        fileCount++;
                    }
                }
            } catch {
                // Ignore errors (permission issues, etc.)
            }
        }

        await walk(dirPath);
        return { size: totalSize, count: fileCount };
    }

    const storage: IAppStorage = {
        getRoot,
        resolvePath,

        async ensureDir(relativePath: string): Promise<void> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                throw new Error("Invalid storage path");
            }
            await fs.mkdir(fullPath, { recursive: true });
        },

        async writeFile(relativePath: string, data: string | Buffer): Promise<void> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                throw new Error("Invalid storage path");
            }

            // Ensure parent directory exists
            await fs.mkdir(path.dirname(fullPath), { recursive: true });

            if (Buffer.isBuffer(data)) {
                await fs.writeFile(fullPath, data);
            } else {
                await fs.writeFile(fullPath, data, "utf-8");
            }
        },

        async readFile(relativePath: string): Promise<Buffer | null> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                return null;
            }

            try {
                return await fs.readFile(fullPath);
            } catch (error: unknown) {
                if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                    return null;
                }
                throw error;
            }
        },

        async readTextFile(relativePath: string): Promise<string | null> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                return null;
            }

            try {
                return await fs.readFile(fullPath, "utf-8");
            } catch (error: unknown) {
                if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                    return null;
                }
                throw error;
            }
        },

        async deleteFile(relativePath: string): Promise<boolean> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                return false;
            }

            try {
                await fs.unlink(fullPath);
                return true;
            } catch (error: unknown) {
                if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                    return false;
                }
                throw error;
            }
        },

        async deleteDir(relativePath: string): Promise<boolean> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                return false;
            }

            try {
                await fs.rm(fullPath, { recursive: true, force: true });
                return true;
            } catch (error: unknown) {
                if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                    return false;
                }
                throw error;
            }
        },

        async listDir(relativePath: string): Promise<StorageEntry[] | null> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                return null;
            }

            try {
                const root = await getRoot();
                const entries = await fs.readdir(fullPath, { withFileTypes: true });
                const results: StorageEntry[] = [];

                for (const entry of entries) {
                    const entryPath = path.join(fullPath, entry.name);
                    const stats = await fs.stat(entryPath);
                    results.push({
                        name: entry.name,
                        path: path.relative(root, entryPath),
                        isDirectory: entry.isDirectory(),
                        isFile: entry.isFile(),
                        size: stats.size,
                        createdAt: Math.floor(stats.birthtimeMs),
                        modifiedAt: Math.floor(stats.mtimeMs),
                    });
                }

                return results;
            } catch (error: unknown) {
                if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                    return null;
                }
                throw error;
            }
        },

        async exists(relativePath: string): Promise<boolean> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                return false;
            }

            try {
                await fs.access(fullPath);
                return true;
            } catch {
                return false;
            }
        },

        async stat(relativePath: string): Promise<StorageStats | null> {
            const fullPath = await resolvePath(relativePath);
            if (!fullPath) {
                return null;
            }

            try {
                const stats = await fs.stat(fullPath);
                return {
                    size: stats.size,
                    createdAt: Math.floor(stats.birthtimeMs),
                    modifiedAt: Math.floor(stats.mtimeMs),
                    isDirectory: stats.isDirectory(),
                    isFile: stats.isFile(),
                };
            } catch (error: unknown) {
                if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                    return null;
                }
                throw error;
            }
        },

        async getInfo(): Promise<StorageInfo> {
            const root = await getRoot();
            const { size, count } = await calculateDirectorySize(root);

            // Note: Node.js doesn't have a built-in cross-platform disk space API.
            // For accurate available space, consider using a package like 'diskusage'.
            // For now, we return a reasonable default estimate.
            const available = 10 * 1024 * 1024 * 1024; // 10GB estimate

            return {
                used: size,
                available,
                count,
            };
        },
    };

    return storage;
}

/**
 * Create a Node.js fs-based storage filesystem.
 */
export function createNodeStorageFileSystem(
    fsPromises: typeof import("node:fs").promises
): IStorageFileSystem {
    return {
        readFile: ((path: string, encoding?: "utf-8") => {
            if (encoding) {
                return fsPromises.readFile(path, encoding);
            }
            return fsPromises.readFile(path);
        }) as IStorageFileSystem["readFile"],
        writeFile: async (path, data, encoding) => {
            if (encoding) {
                await fsPromises.writeFile(path, data, encoding);
            } else {
                await fsPromises.writeFile(path, data);
            }
        },
        mkdir: async (path, options) => {
            await fsPromises.mkdir(path, options);
        },
        readdir: (path, options) => fsPromises.readdir(path, options),
        unlink: (path) => fsPromises.unlink(path),
        rm: (path, options) => fsPromises.rm(path, options),
        stat: (path) => fsPromises.stat(path),
        access: (path) => fsPromises.access(path),
    };
}

/**
 * Create a Node.js path module wrapper.
 */
export function createNodeStoragePathUtils(
    pathModule: typeof import("node:path")
): IStoragePathUtils {
    return {
        join: (...segments) => pathModule.join(...segments),
        dirname: (p) => pathModule.dirname(p),
        normalize: (p) => pathModule.normalize(p),
        resolve: (...segments) => pathModule.resolve(...segments),
        isAbsolute: (p) => pathModule.isAbsolute(p),
        relative: (from, to) => pathModule.relative(from, to),
        sep: pathModule.sep,
    };
}
