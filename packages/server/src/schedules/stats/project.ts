import { PeriodType, Prisma } from "@prisma/client";
import { PrismaType } from "../../types";
import { batch } from "../../utils/batch";
import { batchGroup } from "../../utils/batchGroup";

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
): Promise<BatchDirectoryCountsResult> => batchGroup<Prisma.project_version_directoryFindManyArgs, BatchDirectoryCountsResult>({
    initialResult: Object.fromEntries(projectVersionIds.map(id => [id, {
        directories: 0,
        apis: 0,
        notes: 0,
        organizations: 0,
        projects: 0,
        routines: 0,
        smartContracts: 0,
        standards: 0,
    }])),
    processBatch: async (batch, result) => {
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
    },
    objectType: "ProjectVersionDirectory",
    prisma,
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
    where: {
        projectVersion: { id: { in: projectVersionIds } },
    },
});

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
): Promise<BatchDirectoryRunCountsResult> => batchGroup<Prisma.run_projectFindManyArgs, BatchDirectoryRunCountsResult>({
    initialResult: Object.fromEntries(projectVersionIds.map(id => [id, {
        runsStarted: 0,
        runsCompleted: 0,
        runCompletionTimeAverage: 0,
        runContextSwitchesAverage: 0,
    }])),
    processBatch: async (batch, result) => {
        // For each run, increment the counts for the project version
        batch.forEach(run => {
            const versionId = run.projectVersion?.id;
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
    objectType: "RunProject",
    prisma,
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
    where: {
        projectVersion: { id: { in: projectVersionIds } },
        OR: [
            { startedAt: { gte: periodStart, lte: periodEnd } },
            { completedAt: { gte: periodStart, lte: periodEnd } },
        ],
    },
});

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
) => await batch<Prisma.project_versionFindManyArgs>({
    objectType: "ProjectVersion",
    processBatch: async (batch, prisma) => {
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
            })),
        });
    },
    select: {
        id: true,
        root: {
            select: { id: true },
        },
    },
    trace: "0420",
    traceObject: { periodType, periodStart, periodEnd },
    where: {
        isDeleted: false,
        isLatest: true,
        root: { isDeleted: false },
    },
});
