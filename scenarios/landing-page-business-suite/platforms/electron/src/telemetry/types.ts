/**
 * Telemetry Module Types
 *
 * DOC: docs/internal/SEAMS.md#telemetry-types
 *
 * Type definitions for the telemetry recording and uploading system.
 * Defines seams for filesystem and HTTP operations to enable testing.
 */

// ===== Telemetry Data Types =====

export type TelemetryLevel = "info" | "warn" | "error";

export type TelemetryDetails = Record<string, unknown>;

/**
 * A single telemetry event payload.
 */
export interface TelemetryEvent {
    timestamp: string;
    event: string;
    level: TelemetryLevel;
    session_id: string;
    session_kind: string;
    deploymentMode: string;
    serverType: string;
    details: TelemetryDetails;
}

/**
 * Configuration for the telemetry recorder.
 */
export interface TelemetryConfig {
    /** Session ID for correlating events */
    sessionId: string;
    /** Session kind (e.g., "app", "smoke-test") */
    sessionKind: string;
    /** Deployment mode from app config */
    deploymentMode: string;
    /** Server type from app config */
    serverType: string;
    /** Path to the telemetry file */
    filePath: string;
}

/**
 * Session tracking information for outcome recording.
 */
export interface SessionInfo {
    startedAt: string | null;
    readyAt: string | null;
    failureMessage: string | null;
}

// ===== Seam Interfaces =====

/**
 * Filesystem operations needed by the telemetry module.
 * This seam allows injecting mock filesystems for testing.
 */
export interface IFileSystem {
    appendFile(path: string, content: string): Promise<void>;
    readFile(path: string, encoding: "utf-8"): Promise<string>;
    writeFile(path: string, content: string): Promise<void>;
    stat(path: string): Promise<{ size: number; mtimeMs: number }>;
    mkdir(path: string, options?: { recursive?: boolean }): Promise<void>;
}

/**
 * HTTP client for uploading telemetry.
 * This seam allows injecting mock HTTP clients for testing.
 */
export interface IHttpClient {
    post(url: string, body: string, headers?: Record<string, string>): Promise<{
        ok: boolean;
        status: number;
        text(): Promise<string>;
    }>;
}

/**
 * Path utilities needed by the telemetry module.
 */
export interface IPathUtils {
    join(...segments: string[]): string;
    dirname(path: string): string;
}

// ===== Recorder Interface =====

/**
 * Interface for telemetry recording.
 * Handles initialization, event recording, and session outcome tracking.
 */
export interface ITelemetryRecorder {
    /**
     * Initialize the telemetry system.
     * Creates the telemetry file if it doesn't exist.
     */
    initialize(): Promise<void>;

    /**
     * Record a telemetry event.
     * @param event - Event name
     * @param details - Additional event details
     * @param level - Log level (info, warn, error)
     */
    record(event: string, details?: TelemetryDetails, level?: TelemetryLevel): Promise<void>;

    /**
     * Record the session outcome (success or failure).
     * Can only be called once per session.
     * @param reason - Optional failure reason
     */
    recordSessionOutcome(reason?: string): Promise<void>;

    /**
     * Get the telemetry file path.
     * @returns The file path, or null if not initialized
     */
    getFilePath(): string | null;

    /**
     * Update session tracking info.
     */
    setSessionStarted(timestamp: string): void;
    setSessionReady(timestamp: string): void;
    setSessionFailure(message: string): void;
}

// ===== Uploader Interface =====

/**
 * State for tracking upload signatures to avoid duplicates.
 */
export interface TelemetryUploadState {
    lastSignature: string | null;
    loaded: boolean;
}

/**
 * Interface for telemetry uploading.
 * Handles reading events and uploading to a remote server.
 */
export interface ITelemetryUploader {
    /**
     * Upload a telemetry file to the given URL.
     * @param filePath - Path to the telemetry file
     * @param uploadURL - URL to upload to
     * @param reason - Reason for upload (for logging)
     * @param force - Force upload even if signature matches
     */
    upload(filePath: string, uploadURL: string, reason: string, force?: boolean): Promise<void>;

    /**
     * Automatically upload telemetry if configured.
     * @param reason - Reason for upload
     * @returns true if upload was successful
     */
    autoUploadIfConfigured(reason: string): Promise<boolean>;
}

/**
 * Upload payload structure sent to the telemetry server.
 */
export interface TelemetryUploadPayload {
    scenario_name: string;
    deployment_mode: string;
    source: string;
    events: Array<Record<string, unknown>>;
}
