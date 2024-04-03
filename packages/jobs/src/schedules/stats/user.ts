import { batch, batchGroup, logger, prismaInstance } from "@local/server";
import { PeriodType, Prisma, QuizAttemptStatus } from "@prisma/client";

type BatchApisResult = Record<string, {
    apisCreated: number;
}>

type BatchOrganizationsResult = Record<string, {
    organizationsCreated: number;
}>

type BatchProjectsResult = Record<string, {
    projectsCreated: number;
    projectsCompleted: number;
    projectCompletionTimeAverage: number;
}>

type BatchQuizzesResult = Record<string, {
    quizzesPassed: number;
    quizzesFailed: number;
}>

type BatchRoutinesResult = Record<string, {
    routinesCreated: number;
    routinesCompleted: number;
    routineCompletionTimeAverage: number;
}>

type BatchRunProjectsResult = Record<string, {
    runProjectsStarted: number;
    runProjectsCompleted: number;
    runProjectCompletionTimeAverage: number;
    runProjectContextSwitchesAverage: number;
}>

type BatchRunRoutinesResult = Record<string, {
    runRoutinesStarted: number;
    runRoutinesCompleted: number;
    runRoutineCompletionTimeAverage: number;
    runRoutineContextSwitchesAverage: number;
}>

type BatchSmartContractsResult = Record<string, {
    smartContractsCreated: number;
    smartContractsCompleted: number;
    smartContractCompletionTimeAverage: number;
}>

type BatchStandardsResult = Record<string, {
    standardsCreated: number;
    standardsCompleted: number;
    standardCompletionTimeAverage: number;
}>

/**
 * Batch collects api stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to api stats
 */
const batchApis = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchApisResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        apisCreated: 0,
    }]));
    try {
        return await batchGroup<Prisma.apiFindManyArgs, BatchApisResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(api => {
                    const userId = api.createdById;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    currResult.apisCreated += 1;
                });
            },
            objectType: "Api",
            select: {
                id: true,
                createdById: true,
            },
            where: {
                created_at: { gte: periodStart, lt: periodEnd },
                createdById: { in: userIds },
                isDeleted: false,
            },
        });
    } catch (error) {
        logger.error("batchApis caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects organization stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to organization stats
 */
const batchOrganizations = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchOrganizationsResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        organizationsCreated: 0,
    }]));
    try {
        return await batchGroup<Prisma.organizationFindManyArgs, BatchOrganizationsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(organization => {
                    const userId = organization.createdById;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    currResult.organizationsCreated += 1;
                });
            },
            objectType: "Organization",
            select: {
                id: true,
                createdById: true,
            },
            where: {
                created_at: { gte: periodStart, lt: periodEnd },
                createdById: { in: userIds },
            },
        });
    } catch (error) {
        logger.error("batchOrganizations caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects project stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to project stats
 */
const batchProjects = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchProjectsResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        projectsCreated: 0,
        projectsCompleted: 0,
        projectCompletionTimeAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.projectFindManyArgs, BatchProjectsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(project => {
                    const userId = project.createdById;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    currResult.projectsCreated += 1;
                    if (project.hasCompleteVersion) {
                        currResult.projectsCompleted += 1;
                        if (project.completedAt) currResult.projectCompletionTimeAverage += (new Date(project.completedAt).getTime() - new Date(project.created_at).getTime());
                    }
                });
            },
            finalizeResult: (result) => {
                // Calculate averages
                Object.keys(result).forEach(userId => {
                    const currResult = result[userId];
                    if (!currResult) return;
                    if (currResult.projectsCompleted > 0) {
                        currResult.projectCompletionTimeAverage /= currResult.projectsCompleted;
                    }
                });
                return result;
            },
            objectType: "Project",
            select: {
                id: true,
                completedAt: true,
                created_at: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchProjects caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects quiz stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to quiz stats
 */
const batchQuizzes = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchQuizzesResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        quizzesPassed: 0,
        quizzesFailed: 0,
    }]));
    try {
        return await batchGroup<Prisma.quiz_attemptFindManyArgs, BatchQuizzesResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(attempt => {
                    const userId = attempt.userId;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    if (attempt.status === QuizAttemptStatus.Passed) {
                        currResult.quizzesPassed += 1;
                    } else if (attempt.status === QuizAttemptStatus.Failed) {
                        currResult.quizzesFailed += 1;
                    }
                });
            },
            objectType: "QuizAttempt",
            select: {
                id: true,
                userId: true,
                status: true,
            },
            where: {
                userId: { in: userIds },
                updated_at: { gte: periodStart, lte: periodEnd },
                status: { in: [QuizAttemptStatus.Passed, QuizAttemptStatus.Failed] },
            },
        });
    } catch (error) {
        logger.error("batchQuizzes caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects routine stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to routine stats
 */
const batchRoutines = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRoutinesResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        routinesCreated: 0,
        routinesCompleted: 0,
        routineCompletionTimeAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.routineFindManyArgs, BatchRoutinesResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(routine => {
                    const userId = routine.createdById;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    currResult.routinesCreated += 1;
                    if (routine.hasCompleteVersion) {
                        currResult.routinesCompleted += 1;
                        if (routine.completedAt) currResult.routineCompletionTimeAverage += (new Date(routine.completedAt).getTime() - new Date(routine.created_at).getTime());
                    }
                });
            },
            finalizeResult: (result) => {
                // Calculate averages
                Object.keys(result).forEach(userId => {
                    const currResult = result[userId];
                    if (!currResult) return;
                    if (currResult.routinesCompleted > 0) {
                        currResult.routineCompletionTimeAverage /= currResult.routinesCompleted;
                    }
                });
                return result;
            },
            objectType: "Routine",
            select: {
                id: true,
                completedAt: true,
                created_at: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchRoutines caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects run project stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to run project stats
 */
const batchRunProjects = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunProjectsResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        runProjectsStarted: 0,
        runProjectsCompleted: 0,
        runProjectCompletionTimeAverage: 0,
        runProjectContextSwitchesAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.run_projectFindManyArgs, BatchRunProjectsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each run, increment the counts for the project version
                batch.forEach(run => {
                    const userId = run.user?.id;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    // If runStarted within period, increment runsStarted
                    if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                        currResult.runProjectsStarted += 1;
                    }
                    // If runCompleted within period, increment runsCompleted 
                    // and update averages
                    if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                        currResult.runProjectsCompleted += 1;
                        if (run.timeElapsed !== null) currResult.runProjectCompletionTimeAverage += run.timeElapsed;
                        currResult.runProjectContextSwitchesAverage += run.contextSwitches;
                    }
                });
            },
            finalizeResult: (result) => {
                // For the averages, divide by the number of runs completed
                Object.keys(result).forEach(userId => {
                    const currResult = result[userId];
                    if (!currResult) return;
                    if (currResult.runProjectsCompleted > 0) {
                        currResult.runProjectCompletionTimeAverage /= currResult.runProjectsCompleted;
                        currResult.runProjectContextSwitchesAverage /= currResult.runProjectsCompleted;
                    }
                });
                return result;
            },
            objectType: "RunProject",
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
                user: { id: { in: userIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchRunProjects caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects run routine stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to run routine stats
 */
const batchRunRoutines = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunRoutinesResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        runRoutinesStarted: 0,
        runRoutinesCompleted: 0,
        runRoutineCompletionTimeAverage: 0,
        runRoutineContextSwitchesAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.run_routineFindManyArgs, BatchRunRoutinesResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each run, increment the counts for the routine version
                batch.forEach(run => {
                    const userId = run.user?.id;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    // If runStarted within period, increment runsStarted
                    if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                        currResult.runRoutinesStarted += 1;
                    }
                    // If runCompleted within period, increment runsCompleted 
                    // and update averages
                    if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                        currResult.runRoutinesCompleted += 1;
                        if (run.timeElapsed !== null) currResult.runRoutineCompletionTimeAverage += run.timeElapsed;
                        currResult.runRoutineContextSwitchesAverage += run.contextSwitches;
                    }
                });
            },
            finalizeResult: (result) => {
                // For the averages, divide by the number of runs completed
                Object.keys(result).forEach(userId => {
                    const currResult = result[userId];
                    if (!currResult) return;
                    if (currResult.runRoutinesCompleted > 0) {
                        currResult.runRoutineCompletionTimeAverage /= currResult.runRoutinesCompleted;
                        currResult.runRoutineContextSwitchesAverage /= currResult.runRoutinesCompleted;
                    }
                });
                return result;
            },
            objectType: "RunRoutine",
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
                user: { id: { in: userIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchRunRoutines caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects smart contract stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to smart contract stats
 */
const batchSmartContracts = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchSmartContractsResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        smartContractsCreated: 0,
        smartContractsCompleted: 0,
        smartContractCompletionTimeAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.smart_contractFindManyArgs, BatchSmartContractsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(smartContract => {
                    const userId = smartContract.createdById;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    currResult.smartContractsCreated += 1;
                    if (smartContract.hasCompleteVersion) {
                        currResult.smartContractsCompleted += 1;
                        if (smartContract.completedAt) currResult.smartContractCompletionTimeAverage += (new Date(smartContract.completedAt).getTime() - new Date(smartContract.created_at).getTime());
                    }
                });
            },
            finalizeResult: (result) => {
                // Calculate averages
                Object.keys(result).forEach(userId => {
                    const currResult = result[userId];
                    if (!currResult) return;
                    if (currResult.smartContractsCompleted > 0) {
                        currResult.smartContractCompletionTimeAverage /= currResult.smartContractsCompleted;
                    }
                });
                return result;
            },
            objectType: "SmartContract",
            select: {
                id: true,
                completedAt: true,
                created_at: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchSmartContracts caught error", { error });
    }
    return initialResult;
};

/**
 * Batch collects standard stats for a list of users
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to standard stats
 */
const batchStandards = async (
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchStandardsResult> => {
    const initialResult = Object.fromEntries(userIds.map(id => [id, {
        standardsCreated: 0,
        standardsCompleted: 0,
        standardCompletionTimeAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.standardFindManyArgs, BatchStandardsResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each, add stats to the user
                batch.forEach(standard => {
                    const userId = standard.createdById;
                    if (!userId) return;
                    const currResult = result[userId];
                    if (!currResult) return;
                    currResult.standardsCreated += 1;
                    if (standard.hasCompleteVersion) {
                        currResult.standardsCompleted += 1;
                        if (standard.completedAt) currResult.standardCompletionTimeAverage += (new Date(standard.completedAt).getTime() - new Date(standard.created_at).getTime());
                    }
                });
            },
            finalizeResult: (result) => {
                // Calculate averages
                Object.keys(result).forEach(userId => {
                    const currResult = result[userId];
                    if (!currResult) return;
                    if (currResult.standardsCompleted > 0) {
                        currResult.standardCompletionTimeAverage /= currResult.standardsCompleted;
                    }
                });
                return result;
            },
            objectType: "Standard",
            select: {
                id: true,
                completedAt: true,
                created_at: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchStandards caught error", { error });
    }
    return initialResult;
};

/**
 * Creates periodic stats for all users
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logUserStats = async (
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) => {
    try {
        await batch<Prisma.userFindManyArgs>({
            objectType: "User",
            processBatch: async (batch) => {
                // Get user ids, so we can query various tables for stats
                const userIds = batch.map(user => user.id);
                // Batch collect stats
                const apiStats = await batchApis(userIds, periodStart, periodEnd);
                const organizationStats = await batchOrganizations(userIds, periodStart, periodEnd);
                const projectStats = await batchProjects(userIds, periodStart, periodEnd);
                const quizStats = await batchQuizzes(userIds, periodStart, periodEnd);
                const routineStats = await batchRoutines(userIds, periodStart, periodEnd);
                const runProjectStats = await batchRunProjects(userIds, periodStart, periodEnd);
                const runRoutineStats = await batchRunRoutines(userIds, periodStart, periodEnd);
                const smartContractStats = await batchSmartContracts(userIds, periodStart, periodEnd);
                const standardStats = await batchStandards(userIds, periodStart, periodEnd);
                // Create stats for each user
                await prismaInstance.stats_user.createMany({
                    data: batch.map(user => ({
                        userId: user.id,
                        periodStart,
                        periodEnd,
                        periodType,
                        ...(apiStats[user.id] || { apisCreated: 0 }),
                        ...(organizationStats[user.id] || { organizationsCreated: 0 }),
                        ...(projectStats[user.id] || { projectsCompleted: 0, projectCompletionTimeAverage: 0, projectsCreated: 0 }),
                        ...(quizStats[user.id] || { quizzesFailed: 0, quizzesPassed: 0 }),
                        ...(routineStats[user.id] || { routineCompletionTimeAverage: 0, routinesCompleted: 0, routinesCreated: 0 }),
                        ...(runProjectStats[user.id] || { runProjectCompletionTimeAverage: 0, runProjectContextSwitchesAverage: 0, runProjectsCompleted: 0, runProjectsStarted: 0 }),
                        ...(runRoutineStats[user.id] || { runRoutineCompletionTimeAverage: 0, runRoutineContextSwitchesAverage: 0, runRoutinesCompleted: 0, runRoutinesStarted: 0 }),
                        ...(smartContractStats[user.id] || { smartContractCompletionTimeAverage: 0, smartContractsCompleted: 0, smartContractsCreated: 0 }),
                        ...(standardStats[user.id] || { standardCompletionTimeAverage: 0, standardsCompleted: 0, standardsCreated: 0 }),
                    })),
                });
            },
            select: {
                id: true,
            },
            where: {
                updated_at: {
                    gte: new Date(new Date().getTime() - 1000 * 60 * 60 * 24 * 90).toISOString(),
                },
            },
        });
    } catch (error) {
        logger.error("logUserStats caught error", { error, trace: "0426", periodType, periodStart, periodEnd });
    }
};
