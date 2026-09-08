/**
 * IPC Module Types
 *
 * DOC: docs/internal/SEAMS.md#ipc-types
 *
 * Type definitions for IPC handler registration.
 */

import type { IpcMain, BrowserWindow, Dialog, Shell } from "electron";
import type { IAppStorage, StorageEntry, StorageStats, StorageInfo } from "../storage";
import type { IAuthManager } from "../auth";

// ===== IPC Handler Types =====

/**
 * IpcMain interface subset needed by handlers.
 * Uses `any` for args to allow typed handlers while maintaining Electron compatibility.
 */
export interface IIpcMain {
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    handle(channel: string, listener: (event: Electron.IpcMainInvokeEvent, ...args: any[]) => any): void;
    removeHandler(channel: string): void;
}

/**
 * Dialog interface for file operations.
 */
export interface IDialog {
    showSaveDialog(options: Electron.SaveDialogOptions): Promise<Electron.SaveDialogReturnValue>;
    showOpenDialog(options: Electron.OpenDialogOptions): Promise<Electron.OpenDialogReturnValue>;
}

/**
 * Filesystem operations for file handlers.
 */
export interface IFileFs {
    writeFile(path: string, data: string): Promise<void>;
    readFile(path: string, encoding: "utf-8"): Promise<string>;
}

/**
 * System information interface.
 */
export interface SystemInfo {
    platform: NodeJS.Platform;
    arch: string;
    version: string;
}

// ===== Handler Payloads =====

/**
 * Payload for file:save operation.
 */
export interface FileSavePayload {
    content: string;
    defaultPath?: string;
}

/**
 * Payload for storage:write-file operation.
 */
export interface StorageWritePayload {
    path: string;
    data: string | number[];
    isBinary: boolean;
}

// ===== Dependencies =====

/**
 * Dependencies for file handlers.
 */
export interface FileHandlerDependencies {
    dialog: IDialog;
    fs: IFileFs;
}

/**
 * Dependencies for app handlers.
 */
export interface AppHandlerDependencies {
    getMainWindow: () => BrowserWindow | null;
}

/**
 * Dependencies for storage handlers.
 */
export interface StorageHandlerDependencies {
    storage: IAppStorage;
}

/**
 * Dependencies for auth handlers.
 */
export interface AuthHandlerDependencies {
    authManager: IAuthManager;
}

/**
 * Combined dependencies for all handlers.
 */
export interface IpcHandlerDependencies {
    ipcMain: IIpcMain;
    dialog: IDialog;
    fs: IFileFs;
    storage: IAppStorage;
    authManager: IAuthManager;
    getMainWindow: () => BrowserWindow | null;
    systemInfo: SystemInfo;
}

// ===== Handler Registration Result =====

/**
 * Result of handler registration.
 * Returns a cleanup function to remove all handlers.
 */
export interface HandlerRegistration {
    /** Remove all registered handlers */
    unregister: () => void;
    /** List of registered channel names */
    channels: string[];
}
