/**
 * Runtime Module Types
 *
 * DOC: docs/internal/SEAMS.md#runtime-types
 *
 * Type definitions for the bundled runtime process management system.
 * Defines seams for process spawning, HTTP requests, and file operations.
 */

import type { ChildProcess } from "node:child_process";

// ===== Exit Tracking Types =====

/**
 * Tracks runtime process exit status for crash detection.
 */
export interface RuntimeExitInfo {
    exited: boolean;
    code: number | null;
    signal: NodeJS.Signals | null;
    stderr: string;
    exitedAt: Date | null;
}

/**
 * Create a fresh exit info object.
 */
export function createInitialExitInfo(): RuntimeExitInfo {
    return {
        exited: false,
        code: null,
        signal: null,
        stderr: "",
        exitedAt: null,
    };
}

// ===== Runtime API Types =====

export interface RuntimeSecret {
    id: string;
    class: string;
    required: boolean;
    has_value: boolean;
    description?: string;
    prompt?: Record<string, string>;
}

export interface RuntimeSecretsResponse {
    secrets: RuntimeSecret[];
}

export interface RuntimeGPUInfo {
    available: boolean;
    method?: string;
    reason?: string;
    requirements?: Record<string, string>;
}

export interface RuntimeReadyResponse {
    ready: boolean;
    details?: Record<string, { ready: boolean; message?: string; reason?: string }>;
    gpu?: RuntimeGPUInfo;
}

export interface RuntimePortsResponse {
    services?: Record<string, Record<string, number>>;
    apiBase?: string;
}

export interface RuntimeTelemetryResponse {
    path?: string;
    upload_url?: string;
}

export interface RuntimeDiagnostics {
    ready: RuntimeReadyResponse;
    ports: RuntimePortsResponse;
    logs: Record<string, string>;
    gpu?: RuntimeGPUInfo;
    telemetryPath?: string;
    telemetryUploadUrl?: string;
}

export interface RuntimeValidationError {
    code: string;
    service?: string;
    path?: string;
    message: string;
}

export interface RuntimeValidationResponse {
    valid: boolean;
    errors?: RuntimeValidationError[];
    warnings?: RuntimeValidationError[];
    missing_binaries?: Array<{ service_id: string; platform: string; path: string }>;
    invalid_checksums?: Array<{ service_id: string; path: string; expected: string; actual: string }>;
}

// ===== Request Options =====

export interface RuntimeRequestOptions {
    expectText?: boolean;
    method?: string;
    body?: unknown;
}

// ===== Seam Interfaces =====

/**
 * HTTP client for runtime control API.
 */
export interface IRuntimeHttpClient {
    request<T = unknown>(
        url: string,
        opts?: {
            method?: string;
            headers?: Record<string, string>;
            body?: string;
        }
    ): Promise<{ ok: boolean; status: number; text(): Promise<string>; json(): Promise<T> }>;
}

/**
 * Process spawner for runtime binary.
 */
export interface IProcessSpawner {
    spawn(
        command: string,
        args: string[],
        options: {
            stdio: "inherit" | ["inherit", "inherit", "pipe"];
            env?: NodeJS.ProcessEnv;
        }
    ): ChildProcess;
}

/**
 * File system operations for runtime.
 */
export interface IRuntimeFileSystem {
    readFile(path: string, encoding: "utf-8"): Promise<string>;
    access(path: string): Promise<void>;
    stat(path: string): Promise<{ isFile(): boolean; isDirectory(): boolean }>;
}

/**
 * Timer interface for async operations.
 */
export interface ITimer {
    now(): number;
    sleep(ms: number): Promise<void>;
}

// ===== Runtime Control Config =====

export interface RuntimeControlConfig {
    enabled: boolean;
    host: string;
    port: number;
    tokenPath: string;
    telemetryUploadUrl: string;
    logLines: number;
}

// ===== Runtime Manager Interface =====

/**
 * Interface for runtime process management.
 */
export interface IRuntimeManager {
    /**
     * Start the bundled runtime and return the UI URL.
     */
    start(): Promise<string>;

    /**
     * Shutdown the runtime gracefully.
     */
    shutdown(): Promise<void>;

    /**
     * Check if the runtime process is running.
     */
    isRunning(): boolean;

    /**
     * Get exit information for the runtime.
     */
    getExitInfo(): RuntimeExitInfo;

    /**
     * Check if runtime exited unexpectedly.
     */
    hasExitedUnexpectedly(): boolean;
}

/**
 * Interface for runtime control API client.
 */
export interface IRuntimeControlClient {
    /**
     * Make a request to the runtime control API.
     */
    request<T>(endpoint: string, opts?: RuntimeRequestOptions): Promise<T | string>;

    /**
     * Wait for the runtime health endpoint to respond.
     */
    waitForHealth(timeoutMs: number): Promise<void>;

    /**
     * Get runtime diagnostics.
     */
    collectDiagnostics(serviceIds?: string[]): Promise<RuntimeDiagnostics>;

    /**
     * Validate the runtime bundle.
     */
    validate(): Promise<RuntimeValidationResponse | null>;
}
