/**
 * Authentication and authorization error fixtures
 * 
 * These fixtures provide auth-related error scenarios including
 * login failures, session issues, permission errors, and account states.
 */
import { BaseErrorFactory, type EnhancedAuthError, type ErrorContext } from "./types.js";

/**
 * Factory for creating enhanced authentication errors
 */
export class AuthErrorFactory extends BaseErrorFactory<EnhancedAuthError> {
    standard: EnhancedAuthError = {
        code: "AUTH_ERROR",
        message: "Authentication error occurred",
        userImpact: "blocking",
        recovery: { strategy: "fail" },
    };

    withDetails: EnhancedAuthError = {
        code: "AUTH_ERROR",
        message: "Authentication error occurred",
        details: {
            reason: "Generic authentication failure",
        },
        userImpact: "blocking",
        recovery: { strategy: "fail" },
    };

    variants = {
        sessionExpired: {
            code: "SESSION_EXPIRED",
            message: "Your session has expired. Please log in again",
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "redirect_to_login",
            },
            action: {
                type: "login",
                label: "Log In",
                url: "/auth/login",
            },
        } satisfies EnhancedAuthError,

        unauthorized: {
            code: "UNAUTHORIZED",
            message: "You must be logged in to perform this action",
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "redirect_to_login",
            },
        } satisfies EnhancedAuthError,
    };

    create(overrides?: Partial<EnhancedAuthError>): EnhancedAuthError {
        return {
            ...this.standard,
            ...overrides,
        };
    }

    createWithContext(context: ErrorContext): EnhancedAuthError {
        return {
            ...this.standard,
            context,
            details: {
                operation: context.operation,
                userId: context.user?.id,
                userRole: context.user?.role,
            },
        };
    }
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
            userImpact: "blocking",
            recovery: {
                strategy: "retry",
                attempts: 3,
            },
        } satisfies EnhancedAuthError,

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
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,

        emailNotVerified: {
            code: "EMAIL_NOT_VERIFIED",
            message: "Please verify your email address to continue",
            action: {
                type: "verify",
                label: "Resend Verification Email",
                url: "/auth/verify-email",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "verify_email",
            },
        } satisfies EnhancedAuthError,

        twoFactorRequired: {
            code: "2FA_REQUIRED",
            message: "Two-factor authentication required",
            details: {
                methods: ["totp", "sms"],
                sessionToken: "temp_session_123",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "complete_2fa",
            },
        } satisfies EnhancedAuthError,
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
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "redirect_to_login",
            },
        } satisfies EnhancedAuthError,

        invalidToken: {
            code: "INVALID_TOKEN",
            message: "Invalid or malformed authentication token",
            details: {
                reason: "Token signature verification failed",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,

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
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "redirect_to_login",
            },
        } satisfies EnhancedAuthError,

        concurrentSession: {
            code: "CONCURRENT_SESSION",
            message: "You have been logged out because you signed in from another device",
            details: {
                device: "Chrome on Windows",
                location: "New York, US",
                timestamp: new Date().toISOString(),
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "redirect_to_login",
            },
        } satisfies EnhancedAuthError,
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
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,

        missingPermission: {
            code: "MISSING_PERMISSION",
            message: "You don't have permission to access this resource",
            details: {
                requiredPermission: "project:write",
                resource: "project_123",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,

        teamPermission: {
            code: "TEAM_PERMISSION_DENIED",
            message: "Your team role doesn't allow this action",
            details: {
                team: "Development Team",
                requiredRole: "owner",
                currentRole: "viewer",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,

        apiKeyScope: {
            code: "API_KEY_SCOPE_INSUFFICIENT",
            message: "API key doesn't have the required scope",
            details: {
                requiredScope: "write:projects",
                availableScopes: ["read:projects", "read:users"],
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,
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
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "contact_support",
            },
        } satisfies EnhancedAuthError,

        banned: {
            code: "ACCOUNT_BANNED",
            message: "Your account has been permanently banned",
            details: {
                reason: "Repeated violations",
                bannedAt: new Date(Date.now() - 604800000).toISOString(),
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,

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
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "recover_account",
            },
        } satisfies EnhancedAuthError,

        inactive: {
            code: "ACCOUNT_INACTIVE",
            message: "Your account is inactive. Please contact support",
            details: {
                lastActiveAt: new Date(Date.now() - 7776000000).toISOString(),
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "contact_support",
            },
        } satisfies EnhancedAuthError,
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
            userImpact: "blocking",
            recovery: {
                strategy: "retry",
                attempts: 3,
            },
        } satisfies EnhancedAuthError,

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
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "use_existing_provider",
            },
        } satisfies EnhancedAuthError,

        scopeDenied: {
            code: "OAUTH_SCOPE_DENIED",
            message: "Required permissions were not granted",
            details: {
                provider: "github",
                requiredScopes: ["user:email", "read:org"],
                grantedScopes: ["user:email"],
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "retry_oauth_with_scopes",
            },
        } satisfies EnhancedAuthError,
    },

    // API key errors
    apiKey: {
        invalid: {
            code: "INVALID_API_KEY",
            message: "The provided API key is invalid",
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,

        expired: {
            code: "API_KEY_EXPIRED",
            message: "This API key has expired",
            details: {
                expiresAt: new Date(Date.now() - 86400000).toISOString(),
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: "regenerate_api_key",
            },
        } satisfies EnhancedAuthError,

        revoked: {
            code: "API_KEY_REVOKED",
            message: "This API key has been revoked",
            details: {
                revokedAt: new Date(Date.now() - 3600000).toISOString(),
                reason: "Security breach",
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        } satisfies EnhancedAuthError,

        rateLimited: {
            code: "API_KEY_RATE_LIMITED",
            message: "API key rate limit exceeded",
            details: {
                limit: 1000,
                window: "hour",
                reset: new Date(Date.now() + 1800000).toISOString(),
            },
            userImpact: "degraded",
            recovery: {
                strategy: "retry",
                attempts: 3,
                delay: 60000,
                backoffMultiplier: 2,
            },
        } satisfies EnhancedAuthError,
    },

    // Factory functions
    factories: {
        /**
         * Create a custom auth error
         */
        createAuthError: (code: string, message: string, details?: any): EnhancedAuthError => ({
            code,
            message,
            ...(details && { details }),
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        }),

        /**
         * Create a permission error
         */
        createPermissionError: (
            resource: string,
            requiredPermission: string,
            currentPermission?: string,
        ): EnhancedAuthError => ({
            code: "PERMISSION_DENIED",
            message: `You don't have permission to access ${resource}`,
            details: {
                resource,
                requiredPermission,
                ...(currentPermission && { currentPermission }),
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        }),

        /**
         * Create a role-based error
         */
        createRoleError: (requiredRole: string, currentRole: string): EnhancedAuthError => ({
            code: "INSUFFICIENT_ROLE",
            message: `This action requires ${requiredRole} role`,
            details: {
                requiredRole,
                currentRole,
            },
            userImpact: "blocking",
            recovery: {
                strategy: "fail",
            },
        }),

        /**
         * Create a session error with action
         */
        createSessionError: (reason: string, actionUrl?: string): EnhancedAuthError => ({
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
            userImpact: "blocking",
            recovery: {
                strategy: "fallback",
                fallbackAction: actionUrl ? "redirect_to_login" : "show_error",
            },
        }),
    },
};

// Create and export the auth error factory instance
export const authErrorFactory = new AuthErrorFactory();
