import { describe, expect, it, vi } from "vitest";
import { consumeUpdateIntent, writeUpdateIntent, type UpdateRecoveryFileSystem } from "../update-recovery";

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

const intent = { schemaVersion: 1 as const, fromVersion: "1.0.0", toVersion: "1.1.0", requestedAt: "2026-09-06T00:00:00Z", nonce: "nonce-12345678" };

describe("update recovery marker", () => {
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
});
