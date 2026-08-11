/**
 * Auth Module Types
 *
 * DOC: docs/internal/SEAMS.md#auth-types
 *
 * Type definitions for the authentication system.
 * Provides secure token storage and magic link authentication.
 */

// ===== Auth Data Types =====

/**
 * Stored authentication tokens.
 */
export interface StoredTokens {
    accessToken: string;
    refreshToken: string;
    expiresAt: string;
}

/**
 * Stored user information.
 */
export interface StoredUser {
    id: string;
    email: string;
    emailVerified: boolean;
}

/**
 * Auth state change event types.
 */
export type AuthChangeEvent =
    | "tokens-received"
    | "tokens-refreshed"
    | "session-expired"
    | "signed-out";

/**
 * Auth configuration.
 */
export interface AuthConfig {
    /** Custom URL protocol (e.g., "vrooli") */
    protocol: string;
    /** LPBS (Landing Page Business Suite) URL for auth */
    lpbsUrl: string;
    /** Relative path for encrypted token storage */
    tokensFile: string;
    /** Relative path for user info cache */
    userFile: string;
    /** Time before expiry to refresh tokens (ms) */
    tokenRefreshBufferMs: number;
    /** App display name for auth page */
    appDisplayName: string;
}

// ===== Seam Interfaces =====

/**
 * Safe storage operations for encryption.
 * This seam allows injecting mock encryption for testing.
 */
export interface ISafeStorage {
    isEncryptionAvailable(): boolean;
    encryptString(data: string): Buffer;
    decryptString(encrypted: Buffer): string;
}

/**
 * HTTP client for auth API calls.
 * This seam allows injecting mock HTTP for testing.
 */
export interface IAuthHttpClient {
    fetch(url: string, options?: {
        method?: string;
        headers?: Record<string, string>;
        body?: string;
    }): Promise<{
        ok: boolean;
        status: number;
        json(): Promise<unknown>;
    }>;
}

/**
 * Shell operations for opening external URLs.
 */
export interface IShell {
    openExternal(url: string): Promise<void>;
}

/**
 * Timer operations for scheduling.
 */
export interface IAuthTimer {
    now(): number;
    setTimeout(callback: () => void, delayMs: number): NodeJS.Timeout;
    clearTimeout(timer: NodeJS.Timeout): void;
}

/**
 * UUID generator.
 */
export interface IUuidGenerator {
    generate(): string;
}

/**
 * Callback for auth change notifications.
 */
export type AuthChangeCallback = (event: AuthChangeEvent) => void;

/**
 * Callback for window focusing.
 */
export type WindowFocusCallback = () => void;

/**
 * Callback for protocol URL handling.
 */
export type ProtocolUrlCallback = (url: string) => void;

/**
 * Storage operations needed by auth module.
 * Simplified interface compared to full IAppStorage.
 */
export interface IAuthStorage {
    resolvePath(relativePath: string): Promise<string | null>;
    readFile(relativePath: string): Promise<Buffer | null>;
    readTextFile(relativePath: string): Promise<string | null>;
    writeFile(relativePath: string, data: string | Buffer): Promise<void>;
    deleteFile(relativePath: string): Promise<boolean>;
    ensureDir(relativePath: string): Promise<void>;
}

/**
 * Path utilities for auth module.
 */
export interface IAuthPathUtils {
    dirname(path: string): string;
}

// ===== Auth Manager Interface =====

/**
 * Authentication manager interface.
 * Handles token storage, refresh scheduling, and auth flow.
 */
export interface IAuthManager {
    /**
     * Start the sign-in flow by opening the browser to the auth page.
     * @param options - Optional sign-in options
     * @returns The state parameter used for CSRF validation
     */
    signIn(options?: { state?: string }): Promise<{ state: string }>;

    /**
     * Sign out the user and clear all auth data.
     */
    signOut(): Promise<void>;

    /**
     * Get the current access token, refreshing if needed.
     * @returns The access token, or null if not authenticated
     */
    getAccessToken(): Promise<string | null>;

    /**
     * Get the stored user information.
     * @returns User info, or null if not available
     */
    getUser(): Promise<StoredUser | null>;

    /**
     * Check if the user is currently authenticated.
     * @returns true if authenticated with valid tokens
     */
    isAuthenticated(): Promise<boolean>;

    /**
     * Force refresh the tokens.
     * @returns true if refresh was successful
     */
    refresh(): Promise<boolean>;

    /**
     * Handle an auth callback URL (from protocol handler).
     * @param url - The callback URL with tokens
     */
    handleCallback(url: string): Promise<void>;

    /**
     * Initialize auth state (schedule refresh for existing tokens).
     */
    initialize(): Promise<void>;

    /**
     * Cleanup resources (clear timers).
     */
    dispose(): void;
}

/**
 * Auth manager dependencies bundle.
 */
export interface AuthManagerDependencies {
    storage: IAuthStorage;
    safeStorage: ISafeStorage;
    http: IAuthHttpClient;
    shell: IShell;
    timer: IAuthTimer;
    uuid: IUuidGenerator;
    pathUtils: IAuthPathUtils;
    config: AuthConfig;
    onAuthChange: AuthChangeCallback;
    onWindowFocus?: WindowFocusCallback;
    onProtocolUrl?: ProtocolUrlCallback;
    /** Store the rotating LPBS refresh token in the platform credential authority. */
    onRefreshToken?: (refreshToken: string) => Promise<void>;
    /** Resolve the shared LPBS refresh token after an app restart. */
    onGetRefreshToken?: () => Promise<string | null>;
    /** Remove the shared LPBS refresh token during sign-out. */
    onClearRefreshToken?: () => Promise<void>;
}
