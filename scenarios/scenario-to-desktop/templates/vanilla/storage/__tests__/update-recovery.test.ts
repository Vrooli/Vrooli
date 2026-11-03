import { describe, expect, it, vi } from "vitest";
import { classifyUpdateRecovery, consumeUpdateIntent, createUpdateStateWriter, markUpdateIntentApplying, readUpdateIntent, transitionUpdateState, updateStateNames, writeInstalledUpdateReceipt, writeUpdateIntent, writeUpdateState, type UpdateRecoveryFileSystem, type UpdateStateName } from "../update-recovery";
import specification from "../update-lifecycle.spec.json";

function fixture(initial = "") {
    const files = new Map<string, string>();
    if (initial) files.set("/user/update-intent.json", initial);
    const fs: UpdateRecoveryFileSystem = {
        readFile: vi.fn(async (path: string) => {
            const value = files.get(path);
            if (value === undefined) throw Object.assign(new Error("missing"), { code: "ENOENT" });
            return value;
        }),
        writeFile: vi.fn(async (path: string, content: string) => { files.set(path, content); }),
        rename: vi.fn(async (from: string, to: string) => {
            const value = files.get(from);
            if (value === undefined) throw new Error("temporary marker missing");
            files.set(to, value);
            files.delete(from);
        }),
        unlink: vi.fn(async (path: string) => { files.delete(path); }),
    };
    return { fs, files };
}

const intent = { schemaVersion: 1 as const, fromVersion: "1.0.0", toVersion: "1.1.0", requestedAt: "2026-09-06T00:00:00Z", nonce: "nonce-12345678", toArtifactRef: "sha512:new" };

describe("update recovery marker", () => {
    it("conforms to every declared lifecycle state/event pair and trace", () => {
        expect(specification.states).toEqual(["idle", ...updateStateNames]);
        expect(specification.events).toEqual(updateStateNames);
        expect(Object.keys(specification.transitionMatrix)).toEqual(specification.states);
        for (const [state, row] of Object.entries(specification.transitionMatrix)) {
            expect(Object.keys(row)).toEqual(specification.events);
            for (const [event, expected] of Object.entries(row)) {
                const transition = () => transitionUpdateState(state as UpdateStateName | "idle", event as UpdateStateName);
                if (expected === null) expect(transition).toThrow("invalid update transition");
                else expect(transition()).toBe(expected);
            }
        }
        for (const trace of specification.traces) {
            let state = trace.initial as UpdateStateName | "idle";
            for (const event of trace.events) state = transitionUpdateState(state, event as UpdateStateName);
            expect(specification.terminal).toContain(state);
        }
        expect(() => transitionUpdateState("healthy", "unknown" as UpdateStateName)).toThrow();
        expect(() => transitionUpdateState("unknown" as UpdateStateName, "healthy")).toThrow();
    });
    it("serializes updater events and resumes the persisted installation trace after launch", async () => {
        const { fs, files } = fixture();
        const path = "/user/update-state.json";
        const persist = createUpdateStateWriter(fs, path);
        const states: UpdateStateName[] = ["offered", "downloading", "downloading", "verified", "apply_pending", "applying", "installing"];
        await Promise.all(states.map(state => persist({ schemaVersion: 1, state, observedAt: "2026-09-08T00:00:00Z" })));
        const resumed = createUpdateStateWriter(fs, path);
        await resumed({ schemaVersion: 1, state: "relaunched", observedAt: "2026-09-08T00:01:00Z" });
        await resumed({ schemaVersion: 1, state: "healthy", observedAt: "2026-09-08T00:01:01Z" });
        expect(JSON.parse(files.get(path)!).transitions.map((entry: { state: string }) => entry.state)).toEqual(["offered", "downloading", "verified", "apply_pending", "applying", "installing", "relaunched", "healthy"]);
        await expect(resumed({ schemaVersion: 1, state: "applying", observedAt: "now" })).rejects.toThrow("healthy -> applying");
        expect(JSON.parse(files.get(path)!).state).toBe("healthy");
    });

    it("keeps the last durable transition when persistence fails and permits recovery", async () => {
        const { fs, files } = fixture();
        const path = "/user/update-state.json";
        const persist = createUpdateStateWriter(fs, path);
        for (const state of ["offered", "verified", "apply_pending"] as const) await persist({ schemaVersion: 1, state, observedAt: "now" });
        vi.mocked(fs.rename!).mockRejectedValueOnce(new Error("disk full"));
        await expect(persist({ schemaVersion: 1, state: "applying", observedAt: "now" })).rejects.toThrow("disk full");
        expect(JSON.parse(files.get(path)!).state).toBe("apply_pending");
        await expect(persist({ schemaVersion: 1, state: "installing", observedAt: "now" })).rejects.toThrow("apply_pending -> installing");
        await createUpdateStateWriter(fs, path)({ schemaVersion: 1, state: "recovering", observedAt: "now" });
        expect(JSON.parse(files.get(path)!).transitions.map((entry: { state: string }) => entry.state)).toEqual(["offered", "verified", "apply_pending", "recovering"]);
    });
    it("atomically records and classifies an applied update", async () => {
        const { fs, files } = fixture();
        await writeUpdateIntent(fs, "/user/update-intent.json", intent);
        expect(fs.rename).toHaveBeenCalledTimes(1);
        expect((await consumeUpdateIntent(fs, "/user/update-intent.json", "1.1.0")).disposition).toBe("applied");
        expect(files.has("/user/update-intent.json")).toBe(false);
    });

    it("classifies a restart on the predecessor as interrupted", async () => {
        const { fs } = fixture(JSON.stringify(intent));
        const recovery = await consumeUpdateIntent(fs, "/user/update-intent.json", "1.0.0");
        expect(recovery.disposition).toBe("interrupted");
    });

    it("does not claim applied when the successor artifact identity is missing", async () => {
        const { toArtifactRef: _missingArtifactRef, ...missingIdentity } = intent;
        expect(classifyUpdateRecovery(missingIdentity, "1.1.0")).toBe("unknown");
        const recovery = await consumeUpdateIntent(fixture(JSON.stringify(missingIdentity)).fs, "/user/update-intent.json", "1.1.0");
        expect(recovery.disposition).toBe("unknown");
    });

    it("removes a malformed marker without claiming recovery", async () => {
        const { fs, files } = fixture("not-json");
        expect((await consumeUpdateIntent(fs, "/user/update-intent.json", "1.0.0")).disposition).toBe("corrupt");
        expect(files.has("/user/update-intent.json")).toBe(false);
    });

    it("cleans a temporary marker when replacement fails", async () => {
        const { fs, files } = fixture();
        fs.rename = vi.fn(async () => { throw new Error("replacement failed"); });
        await expect(writeUpdateIntent(fs, "/user/update-intent.json", intent)).rejects.toThrow("replacement failed");
        expect([...files.keys()].some((key) => key.includes(".tmp-"))).toBe(false);
    });
    it("retains product, target and artifact identity through the applying boundary", async () => {
        const { fs } = fixture();
        await writeUpdateIntent(fs, "/user/update-intent.json", {
            ...intent,
            applicationId: "com.example.portal",
            channel: "stable",
            platform: "linux",
            arch: "x64",
            fromArtifactRef: "sha256:old",
            toArtifactRef: "sha512:new",
            state: "apply_pending",
        });
        await markUpdateIntentApplying(fs, "/user/update-intent.json");
        await expect(readUpdateIntent(fs, "/user/update-intent.json")).resolves.toMatchObject({
            applicationId: "com.example.portal",
            channel: "stable",
            platform: "linux",
            arch: "x64",
            fromArtifactRef: "sha256:old",
            toArtifactRef: "sha512:new",
            state: "applying",
        });
    });

    it("writes a bounded installed-client receipt atomically", async () => {
        const { fs, files } = fixture();
        await writeInstalledUpdateReceipt(fs, "/user/update-receipt.json", {
            schemaVersion: 1,
            outcome: "applied",
            applicationId: "com.example.portal",
            channel: "stable",
            platform: "linux",
            arch: "x64",
            fromVersion: "1.0.0",
            toVersion: "1.1.0",
            fromArtifactRef: "sha256:old",
            toArtifactRef: "sha512:new",
            installedExecutable: "/opt/portal/portal",
            installedExecutableDigest: "sha256:installed",
            health: "passed",
            observedAt: "2026-09-08T00:00:00Z",
        });
        expect(JSON.parse(files.get("/user/update-receipt.json") ?? "null")).toMatchObject({ outcome: "applied", health: "passed", toArtifactRef: "sha512:new" });
    });

    it("refuses an invalid receipt outcome or health", async () => {
        const { fs } = fixture();
        await expect(writeInstalledUpdateReceipt(fs, "/user/update-receipt.json", {
            schemaVersion: 1,
            outcome: "claimed" as never,
            fromVersion: "1.0.0",
            toVersion: "1.1.0",
            installedExecutable: "/opt/portal/portal",
            health: "passed",
            observedAt: "2026-09-08T00:00:00Z",
        })).rejects.toThrow("invalid installed update receipt");
        await expect(writeInstalledUpdateReceipt(fs, "/user/update-receipt.json", {
            schemaVersion: 1,
            outcome: "applied",
            fromVersion: "1.0.0",
            toVersion: "1.1.0",
            installedExecutable: "/opt/portal/portal",
            health: "unknown" as never,
            observedAt: "2026-09-08T00:00:00Z",
        })).rejects.toThrow("invalid installed update receipt");
    });

    it("records updater state separately from the apply intent", async () => {
        const { fs, files } = fixture();
        await writeUpdateState(fs, "/user/update-state.json", {
            schemaVersion: 1,
            state: "verified",
            applicationId: "com.example.portal",
            channel: "stable",
            platform: "linux",
            arch: "x64",
            fromVersion: "1.0.0",
            toVersion: "1.1.0",
            artifactRef: "sha512:new",
            observedAt: "2026-09-08T00:00:00Z",
        });
        expect(JSON.parse(files.get("/user/update-state.json") ?? "null")).toMatchObject({ state: "verified", artifactRef: "sha512:new" });
    });

    it("preserves the authoritative state envelope", async () => {
        const { fs, files } = fixture();
        await writeUpdateState(fs, "/user/update-state.json", {
            schemaVersion: 99 as never,
            state: "verified",
            observedAt: "2026-09-08T00:00:00Z",
            reason: "checked",
        });
        expect(JSON.parse(files.get("/user/update-state.json") ?? "null")).toMatchObject({ schemaVersion: 1, state: "verified", observedAt: "2026-09-08T00:00:00Z" });
    });
});
