import pkg, { PeriodType, QuizAttemptStatus } from '@prisma/client';
const { PrismaClient } = pkg;

/**
 * Creates periodic site-wide stats
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logSiteStats = async (
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) => {
    // Initialize the Prisma client
    const prisma = new PrismaClient();
    // Find all users active in the past 90 days
    const activeUsers = await prisma.user.count({
        where: {
            updated_at: {
                gte: new Date(new Date().getTime() - 1000 * 60 * 60 * 24 * 90).toISOString(),
            },
        },
    });
    // Find all apis created within the period
    const apisCreated = await prisma.api.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find all organizations created within the period
    const organizationsCreated = await prisma.organization.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
        },
    });
    // Find all projects created within the period
    const projectsCreated = await prisma.project.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find all projects completed within the period
    const projectsCompleted = await prisma.project.count({
        where: {
            completedAt: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find the sum of all completion intervals (completedAt - created_at) 
    // for projects completed within the period 
    // NOTE: Prisma does not support aggregating by DateTime fields, 
    // so we must use a raw query.
    const projectsCompletedSum: number = projectsCompleted > 0 ? await prisma.$queryRaw`
        SELECT SUM(completedAt - created_at) AS time
        FROM project
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd} AND isDeleted = false
    ` : 0;
    // Calculate the average project completion time
    const projectCompletionTimeAverage = projectsCompleted > 0 ? projectsCompletedSum / projectsCompleted : 0;
    // Find all quizzes created within the period
    const quizzesCreated = await prisma.quiz.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
        },
    });
    // Find all quiz attempts completed within the period
    const quizzesCompleted = await prisma.quiz_attempt.count({
        where: {
            updated_at: { gte: periodStart, lte: periodEnd },
            status: { in: [QuizAttemptStatus.Passed, QuizAttemptStatus.Failed] },
        },
    });
    // Find all routines created within the period
    const routinesCreated = await prisma.routine.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find all routines completed within the period
    const routinesCompleted = await prisma.routine.count({
        where: {
            completedAt: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find the sum of all completion intervals (completedAt - created_at)
    // for routines completed within the period
    // NOTE: Prisma does not support aggregating by DateTime fields,
    // so we must use a raw query.
    const routinesCompletedSum: number = routinesCompleted > 0 ? await prisma.$queryRaw`
        SELECT SUM(completedAt - created_at) AS time
        FROM routine
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd} AND isDeleted = false
    ` : 0;
    // Calculate the average routine completion time
    const routineCompletionTimeAverage = routinesCompleted > 0 ? routinesCompletedSum / routinesCompleted : 0;
    // Find the total complexity and simplicity of all routines completed within the period
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
    // Calculate average routine complexity and simplicity
    const routineComplexityAverage = routinesCompleted > 0 ? (routineSimplicitySum._sum.complexity ?? 0) / routinesCompleted : 0;
    const routineSimplicityAverage = routinesCompleted > 0 ? (routineSimplicitySum._sum.simplicity ?? 0) / routinesCompleted : 0;
    // Find all run projects started within the period
    const runProjectsStarted = await prisma.run_project.count({
        where: {
            startedAt: { gte: periodStart, lte: periodEnd },
        },
    });
    // Find all run projects completed within the period
    const runProjectsCompleted = await prisma.run_project.count({
        where: {
            completedAt: { gte: periodStart, lte: periodEnd },
        },
    });
    // Find the sum of all completion intervals (completedAt - startedAt) 
    // for run projects completed within the period
    // NOTE: Prisma does not support aggregating by DateTime fields,
    // so we must use a raw query.
    const runProjectsCompletedSum: number = runProjectsCompleted > 0 ? await prisma.$queryRaw`
        SELECT SUM(completedAt - startedAt) AS time
        FROM run_project
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
    ` : 0;
    // Calculate the average run project completion time
    const runProjectCompletionTimeAverage = runProjectsCompleted > 0 ? runProjectsCompletedSum / runProjectsCompleted : 0;
    // Find the sum of all context switches for run projects completed within the period
    const runProjectContextSwitchesSum = await prisma.run_project.aggregate({
        where: {
            completedAt: { gte: periodStart, lte: periodEnd },
        },
        _sum: {
            contextSwitches: true,
        },
    });
    // Calculate the average run project context switches
    const runProjectContextSwitchesAverage = runProjectsCompleted > 0 ? (runProjectContextSwitchesSum._sum.contextSwitches ?? 0) / runProjectsCompleted : 0;
    // Find all run routines started within the period
    const runRoutinesStarted = await prisma.run_routine.count({
        where: {
            startedAt: { gte: periodStart, lte: periodEnd },
        },
    });
    // Find all run routines completed within the period
    const runRoutinesCompleted = await prisma.run_routine.count({
        where: {
            completedAt: { gte: periodStart, lte: periodEnd },
        },
    });
    // Find the sum of all completion intervals (completedAt - startedAt)
    // for run routines completed within the period
    // NOTE: Prisma does not support aggregating by DateTime fields,
    // so we must use a raw query.
    const runRoutinesCompletedSum: number = runRoutinesCompleted > 0 ? await prisma.$queryRaw`
        SELECT SUM(completedAt - startedAt) AS time
        FROM run_routine
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
    ` : 0;
    // Calculate the average run routine completion time
    const runRoutineCompletionTimeAverage = runRoutinesCompleted > 0 ? runRoutinesCompletedSum / runRoutinesCompleted : 0;
    // Find the sum of all context switches for run routines completed within the period
    const runRoutineContextSwitchesSum = await prisma.run_routine.aggregate({
        where: {
            completedAt: { gte: periodStart, lte: periodEnd },
        },
        _sum: {
            contextSwitches: true,
        },
    });
    // Calculate the average run routine context switches
    const runRoutineContextSwitchesAverage = runRoutinesCompleted > 0 ? (runRoutineContextSwitchesSum._sum.contextSwitches ?? 0) / runRoutinesCompleted : 0;
    // Find all smart contracts created within the period
    const smartContractsCreated = await prisma.smart_contract.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find all smartContracts completed within the period
    const smartContractsCompleted = await prisma.smart_contract.count({
        where: {
            completedAt: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find the sum of all completion intervals (completedAt - created_at)
    // for smart contracts completed within the period
    // NOTE: Prisma does not support aggregating by DateTime fields,
    // so we must use a raw query.
    const smartContractsCompletedSum: number = smartContractsCompleted > 0 ? await prisma.$queryRaw`
        SELECT SUM(completedAt - created_at) AS time
        FROM smart_contract
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
    ` : 0;
    // Calculate the average smart contract completion time
    const smartContractCompletionTimeAverage = smartContractsCompleted > 0 ? smartContractsCompletedSum / smartContractsCompleted : 0;
    // Find all standards created within the period
    const standardsCreated = await prisma.standard.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find all standards completed within the period
    const standardsCompleted = await prisma.standard.count({
        where: {
            completedAt: { gte: periodStart, lte: periodEnd },
            isDeleted: false,
        },
    });
    // Find the sum of all completion intervals (completedAt - created_at)
    // for standards completed within the period
    // NOTE: Prisma does not support aggregating by DateTime fields,
    // so we must use a raw query.
    const standardsCompletedSum: number = standardsCompleted > 0 ? await prisma.$queryRaw`
        SELECT SUM(completedAt - created_at) AS time
        FROM standard
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
    ` : 0;
    // Calculate the average standard completion time
    const standardCompletionTimeAverage = standardsCompleted > 0 ? standardsCompletedSum / standardsCompleted : 0;
    // Find all verified emails created within the period
    const verifiedEmailsCreated = await prisma.email.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
            verified: true,
        },
    });
    // Find all verified wallets created within the period
    const verifiedWalletsCreated = await prisma.wallet.count({
        where: {
            created_at: { gte: periodStart, lte: periodEnd },
            verified: true,
        },
    });
    // Store in database
    await prisma.stats_site.create({
        data: {
            periodStart,
            periodEnd,
            periodType,
            activeUsers,
            apiCalls: 0, //TODO no way to track calls yet
            apisCreated,
            organizationsCreated,
            projectsCreated,
            projectsCompleted,
            projectCompletionTimeAverage,
            quizzesCreated,
            quizzesCompleted,
            routinesCreated,
            routinesCompleted,
            routineCompletionTimeAverage,
            routineComplexityAverage,
            routineSimplicityAverage,
            runProjectsStarted,
            runProjectsCompleted,
            runProjectCompletionTimeAverage,
            runProjectContextSwitchesAverage,
            runRoutinesStarted,
            runRoutinesCompleted,
            runRoutineCompletionTimeAverage,
            runRoutineContextSwitchesAverage,
            smartContractsCreated,
            smartContractsCompleted,
            smartContractCompletionTimeAverage,
            smartContractCalls: 0, //TODO no way to track calls yet
            standardsCreated,
            standardsCompleted,
            standardCompletionTimeAverage,
            verifiedEmailsCreated,
            verifiedWalletsCreated,
        },
    })
    // Close the Prisma client
    await prisma.$disconnect();
}