import pkg, { QuizAttemptStatus } from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
export const logSiteStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    const data = {
        periodStart,
        periodEnd,
        periodType,
        activeUsers: 0,
        apiCalls: 0,
        apisCreated: 0,
        organizationsCreated: 0,
        projectsCreated: 0,
        projectsCompleted: 0,
        projectCompletionTimeAverage: 0,
        quizzesCreated: 0,
        quizzesCompleted: 0,
        routinesCreated: 0,
        routinesCompleted: 0,
        routineCompletionTimeAverage: 0,
        routineComplexityAverage: 0,
        routineSimplicityAverage: 0,
        runProjectsStarted: 0,
        runProjectsCompleted: 0,
        runProjectCompletionTimeAverage: 0,
        runProjectContextSwitchesAverage: 0,
        runRoutinesStarted: 0,
        runRoutinesCompleted: 0,
        runRoutineCompletionTimeAverage: 0,
        runRoutineContextSwitchesAverage: 0,
        smartContractsCreated: 0,
        smartContractsCompleted: 0,
        smartContractCompletionTimeAverage: 0,
        smartContractCalls: 0,
        standardsCreated: 0,
        standardsCompleted: 0,
        standardCompletionTimeAverage: 0,
        verifiedEmailsCreated: 0,
        verifiedWalletsCreated: 0,
    };
    try {
        data.activeUsers = await prisma.user.count({
            where: {
                updated_at: {
                    gte: new Date(new Date().getTime() - 1000 * 60 * 60 * 24 * 90).toISOString(),
                },
            },
        });
        data.apisCreated = await prisma.api.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        data.organizationsCreated = await prisma.organization.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
            },
        });
        data.projectsCreated = await prisma.project.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        data.projectsCompleted = await prisma.project.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        const projectsCompletedSum = data.projectsCompleted > 0 ? await prisma.$queryRaw `
        SELECT SUM(completedAt - created_at) AS time
        FROM project
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd} AND isDeleted = false
    ` : 0;
        data.projectCompletionTimeAverage = data.projectsCompleted > 0 ? projectsCompletedSum / data.projectsCompleted : 0;
        data.quizzesCreated = await prisma.quiz.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
            },
        });
        data.quizzesCompleted = await prisma.quiz_attempt.count({
            where: {
                updated_at: { gte: periodStart, lte: periodEnd },
                status: { in: [QuizAttemptStatus.Passed, QuizAttemptStatus.Failed] },
            },
        });
        data.routinesCreated = await prisma.routine.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        const routinesCompleted = await prisma.routine.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        const routinesCompletedSum = routinesCompleted > 0 ? await prisma.$queryRaw `
        SELECT SUM(completedAt - created_at) AS time
        FROM routine
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd} AND isDeleted = false
    ` : 0;
        data.routineCompletionTimeAverage = routinesCompleted > 0 ? routinesCompletedSum / routinesCompleted : 0;
        const routineSimplicitySum = await prisma.routine_version.aggregate({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
                isLatest: true,
                root: { isDeleted: false },
            },
            _sum: {
                complexity: true,
                simplicity: true,
            },
        });
        data.routineComplexityAverage = routinesCompleted > 0 ? (routineSimplicitySum._sum.complexity ?? 0) / routinesCompleted : 0;
        data.routineSimplicityAverage = routinesCompleted > 0 ? (routineSimplicitySum._sum.simplicity ?? 0) / routinesCompleted : 0;
        data.runProjectsStarted = await prisma.run_project.count({
            where: {
                startedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        const runProjectsCompleted = await prisma.run_project.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        const runProjectsCompletedSum = runProjectsCompleted > 0 ? await prisma.$queryRaw `
        SELECT SUM(completedAt - startedAt) AS time
        FROM run_project
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
    ` : 0;
        data.runProjectCompletionTimeAverage = runProjectsCompleted > 0 ? runProjectsCompletedSum / runProjectsCompleted : 0;
        const runProjectContextSwitchesSum = await prisma.run_project.aggregate({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
            _sum: {
                contextSwitches: true,
            },
        });
        data.runProjectContextSwitchesAverage = runProjectsCompleted > 0 ? (runProjectContextSwitchesSum._sum.contextSwitches ?? 0) / runProjectsCompleted : 0;
        data.runRoutinesStarted = await prisma.run_routine.count({
            where: {
                startedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        const runRoutinesCompleted = await prisma.run_routine.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        const runRoutinesCompletedSum = runRoutinesCompleted > 0 ? await prisma.$queryRaw `
        SELECT SUM(completedAt - startedAt) AS time
        FROM run_routine
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
    ` : 0;
        data.runRoutineCompletionTimeAverage = runRoutinesCompleted > 0 ? runRoutinesCompletedSum / runRoutinesCompleted : 0;
        const runRoutineContextSwitchesSum = await prisma.run_routine.aggregate({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
            _sum: {
                contextSwitches: true,
            },
        });
        data.runRoutineContextSwitchesAverage = runRoutinesCompleted > 0 ? (runRoutineContextSwitchesSum._sum.contextSwitches ?? 0) / runRoutinesCompleted : 0;
        data.smartContractsCreated = await prisma.smart_contract.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        const smartContractsCompleted = await prisma.smart_contract.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        const smartContractsCompletedSum = smartContractsCompleted > 0 ? await prisma.$queryRaw `
        SELECT SUM(completedAt - created_at) AS time
        FROM smart_contract
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
    ` : 0;
        data.smartContractCompletionTimeAverage = smartContractsCompleted > 0 ? smartContractsCompletedSum / smartContractsCompleted : 0;
        data.standardsCreated = await prisma.standard.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        const standardsCompleted = await prisma.standard.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        const standardsCompletedSum = standardsCompleted > 0 ? await prisma.$queryRaw `
        SELECT SUM(completedAt - created_at) AS time
        FROM standard
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
    ` : 0;
        data.standardCompletionTimeAverage = standardsCompleted > 0 ? standardsCompletedSum / standardsCompleted : 0;
        data.verifiedEmailsCreated = await prisma.email.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
                verified: true,
            },
        });
        data.verifiedWalletsCreated = await prisma.wallet.count({
            where: {
                created_at: { gte: periodStart, lte: periodEnd },
                verified: true,
            },
        });
        await prisma.stats_site.create({ data });
    }
    catch (error) {
        logger.error("Caught error logging site statistics", { trace: "0423", data });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=site.js.map