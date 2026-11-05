import { describe, expect, it } from "vitest";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createPackagedLaunchRecorder } from "../packaged-launch";

describe("packaged launch recorder", () => {
    it("maps observed shell events and declares unobserved marks unavailable", async () => {
        const recorder = createPackagedLaunchRecorder({ runId: "run-1", runKind: "demo", condition: "cold" });
        await recorder.emit("demo_spawn", "launcher", "shell_main");
        await recorder.emit("electron_ready", "shell", "shell_main");
        await recorder.emit("runtime_ready", "backend", "backend");
        await recorder.emit("app_interactive", "renderer", "webview_renderer");
        const sample = await recorder.complete();
        expect(sample.schema_version).toBe("packaged-launch.v1");
        expect(sample.condition).toBe("cold");
        expect(sample.marks.find((mark) => mark.name === "process_spawn")?.elapsed_ns).toBe(0);
        expect(sample.marks.find((mark) => mark.name === "backend_ready")?.availability.state).toBe("available");
        expect(sample.marks.find((mark) => mark.name === "first_contentful_paint")?.availability.state).toBe("unavailable");
        expect(sample.metrics.find((metric) => metric.name === "tree_pss_mb")?.availability.state).toBe("unavailable");
    });

    it("keeps the first visible window and exposes runtime waterfall marks", async () => {
        const recorder = createPackagedLaunchRecorder({ runId: "run-window", runKind: "demo" });
        await recorder.emit("splash_shown", "splash", "shell_main", { elapsed_ns: "100000000" });
        await recorder.emit("main_window_shown", "window", "shell_main", { elapsed_ns: "600000000" });
        await recorder.emit("runtime_token_available", "runtime", "backend", { elapsed_ns: "700000000" });
        await recorder.emit("runtime_ipc_port_published", "runtime", "backend", { elapsed_ns: "800000000" });
        await recorder.emit("runtime_health_ready", "runtime", "backend", { elapsed_ns: "900000000" });
        await recorder.emit("runtime_ready", "runtime", "backend", { elapsed_ns: "1000000000" });
        await recorder.emit("runtime_service_event", "api", "backend", { mark_name: "service_spawned:api", observed_at: "2026-10-07T12:00:01.100Z" });
        await recorder.emit("runtime_service_event", "api", "backend", { mark_name: "service_health_success:api:attempt-2", observed_at: "2026-10-07T12:00:01.900Z" });
        const sample = await recorder.complete();
        expect(sample.marks.find((mark) => mark.name === "first_window")?.elapsed_ns).toBe(100000000);
        expect(sample.marks.find((mark) => mark.name === "runtime_token_available")?.availability.state).toBe("available");
        expect(sample.marks.find((mark) => mark.name === "runtime_ipc_port_published")?.availability.state).toBe("available");
        expect(sample.marks.find((mark) => mark.name === "runtime_health_ready")?.availability.state).toBe("available");
        expect(sample.marks.find((mark) => mark.name === "backend_ready")?.elapsed_ns).toBe(1000000000);
        expect(sample.marks.find((mark) => mark.name === "service_spawned:api")?.availability.state).toBe("available");
        expect(sample.marks.find((mark) => mark.name === "service_health_success:api:attempt-2")?.availability.state).toBe("available");
    });

    it("rejects credential-shaped event details", async () => {
        const recorder = createPackagedLaunchRecorder({ runKind: "demo" });
        await expect(recorder.emit("demo_spawn", "shell", "shell_main", { token: "redacted" })).rejects.toThrow("credential-shaped");
    });

    it("persists packaged-launch.v1 snapshots atomically", async () => {
        const directory = await mkdtemp(join(tmpdir(), "scenario-to-desktop-packaged-launch-"));
        const tracePath = join(directory, "sample.json");
        try {
            const recorder = createPackagedLaunchRecorder({ runKind: "demo", tracePath });
            await recorder.emit("demo_spawn", "shell", "shell_main");
            const persisted = JSON.parse(await readFile(tracePath, "utf8")) as { schema_version: string; marks: unknown[] };
            expect(persisted.schema_version).toBe("packaged-launch.v1");
            expect(persisted.marks).toHaveLength(1);
            await recorder.complete();
            expect(JSON.parse(await readFile(tracePath, "utf8")).metrics).toHaveLength(7);
        } finally { await rm(directory, { recursive: true, force: true }); }
    });
});
