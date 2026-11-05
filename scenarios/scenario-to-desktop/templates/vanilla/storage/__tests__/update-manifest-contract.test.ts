import { afterAll, beforeAll, describe, expect, it, vi } from "vitest";
import { execFileSync } from "node:child_process";
import { existsSync, mkdtempSync, readFileSync, rmSync } from "node:fs";
import { tmpdir } from "node:os";
import path from "node:path";
import { createUpdateApplyTransition } from "../update-apply";

const target = { applicationId: "com.vrooli.web-console", serverURL: "http://127.0.0.1:39565/api/v1/updates/web-console", channel: "stable", platform: "linux", arch: "x64" };
let temporary: string;
let manifests: Record<string, { vrooli: Record<string, unknown> }>;
beforeAll(() => {
    let root = process.cwd();
    while (!existsSync(path.join(root, "scenarios/landing-page-business-suite/api/go.mod"))) {
        const parent = path.dirname(root);
        if (parent === root) throw new Error("Vrooli checkout is required for producer/consumer contract test");
        root = parent;
    }
    temporary = mkdtempSync(path.join(tmpdir(), "desktop-manifest-contract-"));
    const output = path.join(temporary, "manifests.json");
    execFileSync("go", ["test", "./handlers/delivery", "-run", "^TestElectronManifestDesktopContract$", "-count=1"], {
        cwd: path.join(root, "scenarios/landing-page-business-suite/api"),
        env: { ...process.env, J10_MANIFEST_FIXTURE_OUT: output }, timeout: 60000,
    });
    manifests = JSON.parse(readFileSync(output, "utf-8"));
}, 65000);
afterAll(() => { if (temporary) rmSync(temporary, { recursive: true, force: true }); });

function consumer(arch = target.arch) {
    const prepare = vi.fn(async () => {}), install = vi.fn();
    const transition = createUpdateApplyTransition({ updater: { autoInstallOnAppQuit: true }, target: { ...target, arch }, prepare, install, onFailure: vi.fn() });
    return { transition, prepare, install };
}

describe("LPBS-produced architecture contract", () => {
    it("accepts canonical amd64 for an Electron x64 client", () => {
        expect(manifests.amd64.vrooli.architecture).toBe("amd64");
        expect(manifests.amd64.vrooli).not.toHaveProperty("arch");
        expect(consumer().transition.downloaded(manifests.amd64).ok).toBe(true);
    });
    it("rejects actual arm64 feed on x64 before Restart Now or ordinary quit", async () => {
        expect(manifests.arm64.vrooli.architecture).toBe("arm64");
        const c = consumer();
        expect(c.transition.downloaded(manifests.arm64).ok).toBe(false);
        expect(c.transition.beforeQuit({ preventDefault: vi.fn() })).toBe(false);
        await c.transition.apply(true);
        expect(c.prepare).not.toHaveBeenCalled();
        expect(c.install).not.toHaveBeenCalled();
    });
    it("accepts canonical arm64 for an Electron arm64 client", () => {
        expect(consumer("arm64").transition.downloaded(manifests.arm64).ok).toBe(true);
    });
    it("does not let a matching legacy arch hide incompatible canonical architecture", () => {
        expect(consumer().transition.downloaded({ ...manifests.arm64, vrooli: { ...manifests.arm64.vrooli, arch: "x64" } }).ok).toBe(false);
    });
    it("rejects malformed explicit architecture", () => {
        expect(consumer().transition.downloaded({ ...manifests.amd64, vrooli: { ...manifests.amd64.vrooli, architecture: 64 } }).ok).toBe(false);
    });
    it("rejects conflicting legacy arch alongside canonical architecture", () => {
        const manifest = { ...manifests.amd64, vrooli: { ...manifests.amd64.vrooli, arch: "arm64" } };
        expect(consumer().transition.downloaded(manifest).ok).toBe(false);
    });
});
