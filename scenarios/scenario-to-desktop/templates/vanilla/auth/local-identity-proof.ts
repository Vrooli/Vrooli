import { request as httpRequest } from "node:http";
import { hostname, tmpdir } from "node:os";
import path from "node:path";

const EXCHANGE_PATH = "/vrooli.scenario_authenticator.v1.accounts.AccountsService/ExchangeMachinePrincipal";

function defaultSocketPath(): string {
    return path.join(tmpdir(), "vrooli-scenario-authenticator-scenario-authenticator.sock");
}

/**
 * Exchange the desktop process' Unix-socket peer credential for a short-lived
 * scenario-authenticator access token. The token is used only in memory as
 * LPBS's declared local identity proof; it is never written to desktop auth
 * storage or returned through a browser callback.
 */
export async function resolveLocalIdentityProof(): Promise<string> {
    const socketPath = process.env.VROOLI_AUTH_SOCKET?.trim() || defaultSocketPath();
    const machineID = hostname().trim();
    if (!machineID) throw new Error("local machine identity is unavailable");
    const body = JSON.stringify({ machineId: machineID });

    return new Promise((resolve, reject) => {
        const request = httpRequest({
            socketPath,
            path: EXCHANGE_PATH,
            method: "POST",
            headers: {
                "Content-Type": "application/json",
                "Content-Length": Buffer.byteLength(body),
            },
            timeout: 5000,
        }, (response) => {
            const chunks: Buffer[] = [];
            response.on("data", (chunk: Buffer | string) => chunks.push(Buffer.from(chunk)));
            response.on("end", () => {
                if (response.statusCode !== 200) {
                    reject(new Error(`local identity exchange failed (${response.statusCode ?? "unknown"})`));
                    return;
                }
                try {
                    const payload = JSON.parse(Buffer.concat(chunks).toString("utf8")) as {
                        tokens?: { accessToken?: unknown; access_token?: unknown };
                    };
                    const token = payload.tokens?.accessToken ?? payload.tokens?.access_token;
                    if (typeof token !== "string" || !token.trim()) {
                        reject(new Error("local identity exchange returned no access token"));
                        return;
                    }
                    resolve(token);
                } catch {
                    reject(new Error("local identity exchange returned invalid JSON"));
                }
            });
        });
        request.on("error", reject);
        request.on("timeout", () => request.destroy(new Error("local identity exchange timed out")));
        request.end(body);
    });
}
