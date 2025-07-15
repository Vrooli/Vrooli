// AI_CHECK: TYPE_SAFETY=server-factory-bigint-migration | LAST: 2025-06-29 - Migrated to BigInt IDs, snake_case tables, correct field names
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";

/**
 * Enhanced database fixture factory for API Key model
 * Uses snake_case table name: 'api_key'
 * 
 * Breaking Changes:
 * - IDs are now BigInt only  
 * - Uses snake_case Prisma types (api_keyCreateInput, api_keyInclude)
 * - Uses correct field names from schema: permissions, limitHard, limitSoft, name, key
 * - Uses direct field assignment for relationships (userId, teamId)
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Support for user and team API keys
 * - Credit limit testing scenarios
 * - Permission configurations
 * - Disabled/enabled states
 */
export class ApiKeyDbFactory extends EnhancedDatabaseFactory<
    { id: bigint },
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
            id: this.generateId(),
            creditsUsed: BigInt(0),
            limitHard: BigInt(25000000000),
            limitSoft: BigInt(25000000000),
            name: "Test API Key",
            key: this.generatePublicId(),
            permissions: {
                readPublic: true,
                readPrivate: false,
                writePrivate: false,
                writeAuth: false,
                readAuth: false,
            },
            stopAtLimit: true,
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.api_keyCreateInput>): Prisma.api_keyCreateInput {
        return {
            id: this.generateId(),
            creditsUsed: BigInt(50),
            limitHard: BigInt(25000000000),
            limitSoft: BigInt(25000000000), 
            name: "Complete Test API Key",
            key: this.generatePublicId(),
            permissions: {
                readPublic: true,
                readPrivate: true,
                writePrivate: true,
                writeAuth: false,
                readAuth: true,
            },
            stopAtLimit: false,
            ...overrides,
        };
    }

    /**
     * Create API key for user
     */
    async createForUser(userId: bigint, overrides?: Partial<Prisma.api_keyCreateInput>) {
        const data = this.generateMinimalData({
            ...overrides,
            user: { connect: { id: userId } },
        });
        return await this.createMinimal(data);
    }

    /**
     * Create API key for team
     */
    async createForTeam(teamId: bigint, overrides?: Partial<Prisma.api_keyCreateInput>) {
        const data = this.generateMinimalData({
            ...overrides,
            team: { connect: { id: teamId } },
        });
        return await this.createMinimal(data);
    }

    /**
     * Create disabled API key
     */
    async createDisabled(overrides?: Partial<Prisma.api_keyCreateInput>) {
        return await this.createMinimal({
            disabledAt: new Date(),
            ...overrides,
        });
    }

    /**
     * Create API key with high credit usage
     */
    async createHighUsage(overrides?: Partial<Prisma.api_keyCreateInput>) {
        return await this.createMinimal({
            creditsUsed: BigInt(24000000000),
            limitHard: BigInt(25000000000),
            ...overrides,
        });
    }

    /**
     * Create API key with exceeded credits
     */
    async createExceededCredits(overrides?: Partial<Prisma.api_keyCreateInput>) {
        return await this.createMinimal({
            creditsUsed: BigInt(26000000000),
            limitHard: BigInt(25000000000),
            ...overrides,
        });
    }

    /**
     * Create API key with read-only permissions
     */
    async createReadOnly(overrides?: Partial<Prisma.api_keyCreateInput>) {
        return await this.createMinimal({
            permissions: {
                readPublic: true,
                readPrivate: false,
                writePrivate: false,
                writeAuth: false,
                readAuth: false,
            },
            ...overrides,
        });
    }

    /**
     * Create API key with full permissions
     */
    async createFullPermissions(overrides?: Partial<Prisma.api_keyCreateInput>) {
        return await this.createMinimal({
            permissions: {
                readPublic: true,
                readPrivate: true,
                writePrivate: true,
                writeAuth: true,
                readAuth: true,
            },
            ...overrides,
        });
    }
}
