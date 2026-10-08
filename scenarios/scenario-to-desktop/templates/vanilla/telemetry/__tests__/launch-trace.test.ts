import { describe, expect, it } from "vitest";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { join } from "node:path";
import { createLaunchTraceRecorder } from "../launch-trace";

describe("launch trace recorder", () => {
    it("keeps protocol and demo identities separate", async () => {
        const protocol = createLaunchTraceRecorder({ runId: "run-1", runKind: "protocol" });
        const demo = createLaunchTraceRecorder({ runId: "run-1", runKind: "demo" });
        await protocol.emit("recorder_started", "orchestrator", "protocol");
        await demo.emit("demo_spawn", "electron", "main");
        expect(protocol.snapshot().run_id).not.toBe(demo.snapshot().run_id);
        expect(protocol.snapshot().events[0]?.monotonic_ns).toBeGreaterThanOrEqual(0);
    });

    it("rejects credential-shaped details", async () => {
        const trace = createLaunchTraceRecorder({ runKind: "demo" });
        await expect(trace.emit("demo_spawn", "electron", "main", { token: "redacted" })).rejects.toThrow("credential-shaped");
    });

    it("persists a completed trace when a path is supplied", async () => {
        const directory = await mkdtemp(join(tmpdir(), "scenario-to-desktop-launch-trace-"));
        const tracePath = join(directory, "trace.json");
        try {
            const trace = createLaunchTraceRecorder({ runKind: "demo", tracePath });
            await trace.emit("demo_spawn", "electron", "main");
            const persisted = JSON.parse(await readFile(tracePath, "utf8")) as { events: unknown[] };
            expect(persisted.events).toHaveLength(1);

            const result = await trace.complete();
            expect(result.completed_at).toBeTruthy();
            expect(result.schema_version).toBe("launch-trace.v1");
            expect(JSON.parse(await readFile(tracePath, "utf8")).completed_at).toBeTruthy();
        } finally {
            await rm(directory, { recursive: true, force: true });
        }
    });
});
