import { describe, expect, it } from "vitest";
import { createAuthManager } from "./manager";
import type { AuthManagerDependencies, IAuthStorage, ISafeStorage } from "./types";

class MemoryStorage implements IAuthStorage {
    files = new Map<string, Buffer>();
    async resolvePath(path: string): Promise<string> { return path; }
    async readFile(path: string): Promise<Buffer | null> { return this.files.get(path) ?? null; }
    async readTextFile(path: string): Promise<string | null> { return this.files.get(path)?.toString("utf8") ?? null; }
    async writeFile(path: string, data: string | Buffer): Promise<void> { this.files.set(path, Buffer.isBuffer(data) ? data : Buffer.from(data)); }
    async deleteFile(path: string): Promise<boolean> { return this.files.delete(path); }
    async ensureDir(): Promise<void> { /* memory storage has no directories */ }
}

const safeStorage: ISafeStorage = {
    isEncryptionAvailable: () => true,
    encryptString: (value) => Buffer.from(`encrypted:${Buffer.from(value, "utf8").toString("base64")}`, "utf8"),
    decryptString: (value) => Buffer.from(value.toString("utf8").replace(/^encrypted:/, ""), "base64").toString("utf8"),
};

function dependencies(storage: IAuthStorage): AuthManagerDependencies {
    return {
        storage, safeStorage,
        http: { fetch: async () => ({ ok: true, status: 200, json: async () => ({ access_token: "access", refresh_token: "refresh", expires_at: new Date(Date.now() + 3600000).toISOString() }) }) },
        shell: { openExternal: async () => undefined },
        timer: { now: () => Date.now(), setTimeout: () => setTimeout(() => undefined, 3600000), clearTimeout: (handle) => clearTimeout(handle) },
        uuid: { generate: () => "fixed-id" },
        pathUtils: { dirname: (value) => value.includes("/") ? value.slice(0, value.lastIndexOf("/")) : "." },
        config: { protocol: "vrooli", lpbsUrl: "https://vrooli.test", tokensFile: "auth/tokens.enc", userFile: "auth/user.json", tokenRefreshBufferMs: 300000, appDisplayName: "Aquila" },
        onAuthChange: () => undefined,
        onLoopbackAuthorization: async () => ({ code: "code", state: "fixed-id", redirectURI: "http://127.0.0.1/callback" }),
        createCodeChallenge: () => "challenge",
    };
}

describe("auth manager durable session", () => {
    it("restores an encrypted sign-in session after recreation", async () => {
        const storage = new MemoryStorage();
        const first = createAuthManager(dependencies(storage));
        await first.signIn({ state: "fixed-id" });
        await new Promise<void>((resolve) => setImmediate(resolve));
        await new Promise<void>((resolve) => setImmediate(resolve));
        first.dispose();

        const encrypted = storage.files.get("auth/tokens.enc");
        expect(encrypted?.toString("utf8")).toContain("encrypted:");
        expect(encrypted?.toString("utf8")).not.toContain("access");

        const second = createAuthManager(dependencies(storage));
        await expect(second.getAccessToken()).resolves.toBe("access");
        second.dispose();
    });
});
