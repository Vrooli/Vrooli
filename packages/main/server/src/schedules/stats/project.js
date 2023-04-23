import pkg from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
const batchDirectoryCounts = async (prisma, projectVersionIds) => {
    const result = Object.fromEntries(projectVersionIds.map(id => [id, {
            directories: 0,
            apis: 0,
            notes: 0,
            organizations: 0,
            projects: 0,
            routines: 0,
            smartContracts: 0,
            standards: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.project_version_directory.findMany({
            where: {
                projectVersion: { id: { in: projectVersionIds } },
            },
            select: {
                id: true,
                projectVersion: {
                    select: { id: true },
                },
                _count: {
                    select: {
                        childApiVersions: true,
                        childNoteVersions: true,
                        childOrganizations: true,
                        childProjectVersions: true,
                        childRoutineVersions: true,
                        childSmartContractVersions: true,
                        childStandardVersions: true,
                    },
                },
            },
            skip,
            take: batchSize,
        });
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(directory => {
            const versionId = directory.projectVersion.id;
            if (!result[versionId]) {
                return;
            }
            result[versionId].directories += 1;
            result[versionId].apis += directory._count.childApiVersions;
            result[versionId].notes += directory._count.childNoteVersions;
            result[versionId].organizations += directory._count.childOrganizations;
            result[versionId].projects += directory._count.childProjectVersions;
            result[versionId].routines += directory._count.childRoutineVersions;
            result[versionId].smartContracts += directory._count.childSmartContractVersions;
            result[versionId].standards += directory._count.childStandardVersions;
        });
    } while (currentBatchSize === batchSize);
    return result;
};
const batchRunCounts = async (prisma, projectVersionIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(projectVersionIds.map(id => [id, {
            runsStarted: 0,
            runsCompleted: 0,
            runCompletionTimeAverage: 0,
            runContextSwitchesAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.run_project.findMany({
            where: {
                projectVersion: { id: { in: projectVersionIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
            select: {
                id: true,
                projectVersion: {
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
            const versionId = run.projectVersion?.id;
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
export const logProjectStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.project_version.findMany({
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
            const directoryCountsByVersion = await batchDirectoryCounts(prisma, batch.map(version => version.id));
            const runCountsByVersion = await batchRunCounts(prisma, batch.map(version => version.id), periodStart, periodEnd);
            await prisma.stats_project.createMany({
                data: batch.map(projectVersion => ({
                    projectId: projectVersion.root.id,
                    periodStart,
                    periodEnd,
                    periodType,
                    directories: directoryCountsByVersion[projectVersion.id].directories,
                    apis: directoryCountsByVersion[projectVersion.id].apis,
                    notes: directoryCountsByVersion[projectVersion.id].notes,
                    organizations: directoryCountsByVersion[projectVersion.id].organizations,
                    projects: directoryCountsByVersion[projectVersion.id].projects,
                    routines: directoryCountsByVersion[projectVersion.id].routines,
                    smartContracts: directoryCountsByVersion[projectVersion.id].smartContracts,
                    standards: directoryCountsByVersion[projectVersion.id].standards,
                    runsStarted: runCountsByVersion[projectVersion.id].runsStarted,
                    runsCompleted: runCountsByVersion[projectVersion.id].runsCompleted,
                    runCompletionTimeAverage: runCountsByVersion[projectVersion.id].runCompletionTimeAverage,
                    runContextSwitchesAverage: runCountsByVersion[projectVersion.id].runContextSwitchesAverage,
                })),
            });
        } while (currentBatchSize === batchSize);
    }
    catch (error) {
        logger.error("Caught error logging project statistics", { trace: "0420", periodType, periodStart, periodEnd });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=project.js.map