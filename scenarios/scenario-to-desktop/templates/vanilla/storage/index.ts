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

export {
    clearUpdateIntent,
    consumeUpdateIntent,
    markUpdateIntentApplying,
    readUpdateIntent,
    writeInstalledUpdateReceipt,
    writeUpdateIntent,
    writeUpdateState,
} from "./update-recovery";

export type { InstalledUpdateReceipt, UpdateIntent, UpdateRecovery, UpdateRecoveryFileSystem, UpdateState, UpdateStateName } from "./update-recovery";
