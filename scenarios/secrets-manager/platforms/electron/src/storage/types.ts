/**
 * Storage Module Types
 *
 * DOC: docs/internal/SEAMS.md#storage-types
 *
 * Type definitions for the app storage system.
 * Provides sandboxed filesystem access within the app's userData directory.
 */

// ===== Storage Entry Types =====

/**
 * Represents a single entry in a directory listing.
 */
export interface StorageEntry {
    /** File or directory name */
    name: string;
    /** Relative path from storage root */
    path: string;
    /** True if entry is a directory */
    isDirectory: boolean;
    /** True if entry is a file */
    isFile: boolean;
    /** Size in bytes (0 for directories) */
    size: number;
    /** Creation timestamp in milliseconds */
    createdAt: number;
    /** Last modified timestamp in milliseconds */
    modifiedAt: number;
}

/**
 * Statistics for a file or directory.
 */
export interface StorageStats {
    /** Size in bytes */
    size: number;
    /** Creation timestamp in milliseconds */
    createdAt: number;
    /** Last modified timestamp in milliseconds */
    modifiedAt: number;
    /** True if path is a directory */
    isDirectory: boolean;
    /** True if path is a file */
    isFile: boolean;
}

/**
 * Storage usage information.
 */
export interface StorageInfo {
    /** Total bytes used */
    used: number;
    /** Available space in bytes (estimate) */
    available: number;
    /** Total file count */
    count: number;
}

// ===== Seam Interfaces =====

/**
 * Directory entry from readdir with file types.
 */
export interface DirentLike {
    name: string;
    isDirectory(): boolean;
    isFile(): boolean;
}

/**
 * Stats object from stat operation.
 */
export interface StatsLike {
    size: number;
    birthtimeMs: number;
    mtimeMs: number;
    isDirectory(): boolean;
    isFile(): boolean;
}

/**
 * Low-level filesystem operations seam.
 * This interface allows injecting mock filesystems for testing.
 */
export interface IStorageFileSystem {
    readFile(path: string): Promise<Buffer>;
    readFile(path: string, encoding: "utf-8"): Promise<string>;
    writeFile(path: string, data: string | Buffer, encoding?: "utf-8"): Promise<void>;
    mkdir(path: string, options?: { recursive?: boolean }): Promise<void>;
    readdir(path: string, options: { withFileTypes: true }): Promise<DirentLike[]>;
    unlink(path: string): Promise<void>;
    rm(path: string, options?: { recursive?: boolean; force?: boolean }): Promise<void>;
    stat(path: string): Promise<StatsLike>;
    access(path: string): Promise<void>;
}

/**
 * Path utilities needed by the storage module.
 */
export interface IStoragePathUtils {
    join(...segments: string[]): string;
    dirname(path: string): string;
    normalize(path: string): string;
    resolve(...segments: string[]): string;
    isAbsolute(path: string): boolean;
    relative(from: string, to: string): string;
    sep: string;
}

/**
 * Configuration for app storage initialization.
 */
export interface StorageConfig {
    /** Base directory for all storage (typically app.getPath("userData")) */
    userDataPath: string;
    /** Subdirectory name within userData for storage */
    storageDirName?: string;
}

// ===== App Storage Interface =====

/**
 * High-level interface for app storage operations.
 * All operations are sandboxed within the storage root directory.
 * Path traversal attacks are blocked by path validation.
 */
export interface IAppStorage {
    /**
     * Get the storage root directory path.
     */
    getRoot(): Promise<string>;

    /**
     * Resolve a relative path to an absolute path within storage root.
     * Returns null if the path would escape the storage root.
     * @param relativePath - Path relative to storage root
     */
    resolvePath(relativePath: string): Promise<string | null>;

    /**
     * Ensure a directory exists, creating it if needed.
     * @param relativePath - Path relative to storage root
     */
    ensureDir(relativePath: string): Promise<void>;

    /**
     * Write data to a file.
     * Parent directories are created automatically.
     * @param relativePath - Path relative to storage root
     * @param data - String or Buffer data to write
     */
    writeFile(relativePath: string, data: string | Buffer): Promise<void>;

    /**
     * Read a file as a Buffer.
     * @param relativePath - Path relative to storage root
     * @returns Buffer contents, or null if file doesn't exist or path is invalid
     */
    readFile(relativePath: string): Promise<Buffer | null>;

    /**
     * Read a file as UTF-8 text.
     * @param relativePath - Path relative to storage root
     * @returns String contents, or null if file doesn't exist or path is invalid
     */
    readTextFile(relativePath: string): Promise<string | null>;

    /**
     * Delete a file.
     * @param relativePath - Path relative to storage root
     * @returns true if deleted, false if didn't exist or path invalid
     */
    deleteFile(relativePath: string): Promise<boolean>;

    /**
     * Delete a directory recursively.
     * @param relativePath - Path relative to storage root
     * @returns true if deleted, false if didn't exist or path invalid
     */
    deleteDir(relativePath: string): Promise<boolean>;

    /**
     * List directory contents.
     * @param relativePath - Path relative to storage root
     * @returns Array of entries, or null if directory doesn't exist or path invalid
     */
    listDir(relativePath: string): Promise<StorageEntry[] | null>;

    /**
     * Check if a path exists.
     * @param relativePath - Path relative to storage root
     * @returns true if exists, false otherwise
     */
    exists(relativePath: string): Promise<boolean>;

    /**
     * Get file or directory statistics.
     * @param relativePath - Path relative to storage root
     * @returns Stats object, or null if doesn't exist or path invalid
     */
    stat(relativePath: string): Promise<StorageStats | null>;

    /**
     * Get storage usage information.
     */
    getInfo(): Promise<StorageInfo>;
}
