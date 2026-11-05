import { describe, expect, it, vi } from "vitest";
import { WindowStateStorage, type IFileSystem } from "./storage";

const state = {
    x: 10,
    y: 20,
    width: 800,
    height: 600,
    isMaximized: false,
    isFullScreen: false,
};

function fixture() {
    const files = new Map<string, string>();
    const fs: IFileSystem = {
        readFile: vi.fn(async (path: string) => {
            const value = files.get(path);
            if (value === undefined) throw Object.assign(new Error("missing"), { code: "ENOENT" });
            return value;
        }),
        writeFile: vi.fn(async (path: string, content: string) => { files.set(path, content); }),
        exists: vi.fn(async (path: string) => files.has(path)),
        rename: vi.fn(async (from: string, to: string) => {
            const value = files.get(from);
            if (value === undefined) throw new Error("temporary file missing");
            files.set(to, value);
            files.delete(from);
        }),
        unlink: vi.fn(async (path: string) => { files.delete(path); }),
    };
    return { fs, files };
}

describe("WindowStateStorage atomic persistence", () => {
    it("replaces state beside the destination", async () => {
        const { fs, files } = fixture();
        const storage = new WindowStateStorage({
            fileSystem: fs,
            pathProvider: { getUserDataPath: () => "/mock/userData", join: (...parts) => parts.join("/") },
        });

        await storage.save(state);

        expect(fs.rename).toHaveBeenCalledTimes(1);
        expect(files.get("/mock/userData/window-state.json")).toContain('"width": 800');
        expect([...files.keys()].some((key) => key.includes(".tmp-"))).toBe(false);
    });

    it("cleans the temporary state when replacement fails", async () => {
        const { fs, files } = fixture();
        fs.rename = vi.fn(async () => { throw new Error("replacement failed"); });
        const storage = new WindowStateStorage({
            fileSystem: fs,
            pathProvider: { getUserDataPath: () => "/mock/userData", join: (...parts) => parts.join("/") },
        });

        await expect(storage.save(state)).resolves.toBeUndefined();
        expect([...files.keys()].some((key) => key.includes(".tmp-"))).toBe(false);
    });
});
