/**
 * Telemetry Recorder
 *
 * DOC: docs/internal/SEAMS.md#telemetry-recorder
 *
 * Records telemetry events to a local JSONL file.
 * Uses filesystem seam for testability.
 */

import type {
    IFileSystem,
    ITelemetryRecorder,
    TelemetryConfig,
    TelemetryDetails,
    TelemetryEvent,
    TelemetryLevel,
    SessionInfo,
} from "./types";

/**
 * Create a telemetry recorder with injected dependencies.
 */
export function createTelemetryRecorder(
    fs: IFileSystem,
    config: TelemetryConfig
): ITelemetryRecorder {
    let initialized = false;
    let sessionOutcomeRecorded = false;
    const sessionInfo: SessionInfo = {
        startedAt: null,
        readyAt: null,
        failureMessage: null,
    };

    /**
     * Build a telemetry event payload.
     */
    function buildPayload(
        event: string,
        details: TelemetryDetails,
        level: TelemetryLevel
    ): TelemetryEvent {
        return {
            timestamp: new Date().toISOString(),
            event,
            level,
            session_id: config.sessionId,
            session_kind: config.sessionKind,
            deploymentMode: config.deploymentMode,
            serverType: config.serverType,
            details,
        };
    }

    const recorder: ITelemetryRecorder = {
        async initialize(): Promise<void> {
            if (initialized) return;

            try {
                // Create or touch the telemetry file
                await fs.appendFile(config.filePath, "");
                initialized = true;
                console.log(`[Telemetry] Initialized at ${config.filePath}`);
            } catch (error) {
                console.warn("[Telemetry] Failed to initialize:", error);
                throw error;
            }
        },

        async record(
            event: string,
            details: TelemetryDetails = {},
            level: TelemetryLevel = "info"
        ): Promise<void> {
            if (!initialized) return;

            const payload = buildPayload(event, details, level);

            try {
                await fs.appendFile(config.filePath, JSON.stringify(payload) + "\n");
            } catch (error) {
                console.warn("[Telemetry] Failed to write entry:", error);
            }
        },

        async recordSessionOutcome(reason?: string): Promise<void> {
            if (sessionOutcomeRecorded) return;
            if (!initialized) return;

            const failedReason =
                sessionInfo.failureMessage ||
                reason ||
                (sessionInfo.readyAt ? "" : "app_exit_before_ready");
            const succeeded = failedReason === "";

            await recorder.record(
                succeeded ? "app_session_succeeded" : "app_session_failed",
                {
                    started_at: sessionInfo.startedAt,
                    ready_at: sessionInfo.readyAt,
                    reason: failedReason,
                },
                succeeded ? "info" : "error"
            );

            sessionOutcomeRecorded = true;
        },

        getFilePath(): string | null {
            return initialized ? config.filePath : null;
        },

        setSessionStarted(timestamp: string): void {
            sessionInfo.startedAt = timestamp;
        },

        setSessionReady(timestamp: string): void {
            sessionInfo.readyAt = timestamp;
        },

        setSessionFailure(message: string): void {
            sessionInfo.failureMessage = message;
        },
    };

    return recorder;
}

/**
 * Read telemetry events from a file.
 * @param fs - Filesystem interface
 * @param filePath - Path to the telemetry file
 * @param limit - Maximum number of events to read
 * @returns Array of parsed event objects
 */
export async function readTelemetryEvents(
    fs: IFileSystem,
    filePath: string,
    limit = 500
): Promise<Array<Record<string, unknown>>> {
    const raw = await fs.readFile(filePath, "utf-8");
    const lines = raw
        .split(/\r?\n/)
        .map((line) => line.trim())
        .filter(Boolean)
        .slice(0, limit);

    const events: Array<Record<string, unknown>> = [];
    lines.forEach((line, index) => {
        try {
            const parsed: unknown = JSON.parse(line);
            if (parsed && typeof parsed === "object") {
                events.push(parsed as Record<string, unknown>);
            }
        } catch (error) {
            throw new Error(`Telemetry line ${index + 1} is invalid JSON: ${error}`);
        }
    });

    return events;
}
