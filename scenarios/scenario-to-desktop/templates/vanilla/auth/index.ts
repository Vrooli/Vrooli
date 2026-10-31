/**
 * Auth Module
 *
 * DOC: docs/internal/SEAMS.md#auth-module
 *
 * Barrel exports for the authentication system.
 */

// Types
export type {
    StoredTokens,
    StoredUser,
    AuthChangeEvent,
    AuthConfig,
    DesktopLinkOptions,
    DesktopLinkQuery,
    DesktopLinkState,
    DesktopLinkStatus,
    DesktopDisconnectResult,
    ISafeStorage,
    IAuthHttpClient,
    IShell,
    IAuthTimer,
    IUuidGenerator,
    AuthChangeCallback,
    WindowFocusCallback,
    ProtocolUrlCallback,
    IAuthStorage,
    IAuthPathUtils,
    IAuthManager,
    AuthManagerDependencies,
} from "./types";

// Implementation
export {
    createAuthManager,
    parseConfiguredCredentialStatus,
    createElectronSafeStorage,
    createElectronAuthHttpClient,
    createElectronShell,
    createRealAuthTimer,
    createRealUuidGenerator,
    createDefaultAuthConfig,
} from "./manager";
