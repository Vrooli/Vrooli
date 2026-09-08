import { createServer, type Server } from "node:http";
import { randomUUID } from "node:crypto";
import { unlink } from "node:fs/promises";
import path from "node:path";
import { tmpdir } from "node:os";
import { afterEach, describe, expect, it } from "vitest";
import { resolveLocalIdentityProof } from "./local-identity-proof";

const originalSocket = process.env.VROOLI_AUTH_SOCKET;
let server: Server | undefined;
let socketPath = "";

afterEach(async () => {
    server?.close();
    server = undefined;
    if (socketPath) await unlink(socketPath).catch(() => undefined);
    socketPath = "";
    if (originalSocket === undefined) delete process.env.VROOLI_AUTH_SOCKET;
    else process.env.VROOLI_AUTH_SOCKET = originalSocket;
});

describe("resolveLocalIdentityProof", () => {
    it("exchanges the local socket peer for an authenticator proof", async () => {
        socketPath = path.join(tmpdir(), `vrooli-auth-proof-${randomUUID()}.sock`);
        server = createServer((request, response) => {
            expect(request.url).toContain("ExchangeMachinePrincipal");
            expect(request.headers["content-type"]).toBe("application/json");
            response.statusCode = 200;
            response.setHeader("Content-Type", "application/json");
            response.end(JSON.stringify({ tokens: { accessToken: "local-proof" } }));
        });
        await new Promise<void>((resolve, reject) => {
            server!.once("error", reject);
            server!.listen(socketPath, resolve);
        });
        process.env.VROOLI_AUTH_SOCKET = socketPath;

        await expect(resolveLocalIdentityProof()).resolves.toBe("local-proof");
    });
});
