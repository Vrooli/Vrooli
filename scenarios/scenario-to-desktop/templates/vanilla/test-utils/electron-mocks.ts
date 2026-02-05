/**
 * Electron Module Mocks
 *
 * DOC: docs/internal/SEAMS.md#electron-mocks
 *
 * Provides mock factories for Electron modules used in main process testing.
 * These mocks allow testing without requiring an Electron runtime.
 *
 * Usage:
 * ```typescript
 * import { createMockApp, createMockBrowserWindow } from "../test-utils/electron-mocks";
 *
 * const app = createMockApp();
 * const window = createMockBrowserWindow();
 * ```
 */

import { vi, type Mock } from "vitest";

// ===== Type Definitions =====

/**
 * Mock BrowserWindow instance with full tracking capabilities.
 */
export interface MockBrowserWindow {
    // Window state
    loadFile: Mock;
    loadURL: Mock;
    show: Mock;
    hide: Mock;
    close: Mock;
    destroy: Mock;
    minimize: Mock;
    maximize: Mock;
    unmaximize: Mock;
    restore: Mock;
    setFullScreen: Mock;
    focus: Mock;

    // State queries
    isDestroyed: Mock;
    isVisible: Mock;
    isMinimized: Mock;
    isMaximized: Mock;
    isFullScreen: Mock;
    isFocused: Mock;
    getBounds: Mock;
    getNormalBounds: Mock;

    // Web contents
    webContents: MockWebContents;

    // Event handling
    on: Mock;
    once: Mock;
    removeListener: Mock;
    removeAllListeners: Mock;

    // Internal tracking for tests
    _eventHandlers: Map<string, Set<Function>>;
    _simulateEvent: (event: string, ...args: unknown[]) => void;
    _state: {
        destroyed: boolean;
        visible: boolean;
        minimized: boolean;
        maximized: boolean;
        fullScreen: boolean;
        focused: boolean;
        bounds: { x: number; y: number; width: number; height: number };
    };
}

/**
 * Mock WebContents instance.
 */
export interface MockWebContents {
    send: Mock;
    on: Mock;
    once: Mock;
    removeListener: Mock;
    removeAllListeners: Mock;
    openDevTools: Mock;
    closeDevTools: Mock;
    setWindowOpenHandler: Mock;
    _eventHandlers: Map<string, Set<Function>>;
    _simulateEvent: (event: string, ...args: unknown[]) => void;
}

/**
 * Mock App module.
 */
export interface MockApp {
    whenReady: Mock;
    isReady: Mock;
    quit: Mock;
    exit: Mock;
    relaunch: Mock;
    isPackaged: boolean;
    getPath: Mock;
    getAppPath: Mock;
    getName: Mock;
    getVersion: Mock;
    requestSingleInstanceLock: Mock;
    setAsDefaultProtocolClient: Mock;
    on: Mock;
    once: Mock;
    removeListener: Mock;
    _eventHandlers: Map<string, Set<Function>>;
    _simulateEvent: (event: string, ...args: unknown[]) => void;
    _paths: Map<string, string>;
}

/**
 * Mock IpcMain module.
 */
export interface MockIpcMain {
    handle: Mock;
    handleOnce: Mock;
    removeHandler: Mock;
    on: Mock;
    once: Mock;
    removeListener: Mock;
    removeAllListeners: Mock;
    _handlers: Map<string, Function>;
    _eventHandlers: Map<string, Set<Function>>;
    _simulateInvoke: <T>(channel: string, event: unknown, ...args: unknown[]) => Promise<T>;
    _simulateEvent: (channel: string, event: unknown, ...args: unknown[]) => void;
}

/**
 * Mock Dialog module.
 */
export interface MockDialog {
    showMessageBox: Mock;
    showErrorBox: Mock;
    showOpenDialog: Mock;
    showSaveDialog: Mock;
}

/**
 * Mock Shell module.
 */
export interface MockShell {
    openExternal: Mock;
    openPath: Mock;
}

/**
 * Mock SafeStorage module.
 */
export interface MockSafeStorage {
    isEncryptionAvailable: Mock;
    encryptString: Mock;
    decryptString: Mock;
}

/**
 * Mock Screen module.
 */
export interface MockScreen {
    getAllDisplays: Mock;
    getPrimaryDisplay: Mock;
    getDisplayNearestPoint: Mock;
}

/**
 * Mock clipboard module.
 */
export interface MockClipboard {
    writeText: Mock;
    readText: Mock;
}

/**
 * Mock nativeImage module.
 */
export interface MockNativeImage {
    createFromPath: Mock;
    _mockImage: {
        resize: Mock;
        isEmpty: Mock;
    };
}

// ===== Factory Functions =====

/**
 * Create a mock WebContents instance.
 */
export function createMockWebContents(): MockWebContents {
    const eventHandlers = new Map<string, Set<Function>>();

    const mockWebContents: MockWebContents = {
        send: vi.fn(),
        on: vi.fn((event: string, handler: Function) => {
            if (!eventHandlers.has(event)) {
                eventHandlers.set(event, new Set());
            }
            eventHandlers.get(event)!.add(handler);
            return mockWebContents;
        }),
        once: vi.fn((event: string, handler: Function) => {
            if (!eventHandlers.has(event)) {
                eventHandlers.set(event, new Set());
            }
            const wrappedHandler = (...args: unknown[]) => {
                eventHandlers.get(event)?.delete(wrappedHandler);
                handler(...args);
            };
            eventHandlers.get(event)!.add(wrappedHandler);
            return mockWebContents;
        }),
        removeListener: vi.fn((event: string, handler: Function) => {
            eventHandlers.get(event)?.delete(handler);
            return mockWebContents;
        }),
        removeAllListeners: vi.fn((event?: string) => {
            if (event) {
                eventHandlers.delete(event);
            } else {
                eventHandlers.clear();
            }
            return mockWebContents;
        }),
        openDevTools: vi.fn(),
        closeDevTools: vi.fn(),
        setWindowOpenHandler: vi.fn(),
        _eventHandlers: eventHandlers,
        _simulateEvent: (event: string, ...args: unknown[]) => {
            eventHandlers.get(event)?.forEach((handler) => handler(...args));
        },
    };

    return mockWebContents;
}

/**
 * Create a mock BrowserWindow instance.
 */
export function createMockBrowserWindow(options?: {
    bounds?: { x: number; y: number; width: number; height: number };
    visible?: boolean;
    maximized?: boolean;
    fullScreen?: boolean;
}): MockBrowserWindow {
    const eventHandlers = new Map<string, Set<Function>>();
    const state = {
        destroyed: false,
        visible: options?.visible ?? false,
        minimized: false,
        maximized: options?.maximized ?? false,
        fullScreen: options?.fullScreen ?? false,
        focused: false,
        bounds: options?.bounds ?? { x: 100, y: 100, width: 800, height: 600 },
    };

    const webContents = createMockWebContents();

    const mockWindow: MockBrowserWindow = {
        // Window operations
        loadFile: vi.fn().mockResolvedValue(undefined),
        loadURL: vi.fn().mockResolvedValue(undefined),
        show: vi.fn(() => {
            state.visible = true;
        }),
        hide: vi.fn(() => {
            state.visible = false;
        }),
        close: vi.fn(() => {
            mockWindow._simulateEvent("close");
        }),
        destroy: vi.fn(() => {
            state.destroyed = true;
            mockWindow._simulateEvent("closed");
        }),
        minimize: vi.fn(() => {
            state.minimized = true;
        }),
        maximize: vi.fn(() => {
            state.maximized = true;
            state.minimized = false;
        }),
        unmaximize: vi.fn(() => {
            state.maximized = false;
        }),
        restore: vi.fn(() => {
            state.minimized = false;
        }),
        setFullScreen: vi.fn((fullScreen: boolean) => {
            state.fullScreen = fullScreen;
            if (fullScreen) {
                mockWindow._simulateEvent("enter-full-screen");
            } else {
                mockWindow._simulateEvent("leave-full-screen");
            }
        }),
        focus: vi.fn(() => {
            state.focused = true;
        }),

        // State queries
        isDestroyed: vi.fn(() => state.destroyed),
        isVisible: vi.fn(() => state.visible && !state.destroyed),
        isMinimized: vi.fn(() => state.minimized),
        isMaximized: vi.fn(() => state.maximized),
        isFullScreen: vi.fn(() => state.fullScreen),
        isFocused: vi.fn(() => state.focused),
        getBounds: vi.fn(() => ({ ...state.bounds })),
        getNormalBounds: vi.fn(() => ({ ...state.bounds })),

        // Web contents
        webContents,

        // Event handling
        on: vi.fn((event: string, handler: Function) => {
            if (!eventHandlers.has(event)) {
                eventHandlers.set(event, new Set());
            }
            eventHandlers.get(event)!.add(handler);
            return mockWindow;
        }),
        once: vi.fn((event: string, handler: Function) => {
            if (!eventHandlers.has(event)) {
                eventHandlers.set(event, new Set());
            }
            const wrappedHandler = (...args: unknown[]) => {
                eventHandlers.get(event)?.delete(wrappedHandler);
                handler(...args);
            };
            eventHandlers.get(event)!.add(wrappedHandler);
            return mockWindow;
        }),
        removeListener: vi.fn((event: string, handler: Function) => {
            eventHandlers.get(event)?.delete(handler);
            return mockWindow;
        }),
        removeAllListeners: vi.fn((event?: string) => {
            if (event) {
                eventHandlers.delete(event);
            } else {
                eventHandlers.clear();
            }
            return mockWindow;
        }),

        // Internal tracking
        _eventHandlers: eventHandlers,
        _simulateEvent: (event: string, ...args: unknown[]) => {
            eventHandlers.get(event)?.forEach((handler) => handler(...args));
        },
        _state: state,
    };

    return mockWindow;
}

/**
 * Create a mock BrowserWindow constructor.
 */
export function createMockBrowserWindowConstructor(): Mock & {
    getAllWindows: Mock;
    _instances: MockBrowserWindow[];
} {
    const instances: MockBrowserWindow[] = [];

    const MockConstructor = vi.fn((options?: Record<string, unknown>) => {
        const window = createMockBrowserWindow({
            bounds: {
                x: (options?.x as number) ?? 100,
                y: (options?.y as number) ?? 100,
                width: (options?.width as number) ?? 800,
                height: (options?.height as number) ?? 600,
            },
            visible: options?.show !== false,
            maximized: false,
            fullScreen: (options?.fullscreen as boolean) ?? false,
        });
        instances.push(window);
        return window;
    }) as Mock & { getAllWindows: Mock; _instances: MockBrowserWindow[] };

    MockConstructor.getAllWindows = vi.fn(() => instances.filter((w) => !w._state.destroyed));
    MockConstructor._instances = instances;

    return MockConstructor;
}

/**
 * Create a mock App module.
 */
export function createMockApp(options?: {
    isPackaged?: boolean;
    paths?: Map<string, string>;
}): MockApp {
    const eventHandlers = new Map<string, Set<Function>>();
    const paths = options?.paths ?? new Map([
        ["userData", "/mock/userData"],
        ["home", "/mock/home"],
        ["appData", "/mock/appData"],
        ["temp", "/mock/temp"],
        ["logs", "/mock/logs"],
    ]);

    const mockApp: MockApp = {
        whenReady: vi.fn().mockResolvedValue(undefined),
        isReady: vi.fn().mockReturnValue(true),
        quit: vi.fn(),
        exit: vi.fn(),
        relaunch: vi.fn(),
        isPackaged: options?.isPackaged ?? false,
        getPath: vi.fn((name: string) => paths.get(name) ?? `/mock/${name}`),
        getAppPath: vi.fn().mockReturnValue("/mock/app"),
        getName: vi.fn().mockReturnValue("MockApp"),
        getVersion: vi.fn().mockReturnValue("1.0.0"),
        requestSingleInstanceLock: vi.fn().mockReturnValue(true),
        setAsDefaultProtocolClient: vi.fn().mockReturnValue(true),
        on: vi.fn((event: string, handler: Function) => {
            if (!eventHandlers.has(event)) {
                eventHandlers.set(event, new Set());
            }
            eventHandlers.get(event)!.add(handler);
            return mockApp;
        }),
        once: vi.fn((event: string, handler: Function) => {
            if (!eventHandlers.has(event)) {
                eventHandlers.set(event, new Set());
            }
            const wrappedHandler = (...args: unknown[]) => {
                eventHandlers.get(event)?.delete(wrappedHandler);
                handler(...args);
            };
            eventHandlers.get(event)!.add(wrappedHandler);
            return mockApp;
        }),
        removeListener: vi.fn((event: string, handler: Function) => {
            eventHandlers.get(event)?.delete(handler);
            return mockApp;
        }),
        _eventHandlers: eventHandlers,
        _simulateEvent: (event: string, ...args: unknown[]) => {
            eventHandlers.get(event)?.forEach((handler) => handler(...args));
        },
        _paths: paths,
    };

    return mockApp;
}

/**
 * Create a mock IpcMain module.
 */
export function createMockIpcMain(): MockIpcMain {
    const handlers = new Map<string, Function>();
    const eventHandlers = new Map<string, Set<Function>>();

    const mockIpcMain: MockIpcMain = {
        handle: vi.fn((channel: string, handler: Function) => {
            handlers.set(channel, handler);
        }),
        handleOnce: vi.fn((channel: string, handler: Function) => {
            const wrappedHandler = async (...args: unknown[]) => {
                handlers.delete(channel);
                return handler(...args);
            };
            handlers.set(channel, wrappedHandler);
        }),
        removeHandler: vi.fn((channel: string) => {
            handlers.delete(channel);
        }),
        on: vi.fn((channel: string, handler: Function) => {
            if (!eventHandlers.has(channel)) {
                eventHandlers.set(channel, new Set());
            }
            eventHandlers.get(channel)!.add(handler);
            return mockIpcMain;
        }),
        once: vi.fn((channel: string, handler: Function) => {
            if (!eventHandlers.has(channel)) {
                eventHandlers.set(channel, new Set());
            }
            const wrappedHandler = (...args: unknown[]) => {
                eventHandlers.get(channel)?.delete(wrappedHandler);
                handler(...args);
            };
            eventHandlers.get(channel)!.add(wrappedHandler);
            return mockIpcMain;
        }),
        removeListener: vi.fn((channel: string, handler: Function) => {
            eventHandlers.get(channel)?.delete(handler);
            return mockIpcMain;
        }),
        removeAllListeners: vi.fn((channel?: string) => {
            if (channel) {
                eventHandlers.delete(channel);
            } else {
                eventHandlers.clear();
            }
            return mockIpcMain;
        }),
        _handlers: handlers,
        _eventHandlers: eventHandlers,
        _simulateInvoke: async <T>(channel: string, event: unknown, ...args: unknown[]): Promise<T> => {
            const handler = handlers.get(channel);
            if (!handler) {
                throw new Error(`No handler registered for channel: ${channel}`);
            }
            return handler(event, ...args) as T;
        },
        _simulateEvent: (channel: string, event: unknown, ...args: unknown[]) => {
            eventHandlers.get(channel)?.forEach((handler) => handler(event, ...args));
        },
    };

    return mockIpcMain;
}

/**
 * Create a mock Dialog module.
 */
export function createMockDialog(): MockDialog {
    return {
        showMessageBox: vi.fn().mockResolvedValue({ response: 0, checkboxChecked: false }),
        showErrorBox: vi.fn(),
        showOpenDialog: vi.fn().mockResolvedValue({ canceled: false, filePaths: ["/mock/file.txt"] }),
        showSaveDialog: vi.fn().mockResolvedValue({ canceled: false, filePath: "/mock/save.txt" }),
    };
}

/**
 * Create a mock Shell module.
 */
export function createMockShell(): MockShell {
    return {
        openExternal: vi.fn().mockResolvedValue(undefined),
        openPath: vi.fn().mockResolvedValue(""),
    };
}

/**
 * Create a mock SafeStorage module.
 */
export function createMockSafeStorage(options?: { encryptionAvailable?: boolean }): MockSafeStorage {
    const encryptionAvailable = options?.encryptionAvailable ?? true;

    return {
        isEncryptionAvailable: vi.fn().mockReturnValue(encryptionAvailable),
        encryptString: vi.fn((str: string) => Buffer.from(`encrypted:${str}`)),
        decryptString: vi.fn((buffer: Buffer) => {
            const str = buffer.toString();
            if (str.startsWith("encrypted:")) {
                return str.slice("encrypted:".length);
            }
            return str;
        }),
    };
}

/**
 * Create a mock Screen module.
 */
export function createMockScreen(displays?: Array<{ id: number; x: number; y: number; width: number; height: number }>): MockScreen {
    const defaultDisplay = { id: 1, x: 0, y: 0, width: 1920, height: 1080 };
    const allDisplays = displays ?? [defaultDisplay];
    const primary = allDisplays[0] ?? defaultDisplay;

    return {
        getAllDisplays: vi.fn().mockReturnValue(
            allDisplays.map((d) => ({
                id: d.id,
                bounds: { x: d.x, y: d.y, width: d.width, height: d.height },
                workArea: { x: d.x, y: d.y, width: d.width, height: d.height },
            }))
        ),
        getPrimaryDisplay: vi.fn().mockReturnValue({
            id: primary.id,
            bounds: { x: primary.x, y: primary.y, width: primary.width, height: primary.height },
            workArea: { x: primary.x, y: primary.y, width: primary.width, height: primary.height },
        }),
        getDisplayNearestPoint: vi.fn().mockReturnValue({
            id: primary.id,
            bounds: { x: primary.x, y: primary.y, width: primary.width, height: primary.height },
        }),
    };
}

/**
 * Create a mock Clipboard module.
 */
export function createMockClipboard(): MockClipboard {
    let content = "";
    return {
        writeText: vi.fn((text: string) => {
            content = text;
        }),
        readText: vi.fn(() => content),
    };
}

/**
 * Create a mock NativeImage module.
 */
export function createMockNativeImage(): MockNativeImage {
    const mockImage = {
        resize: vi.fn().mockReturnThis(),
        isEmpty: vi.fn().mockReturnValue(false),
    };

    return {
        createFromPath: vi.fn().mockReturnValue(mockImage),
        _mockImage: mockImage,
    };
}

// ===== Export mock Electron module for import aliasing =====

export const app = createMockApp();
export const BrowserWindow = createMockBrowserWindowConstructor();
export const ipcMain = createMockIpcMain();
export const dialog = createMockDialog();
export const shell = createMockShell();
export const safeStorage = createMockSafeStorage();
export const screen = createMockScreen();
export const clipboard = createMockClipboard();
export const nativeImage = createMockNativeImage();
export const net = {
    fetch: vi.fn().mockResolvedValue({
        ok: true,
        status: 200,
        text: vi.fn().mockResolvedValue(""),
        json: vi.fn().mockResolvedValue({}),
    }),
};

// Re-export for direct access (useful when aliasing electron imports)
export default {
    app,
    BrowserWindow,
    ipcMain,
    dialog,
    shell,
    safeStorage,
    screen,
    clipboard,
    nativeImage,
    net,
};
