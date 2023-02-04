import pkg, { PeriodType } from '@prisma/client';
import { PrismaType } from '../types';
const { PrismaClient } = pkg;

type BatchDirectoryCountsResult = Record<string, {
    directories: number;
    apis: number;
    notes: number;
    organizations: number;
    projects: number;
    routines: number;
    smartContracts: number;
    standards: number;
}>

type BatchDirectoryRunCountsResult = Record<string, {
    runsStarted: number;
    runsCompleted: number;
    runCompletionTimeAverage: number;
    runContextSwitchesAverage: number;
}>

/**
 * Batch collects directory counts for a list of project versions
 * @param prisma The Prisma client
 * @param projectVersionIds The IDs of the project versions to collect directory counts for
 * @returns A map of project version IDs to various directory counts
 */
const batchDirectoryCounts = async (
    prisma: PrismaType,
    projectVersionIds: string[],
): Promise<BatchDirectoryCountsResult> => {
    // Initialize return value
    const result: BatchDirectoryCountsResult = Object.fromEntries(projectVersionIds.map(id => [id, {
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
        // Find all directories associated with the project versions
        const batch = await prisma.project_version_directory.findMany({
            where: {
                projectVersion: { id: { in: projectVersionIds } },
            },
            select: {
                id: true,
                projectVersion: {
                    select: { id: true }
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
                    }
                }
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each directory, increment the counts for the project version
        batch.forEach(directory => {
            const versionId = directory.projectVersion.id;
            if (!result[versionId]) { return; }
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
}

/**
 * Batch collects run counts for a list of project versions
 * @param prisma The Prisma client
 * @param projectVersionIds The IDs of the project versions to collect run counts for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of project version IDs to various run counts
 */
const batchRunCounts = async (
    prisma: PrismaType,
    projectVersionIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchDirectoryRunCountsResult> => {
    // Initialize return value
    const result: BatchDirectoryRunCountsResult = Object.fromEntries(projectVersionIds.map(id => [id, {
        runsStarted: 0,
        runsCompleted: 0,
        runCompletionTimeAverage: 0,
        runContextSwitchesAverage: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all runs associated with the project versions
        const batch = await prisma.run_project.findMany({
            where: {
                projectVersion: { id: { in: projectVersionIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ]
            },
            select: {
                id: true,
                projectVersion: {
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
        // For each run, increment the counts for the project version
        batch.forEach(run => {
            const versionId = run.projectVersion?.id;
            if (!versionId || !result[versionId]) { return }
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
 * Creates periodic stats for all projects
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logProjectStats = async (
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
        // Find all latest (so should only be associated with one project) project versions
        const batch = await prisma.project_version.findMany({
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
        // Find and count all directories associated with the latest project versions
        const directoryCountsByVersion = await batchDirectoryCounts(prisma, batch.map(version => version.id));
        // Find and count all runs associated with the latest project versions, which 
        // have been started or completed within the period
        const runCountsByVersion = await batchRunCounts(prisma, batch.map(version => version.id), periodStart, periodEnd);
        // Create stats for each project
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
            }))
        });
    } while (currentBatchSize === batchSize);
}