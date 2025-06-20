import { generatePK } from "@vrooli/shared";
import { type Prisma, type user_auth } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { 
    DbTestFixtures, 
} from "./types.js";

/**
 * Enhanced database fixture factory for Auth (user_auth) model
 * Provides comprehensive testing capabilities for authentication records
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for different auth providers (Password, OAuth)
 * - Session management
 * - Password reset testing
 * - OAuth token handling
 * - Predefined test scenarios
 */
export class AuthDbFactory extends EnhancedDbFactory<
    Prisma.user_authCreateInput,
    Prisma.user_authUpdateInput
> {
    /**
     * Get complete test fixtures for Auth model
     */
    protected getFixtures(): DbTestFixtures<Prisma.user_authCreateInput, Prisma.user_authUpdateInput> {
        const userId = generatePK();
        
        return {
            minimal: {
                id: generatePK(),
                provider: "Password",
                hashed_password: "$2b$10$dummy.hashed.password.for.testing",
                user: {
                    connect: { id: userId }
                },
            },
            complete: {
                id: generatePK(),
                provider: "Password",
                hashed_password: "$2b$10$complex.hashed.password.with.salt",
                resetPasswordCode: `reset_${Math.random().toString(36).substring(2, 34)}`,
                lastResetPasswordRequestAttempt: new Date(Date.now() - 60000), // 1 minute ago
                last_used_at: new Date(),
                user: {
                    connect: { id: userId }
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, provider, and user connection
                    hashed_password: "$2b$10$password.hash",
                },
                invalidTypes: {
                    id: 123 as any, // Should be bigint
                    provider: null as any, // Should be string
                    hashed_password: true as any, // Should be string
                },
                missingUserConnection: {
                    id: generatePK(),
                    provider: "Password",
                    hashed_password: "$2b$10$password.hash",
                    // Missing user connection
                },
            },
            edgeCases: {
                oauthWithTokens: {
                    id: generatePK(),
                    provider: "Google",
                    provider_user_id: `google_${Math.random().toString(36).substring(2, 18)}`,
                    access_token: "encrypted_access_token_" + Math.random().toString(36).substring(2, 66),
                    refresh_token: "encrypted_refresh_token_" + Math.random().toString(36).substring(2, 66),
                    token_expires_at: new Date(Date.now() + 3600000), // 1 hour from now
                    granted_scopes: ["email", "profile", "openid"],
                    last_used_at: new Date(),
                    user: {
                        connect: { id: userId }
                    },
                },
                passwordWithResetCode: {
                    id: generatePK(),
                    provider: "Password",
                    hashed_password: "$2b$10$password.with.reset.code",
                    resetPasswordCode: `reset_${Math.random().toString(36).substring(2, 66)}`,
                    lastResetPasswordRequestAttempt: new Date(),
                    user: {
                        connect: { id: userId }
                    },
                },
                expiredOauth: {
                    id: generatePK(),
                    provider: "GitHub",
                    provider_user_id: `github_${Math.random().toString(36).substring(2, 18)}`,
                    access_token: "expired_access_token_" + Math.random().toString(36).substring(2, 66),
                    refresh_token: "expired_refresh_token_" + Math.random().toString(36).substring(2, 66),
                    token_expires_at: new Date(Date.now() - 3600000), // 1 hour ago
                    granted_scopes: ["read:user", "user:email"],
                    last_used_at: new Date(Date.now() - 86400000), // 1 day ago
                    user: {
                        connect: { id: userId }
                    },
                },
            },
            updates: {
                minimal: {
                    last_used_at: new Date(),
                },
                complete: {
                    hashed_password: "$2b$10$updated.password.hash",
                    resetPasswordCode: null,
                    lastResetPasswordRequestAttempt: null,
                    access_token: "new_encrypted_token",
                    refresh_token: "new_encrypted_refresh",
                    token_expires_at: new Date(Date.now() + 7200000),
                    granted_scopes: ["email", "profile", "new_scope"],
                    last_used_at: new Date(),
                },
            },
        };
    }
}

// Export factory creator function
export const createAuthDbFactory = () => new AuthDbFactory();

// Export the class for type usage
export { AuthDbFactory as AuthDbFactoryClass };