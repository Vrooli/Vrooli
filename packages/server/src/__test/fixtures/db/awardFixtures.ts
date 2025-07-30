import { generatePK } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";
import { EnhancedDbFactory } from "./EnhancedDbFactory.js";
import type { DbTestFixtures, BulkSeedOptions, BulkSeedResult, DbErrorScenarios } from "./types.js";

/**
 * Database fixtures for Award model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing - using factory function to avoid module-level generatePK() calls
export function getAwardDbIds() {
    return {
        award1: generatePK(),
        award2: generatePK(),
        award3: generatePK(),
        user1: generatePK(),
        user2: generatePK(),
    };
}

/**
 * Enhanced test fixtures for Award model following standard structure
 */
export function getAwardDbFixtures(): DbTestFixtures<Prisma.awardCreateInput> {
    return {
    minimal: {
        id: generatePK(),
        user: { connect: { id: getAwardDbIds().user1 } },
        category: "TestCategory",
        progress: 0,
    },
    complete: {
        id: generatePK(),
        user: { connect: { id: getAwardDbIds().user1 } },
        category: "CompleteCategory",
        progress: 100,
        timeCompleted: new Date(),
    },
    invalid: {
        missingRequired: {
            // Missing required user and category
            progress: 0,
        },
        invalidTypes: {
            id: "not-a-valid-snowflake",
            user: "invalid-user-reference", // Should be connect object
            category: 123, // Should be string
            progress: "invalid", // Should be number
            timeCompleted: "not-a-date", // Should be Date
        },
        invalidProgress: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "InvalidProgress",
            progress: -10, // Negative progress
        },
        progressTooHigh: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "TooHighProgress",
            progress: 150, // Progress over 100
        },
    },
    edgeCases: {
        zeroProgress: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "ZeroProgress",
            progress: 0,
        },
        maxProgress: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "MaxProgress",
            progress: 100,
            timeCompleted: new Date(),
        },
        partialProgress: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "PartialProgress",
            progress: 50,
        },
        longCategoryName: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "A".repeat(100), // Very long category name
            progress: 25,
        },
        specialCharacters: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "Special!@#$%^&*()_+{}|:<>?[]\\/.,;'\"Category",
            progress: 75,
        },
        completedRecently: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "RecentlyCompleted",
            progress: 100,
            timeCompleted: new Date(Date.now() - 1000), // 1 second ago
        },
        completedLongAgo: {
            id: generatePK(),
            user: { connect: { id: getAwardDbIds().user1 } },
            category: "LongAgoCompleted",
            progress: 100,
            timeCompleted: new Date(Date.now() - 365 * 24 * 60 * 60 * 1000), // 1 year ago
        },
    },
    };
}

/**
 * Enhanced factory for creating award database fixtures
 */
export class AwardDbFactory extends EnhancedDbFactory<Prisma.awardCreateInput> {
    
    /**
     * Get the test fixtures for Award model
     */
    protected getFixtures(): DbTestFixtures<Prisma.awardCreateInput> {
        return getAwardDbFixtures();
    }

    /**
     * Generate fresh identifiers for Award (no publicId or handle)
     */
    protected generateFreshIdentifiers(): Record<string, any> {
        return {
            id: generatePK(),
        };
    }

    /**
     * Get Award-specific error scenarios
     */
    protected getErrorScenarios(): DbErrorScenarios {
        return {
            constraints: {
                uniqueViolation: {
                    id: getAwardDbIds().award1, // Duplicate ID
                    user: { connect: { id: getAwardDbIds().user1 } },
                    category: "UniqueViolation",
                    progress: 0,
                },
                foreignKeyViolation: {
                    id: generatePK(),
                    user: { connect: { id: "non-existent-user-id" } },
                    category: "ForeignKeyViolation",
                    progress: 0,
                },
                checkConstraintViolation: {
                    id: generatePK(),
                    user: { connect: { id: getAwardDbIds().user1 } },
                    category: "", // Empty category
                    progress: -1, // Negative progress
                },
            },
            validation: {
                requiredFieldMissing: getAwardDbFixtures().invalid.missingRequired,
                invalidDataType: getAwardDbFixtures().invalid.invalidTypes,
                outOfRange: {
                    id: generatePK(),
                    user: { connect: { id: getAwardDbIds().user1 } },
                    category: "a".repeat(500), // Category too long
                    progress: 1000, // Progress way too high
                },
            },
            businessLogic: {
                completedWithoutFullProgress: {
                    id: generatePK(),
                    user: { connect: { id: getAwardDbIds().user1 } },
                    category: "InconsistentCompletion",
                    progress: 50, // Not 100 but has completion time
                    timeCompleted: new Date(),
                },
                notCompletedWithFullProgress: {
                    id: generatePK(),
                    user: { connect: { id: getAwardDbIds().user1 } },
                    category: "InconsistentProgress",
                    progress: 100, // 100 progress but no completion time
                },
            },
        };
    }

    /**
     * Award-specific validation
     */
    protected validateSpecific(data: Prisma.awardCreateInput): { errors: string[]; warnings: string[] } {
        const errors: string[] = [];
        const warnings: string[] = [];

        // Check required fields specific to Award
        if (!data.user) errors.push("Award user is required");
        if (!data.category) errors.push("Award category is required");
        if (data.progress === undefined) errors.push("Award progress is required");

        // Check business logic
        if (typeof data.progress === "number") {
            if (data.progress < 0) {
                errors.push("Progress cannot be negative");
            }
            if (data.progress > 100) {
                warnings.push("Progress over 100 is unusual");
            }
            if (data.progress === 100 && !data.timeCompleted) {
                warnings.push("Completed award should have timeCompleted");
            }
            if (data.progress < 100 && data.timeCompleted) {
                warnings.push("Incomplete award should not have timeCompleted");
            }
        }

        // Check category
        if (data.category && typeof data.category === "string") {
            if (data.category.length === 0) {
                errors.push("Category cannot be empty");
            }
            if (data.category.length > 255) {
                errors.push("Category is too long");
            }
        }

        return { errors, warnings };
    }

    // Static methods for backward compatibility
    static createMinimal(
        userId: string,
        category: string,
        overrides?: Partial<Prisma.awardCreateInput>,
    ): Prisma.awardCreateInput {
        const factory = new AwardDbFactory();
        return factory.createMinimal({
            user: { connect: { id: userId } },
            category,
            ...overrides,
        });
    }

    static createCompleted(
        userId: string,
        category: string,
        timeCompleted: Date,
        overrides?: Partial<Prisma.awardCreateInput>,
    ): Prisma.awardCreateInput {
        return this.createMinimal(userId, category, {
            progress: 100,
            timeCompleted,
            ...overrides,
        });
    }

    static createInProgress(
        userId: string,
        category: string,
        progress: number,
        overrides?: Partial<Prisma.awardCreateInput>,
    ): Prisma.awardCreateInput {
        return this.createMinimal(userId, category, {
            progress,
            ...overrides,
        });
    }
}

/**
 * Enhanced helper to seed multiple test awards with comprehensive options
 */
export async function seedAwards(
    prisma: any,
    options: {
        userId: string;
        categories: Array<{ name: string; progress?: number; completed?: boolean }>;
    },
): Promise<BulkSeedResult<any>> {
    const awards = [];
    let completedCount = 0;
    let inProgressCount = 0;
    let startedCount = 0;

    for (const cat of options.categories) {
        let awardData: Prisma.awardCreateInput;

        if (cat.completed) {
            awardData = AwardDbFactory.createCompleted(
                options.userId,
                cat.name,
                new Date(),
            );
            completedCount++;
        } else if (cat.progress !== undefined) {
            awardData = AwardDbFactory.createInProgress(
                options.userId,
                cat.name,
                cat.progress,
            );
            inProgressCount++;
        } else {
            awardData = AwardDbFactory.createMinimal(
                options.userId,
                cat.name,
            );
            startedCount++;
        }

        const award = await prisma.award.create({ data: awardData });
        awards.push(award);
    }

    return {
        records: awards,
        summary: {
            total: awards.length,
            withAuth: 0, // Awards don't have auth
            bots: 0, // Awards don't have bots
            teams: 0, // Awards don't have teams
            completed: completedCount,
            inProgress: inProgressCount,
            started: startedCount,
        },
    };
}
