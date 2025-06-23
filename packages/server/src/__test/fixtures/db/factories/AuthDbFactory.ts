import { generatePK, generatePublicId, nanoid } from "../idHelpers.js";
import { type user_auth, type Prisma, type PrismaClient } from "@prisma/client";
import { DatabaseFixtureFactory } from "../DatabaseFixtureFactory.js";
import type { RelationConfig } from "../DatabaseFixtureFactory.js";

interface AuthRelationConfig extends RelationConfig {
    withSessions?: boolean | number;
    withUser?: boolean;
}

/**
 * Database fixture factory for user_auth model
 * Handles authentication records including password and OAuth providers
 */
export class AuthDbFactory extends DatabaseFixtureFactory<
    user_auth,
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

    protected getMinimalData(overrides?: Partial<Prisma.user_authCreateInput>): Prisma.user_authCreateInput {
        return {
            id: generatePK(),
            provider: "Password",
            hashed_password: "$2b$10$dummy.hashed.password.for.testing",
            user: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    protected getCompleteData(overrides?: Partial<Prisma.user_authCreateInput>): Prisma.user_authCreateInput {
        return {
            id: generatePK(),
            provider: "Password",
            hashed_password: "$2b$10$complex.hashed.password.with.salt",
            resetPasswordCode: `reset_${nanoid(32)}`,
            lastResetPasswordRequestAttempt: new Date(Date.now() - 60 * 60 * 1000), // 1 hour ago
            last_used_at: new Date(),
            user: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    /**
     * Create OAuth authentication
     */
    async createOAuth(
        provider: "Google" | "GitHub" | "Twitter" | "Discord",
        overrides?: Partial<Prisma.user_authCreateInput>,
    ): Promise<user_auth> {
        const data: Prisma.user_authCreateInput = {
            id: generatePK(),
            provider,
            provider_user_id: `${provider.toLowerCase()}_${nanoid(16)}`,
            access_token: `encrypted_access_${nanoid(64)}`,
            refresh_token: `encrypted_refresh_${nanoid(64)}`,
            token_expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000), // 30 days
            granted_scopes: this.getDefaultScopesForProvider(provider),
            last_used_at: new Date(),
            user: { connect: { id: generatePK() } },
            ...overrides,
        };
        
        const result = await this.prisma.user_auth.create({ data });
        this.trackCreatedId(result.id.toString());
        return result;
    }

    /**
     * Create locked account authentication
     */
    async createLocked(overrides?: Partial<Prisma.user_authCreateInput>): Promise<user_auth> {
        return this.createMinimal({
            hashed_password: "$2b$10$locked.account.password.hash",
            lastResetPasswordRequestAttempt: new Date(), // Recent attempt
            ...overrides,
        });
    }

    /**
     * Create auth with pending password reset
     */
    async createWithPasswordReset(overrides?: Partial<Prisma.user_authCreateInput>): Promise<user_auth> {
        return this.createComplete({
            resetPasswordCode: `reset_${nanoid(32)}`,
            lastResetPasswordRequestAttempt: new Date(Date.now() - 10 * 60 * 1000), // 10 minutes ago
            ...overrides,
        });
    }

    protected getDefaultInclude(): Prisma.user_authInclude {
        return {
            user: true,
            sessions: {
                orderBy: { createdAt: "desc" },
                take: 5,
            },
            _count: {
                select: {
                    sessions: true,
                },
            },
        };
    }

    protected async applyRelationships(
        baseData: Prisma.user_authCreateInput,
        config: AuthRelationConfig,
        tx: any,
    ): Promise<Prisma.user_authCreateInput> {
        const data = { ...baseData };

        // Handle user connection (required)
        if (config.withUser || !data.user) {
            const user = await tx.user.create({
                data: {
                    id: generatePK(),
                    publicId: generatePublicId(),
                    name: "Auth Test User",
                    handle: `auth_user_${nanoid(8)}`,
                    status: "Unlocked",
                    isBot: false,
                    isBotDepictingPerson: false,
                    isPrivate: false,
                },
            });
            data.user = { connect: { id: user.id } };
        }

        // Handle sessions
        if (config.withSessions) {
            const sessionCount = typeof config.withSessions === "number" ? config.withSessions : 1;
            data.sessions = {
                create: Array.from({ length: sessionCount }, (_, i) => ({
                    id: generatePK(),
                    expires_at: new Date(Date.now() + (30 - i) * 24 * 60 * 60 * 1000), // Staggered expiry
                    last_refresh_at: new Date(Date.now() - i * 60 * 60 * 1000),
                    device_info: `Device ${i + 1}`,
                    ip_address: `192.168.1.${100 + i}`,
                    user: data.user, // Connect to same user
                })),
            };
        }

        return data;
    }

    /**
     * Get default OAuth scopes for provider
     */
    private getDefaultScopesForProvider(provider: string): string[] {
        const scopeMap: Record<string, string[]> = {
            Google: ["email", "profile", "openid"],
            GitHub: ["read:user", "user:email"],
            Twitter: ["tweet.read", "users.read", "offline.access"],
            Discord: ["identify", "email"],
        };
        return scopeMap[provider] || [];
    }

    /**
     * Create test scenarios
     */
    async createMultiAuthUser(): Promise<user_auth[]> {
        const user = await this.prisma.user.create({
            data: {
                id: generatePK(),
                publicId: generatePublicId(),
                name: "Multi-Auth User",
                handle: `multiauth_${nanoid(8)}`,
                status: "Unlocked",
                isBot: false,
                isBotDepictingPerson: false,
                isPrivate: false,
            },
        });

        const auths = await Promise.all([
            this.createMinimal({ user: { connect: { id: user.id } } }),
            this.createOAuth("Google", { user: { connect: { id: user.id } } }),
            this.createOAuth("GitHub", { user: { connect: { id: user.id } } }),
        ]);

        return auths;
    }

    async createExpiredOAuth(): Promise<user_auth> {
        return this.createOAuth("Google", {
            token_expires_at: new Date(Date.now() - 24 * 60 * 60 * 1000), // Expired 1 day ago
            last_used_at: new Date(Date.now() - 2 * 24 * 60 * 60 * 1000), // Last used 2 days ago
        });
    }

    protected async checkModelConstraints(record: user_auth): Promise<string[]> {
        const violations: string[] = [];
        
        // Check provider is valid
        const validProviders = ["Password", "Google", "GitHub", "Twitter", "Discord"];
        if (!validProviders.includes(record.provider)) {
            violations.push(`Invalid provider: ${record.provider}`);
        }

        // Check password auth has hashed password
        if (record.provider === "Password" && !record.hashed_password) {
            violations.push("Password provider must have hashed_password");
        }

        // Check OAuth has provider_user_id
        if (record.provider !== "Password" && !record.provider_user_id) {
            violations.push("OAuth provider must have provider_user_id");
        }

        // Check token expiry is in future for active OAuth
        if (record.access_token && record.token_expires_at) {
            if (record.token_expires_at < new Date()) {
                violations.push("OAuth tokens appear to be expired");
            }
        }

        // Check user exists
        const user = await this.prisma.user.findUnique({
            where: { id: record.user_id },
        });
        if (!user) {
            violations.push("Associated user not found");
        }

        return violations;
    }

    /**
     * Get invalid data scenarios
     */
    getInvalidScenarios(): Record<string, any> {
        return {
            missingRequired: {
                // Missing id, provider
                hashed_password: "$2b$10$invalid.password.hash",
            },
            invalidTypes: {
                id: "not-a-snowflake",
                provider: 123, // Should be string
                user_id: "not-a-bigint", // Should be BigInt
                granted_scopes: "not-an-array", // Should be array
            },
            invalidProvider: {
                id: generatePK(),
                provider: "InvalidProvider",
                hashed_password: "$2b$10$dummy.password.hash",
                user: { connect: { id: generatePK() } },
            },
            passwordWithoutHash: {
                id: generatePK(),
                provider: "Password",
                // Missing hashedPassword
                user: { connect: { id: generatePK() } },
            },
            oauthWithoutProviderId: {
                id: generatePK(),
                provider: "Google",
                // Missing provider_user_id
                access_token: "encrypted_token",
                user: { connect: { id: generatePK() } },
            },
        };
    }

    /**
     * Get edge case scenarios
     */
    getEdgeCaseScenarios(): Record<string, Prisma.user_authCreateInput> {
        return {
            maxScopesOAuth: {
                ...this.getMinimalData(),
                provider: "Google",
                provider_user_id: `google_${nanoid(16)}`,
                granted_scopes: Array.from({ length: 50 }, (_, i) => `scope_${i}`),
            },
            recentPasswordReset: {
                ...this.getCompleteData(),
                resetPasswordCode: `urgent_reset_${nanoid(64)}`,
                lastResetPasswordRequestAttempt: new Date(Date.now() - 30 * 1000), // 30 seconds ago
            },
            longTokens: {
                ...this.getMinimalData(),
                provider: "GitHub",
                provider_user_id: `github_${nanoid(16)}`,
                access_token: `encrypted_${nanoid(1000)}`, // Near max length
                refresh_token: `encrypted_${nanoid(1000)}`,
            },
            frequentlyUsedAuth: {
                ...this.getCompleteData(),
                last_used_at: new Date(),
                sessions: {
                    create: Array.from({ length: 10 }, (_, i) => ({
                        id: generatePK(),
                        expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
                        last_refresh_at: new Date(Date.now() - i * 60 * 1000),
                        device_info: `Frequent Device ${i}`,
                        ip_address: `10.0.0.${i}`,
                        user: { connect: { id: generatePK() } },
                    })),
                },
            },
        };
    }

    protected getCascadeInclude(): any {
        return {
            sessions: true,
            user: true,
        };
    }

    protected async deleteRelatedRecords(
        record: user_auth & {
            sessions?: any[];
        },
        remainingDepth: number,
        tx: any,
    ): Promise<void> {
        // Delete sessions
        if (record.sessions?.length) {
            await tx.session.deleteMany({
                where: { auth_id: record.id },
            });
        }
        
        // Note: We don't delete the user as other auths might be using it
    }
}

// Export factory creator function
export const createAuthDbFactory = (prisma: PrismaClient) => new AuthDbFactory(prisma);
