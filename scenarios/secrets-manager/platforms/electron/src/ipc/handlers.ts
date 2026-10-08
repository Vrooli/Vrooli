/**
 * IPC Handlers Implementation
 *
 * DOC: docs/internal/SEAMS.md#ipc-handlers
 *
 * Registration functions for IPC handlers organized by domain.
 */

import {
    FILE_CHANNELS,
    SYSTEM_CHANNELS,
    APP_CHANNELS,
    STORAGE_CHANNELS,
    AUTH_CHANNELS,
} from "./channels";

import type {
    IpcHandlerDependencies,
    HandlerRegistration,
    IIpcMain,
    FileSavePayload,
    StorageWritePayload,
    FileHandlerDependencies,
    AppHandlerDependencies,
    StorageHandlerDependencies,
    AuthHandlerDependencies,
    SystemInfo,
} from "./types";

/**
 * Register file operation handlers.
 */
export function registerFileHandlers(
    ipcMain: IIpcMain,
    deps: FileHandlerDependencies
): HandlerRegistration {
    const channels: string[] = [];

    // file:save - Save content to a file
    ipcMain.handle(FILE_CHANNELS.SAVE, async (_event, data: FileSavePayload) => {
        const dialogOptions: Electron.SaveDialogOptions = {
            filters: [
                { name: "Text Files", extensions: ["txt", "md"] },
                { name: "All Files", extensions: ["*"] },
            ],
        };
        // Only include defaultPath if it has a value (exactOptionalPropertyTypes)
        if (data.defaultPath) {
            dialogOptions.defaultPath = data.defaultPath;
        }
        const result = await deps.dialog.showSaveDialog(dialogOptions);

        if (result.canceled || !result.filePath) {
            return { saved: false };
        }

        await deps.fs.writeFile(result.filePath, data.content);
        return { saved: true, filePath: result.filePath };
    });
    channels.push(FILE_CHANNELS.SAVE);

    // file:open - Open a file and read its content
    ipcMain.handle(FILE_CHANNELS.OPEN, async () => {
        const result = await deps.dialog.showOpenDialog({
            properties: ["openFile"],
            filters: [
                { name: "Text Files", extensions: ["txt", "md", "json"] },
                { name: "All Files", extensions: ["*"] },
            ],
        });

        if (result.canceled || result.filePaths.length === 0) {
            return { opened: false };
        }

        const filePath = result.filePaths[0];
        if (!filePath) {
            return { opened: false };
        }

        const content = await deps.fs.readFile(filePath, "utf-8");
        return { opened: true, filePath, content };
    });
    channels.push(FILE_CHANNELS.OPEN);

    return {
        unregister: () => channels.forEach(ch => ipcMain.removeHandler(ch)),
        channels,
    };
}

/**
 * Register system information handlers.
 */
export function registerSystemHandlers(
    ipcMain: IIpcMain,
    systemInfo: SystemInfo
): HandlerRegistration {
    const channels: string[] = [];

    // system:info - Get system information
    ipcMain.handle(SYSTEM_CHANNELS.INFO, () => ({
        platform: systemInfo.platform,
        arch: systemInfo.arch,
        version: systemInfo.version,
    }));
    channels.push(SYSTEM_CHANNELS.INFO);

    return {
        unregister: () => channels.forEach(ch => ipcMain.removeHandler(ch)),
        channels,
    };
}

/**
 * Register app window control handlers.
 */
export function registerAppHandlers(
    ipcMain: IIpcMain,
    deps: AppHandlerDependencies
): HandlerRegistration {
    const channels: string[] = [];

    // app:minimize - Minimize the main window
    ipcMain.handle(APP_CHANNELS.MINIMIZE, () => {
        const win = deps.getMainWindow();
        win?.minimize();
    });
    channels.push(APP_CHANNELS.MINIMIZE);

    // app:maximize - Toggle maximize state
    ipcMain.handle(APP_CHANNELS.MAXIMIZE, () => {
        const win = deps.getMainWindow();
        if (win?.isMaximized()) {
            win.unmaximize();
        } else {
            win?.maximize();
        }
    });
    channels.push(APP_CHANNELS.MAXIMIZE);

    // app:close - Close the main window
    ipcMain.handle(APP_CHANNELS.CLOSE, () => {
        const win = deps.getMainWindow();
        win?.close();
    });
    channels.push(APP_CHANNELS.CLOSE);

    return {
        unregister: () => channels.forEach(ch => ipcMain.removeHandler(ch)),
        channels,
    };
}

/**
 * Register storage operation handlers.
 */
export function registerStorageHandlers(
    ipcMain: IIpcMain,
    deps: StorageHandlerDependencies
): HandlerRegistration {
    const channels: string[] = [];
    const { storage } = deps;

    // storage:get-path - Get the storage root path
    ipcMain.handle(STORAGE_CHANNELS.GET_PATH, async () => {
        return storage.getRoot();
    });
    channels.push(STORAGE_CHANNELS.GET_PATH);

    // storage:ensure-dir - Ensure a directory exists
    ipcMain.handle(STORAGE_CHANNELS.ENSURE_DIR, async (_event, relativePath: string) => {
        await storage.ensureDir(relativePath);
    });
    channels.push(STORAGE_CHANNELS.ENSURE_DIR);

    // storage:write-file - Write a file (string or binary)
    ipcMain.handle(STORAGE_CHANNELS.WRITE_FILE, async (_event, data: StorageWritePayload) => {
        if (data.isBinary && Array.isArray(data.data)) {
            // Binary data comes as number array, convert to Buffer
            await storage.writeFile(data.path, Buffer.from(data.data));
        } else {
            // String data
            await storage.writeFile(data.path, data.data as string);
        }
    });
    channels.push(STORAGE_CHANNELS.WRITE_FILE);

    // storage:read-file - Read file as binary (returns number array for IPC)
    ipcMain.handle(STORAGE_CHANNELS.READ_FILE, async (_event, relativePath: string) => {
        const buffer = await storage.readFile(relativePath);
        if (!buffer) return null;
        // Convert Buffer to number array for IPC serialization
        return Array.from(buffer);
    });
    channels.push(STORAGE_CHANNELS.READ_FILE);

    // storage:read-text-file - Read file as UTF-8 text
    ipcMain.handle(STORAGE_CHANNELS.READ_TEXT_FILE, async (_event, relativePath: string) => {
        return storage.readTextFile(relativePath);
    });
    channels.push(STORAGE_CHANNELS.READ_TEXT_FILE);

    // storage:delete-file - Delete a file
    ipcMain.handle(STORAGE_CHANNELS.DELETE_FILE, async (_event, relativePath: string) => {
        return storage.deleteFile(relativePath);
    });
    channels.push(STORAGE_CHANNELS.DELETE_FILE);

    // storage:delete-dir - Delete a directory recursively
    ipcMain.handle(STORAGE_CHANNELS.DELETE_DIR, async (_event, relativePath: string) => {
        return storage.deleteDir(relativePath);
    });
    channels.push(STORAGE_CHANNELS.DELETE_DIR);

    // storage:list-dir - List directory contents
    ipcMain.handle(STORAGE_CHANNELS.LIST_DIR, async (_event, relativePath: string) => {
        return storage.listDir(relativePath);
    });
    channels.push(STORAGE_CHANNELS.LIST_DIR);

    // storage:exists - Check if path exists
    ipcMain.handle(STORAGE_CHANNELS.EXISTS, async (_event, relativePath: string) => {
        return storage.exists(relativePath);
    });
    channels.push(STORAGE_CHANNELS.EXISTS);

    // storage:stat - Get file/directory stats
    ipcMain.handle(STORAGE_CHANNELS.STAT, async (_event, relativePath: string) => {
        return storage.stat(relativePath);
    });
    channels.push(STORAGE_CHANNELS.STAT);

    // storage:get-info - Get storage usage statistics
    ipcMain.handle(STORAGE_CHANNELS.GET_INFO, async () => {
        return storage.getInfo();
    });
    channels.push(STORAGE_CHANNELS.GET_INFO);

    return {
        unregister: () => channels.forEach(ch => ipcMain.removeHandler(ch)),
        channels,
    };
}

/**
 * Register authentication handlers.
 */
export function registerAuthHandlers(
    ipcMain: IIpcMain,
    deps: AuthHandlerDependencies
): HandlerRegistration {
    const channels: string[] = [];
    const { authManager } = deps;

    // auth:sign-in - Start sign-in flow
    ipcMain.handle(AUTH_CHANNELS.SIGN_IN, async (_event, options?: { state?: string }) => {
        return authManager.signIn(options);
    });
    channels.push(AUTH_CHANNELS.SIGN_IN);

    // auth:sign-out - Sign out
    ipcMain.handle(AUTH_CHANNELS.SIGN_OUT, async () => {
        await authManager.signOut();
    });
    channels.push(AUTH_CHANNELS.SIGN_OUT);

    // auth:get-access-token - Get current access token
    ipcMain.handle(AUTH_CHANNELS.GET_ACCESS_TOKEN, async () => {
        return authManager.getAccessToken();
    });
    channels.push(AUTH_CHANNELS.GET_ACCESS_TOKEN);

    // auth:get-user - Get user info
    ipcMain.handle(AUTH_CHANNELS.GET_USER, async () => {
        return authManager.getUser();
    });
    channels.push(AUTH_CHANNELS.GET_USER);

    // auth:is-authenticated - Check if authenticated
    ipcMain.handle(AUTH_CHANNELS.IS_AUTHENTICATED, async () => {
        return authManager.isAuthenticated();
    });
    channels.push(AUTH_CHANNELS.IS_AUTHENTICATED);

    // auth:refresh - Force token refresh
    ipcMain.handle(AUTH_CHANNELS.REFRESH, async () => {
        return authManager.refresh();
    });
    channels.push(AUTH_CHANNELS.REFRESH);

    return {
        unregister: () => channels.forEach(ch => ipcMain.removeHandler(ch)),
        channels,
    };
}

/**
 * Register all IPC handlers.
 */
export function registerAllHandlers(deps: IpcHandlerDependencies): HandlerRegistration {
    const registrations = [
        registerFileHandlers(deps.ipcMain, { dialog: deps.dialog, fs: deps.fs }),
        registerSystemHandlers(deps.ipcMain, deps.systemInfo),
        registerAppHandlers(deps.ipcMain, { getMainWindow: deps.getMainWindow }),
        registerStorageHandlers(deps.ipcMain, { storage: deps.storage }),
        registerAuthHandlers(deps.ipcMain, { authManager: deps.authManager }),
    ];

    const allChannels = registrations.flatMap(r => r.channels);

    return {
        unregister: () => registrations.forEach(r => r.unregister()),
        channels: allChannels,
    };
}
