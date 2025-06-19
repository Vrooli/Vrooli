import { generatePK, StatPeriodType } from "@vrooli/shared";
import { type Prisma } from "@prisma/client";

/**
 * Database fixtures for StatsTeam model - used for seeding test data
 * These follow Prisma's shape for database operations
 */

// Consistent IDs for testing
export const statsTeamDbIds = {
    statsTeam1: generatePK(),
    statsTeam2: generatePK(),
    statsTeam3: generatePK(),
    team1: generatePK(),
    team2: generatePK(),
    team3: generatePK(),
};

/**
 * Minimal stats team data for database creation
 */
export const minimalStatsTeamDb: Prisma.stats_teamCreateInput = {
    id: statsTeamDbIds.statsTeam1,
    teamId: statsTeamDbIds.team1,
    periodStart: new Date("2024-01-01T00:00:00Z"),
    periodEnd: new Date("2024-01-31T23:59:59Z"),
    periodType: "Monthly",
    resources: 5,
    members: 3,
    runsStarted: 10,
    runsCompleted: 8,
    runCompletionTimeAverage: 120.5,
    runContextSwitchesAverage: 2.3,
    team: {
        connect: { id: statsTeamDbIds.team1 },
    },
};

/**
 * Complete stats team with varied metrics
 */
export const completeStatsTeamDb: Prisma.stats_teamCreateInput = {
    id: statsTeamDbIds.statsTeam2,
    teamId: statsTeamDbIds.team2,
    periodStart: new Date("2024-02-01T00:00:00Z"),
    periodEnd: new Date("2024-02-29T23:59:59Z"),
    periodType: "Monthly",
    resources: 25,
    members: 8,
    runsStarted: 150,
    runsCompleted: 142,
    runCompletionTimeAverage: 95.7,
    runContextSwitchesAverage: 1.8,
    team: {
        connect: { id: statsTeamDbIds.team2 },
    },
};

/**
 * High-performance team stats
 */
export const highPerformanceStatsTeamDb: Prisma.stats_teamCreateInput = {
    id: statsTeamDbIds.statsTeam3,
    teamId: statsTeamDbIds.team3,
    periodStart: new Date("2024-03-01T00:00:00Z"),
    periodEnd: new Date("2024-03-31T23:59:59Z"),
    periodType: "Monthly",
    resources: 50,
    members: 12,
    runsStarted: 300,
    runsCompleted: 295,
    runCompletionTimeAverage: 45.2,
    runContextSwitchesAverage: 0.9,
    team: {
        connect: { id: statsTeamDbIds.team3 },
    },
};

/**
 * Factory for creating stats team database fixtures with overrides
 */
export class StatsTeamDbFactory {
    static createMinimal(overrides?: Partial<Prisma.stats_teamCreateInput>): Prisma.stats_teamCreateInput {
        return {
            ...minimalStatsTeamDb,
            id: generatePK(),
            teamId: generatePK(),
            team: {
                connect: { id: overrides?.teamId || generatePK() },
            },
            ...overrides,
        };
    }

    static createComplete(overrides?: Partial<Prisma.stats_teamCreateInput>): Prisma.stats_teamCreateInput {
        return {
            ...completeStatsTeamDb,
            id: generatePK(),
            teamId: generatePK(),
            team: {
                connect: { id: overrides?.teamId || generatePK() },
            },
            ...overrides,
        };
    }

    static createHighPerformance(overrides?: Partial<Prisma.stats_teamCreateInput>): Prisma.stats_teamCreateInput {
        return {
            ...highPerformanceStatsTeamDb,
            id: generatePK(),
            teamId: generatePK(),
            team: {
                connect: { id: overrides?.teamId || generatePK() },
            },
            ...overrides,
        };
    }

    /**
     * Create stats team for specific period type
     */
    static createForPeriod(
        teamId: string,
        periodType: StatPeriodType,
        periodStart: Date,
        periodEnd: Date,
        overrides?: Partial<Prisma.stats_teamCreateInput>
    ): Prisma.stats_teamCreateInput {
        return {
            ...minimalStatsTeamDb,
            id: generatePK(),
            teamId,
            periodType,
            periodStart,
            periodEnd,
            team: {
                connect: { id: teamId },
            },
            ...overrides,
        };
    }

    /**
     * Create stats team with specific metrics
     */
    static createWithMetrics(
        teamId: string,
        metrics: {
            resources?: number;
            members?: number;
            runsStarted?: number;
            runsCompleted?: number;
            runCompletionTimeAverage?: number;
            runContextSwitchesAverage?: number;
        },
        overrides?: Partial<Prisma.stats_teamCreateInput>
    ): Prisma.stats_teamCreateInput {
        return {
            ...minimalStatsTeamDb,
            id: generatePK(),
            teamId,
            resources: metrics.resources ?? 5,
            members: metrics.members ?? 3,
            runsStarted: metrics.runsStarted ?? 10,
            runsCompleted: metrics.runsCompleted ?? 8,
            runCompletionTimeAverage: metrics.runCompletionTimeAverage ?? 120.5,
            runContextSwitchesAverage: metrics.runContextSwitchesAverage ?? 2.3,
            team: {
                connect: { id: teamId },
            },
            ...overrides,
        };
    }

    /**
     * Create stats team with zero metrics (new team)
     */
    static createEmpty(teamId: string, overrides?: Partial<Prisma.stats_teamCreateInput>): Prisma.stats_teamCreateInput {
        return {
            ...minimalStatsTeamDb,
            id: generatePK(),
            teamId,
            resources: 0,
            members: 1,
            runsStarted: 0,
            runsCompleted: 0,
            runCompletionTimeAverage: 0,
            runContextSwitchesAverage: 0,
            team: {
                connect: { id: teamId },
            },
            ...overrides,
        };
    }
}

/**
 * Helper to create stats team time series data
 */
export function createStatsTeamTimeSeries(
    teamId: string,
    periodType: StatPeriodType,
    count: number,
    startDate: Date = new Date("2024-01-01T00:00:00Z")
): Prisma.stats_teamCreateInput[] {
    const stats = [];
    const msPerPeriod = getPeriodMs(periodType);

    for (let i = 0; i < count; i++) {
        const periodStart = new Date(startDate.getTime() + (i * msPerPeriod));
        const periodEnd = new Date(periodStart.getTime() + msPerPeriod - 1);

        // Generate realistic growth metrics
        const baseMetrics = {
            resources: Math.floor(5 + (i * 2)),
            members: Math.floor(3 + (i * 0.5)),
            runsStarted: Math.floor(10 + (i * 5)),
            runsCompleted: Math.floor(8 + (i * 4)),
            runCompletionTimeAverage: Math.max(30, 120 - (i * 5)),
            runContextSwitchesAverage: Math.max(0.5, 2.5 - (i * 0.1)),
        };

        stats.push(
            StatsTeamDbFactory.createForPeriod(
                teamId,
                periodType,
                periodStart,
                periodEnd,
                baseMetrics
            )
        );
    }

    return stats;
}

/**
 * Helper to seed multiple stats teams for testing
 */
export async function seedStatsTeams(
    prisma: any,
    teamIds: string[],
    options?: {
        periodType?: StatPeriodType;
        periodsPerTeam?: number;
        withTimeSeries?: boolean;
    }
) {
    const statsTeams = [];
    const periodType = options?.periodType || "Monthly";
    const periodsPerTeam = options?.periodsPerTeam || 3;

    for (const teamId of teamIds) {
        if (options?.withTimeSeries) {
            const timeSeries = createStatsTeamTimeSeries(teamId, periodType, periodsPerTeam);
            for (const stats of timeSeries) {
                const created = await prisma.stats_team.create({ data: stats });
                statsTeams.push(created);
            }
        } else {
            const stats = StatsTeamDbFactory.createMinimal({ teamId });
            const created = await prisma.stats_team.create({ data: stats });
            statsTeams.push(created);
        }
    }

    return statsTeams;
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