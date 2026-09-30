/**
 * Storage Module
 *
 * DOC: docs/internal/SEAMS.md#storage-module
 *
 * Barrel exports for the app storage system.
 */

// Types
export type {
    StorageEntry,
    StorageStats,
    StorageInfo,
    DirentLike,
    StatsLike,
    IStorageFileSystem,
    IStoragePathUtils,
    StorageConfig,
    IAppStorage,
} from "./types";

// Implementation
export {
    createAppStorage,
    createNodeStorageFileSystem,
    createNodeStoragePathUtils,
} from "./app-storage";
