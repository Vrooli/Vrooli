import pkg, { PeriodType, QuizAttemptStatus } from '@prisma/client';
import { PrismaType } from '../types';
const { PrismaClient } = pkg;

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
): Promise<BatchApisResult> => {
    // Initialize return value
    const result: BatchApisResult = Object.fromEntries(userIds.map(id => [id, {
        apisCreated: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all apis created by the users in the period
        const batch = await prisma.api.findMany({
            where: {
                created_at: { gte: periodStart, lt: periodEnd },
                createdById: { in: userIds },
                isDeleted: false,
            },
            select: {
                id: true,
                createdById: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each, add stats to the user
        batch.forEach(api => {
            const userId = api.createdById;
            if (!userId || !result[userId]) return;
            result[userId].apisCreated += 1;
        });
    } while (currentBatchSize === batchSize);
    return result;
}

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
): Promise<BatchOrganizationsResult> => {
    // Initialize return value
    const result: BatchOrganizationsResult = Object.fromEntries(userIds.map(id => [id, {
        organizationsCreated: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all organizations created by the users in the period
        const batch = await prisma.organization.findMany({
            where: {
                created_at: { gte: periodStart, lt: periodEnd },
                createdById: { in: userIds },
            },
            select: {
                id: true,
                createdById: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each, add stats to the user
        batch.forEach(organization => {
            const userId = organization.createdById;
            if (!userId || !result[userId]) return;
            result[userId].organizationsCreated += 1;
        });
    } while (currentBatchSize === batchSize);
    return result;
}

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
): Promise<BatchProjectsResult> => {
    // Initialize return value
    const result: BatchProjectsResult = Object.fromEntries(userIds.map(id => [id, {
        projectsCreated: 0,
        projectsCompleted: 0,
        projectCompletionTimeAverage: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all projects created by the users in the period
        const batch = await prisma.project.findMany({
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ]
            },
            select: {
                id: true,
                completedAt: true,
                created_at: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each, add stats to the user
        batch.forEach(project => {
            const userId = project.createdById;
            if (!userId || !result[userId]) return;
            result[userId].projectsCreated += 1;
            if (project.hasCompleteVersion) {
                result[userId].projectsCompleted += 1;
                if (project.completedAt) result[userId].projectCompletionTimeAverage += (new Date(project.completedAt).getTime() - new Date(project.created_at).getTime())
            }
        });
    } while (currentBatchSize === batchSize);
    // Calculate averages
    Object.keys(result).forEach(userId => {
        const user = result[userId];
        if (user.projectsCompleted > 0) {
            user.projectCompletionTimeAverage /= user.projectsCompleted;
        }
    });
    return result;
}

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
): Promise<BatchQuizzesResult> => {
    // Initialize return value
    const result: BatchQuizzesResult = Object.fromEntries(userIds.map(id => [id, {
        quizzesPassed: 0,
        quizzesFailed: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all quiz attempts created by the users in the period
        const batch = await prisma.quiz_attempt.findMany({
            where: {
                userId: { in: userIds },
                updated_at: { gte: periodStart, lte: periodEnd },
                status: { in: [QuizAttemptStatus.Passed, QuizAttemptStatus.Failed] },
            },
            select: {
                id: true,
                userId: true,
                status: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each, add stats to the user
        batch.forEach(attempt => {
            const userId = attempt.userId;
            if (!userId || !result[userId]) return;
            if (attempt.status === QuizAttemptStatus.Passed) {
                result[userId].quizzesPassed += 1;
            } else if (attempt.status === QuizAttemptStatus.Failed) {
                result[userId].quizzesFailed += 1;
            }
        });
    } while (currentBatchSize === batchSize);
    return result;
}

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
): Promise<BatchRoutinesResult> => {
    // Initialize return value
    const result: BatchRoutinesResult = Object.fromEntries(userIds.map(id => [id, {
        routinesCreated: 0,
        routinesCompleted: 0,
        routineCompletionTimeAverage: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all routines created by the users in the period
        const batch = await prisma.routine.findMany({
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ]
            },
            select: {
                id: true,
                completedAt: true,
                created_at: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each, add stats to the user
        batch.forEach(routine => {
            const userId = routine.createdById;
            if (!userId || !result[userId]) return;
            result[userId].routinesCreated += 1;
            if (routine.hasCompleteVersion) {
                result[userId].routinesCompleted += 1;
                if (routine.completedAt) result[userId].routineCompletionTimeAverage += (new Date(routine.completedAt).getTime() - new Date(routine.created_at).getTime())
            }
        });
    } while (currentBatchSize === batchSize);
    // Calculate averages
    Object.keys(result).forEach(userId => {
        const user = result[userId];
        if (user.routinesCompleted > 0) {
            user.routineCompletionTimeAverage /= user.routinesCompleted;
        }
    });
    return result;
}

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
): Promise<BatchRunProjectsResult> => {
    // Initialize return value
    const result: BatchRunProjectsResult = Object.fromEntries(userIds.map(id => [id, {
        runProjectsStarted: 0,
        runProjectsCompleted: 0,
        runProjectCompletionTimeAverage: 0,
        runProjectContextSwitchesAverage: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all runs associated with the users
        const batch = await prisma.run_project.findMany({
            where: {
                user: { id: { in: userIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ]
            },
            select: {
                id: true,
                user: {
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
            const userId = run.user?.id;
            if (!userId || !result[userId]) { return }
            // If runStarted within period, increment runsStarted
            if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                result[userId].runProjectsStarted += 1;
            }
            // If runCompleted within period, increment runsCompleted 
            // and update averages
            if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                result[userId].runProjectsCompleted += 1;
                if (run.timeElapsed !== null) result[userId].runProjectCompletionTimeAverage += run.timeElapsed;
                result[userId].runProjectContextSwitchesAverage += run.contextSwitches;
            }
        });
    } while (currentBatchSize === batchSize);
    // For the averages, divide by the number of runs completed
    Object.keys(result).forEach(userId => {
        if (result[userId].runProjectsCompleted > 0) {
            result[userId].runProjectCompletionTimeAverage /= result[userId].runProjectsCompleted;
            result[userId].runProjectContextSwitchesAverage /= result[userId].runProjectsCompleted;
        }
    });
    return result;
}

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
): Promise<BatchRunRoutinesResult> => {
    // Initialize return value
    const result: BatchRunRoutinesResult = Object.fromEntries(userIds.map(id => [id, {
        runRoutinesStarted: 0,
        runRoutinesCompleted: 0,
        runRoutineCompletionTimeAverage: 0,
        runRoutineContextSwitchesAverage: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all runs associated with the users
        const batch = await prisma.run_routine.findMany({
            where: {
                user: { id: { in: userIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ]
            },
            select: {
                id: true,
                user: {
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
            const userId = run.user?.id;
            if (!userId || !result[userId]) { return }
            // If runStarted within period, increment runsStarted
            if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                result[userId].runRoutinesStarted += 1;
            }
            // If runCompleted within period, increment runsCompleted 
            // and update averages
            if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                result[userId].runRoutinesCompleted += 1;
                if (run.timeElapsed !== null) result[userId].runRoutineCompletionTimeAverage += run.timeElapsed;
                result[userId].runRoutineContextSwitchesAverage += run.contextSwitches;
            }
        });
    } while (currentBatchSize === batchSize);
    // For the averages, divide by the number of runs completed
    Object.keys(result).forEach(userId => {
        if (result[userId].runRoutinesCompleted > 0) {
            result[userId].runRoutineCompletionTimeAverage /= result[userId].runRoutinesCompleted;
            result[userId].runRoutineContextSwitchesAverage /= result[userId].runRoutinesCompleted;
        }
    });
    return result;
}

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
): Promise<BatchSmartContractsResult> => {
    // Initialize return value
    const result: BatchSmartContractsResult = Object.fromEntries(userIds.map(id => [id, {
        smartContractsCreated: 0,
        smartContractsCompleted: 0,
        smartContractCompletionTimeAverage: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all smartContracts created by the users in the period
        const batch = await prisma.smart_contract.findMany({
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ]
            },
            select: {
                id: true,
                completedAt: true,
                created_at: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each, add stats to the user
        batch.forEach(smartContract => {
            const userId = smartContract.createdById;
            if (!userId || !result[userId]) return;
            result[userId].smartContractsCreated += 1;
            if (smartContract.hasCompleteVersion) {
                result[userId].smartContractsCompleted += 1;
                if (smartContract.completedAt) result[userId].smartContractCompletionTimeAverage += (new Date(smartContract.completedAt).getTime() - new Date(smartContract.created_at).getTime())
            }
        });
    } while (currentBatchSize === batchSize);
    // Calculate averages
    Object.keys(result).forEach(userId => {
        const user = result[userId];
        if (user.smartContractsCompleted > 0) {
            user.smartContractCompletionTimeAverage /= user.smartContractsCompleted;
        }
    });
    return result;
}

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
): Promise<BatchStandardsResult> => {
    // Initialize return value
    const result: BatchStandardsResult = Object.fromEntries(userIds.map(id => [id, {
        standardsCreated: 0,
        standardsCompleted: 0,
        standardCompletionTimeAverage: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all standards created by the users in the period
        const batch = await prisma.standard.findMany({
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ]
            },
            select: {
                id: true,
                completedAt: true,
                created_at: true,
                createdById: true,
                hasCompleteVersion: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each, add stats to the user
        batch.forEach(standard => {
            const userId = standard.createdById;
            if (!userId || !result[userId]) return;
            result[userId].standardsCreated += 1;
            if (standard.hasCompleteVersion) {
                result[userId].standardsCompleted += 1;
                if (standard.completedAt) result[userId].standardCompletionTimeAverage += (new Date(standard.completedAt).getTime() - new Date(standard.created_at).getTime())
            }
        });
    } while (currentBatchSize === batchSize);
    // Calculate averages
    Object.keys(result).forEach(userId => {
        const user = result[userId];
        if (user.standardsCompleted > 0) {
            user.standardCompletionTimeAverage /= user.standardsCompleted;
        }
    });
    return result;
}

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
    // Initialize the Prisma client
    const prisma = new PrismaClient();
    // We may be dealing with a lot of data, so we need to do this in batches
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all users active in the past 90 days
        const batch = await prisma.user.findMany({
            where: {
                updated_at: {
                    gte: new Date(new Date().getTime() - 1000 * 60 * 60 * 24 * 90).toISOString(),
                },
            },
            select: {
                id: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
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
                ...apiStats[user.id],
                ...organizationStats[user.id],
                ...projectStats[user.id],
                ...quizStats[user.id],
                ...routineStats[user.id],
                ...runProjectStats[user.id],
                ...runRoutineStats[user.id],
                ...smartContractStats[user.id],
                ...standardStats[user.id],
            }))
        });
    } while (currentBatchSize === batchSize);
}