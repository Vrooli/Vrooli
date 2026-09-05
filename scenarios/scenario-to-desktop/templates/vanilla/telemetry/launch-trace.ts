import { createHash, randomUUID } from "node:crypto";
import * as fs from "node:fs/promises";
import * as path from "node:path";
import { performance as nodePerformance } from "node:perf_hooks";

export const LAUNCH_TRACE_SCHEMA_VERSION = "launch-trace.v1";

export type LaunchRunKind = "protocol" | "demo";
export type LaunchTraceLevel = "info" | "warn" | "error";

export type LaunchEventName =
    | "recorder_started"
    | "protocol_started"
    | "protocol_completed"
    | "demo_spawn"
    | "electron_ready"
    | "splash_created"
    | "splash_load_completed"
    | "splash_ready_to_show"
    | "splash_shown"
    | "splash_first_paint"
    | "runtime_spawned"
    | "runtime_token_available"
    | "runtime_health_ready"
    | "runtime_ready"
    | "port_discovered"
    | "server_ready"
    | "main_window_created"
    | "main_window_load_completed"
    | "main_window_shown"
    | "app_ready"
    | "journey_started"
    | "recording_ended";

export interface LaunchTraceEvent {
    name: LaunchEventName;
    component: string;
    role: string;
    monotonic_ns: number;
    wall_time: string;
    details?: Record<string, string>;
}

export interface LaunchTrace {
    schema_version: string;
    run_id: string;
    run_kind: LaunchRunKind;
    started_at: string;
    completed_at?: string;
    events: LaunchTraceEvent[];
}

export interface LaunchTraceRecorder {
    emit(name: LaunchEventName, component: string, role: string, details?: Record<string, string>, level?: LaunchTraceLevel): Promise<void>;
    complete(): Promise<LaunchTrace>;
    snapshot(): LaunchTrace;
}

export interface LaunchTraceConfig {
    runId?: string;
    runKind: LaunchRunKind;
    tracePath?: string;
    record?: (event: string, details?: Record<string, unknown>, level?: LaunchTraceLevel) => Promise<void>;
}

const forbiddenDetailKey = /(token|secret|password|credential|authorization|api[_-]?key|private[_-]?key)/i;
const forbiddenDetailValue = /(bearer\s+|-----begin [^-]+ key-----)/i;

function monotonicNs(start: number): number {
    return Math.max(0, Math.round((nodePerformance.now() - start) * 1_000_000));
}

function validateDetails(details: Record<string, string> | undefined): void {
    for (const [key, value] of Object.entries(details ?? {})) {
        if (forbiddenDetailKey.test(key) || forbiddenDetailValue.test(value)) {
            throw new Error(`credential-shaped launch detail is not allowed: ${key}`);
        }
    }
}

function traceRunId(config: LaunchTraceConfig): string {
    const base = config.runId?.trim() || process.env.SMOKE_TEST_RUN_ID?.trim() || randomUUID();
    return `${base}:${config.runKind}`;
}

export function createLaunchTraceRecorder(config: LaunchTraceConfig): LaunchTraceRecorder {
    const start = nodePerformance.now();
    const startedAt = new Date().toISOString();
    const trace: LaunchTrace = {
        schema_version: LAUNCH_TRACE_SCHEMA_VERSION,
        run_id: traceRunId(config),
        run_kind: config.runKind,
        started_at: startedAt,
        events: [],
    };

    async function emit(name: LaunchEventName, component: string, role: string, details: Record<string, string> = {}, level: LaunchTraceLevel = "info"): Promise<void> {
        if (!component.trim() || !role.trim()) throw new Error("launch trace component and role are required");
        validateDetails(details);
        const event: LaunchTraceEvent = { name, component, role, monotonic_ns: monotonicNs(start), wall_time: new Date().toISOString() };
        if (Object.keys(details).length > 0) event.details = { ...details };
        trace.events.push(event);
        // Persist every event, not only completion. A supervisor timeout or a
        // platform-level quit can interrupt the final async callback; the
        // partial trace is still valuable and never appears as an empty file.
        await persist(trace);
        await config.record?.(`launch_trace:${name}`, { trace_run_id: trace.run_id, run_kind: trace.run_kind, component, role, monotonic_ns: event.monotonic_ns, ...details }, level);
    }

    async function persist(value: LaunchTrace): Promise<void> {
        if (!config.tracePath?.trim()) return;
        await fs.mkdir(path.dirname(config.tracePath), { recursive: true });
        // Never expose a truncated JSON trace if the short-lived Electron
        // process is interrupted during a write. The evidence reader must see
        // either the previous complete snapshot or no snapshot at all.
        const temporaryPath = path.join(
            path.dirname(config.tracePath),
            `.${path.basename(config.tracePath)}.${process.pid}.${randomUUID()}.tmp`,
        );
        try {
            await fs.writeFile(temporaryPath, `${JSON.stringify(value, null, 2)}\n`, { mode: 0o600 });
            try {
                await fs.rename(temporaryPath, config.tracePath);
            } catch (error) {
                // Windows cannot rename over an existing file. Preserve the
                // same complete-snapshot guarantee there by replacing the
                // old snapshot only after the temporary write succeeded.
                const code = (error as NodeJS.ErrnoException).code;
                if (code !== "EEXIST" && code !== "EPERM") throw error;
                await fs.rm(config.tracePath, { force: true });
                await fs.rename(temporaryPath, config.tracePath);
            }
        } finally {
            await fs.rm(temporaryPath, { force: true });
        }
    }

    return {
        emit,
        async complete(): Promise<LaunchTrace> {
            trace.completed_at = new Date().toISOString();
            const value = JSON.parse(JSON.stringify(trace)) as LaunchTrace;
            await persist(value);
            return value;
        },
        snapshot(): LaunchTrace {
            return JSON.parse(JSON.stringify(trace)) as LaunchTrace;
        },
    };
}

export function sha256File(data: Buffer): string {
    return `sha256:${createHash("sha256").update(data).digest("hex")}`;
}
