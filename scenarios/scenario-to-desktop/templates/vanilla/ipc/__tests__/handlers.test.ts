/**
 * IPC Handlers Tests
 *
 * DOC: docs/internal/SEAMS.md#ipc-handlers-tests
 *
 * Tests for IPC handler registration and execution.
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import type {
    IIpcMain,
    IDialog,
    IFileFs,
    SystemInfo,
    FileHandlerDependencies,
    AppHandlerDependencies,
    StorageHandlerDependencies,
    AuthHandlerDependencies,
} from "../types";
import type { IAppStorage } from "../../storage";
import type { IAuthManager } from "../../auth";
import {
    registerFileHandlers,
    registerSystemHandlers,
    registerAppHandlers,
    registerStorageHandlers,
    registerAuthHandlers,
    registerAllHandlers,
} from "../handlers";
import {
    FILE_CHANNELS,
    SYSTEM_CHANNELS,
    APP_CHANNELS,
    STORAGE_CHANNELS,
    AUTH_CHANNELS,
} from "../channels";

// ===== Mock Factories =====

type HandlerFn = (event: unknown, ...args: unknown[]) => unknown;

interface MockIpcMain extends IIpcMain {
    _handlers: Map<string, HandlerFn>;
    _invoke(channel: string, ...args: unknown[]): Promise<unknown>;
}

function createMockIpcMain(): MockIpcMain {
    const handlers = new Map<string, HandlerFn>();

    return {
        _handlers: handlers,
        handle: vi.fn((channel: string, listener: HandlerFn) => {
            handlers.set(channel, listener);
        }),
        removeHandler: vi.fn((channel: string) => {
            handlers.delete(channel);
        }),
        async _invoke(channel: string, ...args: unknown[]): Promise<unknown> {
            const handler = handlers.get(channel);
            if (!handler) throw new Error(`No handler for channel: ${channel}`);
            return handler({}, ...args);
        },
    };
}

function createMockDialog(): IDialog & {
    _mockSaveResult: Electron.SaveDialogReturnValue;
    _mockOpenResult: Electron.OpenDialogReturnValue;
} {
    return {
        _mockSaveResult: { canceled: false, filePath: "/mock/file.txt" },
        _mockOpenResult: { canceled: false, filePaths: ["/mock/file.txt"] },
        showSaveDialog: vi.fn(async function(this: { _mockSaveResult: Electron.SaveDialogReturnValue }) {
            return this._mockSaveResult;
        }),
        showOpenDialog: vi.fn(async function(this: { _mockOpenResult: Electron.OpenDialogReturnValue }) {
            return this._mockOpenResult;
        }),
    };
}

function createMockFileFs(): IFileFs & { _files: Map<string, string> } {
    const files = new Map<string, string>();
    return {
        _files: files,
        writeFile: vi.fn(async (path: string, data: string) => {
            files.set(path, data);
        }),
        readFile: vi.fn(async (path: string) => {
            const content = files.get(path);
            if (content === undefined) throw new Error("ENOENT");
            return content;
        }),
    };
}

function createMockWindow(): Electron.BrowserWindow & {
    _minimized: boolean;
    _maximized: boolean;
    _closed: boolean;
} {
    return {
        _minimized: false,
        _maximized: false,
        _closed: false,
        minimize: vi.fn(function(this: { _minimized: boolean }) { this._minimized = true; }),
        maximize: vi.fn(function(this: { _maximized: boolean }) { this._maximized = true; }),
        unmaximize: vi.fn(function(this: { _maximized: boolean }) { this._maximized = false; }),
        isMaximized: vi.fn(function(this: { _maximized: boolean }) { return this._maximized; }),
        close: vi.fn(function(this: { _closed: boolean }) { this._closed = true; }),
    } as unknown as Electron.BrowserWindow & { _minimized: boolean; _maximized: boolean; _closed: boolean };
}

function createMockStorage(): IAppStorage & { _files: Map<string, string | Buffer> } {
    const files = new Map<string, string | Buffer>();
    return {
        _files: files,
        getRoot: vi.fn(async () => "/mock/storage"),
        resolvePath: vi.fn(async (path: string) => `/mock/storage/${path}`),
        ensureDir: vi.fn(async () => {}),
        writeFile: vi.fn(async (path: string, data: string | Buffer) => { files.set(path, data); }),
        readFile: vi.fn(async (path: string) => {
            const content = files.get(path);
            return content !== undefined ? (Buffer.isBuffer(content) ? content : Buffer.from(content)) : null;
        }),
        readTextFile: vi.fn(async (path: string) => {
            const content = files.get(path);
            return content !== undefined ? (Buffer.isBuffer(content) ? content.toString() : content) : null;
        }),
        deleteFile: vi.fn(async (path: string) => files.delete(path)),
        deleteDir: vi.fn(async (path: string) => files.delete(path)),
        listDir: vi.fn(async () => []),
        exists: vi.fn(async (path: string) => files.has(path)),
        stat: vi.fn(async () => ({ size: 100, createdAt: Date.now(), modifiedAt: Date.now(), isDirectory: false, isFile: true })),
        getInfo: vi.fn(async () => ({ used: 1000, available: 10000000, count: 5 })),
    };
}

function createMockAuthManager(): IAuthManager {
    return {
        signIn: vi.fn(async (opts?: { state?: string }) => ({ state: opts?.state ?? "test-state" })),
        signOut: vi.fn(async () => {}),
        getAccessToken: vi.fn(async () => "mock-access-token"),
        getEntitlementLease: vi.fn(async () => null),
        getUser: vi.fn(async () => ({ id: "user1", email: "test@example.com", emailVerified: true })),
        isAuthenticated: vi.fn(async () => true),
        refresh: vi.fn(async () => true),
        handleCallback: vi.fn(async () => {}),
        initialize: vi.fn(async () => {}),
        dispose: vi.fn(),
    };
}

// ===== Tests =====

describe("registerFileHandlers", () => {
    let ipcMain: MockIpcMain;
    let deps: FileHandlerDependencies & { dialog: ReturnType<typeof createMockDialog>; fs: ReturnType<typeof createMockFileFs> };

    beforeEach(() => {
        ipcMain = createMockIpcMain();
        deps = {
            dialog: createMockDialog(),
            fs: createMockFileFs(),
        };
    });

    it("registers file:save handler", () => {
        registerFileHandlers(ipcMain, deps);

        expect(ipcMain._handlers.has(FILE_CHANNELS.SAVE)).toBe(true);
    });

    it("registers file:open handler", () => {
        registerFileHandlers(ipcMain, deps);

        expect(ipcMain._handlers.has(FILE_CHANNELS.OPEN)).toBe(true);
    });

    it("file:save writes content to selected file", async () => {
        registerFileHandlers(ipcMain, deps);

        const result = await ipcMain._invoke(FILE_CHANNELS.SAVE, { content: "Hello", defaultPath: "test.txt" });

        expect(result).toEqual({ saved: true, filePath: "/mock/file.txt" });
        expect(deps.fs._files.get("/mock/file.txt")).toBe("Hello");
    });

    it("file:save returns canceled when dialog canceled", async () => {
        deps.dialog._mockSaveResult = { canceled: true, filePath: "" };
        registerFileHandlers(ipcMain, deps);

        const result = await ipcMain._invoke(FILE_CHANNELS.SAVE, { content: "Hello" });

        expect(result).toEqual({ saved: false });
    });

    it("file:open reads content from selected file", async () => {
        deps.fs._files.set("/mock/file.txt", "File content");
        registerFileHandlers(ipcMain, deps);

        const result = await ipcMain._invoke(FILE_CHANNELS.OPEN) as { opened: boolean; content?: string };

        expect(result.opened).toBe(true);
        expect(result.content).toBe("File content");
    });

    it("unregister removes all handlers", () => {
        const registration = registerFileHandlers(ipcMain, deps);

        registration.unregister();

        expect(ipcMain._handlers.has(FILE_CHANNELS.SAVE)).toBe(false);
        expect(ipcMain._handlers.has(FILE_CHANNELS.OPEN)).toBe(false);
    });
});

describe("registerSystemHandlers", () => {
    let ipcMain: MockIpcMain;
    let systemInfo: SystemInfo;

    beforeEach(() => {
        ipcMain = createMockIpcMain();
        systemInfo = { platform: "linux", arch: "x64", version: "1.0.0" };
    });

    it("registers system:info handler", () => {
        registerSystemHandlers(ipcMain, systemInfo);

        expect(ipcMain._handlers.has(SYSTEM_CHANNELS.INFO)).toBe(true);
    });

    it("system:info returns system information", async () => {
        registerSystemHandlers(ipcMain, systemInfo);

        const result = await ipcMain._invoke(SYSTEM_CHANNELS.INFO);

        expect(result).toEqual({ platform: "linux", arch: "x64", version: "1.0.0" });
    });
});

describe("registerAppHandlers", () => {
    let ipcMain: MockIpcMain;
    let mockWindow: ReturnType<typeof createMockWindow>;
    let deps: AppHandlerDependencies;

    beforeEach(() => {
        ipcMain = createMockIpcMain();
        mockWindow = createMockWindow();
        deps = { getMainWindow: () => mockWindow };
    });

    it("app:minimize minimizes the window", async () => {
        registerAppHandlers(ipcMain, deps);

        await ipcMain._invoke(APP_CHANNELS.MINIMIZE);

        expect(mockWindow.minimize).toHaveBeenCalled();
    });

    it("app:maximize toggles maximize state", async () => {
        registerAppHandlers(ipcMain, deps);

        // First call maximizes
        await ipcMain._invoke(APP_CHANNELS.MAXIMIZE);
        expect(mockWindow.maximize).toHaveBeenCalled();

        // Second call unmaximizes (because isMaximized returns true now)
        mockWindow._maximized = true;
        await ipcMain._invoke(APP_CHANNELS.MAXIMIZE);
        expect(mockWindow.unmaximize).toHaveBeenCalled();
    });

    it("app:close closes the window", async () => {
        registerAppHandlers(ipcMain, deps);

        await ipcMain._invoke(APP_CHANNELS.CLOSE);

        expect(mockWindow.close).toHaveBeenCalled();
    });

    it("handles null window gracefully", async () => {
        deps = { getMainWindow: () => null };
        registerAppHandlers(ipcMain, deps);

        // Should not throw
        await expect(ipcMain._invoke(APP_CHANNELS.MINIMIZE)).resolves.toBeUndefined();
    });
});

describe("registerStorageHandlers", () => {
    let ipcMain: MockIpcMain;
    let deps: StorageHandlerDependencies & { storage: ReturnType<typeof createMockStorage> };

    beforeEach(() => {
        ipcMain = createMockIpcMain();
        deps = { storage: createMockStorage() };
    });

    it("registers all storage handlers", () => {
        registerStorageHandlers(ipcMain, deps);

        expect(ipcMain._handlers.has(STORAGE_CHANNELS.GET_PATH)).toBe(true);
        expect(ipcMain._handlers.has(STORAGE_CHANNELS.WRITE_FILE)).toBe(true);
        expect(ipcMain._handlers.has(STORAGE_CHANNELS.READ_FILE)).toBe(true);
        expect(ipcMain._handlers.has(STORAGE_CHANNELS.DELETE_FILE)).toBe(true);
        expect(ipcMain._handlers.has(STORAGE_CHANNELS.EXISTS)).toBe(true);
    });

    it("storage:get-path returns storage root", async () => {
        registerStorageHandlers(ipcMain, deps);

        const result = await ipcMain._invoke(STORAGE_CHANNELS.GET_PATH);

        expect(result).toBe("/mock/storage");
    });

    it("storage:write-file writes string data", async () => {
        registerStorageHandlers(ipcMain, deps);

        await ipcMain._invoke(STORAGE_CHANNELS.WRITE_FILE, { path: "test.txt", data: "Hello", isBinary: false });

        expect(deps.storage._files.get("test.txt")).toBe("Hello");
    });

    it("storage:write-file writes binary data", async () => {
        registerStorageHandlers(ipcMain, deps);

        await ipcMain._invoke(STORAGE_CHANNELS.WRITE_FILE, { path: "test.bin", data: [0, 1, 2], isBinary: true });

        const written = deps.storage._files.get("test.bin");
        expect(Buffer.isBuffer(written)).toBe(true);
        expect(Array.from(written as Buffer)).toEqual([0, 1, 2]);
    });

    it("storage:read-file returns number array", async () => {
        deps.storage._files.set("test.txt", Buffer.from([65, 66, 67])); // "ABC"
        registerStorageHandlers(ipcMain, deps);

        const result = await ipcMain._invoke(STORAGE_CHANNELS.READ_FILE, "test.txt");

        expect(result).toEqual([65, 66, 67]);
    });

    it("storage:exists returns boolean", async () => {
        deps.storage._files.set("exists.txt", "content");
        registerStorageHandlers(ipcMain, deps);

        const exists = await ipcMain._invoke(STORAGE_CHANNELS.EXISTS, "exists.txt");
        const notExists = await ipcMain._invoke(STORAGE_CHANNELS.EXISTS, "missing.txt");

        expect(exists).toBe(true);
        expect(notExists).toBe(false);
    });
});

describe("registerAuthHandlers", () => {
    let ipcMain: MockIpcMain;
    let deps: AuthHandlerDependencies;

    beforeEach(() => {
        ipcMain = createMockIpcMain();
        deps = { authManager: createMockAuthManager() };
    });

    it("registers all auth handlers", () => {
        registerAuthHandlers(ipcMain, deps);

        expect(ipcMain._handlers.has(AUTH_CHANNELS.SIGN_IN)).toBe(true);
        expect(ipcMain._handlers.has(AUTH_CHANNELS.SIGN_OUT)).toBe(true);
        expect(ipcMain._handlers.has(AUTH_CHANNELS.GET_ACCESS_TOKEN)).toBe(true);
        expect(ipcMain._handlers.has(AUTH_CHANNELS.GET_USER)).toBe(true);
        expect(ipcMain._handlers.has(AUTH_CHANNELS.IS_AUTHENTICATED)).toBe(true);
        expect(ipcMain._handlers.has(AUTH_CHANNELS.REFRESH)).toBe(true);
    });

    it("auth:sign-in calls authManager.signIn", async () => {
        registerAuthHandlers(ipcMain, deps);

        const result = await ipcMain._invoke(AUTH_CHANNELS.SIGN_IN, { state: "my-state" });

        expect(deps.authManager.signIn).toHaveBeenCalledWith({ state: "my-state" });
        expect(result).toEqual({ state: "my-state" });
    });

    it("auth:get-access-token returns token", async () => {
        registerAuthHandlers(ipcMain, deps);

        const result = await ipcMain._invoke(AUTH_CHANNELS.GET_ACCESS_TOKEN);

        expect(result).toBe("mock-access-token");
    });

    it("auth:is-authenticated returns boolean", async () => {
        registerAuthHandlers(ipcMain, deps);

        const result = await ipcMain._invoke(AUTH_CHANNELS.IS_AUTHENTICATED);

        expect(result).toBe(true);
    });
});

describe("registerAllHandlers", () => {
    it("registers all handler groups", () => {
        const ipcMain = createMockIpcMain();
        const deps = {
            ipcMain,
            dialog: createMockDialog(),
            fs: createMockFileFs(),
            storage: createMockStorage(),
            authManager: createMockAuthManager(),
            getMainWindow: () => createMockWindow(),
            systemInfo: { platform: "linux" as const, arch: "x64", version: "1.0.0" },
        };

        const registration = registerAllHandlers(deps);

        // Check that handlers from all groups are registered
        expect(ipcMain._handlers.has(FILE_CHANNELS.SAVE)).toBe(true);
        expect(ipcMain._handlers.has(SYSTEM_CHANNELS.INFO)).toBe(true);
        expect(ipcMain._handlers.has(APP_CHANNELS.MINIMIZE)).toBe(true);
        expect(ipcMain._handlers.has(STORAGE_CHANNELS.GET_PATH)).toBe(true);
        expect(ipcMain._handlers.has(AUTH_CHANNELS.SIGN_IN)).toBe(true);

        // Check channels list
        expect(registration.channels.length).toBeGreaterThan(10);
    });

    it("unregister removes all handlers", () => {
        const ipcMain = createMockIpcMain();
        const deps = {
            ipcMain,
            dialog: createMockDialog(),
            fs: createMockFileFs(),
            storage: createMockStorage(),
            authManager: createMockAuthManager(),
            getMainWindow: () => null,
            systemInfo: { platform: "linux" as const, arch: "x64", version: "1.0.0" },
        };

        const registration = registerAllHandlers(deps);
        registration.unregister();

        expect(ipcMain._handlers.size).toBe(0);
    });
});
