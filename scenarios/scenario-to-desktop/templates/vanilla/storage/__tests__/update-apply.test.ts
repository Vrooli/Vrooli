import { describe, expect, it, vi } from "vitest";
import { readFileSync } from "node:fs";
import { mkdtemp, readFile, rm } from "node:fs/promises";
import * as fs from "node:fs/promises";
import { tmpdir } from "node:os";
import path from "node:path";
import { createUpdateApplyTransition } from "../update-apply";
import { consumeUpdateIntent, markUpdateIntentApplying, readUpdateIntent, writeUpdateIntent } from "../update-recovery";

const target = { applicationId: "app.console", serverURL: "http://127.0.0.1:39565/api/v1/updates/web-console", channel: "stable", platform: "linux", arch: "x64" };
const offer = { version: "0.0.2", sha512: "successor", vrooli: { applicationId: target.applicationId, applicationKey: "web-console", channel: "stable", platform: "linux", arch: "x64", artifactRef: "sha512:successor" } };

async function fixture() {
    const dir = await mkdtemp(path.join(tmpdir(), "desktop-update-apply-"));
    const marker = path.join(dir, "intent.json");
    const updater = { autoInstallOnAppQuit: true };
    const installs: boolean[] = [];
    const failures = vi.fn();
    const prepare = vi.fn(async update => {
        await writeUpdateIntent(fs, marker, { schemaVersion: 1, fromVersion: "0.0.1", toVersion: update.version, requestedAt: new Date().toISOString(), nonce: "test-nonce-1234", toArtifactRef: update.artifactRef, state: "apply_pending" });
        await markUpdateIntentApplying(fs, marker);
    });
    const install = vi.fn((restart: boolean) => { installs.push(restart); });
    const transition = createUpdateApplyTransition({ updater, target, prepare, install, onFailure: failures });
    return { marker, updater, installs, failures, prepare, install, transition, cleanup: () => rm(dir, { recursive: true, force: true }) };
}

describe("verified update apply boundary", () => {
    it.each([
        ["application", { ...offer, vrooli: { ...offer.vrooli, applicationId: "unrelated.app" } }],
        ["channel", { ...offer, vrooli: { ...offer.vrooli, channel: "beta" } }],
        ["platform", { ...offer, vrooli: { ...offer.vrooli, platform: "win" } }],
        ["architecture", { ...offer, vrooli: { ...offer.vrooli, arch: "arm64" } }],
        ["digest binding", { ...offer, vrooli: { ...offer.vrooli, artifactRef: "sha512:other" } }],
        ["missing digest", { ...offer, sha512: "" }],
    ])("cannot install rejected %s metadata on ordinary quit", async (_name, info) => {
        const f = await fixture();
        try {
            expect(f.transition.downloaded(offer).ok).toBe(true);
            expect(f.transition.downloaded(info).ok).toBe(false);
            const event = { preventDefault: vi.fn() };
            expect(f.transition.beforeQuit(event)).toBe(false);
            await f.transition.apply(true);
            expect(f.updater.autoInstallOnAppQuit).toBe(false);
            expect(f.install).not.toHaveBeenCalled();
            expect(f.prepare).not.toHaveBeenCalled();
            expect(await readUpdateIntent(fs, f.marker)).toBeNull();
        } finally { await f.cleanup(); }
    });

    it.each([true, false])("records the same recoverable intent before install (restart=%s)", async restart => {
        const f = await fixture();
        try {
            f.transition.downloaded(offer);
            expect(f.transition.phase).toBe("pending");
            expect(f.installs).toEqual([]); // Choosing Later leaves the verified update staged.
            f.install.mockImplementation((actualRestart: boolean) => {
                expect(actualRestart).toBe(restart);
                expect(JSON.parse(readFileSync(f.marker, "utf-8"))).toMatchObject({ state: "applying", toVersion: "0.0.2", toArtifactRef: "sha512:successor" });
            });
            if (restart) await f.transition.apply(true);
            else {
                const event = { preventDefault: vi.fn() };
                expect(f.transition.beforeQuit(event)).toBe(true);
                expect(event.preventDefault).toHaveBeenCalledOnce();
                await f.transition.apply(false);
            }
            expect(JSON.parse(await readFile(f.marker, "utf-8"))).toMatchObject({ state: "applying", fromVersion: "0.0.1", toVersion: "0.0.2", toArtifactRef: "sha512:successor" });
            expect(f.install).toHaveBeenCalledOnce();
            expect(f.transition.phase).toBe("installing");
            expect(f.transition.beforeQuit({ preventDefault: vi.fn() })).toBe(false);
            expect((await consumeUpdateIntent(fs, f.marker, "0.0.2")).disposition).toBe("applied");
        } finally { await f.cleanup(); }
    });

    it("does not install if the durable intent cannot be written", async () => {
        const f = await fixture();
        try {
            f.prepare.mockRejectedValue(new Error("disk full"));
            f.transition.downloaded(offer);
            f.transition.beforeQuit({ preventDefault: vi.fn() });
            await expect(f.transition.apply(false)).rejects.toThrow("disk full");
            expect(f.install).not.toHaveBeenCalled();
            expect(f.failures).toHaveBeenCalledOnce();
            expect(f.transition.phase).toBe("failed");
            expect(f.transition.beforeQuit({ preventDefault: vi.fn() })).toBe(false);
        } finally { await f.cleanup(); }
    });

    it("serializes repeated quit and Restart Now while persistence is in progress", async () => {
        const f = await fixture();
        try {
            let release!: () => void;
            const barrier = new Promise<void>(resolve => { release = resolve; });
            const persist = f.prepare.getMockImplementation()!;
            f.prepare.mockImplementation(async update => { await barrier; await persist(update); });
            f.transition.downloaded(offer);
            f.transition.beforeQuit({ preventDefault: vi.fn() });
            f.transition.beforeQuit({ preventDefault: vi.fn() });
            const pending = f.transition.apply(true);
            expect(f.transition.phase).toBe("applying");
            expect(f.transition.downloaded({ ...offer, version: "0.0.3" }).ok).toBe(false);
            expect(f.transition.phase).toBe("applying");
            expect(f.install).not.toHaveBeenCalled();
            release();
            await pending;
            expect(f.prepare).toHaveBeenCalledOnce();
            expect(f.installs).toEqual([false]);
        } finally { await f.cleanup(); }
    });

    it("does not install after an updater error interrupts preparation", async () => {
        const f = await fixture();
        try {
            let release!: () => void;
            f.prepare.mockImplementation(() => new Promise<void>(resolve => { release = resolve; }));
            f.transition.downloaded(offer);
            const applying = f.transition.apply(true);
            await Promise.resolve();
            f.transition.failed();
            release();
            await expect(applying).rejects.toThrow("updater failure");
            expect(f.transition.phase).toBe("failed");
            expect(f.install).not.toHaveBeenCalled();
        } finally { await f.cleanup(); }
    });

    it("refuses a new update until cancelled preparation has finished", async () => {
        const f = await fixture();
        try {
            const releases: Array<() => void> = [];
            f.prepare.mockImplementation(() => new Promise<void>(resolve => { releases.push(resolve); }));
            f.transition.downloaded(offer);
            const old = f.transition.apply(true);
            await Promise.resolve();
            f.transition.failed();
            f.transition.failed(); // Repeated errors must preserve the in-flight operation.
            expect(f.transition.downloaded({ ...offer, version: "0.0.3" }).ok).toBe(false);
            releases[0]();
            await expect(old).rejects.toThrow("updater failure");
            expect(f.transition.phase).toBe("failed");
            expect(f.install).not.toHaveBeenCalled();
            expect(f.transition.downloaded({ ...offer, version: "0.0.3" }).ok).toBe(true);
            const current = f.transition.apply(false);
            await Promise.resolve();
            releases[1]();
            await current;
            expect(f.installs).toEqual([false]);
        } finally { await f.cleanup(); }
    });
});
