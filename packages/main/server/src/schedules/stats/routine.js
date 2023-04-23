import pkg from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
const batchRunCounts = async (prisma, routineVersionIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(routineVersionIds.map(id => [id, {
            runsStarted: 0,
            runsCompleted: 0,
            runCompletionTimeAverage: 0,
            runContextSwitchesAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.run_routine.findMany({
            where: {
                routineVersion: { id: { in: routineVersionIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
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
            skip,
            take: batchSize,
        });
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(run => {
            const versionId = run.routineVersion?.id;
            if (!versionId || !result[versionId]) {
                return;
            }
            if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                result[versionId].runsStarted += 1;
            }
            if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                result[versionId].runsCompleted += 1;
                if (run.timeElapsed !== null)
                    result[versionId].runCompletionTimeAverage += run.timeElapsed;
                result[versionId].runContextSwitchesAverage += run.contextSwitches;
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(versionId => {
        if (result[versionId].runsCompleted > 0) {
            result[versionId].runCompletionTimeAverage /= result[versionId].runsCompleted;
            result[versionId].runContextSwitchesAverage /= result[versionId].runsCompleted;
        }
    });
    return result;
};
export const logRoutineStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.routine_version.findMany({
                where: {
                    isDeleted: false,
                    isLatest: true,
                    root: { isDeleted: false },
                },
                select: {
                    id: true,
                    root: {
                        select: { id: true },
                    },
                },
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            const runCountsByVersion = await batchRunCounts(prisma, batch.map(version => version.id), periodStart, periodEnd);
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
        } while (currentBatchSize === batchSize);
    }
    catch (error) {
        logger.error("Caught error logging routine statistics", { trace: "0422", periodType, periodStart, periodEnd });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=routine.js.map