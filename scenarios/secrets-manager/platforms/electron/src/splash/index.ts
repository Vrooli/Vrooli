/**
 * Splash Window Module
 *
 * DOC: docs/internal/SEAMS.md#splash-window-module
 *
 * Exports all splash window components for use in main.ts.
 * This module provides:
 * - Type definitions for type safety
 * - SplashWindowManager for window lifecycle
 * - Server readiness checking with proper validation
 * - Factory functions for production use
 */

// Type exports
export type {
    StartupPhase,
    SplashStatus,
    SplashWindowConfig,
    SplashCloseResult,
    ISplashWindowManager,
    IWindowFactory,
} from "./types";

export {
    SPLASH_IPC_CHANNELS,
    DEFAULT_SPLASH_CONFIG,
    PHASE_MESSAGES,
} from "./types";

// Manager exports
export type {
    IPathResolver,
    IIpcMain,
    IClipboard,
    SplashManagerDeps,
} from "./manager";

export {
    SplashWindowManager,
    createSplashManager,
} from "./manager";

// Server readiness exports
export type {
    ReadinessResult,
    ReadinessConfig,
    IHttpClient,
    ITimer,
    ReadinessProgressCallback,
} from "./server-readiness";

export {
    DEFAULT_READINESS_CONFIG,
    realTimer,
    isAcceptableStatus,
    checkServerReadiness,
    createElectronReadinessChecker,
} from "./server-readiness";
