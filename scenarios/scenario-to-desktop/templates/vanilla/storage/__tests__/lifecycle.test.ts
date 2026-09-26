import { mkdtemp, mkdir, readFile, rm, stat, readdir, unlink, writeFile, rename, access } from "node:fs/promises";
import { tmpdir } from "node:os";
import * as path from "node:path";
import { afterEach, describe, expect, it } from "vitest";
import type { IStorageFileSystem, IStoragePathUtils } from "../types";
import { createAppStorage } from "../app-storage";
import { consumeUpdateIntent, writeUpdateIntent, type UpdateRecoveryFileSystem } from "../update-recovery";

const roots: string[] = [];

const storagePath: IStoragePathUtils = {
    join: path.join,
    dirname: path.dirname,
    normalize: path.normalize,
    resolve: path.resolve,
    isAbsolute: path.isAbsolute,
    relative: path.relative,
    sep: path.sep,
};

const storageFileSystem = {
    readFile: async (filePath: string, encoding?: "utf-8") => encoding
        ? readFile(filePath, encoding)
        : readFile(filePath),
    writeFile: async (filePath: string, data: string | Buffer, encoding?: "utf-8") => {
        await writeFile(filePath, data, encoding);
    },
    rename,
    mkdir: async (directory: string, options?: { recursive?: boolean }) => {
        await mkdir(directory, options);
    },
    readdir,
    unlink,
    rm,
    stat,
    access,
} as IStorageFileSystem;

const recoveryFileSystem: UpdateRecoveryFileSystem = {
    readFile: async (filePath, encoding) => readFile(filePath, encoding),
    writeFile: async (filePath, content, encoding) => writeFile(filePath, content, encoding),
    rename,
    unlink,
};

async function fixture(): Promise<{ root: string; installRoot: string; userData: string }> {
    const root = await mkdtemp(path.join(tmpdir(), "vrooli-desktop-lifecycle-"));
    roots.push(root);
    const installRoot = path.join(root, "Portal");
    const userData = path.join(root, "userData");
    await mkdir(installRoot, { recursive: true });
    await mkdir(userData, { recursive: true });
    return { root, installRoot, userData };
}

afterEach(async () => {
    await Promise.all(roots.splice(0).map((root) => rm(root, { recursive: true, force: true })));
});

describe("installed desktop lifecycle persistence", () => {
    it("initializes a clean install without inheriting predecessor user state", async () => {
        const { installRoot, userData } = await fixture();
        expect(await readdir(installRoot)).toEqual([]);

        await writeFile(path.join(installRoot, "Portal-1.0.0.AppImage"), "version-1");
        const storage = createAppStorage(storageFileSystem, storagePath, { userDataPath: userData });

        expect(await storage.readTextFile("conversation/state.json")).toBeNull();
        await storage.writeFile("runtime/install-state.json", JSON.stringify({ version: "1.0.0", profile: "client", ready: true }));
        expect(JSON.parse((await storage.readTextFile("runtime/install-state.json")) ?? "null")).toEqual({ version: "1.0.0", profile: "client", ready: true });
    });

    it("preserves conversation state and helper identity across an update", async () => {
        const { installRoot, userData } = await fixture();
        await writeFile(path.join(installRoot, "Portal-1.0.0.AppImage"), "version-1");

        const firstInstall = createAppStorage(storageFileSystem, storagePath, { userDataPath: userData });
        await firstInstall.writeFile("conversation/state.json", JSON.stringify({ threadId: "thread-1", draft: "kept" }));
        await firstInstall.writeFile("helper/identity.json", JSON.stringify({ installId: "helper-1" }));

        await rm(installRoot, { recursive: true, force: true });
        await mkdir(installRoot, { recursive: true });
        await writeFile(path.join(installRoot, "Portal-1.1.0.AppImage"), "version-2");

        const updatedInstall = createAppStorage(storageFileSystem, storagePath, { userDataPath: userData });
        expect(JSON.parse((await updatedInstall.readTextFile("conversation/state.json")) ?? "null")).toEqual({ threadId: "thread-1", draft: "kept" });
        expect(JSON.parse((await updatedInstall.readTextFile("helper/identity.json")) ?? "null")).toEqual({ installId: "helper-1" });
        expect(await access(path.join(installRoot, "Portal-1.1.0.AppImage"))).toBeUndefined();
    });

    it("reports an interrupted update while leaving user state readable", async () => {
        const { userData } = await fixture();
        const storage = createAppStorage(storageFileSystem, storagePath, { userDataPath: userData });
        await storage.writeFile("conversation/state.json", JSON.stringify({ threadId: "thread-2" }));

        const marker = path.join(userData, "update-intent.json");
        await writeUpdateIntent(recoveryFileSystem, marker, {
            schemaVersion: 1,
            fromVersion: "1.1.0",
            toVersion: "1.2.0",
            requestedAt: "2026-09-06T00:00:00Z",
            nonce: "lifecycle-12345678",
        });

        const recovery = await consumeUpdateIntent(recoveryFileSystem, marker, "1.1.0");
        expect(recovery.disposition).toBe("interrupted");
        expect(JSON.parse((await storage.readTextFile("conversation/state.json")) ?? "null")).toEqual({ threadId: "thread-2" });
        await expect(access(marker)).rejects.toMatchObject({ code: "ENOENT" });
    });

    it("removes only the owned install tree when retention is selected", async () => {
        const { installRoot, userData } = await fixture();
        const storage = createAppStorage(storageFileSystem, storagePath, { userDataPath: userData });
        await storage.writeFile("conversation/state.json", JSON.stringify({ threadId: "retained" }));
        await writeFile(path.join(installRoot, "owned-runtime"), "remove-me");

        await rm(installRoot, { recursive: true, force: true });

        await expect(access(installRoot)).rejects.toMatchObject({ code: "ENOENT" });
        expect(JSON.parse((await storage.readTextFile("conversation/state.json")) ?? "null")).toEqual({ threadId: "retained" });
    });
});
