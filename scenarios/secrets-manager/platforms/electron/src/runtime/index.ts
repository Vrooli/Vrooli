/**
 * Runtime Module
 *
 * DOC: docs/internal/SEAMS.md#runtime-module
 *
 * Barrel exports for the bundled runtime process management system.
 */

// Types
export type {
    RuntimeExitInfo,
    RuntimeSecret,
    RuntimeSecretsResponse,
    RuntimeGPUInfo,
    RuntimeReadyResponse,
    RuntimePortsResponse,
    RuntimeTelemetryResponse,
    RuntimeDiagnostics,
    RuntimeValidationError,
    RuntimeValidationResponse,
    RuntimeRequestOptions,
    IRuntimeHttpClient,
    IProcessSpawner,
    IRuntimeFileSystem,
    ITimer,
    RuntimeControlConfig,
    IRuntimeManager,
    IRuntimeControlClient,
} from "./types";

export { createInitialExitInfo } from "./types";

// Exit Tracker
export {
    createExitTracker,
    matchErrorPattern,
    DEFAULT_ERROR_PATTERNS,
    type ErrorPattern,
} from "./exit-tracker";

// Control Client
export {
    createRuntimeControlClient,
    createFetchRuntimeHttpClient,
    createRealTimer,
    createNodeRuntimeFileSystem,
} from "./control-client";
