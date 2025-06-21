import { generatePK, generatePublicId } from "./idHelpers.js";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.js";

interface SessionRelationConfig extends RelationConfig {
    withUser?: boolean;
    userId?: bigint;
    expired?: boolean;
}

/**
 * Enhanced database fixture factory for Session model
 * Provides comprehensive testing capabilities for user sessions
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Session token management
 * - Device and IP tracking
 * - Session expiry handling
 * - Revocation testing
 * - Predefined test scenarios
 */
export class SessionDbFactory extends EnhancedDatabaseFactory<
    any, // Using any temporarily to avoid type issues
    Prisma.sessionCreateInput,
    Prisma.sessionInclude,
    Prisma.sessionUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('session', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.session;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.sessionCreateInput>): Prisma.sessionCreateInput {
        return {
            id: generatePK(),
            secret: `session_${generatePublicId()}`,
            expiresAt: new Date(Date.now() + 24 * 60 * 60 * 1000), // 24 hours from now
            user: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.sessionCreateInput>): Prisma.sessionCreateInput {
        return {
            ...this.generateMinimalData(),
            secret: `complete_session_${generatePublicId()}`,
            theme: "light",
            device: "desktop",
            ipAddress: "192.168.1.100",
            userAgent: "Mozilla/5.0 (Test Browser)",
            ...overrides,
        };
    }

    /**
     * Get complete test fixtures for Session model
     */
    protected getFixtures(): DbTestFixtures<Prisma.sessionCreateInput, Prisma.sessionUpdateInput> {
        const userId = generatePK();
        
        return {
            minimal: this.generateMinimalData(),
            complete: this.generateCompleteData(),
                    platform: "desktop",
                    language: "en-US",
                    timezone: "America/New_York"
                }),
                ip_address: "192.168.1.100",
                user: {
                    connect: { id: userId }
                },
                auth: {
                    connect: { id: authId }
                },
            },
            invalid: {
                missingRequired: {
                    // Missing id, expires_at, user, and auth connections
                    device_info: "Some device",
                },
                invalidTypes: {
                    id: 123 as any, // Should be bigint
                    expires_at: "not-a-date" as any, // Should be Date
                    revokedAt: "not-a-date" as any, // Should be Date or null
                },
                missingUserAuth: {
                    id: generatePK(),
                    expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
                    // Missing user and auth connections
                },
            },
            edgeCases: {
                recentlyRefreshed: {
                    id: generatePK(),
                    last_refresh_at: new Date(),
                    expires_at: new Date(Date.now() + 30 * 24 * 60 * 60 * 1000),
                    device_info: JSON.stringify({
                        userAgent: "Mozilla/5.0 (iPhone; CPU iPhone OS 17_0 like Mac OS X)",
                        browser: "Safari",
                        version: "17.0",
                        os: "iOS",
                        platform: "mobile",
                        language: "en-US"
                    }),
                    ip_address: "10.0.0.5",
                    user: {
                        connect: { id: userId }
                    },
                    auth: {
                        connect: { id: authId }
                    },
                },
                revokedSession: {
                    id: generatePK(),
                    expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000),
                    revokedAt: new Date(),
                    device_info: JSON.stringify({
                        userAgent: "Mozilla/5.0 (Linux; Android 13) AppleWebKit/537.36",
                        browser: "Chrome",
                        version: "114.0.0.0",
                        os: "Android",
                        platform: "mobile",
                        reason: "Suspicious activity detected"
                    }),
                    ip_address: "203.0.113.0",
                    user: {
                        connect: { id: userId }
                    },
                    auth: {
                        connect: { id: authId }
                    },
                },
                expiredSession: {
                    id: generatePK(),
                    last_refresh_at: new Date(Date.now() - 31 * 24 * 60 * 60 * 1000), // 31 days ago
                    expires_at: new Date(Date.now() - 86400000), // 1 day ago (expired)
                    device_info: JSON.stringify({
                        userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7)",
                        browser: "Safari",
                        version: "16.5",
                        os: "macOS",
                        platform: "desktop"
                    }),
                    ip_address: "172.16.0.10",
                    user: {
                        connect: { id: userId }
                    },
                    auth: {
                        connect: { id: authId }
                    },
                },
                longExpiry: {
                    id: generatePK(),
                    last_refresh_at: new Date(),
                    expires_at: new Date(Date.now() + 365 * 24 * 60 * 60 * 1000), // 1 year from now
                    device_info: JSON.stringify({
                        userAgent: "VrooliApp/1.0 (Enterprise Edition)",
                        browser: "Embedded",
                        version: "1.0",
                        os: "Enterprise",
                        platform: "api",
                        session_type: "long_lived"
                    }),
                    ip_address: "10.10.10.100",
                    user: {
                        connect: { id: userId }
                    },
                    auth: {
                        connect: { id: authId }
                    },
                },
                neverRefreshed: {
                    id: generatePK(),
                    last_refresh_at: null, // Never refreshed
                    expires_at: new Date(Date.now() + 7 * 24 * 60 * 60 * 1000), // 7 days
                    device_info: JSON.stringify({
                        userAgent: "curl/7.68.0",
                        browser: "CLI",
                        version: "7.68.0",
                        os: "Linux",
                        platform: "api"
                    }),
                    ip_address: "127.0.0.1",
                    user: {
                        connect: { id: userId }
                    },
                    auth: {
                        connect: { id: authId }
                    },
                },
            },
            updates: {
                minimal: {
                    last_refresh_at: new Date(),
                },
                complete: {
                    last_refresh_at: new Date(),
                    expires_at: new Date(Date.now() + 60 * 24 * 60 * 60 * 1000), // Extend to 60 days
                    device_info: JSON.stringify({
                        userAgent: "Mozilla/5.0 (Updated Browser)",
                        browser: "Updated",
                        version: "Latest",
                        last_updated: new Date().toISOString()
                    }),
                    ip_address: "192.168.1.101",
                },
            },
        };
    }

}

// Export factory creator function
export const createSessionDbFactory = (prisma: PrismaClient) => new SessionDbFactory(prisma);

// Export the class for type usage
export { SessionDbFactory as SessionDbFactoryClass };