import pkg, { QuizAttemptStatus } from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
const batchApis = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            apisCreated: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
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
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(api => {
            const userId = api.createdById;
            if (!userId || !result[userId])
                return;
            result[userId].apisCreated += 1;
        });
    } while (currentBatchSize === batchSize);
    return result;
};
const batchOrganizations = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            organizationsCreated: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
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
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(organization => {
            const userId = organization.createdById;
            if (!userId || !result[userId])
                return;
            result[userId].organizationsCreated += 1;
        });
    } while (currentBatchSize === batchSize);
    return result;
};
const batchProjects = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            projectsCreated: 0,
            projectsCompleted: 0,
            projectCompletionTimeAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.project.findMany({
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
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
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(project => {
            const userId = project.createdById;
            if (!userId || !result[userId])
                return;
            result[userId].projectsCreated += 1;
            if (project.hasCompleteVersion) {
                result[userId].projectsCompleted += 1;
                if (project.completedAt)
                    result[userId].projectCompletionTimeAverage += (new Date(project.completedAt).getTime() - new Date(project.created_at).getTime());
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(userId => {
        const user = result[userId];
        if (user.projectsCompleted > 0) {
            user.projectCompletionTimeAverage /= user.projectsCompleted;
        }
    });
    return result;
};
const batchQuizzes = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            quizzesPassed: 0,
            quizzesFailed: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
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
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(attempt => {
            const userId = attempt.userId;
            if (!userId || !result[userId])
                return;
            if (attempt.status === QuizAttemptStatus.Passed) {
                result[userId].quizzesPassed += 1;
            }
            else if (attempt.status === QuizAttemptStatus.Failed) {
                result[userId].quizzesFailed += 1;
            }
        });
    } while (currentBatchSize === batchSize);
    return result;
};
const batchRoutines = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            routinesCreated: 0,
            routinesCompleted: 0,
            routineCompletionTimeAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.routine.findMany({
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
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
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(routine => {
            const userId = routine.createdById;
            if (!userId || !result[userId])
                return;
            result[userId].routinesCreated += 1;
            if (routine.hasCompleteVersion) {
                result[userId].routinesCompleted += 1;
                if (routine.completedAt)
                    result[userId].routineCompletionTimeAverage += (new Date(routine.completedAt).getTime() - new Date(routine.created_at).getTime());
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(userId => {
        const user = result[userId];
        if (user.routinesCompleted > 0) {
            user.routineCompletionTimeAverage /= user.routinesCompleted;
        }
    });
    return result;
};
const batchRunProjects = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            runProjectsStarted: 0,
            runProjectsCompleted: 0,
            runProjectCompletionTimeAverage: 0,
            runProjectContextSwitchesAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.run_project.findMany({
            where: {
                user: { id: { in: userIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
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
            skip,
            take: batchSize,
        });
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(run => {
            const userId = run.user?.id;
            if (!userId || !result[userId]) {
                return;
            }
            if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                result[userId].runProjectsStarted += 1;
            }
            if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                result[userId].runProjectsCompleted += 1;
                if (run.timeElapsed !== null)
                    result[userId].runProjectCompletionTimeAverage += run.timeElapsed;
                result[userId].runProjectContextSwitchesAverage += run.contextSwitches;
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(userId => {
        if (result[userId].runProjectsCompleted > 0) {
            result[userId].runProjectCompletionTimeAverage /= result[userId].runProjectsCompleted;
            result[userId].runProjectContextSwitchesAverage /= result[userId].runProjectsCompleted;
        }
    });
    return result;
};
const batchRunRoutines = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            runRoutinesStarted: 0,
            runRoutinesCompleted: 0,
            runRoutineCompletionTimeAverage: 0,
            runRoutineContextSwitchesAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.run_routine.findMany({
            where: {
                user: { id: { in: userIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
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
            skip,
            take: batchSize,
        });
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(run => {
            const userId = run.user?.id;
            if (!userId || !result[userId]) {
                return;
            }
            if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                result[userId].runRoutinesStarted += 1;
            }
            if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                result[userId].runRoutinesCompleted += 1;
                if (run.timeElapsed !== null)
                    result[userId].runRoutineCompletionTimeAverage += run.timeElapsed;
                result[userId].runRoutineContextSwitchesAverage += run.contextSwitches;
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(userId => {
        if (result[userId].runRoutinesCompleted > 0) {
            result[userId].runRoutineCompletionTimeAverage /= result[userId].runRoutinesCompleted;
            result[userId].runRoutineContextSwitchesAverage /= result[userId].runRoutinesCompleted;
        }
    });
    return result;
};
const batchSmartContracts = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            smartContractsCreated: 0,
            smartContractsCompleted: 0,
            smartContractCompletionTimeAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.smart_contract.findMany({
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
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
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(smartContract => {
            const userId = smartContract.createdById;
            if (!userId || !result[userId])
                return;
            result[userId].smartContractsCreated += 1;
            if (smartContract.hasCompleteVersion) {
                result[userId].smartContractsCompleted += 1;
                if (smartContract.completedAt)
                    result[userId].smartContractCompletionTimeAverage += (new Date(smartContract.completedAt).getTime() - new Date(smartContract.created_at).getTime());
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(userId => {
        const user = result[userId];
        if (user.smartContractsCompleted > 0) {
            user.smartContractCompletionTimeAverage /= user.smartContractsCompleted;
        }
    });
    return result;
};
const batchStandards = async (prisma, userIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(userIds.map(id => [id, {
            standardsCreated: 0,
            standardsCompleted: 0,
            standardCompletionTimeAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.standard.findMany({
            where: {
                createdById: { in: userIds },
                isDeleted: false,
                OR: [
                    { created_at: { gte: periodStart, lt: periodEnd } },
                    { completedAt: { gte: periodStart, lt: periodEnd } },
                ],
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
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(standard => {
            const userId = standard.createdById;
            if (!userId || !result[userId])
                return;
            result[userId].standardsCreated += 1;
            if (standard.hasCompleteVersion) {
                result[userId].standardsCompleted += 1;
                if (standard.completedAt)
                    result[userId].standardCompletionTimeAverage += (new Date(standard.completedAt).getTime() - new Date(standard.created_at).getTime());
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(userId => {
        const user = result[userId];
        if (user.standardsCompleted > 0) {
            user.standardCompletionTimeAverage /= user.standardsCompleted;
        }
    });
    return result;
};
export const logUserStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
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
            skip += batchSize;
            currentBatchSize = batch.length;
            const userIds = batch.map(user => user.id);
            const apiStats = await batchApis(prisma, userIds, periodStart, periodEnd);
            const organizationStats = await batchOrganizations(prisma, userIds, periodStart, periodEnd);
            const projectStats = await batchProjects(prisma, userIds, periodStart, periodEnd);
            const quizStats = await batchQuizzes(prisma, userIds, periodStart, periodEnd);
            const routineStats = await batchRoutines(prisma, userIds, periodStart, periodEnd);
            const runProjectStats = await batchRunProjects(prisma, userIds, periodStart, periodEnd);
            const runRoutineStats = await batchRunRoutines(prisma, userIds, periodStart, periodEnd);
            const smartContractStats = await batchSmartContracts(prisma, userIds, periodStart, periodEnd);
            const standardStats = await batchStandards(prisma, userIds, periodStart, periodEnd);
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
                })),
            });
        } while (currentBatchSize === batchSize);
    }
    catch (error) {
        logger.error("Caught error logging user statistics", { trace: "0426", periodType, periodStart, periodEnd });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=user.js.map