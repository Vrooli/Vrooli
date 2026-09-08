/**
 * Auth Manager Implementation
 *
 * DOC: docs/internal/SEAMS.md#auth-manager
 *
 * Handles authentication flow, token storage, and refresh scheduling.
 * Uses Electron's safeStorage for secure token encryption.
 */

import type {
    IAuthManager,
    AuthManagerDependencies,
    StoredTokens,
    StoredUser,
    DesktopLinkOptions,
} from "./types";

/**
 * Create an auth manager with injected dependencies.
 */
export function createAuthManager(deps: AuthManagerDependencies): IAuthManager {
    const {
        storage,
        safeStorage,
        http,
        timer,
        uuid,
        pathUtils,
        config,
        onAuthChange,
        onProtocolUrl,
        onLoopbackAuthorization,
        createCodeChallenge,
        onStoreEntitlementLease,
        onGetEntitlementLease,
        onClearEntitlementLease,
        onResolveLocalIdentityProof,
    } = deps;

    let tokenRefreshTimer: NodeJS.Timeout | null = null;
    let memoryTokens: StoredTokens | null = null;

    // The credential authority is the recovery source for the signed lease.
    // It intentionally does not recover website access or refresh tokens.
    async function getAuthorityLease(): Promise<string | null> {
        const lease = await onGetEntitlementLease?.();
        return lease || null;
    }

    async function leaseOnlyTokens(lease: string | null): Promise<StoredTokens | null> {
        if (!lease) return null;
        return {
            accessToken: "",
            refreshToken: "",
            expiresAt: new Date(0).toISOString(),
            entitlementLease: lease,
        };
    }

    /**
     * Store tokens securely using Electron's safeStorage.
     */
    async function storeAuthTokens(tokens: StoredTokens): Promise<void> {
        const tokensPath = await storage.resolvePath(config.tokensFile);
        if (!tokensPath) {
            throw new Error("Invalid storage path for tokens");
        }

        // Ensure parent directory exists
        const parentDir = pathUtils.dirname(config.tokensFile);
        if (parentDir && parentDir !== ".") {
            await storage.ensureDir(parentDir);
        }

        // Persist only the signed lease. Website access and refresh tokens are
        // process-memory compatibility state and never cross this durable
        // storage boundary.
        const persisted = tokens.entitlementLease ? JSON.stringify({ entitlementLease: tokens.entitlementLease }) : "";
        if (safeStorage.isEncryptionAvailable()) {
            if (persisted) {
                const encrypted = safeStorage.encryptString(persisted);
                await storage.writeFile(config.tokensFile, encrypted);
            } else {
                try {
                    await storage.deleteFile(config.tokensFile);
                } catch (error: unknown) {
                    const code = error instanceof Error && "code" in error ? error.code : undefined;
                    if (code !== "ENOENT") throw error;
                }
            }
        } else {
            // The platform credential authority is the secure fallback. Never
            // place a lease, access token, or refresh token in an unencrypted file.
            if (tokens.entitlementLease && onStoreEntitlementLease) {
                await onStoreEntitlementLease(tokens.entitlementLease);
            }
            try {
                await storage.deleteFile(config.tokensFile);
            } catch (error: unknown) {
                const code = error instanceof Error && "code" in error ? error.code : undefined;
                if (code !== "ENOENT") throw error;
            }
        }

        if (tokens.entitlementLease && onStoreEntitlementLease && safeStorage.isEncryptionAvailable()) {
            await onStoreEntitlementLease(tokens.entitlementLease);
        }
        memoryTokens = tokens;
    }

    /**
     * Retrieve stored tokens.
     */
    async function getStoredTokens(): Promise<StoredTokens | null> {
        if (memoryTokens) return memoryTokens;
        try {
            const fileContent = await storage.readFile(config.tokensFile);
            if (!safeStorage.isEncryptionAvailable()) {
                // Plaintext auth files are not accepted. Recover only the
                // signed lease from the platform authority when available.
                if (fileContent) {
                    try {
                        await storage.deleteFile(config.tokensFile);
                    } catch {
                        // The authority remains the source of truth even if
                        // stale local cleanup cannot be completed.
                    }
                }
                return leaseOnlyTokens(await getAuthorityLease());
            }

            if (!fileContent) return leaseOnlyTokens(await getAuthorityLease());

            const decrypted = safeStorage.decryptString(fileContent);
            const parsed = JSON.parse(decrypted) as {
                entitlementLease?: unknown;
                accessToken?: unknown;
                refreshToken?: unknown;
            };
            if (typeof parsed.accessToken === "string" || typeof parsed.refreshToken === "string") {
                // A previous version persisted website credentials. Refuse to
                // import them and recover only a separately stored lease.
                try {
                    await storage.deleteFile(config.tokensFile);
                } catch {
                    // The old credential is unusable even if cleanup fails.
                }
                return leaseOnlyTokens(await getAuthorityLease());
            }
            return leaseOnlyTokens(
                typeof parsed.entitlementLease === "string"
                    ? parsed.entitlementLease
                    : await getAuthorityLease(),
            );
        } catch (error: unknown) {
            if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                return leaseOnlyTokens(await getAuthorityLease());
            }
            // Do not log the parse/decryption error: runtimes may include a
            // prefix of the unreadable file contents in the exception text.
            // The shared credential authority is the only recovery source.
            console.error("[Auth] Failed to read encrypted tokens; recovering from the shared authority");
            // A keychain/profile reset can make an otherwise valid encrypted
            // file unreadable. Recover only the lease from the platform
            // authority; never fall back to plaintext parsing.
            return leaseOnlyTokens(await getAuthorityLease());
        }
    }

    /**
     * Store user info.
     */
    async function storeUserInfo(user: StoredUser): Promise<void> {
        // Ensure parent directory exists
        const parentDir = pathUtils.dirname(config.userFile);
        if (parentDir && parentDir !== ".") {
            await storage.ensureDir(parentDir);
        }

        await storage.writeFile(config.userFile, JSON.stringify(user, null, 2));
    }

    /**
     * Get stored user info.
     */
    async function getStoredUser(): Promise<StoredUser | null> {
        try {
            const content = await storage.readTextFile(config.userFile);
            if (!content) {
                return null;
            }
            return JSON.parse(content) as StoredUser;
        } catch (error: unknown) {
            if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                return null;
            }
            console.error("[Auth] Failed to read user info:", error);
            return null;
        }
    }

    /**
     * Clear all auth data.
     */
    async function clearAuthData(): Promise<void> {
        memoryTokens = null;
        if (tokenRefreshTimer) {
            timer.clearTimeout(tokenRefreshTimer);
            tokenRefreshTimer = null;
        }

        try {
            await storage.deleteFile(config.tokensFile);
        } catch (error: unknown) {
            const code = error instanceof Error && "code" in error ? error.code : undefined;
            if (code !== "ENOENT") {
                console.error("[Auth] Failed to delete tokens:", error);
            }
        }

        try {
            await storage.deleteFile(config.userFile);
        } catch (error: unknown) {
            const code = error instanceof Error && "code" in error ? error.code : undefined;
            if (code !== "ENOENT") {
                console.error("[Auth] Failed to delete user info:", error);
            }
        }

        if (onClearEntitlementLease) {
            try {
                await onClearEntitlementLease();
            } catch (error) {
                console.warn("[Auth] Failed to clear entitlement lease:", error);
            }
        }
    }

    /**
     * Schedule token refresh before expiry.
     */
    function scheduleTokenRefresh(expiresAt: string): void {
        if (tokenRefreshTimer) {
            timer.clearTimeout(tokenRefreshTimer);
        }

        const expiryTime = new Date(expiresAt).getTime();
        const refreshTime = expiryTime - config.tokenRefreshBufferMs;
        const delay = Math.max(0, refreshTime - timer.now());

        tokenRefreshTimer = timer.setTimeout(async () => {
            await refreshTokens();
        }, delay);

        console.log(`[Auth] Token refresh scheduled in ${Math.round(delay / 1000 / 60)} minutes`);
    }

    /**
     * Refresh tokens using the refresh token.
     */
    async function refreshTokens(): Promise<boolean> {
        const tokens = await getStoredTokens();
        if (!tokens?.refreshToken) {
            console.log("[Auth] No refresh token available");
            onAuthChange("session-expired");
            return false;
        }

        try {
            const response = await http.fetch(`${config.lpbsUrl}/api/v1/auth/refresh`, {
                method: "POST",
                headers: { "Content-Type": "application/json" },
                body: JSON.stringify({ refresh_token: tokens.refreshToken }),
            });

            if (!response.ok) {
                console.error("[Auth] Token refresh failed:", response.status);
                onAuthChange("session-expired");
                return false;
            }

            const newTokens = await response.json() as {
                access_token: string;
                refresh_token: string;
                expires_at: string;
            };

            await storeAuthTokens({
                accessToken: newTokens.access_token,
                refreshToken: newTokens.refresh_token,
                expiresAt: newTokens.expires_at,
            });
            scheduleTokenRefresh(newTokens.expires_at);
            onAuthChange("tokens-refreshed");
            console.log("[Auth] Tokens refreshed successfully");
            return true;
        } catch (error) {
            console.error("[Auth] Token refresh error:", error);
            onAuthChange("session-expired");
            return false;
        }
    }

    async function completeLoopbackAuthorization(verifier: string, state: string): Promise<void> {
        if (!onLoopbackAuthorization || !createCodeChallenge) {
            throw new Error("loopback authorization is not configured");
        }
        const challenge = createCodeChallenge(verifier);
        const callback = await onLoopbackAuthorization((redirectURI) => {
            const authUrl = new URL(`${config.lpbsUrl}/auth/login`);
            authUrl.searchParams.set("redirect_uri", redirectURI);
            authUrl.searchParams.set("code_challenge", challenge);
            authUrl.searchParams.set("code_challenge_method", "S256");
            authUrl.searchParams.set("app", config.appDisplayName);
            authUrl.searchParams.set("state", state);
            return authUrl.toString();
        });
        if (callback.state !== state || !callback.code) {
            throw new Error("loopback authorization state/code rejected");
        }
        const response = await http.fetch(`${config.lpbsUrl}/api/v1/auth/token`, {
            method: "POST",
            headers: { "Content-Type": "application/json" },
            body: JSON.stringify({
                code: callback.code,
                code_verifier: verifier,
                redirect_uri: callback.redirectURI,
            }),
        });
        if (!response.ok) {
            throw new Error(`authorization code exchange failed (${response.status})`);
        }
        const tokens = await response.json() as {
            access_token: string;
            refresh_token: string;
            expires_at: string;
        };
        await storeAuthTokens({
            accessToken: tokens.access_token,
            refreshToken: tokens.refresh_token,
            expiresAt: tokens.expires_at,
        });
        scheduleTokenRefresh(tokens.expires_at);
        onAuthChange("tokens-received");
    }

    async function completeDesktopLink(options: DesktopLinkOptions & { state: string }): Promise<void> {
        if (!onLoopbackAuthorization || !createCodeChallenge) {
            throw new Error("desktop linking is not configured");
        }
        if (!onResolveLocalIdentityProof) {
            throw new Error("declared local identity proof is unavailable");
        }
        if (!options.installationId || !options.resource || options.audience !== `scenario:${options.resource}` || options.scopes.length === 0 || options.scopes.some((scope) => !scope.startsWith(`${options.resource}:`))) {
            throw new Error("desktop link binding is invalid");
        }

        const verifier = `${uuid.generate()}${uuid.generate()}`;
        const challenge = createCodeChallenge(verifier);
        const callback = await onLoopbackAuthorization((redirectURI) => {
            const authUrl = new URL(`${config.lpbsUrl}/auth/login`);
            authUrl.searchParams.set("redirect_uri", redirectURI);
            authUrl.searchParams.set("code_challenge", challenge);
            authUrl.searchParams.set("code_challenge_method", "S256");
            authUrl.searchParams.set("app", config.appDisplayName);
            authUrl.searchParams.set("state", options.state);
            authUrl.searchParams.set("desktop_link", "true");
            authUrl.searchParams.set("installation_id", options.installationId);
            authUrl.searchParams.set("resource", options.resource);
            authUrl.searchParams.set("audience", options.audience);
            authUrl.searchParams.set("scopes", options.scopes.join(","));
            return authUrl.toString();
        });
        if (callback.state !== options.state || !callback.code) {
            throw new Error("desktop link state/code rejected");
        }

        const localProof = await onResolveLocalIdentityProof();
        if (!localProof) {
            throw new Error("verified local identity proof is unavailable");
        }
        const response = await http.fetch(`${config.lpbsUrl}/api/v1/desktop/links/redeem`, {
            method: "POST",
            headers: { "Authorization": `Bearer ${localProof}`, "Content-Type": "application/json" },
            body: JSON.stringify({
                code: callback.code,
                code_verifier: verifier,
                installation_id: options.installationId,
                resource: options.resource,
            }),
        });
        if (!response.ok) {
            throw new Error(`desktop link redemption failed (${response.status})`);
        }
        const result = await response.json() as {
            entitlement_lease?: { token?: string; expires_at?: string };
        };
        const lease = result.entitlement_lease;
        if (!lease || typeof lease.token !== "string" || !lease.token || typeof lease.expires_at !== "string") {
            throw new Error("desktop link response did not contain a signed entitlement lease");
        }
        await storeAuthTokens({
            accessToken: "",
            refreshToken: "",
            expiresAt: lease.expires_at,
            entitlementLease: lease.token,
        });
        onAuthChange("tokens-received");
    }

    const manager: IAuthManager = {
        async signIn(options?: { state?: string }): Promise<{ state: string }> {
            const state = options?.state ?? uuid.generate();

            const verifier = `${uuid.generate()}${uuid.generate()}`;
            void completeLoopbackAuthorization(verifier, state).catch((error: unknown) => {
                console.error("[Auth] loopback sign-in failed:", error);
                onAuthChange("session-expired");
            });

            return { state };
        },

        async connectDesktop(options: DesktopLinkOptions & { state?: string }): Promise<void> {
            const state = options.state ?? uuid.generate();
            await completeDesktopLink({ ...options, state });
        },

        async signOut(): Promise<void> {
            // Get tokens BEFORE clearing (needed for logout API call)
            const tokens = await getStoredTokens();

            // Clear local auth data
            await clearAuthData();

            // Try to call logout endpoint (best effort)
            if (tokens?.accessToken) {
                try {
                    await http.fetch(`${config.lpbsUrl}/api/v1/auth/logout`, {
                        method: "POST",
                        headers: { "Authorization": `Bearer ${tokens.accessToken}` },
                    });
                } catch (error) {
                    console.warn("[Auth] Logout API call failed:", error);
                }
            }

            onAuthChange("signed-out");
            console.log("[Auth] Signed out");
        },

        async getAccessToken(): Promise<string | null> {
            const tokens = await getStoredTokens();
            if (!tokens || !tokens.accessToken) return null;

            // Check if token is expired
            const expiresAt = new Date(tokens.expiresAt).getTime();
            if (timer.now() >= expiresAt) {
                // Try to refresh
                const refreshed = await refreshTokens();
                if (!refreshed) return null;

                const newTokens = await getStoredTokens();
                return newTokens?.accessToken ?? null;
            }

            return tokens.accessToken;
        },

        async getEntitlementLease(): Promise<string | null> {
            const tokens = await getStoredTokens();
            return tokens?.entitlementLease ?? null;
        },

        async getUser(): Promise<StoredUser | null> {
            return getStoredUser();
        },

        async isAuthenticated(): Promise<boolean> {
            const tokens = await getStoredTokens();
            if (!tokens || !tokens.accessToken) return false;

            const expiresAt = new Date(tokens.expiresAt).getTime();
            // Allow some grace period for refresh
            return timer.now() < expiresAt + config.tokenRefreshBufferMs;
        },

        async refresh(): Promise<boolean> {
            return refreshTokens();
        },

        async handleCallback(url: string): Promise<void> {
            try {
                const parsed = new URL(url);

                // Check if this is an auth callback
                // For custom protocols like "vrooli://auth/callback":
                // - hostname = "auth", pathname = "/callback"
                // For http-like URLs "http://localhost/auth/callback":
                // - pathname = "/auth/callback"
                const isAuthCallback =
                    (parsed.hostname === "auth" && parsed.pathname === "/callback") ||
                    parsed.pathname === "/auth/callback";

                if (!isAuthCallback) {
                    // Not an auth callback, forward for generic handling
                    onProtocolUrl?.(url);
                    return;
                }

                // A custom scheme is never an authentication credential channel.
                // The supported flow consumes the code on the process-owned
                // loopback listener before this generic protocol handler runs.
                if (parsed.hash.includes("access_token") || parsed.hash.includes("refresh_token") || parsed.searchParams.has("access_token") || parsed.searchParams.has("refresh_token")) {
                    console.error("[Auth] rejected token-bearing custom-scheme callback");
                    onAuthChange("session-expired");
                    return;
                }
                onProtocolUrl?.(url);
            } catch (error) {
                console.error("[Auth] Failed to handle auth callback:", error);
                onAuthChange("session-expired");
            }
        },

        async initialize(): Promise<void> {
            // Check for existing tokens and schedule refresh if valid
            const tokens = await getStoredTokens();
            if (tokens?.refreshToken && tokens.expiresAt) {
                const expiresAt = new Date(tokens.expiresAt).getTime();
                if (timer.now() < expiresAt) {
                    scheduleTokenRefresh(tokens.expiresAt);
                }
            }
        },

        dispose(): void {
            if (tokenRefreshTimer) {
                timer.clearTimeout(tokenRefreshTimer);
                tokenRefreshTimer = null;
            }
        },
    };

    return manager;
}

/**
 * Create a real Electron safeStorage adapter.
 */
export function createElectronSafeStorage(
    electronSafeStorage: typeof import("electron").safeStorage
): import("./types").ISafeStorage {
    return {
        isEncryptionAvailable: () => electronSafeStorage.isEncryptionAvailable(),
        encryptString: (data) => electronSafeStorage.encryptString(data),
        decryptString: (encrypted) => electronSafeStorage.decryptString(encrypted),
    };
}

/**
 * Create a real Electron net-based HTTP client.
 */
export function createElectronAuthHttpClient(
    electronNet: typeof import("electron").net,
    defaultHeaders?: Record<string, string>,
    allowedOrigins?: Set<string>
): import("./types").IAuthHttpClient {
    return {
        fetch: (url, options) => {
            const init: RequestInit = {};
            if (options?.method) init.method = options.method;
            let validationHeaders: Record<string, string> | undefined;
            if (defaultHeaders) {
                try {
                    if (!allowedOrigins || allowedOrigins.has(new URL(url).origin)) validationHeaders = defaultHeaders;
                } catch {
                    validationHeaders = undefined;
                }
            }
            init.headers = { ...validationHeaders, ...options?.headers };
            if (options?.body) init.body = options.body;
            return electronNet.fetch(url, init) as Promise<{
                ok: boolean;
                status: number;
                json(): Promise<unknown>;
            }>;
        },
    };
}

/**
 * Create a real shell adapter.
 */
export function createElectronShell(
    electronShell: typeof import("electron").shell
): import("./types").IShell {
    return {
        openExternal: (url) => electronShell.openExternal(url),
    };
}

/**
 * Create a real timer adapter.
 */
export function createRealAuthTimer(): import("./types").IAuthTimer {
    return {
        now: () => Date.now(),
        setTimeout: (callback, delay) => setTimeout(callback, delay),
        clearTimeout: (timer) => clearTimeout(timer),
    };
}

/**
 * Create a real UUID generator.
 */
export function createRealUuidGenerator(
    randomUUID: () => string
): import("./types").IUuidGenerator {
    return {
        generate: randomUUID,
    };
}

/**
 * Create default auth config.
 */
export function createDefaultAuthConfig(overrides?: Partial<import("./types").AuthConfig>): import("./types").AuthConfig {
    return {
        protocol: "vrooli",
        lpbsUrl: "https://vrooli.com",
        tokensFile: "auth/tokens.enc",
        userFile: "auth/user.json",
        tokenRefreshBufferMs: 5 * 60 * 1000, // 5 minutes
        appDisplayName: "Vrooli App",
        ...overrides,
    };
}
