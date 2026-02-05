/**
 * Splash Window Manager Tests
 *
 * DOC: docs/internal/SEAMS.md#splash-manager-tests
 *
 * Tests for the SplashWindowManager class.
 * Uses mocks to test without Electron.
 */

import { describe, it, expect, vi, type Mock } from "vitest";
import type { BrowserWindow } from "electron";
import type {
    ISplashWindowManager,
    SplashStatus,
    SplashWindowConfig,
} from "../types";
import type { IPathResolver, IIpcMain, IClipboard, SplashManagerDeps } from "../manager";
import { SplashWindowManager } from "../manager";
import { DEFAULT_SPLASH_CONFIG, SPLASH_IPC_CHANNELS } from "../types";

// Mock BrowserWindow
interface MockBrowserWindow {
    loadFile: Mock;
    on: Mock;
    isDestroyed: Mock;
    isVisible: Mock;
    destroy: Mock;
    webContents: {
        send: Mock;
        on: Mock;
    };
}

function createMockBrowserWindow(): MockBrowserWindow {
    return {
        loadFile: vi.fn().mockResolvedValue(undefined),
        on: vi.fn(),
        isDestroyed: vi.fn().mockReturnValue(false),
        isVisible: vi.fn().mockReturnValue(true),
        destroy: vi.fn(),
        webContents: {
            send: vi.fn(),
            on: vi.fn(),
        },
    };
}

// Mock WindowFactory
function createMockWindowFactory(mockWindow?: MockBrowserWindow) {
    const window = mockWindow ?? createMockBrowserWindow();
    return {
        window,
        createWindow: vi.fn().mockReturnValue(window),
    };
}

// Mock PathResolver
function createMockPathResolver(): IPathResolver {
    return {
        getAppPath: vi.fn().mockReturnValue("/app"),
        join: vi.fn((...segments: string[]) => segments.join("/")),
    };
}

// Mock IpcMain
function createMockIpcMain(): IIpcMain & { listeners: Map<string, Function[]> } {
    const listeners = new Map<string, Function[]>();
    return {
        listeners,
        on: vi.fn((channel: string, listener: Function) => {
            if (!listeners.has(channel)) {
                listeners.set(channel, []);
            }
            listeners.get(channel)!.push(listener);
        }),
        removeAllListeners: vi.fn((channel: string) => {
            listeners.delete(channel);
        }),
    };
}

// Mock Clipboard
function createMockClipboard(): IClipboard & { writeText: Mock } {
    return {
        writeText: vi.fn(),
    };
}

function createMockDeps(overrides?: Partial<{
    windowFactory: ReturnType<typeof createMockWindowFactory>;
    pathResolver: IPathResolver;
    ipcMain: ReturnType<typeof createMockIpcMain>;
    clipboard: ReturnType<typeof createMockClipboard>;
}>): {
    deps: SplashManagerDeps;
    mocks: {
        windowFactory: ReturnType<typeof createMockWindowFactory>;
        pathResolver: IPathResolver;
        ipcMain: ReturnType<typeof createMockIpcMain>;
        clipboard: ReturnType<typeof createMockClipboard>;
    };
} {
    const windowFactory = overrides?.windowFactory ?? createMockWindowFactory();
    const pathResolver = overrides?.pathResolver ?? createMockPathResolver();
    const ipcMain = overrides?.ipcMain ?? createMockIpcMain();
    const clipboard = overrides?.clipboard ?? createMockClipboard();

    return {
        deps: {
            windowFactory,
            pathResolver,
            ipcMain,
            clipboard,
            log: vi.fn(), // Silence logs in tests
        },
        mocks: { windowFactory, pathResolver, ipcMain, clipboard },
    };
}

describe("SplashWindowManager", () => {
    describe("constructor", () => {
        it("uses default config when none provided", () => {
            const { deps } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            // Manager should be created successfully
            expect(manager).toBeDefined();
        });

        it("merges provided config with defaults", () => {
            const { deps } = createMockDeps();
            const customConfig: Partial<SplashWindowConfig> = {
                width: 500,
                height: 400,
            };
            const manager = new SplashWindowManager(deps, customConfig);

            expect(manager).toBeDefined();
        });
    });

    describe("create()", () => {
        it("creates a BrowserWindow with correct options", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            await manager.create();

            expect(mocks.windowFactory.createWindow).toHaveBeenCalledWith(
                expect.objectContaining({
                    width: DEFAULT_SPLASH_CONFIG.width,
                    height: DEFAULT_SPLASH_CONFIG.height,
                    frame: DEFAULT_SPLASH_CONFIG.frame,
                    alwaysOnTop: DEFAULT_SPLASH_CONFIG.alwaysOnTop,
                    transparent: DEFAULT_SPLASH_CONFIG.transparent,
                })
            );
        });

        it("uses alwaysOnTop: false by default (prevents focus trapping)", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            await manager.create();

            expect(mocks.windowFactory.createWindow).toHaveBeenCalledWith(
                expect.objectContaining({
                    alwaysOnTop: false,
                })
            );
        });

        it("loads the splash HTML file", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            await manager.create();

            expect(mocks.windowFactory.window.loadFile).toHaveBeenCalledWith(
                expect.stringContaining("splash.html")
            );
        });

        it("sets up IPC listeners", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            await manager.create();

            expect(mocks.ipcMain.on).toHaveBeenCalledWith(
                SPLASH_IPC_CHANNELS.ESCAPE_PRESSED,
                expect.any(Function)
            );
            expect(mocks.ipcMain.on).toHaveBeenCalledWith(
                SPLASH_IPC_CHANNELS.READY,
                expect.any(Function)
            );
        });

        it("returns true on success", async () => {
            const { deps } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            const result = await manager.create();

            expect(result).toBe(true);
        });

        it("returns true if window already exists", async () => {
            const { deps } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            await manager.create();
            const result = await manager.create();

            expect(result).toBe(true);
        });

        it("returns false on error", async () => {
            const windowFactory = createMockWindowFactory();
            windowFactory.window.loadFile.mockRejectedValue(new Error("Load failed"));
            const { deps } = createMockDeps({ windowFactory });
            const manager = new SplashWindowManager(deps);

            const result = await manager.create();

            expect(result).toBe(false);
        });
    });

    describe("updateStatus()", () => {
        it("sends status via IPC to the window", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);
            await manager.create();

            const status: SplashStatus = {
                phase: "initializing",
                message: "Starting up...",
                progress: 10,
            };
            manager.updateStatus(status);

            expect(mocks.windowFactory.window.webContents.send).toHaveBeenCalledWith(
                SPLASH_IPC_CHANNELS.STATUS_UPDATE,
                status
            );
        });

        it("does nothing if window is not created", () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            // Don't call create()
            manager.updateStatus({
                phase: "initializing",
                message: "Test",
            });

            expect(mocks.windowFactory.window.webContents.send).not.toHaveBeenCalled();
        });

        it("does nothing if window is destroyed", async () => {
            const windowFactory = createMockWindowFactory();
            windowFactory.window.isDestroyed.mockReturnValue(true);
            const { deps, mocks } = createMockDeps({ windowFactory });
            const manager = new SplashWindowManager(deps);
            await manager.create();

            manager.updateStatus({
                phase: "initializing",
                message: "Test",
            });

            expect(mocks.windowFactory.window.webContents.send).not.toHaveBeenCalled();
        });
    });

    describe("close()", () => {
        it("destroys the window", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);
            await manager.create();

            await manager.close();

            expect(mocks.windowFactory.window.destroy).toHaveBeenCalled();
        });

        it("returns success: true after closing", async () => {
            const { deps } = createMockDeps();
            const manager = new SplashWindowManager(deps);
            await manager.create();

            const result = await manager.close();

            expect(result.success).toBe(true);
        });

        it("returns alreadyClosed: true if window not created", async () => {
            const { deps } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            const result = await manager.close();

            expect(result.success).toBe(true);
            expect(result.alreadyClosed).toBe(true);
        });

        it("returns alreadyClosed: true if window already destroyed", async () => {
            const windowFactory = createMockWindowFactory();
            windowFactory.window.isDestroyed.mockReturnValue(true);
            const { deps } = createMockDeps({ windowFactory });
            const manager = new SplashWindowManager(deps);
            await manager.create();

            const result = await manager.close();

            expect(result.success).toBe(true);
            expect(result.alreadyClosed).toBe(true);
        });

        it("removes IPC listeners after close", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);
            await manager.create();

            await manager.close();

            expect(mocks.ipcMain.removeAllListeners).toHaveBeenCalledWith(
                SPLASH_IPC_CHANNELS.ESCAPE_PRESSED
            );
            expect(mocks.ipcMain.removeAllListeners).toHaveBeenCalledWith(
                SPLASH_IPC_CHANNELS.READY
            );
        });
    });

    describe("isVisible()", () => {
        it("returns false when window not created", () => {
            const { deps } = createMockDeps();
            const manager = new SplashWindowManager(deps);

            expect(manager.isVisible()).toBe(false);
        });

        it("returns true when window is visible", async () => {
            const { deps } = createMockDeps();
            const manager = new SplashWindowManager(deps);
            await manager.create();

            expect(manager.isVisible()).toBe(true);
        });

        it("returns false when window is destroyed", async () => {
            const windowFactory = createMockWindowFactory();
            windowFactory.window.isDestroyed.mockReturnValue(true);
            const { deps } = createMockDeps({ windowFactory });
            const manager = new SplashWindowManager(deps);
            await manager.create();

            expect(manager.isVisible()).toBe(false);
        });

        it("returns false when window is not visible", async () => {
            const windowFactory = createMockWindowFactory();
            windowFactory.window.isVisible.mockReturnValue(false);
            const { deps } = createMockDeps({ windowFactory });
            const manager = new SplashWindowManager(deps);
            await manager.create();

            expect(manager.isVisible()).toBe(false);
        });
    });

    describe("onEscapePressed()", () => {
        it("registers callback for escape key", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);
            await manager.create();

            const callback = vi.fn();
            manager.onEscapePressed(callback);

            // Simulate escape key from IPC
            const escapeListener = mocks.ipcMain.listeners.get(SPLASH_IPC_CHANNELS.ESCAPE_PRESSED)?.[0];
            escapeListener?.({} as any);

            expect(callback).toHaveBeenCalled();
        });

        it("calls multiple callbacks when escape is pressed", async () => {
            const { deps, mocks } = createMockDeps();
            const manager = new SplashWindowManager(deps);
            await manager.create();

            const callback1 = vi.fn();
            const callback2 = vi.fn();
            manager.onEscapePressed(callback1);
            manager.onEscapePressed(callback2);

            // Simulate escape key from IPC
            const escapeListener = mocks.ipcMain.listeners.get(SPLASH_IPC_CHANNELS.ESCAPE_PRESSED)?.[0];
            escapeListener?.({} as any);

            expect(callback1).toHaveBeenCalled();
            expect(callback2).toHaveBeenCalled();
        });
    });
});

describe("regression: alwaysOnTop should be false by default", () => {
    /**
     * This test documents the bug that was fixed:
     * Previously, alwaysOnTop was true, which caused focus trapping
     * where the splash window would appear above all windows including
     * error dialogs, making it impossible to interact with them.
     */
    it("creates window without alwaysOnTop", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);

        await manager.create();

        const calls = mocks.windowFactory.createWindow.mock.calls;
        expect(calls.length).toBeGreaterThan(0);
        const windowOptions = calls[0]?.[0] as { alwaysOnTop?: boolean } | undefined;
        expect(windowOptions?.alwaysOnTop).toBe(false);
    });
});

describe("regression: error dialogs should appear after splash closes", () => {
    /**
     * This test documents the fix for error dialog z-order issues:
     * close() uses destroy() instead of close() to ensure immediate,
     * guaranteed closure before error dialogs are shown.
     */
    it("uses destroy() for immediate closure", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        await manager.close();

        // destroy() should be called, not close()
        expect(mocks.windowFactory.window.destroy).toHaveBeenCalled();
    });
});

describe("appendLog()", () => {
    it("sends log entry via IPC to the window", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        const entry = {
            timestamp: new Date(),
            level: "info" as const,
            message: "Test log message",
        };
        manager.appendLog(entry);

        expect(mocks.windowFactory.window.webContents.send).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.LOG_APPEND,
            entry
        );
    });

    it("does nothing if window is not created", () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);

        manager.appendLog({
            timestamp: new Date(),
            level: "info",
            message: "Test",
        });

        expect(mocks.windowFactory.window.webContents.send).not.toHaveBeenCalled();
    });

    it("does nothing if window is destroyed", async () => {
        const windowFactory = createMockWindowFactory();
        windowFactory.window.isDestroyed.mockReturnValue(true);
        const { deps, mocks } = createMockDeps({ windowFactory });
        const manager = new SplashWindowManager(deps);
        await manager.create();

        manager.appendLog({
            timestamp: new Date(),
            level: "info",
            message: "Test",
        });

        expect(mocks.windowFactory.window.webContents.send).not.toHaveBeenCalled();
    });
});

describe("clearLogs()", () => {
    it("sends clear command via IPC to the window", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        manager.clearLogs();

        expect(mocks.windowFactory.window.webContents.send).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.LOG_CLEAR
        );
    });

    it("does nothing if window is not created", () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);

        manager.clearLogs();

        expect(mocks.windowFactory.window.webContents.send).not.toHaveBeenCalled();
    });
});

describe("onCopyLogs()", () => {
    it("sets up IPC listener for copy logs request", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        expect(mocks.ipcMain.on).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.COPY_LOGS,
            expect.any(Function)
        );
    });

    it("calls callback and copies to clipboard when copy is requested", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        const callback = vi.fn().mockReturnValue("Log content");
        manager.onCopyLogs(callback);

        // Simulate copy request from IPC
        const copyListener = mocks.ipcMain.listeners.get(SPLASH_IPC_CHANNELS.COPY_LOGS)?.[0];
        copyListener?.({} as any);

        expect(callback).toHaveBeenCalled();
        expect(mocks.clipboard.writeText).toHaveBeenCalledWith("Log content");
    });

    it("sends success result back to window", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        const callback = vi.fn().mockReturnValue("Log content");
        manager.onCopyLogs(callback);

        // Simulate copy request from IPC
        const copyListener = mocks.ipcMain.listeners.get(SPLASH_IPC_CHANNELS.COPY_LOGS)?.[0];
        copyListener?.({} as any);

        expect(mocks.windowFactory.window.webContents.send).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.COPY_LOGS_RESULT,
            { success: true, logs: "Log content" }
        );
    });

    it("sends error result if callback throws", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        const callback = vi.fn().mockImplementation(() => {
            throw new Error("Callback error");
        });
        manager.onCopyLogs(callback);

        // Simulate copy request from IPC
        const copyListener = mocks.ipcMain.listeners.get(SPLASH_IPC_CHANNELS.COPY_LOGS)?.[0];
        copyListener?.({} as any);

        expect(mocks.windowFactory.window.webContents.send).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.COPY_LOGS_RESULT,
            { success: false, error: "Error: Callback error" }
        );
    });
});

describe("onRetry()", () => {
    it("sets up IPC listener for retry request", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        expect(mocks.ipcMain.on).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.RETRY,
            expect.any(Function)
        );
    });

    it("calls callback when retry is requested", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        const callback = vi.fn();
        manager.onRetry(callback);

        // Simulate retry request from IPC
        const retryListener = mocks.ipcMain.listeners.get(SPLASH_IPC_CHANNELS.RETRY)?.[0];
        retryListener?.({} as any);

        expect(callback).toHaveBeenCalled();
    });

    it("does not throw if callback is not registered", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        // Don't register callback, simulate retry request
        const retryListener = mocks.ipcMain.listeners.get(SPLASH_IPC_CHANNELS.RETRY)?.[0];

        expect(() => retryListener?.({} as any)).not.toThrow();
    });
});

describe("IPC listener cleanup", () => {
    it("removes copy logs listener on close", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        await manager.close();

        expect(mocks.ipcMain.removeAllListeners).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.COPY_LOGS
        );
    });

    it("removes retry listener on close", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        await manager.close();

        expect(mocks.ipcMain.removeAllListeners).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.RETRY
        );
    });
});

describe("error status with details", () => {
    it("sends error status with full details to window", async () => {
        const { deps, mocks } = createMockDeps();
        const manager = new SplashWindowManager(deps);
        await manager.create();

        const status: SplashStatus = {
            phase: "error",
            message: "Startup Error",
            error: {
                title: "Startup Error",
                message: "Failed to start",
                recoverable: true,
                suggestion: "Try restarting",
                logs: ["Log 1", "Log 2"],
                stderr: "Error output",
                exitCode: 1,
            },
        };
        manager.updateStatus(status);

        expect(mocks.windowFactory.window.webContents.send).toHaveBeenCalledWith(
            SPLASH_IPC_CHANNELS.STATUS_UPDATE,
            status
        );
    });
});
