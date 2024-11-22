import { batch, batchGroup, logger, prismaInstance } from "@local/server";
import { PeriodType, Prisma } from "@prisma/client";

type BatchRunCountsResult = Record<string, {
    runsStarted: number;
    runsCompleted: number;
    runCompletionTimeAverage: number;
    runContextSwitchesAverage: number;
}>

/**
 * Batch collects run counts for a list of routine versions
 * @param routineVersionIds The IDs of the routine versions to collect run counts for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of routine version IDs to various run counts
 */
async function batchRunCounts(
    routineVersionIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunCountsResult> {
    const initialResult = Object.fromEntries(routineVersionIds.map(id => [id, {
        runsStarted: 0,
        runsCompleted: 0,
        runCompletionTimeAverage: 0,
        runContextSwitchesAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.run_routineFindManyArgs, BatchRunCountsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each run, increment the counts for the routine version
                batch.forEach(run => {
                    const versionId = run.routineVersion?.id;
                    if (!versionId) return;
                    const currResult = result[versionId];
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
                Object.keys(result).forEach(versionId => {
                    const currResult = result[versionId];
                    if (!currResult) return;
                    if (currResult.runsCompleted > 0) {
                        currResult.runCompletionTimeAverage /= currResult.runsCompleted;
                        currResult.runContextSwitchesAverage /= currResult.runsCompleted;
                    }
                });
                return result;
            },
            objectType: "RunRoutine",
            select: {
                id: true,
                routineVersion: {
                    select: { id: true },
                },
                completedAt: true,
                contextSwitches: true,
                startedAt: true,
                timeElapsed: true,
            },
            where: {
                routineVersion: { id: { in: routineVersionIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchRunCounts caught error", { error });
    }
    return initialResult;
};

/**
 * Creates periodic stats for all routines
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export async function logRoutineStats(
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) {
    try {
        await batch<Prisma.routine_versionFindManyArgs>({
            objectType: "RoutineVersion",
            processBatch: async (batch) => {
                // Find and count all runs associated with the latest routine versions, which 
                // have been started or completed within the period
                const runCountsByVersion = await batchRunCounts(batch.map(version => version.id), periodStart, periodEnd);
                // Create stats for each routine
                await prismaInstance.stats_routine.createMany({
                    data: batch.map(routineVersion => {
                        const runCounts = runCountsByVersion[routineVersion.id];
                        if (!runCounts) return;
                        return {
                            routineId: routineVersion.root.id,
                            periodStart,
                            periodEnd,
                            periodType,
                            ...runCounts,
                        };
                    }).filter((data): data is Exclude<typeof data, undefined> => !!data),
                });
            },
            select: {
                id: true,
                root: {
                    select: { id: true },
                },
            },
            where: {
                isDeleted: false,
                isLatest: true,
                root: { isDeleted: false },
            },
        });
    } catch (error) {
        logger.error("logRoutineStats caught error", { error, trace: "0422", periodType, periodStart, periodEnd });
    }
};
