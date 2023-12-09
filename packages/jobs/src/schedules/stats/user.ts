import { PrismaType, batch, batchGroup } from "@local/server";
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
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to api stats
 */
const batchApis = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchApisResult> => batchGroup<Prisma.apiFindManyArgs, BatchApisResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        apisCreated: 0,
    }])),
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
    prisma,
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

/**
 * Batch collects organization stats for a list of users
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to organization stats
 */
const batchOrganizations = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchOrganizationsResult> => batchGroup<Prisma.organizationFindManyArgs, BatchOrganizationsResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        organizationsCreated: 0,
    }])),
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
    prisma,
    select: {
        id: true,
        createdById: true,
    },
    where: {
        created_at: { gte: periodStart, lt: periodEnd },
        createdById: { in: userIds },
    },
});

/**
 * Batch collects project stats for a list of users
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to project stats
 */
const batchProjects = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchProjectsResult> => batchGroup<Prisma.projectFindManyArgs, BatchProjectsResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        projectsCreated: 0,
        projectsCompleted: 0,
        projectCompletionTimeAverage: 0,
    }])),
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
    prisma,
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

/**
 * Batch collects quiz stats for a list of users
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to quiz stats
 */
const batchQuizzes = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchQuizzesResult> => batchGroup<Prisma.quiz_attemptFindManyArgs, BatchQuizzesResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        quizzesPassed: 0,
        quizzesFailed: 0,
    }])),
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
    prisma,
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

/**
 * Batch collects routine stats for a list of users
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to routine stats
 */
const batchRoutines = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRoutinesResult> => batchGroup<Prisma.routineFindManyArgs, BatchRoutinesResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        routinesCreated: 0,
        routinesCompleted: 0,
        routineCompletionTimeAverage: 0,
    }])),
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
    prisma,
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

/**
 * Batch collects run project stats for a list of users
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to run project stats
 */
const batchRunProjects = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunProjectsResult> => batchGroup<Prisma.run_projectFindManyArgs, BatchRunProjectsResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        runProjectsStarted: 0,
        runProjectsCompleted: 0,
        runProjectCompletionTimeAverage: 0,
        runProjectContextSwitchesAverage: 0,
    }])),
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
    prisma,
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

/**
 * Batch collects run routine stats for a list of users
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to run routine stats
 */
const batchRunRoutines = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunRoutinesResult> => batchGroup<Prisma.run_routineFindManyArgs, BatchRunRoutinesResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        runRoutinesStarted: 0,
        runRoutinesCompleted: 0,
        runRoutineCompletionTimeAverage: 0,
        runRoutineContextSwitchesAverage: 0,
    }])),
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
    prisma,
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

/**
 * Batch collects smart contract stats for a list of users
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to smart contract stats
 */
const batchSmartContracts = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchSmartContractsResult> => batchGroup<Prisma.smart_contractFindManyArgs, BatchSmartContractsResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        smartContractsCreated: 0,
        smartContractsCompleted: 0,
        smartContractCompletionTimeAverage: 0,
    }])),
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
    prisma,
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

/**
 * Batch collects standard stats for a list of users
 * @param prisma The Prisma client
 * @param userIds The IDs of the users to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of user IDs to standard stats
 */
const batchStandards = async (
    prisma: PrismaType,
    userIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchStandardsResult> => batchGroup<Prisma.standardFindManyArgs, BatchStandardsResult>({
    initialResult: Object.fromEntries(userIds.map(id => [id, {
        standardsCreated: 0,
        standardsCompleted: 0,
        standardCompletionTimeAverage: 0,
    }])),
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
    prisma,
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
) => await batch<Prisma.userFindManyArgs>({
    objectType: "User",
    processBatch: async (batch, prisma) => {
        // Get user ids, so we can query various tables for stats
        const userIds = batch.map(user => user.id);
        // Batch collect stats
        const apiStats = await batchApis(prisma, userIds, periodStart, periodEnd);
        const organizationStats = await batchOrganizations(prisma, userIds, periodStart, periodEnd);
        const projectStats = await batchProjects(prisma, userIds, periodStart, periodEnd);
        const quizStats = await batchQuizzes(prisma, userIds, periodStart, periodEnd);
        const routineStats = await batchRoutines(prisma, userIds, periodStart, periodEnd);
        const runProjectStats = await batchRunProjects(prisma, userIds, periodStart, periodEnd);
        const runRoutineStats = await batchRunRoutines(prisma, userIds, periodStart, periodEnd);
        const smartContractStats = await batchSmartContracts(prisma, userIds, periodStart, periodEnd);
        const standardStats = await batchStandards(prisma, userIds, periodStart, periodEnd);
        // Create stats for each user
        await prisma.stats_user.createMany({
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
    trace: "0426",
    traceObject: { periodType, periodStart, periodEnd },
    where: {
        updated_at: {
            gte: new Date(new Date().getTime() - 1000 * 60 * 60 * 24 * 90).toISOString(),
        },
    },
});