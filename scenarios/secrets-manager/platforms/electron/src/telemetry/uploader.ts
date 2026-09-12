/**
 * Telemetry Uploader
 *
 * DOC: docs/internal/SEAMS.md#telemetry-uploader
 *
 * Uploads telemetry events to a remote server.
 * Uses filesystem and HTTP seams for testability.
 */

import type {
    IFileSystem,
    IHttpClient,
    IPathUtils,
    ITelemetryUploader,
    TelemetryUploadPayload,
    TelemetryUploadState,
} from "./types";
import { readTelemetryEvents } from "./recorder";

/**
 * Configuration for the telemetry uploader.
 */
export interface TelemetryUploaderConfig {
    /** Scenario name for upload payload */
    scenarioName: string;
    /** Deployment mode for upload payload */
    deploymentMode: string;
    /** Path to the telemetry upload state file */
    statePath: string;
    /** Callback to record upload success event */
    onUploadSuccess?: (uploadURL: string, eventCount: number, reason: string) => Promise<void>;
}

/**
 * Create a telemetry uploader with injected dependencies.
 */
export function createTelemetryUploader(
    fs: IFileSystem,
    http: IHttpClient,
    pathUtils: IPathUtils,
    config: TelemetryUploaderConfig
): ITelemetryUploader {
    const state: TelemetryUploadState = {
        lastSignature: null,
        loaded: false,
    };

    /**
     * Load the upload state from disk.
     */
    async function loadState(): Promise<void> {
        if (state.loaded) return;

        try {
            const raw = await fs.readFile(config.statePath, "utf-8");
            const parsed = JSON.parse(raw) as { lastSignature?: string };
            state.lastSignature = parsed.lastSignature ?? null;
        } catch {
            // Missing state file is fine - first upload
        } finally {
            state.loaded = true;
        }
    }

    /**
     * Save the upload state to disk.
     */
    async function saveState(signature: string, reason: string): Promise<void> {
        try {
            await fs.mkdir(pathUtils.dirname(config.statePath), { recursive: true });
            await fs.writeFile(
                config.statePath,
                JSON.stringify({ lastSignature: signature, reason }, null, 2)
            );
        } catch {
            // Best-effort persistence
        }
    }

    /**
     * Generate a signature for deduplication.
     */
    function generateSignature(filePath: string, stats: { mtimeMs: number; size: number }, uploadURL: string): string {
        return `${filePath}:${stats.mtimeMs}:${stats.size}:${uploadURL}`;
    }

    const uploader: ITelemetryUploader = {
        async upload(filePath: string, uploadURL: string, reason: string, force = false): Promise<void> {
            await loadState();

            const stats = await fs.stat(filePath);
            if (stats.size === 0) {
                throw new Error("Telemetry file is empty");
            }

            const signature = generateSignature(filePath, stats, uploadURL);
            if (!force && signature === state.lastSignature) {
                console.log("[Telemetry] Already uploaded for this file signature");
                return;
            }

            const events = await readTelemetryEvents(fs, filePath);
            if (events.length === 0) {
                throw new Error("No telemetry events found to upload");
            }

            const payload: TelemetryUploadPayload = {
                scenario_name: config.scenarioName,
                deployment_mode: config.deploymentMode,
                source: "desktop-runtime",
                events,
            };

            const response = await http.post(
                uploadURL,
                JSON.stringify(payload),
                { "Content-Type": "application/json" }
            );

            if (!response.ok) {
                const text = await response.text().catch(() => "");
                throw new Error(text || `Telemetry upload failed (${response.status})`);
            }

            state.lastSignature = signature;
            await saveState(signature, reason);

            if (config.onUploadSuccess) {
                await config.onUploadSuccess(uploadURL, events.length, reason);
            }
        },

        async autoUploadIfConfigured(_reason: string): Promise<boolean> {
            // This method is typically overridden by the orchestrator
            // that has access to runtime configuration
            return false;
        },
    };

    return uploader;
}

/**
 * Create a fetch-based HTTP client for production use.
 */
export function createFetchHttpClient(): IHttpClient {
    return {
        async post(url: string, body: string, headers?: Record<string, string>) {
            const response = await fetch(url, {
                method: "POST",
                headers: headers ?? {},
                body,
            });
            return {
                ok: response.ok,
                status: response.status,
                text: () => response.text(),
            };
        },
    };
}

/**
 * Create a Node.js fs-based filesystem interface for production use.
 */
export function createNodeFileSystem(fsPromises: typeof import("node:fs").promises): IFileSystem {
    return {
        appendFile: (path, content) => fsPromises.appendFile(path, content),
        readFile: (path, encoding) => fsPromises.readFile(path, encoding),
        writeFile: (path, content) => fsPromises.writeFile(path, content),
        stat: (path) => fsPromises.stat(path),
        mkdir: async (path, options) => {
            await fsPromises.mkdir(path, options);
        },
    };
}

/**
 * Create a Node.js path-based path utilities interface.
 */
export function createNodePathUtils(pathModule: typeof import("node:path")): IPathUtils {
    return {
        join: (...segments) => pathModule.join(...segments),
        dirname: (path) => pathModule.dirname(path),
    };
}
