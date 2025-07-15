// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-04
import { type PeriodType, type Prisma } from "@prisma/client";
import { DbProvider, batch, logger } from "@vrooli/server";
import { generatePK } from "@vrooli/shared";

// Select shape for resource version batching
const resourceVersionSelect = {
    id: true,
    root: { select: { id: true } },
} as const;
// Payload type for resource version batching
type ResourceVersionPayload = Prisma.resource_versionGetPayload<{ select: typeof resourceVersionSelect }>;

type BatchRunCountsResult = Record<string, {
    runsStarted: number;
    runsCompleted: number;
    runCompletionTimeAverage: number;
    runContextSwitchesAverage: number;
}>

// Define the type for run records as returned by the select clause
type RunRecord = {
    id: bigint;
    resourceVersion: {
        id: bigint;
    } | null;
    completedAt: string | null;
    contextSwitches: number | null;
    startedAt: string | null;
    timeElapsed: number | null;
}

/**
 * Batch collects run counts for a list of resource versions
 * @param resourceVersionIds The IDs of the resource versions to collect run counts for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of resource version IDs to various run counts
 */
async function batchRunCounts(
    resourceVersionIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunCountsResult> {
    const initialResult = Object.fromEntries(resourceVersionIds.map(id => [id, {
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
                    const versionId = run.resourceVersion?.id;
                    if (!versionId) return;
                    
                    // Safely convert versionId to string  
                    const versionIdStr = versionId.toString();
                    
                    const currResult = result[versionIdStr];
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
                        if (run.contextSwitches !== null && run.contextSwitches !== undefined) {
                            currResult.runContextSwitchesAverage += run.contextSwitches;
                        }
                    }
                });
            },
            select: {
                id: true,
                resourceVersion: {
                    select: { id: true },
                },
                completedAt: true,
                contextSwitches: true,
                startedAt: true,
                timeElapsed: true,
            },
            where: {
                resourceVersion: { id: { in: resourceVersionIds.map(id => BigInt(id)) } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
        });
        
        // For the averages, divide by the number of runs completed
        Object.entries(result).forEach(([versionId, currResult]) => {
            if (!currResult || typeof versionId !== "string") return;
            if (currResult.runsCompleted > 0) {
                currResult.runCompletionTimeAverage /= currResult.runsCompleted;
                currResult.runContextSwitchesAverage /= currResult.runsCompleted;
            }
        });
        
        return result;
    } catch (error) {
        logger.error("batchRunCounts caught error", { error });
    }
    return initialResult;
}

/**
 * Creates periodic stats for all resources
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export async function logResourceStats(
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
): Promise<void> {
    try {
        await batch<Prisma.resource_versionFindManyArgs, ResourceVersionPayload>({
            objectType: "ResourceVersion",
            processBatch: async (batch: ResourceVersionPayload[]) => {
                // Find and count all runs associated with the latest routine versions, which 
                // have been started or completed within the period
                const runCountsByVersion = await batchRunCounts(batch.map(version => version.id.toString()), periodStart, periodEnd);
                // Create stats for each routine
                await DbProvider.get().stats_resource.createMany({
                    data: batch.map(resourceVersion => {
                        const runCounts = runCountsByVersion[resourceVersion.id.toString()];
                        if (!runCounts) return;
                        return {
                            id: generatePK(),
                            resourceId: resourceVersion.root.id,
                            periodStart,
                            periodEnd,
                            periodType,
                            references: 0,
                            referencedBy: 0,
                            ...runCounts,
                        };
                    }).filter((data): data is Exclude<typeof data, undefined> => !!data),
                });
            },
            select: resourceVersionSelect,
            where: {
                isDeleted: false,
                isLatest: true,
                root: { isDeleted: false },
            },
        });
    } catch (error) {
        logger.error("logResourceStats caught error", { error, trace: "0422", periodType, periodStart, periodEnd });
    }
}
