import { generatePK } from "@vrooli/shared";
import { type Prisma, PeriodType } from "@prisma/client";

/**
 * Database fixtures for StatsResource model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const statsResourceDbIds = {
    stats1: generatePK(),
    stats2: generatePK(),
    stats3: generatePK(),
    stats4: generatePK(),
    stats5: generatePK(),
    resource1: generatePK(),
    resource2: generatePK(),
    resource3: generatePK(),
};

/**
 * Minimal stats resource data for database creation
 */
export const minimalStatsResourceDb: Prisma.stats_resourceCreateInput = {
    id: statsResourceDbIds.stats1,
    resourceId: statsResourceDbIds.resource1,
    periodStart: new Date("2024-01-01T00:00:00Z"),
    periodEnd: new Date("2024-01-31T23:59:59Z"),
    periodType: PeriodType.Monthly,
    references: 0,
    referencedBy: 0,
    runsStarted: 0,
    runsCompleted: 0,
    runCompletionTimeAverage: 0.0,
    runContextSwitchesAverage: 0.0,
};

/**
 * Complete stats resource with realistic usage data
 */
export const completeStatsResourceDb: Prisma.stats_resourceCreateInput = {
    id: statsResourceDbIds.stats2,
    resourceId: statsResourceDbIds.resource2,
    periodStart: new Date("2024-02-01T00:00:00Z"),
    periodEnd: new Date("2024-02-29T23:59:59Z"),
    periodType: PeriodType.Monthly,
    references: 15,
    referencedBy: 8,
    runsStarted: 42,
    runsCompleted: 38,
    runCompletionTimeAverage: 125.5,
    runContextSwitchesAverage: 2.3,
};

/**
 * Daily stats resource
 */
export const dailyStatsResourceDb: Prisma.stats_resourceCreateInput = {
    id: statsResourceDbIds.stats3,
    resourceId: statsResourceDbIds.resource1,
    periodStart: new Date("2024-03-15T00:00:00Z"),
    periodEnd: new Date("2024-03-15T23:59:59Z"),
    periodType: PeriodType.Daily,
    references: 2,
    referencedBy: 1,
    runsStarted: 8,
    runsCompleted: 7,
    runCompletionTimeAverage: 89.2,
    runContextSwitchesAverage: 1.8,
};

/**
 * Weekly stats resource
 */
export const weeklyStatsResourceDb: Prisma.stats_resourceCreateInput = {
    id: statsResourceDbIds.stats4,
    resourceId: statsResourceDbIds.resource2,
    periodStart: new Date("2024-03-01T00:00:00Z"),
    periodEnd: new Date("2024-03-07T23:59:59Z"),
    periodType: PeriodType.Weekly,
    references: 5,
    referencedBy: 3,
    runsStarted: 18,
    runsCompleted: 16,
    runCompletionTimeAverage: 67.8,
    runContextSwitchesAverage: 1.5,
};

/**
 * Yearly stats resource
 */
export const yearlyStatsResourceDb: Prisma.stats_resourceCreateInput = {
    id: statsResourceDbIds.stats5,
    resourceId: statsResourceDbIds.resource3,
    periodStart: new Date("2024-01-01T00:00:00Z"),
    periodEnd: new Date("2024-12-31T23:59:59Z"),
    periodType: PeriodType.Yearly,
    references: 156,
    referencedBy: 89,
    runsStarted: 520,
    runsCompleted: 487,
    runCompletionTimeAverage: 203.7,
    runContextSwitchesAverage: 3.2,
};

/**
 * Factory for creating stats resource database fixtures with overrides
 */
export class StatsResourceDbFactory {
    static createMinimal(
        resourceId: string,
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        return {
            ...minimalStatsResourceDb,
            id: generatePK(),
            resourceId: resourceId,
            ...overrides,
        };
    }

    static createComplete(
        resourceId: string,
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        return {
            ...completeStatsResourceDb,
            id: generatePK(),
            resourceId: resourceId,
            ...overrides,
        };
    }

    static createDaily(
        resourceId: string,
        date: Date = new Date("2024-03-15"),
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        const periodStart = new Date(date);
        periodStart.setHours(0, 0, 0, 0);
        const periodEnd = new Date(date);
        periodEnd.setHours(23, 59, 59, 999);

        return {
            ...dailyStatsResourceDb,
            id: generatePK(),
            resourceId: resourceId,
            periodStart,
            periodEnd,
            periodType: PeriodType.Daily,
            ...overrides,
        };
    }

    static createWeekly(
        resourceId: string,
        weekStart: Date = new Date("2024-03-01"),
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        const periodStart = new Date(weekStart);
        periodStart.setHours(0, 0, 0, 0);
        const periodEnd = new Date(weekStart);
        periodEnd.setDate(periodEnd.getDate() + 6);
        periodEnd.setHours(23, 59, 59, 999);

        return {
            ...weeklyStatsResourceDb,
            id: generatePK(),
            resourceId: resourceId,
            periodStart,
            periodEnd,
            periodType: PeriodType.Weekly,
            ...overrides,
        };
    }

    static createMonthly(
        resourceId: string,
        year: number = 2024,
        month: number = 2, // 1-based month
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        const periodStart = new Date(year, month - 1, 1); // month is 0-based in Date constructor
        const periodEnd = new Date(year, month, 0, 23, 59, 59, 999); // Last day of month

        return {
            ...completeStatsResourceDb,
            id: generatePK(),
            resourceId: resourceId,
            periodStart,
            periodEnd,
            periodType: PeriodType.Monthly,
            ...overrides,
        };
    }

    static createYearly(
        resourceId: string,
        year: number = 2024,
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        const periodStart = new Date(year, 0, 1); // January 1st
        const periodEnd = new Date(year, 11, 31, 23, 59, 59, 999); // December 31st

        return {
            ...yearlyStatsResourceDb,
            id: generatePK(),
            resourceId: resourceId,
            periodStart,
            periodEnd,
            periodType: PeriodType.Yearly,
            ...overrides,
        };
    }

    /**
     * Create high-performance stats for popular resources
     */
    static createHighPerformance(
        resourceId: string,
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        return {
            ...this.createComplete(resourceId),
            references: 250,
            referencedBy: 120,
            runsStarted: 1000,
            runsCompleted: 950,
            runCompletionTimeAverage: 45.2,
            runContextSwitchesAverage: 0.8,
            ...overrides,
        };
    }

    /**
     * Create low-performance stats for problematic resources
     */
    static createLowPerformance(
        resourceId: string,
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        return {
            ...this.createComplete(resourceId),
            references: 3,
            referencedBy: 1,
            runsStarted: 20,
            runsCompleted: 8,
            runCompletionTimeAverage: 850.7,
            runContextSwitchesAverage: 12.5,
            ...overrides,
        };
    }

    /**
     * Create stats with no activity (new or unused resource)
     */
    static createNoActivity(
        resourceId: string,
        overrides?: Partial<Prisma.stats_resourceCreateInput>
    ): Prisma.stats_resourceCreateInput {
        return {
            ...this.createMinimal(resourceId),
            references: 0,
            referencedBy: 0,
            runsStarted: 0,
            runsCompleted: 0,
            runCompletionTimeAverage: 0.0,
            runContextSwitchesAverage: 0.0,
            ...overrides,
        };
    }
}

/**
 * Helper to seed multiple stats records for a resource across time periods
 */
export async function seedResourceStatsHistory(
    prisma: any,
    resourceId: string,
    options?: {
        startDate?: Date;
        endDate?: Date;
        periodType?: PeriodType;
        includeNoActivity?: boolean;
        basePerformance?: "high" | "low" | "normal";
    }
) {
    const {
        startDate = new Date("2024-01-01"),
        endDate = new Date("2024-03-31"),
        periodType = PeriodType.Monthly,
        includeNoActivity = false,
        basePerformance = "normal",
    } = options || {};

    const statsRecords = [];
    const current = new Date(startDate);

    while (current <= endDate) {
        let statsData;

        if (includeNoActivity && Math.random() < 0.2) {
            // 20% chance of no activity period
            statsData = StatsResourceDbFactory.createNoActivity(resourceId, {
                periodStart: new Date(current),
                periodEnd: getNextPeriodEnd(current, periodType),
                periodType,
            });
        } else {
            // Create stats based on performance level
            switch (basePerformance) {
                case "high":
                    statsData = StatsResourceDbFactory.createHighPerformance(resourceId, {
                        periodStart: new Date(current),
                        periodEnd: getNextPeriodEnd(current, periodType),
                        periodType,
                    });
                    break;
                case "low":
                    statsData = StatsResourceDbFactory.createLowPerformance(resourceId, {
                        periodStart: new Date(current),
                        periodEnd: getNextPeriodEnd(current, periodType),
                        periodType,
                    });
                    break;
                default:
                    statsData = StatsResourceDbFactory.createComplete(resourceId, {
                        periodStart: new Date(current),
                        periodEnd: getNextPeriodEnd(current, periodType),
                        periodType,
                    });
            }
        }

        const record = await prisma.stats_resource.create({ data: statsData });
        statsRecords.push(record);

        // Move to next period
        advanceToNextPeriod(current, periodType);
    }

    return statsRecords;
}

/**
 * Helper to get the end date for a period
 */
function getNextPeriodEnd(start: Date, periodType: PeriodType): Date {
    const end = new Date(start);
    
    switch (periodType) {
        case PeriodType.Daily:
            end.setHours(23, 59, 59, 999);
            break;
        case PeriodType.Weekly:
            end.setDate(end.getDate() + 6);
            end.setHours(23, 59, 59, 999);
            break;
        case PeriodType.Monthly:
            end.setMonth(end.getMonth() + 1);
            end.setDate(0); // Last day of previous month
            end.setHours(23, 59, 59, 999);
            break;
        case PeriodType.Yearly:
            end.setFullYear(end.getFullYear() + 1);
            end.setMonth(0, 0); // December 31st of current year
            end.setHours(23, 59, 59, 999);
            break;
    }
    
    return end;
}

/**
 * Helper to advance date to next period start
 */
function advanceToNextPeriod(date: Date, periodType: PeriodType): void {
    switch (periodType) {
        case PeriodType.Daily:
            date.setDate(date.getDate() + 1);
            break;
        case PeriodType.Weekly:
            date.setDate(date.getDate() + 7);
            break;
        case PeriodType.Monthly:
            date.setMonth(date.getMonth() + 1);
            break;
        case PeriodType.Yearly:
            date.setFullYear(date.getFullYear() + 1);
            break;
    }
    date.setHours(0, 0, 0, 0);
}