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
import { createAuthManager } from "../manager";

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

describe("createAuthManager", () => {
    let deps: ReturnType<typeof createTestDependencies>;

    beforeEach(() => {
        deps = createTestDependencies();
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
                body: { entitlement_lease: { token: "signed-lease", expires_at: "2030-01-01T00:00:00Z" } },
            });
            const manager = createAuthManager(deps);

            await manager.connectDesktop({
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

        it("returns null when refresh fails", async () => {
            const pastDate = new Date(Date.now() - 1000).toISOString();
            const tokens = { accessToken: "old-token", refreshToken: "rt", expiresAt: pastDate };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/refresh", {
                ok: false,
                status: 401,
                body: { error: "invalid_grant" },
            });

            const manager = createAuthManager(deps);
            const token = await manager.getAccessToken();

            expect(token).toBeNull();
            expect(deps.authChangeEvents).not.toContain("session-expired");
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
            const tokens = { accessToken: "at", refreshToken: "rt", expiresAt: new Date().toISOString() };
            deps.storage._files.set("auth/tokens.enc", deps.safeStorage.encryptString(JSON.stringify(tokens)));

            deps.http._responses.set("https://test.vrooli.com/api/v1/auth/refresh", {
                ok: false,
                status: 401,
                body: { error: "invalid_grant" },
            });

            const manager = createAuthManager(deps);
            const success = await manager.refresh();

            expect(success).toBe(false);
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
