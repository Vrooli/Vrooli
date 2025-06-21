import { generatePK, generatePublicId } from "./idHelpers.js";
import { type Prisma, type PrismaClient } from "@prisma/client";
import { EnhancedDatabaseFactory } from "./EnhancedDatabaseFactory.js";
import type { 
    DbTestFixtures, 
    RelationConfig,
} from "./types.js";

interface AwardRelationConfig extends RelationConfig {
    userId?: string;
    category?: string;
    progress?: number;
}

/**
 * Enhanced database fixture factory for Award model
 * Provides comprehensive testing capabilities for user awards
 * 
 * Features:
 * - Type-safe Prisma integration
 * - Different award categories
 * - Progress tracking scenarios
 * - Tier completion testing
 * - User achievement patterns
 * - Predefined test scenarios
 */
export class AwardDbFactory extends EnhancedDatabaseFactory<
    Prisma.awardCreateInput,
    Prisma.awardCreateInput,
    Prisma.awardInclude,
    Prisma.awardUpdateInput
> {
    constructor(prisma: PrismaClient) {
        super('award', prisma);
    }

    protected getPrismaDelegate() {
        return this.prisma.award;
    }

    protected generateMinimalData(overrides?: Partial<Prisma.awardCreateInput>): Prisma.awardCreateInput {
        return {
            id: generatePK(),
            category: "first_routine",
            progress: 1,
            user: { connect: { id: generatePK() } },
            ...overrides,
        };
    }

    protected generateCompleteData(overrides?: Partial<Prisma.awardCreateInput>): Prisma.awardCreateInput {
        return {
            ...this.generateMinimalData(),
            category: "api_master",
            progress: 25,
            tier: 3,
            ...overrides,
        };
    }

    /**
     * Get complete test fixtures for Award model
     */
    protected getFixtures(): DbTestFixtures<Prisma.awardCreateInput, Prisma.awardUpdateInput> {
        const userId = generatePK();
        
        return {
            minimal: this.generateMinimalData(),
            complete: this.generateCompleteData(),
            invalid: {
                missingRequired: {
                    // Missing id, category, and user
                    progress: 0,
                },
                invalidTypes: {
                    id: 123 as any, // Should be bigint
                    category: null as any, // Should be string
                    progress: "50" as any, // Should be number
                    tierCompletedAt: "not-a-date" as any, // Should be Date
                },
                negativeProgress: {
                    id: generatePK(),
                    category: "invalid_progress",
                    progress: -10,
                    user: {
                        connect: { id: userId }
                    },
                },
            },
            edgeCases: {
                zeroProgress: {
                    id: generatePK(),
                    category: "beginner_badge",
                    progress: 0,
                    user: {
                        connect: { id: userId }
                    },
                },
                partialProgress: {
                    id: generatePK(),
                    category: "collaboration_expert",
                    progress: 75,
                    user: {
                        connect: { id: userId }
                    },
                },
                maxProgress: {
                    id: generatePK(),
                    category: "super_achiever",
                    progress: 999999,
                    tierCompletedAt: new Date(),
                    user: {
                        connect: { id: userId }
                    },
                },
                recentCompletion: {
                    id: generatePK(),
                    category: "fresh_graduate",
                    progress: 100,
                    tierCompletedAt: new Date(Date.now() - 3600000), // 1 hour ago
                    user: {
                        connect: { id: userId }
                    },
                },
                longTimeAchievement: {
                    id: generatePK(),
                    category: "veteran_user",
                    progress: 100,
                    tierCompletedAt: new Date(Date.now() - 365 * 24 * 3600000), // 1 year ago
                    user: {
                        connect: { id: userId }
                    },
                },
            },
            updates: {
                minimal: {
                    progress: 50,
                },
                complete: {
                    progress: 100,
                    tierCompletedAt: new Date(),
                },
            },
        };
    }

    /**
     * Create award with specific progress
     */
    async createWithProgress(config: {
        userId: string;
        category: string;
        progress: number;
        completed?: boolean;
    }) {
        const baseData = this.getFixtures().minimal;
        
        return await this.create({
            ...baseData,
            category: config.category,
            progress: config.progress,
            tierCompletedAt: config.completed ? new Date() : undefined,
            user: { connect: { id: config.userId } },
        });
    }

    /**
     * Create completed award
     */
    async createCompleted(config: {
        userId: string;
        category: string;
        completedAt?: Date;
    }) {
        const baseData = this.getFixtures().complete;
        
        return await this.create({
            ...baseData,
            category: config.category,
            progress: 100,
            tierCompletedAt: config.completedAt || new Date(),
            user: { connect: { id: config.userId } },
        });
    }

    /**
     * Create multiple awards for a user
     */
    async createUserAwards(config: {
        userId: string;
        awards: Array<{
            category: string;
            progress: number;
            completed?: boolean;
        }>;
    }) {
        const results = [];
        
        for (const award of config.awards) {
            const result = await this.createWithProgress({
                userId: config.userId,
                category: award.category,
                progress: award.progress,
                completed: award.completed,
            });
            results.push(result);
        }
        
        return results;
    }
}

// Export factory creator function
export const createAwardDbFactory = (prisma: PrismaClient) => new AwardDbFactory(prisma);

// Export the class for type usage
export { AwardDbFactory as AwardDbFactoryClass };