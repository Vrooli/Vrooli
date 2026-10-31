/**
 * Auth Manager Tests
 *
 * DOC: docs/internal/SEAMS.md#auth-manager-tests
 *
 * Tests for the authentication manager with focus on:
 * - Token storage and encryption
 * - Sign in/out flows
 * - Token refresh scheduling
 * - CSRF protection
 */

import { describe, it, expect, beforeEach, vi } from "vitest";
import type {
    AuthManagerDependencies,
    IAuthManager,
    AuthConfig,
    IAuthStorage,
    ISafeStorage,
    IAuthHttpClient,
    IShell,
    IAuthTimer,
    IUuidGenerator,
    IAuthPathUtils,
    AuthChangeEvent,
} from "../types";
import { createAuthManager, parseConfiguredCredentialStatus } from "../manager";

// ===== Mock Factories =====

interface MockStorage extends IAuthStorage {
    _files: Map<string, string | Buffer>;
    _dirs: Set<string>;
}

function createMockStorage(): MockStorage {
    const files = new Map<string, string | Buffer>();
    const dirs = new Set<string>(["auth"]);

    return {
        _files: files,
        _dirs: dirs,

        resolvePath: vi.fn(async (relativePath: string) => {
            // Simple mock - just return a "resolved" path
            return `/mock/storage/${relativePath}`;
        }),

        readFile: vi.fn(async (relativePath: string): Promise<Buffer | null> => {
            const content = files.get(relativePath);
            if (content === undefined) return null;
            return Buffer.isBuffer(content) ? content : Buffer.from(content);
        }),

        readTextFile: vi.fn(async (relativePath: string): Promise<string | null> => {
            const content = files.get(relativePath);
            if (content === undefined) return null;
            return Buffer.isBuffer(content) ? content.toString("utf-8") : content;
        }),

        writeFile: vi.fn(async (relativePath: string, data: string | Buffer) => {
            files.set(relativePath, data);
        }),

        deleteFile: vi.fn(async (relativePath: string): Promise<boolean> => {
            return files.delete(relativePath);
        }),

        ensureDir: vi.fn(async (_relativePath: string) => {
            // Just succeed
        }),
    };
}

interface MockSafeStorage extends ISafeStorage {
    _encryptionAvailable: boolean;
}

function createMockSafeStorage(encryptionAvailable = true): MockSafeStorage {
    return {
        _encryptionAvailable: encryptionAvailable,
        isEncryptionAvailable: vi.fn(() => encryptionAvailable),
        encryptString: vi.fn((data: string) => Buffer.from(`encrypted:${data}`)),
        decryptString: vi.fn((encrypted: Buffer) => {
            const str = encrypted.toString();
            if (str.startsWith("encrypted:")) {
                return str.slice(10);
            }
            return str;
        }),
    };
}

interface MockHttpResponse {
    ok: boolean;
    status: number;
    body: unknown;
}

interface MockHttpClient extends IAuthHttpClient {
    _responses: Map<string, MockHttpResponse>;
    _requests: Array<{ url: string; options?: { method?: string; headers?: Record<string, string>; body?: string } }>;
}

function createMockHttpClient(): MockHttpClient {
    const responses = new Map<string, MockHttpResponse>();
    const requests: Array<{ url: string; options?: { method?: string; headers?: Record<string, string>; body?: string } }> = [];

    return {
        _responses: responses,
        _requests: requests,

        fetch: vi.fn(async (url: string, options?: { method?: string; headers?: Record<string, string>; body?: string }) => {
            // Conditionally add options to satisfy exactOptionalPropertyTypes
            if (options !== undefined) {
                requests.push({ url, options });
            } else {
                requests.push({ url });
            }
            const response = responses.get(url) ?? { ok: true, status: 200, body: {} };
            return {
                ok: response.ok,
                status: response.status,
                json: async () => response.body,
            };
        }),
    };
}

function createMockShell(): IShell & { _openedUrls: string[] } {
    const openedUrls: string[] = [];
    return {
        _openedUrls: openedUrls,
        openExternal: vi.fn(async (url: string) => {
            openedUrls.push(url);
        }),
    };
}

interface MockTimer extends IAuthTimer {
    _currentTime: number;
    _scheduledCallbacks: Array<{ callback: () => void; executeAt: number; id: NodeJS.Timeout }>;
    _setTime(ms: number): void;
    _advanceTime(ms: number): void;
}

function createMockTimer(): MockTimer {
    let currentTime = Date.now();
    const scheduledCallbacks: Array<{ callback: () => void; executeAt: number; id: NodeJS.Timeout }> = [];
    let nextId = 1;

    return {
        _currentTime: currentTime,
        _scheduledCallbacks: scheduledCallbacks,
        _setTime(ms: number) { currentTime = ms; },
        _advanceTime(ms: number) {
            currentTime += ms;
            // Execute any callbacks that should fire
            const toExecute = scheduledCallbacks.filter(c => c.executeAt <= currentTime);
            for (const entry of toExecute) {
                const idx = scheduledCallbacks.indexOf(entry);
                if (idx >= 0) scheduledCallbacks.splice(idx, 1);
                entry.callback();
            }
        },
        now: vi.fn(() => currentTime),
        setTimeout: vi.fn((callback: () => void, delayMs: number) => {
            const id = { unref: () => {} } as unknown as NodeJS.Timeout;
            (id as unknown as { _id: number })._id = nextId++;
            scheduledCallbacks.push({ callback, executeAt: currentTime + delayMs, id });
            return id;
        }),
        clearTimeout: vi.fn((timer: NodeJS.Timeout) => {
            const idx = scheduledCallbacks.findIndex(c => c.id === timer);
            if (idx >= 0) scheduledCallbacks.splice(idx, 1);
        }),
    };
}

function createMockUuidGenerator(): IUuidGenerator & { _nextUuid: string } {
    return {
        _nextUuid: "test-uuid-12345",
        generate: vi.fn(function(this: { _nextUuid: string }) { return this._nextUuid; }),
    };
}

function createMockPathUtils(): IAuthPathUtils {
    return {
        dirname: vi.fn((p: string) => {
            const parts = p.split("/");
            parts.pop();
            return parts.join("/") || ".";
        }),
    };
}

function createTestConfig(overrides?: Partial<AuthConfig>): AuthConfig {
    return {
        protocol: "testapp",
        lpbsUrl: "https://test.vrooli.com",
        tokensFile: "auth/tokens.enc",
        userFile: "auth/user.json",
        tokenRefreshBufferMs: 5 * 60 * 1000, // 5 minutes
        appDisplayName: "Test App",
        ...overrides,
    };
}

function createTestDependencies(overrides?: Partial<AuthManagerDependencies>): AuthManagerDependencies & {
    storage: MockStorage;
    safeStorage: MockSafeStorage;
    http: MockHttpClient;
    shell: IShell & { _openedUrls: string[] };
    timer: MockTimer;
    uuid: IUuidGenerator & { _nextUuid: string };
    authChangeEvents: AuthChangeEvent[];
    windowFocusCalled: boolean;
    protocolUrls: string[];
} {
    const authChangeEvents: AuthChangeEvent[] = [];
    let windowFocusCalled = false;
    const protocolUrls: string[] = [];
    const shell = createMockShell();

    const deps: AuthManagerDependencies = {
        storage: createMockStorage(),
        safeStorage: createMockSafeStorage(),
        http: createMockHttpClient(),
        shell,
        timer: createMockTimer(),
        uuid: createMockUuidGenerator(),
        pathUtils: createMockPathUtils(),
        config: createTestConfig(),
        onAuthChange: (event) => authChangeEvents.push(event),
        onWindowFocus: () => { windowFocusCalled = true; },
        onProtocolUrl: (url) => protocolUrls.push(url),
        onLoopbackAuthorization: async (buildAuthorizationURL) => {
            await shell.openExternal(buildAuthorizationURL("http://127.0.0.1:43123/callback"));
            return await new Promise<never>(() => {});
        },
        createCodeChallenge: () => "test-challenge",
        ...overrides,
    };

    return {
        ...deps,
        storage: deps.storage as MockStorage,
        safeStorage: deps.safeStorage as MockSafeStorage,
        http: deps.http as MockHttpClient,
        shell: deps.shell as IShell & { _openedUrls: string[] },
        timer: deps.timer as MockTimer,
        uuid: deps.uuid as IUuidGenerator & { _nextUuid: string },
        authChangeEvents,
        get windowFocusCalled() { return windowFocusCalled; },
        protocolUrls,
    };
}

// ===== Tests =====

describe("parseConfiguredCredentialStatus", () => {
    it("accepts only object responses with an explicit configured boolean", () => {
        expect(parseConfiguredCredentialStatus({ configured: true })).toBe(true);
        expect(parseConfiguredCredentialStatus({ configured: false })).toBe(false);
        expect(() => parseConfiguredCredentialStatus("text response")).toThrow("credential status response is invalid");
        expect(() => parseConfiguredCredentialStatus({})).toThrow("credential status response is invalid");
        expect(() => parseConfiguredCredentialStatus({ configured: "true" })).toThrow("credential status response is invalid");
    });
});

describe("createAuthManager", () => {
    let deps: ReturnType<typeof createTestDependencies>;

    beforeEach(() => {
        deps = createTestDependencies();
    });

    async function signInInMemorySession(
        accessToken = "session-access",
        refreshToken = "session-refresh",
        expiresAt = "2099-01-01T00:00:00Z",
    ): Promise<IAuthManager> {
        deps = createTestDependencies({
            onLoopbackAuthorization: async (buildAuthorizationURL) => {
                const redirectURI = "http://127.0.0.1:43123/callback";
                const url = new URL(buildAuthorizationURL(redirectURI));
                return { code: "session-code", state: url.searchParams.get("state") ?? "", redirectURI };
            },
        });
        deps.http._responses.set("https://test.vrooli.com/api/v1/auth/token", {
            ok: true, status: 200,
            body: { access_token: accessToken, refresh_token: refreshToken, expires_at: expiresAt },
        });
        const manager = createAuthManager(deps);
        await manager.signIn();
        await vi.waitFor(() => expect(deps.authChangeEvents).toContain("tokens-received"));
        return manager;
    }

    describe("LPBS credit-session sharing", () => {
        it("returns only the authenticated LPBS wallet facts and records when they were fetched", async () => {
            const manager = await signInInMemorySession("wallet-access");
            const endpoint = "https://test.vrooli.com/landing_page_business_suite.v1.AccountService/GetMyCredits";
            deps.http._responses.set(endpoint, { ok: true, status: 200, body: {
                balance: { balanceCredits: "1250", updatedAt: "2026-10-19T12:00:00Z" },
                displayCreditsLabel: "credits", displayCreditsMultiplier: 1,
            } });
            const result = await manager.getCreditBalance();
            expect(result).toMatchObject({ state: "available", balanceCredits: "1250", displayCreditsLabel: "credits", displayCreditsMultiplier: 1, walletUpdatedAt: "2026-10-19T12:00:00Z", source: "LPBS" });
            expect(result.state === "available" && Date.parse(result.fetchedAt)).toBeTypeOf("number");
            const request = deps.http._requests.find((entry) => entry.url === endpoint);
            expect(request?.options).toMatchObject({ method: "POST", headers: { Authorization: "Bearer wallet-access", "Connect-Protocol-Version": "1" }, body: "{}" });
            expect(JSON.stringify(result)).not.toContain("wallet-access");
        });

        it("keeps missing or invalid wallet data unknown and makes no request while signed out", async () => {
            const manager = createAuthManager(deps);
            await expect(manager.getCreditBalance()).resolves.toEqual({ state: "signed_out" });
            expect(deps.http._requests).toHaveLength(0);
            const signedIn = await signInInMemorySession();
            const endpoint = "https://test.vrooli.com/landing_page_business_suite.v1.AccountService/GetMyCredits";
            deps.http._responses.set(endpoint, { ok: true, status: 200, body: { displayCreditsLabel: "credits", displayCreditsMultiplier: 1 } });
            await expect(signedIn.getCreditBalance()).resolves.toEqual({ state: "unavailable" });
        });

        it("provisions only after an explicit choice without persisting website credentials", async () => {
            const provision = vi.fn(async (_refreshToken: string) => {});
            const clear = vi.fn(async () => {});
            deps = createTestDependencies({
                onLoopbackAuthorization: async (buildAuthorizationURL) => {
                    const redirectURI = "http://127.0.0.1:43123/callback";
                    const url = new URL(buildAuthorizationURL(redirectURI));
                    return { code: "session-code", state: url.searchParams.get("state") ?? "", redirectURI };
                },
                onStoreLPBSCreditRefreshToken: provision,
                onHasLPBSCreditRefreshToken: async () => false,
                onClearLPBSCreditRefreshToken: clear,
            });
            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/token", {
                ok: true, status: 200, body: { access_token: "access-secret", refresh_token: "refresh-secret", expires_at: "2099-01-01T00:00:00Z" },
            });
            const manager = createAuthManager(deps);
            await manager.signIn();
            await vi.waitFor(async () => expect(await manager.getAccessToken()).toBe("access-secret"));

            await expect(manager.getCreditSessionShareStatus()).resolves.toEqual({ state: "not_shared" });
            await expect(manager.setCreditSessionSharing(true)).resolves.toEqual({ state: "shared" });
            expect(provision).toHaveBeenCalledOnce();
            expect(provision).toHaveBeenCalledWith("refresh-secret");
            expect(deps.storage._files.has("auth/tokens.enc")).toBe(false);
            expect(clear).not.toHaveBeenCalled();
            const restartedManager = createAuthManager({ ...deps, onHasLPBSCreditRefreshToken: async () => true });
            await expect(restartedManager.getCreditSessionShareStatus()).resolves.toEqual({ state: "shared" });
        });

        it("refuses to share while signed out and clears the exact refresh credential on opt-out and sign-out", async () => {
            const clear = vi.fn(async () => {});
            const manager = createAuthManager({
                ...deps,
                onStoreLPBSCreditRefreshToken: vi.fn(async () => {}),
                onHasLPBSCreditRefreshToken: async () => true,
                onClearLPBSCreditRefreshToken: clear,
            });
            await expect(manager.setCreditSessionSharing(true)).resolves.toEqual({ state: "signed_out" });
            await expect(manager.setCreditSessionSharing(false)).resolves.toEqual({ state: "not_shared", localCleared: true, remoteRevocation: "uncertain" });
            await manager.signOut();
            expect(clear).toHaveBeenCalledTimes(2);
        });

        it("replaces the shared owner credential when LPBS rotates a refresh token", async () => {
            const provision = vi.fn(async (_refreshToken: string) => {});
            const clear = vi.fn(async () => {});
            const expiresAt = "2099-01-01T00:00:00Z";
            deps = createTestDependencies({
                onLoopbackAuthorization: async (buildAuthorizationURL) => {
                    const redirectURI = "http://127.0.0.1:43123/callback";
                    const url = new URL(buildAuthorizationURL(redirectURI));
                    return { code: "session-code", state: url.searchParams.get("state") ?? "", redirectURI };
                },
                onStoreLPBSCreditRefreshToken: provision,
                onClearLPBSCreditRefreshToken: clear,
            });
            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/token", {
                ok: true, status: 200, body: { access_token: "old-access", refresh_token: "old-refresh", expires_at: expiresAt },
            });
            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/refresh", { ok: true, status: 200, body: { access_token: "new-access", refresh_token: "new-refresh", expires_at: expiresAt } });
            const manager = createAuthManager(deps);
            await manager.signIn();
            await vi.waitFor(async () => expect(await manager.getAccessToken()).toBe("old-access"));
            await expect(manager.setCreditSessionSharing(true)).resolves.toMatchObject({ state: "shared" });

            await expect(manager.refresh()).resolves.toBe(true);
            expect(provision).toHaveBeenCalledWith("new-refresh");
            expect(provision).toHaveBeenCalledTimes(2);
            expect(clear).not.toHaveBeenCalled();
        });
    });

    describe("signIn", () => {
        it("opens a loopback PKCE URL with correct parameters", async () => {
            const manager = createAuthManager(deps);

            await manager.signIn();

            expect(deps.shell._openedUrls.length).toBe(1);
            const url = new URL(deps.shell._openedUrls[0] ?? "");
            expect(url.origin).toBe("https://test.vrooli.com");
            expect(url.pathname).toBe("/auth/login");
            expect(url.searchParams.get("redirect_uri")).toBe("http://127.0.0.1:43123/callback");
            expect(url.searchParams.get("code_challenge")).toBe("test-challenge");
            expect(url.searchParams.get("code_challenge_method")).toBe("S256");
            expect(url.searchParams.get("app")).toBe("Test App");
            expect(url.searchParams.get("state")).toBe("test-uuid-12345");
        });

        it("uses provided state parameter", async () => {
            const manager = createAuthManager(deps);

            const result = await manager.signIn({ state: "custom-state" });

            expect(result.state).toBe("custom-state");
            const url = new URL(deps.shell._openedUrls[0] ?? "");
            expect(url.searchParams.get("state")).toBe("custom-state");
        });

        it("returns generated state for CSRF validation", async () => {
            const manager = createAuthManager(deps);

            const result = await manager.signIn();

            expect(result.state).toBe("test-uuid-12345");
        });
    });

    describe("connectDesktop", () => {
        it("redeems a browser-issued scoped link with a verified local proof and persists only the lease", async () => {
            let openedUrl = "";
            const storeLease = vi.fn(async (_lease: string) => {});
            deps = createTestDependencies({
                onLoopbackAuthorization: async (buildAuthorizationURL) => {
                    openedUrl = buildAuthorizationURL("http://127.0.0.1:43123/callback");
                    return { code: "link-code", state: "link-state", redirectURI: "http://127.0.0.1:43123/callback" };
                },
                onResolveLocalIdentityProof: async () => "local-proof",
                onStoreEntitlementLease: storeLease,
            });
            deps.http._responses.set("https://test.vrooli.com/api/v1/desktop/links/redeem", {
                ok: true,
                status: 200,
                body: {
                    link: { local_principal: "local-1", business_account_id: "business-1", installation_id: "install-1", resource: "demo", audience: "scenario:demo", scopes: ["demo:read"] },
                    entitlement_lease: { token: "signed-lease", expires_at: "2030-01-01T00:00:00Z" },
                },
            });
            const manager = createAuthManager(deps);

            const linked = await manager.connectDesktop({
                installationId: "install-1",
                resource: "demo",
                audience: "scenario:demo",
                scopes: ["demo:read"],
                state: "link-state",
            });

            const authUrl = new URL(openedUrl);
            expect(authUrl.pathname).toBe("/auth/login");
            expect(authUrl.searchParams.get("desktop_link")).toBe("true");
            expect(authUrl.searchParams.get("installation_id")).toBe("install-1");
            expect(authUrl.searchParams.get("audience")).toBe("scenario:demo");
            expect(authUrl.searchParams.get("scopes")).toBe("demo:read");
            const redemption = deps.http._requests.find((request) => request.url.endsWith("/desktop/links/redeem"));
            expect(redemption?.options?.headers?.Authorization).toBe("Bearer local-proof");
            expect(JSON.parse(String(redemption?.options?.body))).toMatchObject({
                code: "link-code",
                installation_id: "install-1",
                resource: "demo",
            });
            expect(await manager.getEntitlementLease()).toBe("signed-lease");
            expect(linked).toEqual({ state: "linked", localPrincipal: "local-1", businessAccountId: "business-1", scopes: ["demo:read"] });
            expect(await manager.getAccessToken()).toBeNull();
            expect(await manager.isAuthenticated()).toBe(false);
            expect(storeLease).toHaveBeenCalledWith("signed-lease");
            const persisted = deps.safeStorage.decryptString(deps.storage._files.get("auth/tokens.enc") as Buffer);
            expect(JSON.parse(persisted)).toEqual({ entitlementLease: "signed-lease" });
        });

        it("rejects a cross-resource audience before opening the browser", async () => {
            const manager = createAuthManager({
                ...deps,
                onResolveLocalIdentityProof: async () => "local-proof",
            });

            await expect(manager.connectDesktop({
                installationId: "install-1",
                resource: "demo",
                audience: "scenario:other",
                scopes: ["demo:read"],
            })).rejects.toThrow("desktop link binding is invalid");
            expect(deps.shell._openedUrls).toHaveLength(0);
        });

        it("loads only the bound LPBS local link and maps missing, expired and unavailable states", async () => {
            deps = createTestDependencies({
                onResolveLocalIdentityProof: async () => "local-proof",
                onGetDesktopInstallationId: async () => "install-1",
            });
            const manager = createAuthManager(deps);
            const statusURL = "https://test.vrooli.com/api/v1/desktop/links/local?installation_id=install-1&resource=demo";
            deps.http._responses.set(statusURL, { ok: true, status: 200, body: {
                link: { local_principal: "local-1", business_account_id: "business-1", installation_id: "install-1", resource: "demo", audience: "scenario:demo", scopes: ["demo:read"] },
            } });
            await expect(manager.getDesktopLinkStatus({ resource: "demo" })).resolves.toMatchObject({ state: "linked", businessAccountId: "business-1", scopes: ["demo:read"] });
            expect(deps.http._requests.at(-1)?.options?.headers?.Authorization).toBe("Bearer local-proof");

            deps.http._responses.set(statusURL, { ok: false, status: 404, body: {} });
            await expect(manager.getDesktopLinkStatus({ resource: "demo" })).resolves.toMatchObject({ state: "unlinked" });
            deps.http._responses.set(statusURL, { ok: false, status: 401, body: {} });
            await expect(manager.getDesktopLinkStatus({ resource: "demo" })).resolves.toMatchObject({ state: "expired" });
            deps.http._responses.set(statusURL, { ok: false, status: 503, body: {} });
            await expect(manager.getDesktopLinkStatus({ resource: "demo" })).resolves.toMatchObject({ state: "unavailable" });
            deps.http._responses.set(statusURL, { ok: true, status: 200, body: { link: {
                local_principal: "local-1", business_account_id: "business-1", installation_id: "other-install", resource: "demo", audience: "scenario:demo", scopes: ["demo:read"],
            } } });
            await expect(manager.getDesktopLinkStatus({ resource: "demo" })).resolves.toMatchObject({ state: "unavailable", scopes: [] });
        });

        it("clears local auth even when remote unlink cannot be confirmed", async () => {
            const clearLease = vi.fn(async () => {});
            deps = createTestDependencies({
                onResolveLocalIdentityProof: async () => { throw new Error("offline"); },
                onGetDesktopInstallationId: async () => "install-1",
                onClearEntitlementLease: clearLease,
            });
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify({ entitlementLease: "opaque-lease" })));
            const manager = createAuthManager(deps);

            await expect(manager.disconnectDesktop({ resource: "demo" })).resolves.toEqual({ localCleared: true, remoteRevocation: "uncertain" });
            expect(deps.storage._files.has("auth/tokens.enc")).toBe(false);
            expect(clearLease).toHaveBeenCalledOnce();
        });

        it("reports failed shared LPBS credential cleanup when remote unlink is uncertain", async () => {
            const clearCreditCredential = vi.fn(async () => { throw new Error("credential owner unavailable"); });
            deps = createTestDependencies({
                onResolveLocalIdentityProof: async () => { throw new Error("offline"); },
                onGetDesktopInstallationId: async () => "install-1",
                onClearLPBSCreditRefreshToken: clearCreditCredential,
            });
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify({ entitlementLease: "opaque-lease" })));
            const manager = createAuthManager(deps);

            await expect(manager.disconnectDesktop({ resource: "demo" })).resolves.toEqual({ localCleared: false, remoteRevocation: "uncertain" });
            expect(deps.storage._files.has("auth/tokens.enc")).toBe(false);
            expect(clearCreditCredential).toHaveBeenCalledOnce();
            expect(deps.http._requests).toHaveLength(0);
        });

        it("reports confirmed remote revoke separately from local cleanup failure", async () => {
            deps = createTestDependencies({
                onResolveLocalIdentityProof: async () => "local-proof",
                onGetDesktopInstallationId: async () => "install-1",
                onClearEntitlementLease: async () => { throw new Error("credential owner unavailable"); },
            });
            const manager = createAuthManager(deps);
            deps.http._responses.set("https://test.vrooli.com/api/v1/desktop/links/local", { ok: true, status: 204, body: {} });

            await expect(manager.disconnectDesktop({ resource: "demo" })).resolves.toEqual({ localCleared: false, remoteRevocation: "confirmed" });
            const request = deps.http._requests.find((entry) => entry.options?.method === "DELETE");
            expect(request?.url).toBe("https://test.vrooli.com/api/v1/desktop/links/local");
            expect(request?.options?.headers?.Authorization).toBe("Bearer local-proof");
            expect(JSON.parse(String(request?.options?.body))).toEqual({ installation_id: "install-1", resource: "demo" });
        });

        it("does not treat an unexpected successful status as confirmed revocation", async () => {
            deps = createTestDependencies({
                onResolveLocalIdentityProof: async () => "local-proof",
                onGetDesktopInstallationId: async () => "install-1",
            });
            deps.http._responses.set("https://test.vrooli.com/api/v1/desktop/links/local", { ok: true, status: 200, body: {} });
            const manager = createAuthManager(deps);

            await expect(manager.disconnectDesktop({ resource: "demo" })).resolves.toEqual({ localCleared: true, remoteRevocation: "uncertain" });
        });
    });

    describe("handleCallback", () => {
        it("rejects callbacks that carry tokens", async () => {
            const manager = createAuthManager(deps);
            const callback = new URL("testapp://auth/callback");
            callback.hash = new URLSearchParams({
                access_token: "at123",
                refresh_token: "rt456",
            }).toString();
            await manager.handleCallback(callback.toString());

            expect(deps.authChangeEvents).toContain("session-expired");
            expect(deps.storage._files.has("auth/tokens.enc")).toBe(false);
        });

        it("forwards non-auth callbacks to protocol URL handler", async () => {
            const manager = createAuthManager(deps);

            await manager.handleCallback("testapp://some/other/path");

            expect(deps.protocolUrls).toContain("testapp://some/other/path");
            expect(deps.authChangeEvents).toHaveLength(0);
        });

        it("forwards code-only callbacks without treating them as credentials", async () => {
            const manager = createAuthManager(deps);
            await manager.handleCallback("testapp://auth/callback?code=one-use-code&state=test-uuid-12345");
            expect(deps.protocolUrls).toContain("testapp://auth/callback?code=one-use-code&state=test-uuid-12345");
            expect(deps.authChangeEvents).toHaveLength(0);
        });
    });

    describe("signOut", () => {
        let authenticatedManager: ReturnType<typeof createAuthManager>;

        beforeEach(async () => {
            // A restart restores only the signed entitlement lease. Website
            // access and refresh tokens must not be durable.
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify({ entitlementLease: "signed-lease" })));
            authenticatedManager = createAuthManager(deps);
            await authenticatedManager.initialize();
            deps.authChangeEvents.length = 0; // Clear events
            deps.http._requests.length = 0; // Clear requests from setup
        });

        it("clears stored tokens", async () => {
            await authenticatedManager.signOut();

            expect(deps.storage._files.has("auth/tokens.enc")).toBe(false);
        });

        it("clears stored user info", async () => {
            deps.storage._files.set("auth/user.json", JSON.stringify({ id: "user123" }));

            await authenticatedManager.signOut();

            expect(deps.storage._files.has("auth/user.json")).toBe(false);
        });

        it("does not call a website logout endpoint after restart", async () => {
            await authenticatedManager.signOut();

            const logoutRequest = deps.http._requests.find(r => r.url.includes("/logout"));
            expect(logoutRequest).toBeUndefined();
        });

        it("notifies signed-out event", async () => {
            await authenticatedManager.signOut();

            expect(deps.authChangeEvents).toContain("signed-out");
        });

        it("clears refresh timer", async () => {
            expect(deps.timer._scheduledCallbacks.length).toBe(0);

            await authenticatedManager.signOut();

            expect(deps.timer._scheduledCallbacks.length).toBe(0);
        });
    });

    describe("getAccessToken", () => {
        it("returns null when not authenticated", async () => {
            const manager = createAuthManager(deps);

            const token = await manager.getAccessToken();

            expect(token).toBeNull();
        });

        it("does not restore an access token from durable storage", async () => {
            const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString(); // 1 hour from now
            const tokens = { accessToken: "valid-token", refreshToken: "rt", expiresAt: futureDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const manager = createAuthManager(deps);
            const token = await manager.getAccessToken();

            expect(token).toBeNull();
        });

        it("does not refresh a website session after restart", async () => {
            const pastDate = new Date(Date.now() - 1000).toISOString(); // Already expired
            const tokens = { accessToken: "old-token", refreshToken: "rt", expiresAt: pastDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString();
            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/refresh", {
                ok: true,
                status: 200,
                body: { access_token: "new-token", refresh_token: "new-rt", expires_at: futureDate },
            });

            const manager = createAuthManager(deps);
            const token = await manager.getAccessToken();

            expect(token).toBeNull();
            expect(deps.http._requests).toHaveLength(0);
        });

        it("reports an expired in-memory session when token refresh fails", async () => {
            const pastDate = new Date(Date.now() - 1000).toISOString();
            const manager = await signInInMemorySession("old-token", "rt", pastDate);
            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/refresh", {
                ok: false,
                status: 401,
                body: { error: "invalid_grant" },
            });

            const token = await manager.getAccessToken();

            expect(token).toBeNull();
            expect(deps.authChangeEvents).toContain("session-expired");
            expect(deps.http._requests.some((request) => request.url.endsWith("/auth/refresh"))).toBe(true);
        });
    });

    describe("getUser", () => {
        it("returns null when no user stored", async () => {
            const manager = createAuthManager(deps);

            const user = await manager.getUser();

            expect(user).toBeNull();
        });

        it("returns stored user info", async () => {
            const userInfo = { id: "user123", email: "test@example.com", emailVerified: true };
            deps.storage._files.set("auth/user.json", JSON.stringify(userInfo));

            const manager = createAuthManager(deps);
            const user = await manager.getUser();

            expect(user).toEqual(userInfo);
        });
    });

    describe("isAuthenticated", () => {
        it("returns false when no tokens", async () => {
            const manager = createAuthManager(deps);

            const authenticated = await manager.isAuthenticated();

            expect(authenticated).toBe(false);
        });

        it("does not treat durable website credentials as authenticated", async () => {
            const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString();
            const tokens = { accessToken: "at", refreshToken: "rt", expiresAt: futureDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const manager = createAuthManager(deps);
            const authenticated = await manager.isAuthenticated();

            expect(authenticated).toBe(false);
        });

        it("does not treat durable credentials within the refresh buffer as authenticated", async () => {
            // Token expires in 2 minutes, but buffer is 5 minutes
            const almostExpired = new Date(Date.now() + 2 * 60 * 1000).toISOString();
            const tokens = { accessToken: "at", refreshToken: "rt", expiresAt: almostExpired };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const manager = createAuthManager(deps);
            const authenticated = await manager.isAuthenticated();

            expect(authenticated).toBe(false);
        });

        it("returns false when expired beyond buffer", async () => {
            // Token expired 10 minutes ago
            const pastDate = new Date(Date.now() - 10 * 60 * 1000).toISOString();
            const tokens = { accessToken: "at", refreshToken: "rt", expiresAt: pastDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const manager = createAuthManager(deps);
            const authenticated = await manager.isAuthenticated();

            expect(authenticated).toBe(false);
        });
    });

    describe("refresh", () => {
        it("refuses to refresh a website session after restart", async () => {
            const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString();
            const tokens = { accessToken: "old", refreshToken: "rt", expiresAt: futureDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const newFuture = new Date(Date.now() + 2 * 60 * 60 * 1000).toISOString();
            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/refresh", {
                ok: true,
                status: 200,
                body: { access_token: "new", refresh_token: "new-rt", expires_at: newFuture },
            });

            const manager = createAuthManager(deps);
            const success = await manager.refresh();

            expect(success).toBe(false);
            expect(deps.authChangeEvents).toContain("session-expired");
        });

        it("returns false when no refresh token", async () => {
            const manager = createAuthManager(deps);

            const success = await manager.refresh();

            expect(success).toBe(false);
            expect(deps.authChangeEvents).toContain("session-expired");
        });

        it("returns false on API error", async () => {
            const manager = await signInInMemorySession();
            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/refresh", {
                ok: false,
                status: 401,
                body: { error: "invalid_grant" },
            });

            const success = await manager.refresh();

            expect(success).toBe(false);
            expect(deps.http._requests.some((request) => request.url.endsWith("/auth/refresh"))).toBe(true);
        });
    });

    describe("initialize", () => {
        it("does not schedule a website refresh from durable storage", async () => {
            const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString(); // 1 hour
            const tokens = { accessToken: "at", refreshToken: "rt", expiresAt: futureDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const manager = createAuthManager(deps);
            await manager.initialize();

            expect(deps.timer._scheduledCallbacks.length).toBe(0);
        });

        it("does nothing when no tokens exist", async () => {
            const manager = createAuthManager(deps);

            await manager.initialize();

            expect(deps.timer._scheduledCallbacks.length).toBe(0);
        });

        it("does not schedule refresh for expired tokens", async () => {
            const pastDate = new Date(Date.now() - 1000).toISOString();
            const tokens = { accessToken: "at", refreshToken: "rt", expiresAt: pastDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const manager = createAuthManager(deps);
            await manager.initialize();

            expect(deps.timer._scheduledCallbacks.length).toBe(0);
        });
    });

    describe("dispose", () => {
        it("clears refresh timer", async () => {
            const futureDate = new Date(Date.now() + 60 * 60 * 1000).toISOString();
            const tokens = { accessToken: "at", refreshToken: "rt", expiresAt: futureDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            const manager = createAuthManager(deps);
            await manager.initialize();
            expect(deps.timer._scheduledCallbacks.length).toBe(0);

            manager.dispose();

            expect(deps.timer._scheduledCallbacks.length).toBe(0);
        });
    });

    describe("credential recovery boundary", () => {
        it("does not recover a website refresh token from the platform authority", async () => {
            deps = createTestDependencies({
                onGetEntitlementLease: async () => "signed-lease",
            });

            const manager = createAuthManager(deps);
            const token = await manager.getAccessToken();

            expect(token).toBeNull();
            expect(await manager.getEntitlementLease()).toBe("signed-lease");
            expect(deps.http._requests).toHaveLength(0);
        });

        it("does not parse an unreadable encrypted file as a website credential", async () => {
            deps.storage._files.set("auth/tokens.enc", "not-an-encrypted-token-file");
            const manager = createAuthManager(deps);

            expect(await manager.getAccessToken()).toBeNull();
            expect(deps.http._requests).toHaveLength(0);
        });

        it("deletes plaintext legacy credentials without recovering them", async () => {
            deps = createTestDependencies({
                safeStorage: createMockSafeStorage(false),
                onGetEntitlementLease: async () => "signed-lease",
            });
            deps.storage._files.set("auth/tokens.enc", JSON.stringify({ accessToken: "plaintext-access", refreshToken: "rt" }));

            const manager = createAuthManager(deps);

            expect(await manager.getAccessToken()).toBeNull();
            expect(await manager.getEntitlementLease()).toBe("signed-lease");
            expect(deps.storage._files.has("auth/tokens.enc")).toBe(false);
        });
    });
});
