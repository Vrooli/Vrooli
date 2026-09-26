/**
 * IPC Module
 *
 * DOC: docs/internal/SEAMS.md#ipc-module
 *
 * Barrel exports for IPC handler registration.
 */

// Channel Constants
export {
    FILE_CHANNELS,
    SYSTEM_CHANNELS,
    APP_CHANNELS,
    STORAGE_CHANNELS,
    AUTH_CHANNELS,
    ALL_CHANNELS,
    type ChannelName,
} from "./channels";

// Types
export type {
    IIpcMain,
    IDialog,
    IFileFs,
    SystemInfo,
    FileSavePayload,
    StorageWritePayload,
    FileHandlerDependencies,
    AppHandlerDependencies,
    StorageHandlerDependencies,
    AuthHandlerDependencies,
    IpcHandlerDependencies,
    HandlerRegistration,
} from "./types";

// Handlers
export {
    registerFileHandlers,
    registerSystemHandlers,
    registerAppHandlers,
    registerStorageHandlers,
    registerAuthHandlers,
    registerAllHandlers,
} from "./handlers";
