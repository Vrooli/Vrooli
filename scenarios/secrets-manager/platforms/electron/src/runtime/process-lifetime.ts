import type { ChildProcess } from "node:child_process";

export function bundledRuntimeSpawnOptions(platform: NodeJS.Platform = process.platform): { detached: boolean } {
    // A dedicated session/process group gives the desktop runtime supervisor a
    // precise kill boundary, independent of the shell or host agent that opened
    // the desktop app.
    return { detached: platform !== "win32" };
}

export function signalBundledRuntimeGroup(
    child: Pick<ChildProcess, "pid" | "kill">,
    signal: NodeJS.Signals,
    platform: NodeJS.Platform = process.platform,
    signalProcess: typeof process.kill = process.kill.bind(process),
): void {
    if (!child.pid) return;
    if (platform === "win32") {
        try { child.kill(signal); } catch { /* process already exited */ }
        return;
    }
    try {
        signalProcess(-child.pid, signal);
    } catch {
        // The leader may already be gone while a group member remains. Falling
        // back to the direct child is safe and keeps shutdown best-effort.
        try { child.kill(signal); } catch { /* process already exited */ }
    }
}
