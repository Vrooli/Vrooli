/**
 * Telemetry Module
 *
 * DOC: docs/internal/SEAMS.md#telemetry-module
 *
 * Barrel exports for the telemetry recording and uploading system.
 */

// Types
export type {
    TelemetryLevel,
    TelemetryDetails,
    TelemetryEvent,
    TelemetryConfig,
    SessionInfo,
    IFileSystem,
    IHttpClient,
    IPathUtils,
    ITelemetryRecorder,
    ITelemetryUploader,
    TelemetryUploadState,
    TelemetryUploadPayload,
} from "./types";

// Recorder
export {
    createTelemetryRecorder,
    readTelemetryEvents,
} from "./recorder";

// Uploader
export {
    createTelemetryUploader,
    createFetchHttpClient,
    createNodeFileSystem,
    createNodePathUtils,
    type TelemetryUploaderConfig,
} from "./uploader";

export {
    createPackagedLaunchRecorder,
    PACKAGED_LAUNCH_SCHEMA_VERSION,
    type PackagedLaunchRecorder,
    type LaunchEventName,
} from "./packaged-launch";

export { createLaunchProfiler, type LaunchProfiler, type ProfileArtifact, type ProfileMode } from "./profiling";
