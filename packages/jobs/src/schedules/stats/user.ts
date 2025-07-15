// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-04
import { type PeriodType, type Prisma } from "@prisma/client";
import { batch, batchGroup, DbProvider, logger } from "@vrooli/server";
import { generatePK } from "@vrooli/shared";

// Time constants
const MILLISECONDS_PER_SECOND = 1000;
const SECONDS_PER_MINUTE = 60;
const MINUTES_PER_HOUR = 60;
const HOURS_PER_DAY = 24;
const DAYS_FOR_ACTIVITY_WINDOW = 90; // 90 days for recent activity
const ACTIVITY_WINDOW_MS = MILLISECONDS_PER_SECOND * SECONDS_PER_MINUTE * MINUTES_PER_HOUR * HOURS_PER_DAY * DAYS_FOR_ACTIVITY_WINDOW;

// Select shape for user stats batching
const userStatsSelect = { id: true } as const;
// Payload type for user stats batching
type UserPayload = Prisma.userGetPayload<{ select: typeof userStatsSelect }>;

type BatchTeamsResult = Record<string, {
    teamsCreated: number;
}>

type BatchResourcesResult = Record<string, {
    resourcesCreatedByType: Record<string, number>;
    resourcesCompletedByType: Record<string, number>;
    resourceCompletionTimeAverageByType: Record<string, number>;
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
            processBatch: async (batch: BatchTeamsResult[], result: BatchTeamsResult) => {
                // Cast the batch to the correct type since the batchGroup function has a type issue
                const teamBatch = batch as unknown as Prisma.teamGetPayload<{ select: { id: true; createdById: true } }>[];
                // For each, add stats to the user
                teamBatch.forEach(team => {
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
        resourcesCreatedByType: {} as Record<string, number>,
        resourcesCompletedByType: {} as Record<string, number>,
        resourceCompletionTimeAverageByType: {} as Record<string, number>,
    }]));
    try {
        // Define the resource record type based on the select clause
        type ResourceRecord = {
            id: bigint;
            completedAt: string | null;
            createdAt: string;
            createdById: bigint;
            hasCompleteVersion: boolean;
            resourceType: string;
        };
        
        const result = initialResult;
        
        await batch<Prisma.resourceFindManyArgs, ResourceRecord>({
            objectType: "Resource",
            processBatch: async (batch: ResourceRecord[]) => {
                // For each, add stats to the user by resource type
                batch.forEach(resource => {
                    const userId = resource.createdById;
                    if (!userId) return;
                    const currResult = result[userId.toString()];
                    if (!currResult) return;
                    
                    const resourceType = resource.resourceType;
                    
                    // Count created resources by type
                    if (!currResult.resourcesCreatedByType[resourceType]) {
                        currResult.resourcesCreatedByType[resourceType] = 0;
                    }
                    currResult.resourcesCreatedByType[resourceType] += 1;
                    
                    // Count completed resources by type
                    if (resource.hasCompleteVersion) {
                        if (!currResult.resourcesCompletedByType[resourceType]) {
                            currResult.resourcesCompletedByType[resourceType] = 0;
                        }
                        currResult.resourcesCompletedByType[resourceType] += 1;
                        
                        // Track completion time for averaging
                        if (resource.completedAt) {
                            if (!currResult.resourceCompletionTimeAverageByType[resourceType]) {
                                currResult.resourceCompletionTimeAverageByType[resourceType] = 0;
                            }
                            currResult.resourceCompletionTimeAverageByType[resourceType] += (new Date(resource.completedAt).getTime() - new Date(resource.createdAt).getTime());
                        }
                    }
                });
            },
            select: {
                id: true,
                completedAt: true,
                createdAt: true,
                createdById: true,
                hasCompleteVersion: true,
                resourceType: true,
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
        
        // Calculate averages by type
        Object.entries(result).forEach(([userId, currResult]) => {
            if (!currResult || typeof userId !== "string") return;
            
            Object.entries(currResult.resourceCompletionTimeAverageByType).forEach(([resourceType, totalTime]) => {
                const completedCount = currResult.resourcesCompletedByType[resourceType];
                if (completedCount && completedCount > 0 && typeof totalTime === "number") {
                    currResult.resourceCompletionTimeAverageByType[resourceType] = totalTime / completedCount;
                }
            });
        });
        
        return result;
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
        // Define the run record type based on the select clause
        type RunRecord = {
            id: bigint;
            user: {
                id: bigint;
            } | null;
            completedAt: string | null;
            contextSwitches: number | null;
            startedAt: string | null;
            timeElapsed: number | null;
        };
        
        const result = initialResult;
        
        await batch<Prisma.runFindManyArgs, RunRecord>({
            objectType: "Run",
            processBatch: async (batch: RunRecord[]) => {
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
                        if (run.contextSwitches !== null) currResult.runContextSwitchesAverage += run.contextSwitches;
                    }
                });
            },
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
        
        // For the averages, divide by the number of runs completed
        Object.entries(result).forEach(([userId, currResult]) => {
            if (!currResult || typeof userId !== "string") return;
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
            processBatch: async (batch: UserPayload[]) => {
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
                        periodStart: new Date(periodStart),
                        periodEnd: new Date(periodEnd),
                        periodType,
                        ...(resourceStats[user.id.toString()] || {}),
                        ...(runStats[user.id.toString()] || { runCompletionTimeAverage: 0, runContextSwitchesAverage: 0, runsCompleted: 0, runsStarted: 0 }),
                        ...(teamStats[user.id.toString()] || { teamsCreated: 0 }),
                    })),
                });
            },
            select: userStatsSelect,
            where: {
                updatedAt: {
                    gte: new Date(new Date().getTime() - ACTIVITY_WINDOW_MS).toISOString(),
                },
            },
        });
    } catch (error) {
        logger.error("logUserStats caught error", { error, trace: "0426", periodType, periodStart, periodEnd });
    }
}
