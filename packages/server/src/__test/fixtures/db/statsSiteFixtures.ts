// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-03 - Fixed type safety issues: replaced any with PrismaClient type
import { generatePK, StatPeriodType } from "@vrooli/shared";
import { type Prisma, type PrismaClient } from "@prisma/client";

/**
 * Database fixtures for StatsSite model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const statsSiteDbIds = {
    daily1: generatePK(),
    daily2: generatePK(),
    weekly1: generatePK(),
    monthly1: generatePK(),
    yearly1: generatePK(),
};

/**
 * Minimal StatsSite data for database creation
 */
export const minimalStatsSiteDb: Prisma.stats_siteCreateInput = {
    id: statsSiteDbIds.daily1,
    periodStart: new Date("2024-01-01T00:00:00Z"),
    periodEnd: new Date("2024-01-01T23:59:59Z"),
    periodType: "Daily",
    activeUsers: 10,
    teamsCreated: 2,
    verifiedEmailsCreated: 8,
    verifiedWalletsCreated: 3,
    routineComplexityAverage: 5.5,
    runCompletionTimeAverage: 120.0,
    runContextSwitchesAverage: 2.3,
    runsCompleted: 25,
    runsStarted: 30,
};

/**
 * Complete StatsSite with all optional fields
 */
export const completeStatsSiteDb: Prisma.stats_siteCreateInput = {
    id: statsSiteDbIds.daily2,
    periodStart: new Date("2024-01-02T00:00:00Z"),
    periodEnd: new Date("2024-01-02T23:59:59Z"),
    periodType: "Daily",
    activeUsers: 50,
    teamsCreated: 5,
    verifiedEmailsCreated: 35,
    verifiedWalletsCreated: 15,
    resourcesCreatedByType: {
        "ROUTINE": 25,
        "CODE": 12,
        "PROJECT": 8,
        "API": 3,
        "STANDARD": 2,
    },
    resourcesCompletedByType: {
        "ROUTINE": 20,
        "CODE": 10,
        "PROJECT": 5,
        "API": 2,
        "STANDARD": 1,
    },
    resourceCompletionTimeAverageByType: {
        "ROUTINE": 180.5,
        "CODE": 45.2,
        "PROJECT": 600.0,
        "API": 90.3,
        "STANDARD": 120.7,
    },
    routineComplexityAverage: 7.2,
    runCompletionTimeAverage: 156.8,
    runContextSwitchesAverage: 3.1,
    runsCompleted: 85,
    runsStarted: 95,
};

/**
 * Weekly StatsSite fixture
 */
export const weeklyStatsSiteDb: Prisma.stats_siteCreateInput = {
    id: statsSiteDbIds.weekly1,
    periodStart: new Date("2024-01-01T00:00:00Z"),
    periodEnd: new Date("2024-01-07T23:59:59Z"),
    periodType: "Weekly",
    activeUsers: 200,
    teamsCreated: 15,
    verifiedEmailsCreated: 150,
    verifiedWalletsCreated: 45,
    resourcesCreatedByType: {
        "ROUTINE": 100,
        "CODE": 60,
        "PROJECT": 25,
        "API": 8,
        "STANDARD": 5,
    },
    resourcesCompletedByType: {
        "ROUTINE": 85,
        "CODE": 55,
        "PROJECT": 20,
        "API": 7,
        "STANDARD": 4,
    },
    resourceCompletionTimeAverageByType: {
        "ROUTINE": 165.3,
        "CODE": 42.8,
        "PROJECT": 580.2,
        "API": 85.1,
        "STANDARD": 115.6,
    },
    routineComplexityAverage: 6.8,
    runCompletionTimeAverage: 142.5,
    runContextSwitchesAverage: 2.9,
    runsCompleted: 420,
    runsStarted: 465,
};

/**
 * Monthly StatsSite fixture
 */
export const monthlyStatsSiteDb: Prisma.stats_siteCreateInput = {
    id: statsSiteDbIds.monthly1,
    periodStart: new Date("2024-01-01T00:00:00Z"),
    periodEnd: new Date("2024-01-31T23:59:59Z"),
    periodType: "Monthly",
    activeUsers: 1250,
    teamsCreated: 85,
    verifiedEmailsCreated: 950,
    verifiedWalletsCreated: 280,
    resourcesCreatedByType: {
        "ROUTINE": 650,
        "CODE": 380,
        "PROJECT": 150,
        "API": 45,
        "STANDARD": 25,
    },
    resourcesCompletedByType: {
        "ROUTINE": 580,
        "CODE": 340,
        "PROJECT": 125,
        "API": 40,
        "STANDARD": 22,
    },
    resourceCompletionTimeAverageByType: {
        "ROUTINE": 158.7,
        "CODE": 41.2,
        "PROJECT": 555.8,
        "API": 82.4,
        "STANDARD": 112.3,
    },
    routineComplexityAverage: 6.5,
    runCompletionTimeAverage: 148.2,
    runContextSwitchesAverage: 2.7,
    runsCompleted: 2850,
    runsStarted: 3200,
};

/**
 * Yearly StatsSite fixture
 */
export const yearlyStatsSiteDb: Prisma.stats_siteCreateInput = {
    id: statsSiteDbIds.yearly1,
    periodStart: new Date("2024-01-01T00:00:00Z"),
    periodEnd: new Date("2024-12-31T23:59:59Z"),
    periodType: "Yearly",
    activeUsers: 15000,
    teamsCreated: 1200,
    verifiedEmailsCreated: 12000,
    verifiedWalletsCreated: 3500,
    resourcesCreatedByType: {
        "ROUTINE": 8500,
        "CODE": 4800,
        "PROJECT": 2000,
        "API": 600,
        "STANDARD": 300,
    },
    resourcesCompletedByType: {
        "ROUTINE": 7650,
        "CODE": 4320,
        "PROJECT": 1700,
        "API": 540,
        "STANDARD": 270,
    },
    resourceCompletionTimeAverageByType: {
        "ROUTINE": 152.3,
        "CODE": 39.8,
        "PROJECT": 520.5,
        "API": 78.9,
        "STANDARD": 108.7,
    },
    routineComplexityAverage: 6.2,
    runCompletionTimeAverage: 145.8,
    runContextSwitchesAverage: 2.5,
    runsCompleted: 35000,
    runsStarted: 38500,
};

/**
 * Factory for creating StatsSite database fixtures with overrides
 */
export class StatsSiteDbFactory {
    static createMinimal(overrides?: Partial<Prisma.stats_siteCreateInput>): Prisma.stats_siteCreateInput {
        return {
            ...minimalStatsSiteDb,
            id: generatePK(),
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.stats_siteCreateInput>): Prisma.stats_siteCreateInput {
        return {
            ...completeStatsSiteDb,
            id: generatePK(),
            ...overrides,
        };
    }

    static createForPeriod(
        periodType: "Hourly" | "Daily" | "Weekly" | "Monthly" | "Yearly",
        periodStart: Date,
        periodEnd: Date,
        overrides?: Partial<Prisma.stats_siteCreateInput>,
    ): Prisma.stats_siteCreateInput {
        const baseStats = this.createComplete({
            periodType,
            periodStart,
            periodEnd,
            ...overrides,
        });

        // Scale stats based on period type
        const scalingFactors = {
            Hourly: 0.04,  // 1/24th of daily
            Daily: 1,
            Weekly: 7,
            Monthly: 30,
            Yearly: 365,
        };

        const scale = scalingFactors[periodType];
        
        return {
            ...baseStats,
            activeUsers: Math.round(baseStats.activeUsers * scale * 0.8), // Not linear scaling for users
            teamsCreated: Math.round(baseStats.teamsCreated * scale),
            verifiedEmailsCreated: Math.round(baseStats.verifiedEmailsCreated * scale),
            verifiedWalletsCreated: Math.round(baseStats.verifiedWalletsCreated * scale),
            runsStarted: Math.round(baseStats.runsStarted * scale),
            runsCompleted: Math.round(baseStats.runsCompleted * scale),
        };
    }

    static createWithResourceStats(
        resourceStats: {
            created: Record<string, number>;
            completed: Record<string, number>;
            averageCompletionTimes: Record<string, number>;
        },
        overrides?: Partial<Prisma.stats_siteCreateInput>,
    ): Prisma.stats_siteCreateInput {
        return {
            ...this.createMinimal(overrides),
            resourcesCreatedByType: resourceStats.created,
            resourcesCompletedByType: resourceStats.completed,
            resourceCompletionTimeAverageByType: resourceStats.averageCompletionTimes,
        };
    }

    static createSequence(
        count: number,
        periodType: "Daily" | "Weekly" | "Monthly" = "Daily",
        startDate: Date = new Date("2024-01-01T00:00:00Z"),
    ): Prisma.stats_siteCreateInput[] {
        const sequence = [];
        const intervalMs = {
            Daily: 24 * 60 * 60 * 1000,
            Weekly: 7 * 24 * 60 * 60 * 1000,
            Monthly: 30 * 24 * 60 * 60 * 1000,
        };

        for (let i = 0; i < count; i++) {
            const periodStart = new Date(startDate.getTime() + (i * intervalMs[periodType]));
            const periodEnd = new Date(periodStart.getTime() + intervalMs[periodType] - 1);
            
            sequence.push(this.createForPeriod(periodType, periodStart, periodEnd, {
                // Vary stats slightly for each period
                activeUsers: Math.round(50 + (Math.random() * 20) - 10), // 40-60 range
                teamsCreated: Math.round(5 + (Math.random() * 4) - 2), // 3-7 range
            }));
        }

        return sequence;
    }
}

/**
 * Helper to seed StatsSite data for testing
 */
export async function seedStatsSite(
    prisma: any,
    options: {
        periods?: Array<{
            type: "Hourly" | "Daily" | "Weekly" | "Monthly" | "Yearly";
            start: Date;
            end: Date;
        }>;
        count?: number;
        withResourceStats?: boolean;
    } = {},
) {
    const stats = [];

    if (options.periods) {
        // Create specific periods
        for (const period of options.periods) {
            const stat = await prisma.stats_site.create({
                data: StatsSiteDbFactory.createForPeriod(
                    period.type,
                    period.start,
                    period.end,
                    options.withResourceStats ? {
                        resourcesCreatedByType: { "ROUTINE": 10, "CODE": 5 },
                        resourcesCompletedByType: { "ROUTINE": 8, "CODE": 4 },
                        resourceCompletionTimeAverageByType: { "ROUTINE": 120.5, "CODE": 60.2 },
                    } : {},
                ),
            });
            stats.push(stat);
        }
    } else {
        // Create sequence of daily stats
        const count = options.count || 7;
        const sequence = StatsSiteDbFactory.createSequence(count);
        
        for (const data of sequence) {
            const stat = await prisma.stats_site.create({ data });
            stats.push(stat);
        }
    }

    return stats;
}

/**
 * Helper to create stats covering different time periods for comprehensive testing
 */
export async function seedComprehensiveStatsSite(prisma: PrismaClient) {
    const stats = [];

    // Create one stat for each period type
    const fixtures = [
        minimalStatsSiteDb,
        completeStatsSiteDb,
        weeklyStatsSiteDb,
        monthlyStatsSiteDb,
        yearlyStatsSiteDb,
    ];

    for (const fixture of fixtures) {
        const stat = await prisma.stats_site.create({
            data: {
                ...fixture,
                id: generatePK(), // Generate new ID to avoid conflicts
            },
        });
        stats.push(stat);
    }

    return stats;
}
