/**
 * IPC Channel Constants
 *
 * DOC: docs/internal/SEAMS.md#ipc-channels
 *
 * Centralized channel name constants for type-safe IPC communication.
 */

// ===== File Channels =====

export const FILE_CHANNELS = {
    SAVE: "file:save",
    OPEN: "file:open",
} as const;

// ===== System Channels =====

export const SYSTEM_CHANNELS = {
    INFO: "system:info",
} as const;

// ===== App Channels =====

export const APP_CHANNELS = {
    MINIMIZE: "app:minimize",
    MAXIMIZE: "app:maximize",
    CLOSE: "app:close",
} as const;

// ===== Storage Channels =====

export const STORAGE_CHANNELS = {
    GET_PATH: "storage:get-path",
    ENSURE_DIR: "storage:ensure-dir",
    WRITE_FILE: "storage:write-file",
    READ_FILE: "storage:read-file",
    READ_TEXT_FILE: "storage:read-text-file",
    DELETE_FILE: "storage:delete-file",
    DELETE_DIR: "storage:delete-dir",
    LIST_DIR: "storage:list-dir",
    EXISTS: "storage:exists",
    STAT: "storage:stat",
    GET_INFO: "storage:get-info",
} as const;

// ===== Auth Channels =====

export const AUTH_CHANNELS = {
    SIGN_IN: "auth:sign-in",
    SIGN_OUT: "auth:sign-out",
    GET_ACCESS_TOKEN: "auth:get-access-token",
    GET_USER: "auth:get-user",
    IS_AUTHENTICATED: "auth:is-authenticated",
    REFRESH: "auth:refresh",
    CHANGED: "auth:changed", // Event sent to renderer
} as const;

// ===== All Channel Names =====

export const ALL_CHANNELS = {
    ...FILE_CHANNELS,
    ...SYSTEM_CHANNELS,
    ...APP_CHANNELS,
    ...STORAGE_CHANNELS,
    ...AUTH_CHANNELS,
} as const;

export type ChannelName = typeof ALL_CHANNELS[keyof typeof ALL_CHANNELS];
