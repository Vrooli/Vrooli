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
} from "./types";

/**
 * Create an auth manager with injected dependencies.
 */
export function createAuthManager(deps: AuthManagerDependencies): IAuthManager {
    const {
        storage,
        safeStorage,
        http,
        shell,
        timer,
        uuid,
        pathUtils,
        config,
        onAuthChange,
        onWindowFocus,
        onProtocolUrl,
        onRefreshToken,
        onGetRefreshToken,
        onClearRefreshToken,
    } = deps;

    let tokenRefreshTimer: NodeJS.Timeout | null = null;
    let pendingAuthState: string | null = null;
    let memoryTokens: StoredTokens | null = null;

    // The credential authority is the recovery source when encrypted local
    // storage is unavailable, has been removed, or can no longer be opened
    // after a platform keychain reset. It intentionally returns an expired
    // access-token placeholder so callers must perform a normal refresh.
    async function getAuthorityTokens(): Promise<StoredTokens | null> {
        const refreshToken = await onGetRefreshToken?.();
        if (!refreshToken) return null;
        return {
            accessToken: "",
            refreshToken,
            expiresAt: new Date(0).toISOString(),
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

        // Encrypt the tokens using Electron's safeStorage
        const tokenJson = JSON.stringify(tokens);
        if (safeStorage.isEncryptionAvailable()) {
            const encrypted = safeStorage.encryptString(tokenJson);
            await storage.writeFile(config.tokensFile, encrypted);
        } else {
            // The platform credential authority is the secure fallback. Never
            // place an access or refresh token in an unencrypted file.
            if (!onRefreshToken) {
                throw new Error("secure token storage is unavailable on this platform");
            }
            await onRefreshToken(tokens.refreshToken);
            try {
                await storage.deleteFile(config.tokensFile);
            } catch (error: unknown) {
                const code = error instanceof Error && "code" in error ? error.code : undefined;
                if (code !== "ENOENT") throw error;
            }
        }

        // Keep the durable shared credential current on every rotation. The
        // access token remains process memory only for the scenario runtime.
        if (onRefreshToken && safeStorage.isEncryptionAvailable()) {
            await onRefreshToken(tokens.refreshToken);
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
                // Plaintext token files are not accepted. Recover the refresh
                // token from the platform authority when available.
                if (fileContent) {
                    try {
                        await storage.deleteFile(config.tokensFile);
                    } catch {
                        // The authority remains the source of truth even if
                        // stale local cleanup cannot be completed.
                    }
                }
                return getAuthorityTokens();
            }

            if (!fileContent) return getAuthorityTokens();

            const decrypted = safeStorage.decryptString(fileContent);
            return JSON.parse(decrypted) as StoredTokens;
        } catch (error: unknown) {
            if (error instanceof Error && "code" in error && error.code === "ENOENT") {
                return getAuthorityTokens();
            }
            // Do not log the parse/decryption error: runtimes may include a
            // prefix of the unreadable file contents in the exception text.
            // The shared credential authority is the only recovery source.
            console.error("[Auth] Failed to read encrypted tokens; recovering from the shared authority");
            // A keychain/profile reset can make an otherwise valid encrypted
            // file unreadable. Recover only the refresh token from the
            // platform authority; never fall back to plaintext parsing.
            return getAuthorityTokens();
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

        if (onClearRefreshToken) {
            try {
                await onClearRefreshToken();
            } catch (error) {
                console.warn("[Auth] Failed to clear shared subscription session:", error);
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
            await refreshEntitlementLease(newTokens.access_token);
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

    async function refreshEntitlementLease(accessToken: string): Promise<void> {
        try {
            const response = await http.fetch(`${config.lpbsUrl}/api/v1/entitlements`, {
                headers: { Authorization: `Bearer ${accessToken}` },
            });
            if (!response.ok) return;
            const payload = await response.json() as { lease?: string };
            if (!payload.lease) return;
            const tokens = await getStoredTokens();
            if (!tokens) return;
            await storeAuthTokens({ ...tokens, accessToken, entitlementLease: payload.lease });
        } catch {
            // A still-valid cached lease remains usable while LPBS is offline.
        }
    }

    const manager: IAuthManager = {
        async signIn(options?: { state?: string }): Promise<{ state: string }> {
            const state = options?.state ?? uuid.generate();

            // Store state for CSRF validation in callback
            pendingAuthState = state;

            const authUrl = new URL(`${config.lpbsUrl}/auth/login`);
            authUrl.searchParams.set("redirect_uri", `${config.protocol}://auth/callback`);
            authUrl.searchParams.set("app", config.appDisplayName);
            authUrl.searchParams.set("state", state);

            // Open in default browser
            await shell.openExternal(authUrl.toString());

            return { state };
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
            if (!tokens) return null;

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
            if (!tokens) return false;

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

                // Extract tokens from URL fragment
                const fragmentParams = new URLSearchParams(parsed.hash.slice(1));
                const accessToken = fragmentParams.get("access_token");
                const refreshToken = fragmentParams.get("refresh_token");
                const expiresAt = fragmentParams.get("expires_at");
                const state = fragmentParams.get("state");

                // Validate CSRF state parameter
                if (pendingAuthState && state !== pendingAuthState) {
                    console.error("[Auth] CSRF validation failed: state mismatch");
                    onAuthChange("session-expired");
                    pendingAuthState = null;
                    return;
                }
                pendingAuthState = null; // Clear after validation

                if (!accessToken || !refreshToken || !expiresAt) {
                    console.error("[Auth] Missing tokens in callback URL");
                    onAuthChange("session-expired");
                    return;
                }

                // Store tokens securely
                await storeAuthTokens({
                    accessToken,
                    refreshToken,
                    expiresAt,
                });
                await refreshEntitlementLease(accessToken);
                // Schedule token refresh
                scheduleTokenRefresh(expiresAt);

                // Fetch user info
                try {
                    const userResponse = await http.fetch(`${config.lpbsUrl}/api/v1/auth/me`, {
                        headers: { "Authorization": `Bearer ${accessToken}` },
                    });

                    if (userResponse.ok) {
                        const userData = await userResponse.json() as { user: StoredUser };
                        await storeUserInfo(userData.user);
                    }
                } catch (error) {
                    console.warn("[Auth] Failed to fetch user info:", error);
                }

                // Notify renderer
                onAuthChange("tokens-received");
                console.log("[Auth] Authentication successful");

                // Focus the main window
                onWindowFocus?.();
            } catch (error) {
                console.error("[Auth] Failed to handle auth callback:", error);
                onAuthChange("session-expired");
            }
        },

        async initialize(): Promise<void> {
            // Check for existing tokens and schedule refresh if valid
            const tokens = await getStoredTokens();
            if (tokens?.expiresAt) {
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
