/**
 * Authentication and authorization error fixtures
 * 
 * These fixtures provide auth-related error scenarios including
 * login failures, session issues, permission errors, and account states.
 * 
 * Enhanced with VrooliError interface compatibility for cross-package validation.
 */
import { BaseErrorFactory, BaseErrorFixture, type EnhancedAuthError, type ErrorContext } from "./types.js";
import { type TranslationKeyError } from "../../types.js";

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


// Create and export the auth error factory instance
export const authErrorFactory = new AuthErrorFactory();

/**
 * VrooliError-compatible auth error fixtures for cross-package validation.
 * These implement the ParseableError interface and can be validated with errorTestUtils.
 */
export const authErrorFixtures = {
    invalidCredentials: new BaseErrorFixture(
        "InvalidCredentials" as TranslationKeyError,
        "0062-TEST",
        { reason: "Invalid username or password" },
    ),
    
    sessionExpired: new BaseErrorFixture(
        "SessionExpired" as TranslationKeyError,
        "0063-TEST",
        { action: "redirect_to_login" },
    ),
    
    accessDenied: new BaseErrorFixture(
        "AccessDenied" as TranslationKeyError,
        "0064-TEST",
        { requiredPermission: "admin" },
    ),
    
    accountLocked: new BaseErrorFixture(
        "AccountLocked" as TranslationKeyError,
        "0065-TEST",
        { lockoutDuration: 3600, reason: "Too many failed attempts" },
    ),
    
    twoFactorRequired: new BaseErrorFixture(
        "TwoFactorRequired" as TranslationKeyError,
        "0066-TEST",
        { methods: ["totp", "sms"] },
    ),
} as const;

/**
 * Type-safe access to auth error fixture keys
 */
export type AuthErrorFixtureKey = keyof typeof authErrorFixtures;
