/* c8 ignore start */
/**
 * Auth API Response Fixtures
 * 
 * Comprehensive fixtures for authentication endpoints including login,
 * signup, password reset, session management, and OAuth flows.
 */

import type {
    EmailSignUpInput,
    Session,
    SessionUser,
} from "../../../api/types.js";
import { generatePK, nanoid } from "../../../id/index.js";
import {
    DEFAULT_DAILY_CREDIT_LIMIT,
    DEFAULT_ERROR_RATE,
    DEFAULT_EXISTING_USER_CREDITS,
    DEFAULT_FREE_CREDITS,
    DEFAULT_NEW_USER_CREDITS,
    EIGHTY_PERCENT,
    FIFTEEN_MINUTES_MS,
    LOCALHOST_IP,
    LOGIN_DEVICE_ID,
    LOGIN_DEVICE_NAME,
    MIN_PASSWORD_LENGTH,
    NEW_DEVICE_ID,
    NEW_DEVICE_NAME,
    ONE_THOUSAND,
    PRIVATE_IP_BASE,
    PRIVATE_IP_FIRST,
    SHORT_ID_LENGTH,
    TEN_MINUTES_MS,
    TEST_DEVICE_ID,
    TEST_DEVICE_NAME,
    THIRTY_DAYS_MS,
    THIRTY_MINUTES_MS,
} from "../constants.js";
import { BaseAPIResponseFactory } from "./base.js";
import type { MockDataOptions } from "./types.js";

// Auth-specific constants
const PREMIUM_CHANCE_THRESHOLD = EIGHTY_PERCENT;
const DEFAULT_AUTH_DELAY_MS = ONE_THOUSAND;
const DEFAULT_RATE_LIMIT = 5;

/**
 * Auth API response factory
 * 
 * Handles all authentication-related responses including sessions,
 * login/logout, password management, and multi-account switching.
 */
export class AuthResponseFactory extends BaseAPIResponseFactory<
    Session,
    EmailSignUpInput,
    never // Auth doesn't have a standard update operation
> {
    protected readonly entityName = "auth";

    /**
     * Create mock session data
     */
    createMockData(options?: MockDataOptions): Session {
        const scenario = options?.scenario || "minimal";
        const now = new Date().toISOString();

        if (scenario === "minimal" || !options?.overrides?.isLoggedIn) {
            // Guest/logged out session
            return {
                __typename: "Session",
                isLoggedIn: false,
                timeZone: "UTC",
                users: [],
                ...options?.overrides,
            };
        }

        // Logged in session
        const userId = options?.overrides?.users?.[0]?.id || generatePK().toString();
        const sessionUser: SessionUser = {
            __typename: "SessionUser",
            id: userId,
            credits: String(DEFAULT_NEW_USER_CREDITS),
            creditAccountId: `credit_${userId}`,
            creditSettings: {
                defaultCreditAmount: String(DEFAULT_NEW_USER_CREDITS),
                freeCreditAmount: String(DEFAULT_FREE_CREDITS),
                dailyCreditLimit: String(DEFAULT_DAILY_CREDIT_LIMIT),
            },
            handle: `user_${userId.slice(0, SHORT_ID_LENGTH)}`,
            hasPremium: false,
            hasReceivedPhoneVerificationReward: false,
            languages: ["en"],
            name: "Test User",
            profileImage: null,
            theme: "light",
            sessions: [{
                __typename: "SessionUserSession",
                id: `session_${generatePK().toString()}`,
                createdAt: now,
                updatedAt: now,
                lastActivityAt: now,
                expiresAt: new Date(Date.now() + THIRTY_DAYS_MS).toISOString(),
                deviceId: TEST_DEVICE_ID,
                deviceName: TEST_DEVICE_NAME,
                ip: LOCALHOST_IP,
                isActual: true,
            }],
        };

        return {
            __typename: "Session",
            isLoggedIn: true,
            timeZone: "America/New_York",
            users: [sessionUser],
            ...options?.overrides,
        } as Session;
    }

    /**
     * Create session from signup input
     */
    createFromInput(input: EmailSignUpInput): Session {
        const userId = generatePK().toString();
        const now = new Date().toISOString();

        return {
            __typename: "Session",
            isLoggedIn: true,
            timeZone: input.timeZone || "UTC",
            users: [{
                __typename: "SessionUser",
                id: userId,
                credits: String(DEFAULT_NEW_USER_CREDITS), // New user credits
                creditAccountId: `credit_${userId}`,
                creditSettings: {
                    defaultCreditAmount: String(DEFAULT_NEW_USER_CREDITS),
                    freeCreditAmount: String(DEFAULT_FREE_CREDITS),
                    dailyCreditLimit: String(DEFAULT_DAILY_CREDIT_LIMIT),
                },
                handle: input.handle || `user_${userId.slice(0, SHORT_ID_LENGTH)}`,
                hasPremium: false,
                hasReceivedPhoneVerificationReward: false,
                languages: input.languages || ["en"],
                name: input.name,
                profileImage: null,
                theme: input.theme || "light",
                sessions: [{
                    __typename: "SessionUserSession",
                    id: `session_${generatePK().toString()}`,
                    createdAt: now,
                    updatedAt: now,
                    lastActivityAt: now,
                    expiresAt: new Date(Date.now() + THIRTY_DAYS_MS).toISOString(),
                    deviceId: NEW_DEVICE_ID,
                    deviceName: NEW_DEVICE_NAME,
                    ip: LOCALHOST_IP,
                    isActual: true,
                }],
            }],
        };
    }

    /**
     * Update not applicable for auth
     */
    updateFromInput(): Session {
        throw new Error("Auth responses don't support standard update operations");
    }

    /**
     * Validate signup input
     */
    async validateCreateInput(input: EmailSignUpInput): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        const errors: Record<string, string> = {};

        if (!input.email) {
            errors.email = "Email is required";
        } else if (!/^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(input.email)) {
            errors.email = "Invalid email format";
        }

        if (!input.password) {
            errors.password = "Password is required";
        } else if (input.password.length < MIN_PASSWORD_LENGTH) {
            errors.password = `Password must be at least ${MIN_PASSWORD_LENGTH} characters`;
        }

        if (!input.name) {
            errors.name = "Name is required";
        }

        return {
            valid: Object.keys(errors).length === 0,
            errors: Object.keys(errors).length > 0 ? errors : undefined,
        };
    }

    /**
     * Validate not applicable for auth
     */
    async validateUpdateInput(): Promise<{
        valid: boolean;
        errors?: Record<string, string>;
    }> {
        return { valid: false, errors: { general: "Auth doesn't support update operations" } };
    }

    /**
     * Create session from login
     */
    createSessionFromLogin(email: string): Session {
        const existingUserId = generatePK().toString();
        const now = new Date().toISOString();

        return {
            __typename: "Session",
            isLoggedIn: true,
            timeZone: "UTC",
            users: [{
                __typename: "SessionUser",
                id: existingUserId,
                credits: String(DEFAULT_EXISTING_USER_CREDITS), // Existing user might have more credits
                creditAccountId: `credit_${existingUserId}`,
                creditSettings: {
                    defaultCreditAmount: String(DEFAULT_NEW_USER_CREDITS),
                    freeCreditAmount: String(DEFAULT_FREE_CREDITS),
                    dailyCreditLimit: String(DEFAULT_DAILY_CREDIT_LIMIT),
                },
                handle: email.split("@")[0],
                hasPremium: Math.random() > PREMIUM_CHANCE_THRESHOLD, // 20% chance of premium
                hasReceivedPhoneVerificationReward: true,
                languages: ["en", "es"],
                name: "Existing User",
                profileImage: `https://example.com/avatar/${existingUserId}`,
                theme: "dark",
                sessions: [{
                    __typename: "SessionUserSession",
                    id: `session_${generatePK().toString()}`,
                    createdAt: now,
                    updatedAt: now,
                    lastActivityAt: now,
                    expiresAt: new Date(Date.now() + THIRTY_DAYS_MS).toISOString(),
                    deviceId: LOGIN_DEVICE_ID,
                    deviceName: LOGIN_DEVICE_NAME,
                    ip: PRIVATE_IP_FIRST,
                    isActual: true,
                }],
            }],
        };
    }

    /**
     * Create guest session
     */
    createGuestSession(): Session {
        return {
            __typename: "Session",
            isLoggedIn: false,
            timeZone: "UTC",
            users: [],
        };
    }

    /**
     * Create multi-account session
     */
    createMultiAccountSession(accountCount = 2): Session {
        const users: SessionUser[] = [];
        const now = new Date().toISOString();

        for (let i = 0; i < accountCount; i++) {
            const userId = generatePK().toString();
            users.push({
                __typename: "SessionUser",
                id: userId,
                credits: String(DEFAULT_NEW_USER_CREDITS * (i + 1)),
                creditAccountId: `credit_${userId}`,
                creditSettings: {
                    defaultCreditAmount: String(DEFAULT_NEW_USER_CREDITS),
                    freeCreditAmount: String(DEFAULT_FREE_CREDITS),
                    dailyCreditLimit: String(DEFAULT_DAILY_CREDIT_LIMIT),
                },
                handle: `user${i + 1}`,
                hasPremium: i === 0, // First account has premium
                hasReceivedPhoneVerificationReward: true,
                languages: ["en"],
                name: `User ${i + 1}`,
                profileImage: i === 0 ? `https://example.com/avatar/${userId}` : null,
                theme: i === 0 ? "dark" : "light",
                sessions: [{
                    __typename: "SessionUserSession",
                    id: `session_${generatePK().toString()}`,
                    createdAt: now,
                    updatedAt: now,
                    lastActivityAt: now,
                    expiresAt: new Date(Date.now() + THIRTY_DAYS_MS).toISOString(),
                    deviceId: `device-${i}`,
                    deviceName: `Device ${i + 1}`,
                    ip: `${PRIVATE_IP_BASE}${i + 1}`,
                    isActual: i === 0, // First is current session
                }],
            });
        }

        return {
            __typename: "Session",
            isLoggedIn: true,
            timeZone: "America/New_York",
            users,
        };
    }

    /**
     * Create password reset token response
     */
    createPasswordResetResponse(): { success: boolean; message: string } {
        return {
            success: true,
            message: "Password reset email sent",
        };
    }

    /**
     * Create OAuth redirect response
     */
    createOAuthRedirectResponse(provider: string): { redirectUrl: string } {
        return {
            redirectUrl: `https://oauth.provider.com/authorize?client_id=test&redirect_uri=http://localhost:3000/auth/callback&provider=${provider}`,
        };
    }

    /**
     * Create wallet init response
     */
    createWalletInitResponse(): { nonce: string; expiresAt: string } {
        return {
            nonce: nanoid(),
            expiresAt: new Date(Date.now() + TEN_MINUTES_MS).toISOString(),
        };
    }
}

/**
 * Pre-configured auth response scenarios
 */
export const authResponseScenarios = {
    // Success scenarios
    loginSuccess: (email?: string) => {
        const factory = new AuthResponseFactory();
        return factory.createSuccessResponse(
            factory.createSessionFromLogin(email || "test@example.com"),
        );
    },

    signupSuccess: (input?: Partial<EmailSignUpInput>) => {
        const factory = new AuthResponseFactory();
        const defaultInput: EmailSignUpInput = {
            email: "newuser@example.com",
            password: "SecurePass123!",
            name: "New User",
            handle: "newuser",
            languages: ["en"],
            theme: "light",
            marketingEmails: true,
            ...input,
        };
        return factory.createSuccessResponse(
            factory.createFromInput(defaultInput),
        );
    },

    guestSuccess: () => {
        const factory = new AuthResponseFactory();
        return factory.createSuccessResponse(factory.createGuestSession());
    },

    multiAccountSuccess: (count?: number) => {
        const factory = new AuthResponseFactory();
        return factory.createSuccessResponse(factory.createMultiAccountSession(count));
    },

    passwordResetRequestSuccess: () => {
        const factory = new AuthResponseFactory();
        return factory.createPasswordResetResponse();
    },

    oauthRedirectSuccess: (provider = "google") => {
        const factory = new AuthResponseFactory();
        return factory.createOAuthRedirectResponse(provider);
    },

    walletInitSuccess: () => {
        const factory = new AuthResponseFactory();
        return factory.createWalletInitResponse();
    },

    // Error scenarios
    loginValidationError: () => {
        const factory = new AuthResponseFactory();
        return factory.createValidationErrorResponse({
            email: "Invalid email or password",
            password: "Invalid email or password",
        });
    },

    signupValidationError: () => {
        const factory = new AuthResponseFactory();
        return factory.createValidationErrorResponse({
            email: "Email already exists",
            password: "Password must be at least 8 characters",
            name: "Name is required",
            handle: "Handle is already taken",
        });
    },

    sessionExpiredError: () => {
        const factory = new AuthResponseFactory();
        return factory.createPermissionErrorResponse("access", ["authenticated"]);
    },

    accountLockedError: () => {
        const factory = new AuthResponseFactory();
        return factory.createBusinessErrorResponse("state", {
            reason: "Account locked due to too many failed login attempts",
            unlockAt: new Date(Date.now() + THIRTY_MINUTES_MS).toISOString(),
            remainingAttempts: 0,
        });
    },

    emailNotVerifiedError: () => {
        const factory = new AuthResponseFactory();
        return factory.createBusinessErrorResponse("state", {
            reason: "Email not verified",
            action: "Please check your email for verification link",
        });
    },

    rateLimitError: () => {
        const factory = new AuthResponseFactory();
        const REMAINING = 0;
        return factory.createRateLimitErrorResponse(
            DEFAULT_RATE_LIMIT,
            REMAINING,
            new Date(Date.now() + FIFTEEN_MINUTES_MS),
        );
    },

    // MSW handlers
    handlers: {
        success: () => new AuthResponseFactory().createMSWHandlers(),
        withErrors: function createWithErrors(errorRate?: number) {
            return new AuthResponseFactory().createMSWHandlers({ errorRate: errorRate ?? DEFAULT_ERROR_RATE });
        },
        withDelay: function createWithDelay(delay?: number) {
            return new AuthResponseFactory().createMSWHandlers({ delay: delay ?? DEFAULT_AUTH_DELAY_MS });
        },
    },
};

// Export factory instance for direct use
export const authResponseFactory = new AuthResponseFactory();
