import { batch, batchGroup, DbProvider, logger } from "@local/server";
import { PeriodType, Prisma } from "@prisma/client";

type BatchDirectoryCountsResult = Record<string, {
    directories: number;
    apis: number;
    codes: number;
    notes: number;
    projects: number;
    routines: number;
    standards: number;
    teams: number;
}>

type BatchDirectoryRunCountsResult = Record<string, {
    runsStarted: number;
    runsCompleted: number;
    runCompletionTimeAverage: number;
    runContextSwitchesAverage: number;
}>

/**
 * Batch collects directory counts for a list of project versions
 * @param projectVersionIds The IDs of the project versions to collect directory counts for
 * @returns A map of project version IDs to various directory counts
 */
async function batchDirectoryCounts(
    projectVersionIds: string[],
): Promise<BatchDirectoryCountsResult> {
    const initialResult = Object.fromEntries(projectVersionIds.map(id => [id, {
        directories: 0,
        apis: 0,
        codes: 0,
        notes: 0,
        projects: 0,
        routines: 0,
        standards: 0,
        teams: 0,
    }]));
    try {
        return await batchGroup<Prisma.project_version_directoryFindManyArgs, BatchDirectoryCountsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each directory, increment the counts for the project version
                batch.forEach(directory => {
                    const versionId = directory.projectVersion.id;
                    const currResult = result[versionId];
                    if (!currResult) return;
                    currResult.directories += 1;
                    currResult.apis += directory._count.childApiVersions;
                    currResult.codes += directory._count.childCodeVersions;
                    currResult.notes += directory._count.childNoteVersions;
                    currResult.projects += directory._count.childProjectVersions;
                    currResult.routines += directory._count.childRoutineVersions;
                    currResult.standards += directory._count.childStandardVersions;
                    currResult.teams += directory._count.childTeams;
                });
            },
            objectType: "ProjectVersionDirectory",
            select: {
                id: true,
                projectVersion: {
                    select: { id: true },
                },
                _count: {
                    select: {
                        childApiVersions: true,
                        childCodeVersions: true,
                        childNoteVersions: true,
                        childProjectVersions: true,
                        childRoutineVersions: true,
                        childStandardVersions: true,
                        childTeams: true,
                    },
                },
            },
            where: {
                projectVersion: { id: { in: projectVersionIds } },
            },
        });
    } catch (error) {
        logger.error("batchDirectoryCounts caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects run counts for a list of project versions
 * @param projectVersionIds The IDs of the project versions to collect run counts for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of project version IDs to various run counts
 */
async function batchRunCounts(
    projectVersionIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchDirectoryRunCountsResult> {
    const initialResult = Object.fromEntries(projectVersionIds.map(id => [id, {
        runsStarted: 0,
        runsCompleted: 0,
        runCompletionTimeAverage: 0,
        runContextSwitchesAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.run_projectFindManyArgs, BatchDirectoryRunCountsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each run, increment the counts for the project version
                batch.forEach(run => {
                    const versionId = run.projectVersion?.id;
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
            objectType: "RunProject",
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
    } catch (error) {
        logger.error("batchRunCounts caught error", { error });
    }
    return initialResult;
};

/**
 * Creates periodic stats for all projects
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export async function logProjectStats(
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) {
    try {
        await batch<Prisma.project_versionFindManyArgs>({
            objectType: "ProjectVersion",
            processBatch: async (batch) => {
                // Find and count all directories associated with the latest project versions
                const directoryCountsByVersion = await batchDirectoryCounts(batch.map(version => version.id));
                // Find and count all runs associated with the latest project versions, which 
                // have been started or completed within the period
                const runCountsByVersion = await batchRunCounts(batch.map(version => version.id), periodStart, periodEnd);
                // Create stats for each project
                await DbProvider.get().stats_project.createMany({
                    data: batch.map(projectVersion => {
                        const directoryCounts = directoryCountsByVersion[projectVersion.id];
                        const runCounts = runCountsByVersion[projectVersion.id];
                        if (!directoryCounts || !runCounts) return;
                        return {
                            projectId: projectVersion.root.id,
                            periodStart,
                            periodEnd,
                            periodType,
                            ...directoryCounts,
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
        logger.error("logProjectStats caught error", { error, trace: "0420", periodType, periodStart, periodEnd });
    }
};
