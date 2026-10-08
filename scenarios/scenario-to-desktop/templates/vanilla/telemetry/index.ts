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
    createLaunchTraceRecorder,
    sha256File,
    LAUNCH_TRACE_SCHEMA_VERSION,
    type LaunchTraceRecorder,
    type LaunchEventName,
} from "./launch-trace";

export { createLaunchProfiler, type LaunchProfiler, type ProfileArtifact, type ProfileMode } from "./profiling";
