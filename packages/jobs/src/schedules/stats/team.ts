// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-04
import { type PeriodType, type Prisma } from "@prisma/client";
import { batch, DbProvider, logger } from "@vrooli/server";
import { generatePK } from "@vrooli/shared";

// Select shape for team stats batching
const teamStatsSelect = {
    id: true,
    _count: { select: { members: true, resources: true } },
} as const;
// Payload type for team stats batching
type TeamPayload = Prisma.teamGetPayload<{ select: typeof teamStatsSelect }>;

type BatchRunsResult = Record<string, {
    runsStarted: number;
    runsCompleted: number;
    runCompletionTimeAverage: number;
    runContextSwitchesAverage: number;
}>

// Define the type for run records as returned by the select clause
type RunRecord = {
    id: bigint;
    team: {
        id: bigint;
    } | null;
    completedAt: string | null;
    contextSwitches: number | null;
    startedAt: string | null;
    timeElapsed: number | null;
}

/**
 * Batch collects run routine stats for a list of teams
 * @param teamIds The IDs of the teams to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of team IDs to run routine stats
 */
async function batchRuns(
    teamIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunsResult> {
    const initialResult = Object.fromEntries(teamIds.map(id => [id, {
        runsStarted: 0,
        runsCompleted: 0,
        runCompletionTimeAverage: 0,
        runContextSwitchesAverage: 0,
    }]));
    try {
        const result = initialResult;
        
        await batch<Prisma.runFindManyArgs, RunRecord>({
            objectType: "Run",
            processBatch: async (batch: RunRecord[]) => {
                // For each run, increment the counts for the routine version
                batch.forEach(run => {
                    const teamId = run.team?.id;
                    if (!teamId) return;
                    const currResult = result[teamId.toString()];
                    if (!currResult) return;
                    // If runStarted within period, increment runsStarted
                    if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                        currResult.runsStarted += 1;
                    }
                    // If runCompleted within period, increment runsCompleted 
                    // and update averages
                    if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                        currResult.runsCompleted += 1;
                        if (run.timeElapsed !== null) currResult.runCompletionTimeAverage += run.timeElapsed;
                        if (run.contextSwitches !== null) currResult.runContextSwitchesAverage += run.contextSwitches;
                    }
                });
            },
            select: {
                id: true,
                team: {
                    select: { id: true },
                },
                completedAt: true,
                contextSwitches: true,
                startedAt: true,
                timeElapsed: true,
            },
            where: {
                team: { id: { in: teamIds.map(id => BigInt(id)) } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
        });
        
        // For the averages, divide by the number of runs completed
        Object.entries(result).forEach(([teamId, currResult]) => {
            if (!currResult || typeof teamId !== "string") return;
            if (currResult.runsCompleted > 0) {
                currResult.runCompletionTimeAverage /= currResult.runsCompleted;
                currResult.runContextSwitchesAverage /= currResult.runsCompleted;
            }
        });
        
        return result;
    } catch (error) {
        logger.error("batchRuns caught error", { error });
    }
    return initialResult;
}

/**
 * Creates periodic stats for all teams
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export async function logTeamStats(
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
): Promise<void> {
    try {
        await batch<Prisma.teamFindManyArgs, TeamPayload>({
            objectType: "Team",
            processBatch: async (batch: TeamPayload[]) => {
                const runStats = await batchRuns(batch.map(team => team.id.toString()), periodStart, periodEnd);
                await DbProvider.get().stats_team.createMany({
                    data: batch.map(team => {
                        const teamRunStats = runStats[team.id.toString()] ?? {
                            runsStarted: 0,
                            runsCompleted: 0,
                            runCompletionTimeAverage: 0,
                            runContextSwitchesAverage: 0,
                        };
                        return {
                            id: generatePK(),
                            teamId: team.id,
                            periodStart,
                            periodEnd,
                            periodType,
                            members: team._count?.members ?? 0,
                            resources: team._count?.resources ?? 0,
                            ...teamRunStats,
                        };
                    }),
                });
            },
            select: teamStatsSelect,
        });
    } catch (error) {
        logger.error("logTeamStats caught error", { error, trace: "0419", periodType, periodStart, periodEnd });
    }
}
