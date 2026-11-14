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
    /** Short-lived access token; process-memory only. */
    accessToken: string;
    /** Website refresh token; process-memory only. */
    refreshToken: string;
    /** Compatibility session expiry; not a lease authorization boundary. */
    expiresAt: string;
    /** Signed LPBS entitlement lease, when the account has one. */
    entitlementLease?: string;
}

export interface CreditSessionShareStatus {
    state: "shared" | "not_shared" | "signed_out" | "unavailable";
    /** Clearing the local credential cannot revoke already-issued short-lived access tokens. */
    remoteRevocation?: "uncertain";
    localCleared?: boolean;
}

/** A fresh, authenticated LPBS wallet read. Values are display facts only. */
export type CreditBalanceStatus =
    | { state: "available"; balanceCredits: string; displayCreditsLabel: string; displayCreditsMultiplier: number; walletUpdatedAt?: string; fetchedAt: string; source: "LPBS" }
    | { state: "signed_out" | "unavailable" };

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

/** A single, scoped LPBS-to-desktop account-link request. */
export interface DesktopLinkOptions {
    /** Omit to use the stable installation identity owned by the desktop shell. */
    installationId?: string;
    resource: string;
    audience: string;
    scopes: string[];
}

export interface DesktopLinkQuery {
    resource: string;
}

export type DesktopLinkState = "linked" | "unlinked" | "expired" | "revoked" | "unavailable";

/** Sanitized status from the LPBS local-link owner; never contains lease bytes. */
export interface DesktopLinkStatus {
    state: DesktopLinkState;
    localPrincipal?: string;
    businessAccountId?: string;
    scopes: string[];
    message?: string;
}

export interface DesktopDisconnectResult {
    localCleared: boolean;
    remoteRevocation: "confirmed" | "uncertain";
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

/** Result delivered by a process-owned loopback authorization listener. */
export interface LoopbackAuthorizationResult {
    code: string;
    state: string;
    redirectURI: string;
}

/** Binds a browser authorization request to an ephemeral loopback listener. */
export type LoopbackAuthorizationCallback = (
    buildAuthorizationURL: (redirectURI: string) => string,
) => Promise<LoopbackAuthorizationResult>;

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
     * Connect the current desktop installation to an LPBS business account.
     * The local proof must come from the declared local identity provider.
     */
    connectDesktop(options: DesktopLinkOptions & { state?: string }): Promise<DesktopLinkStatus>;

    /** Query LPBS's proof-authenticated local link record for this resource. */
    getDesktopLinkStatus(options: DesktopLinkQuery): Promise<DesktopLinkStatus>;

    /** Revoke remotely when reachable, then always clear local auth material. */
    disconnectDesktop(options: DesktopLinkQuery): Promise<DesktopDisconnectResult>;

    /** Share or clear the current LPBS refresh credential through the credential owner. */
    setCreditSessionSharing(enabled: boolean): Promise<CreditSessionShareStatus>;
    /** Read credential-presence status only; never return credential bytes. */
    getCreditSessionShareStatus(): Promise<CreditSessionShareStatus>;
    /** Read the current user's wallet through LPBS without exposing credentials. */
    getCreditBalance(): Promise<CreditBalanceStatus>;

    /**
     * Sign out the user and clear all auth data.
     */
    signOut(): Promise<void>;

    /**
     * Get the current access token, refreshing if needed.
     * @returns The access token, or null if not authenticated
     */
    getAccessToken(): Promise<string | null>;

    /** Return the last signed entitlement lease received from LPBS. */
    getEntitlementLease(): Promise<string | null>;

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
    /** Runs the browser callback on a process-owned loopback listener. */
    onLoopbackAuthorization?: LoopbackAuthorizationCallback;
    /** Derives the RFC 7636 S256 challenge from a verifier. */
    createCodeChallenge?: (verifier: string) => string;
    /** Store the signed entitlement lease in the platform credential authority. */
    onStoreEntitlementLease?: (lease: string) => Promise<void>;
    /** Resolve the signed entitlement lease after an app restart. */
    onGetEntitlementLease?: () => Promise<string | null>;
    /** Remove the signed entitlement lease during unlink/sign-out. */
    onClearEntitlementLease?: () => Promise<void>;
    /** Explicitly provision the LPBS refresh credential into its declared shared owner identity. */
    onStoreLPBSCreditRefreshToken?: (refreshToken: string) => Promise<void>;
    /** Presence-only query against the declared shared credential owner. */
    onHasLPBSCreditRefreshToken?: () => Promise<boolean>;
    /** Delete the LPBS refresh credential from the shared owner. */
    onClearLPBSCreditRefreshToken?: () => Promise<void>;
    /** Resolve a short-lived proof from the declared local identity provider. */
    onResolveLocalIdentityProof?: () => Promise<string | null>;
    /** Resolve or create the installation identity in desktop-owned private config. */
    onGetDesktopInstallationId?: () => Promise<string>;
}
