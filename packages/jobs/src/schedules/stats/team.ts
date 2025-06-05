import { type PeriodType, type Prisma } from "@prisma/client";
import { batch, batchGroup, DbProvider, logger } from "@vrooli/server";
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
        return await batchGroup<Prisma.runFindManyArgs, BatchRunsResult>({
            initialResult,
            processBatch: async (batch, result) => {
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
                        currResult.runContextSwitchesAverage += run.contextSwitches;
                    }
                });
            },
            finalizeResult: (result) => {
                // For the averages, divide by the number of runs completed
                Object.keys(result).forEach(teamId => {
                    const currResult = result[teamId];
                    if (!currResult) return;
                    if (currResult.runsCompleted > 0) {
                        currResult.runCompletionTimeAverage /= currResult.runsCompleted;
                        currResult.runContextSwitchesAverage /= currResult.runsCompleted;
                    }
                });
                return result;
            },
            objectType: "Run",
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
            processBatch: async (batch) => {
                const runStats = await batchRuns(batch.map(team => team.id.toString()), periodStart, periodEnd);
                await DbProvider.get().stats_team.createMany({
                    data: batch.map(team => ({
                        id: generatePK(),
                        teamId: team.id,
                        periodStart,
                        periodEnd,
                        periodType,
                        members: team._count?.members ?? 0,
                        resources: team._count?.resources ?? 0,
                        ...runStats[team.id.toString()],
                    })),
                });
            },
            select: teamStatsSelect,
        });
    } catch (error) {
        logger.error("logTeamStats caught error", { error, trace: "0419", periodType, periodStart, periodEnd });
    }
}
