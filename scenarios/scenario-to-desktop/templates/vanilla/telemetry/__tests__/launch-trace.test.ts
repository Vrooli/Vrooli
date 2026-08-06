import { describe, expect, it } from "vitest";
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
        const trace = createLaunchTraceRecorder({ runKind: "demo", tracePath: "/tmp/scenario-to-desktop-launch-trace-test.json" });
        await trace.emit("demo_spawn", "electron", "main");
        const result = await trace.complete();
        expect(result.completed_at).toBeTruthy();
        expect(result.schema_version).toBe("launch-trace.v1");
    });
});
