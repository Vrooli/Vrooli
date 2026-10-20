import { describe, expect, it, vi } from "vitest";
import { bundledRuntimeSpawnOptions, signalBundledRuntimeGroup } from "../process-lifetime";

describe("bundled runtime process lifetime", () => {
    it("gives POSIX bundled runtime a dedicated process group", () => {
        expect(bundledRuntimeSpawnOptions("linux")).toEqual({ detached: true });
        expect(bundledRuntimeSpawnOptions("darwin")).toEqual({ detached: true });
        expect(bundledRuntimeSpawnOptions("win32")).toEqual({ detached: false });
    });

    it("signals the owned process group rather than a host launch scope", () => {
        const signalProcess = vi.fn();
        const child = { pid: 4321, kill: vi.fn() };
        signalBundledRuntimeGroup(child, "SIGTERM", "linux", signalProcess as typeof process.kill);
        expect(signalProcess).toHaveBeenCalledWith(-4321, "SIGTERM");
        expect(child.kill).not.toHaveBeenCalled();
    });

    it("falls back to the direct child if the owned process group is already gone", () => {
        const signalProcess = vi.fn(() => {
            throw new Error("process group no longer exists");
        });
        const child = { pid: 4321, kill: vi.fn() };

        signalBundledRuntimeGroup(child, "SIGKILL", "linux", signalProcess as typeof process.kill);

        expect(signalProcess).toHaveBeenCalledWith(-4321, "SIGKILL");
        expect(child.kill).toHaveBeenCalledWith("SIGKILL");
    });
});
