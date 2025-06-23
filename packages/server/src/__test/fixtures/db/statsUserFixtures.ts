import { generatePK, type StatPeriodType } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for StatsUser model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const statsUserDbIds = {
    statsUser1: generatePK(),
    statsUser2: generatePK(),
    statsUser3: generatePK(),
    user1: generatePK(),
    user2: generatePK(),
    user3: generatePK(),
};

/**
 * Minimal stats user data for database creation
 */
export const minimalStatsUserDb: Prisma.stats_userCreateInput = {
    id: statsUserDbIds.statsUser1,
    userId: statsUserDbIds.user1,
    periodStart: new Date("2024-01-01T00:00:00Z"),
    periodEnd: new Date("2024-01-31T23:59:59Z"),
    periodType: "Monthly",
    resourcesCreatedByType: JSON.stringify({ "ROUTINE": 5, "PROJECT": 2, "CODE": 3 }),
    resourcesCompletedByType: JSON.stringify({ "ROUTINE": 4, "PROJECT": 1, "CODE": 2 }),
    resourceCompletionTimeAverageByType: JSON.stringify({ "ROUTINE": 120.5, "PROJECT": 300.0, "CODE": 60.2 }),
    runsStarted: 10,
    runsCompleted: 8,
    runCompletionTimeAverage: 120.5,
    runContextSwitchesAverage: 2.3,
    teamsCreated: 1,
    user: {
        connect: { id: statsUserDbIds.user1 },
    },
};

/**
 * Complete stats user with varied metrics
 */
export const completeStatsUserDb: Prisma.stats_userCreateInput = {
    id: statsUserDbIds.statsUser2,
    userId: statsUserDbIds.user2,
    periodStart: new Date("2024-02-01T00:00:00Z"),
    periodEnd: new Date("2024-02-29T23:59:59Z"),
    periodType: "Monthly",
    resourcesCreatedByType: JSON.stringify({ 
        "ROUTINE": 25, 
        "PROJECT": 8, 
        "CODE": 15, 
        "API": 3,
        "STANDARD": 2, 
    }),
    resourcesCompletedByType: JSON.stringify({ 
        "ROUTINE": 22, 
        "PROJECT": 7, 
        "CODE": 13, 
        "API": 2,
        "STANDARD": 1, 
    }),
    resourceCompletionTimeAverageByType: JSON.stringify({ 
        "ROUTINE": 95.7, 
        "PROJECT": 250.0, 
        "CODE": 45.3,
        "API": 180.5,
        "STANDARD": 420.0, 
    }),
    runsStarted: 150,
    runsCompleted: 142,
    runCompletionTimeAverage: 95.7,
    runContextSwitchesAverage: 1.8,
    teamsCreated: 3,
    user: {
        connect: { id: statsUserDbIds.user2 },
    },
};

/**
 * High-performance user stats
 */
export const highPerformanceStatsUserDb: Prisma.stats_userCreateInput = {
    id: statsUserDbIds.statsUser3,
    userId: statsUserDbIds.user3,
    periodStart: new Date("2024-03-01T00:00:00Z"),
    periodEnd: new Date("2024-03-31T23:59:59Z"),
    periodType: "Monthly",
    resourcesCreatedByType: JSON.stringify({ 
        "ROUTINE": 50, 
        "PROJECT": 15, 
        "CODE": 30, 
        "API": 8,
        "STANDARD": 5,
        "QUIZ": 10, 
    }),
    resourcesCompletedByType: JSON.stringify({ 
        "ROUTINE": 48, 
        "PROJECT": 14, 
        "CODE": 29, 
        "API": 7,
        "STANDARD": 4,
        "QUIZ": 9, 
    }),
    resourceCompletionTimeAverageByType: JSON.stringify({ 
        "ROUTINE": 45.2, 
        "PROJECT": 180.0, 
        "CODE": 25.8,
        "API": 90.3,
        "STANDARD": 200.0,
        "QUIZ": 15.5, 
    }),
    runsStarted: 300,
    runsCompleted: 295,
    runCompletionTimeAverage: 45.2,
    runContextSwitchesAverage: 0.9,
    teamsCreated: 5,
    user: {
        connect: { id: statsUserDbIds.user3 },
    },
};

/**
 * Factory for creating stats user database fixtures with overrides
 */
export class StatsUserDbFactory {
    static createMinimal(overrides?: Partial<Prisma.stats_userCreateInput>): Prisma.stats_userCreateInput {
        return {
            ...minimalStatsUserDb,
            id: generatePK(),
            userId: generatePK(),
            user: {
                connect: { id: overrides?.userId || generatePK() },
            },
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.stats_userCreateInput>): Prisma.stats_userCreateInput {
        return {
            ...completeStatsUserDb,
            id: generatePK(),
            userId: generatePK(),
            user: {
                connect: { id: overrides?.userId || generatePK() },
            },
            ...overrides,
        };
    }

    static createHighPerformance(overrides?: Partial<Prisma.stats_userCreateInput>): Prisma.stats_userCreateInput {
        return {
            ...highPerformanceStatsUserDb,
            id: generatePK(),
            userId: generatePK(),
            user: {
                connect: { id: overrides?.userId || generatePK() },
            },
            ...overrides,
        };
    }

    /**
     * Create stats user for specific period type
     */
    static createForPeriod(
        userId: string,
        periodType: StatPeriodType,
        periodStart: Date,
        periodEnd: Date,
        overrides?: Partial<Prisma.stats_userCreateInput>,
    ): Prisma.stats_userCreateInput {
        return {
            ...minimalStatsUserDb,
            id: generatePK(),
            userId,
            periodType,
            periodStart,
            periodEnd,
            user: {
                connect: { id: userId },
            },
            ...overrides,
        };
    }

    /**
     * Create stats user with specific metrics
     */
    static createWithMetrics(
        userId: string,
        metrics: {
            resourcesCreatedByType?: Record<string, number>;
            resourcesCompletedByType?: Record<string, number>;
            resourceCompletionTimeAverageByType?: Record<string, number>;
            runsStarted?: number;
            runsCompleted?: number;
            runCompletionTimeAverage?: number;
            runContextSwitchesAverage?: number;
            teamsCreated?: number;
        },
        overrides?: Partial<Prisma.stats_userCreateInput>,
    ): Prisma.stats_userCreateInput {
        return {
            ...minimalStatsUserDb,
            id: generatePK(),
            userId,
            resourcesCreatedByType: JSON.stringify(metrics.resourcesCreatedByType ?? { "ROUTINE": 5, "PROJECT": 2 }),
            resourcesCompletedByType: JSON.stringify(metrics.resourcesCompletedByType ?? { "ROUTINE": 4, "PROJECT": 1 }),
            resourceCompletionTimeAverageByType: JSON.stringify(metrics.resourceCompletionTimeAverageByType ?? { "ROUTINE": 120.5, "PROJECT": 300.0 }),
            runsStarted: metrics.runsStarted ?? 10,
            runsCompleted: metrics.runsCompleted ?? 8,
            runCompletionTimeAverage: metrics.runCompletionTimeAverage ?? 120.5,
            runContextSwitchesAverage: metrics.runContextSwitchesAverage ?? 2.3,
            teamsCreated: metrics.teamsCreated ?? 1,
            user: {
                connect: { id: userId },
            },
            ...overrides,
        };
    }

    /**
     * Create stats user with zero metrics (new user)
     */
    static createEmpty(userId: string, overrides?: Partial<Prisma.stats_userCreateInput>): Prisma.stats_userCreateInput {
        return {
            ...minimalStatsUserDb,
            id: generatePK(),
            userId,
            resourcesCreatedByType: JSON.stringify({}),
            resourcesCompletedByType: JSON.stringify({}),
            resourceCompletionTimeAverageByType: JSON.stringify({}),
            runsStarted: 0,
            runsCompleted: 0,
            runCompletionTimeAverage: 0,
            runContextSwitchesAverage: 0,
            teamsCreated: 0,
            user: {
                connect: { id: userId },
            },
            ...overrides,
        };
    }

    /**
     * Create stats user for beginner (just started)
     */
    static createBeginner(userId: string, overrides?: Partial<Prisma.stats_userCreateInput>): Prisma.stats_userCreateInput {
        return {
            ...minimalStatsUserDb,
            id: generatePK(),
            userId,
            resourcesCreatedByType: JSON.stringify({ "ROUTINE": 1, "PROJECT": 1 }),
            resourcesCompletedByType: JSON.stringify({ "ROUTINE": 0, "PROJECT": 0 }),
            resourceCompletionTimeAverageByType: JSON.stringify({}),
            runsStarted: 2,
            runsCompleted: 1,
            runCompletionTimeAverage: 300.0,
            runContextSwitchesAverage: 5.0,
            teamsCreated: 0,
            user: {
                connect: { id: userId },
            },
            ...overrides,
        };
    }
}

/**
 * Helper to create stats user time series data
 */
export function createStatsUserTimeSeries(
    userId: string,
    periodType: StatPeriodType,
    count: number,
    startDate: Date = new Date("2024-01-01T00:00:00Z"),
): Prisma.stats_userCreateInput[] {
    const stats = [];
    const msPerPeriod = getPeriodMs(periodType);

    for (let i = 0; i < count; i++) {
        const periodStart = new Date(startDate.getTime() + (i * msPerPeriod));
        const periodEnd = new Date(periodStart.getTime() + msPerPeriod - 1);

        // Generate realistic growth metrics
        const resourcesCreated = {
            "ROUTINE": Math.floor(5 + (i * 2)),
            "PROJECT": Math.floor(2 + (i * 1)),
            "CODE": Math.floor(3 + (i * 1.5)),
        };
        
        const resourcesCompleted = {
            "ROUTINE": Math.floor(resourcesCreated.ROUTINE * 0.8),
            "PROJECT": Math.floor(resourcesCreated.PROJECT * 0.6),
            "CODE": Math.floor(resourcesCreated.CODE * 0.7),
        };

        const completionTimes = {
            "ROUTINE": Math.max(30, 120 - (i * 5)),
            "PROJECT": Math.max(120, 300 - (i * 10)),
            "CODE": Math.max(20, 60 - (i * 2)),
        };

        const baseMetrics = {
            resourcesCreatedByType: resourcesCreated,
            resourcesCompletedByType: resourcesCompleted,
            resourceCompletionTimeAverageByType: completionTimes,
            runsStarted: Math.floor(10 + (i * 5)),
            runsCompleted: Math.floor(8 + (i * 4)),
            runCompletionTimeAverage: Math.max(30, 120 - (i * 5)),
            runContextSwitchesAverage: Math.max(0.5, 2.5 - (i * 0.1)),
            teamsCreated: Math.floor(i * 0.3),
        };

        stats.push(
            StatsUserDbFactory.createForPeriod(
                userId,
                periodType,
                periodStart,
                periodEnd,
                baseMetrics,
            ),
        );
    }

    return stats;
}

/**
 * Helper to seed multiple stats users for testing
 */
export async function seedStatsUsers(
    prisma: any,
    userIds: string[],
    options?: {
        periodType?: StatPeriodType;
        periodsPerUser?: number;
        withTimeSeries?: boolean;
    },
) {
    const statsUsers = [];
    const periodType = options?.periodType || "Monthly";
    const periodsPerUser = options?.periodsPerUser || 3;

    for (const userId of userIds) {
        if (options?.withTimeSeries) {
            const timeSeries = createStatsUserTimeSeries(userId, periodType, periodsPerUser);
            for (const stats of timeSeries) {
                const created = await prisma.stats_user.create({ data: stats });
                statsUsers.push(created);
            }
        } else {
            const stats = StatsUserDbFactory.createMinimal({ userId });
            const created = await prisma.stats_user.create({ data: stats });
            statsUsers.push(created);
        }
    }

    return statsUsers;
}

/**
 * Helper to get milliseconds per period type
 */
function getPeriodMs(periodType: StatPeriodType): number {
    switch (periodType) {
        case "Hourly":
            return 60 * 60 * 1000;
        case "Daily":
            return 24 * 60 * 60 * 1000;
        case "Weekly":
            return 7 * 24 * 60 * 60 * 1000;
        case "Monthly":
            return 30 * 24 * 60 * 60 * 1000; // Approximate
        case "Yearly":
            return 365 * 24 * 60 * 60 * 1000; // Approximate
        default:
            return 24 * 60 * 60 * 1000;
    }
}
