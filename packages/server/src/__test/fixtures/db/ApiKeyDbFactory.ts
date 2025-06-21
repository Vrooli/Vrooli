import { generatePK, generatePublicId } from "./idHelpers.js";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.js";

interface ApiKeyRelationConfig extends RelationConfig {
    withTeam?: boolean;
    withUser?: boolean;
    teamId?: string;
    userId?: string;
}

/**
 * Enhanced database fixture factory for API Key model
 * Provides comprehensive testing capabilities for API keys
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for user and team API keys
 * - Credit limit testing scenarios
 * - Permission configurations
 * - Disabled/enabled states
 * - Predefined test scenarios
 */
export class ApiKeyDbFactory extends EnhancedDatabaseFactory<
    Prisma.api_keyCreateInput,
    Prisma.api_keyCreateInput,
    Prisma.api_keyInclude,
    Prisma.api_keyUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super("api_key", prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.api_key;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.api_keyCreateInput>): Prisma.api_keyCreateInput {
        return {
            id: generatePK(),
            key: `api_key_${generatePublicId()}`,
            name: "Test API Key",
            user: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.api_keyCreateInput>): Prisma.api_keyCreateInput {
        return {
            ...this.generateMinimalData(),
            creditsUsed: BigInt(1000),
            limitHard: BigInt(10000000),
            limitSoft: BigInt(8000000),
            stopAtLimit: true,
            permissions: { "read": true, "write": false },
            ...overrides,
        };
    }

    /**
     * Get complete test fixtures for API Key model
     */
    protected getFixtures(): DbTestFixtures<Prisma.api_keyCreateInput, Prisma.api_keyUpdateInput> {
        const userId = generatePK();
        const teamId = generatePK();
        
        return {
            minimal: this.generateMinimalData(),
            complete: this.generateCompleteData(),
            invalid: {
                missingRequired: {
                    // Missing id, key, and name
                    user: {
                        connect: { id: userId },
                    },
                },
                invalidTypes: {
                    id: 123 as any, // Should be bigint
                    key: null as any, // Should be string
                    creditsUsed: "1000" as any, // Should be bigint
                    limitHard: -100 as any, // Should be positive
                },
                duplicateKey: {
                    id: generatePK(),
                    key: "existing_api_key", // Assumes this exists
                    name: "Duplicate Key",
                    user: {
                        connect: { id: userId },
                    },
                },
            },
            edgeCases: {
                teamApiKey: {
                    id: generatePK(),
                    key: `team_api_key_${generatePublicId()}`,
                    name: "Team API Key",
                    limitHard: BigInt(50000000),
                    limitSoft: BigInt(40000000),
                    permissions: { "admin": true },
                    team: {
                        connect: { id: teamId },
                    },
                },
                disabledKey: {
                    id: generatePK(),
                    key: `disabled_key_${generatePublicId()}`,
                    name: "Disabled API Key",
                    disabledAt: new Date(),
                    user: {
                        connect: { id: userId },
                    },
                },
                limitReached: {
                    id: generatePK(),
                    key: `limit_reached_${generatePublicId()}`,
                    name: "Limit Reached Key",
                    creditsUsed: BigInt(10000),
                    limitHard: BigInt(10000),
                    limitSoft: BigInt(8000),
                    stopAtLimit: true,
                    user: {
                        connect: { id: userId },
                    },
                },
                noLimits: {
                    id: generatePK(),
                    key: `unlimited_${generatePublicId()}`,
                    name: "Unlimited Key",
                    limitHard: BigInt(999999999999),
                    limitSoft: null,
                    stopAtLimit: false,
                    permissions: { "unlimited": true },
                    user: {
                        connect: { id: userId },
                    },
                },
            },
            updates: {
                minimal: {
                    creditsUsed: BigInt(500),
                },
                complete: {
                    name: "Updated API Key",
                    creditsUsed: BigInt(2000),
                    limitSoft: BigInt(9000000),
                    permissions: { "read": true, "write": true },
                    disabledAt: new Date(),
                },
            },
        };
    }

    /**
     * Create API key with specific ownership
     */
    async createWithOwnership(config: {
        userId?: string;
        teamId?: string;
        overrides?: Partial<Prisma.api_keyCreateInput>;
    }) {
        const baseData = this.getFixtures().complete;
        
        return await this.create({
            ...baseData,
            user: config.userId ? { connect: { id: config.userId } } : undefined,
            team: config.teamId ? { connect: { id: config.teamId } } : undefined,
            ...config.overrides,
        });
    }

    /**
     * Create API key with specific credit limits
     */
    async createWithLimits(config: {
        hardLimit: bigint;
        softLimit?: bigint;
        creditsUsed?: bigint;
        userId: string;
    }) {
        const baseData = this.getFixtures().minimal;
        
        return await this.create({
            ...baseData,
            limitHard: config.hardLimit,
            limitSoft: config.softLimit || null,
            creditsUsed: config.creditsUsed || BigInt(0),
            user: { connect: { id: config.userId } },
        });
    }
}

// Export factory creator function
export const createApiKeyDbFactory = (prisma: PrismaClient) => new ApiKeyDbFactory(prisma);

// Export the class for type usage
export { ApiKeyDbFactory as ApiKeyDbFactoryClass };
