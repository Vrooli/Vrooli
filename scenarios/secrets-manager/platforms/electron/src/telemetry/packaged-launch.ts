import { randomUUID } from "node:crypto";
import * as fs from "node:fs/promises";
import * as path from "node:path";
import { performance as nodePerformance } from "node:perf_hooks";

export const PACKAGED_LAUNCH_SCHEMA_VERSION = "packaged-launch.v1";
export type LaunchRunKind = "protocol" | "demo";
export type LaunchEventName =
    | "recorder_started" | "protocol_started" | "protocol_completed" | "demo_spawn"
    | "electron_ready" | "splash_created" | "splash_load_completed" | "splash_ready_to_show"
    | "splash_shown" | "splash_first_paint" | "runtime_spawned" | "runtime_ipc_port_published"
    | "runtime_token_available" | "runtime_health_ready" | "runtime_ready" | "port_discovered"
    | "server_ready" | "main_window_created" | "main_window_load_completed" | "main_window_shown"
    | "runtime_exec_completed" | "runtime_service_event" | "app_ready" | "first_contentful_paint"
    | "app_interactive" | "journey_started" | "recording_ended";
export type LaunchTraceLevel = "info" | "warn" | "error";

interface Availability { state: "available" | "unavailable"; reason: string }
export interface LaunchMark { name: string; elapsed_ns: number; availability: Availability }
export interface PackagedLaunchSample {
    schema_version: string;
    sample_id: string;
    scenario: string;
    ramp: string;
    strategy: string;
    toggle_attribution: { shared_runtime_readiness: boolean; secure_custom_scheme_ui: boolean; v8_snapshot_code_cache: boolean; electron_fuses: boolean };
    platform: string;
    captured_at: string;
    experiment_id: string;
    host_fingerprint: string;
    artifact_digest: string;
    condition: string;
    display: string;
    profiler_mode: string;
    power_state: string;
    marks: LaunchMark[];
    metrics: Array<{ name: string; value: number; unit: string; availability: Availability }>;
    process_roles: Array<Record<string, unknown>>;
    app_interactive_source: string;
    platform_detail?: { linux_desktop: { artifact_page_cache_residency_pct: number } };
}

export interface PackagedLaunchRecorder {
    emit(name: LaunchEventName, component: string, role: string, details?: Record<string, string>, level?: LaunchTraceLevel): Promise<void>;
    complete(): Promise<PackagedLaunchSample>;
    snapshot(): PackagedLaunchSample;
}
export interface PackagedLaunchConfig {
    runId?: string;
    runKind: LaunchRunKind;
    tracePath?: string;
    scenario?: string;
    strategy?: string;
    optimizationToggles?: { sharedRuntimeReadiness: boolean; secureCustomSchemeUI: boolean; v8SnapshotCodeCache: boolean; electronFuses: boolean };
    condition?: string;
    platform?: string;
    record?: (event: string, details?: Record<string, unknown>, level?: LaunchTraceLevel) => Promise<void>;
}

const forbiddenDetailKey = /(token|secret|password|credential|authorization|api[_-]?key|private[_-]?key)/i;
const forbiddenDetailValue = /(bearer\s+|-----begin [^-]+ key-----)/i;
const markMap: Partial<Record<LaunchEventName, string[]>> = {
    demo_spawn: ["process_spawn"], protocol_started: ["process_spawn"], electron_ready: ["shell_ready"],
    splash_shown: ["first_window"], main_window_shown: ["first_window"], server_ready: ["ui_server_ready"],
    runtime_spawned: ["runtime_spawn"], runtime_exec_completed: ["runtime_exec_complete"], runtime_token_available: ["runtime_token_available"],
    runtime_ipc_port_published: ["runtime_ipc_port_published"], runtime_health_ready: ["runtime_health_ready"],
    runtime_ready: ["backend_ready", "runtime_ready"], port_discovered: ["port_discovered"],
    first_contentful_paint: ["first_contentful_paint"], app_interactive: ["app_interactive"],
};

function validateDetails(details: Record<string, string> | undefined): void {
    for (const [key, value] of Object.entries(details ?? {})) {
        if (forbiddenDetailKey.test(key) || forbiddenDetailValue.test(value)) throw new Error(`credential-shaped launch detail is not allowed: ${key}`);
    }
}

export function createPackagedLaunchRecorder(config: PackagedLaunchConfig): PackagedLaunchRecorder {
    const start = nodePerformance.now();
    const startedAt = new Date().toISOString();
    const marks = new Map<string, LaunchMark>();
    const runId = config.runId?.trim() || process.env.SMOKE_TEST_RUN_ID?.trim() || randomUUID();
    const platform = config.platform || (process.platform === "darwin" ? "macos" : process.platform === "win32" ? "windows" : "linux");
    const sample: PackagedLaunchSample = {
        schema_version: PACKAGED_LAUNCH_SCHEMA_VERSION, sample_id: `${runId}:${config.runKind}`,
        scenario: config.scenario || process.env.S2D_SCENARIO || process.env.VROOLI_DESKTOP_SCENARIO || "", ramp: "desktop",
        strategy: config.strategy || process.env.S2D_STRATEGY || "electron-sidecar",
        toggle_attribution: {
            shared_runtime_readiness: config.optimizationToggles?.sharedRuntimeReadiness === true,
            secure_custom_scheme_ui: config.optimizationToggles?.secureCustomSchemeUI === true,
            v8_snapshot_code_cache: config.optimizationToggles?.v8SnapshotCodeCache === true,
            electron_fuses: config.optimizationToggles?.electronFuses === true,
        }, platform,
        captured_at: startedAt, experiment_id: process.env.S2D_EXPERIMENT_ID || "",
        host_fingerprint: "", artifact_digest: process.env.S2D_ARTIFACT_DIGEST || "",
        condition: config.condition || process.env.SMOKE_TEST_CONDITION || "warm",
        display: process.env.DISPLAY || "", profiler_mode: process.env.S2D_PROFILE_MODE || "disabled",
        power_state: "unknown", marks: [], metrics: [], process_roles: [], app_interactive_source: "",
    };

    async function persist(): Promise<void> {
        if (!config.tracePath?.trim()) return;
        sample.marks = [...marks.values()].sort((a, b) => a.elapsed_ns - b.elapsed_ns);
        await fs.mkdir(path.dirname(config.tracePath), { recursive: true });
        const temporaryPath = path.join(path.dirname(config.tracePath), `.${path.basename(config.tracePath)}.${process.pid}.${randomUUID()}.tmp`);
        try {
            await fs.writeFile(temporaryPath, `${JSON.stringify(sample, null, 2)}\n`, { mode: 0o600 });
            await fs.rename(temporaryPath, config.tracePath).catch(async (error: NodeJS.ErrnoException) => {
                if (error.code !== "EEXIST" && error.code !== "EPERM") throw error;
                await fs.rm(config.tracePath!, { force: true }); await fs.rename(temporaryPath, config.tracePath!);
            });
        } finally { await fs.rm(temporaryPath, { force: true }); }
    }

    return {
        async emit(name, component, role, details = {}, level = "info") {
            if (!component.trim() || !role.trim()) throw new Error("launch mark component and role are required");
            validateDetails(details);
            const mapped = markMap[name] ?? (details.mark_name?.trim() ? [details.mark_name.trim()] : undefined);
            if (mapped) {
                const observedNs = Number(details.elapsed_ns);
                const observedAt = Date.parse(details.observed_at || "");
                for (const mappedName of mapped) {
                    const elapsed = mappedName === "process_spawn" ? 0 : Number.isFinite(observedNs) && observedNs >= 0 ? Math.round(observedNs) : Number.isFinite(observedAt) ? Math.max(0, Math.round((observedAt - Date.parse(startedAt)) * 1_000_000)) : Math.max(0, Math.round((nodePerformance.now() - start) * 1_000_000));
                    const existing = marks.get(mappedName);
                    if (mappedName !== "first_window" || !existing || elapsed < existing.elapsed_ns) {
                        marks.set(mappedName, { name: mappedName, elapsed_ns: elapsed, availability: { state: "available", reason: "observed by the launch producer" } });
                    }
                }
                if (mapped.includes("app_interactive")) sample.app_interactive_source = details.source || "declared_mark";
            }
            await persist();
            await config.record?.(`packaged_launch:${mapped?.join(",") || name}`, { sample_id: sample.sample_id, component, role, ...details }, level);
        },
        async complete() {
            sample.metrics = [];
            for (const name of ["process_spawn", "shell_ready", "first_window", "backend_ready", "first_contentful_paint", "app_interactive"]) {
                if (!marks.has(name)) marks.set(name, { name, elapsed_ns: 0, availability: { state: "unavailable", reason: `${name} was not observed during this launch` } });
            }
            const unavailableMetrics: Array<[string, string]> = [["tree_pss_mb", "MB"], ["tree_uss_mb", "MB"], ["idle_cpu_pct", "%"], ["steady_pss_mb", "MB"], ["artifact_bytes", "bytes"], ["installed_bytes", "bytes"], ["update_bytes", "bytes"]];
            for (const [name, unit] of unavailableMetrics) {
                sample.metrics.push({ name, value: 0, unit, availability: { state: "unavailable", reason: "measured-launch host collector has not enriched this launch" } });
            }
            sample.marks = [...marks.values()].sort((a, b) => a.elapsed_ns - b.elapsed_ns);
            await persist();
            return JSON.parse(JSON.stringify(sample)) as PackagedLaunchSample;
        },
        snapshot() { sample.marks = [...marks.values()].sort((a, b) => a.elapsed_ns - b.elapsed_ns); return JSON.parse(JSON.stringify(sample)) as PackagedLaunchSample; },
    };
}
