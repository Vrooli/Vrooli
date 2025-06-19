/**
 * Authentication and authorization error fixtures
 * 
 * These fixtures provide auth-related error scenarios including
 * login failures, session issues, permission errors, and account states.
 */

export interface AuthError {
    code: string;
    message: string;
    details?: {
        reason?: string;
        requiredRole?: string;
        requiredPermission?: string;
        expiresAt?: string;
        remainingAttempts?: number;
        lockoutDuration?: number;
        [key: string]: any;
    };
    action?: {
        type: "login" | "logout" | "refresh" | "verify" | "upgrade";
        label: string;
        url?: string;
    };
}

export const authErrorFixtures = {
    // Login errors
    login: {
        invalidCredentials: {
            code: "INVALID_CREDENTIALS",
            message: "Invalid email or password",
            details: {
                remainingAttempts: 3,
            },
        } satisfies AuthError,

        accountLocked: {
            code: "ACCOUNT_LOCKED",
            message: "Your account has been locked due to too many failed login attempts",
            details: {
                reason: "Too many failed login attempts",
                lockoutDuration: 3600,
                expiresAt: new Date(Date.now() + 3600000).toISOString(),
            },
            action: {
                type: "verify",
                label: "Unlock Account",
                url: "/auth/unlock",
            },
        } satisfies AuthError,

        emailNotVerified: {
            code: "EMAIL_NOT_VERIFIED",
            message: "Please verify your email address to continue",
            action: {
                type: "verify",
                label: "Resend Verification Email",
                url: "/auth/verify-email",
            },
        } satisfies AuthError,

        twoFactorRequired: {
            code: "2FA_REQUIRED",
            message: "Two-factor authentication required",
            details: {
                methods: ["totp", "sms"],
                sessionToken: "temp_session_123",
            },
        } satisfies AuthError,
    },

    // Session errors
    session: {
        expired: {
            code: "SESSION_EXPIRED",
            message: "Your session has expired. Please log in again",
            action: {
                type: "login",
                label: "Log In",
                url: "/auth/login",
            },
        } satisfies AuthError,

        invalidToken: {
            code: "INVALID_TOKEN",
            message: "Invalid or malformed authentication token",
            details: {
                reason: "Token signature verification failed",
            },
        } satisfies AuthError,

        refreshFailed: {
            code: "REFRESH_FAILED",
            message: "Unable to refresh your session",
            details: {
                reason: "Refresh token expired",
            },
            action: {
                type: "login",
                label: "Log In Again",
                url: "/auth/login",
            },
        } satisfies AuthError,

        concurrentSession: {
            code: "CONCURRENT_SESSION",
            message: "You have been logged out because you signed in from another device",
            details: {
                device: "Chrome on Windows",
                location: "New York, US",
                timestamp: new Date().toISOString(),
            },
        } satisfies AuthError,
    },

    // Permission errors
    permissions: {
        insufficientRole: {
            code: "INSUFFICIENT_ROLE",
            message: "You don't have the required role to perform this action",
            details: {
                requiredRole: "admin",
                currentRole: "member",
            },
        } satisfies AuthError,

        missingPermission: {
            code: "MISSING_PERMISSION",
            message: "You don't have permission to access this resource",
            details: {
                requiredPermission: "project:write",
                resource: "project_123",
            },
        } satisfies AuthError,

        teamPermission: {
            code: "TEAM_PERMISSION_DENIED",
            message: "Your team role doesn't allow this action",
            details: {
                team: "Development Team",
                requiredRole: "owner",
                currentRole: "viewer",
            },
        } satisfies AuthError,

        apiKeyScope: {
            code: "API_KEY_SCOPE_INSUFFICIENT",
            message: "API key doesn't have the required scope",
            details: {
                requiredScope: "write:projects",
                availableScopes: ["read:projects", "read:users"],
            },
        } satisfies AuthError,
    },

    // Account state errors
    accountState: {
        suspended: {
            code: "ACCOUNT_SUSPENDED",
            message: "Your account has been suspended",
            details: {
                reason: "Terms of service violation",
                suspendedAt: new Date(Date.now() - 86400000).toISOString(),
            },
            action: {
                type: "verify",
                label: "Appeal Suspension",
                url: "/support/appeal",
            },
        } satisfies AuthError,

        banned: {
            code: "ACCOUNT_BANNED",
            message: "Your account has been permanently banned",
            details: {
                reason: "Repeated violations",
                bannedAt: new Date(Date.now() - 604800000).toISOString(),
            },
        } satisfies AuthError,

        deleted: {
            code: "ACCOUNT_DELETED",
            message: "This account has been deleted",
            details: {
                deletedAt: new Date(Date.now() - 2592000000).toISOString(),
                recoverable: true,
                recoverDeadline: new Date(Date.now() + 604800000).toISOString(),
            },
            action: {
                type: "verify",
                label: "Recover Account",
                url: "/auth/recover",
            },
        } satisfies AuthError,

        inactive: {
            code: "ACCOUNT_INACTIVE",
            message: "Your account is inactive. Please contact support",
            details: {
                lastActiveAt: new Date(Date.now() - 7776000000).toISOString(),
            },
        } satisfies AuthError,
    },

    // OAuth errors
    oauth: {
        providerError: {
            code: "OAUTH_PROVIDER_ERROR",
            message: "Authentication failed with the external provider",
            details: {
                provider: "google",
                providerError: "access_denied",
            },
        } satisfies AuthError,

        accountLinking: {
            code: "OAUTH_ACCOUNT_EXISTS",
            message: "An account with this email already exists",
            details: {
                email: "user@example.com",
                existingProvider: "email",
                attemptedProvider: "google",
            },
            action: {
                type: "login",
                label: "Log In with Email",
                url: "/auth/login",
            },
        } satisfies AuthError,

        scopeDenied: {
            code: "OAUTH_SCOPE_DENIED",
            message: "Required permissions were not granted",
            details: {
                provider: "github",
                requiredScopes: ["user:email", "read:org"],
                grantedScopes: ["user:email"],
            },
        } satisfies AuthError,
    },

    // API key errors
    apiKey: {
        invalid: {
            code: "INVALID_API_KEY",
            message: "The provided API key is invalid",
        } satisfies AuthError,

        expired: {
            code: "API_KEY_EXPIRED",
            message: "This API key has expired",
            details: {
                expiredAt: new Date(Date.now() - 86400000).toISOString(),
            },
        } satisfies AuthError,

        revoked: {
            code: "API_KEY_REVOKED",
            message: "This API key has been revoked",
            details: {
                revokedAt: new Date(Date.now() - 3600000).toISOString(),
                reason: "Security breach",
            },
        } satisfies AuthError,

        rateLimited: {
            code: "API_KEY_RATE_LIMITED",
            message: "API key rate limit exceeded",
            details: {
                limit: 1000,
                window: "hour",
                reset: new Date(Date.now() + 1800000).toISOString(),
            },
        } satisfies AuthError,
    },

    // Factory functions
    factories: {
        /**
         * Create a custom auth error
         */
        createAuthError: (code: string, message: string, details?: any): AuthError => ({
            code,
            message,
            ...(details && { details }),
        }),

        /**
         * Create a permission error
         */
        createPermissionError: (
            resource: string,
            requiredPermission: string,
            currentPermission?: string,
        ): AuthError => ({
            code: "PERMISSION_DENIED",
            message: `You don't have permission to access ${resource}`,
            details: {
                resource,
                requiredPermission,
                ...(currentPermission && { currentPermission }),
            },
        }),

        /**
         * Create a role-based error
         */
        createRoleError: (requiredRole: string, currentRole: string): AuthError => ({
            code: "INSUFFICIENT_ROLE",
            message: `This action requires ${requiredRole} role`,
            details: {
                requiredRole,
                currentRole,
            },
        }),

        /**
         * Create a session error with action
         */
        createSessionError: (reason: string, actionUrl?: string): AuthError => ({
            code: "SESSION_ERROR",
            message: "There was a problem with your session",
            details: { reason },
            ...(actionUrl && {
                action: {
                    type: "login",
                    label: "Log In Again",
                    url: actionUrl,
                },
            }),
        }),
    },
};