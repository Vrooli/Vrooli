import { createHash } from "node:crypto";
import * as fs from "node:fs/promises";
import * as path from "node:path";
import { Session } from "node:inspector";
import { writeHeapSnapshot } from "node:v8";

export type ProfileMode = "disabled" | "chromium" | "cpu" | "heap" | "all";

export interface ProfileArtifact {
    kind: "chromium_trace" | "main_cpu_profile" | "heap_snapshot";
    path?: string;
    checksum?: string;
    sizeBytes?: number;
    available: boolean;
    reason?: string;
}

export interface LaunchProfiler {
    start(): Promise<void>;
    stop(): Promise<ProfileArtifact[]>;
}

export interface LaunchProfilerDeps {
    startChromium: () => Promise<void>;
    stopChromium: (filePath: string) => Promise<void>;
}

function enabled(mode: ProfileMode, kind: "chromium" | "cpu" | "heap"): boolean {
    return mode === "all" || mode === kind;
}

function checksum(data: Buffer): string {
    return `sha256:${createHash("sha256").update(data).digest("hex")}`;
}

async function fileArtifact(kind: ProfileArtifact["kind"], filePath: string): Promise<ProfileArtifact> {
    const data = await fs.readFile(filePath);
    return { kind, path: filePath, checksum: checksum(data), sizeBytes: data.byteLength, available: data.byteLength > 0 };
}

export function createLaunchProfiler(mode: ProfileMode, directory: string, deps: LaunchProfilerDeps): LaunchProfiler {
    let cpuSession: Session | null = null;
    let cpuStarted = false;
    let started = false;

    return {
        async start(): Promise<void> {
            if (mode === "disabled") return;
            await fs.mkdir(directory, { recursive: true });
            started = true;
            if (enabled(mode, "chromium")) await deps.startChromium();
            if (enabled(mode, "cpu")) {
                cpuSession = new Session();
                cpuSession.connect();
                await new Promise<void>((resolve, reject) => cpuSession!.post("Profiler.enable", (error) => error ? reject(error) : resolve()));
                await new Promise<void>((resolve, reject) => cpuSession!.post("Profiler.start", (error) => error ? reject(error) : resolve()));
                cpuStarted = true;
            }
        },
        async stop(): Promise<ProfileArtifact[]> {
            if (!started || mode === "disabled") return [];
            const artifacts: ProfileArtifact[] = [];
            if (enabled(mode, "chromium")) {
                const filePath = path.join(directory, "chromium-trace.json");
                try { await deps.stopChromium(filePath); artifacts.push(await fileArtifact("chromium_trace", filePath)); }
                catch (error) { artifacts.push({ kind: "chromium_trace", available: false, reason: String(error) }); }
            }
            if (enabled(mode, "cpu")) {
                const filePath = path.join(directory, "main-cpu-profile.json");
                try {
                    if (!cpuSession || !cpuStarted) throw new Error("CPU profiler did not start");
                    const profile = await new Promise<unknown>((resolve, reject) => cpuSession!.post("Profiler.stop", (error, result) => error ? reject(error) : resolve(result)));
                    await fs.writeFile(filePath, `${JSON.stringify(profile)}\n`, { mode: 0o600 });
                    artifacts.push(await fileArtifact("main_cpu_profile", filePath));
                } catch (error) { artifacts.push({ kind: "main_cpu_profile", available: false, reason: String(error) }); }
                cpuSession?.disconnect();
            }
            if (enabled(mode, "heap")) {
                const filePath = path.join(directory, "heap-snapshot.heapsnapshot");
                try { writeHeapSnapshot(filePath); artifacts.push(await fileArtifact("heap_snapshot", filePath)); }
                catch (error) { artifacts.push({ kind: "heap_snapshot", available: false, reason: String(error) }); }
            }
            return artifacts;
        },
    };
}

