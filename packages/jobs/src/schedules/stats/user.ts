import { batch, batchGroup, DbProvider, logger } from "@local/server";
import { generatePK } from "@local/shared";
import { type PeriodType, type Prisma } from "@prisma/client";

// Select shape for user stats batching
const userStatsSelect = { id: true } as const;
// Payload type for user stats batching
type UserPayload = Prisma.userGetPayload<{ select: typeof userStatsSelect }>;

type BatchTeamsResult = Record<string, {
    teamsCreated: number;
}>

type BatchResourcesResult = Record<string, {
    resourcesCreated: number;
    resourcesCompleted: number;
    resourceCompletionTimeAverage: number;
}>

type BatchRunsResult = Record<string, {
    runsStarted: number;
    runsCompleted: number;
    runCompletionTimeAverage: number;
    runContextSwitchesAverage: number;
}>


/**
 * Batch collects team stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to team stats
 */
async function batchTeams(
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchTeamsResult> {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        teamsCreated: 0,
    }]));
    try {
        return await batchGroup<Prisma.teamFindManyArgs, BatchTeamsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(team => {
                    const userId = team.createdById;
                    if (!userId) return;
                    const currResult = result[userId.toString()];
                    if (!currResult) return;
                    currResult.teamsCreated += 1;
                });
            },
            objectType: "Team",
            select: {
                id: true,
                createdById: true,
            },
            where: {
                createdAt: { gte: periodStart, lt: periodEnd },
                createdById: { in: userIds.map(id => BigInt(id)) },
            },
        });
    } catch (error) {
        logger.error("batchTeams caught error", { error });
    }
    return initialResult;
}

/**
 * Batch collects resource stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to resource stats
 */
async function batchResources(
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchResourcesResult> {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        resourcesCreated: 0,
        resourcesCompleted: 0,
        resourceCompletionTimeAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.resourceFindManyArgs, BatchResourcesResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(resource => {
                    const userId = resource.createdById;
                    if (!userId) return;
                    const currResult = result[userId.toString()];
                    if (!currResult) return;
                    currResult.resourcesCreated += 1;
                    if (resource.hasCompleteVersion) {
                        currResult.resourcesCompleted += 1;
                        if (resource.completedAt) currResult.resourceCompletionTimeAverage += (new Date(resource.completedAt).getTime() - new Date(resource.createdAt).getTime());
                    }
                });
            },
            finalizeResult: (result) => {
                // Calculate averages
                Object.keys(result).forEach(userId => {
                    const currResult = result[userId];
                    if (!currResult) return;
                    if (currResult.resourcesCompleted > 0) {
                        currResult.resourceCompletionTimeAverage /= currResult.resourcesCompleted;
                    }
                });
                return result;
            },
            objectType: "Resource",
            select: {
                id: true,
                completedAt: true,
                createdAt: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            where: {
                createdById: { in: userIds.map(id => BigInt(id)) },
                isDeleted: false,
                OR: [
                    { createdAt: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchResources caught error", { error });
    }
    return initialResult;
}

/**
 * Batch collects run stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to run stats
 */
async function batchRuns(
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunsResult> {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
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
                    const userId = run.user?.id;
                    if (!userId) return;
                    const currResult = result[userId.toString()];
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
                Object.keys(result).forEach(userId => {
                    const currResult = result[userId];
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
                user: {
                    select: { id: true },
                },
                completedAt: true,
                contextSwitches: true,
                startedAt: true,
                timeElapsed: true,
            },
            where: {
                user: { id: { in: userIds.map(id => BigInt(id)) } },
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
 * Creates periodic stats for all users
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export async function logUserStats(
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
): Promise<void> {
    try {
        await batch<Prisma.userFindManyArgs, UserPayload>({
            objectType: "User",
            processBatch: async (batch) => {
                // Get user ids, so we can query various tables for stats
                const userIds = batch.map(user => user.id.toString());
                // Batch collect stats
                const resourceStats = await batchResources(userIds, periodStart, periodEnd);
                const runStats = await batchRuns(userIds, periodStart, periodEnd);
                const teamStats = await batchTeams(userIds, periodStart, periodEnd);
                // Create stats for each user
                await DbProvider.get().stats_user.createMany({
                    data: batch.map(user => ({
                        id: generatePK(),
                        userId: user.id,
                        periodStart,
                        periodEnd,
                        periodType,
                        ...(resourceStats[user.id.toString()] || { resourceCompletionTimeAverage: 0, resourcesCompleted: 0, resourcesCreated: 0 }),
                        ...(runStats[user.id.toString()] || { runCompletionTimeAverage: 0, runContextSwitchesAverage: 0, runsCompleted: 0, runsStarted: 0 }),
                        ...(teamStats[user.id.toString()] || { teamsCreated: 0 }),
                    })),
                });
            },
            select: userStatsSelect,
            where: {
                updatedAt: {
                    gte: new Date(new Date().getTime() - 1000 * 60 * 60 * 24 * 90).toISOString(),
                },
            },
        });
    } catch (error) {
        logger.error("logUserStats caught error", { error, trace: "0426", periodType, periodStart, periodEnd });
    }
}
