import pkg, { PeriodType } from '@prisma/client';
import { PrismaType } from '../types';
const { PrismaClient } = pkg;

type BatchDirectoryRunCountsResult = Record<string, {
    runsStarted: number;
    runsCompleted: number;
    runCompletionTimeAverage: number;
    runContextSwitchesAverage: number;
}>

/**
 * Batch collects run counts for a list of routine versions
 * @param prisma The Prisma client
 * @param routineVersionIds The IDs of the routine versions to collect run counts for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of routine version IDs to various run counts
 */
const batchRunCounts = async (
    prisma: PrismaType,
    routineVersionIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchDirectoryRunCountsResult> => {
    // Initialize return value
    const result: BatchDirectoryRunCountsResult = {};
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all runs associated with the routine versions
        const batch = await prisma.run_routine.findMany({
            where: {
                routineVersion: { id: { in: routineVersionIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ]
            },
            select: {
                id: true,
                routineVersion: {
                    select: { id: true }
                },
                completedAt: true,
                contextSwitches: true,
                startedAt: true,
                timeElapsed: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each run, increment the counts for the routine version
        batch.forEach(run => {
            const versionId = run.routineVersion?.id;
            if (!versionId) { return }
            if (!result[versionId]) {
                result[versionId] = {
                    runsStarted: 0,
                    runsCompleted: 0,
                    runCompletionTimeAverage: 0,
                    runContextSwitchesAverage: 0,
                };
            }
            // If runStarted within period, increment runsStarted
            if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                result[versionId].runsStarted += 1;
            }
            // If runCompleted within period, increment runsCompleted 
            // and update averages
            if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                result[versionId].runsCompleted += 1;
                if (run.timeElapsed !== null) result[versionId].runCompletionTimeAverage += run.timeElapsed;
                result[versionId].runContextSwitchesAverage += run.contextSwitches;
            }
        });
    } while (currentBatchSize === batchSize);
    // For the averages, divide by the number of runs completed
    Object.keys(result).forEach(versionId => {
        if (result[versionId].runsCompleted > 0) {
            result[versionId].runCompletionTimeAverage /= result[versionId].runsCompleted;
            result[versionId].runContextSwitchesAverage /= result[versionId].runsCompleted;
        }
    });
    return result;
}

/**
 * Creates periodic stats for all routines
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logRoutineStats = async (
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) => {
    // Initialize the Prisma client
    const prisma = new PrismaClient();
    // We may be dealing with a lot of data, so we need to do this in batches
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all latest (so should only be associated with one routine) routine versions
        const batch = await prisma.routine_version.findMany({
            where: {
                isDeleted: false,
                isLatest: true,
                root: { isDeleted: false },
            },
            select: {
                id: true,
                root: {
                    select: { id: true }
                },
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // Find and count all runs associated with the latest routine versions, which 
        // have been started or completed within the period
        const runCountsByVersion = await batchRunCounts(prisma, batch.map(version => version.id), periodStart, periodEnd);
        // Create stats for each routine
        await prisma.stats_routine.createMany({
            data: batch.map(routineVersion => ({
                routineId: routineVersion.root.id,
                periodStart,
                periodEnd,
                periodType,
                runsStarted: runCountsByVersion[routineVersion.id].runsStarted,
                runsCompleted: runCountsByVersion[routineVersion.id].runsCompleted,
                runCompletionTimeAverage: runCountsByVersion[routineVersion.id].runCompletionTimeAverage,
                runContextSwitchesAverage: runCountsByVersion[routineVersion.id].runContextSwitchesAverage,
            }))
        });
    } while (currentBatchSize === batchSize);
}