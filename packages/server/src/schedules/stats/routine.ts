import { PeriodType, Prisma } from "@prisma/client";
import { PrismaType } from "../../types";
import { batch } from "../../utils/batch";
import { batchGroup } from "../../utils/batchGroup";

type BatchRunCountsResult = Record<string, {
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
): Promise<BatchRunCountsResult> => batchGroup<Prisma.run_routineFindManyArgs, BatchRunCountsResult>({
    initialResult: Object.fromEntries(routineVersionIds.map(id => [id, {
        runsStarted: 0,
        runsCompleted: 0,
        runCompletionTimeAverage: 0,
        runContextSwitchesAverage: 0,
    }])),
    processBatch: async (batch, result) => {
        // For each run, increment the counts for the routine version
        batch.forEach(run => {
            const versionId = run.routineVersion?.id;
            if (!versionId || !result[versionId]) { return; }
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
    },
    finalizeResult: (result) => {
        // For the averages, divide by the number of runs completed
        Object.keys(result).forEach(versionId => {
            if (result[versionId].runsCompleted > 0) {
                result[versionId].runCompletionTimeAverage /= result[versionId].runsCompleted;
                result[versionId].runContextSwitchesAverage /= result[versionId].runsCompleted;
            }
        });
        return result;
    },
    objectType: "RunRoutine",
    prisma,
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
) => await batch<Prisma.routine_versionFindManyArgs>({
    objectType: "RoutineVersion",
    processBatch: async (batch, prisma) => {
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
            })),
        });
    },
    select: {
        id: true,
        root: {
            select: { id: true },
        },
    },
    trace: "0422",
    traceObject: { periodType, periodStart, periodEnd },
    where: {
        isDeleted: false,
        isLatest: true,
        root: { isDeleted: false },
    },
});
