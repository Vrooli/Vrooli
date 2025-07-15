import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.js";

interface AuthRelationConfig extends RelationConfig {
    withUser?: boolean;
    userId?: bigint;
    provider?: string;
}

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
export class AuthDbFactory extends EnhancedDatabaseFactory<
    any, // Using any temporarily to avoid type issues
    Prisma.user_authCreateInput,
    Prisma.user_authInclude,
    Prisma.user_authUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("user_auth", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.user_auth;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.user_authCreateInput>): Prisma.user_authCreateInput {
        return {
            id: this.generateId(),
            provider: "Password",
            hashed_password: "$2b$10$dummy.hashed.password.for.testing",
            user: { connect: { id: this.generateId() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.user_authCreateInput>): Prisma.user_authCreateInput {
        return {
            ...this.generateMinimalData(),
            hashed_password: "$2b$10$complex.hashed.password.with.salt",
            resetPasswordCode: `reset_${this.generatePublicId()}`,
            lastResetPasswordRequestAttempt: new Date(Date.now() - 60000), // 1 minute ago
            last_used_at: new Date(),
            ...overrides,
        };
    }
    /**
     * Get complete test fixtures for Auth model
     */
    protected getFixtures(): DbTestFixtures<Prisma.user_authCreateInput, Prisma.user_authUpdateInput> {
        const userId = this.generateId();
        
        return {
            minimal: this.generateMinimalData(),
            complete: this.generateCompleteData(),
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
                    id: this.generateId(),
                    provider: "Password",
                    hashed_password: "$2b$10$password.hash",
                    // Missing user connection
                },
            },
            edgeCases: {
                oauthWithTokens: {
                    id: this.generateId(),
                    provider: "Google",
                    provider_user_id: `google_${Math.random().toString(36).substring(2, 18)}`,
                    access_token: "encrypted_access_token_" + Math.random().toString(36).substring(2, 66),
                    refresh_token: "encrypted_refresh_token_" + Math.random().toString(36).substring(2, 66),
                    token_expires_at: new Date(Date.now() + 3600000), // 1 hour from now
                    granted_scopes: ["email", "profile", "openid"],
                    last_used_at: new Date(),
                    user: {
                        connect: { id: userId },
                    },
                },
                passwordWithResetCode: {
                    id: this.generateId(),
                    provider: "Password",
                    hashed_password: "$2b$10$password.with.reset.code",
                    resetPasswordCode: `reset_${Math.random().toString(36).substring(2, 66)}`,
                    lastResetPasswordRequestAttempt: new Date(),
                    user: {
                        connect: { id: userId },
                    },
                },
                expiredOauth: {
                    id: this.generateId(),
                    provider: "GitHub",
                    provider_user_id: `github_${Math.random().toString(36).substring(2, 18)}`,
                    access_token: "expired_access_token_" + Math.random().toString(36).substring(2, 66),
                    refresh_token: "expired_refresh_token_" + Math.random().toString(36).substring(2, 66),
                    token_expires_at: new Date(Date.now() - 3600000), // 1 hour ago
                    granted_scopes: ["read:user", "user:email"],
                    last_used_at: new Date(Date.now() - 86400000), // 1 day ago
                    user: {
                        connect: { id: userId },
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
export const createAuthDbFactory = (prisma: PrismaClient) => new AuthDbFactory(prisma);

// Export the class for type usage
export { AuthDbFactory as AuthDbFactoryClass };
