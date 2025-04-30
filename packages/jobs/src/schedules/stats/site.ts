import { DbProvider, logger } from "@local/server";
import { PeriodType, Prisma } from "@prisma/client";

/**
 * Creates periodic site-wide stats
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export async function logSiteStats(
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) {
    // Initialize stats object
    const data: Prisma.stats_siteCreateInput = {
        periodStart,
        periodEnd,
        periodType,
        activeUsers: 0,
        apiCalls: 0, //TODO no way to track calls yet
        apisCreated: 0,
        codesCreated: 0,
        codesCompleted: 0,
        codeCompletionTimeAverage: 0,
        codeCalls: 0, //TODO no way to track calls yet
        projectsCreated: 0,
        projectsCompleted: 0,
        projectCompletionTimeAverage: 0,
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
        standardsCreated: 0,
        standardsCompleted: 0,
        standardCompletionTimeAverage: 0,
        teamsCreated: 0,
        verifiedEmailsCreated: 0,
        verifiedWalletsCreated: 0,
    };
    try {
        // Find all users active in the past 90 days
        data.activeUsers = await DbProvider.get().user.count({
            where: {
                // updatedAt should be sufficient to calculate active users, 
                // since even if they don't explicitly update their profile, 
                // their streak will be updated daily
                updatedAt: {
                    gte: new Date(new Date().getTime() - 1000 * 60 * 60 * 24 * 90).toISOString(),
                },
                // Make sure not to include bots
                isBot: false,
            },
        });
        // Find all apis created within the period
        data.apisCreated = await DbProvider.get().api.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        // Find all teams created within the period
        data.teamsCreated = await DbProvider.get().team.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
            },
        });
        // Find all projects created within the period
        data.projectsCreated = await DbProvider.get().project.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        // Find all projects completed within the period
        data.projectsCompleted = await DbProvider.get().project.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        // Find the sum of all completion intervals (completedAt - createdAt) 
        // for projects completed within the period 
        // NOTE: Prisma does not support aggregating by DateTime fields, 
        // so we must use a raw query.
        const projectsCompletedSum: number = data.projectsCompleted > 0 ? await DbProvider.get().$queryRaw`
        SELECT SUM(completedAt - createdAt) AS time
        FROM project
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd} AND isDeleted = false
` : 0;
        // Calculate the average project completion time
        data.projectCompletionTimeAverage = data.projectsCompleted > 0 ? projectsCompletedSum / data.projectsCompleted : 0;
        // Find all routines created within the period
        data.routinesCreated = await DbProvider.get().routine.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
                isInternal: false, // Exclude routines only used within other routines
            },
        });
        // Find all routines completed within the period
        const routinesCompleted = await DbProvider.get().routine.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
                isInternal: false, // Exclude routines only used within other routines
            },
        });
        // Find the sum of all completion intervals (completedAt - createdAt)
        // for routines completed within the period
        // NOTE: Prisma does not support aggregating by DateTime fields,
        // so we must use a raw query.
        const routinesCompletedSum: number = routinesCompleted > 0 ? await DbProvider.get().$queryRaw`
        SELECT SUM(completedAt - createdAt) AS time
        FROM routine
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd} AND isDeleted = false
` : 0;
        // Calculate the average routine completion time
        data.routineCompletionTimeAverage = routinesCompleted > 0 ? routinesCompletedSum / routinesCompleted : 0;
        // Find the total complexity and simplicity of all routines completed within the period
        const routineSimplicitySum = await DbProvider.get().routine_version.aggregate({
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
        data.routineComplexityAverage = routinesCompleted > 0 ? (routineSimplicitySum._sum.complexity ?? 0) / routinesCompleted : 0;
        data.routineSimplicityAverage = routinesCompleted > 0 ? (routineSimplicitySum._sum.simplicity ?? 0) / routinesCompleted : 0;
        // Find all run projects started within the period
        data.runProjectsStarted = await DbProvider.get().run_project.count({
            where: {
                startedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        // Find all run projects completed within the period
        const runProjectsCompleted = await DbProvider.get().run_project.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        // Find the sum of all completion intervals (completedAt - startedAt) 
        // for run projects completed within the period
        // NOTE: Prisma does not support aggregating by DateTime fields,
        // so we must use a raw query.
        const runProjectsCompletedSum: number = runProjectsCompleted > 0 ? await DbProvider.get().$queryRaw`
        SELECT SUM(completedAt - startedAt) AS time
        FROM run_project
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
` : 0;
        // Calculate the average run project completion time
        data.runProjectCompletionTimeAverage = runProjectsCompleted > 0 ? runProjectsCompletedSum / runProjectsCompleted : 0;
        // Find the sum of all context switches for run projects completed within the period
        const runProjectContextSwitchesSum = await DbProvider.get().run_project.aggregate({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
            _sum: {
                contextSwitches: true,
            },
        });
        // Calculate the average run project context switches
        data.runProjectContextSwitchesAverage = runProjectsCompleted > 0 ? (runProjectContextSwitchesSum._sum.contextSwitches ?? 0) / runProjectsCompleted : 0;
        // Find all run routines started within the period
        data.runRoutinesStarted = await DbProvider.get().run_routine.count({
            where: {
                startedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        // Find all run routines completed within the period
        const runRoutinesCompleted = await DbProvider.get().run_routine.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        // Find the sum of all completion intervals (completedAt - startedAt)
        // for run routines completed within the period
        // NOTE: Prisma does not support aggregating by DateTime fields,
        // so we must use a raw query.
        const runRoutinesCompletedSum: number = runRoutinesCompleted > 0 ? await DbProvider.get().$queryRaw`
        SELECT SUM(completedAt - startedAt) AS time
        FROM run_routine
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
` : 0;
        // Calculate the average run routine completion time
        data.runRoutineCompletionTimeAverage = runRoutinesCompleted > 0 ? runRoutinesCompletedSum / runRoutinesCompleted : 0;
        // Find the sum of all context switches for run routines completed within the period
        const runRoutineContextSwitchesSum = await DbProvider.get().run_routine.aggregate({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
            _sum: {
                contextSwitches: true,
            },
        });
        // Calculate the average run routine context switches
        data.runRoutineContextSwitchesAverage = runRoutinesCompleted > 0 ? (runRoutineContextSwitchesSum._sum.contextSwitches ?? 0) / runRoutinesCompleted : 0;
        // Find all codes created within the period
        data.codesCreated = await DbProvider.get().code.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        // Find all codes completed within the period
        const codesCompleted = await DbProvider.get().code.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
            },
        });
        // Find the sum of all completion intervals (completedAt - createdAt)
        // for codes completed within the period
        // NOTE: Prisma does not support aggregating by DateTime fields,
        // so we must use a raw query.
        const codesCompletedSum: number = codesCompleted > 0 ? await DbProvider.get().$queryRaw`
        SELECT SUM(completedAt - createdAt) AS time
        FROM code
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
` : 0;
        // Calculate the average code completion time
        data.codeCompletionTimeAverage = codesCompleted > 0 ? codesCompletedSum / codesCompleted : 0;
        // Find all standards created within the period
        data.standardsCreated = await DbProvider.get().standard.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
                isInternal: false, // Exclude standards only used within routines
            },
        });
        // Find all standards completed within the period
        const standardsCompleted = await DbProvider.get().standard.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
                isDeleted: false,
                isInternal: false, // Exclude standards only used within routines
            },
        });
        // Find the sum of all completion intervals (completedAt - createdAt)
        // for standards completed within the period
        // NOTE: Prisma does not support aggregating by DateTime fields,
        // so we must use a raw query.
        const standardsCompletedSum: number = standardsCompleted > 0 ? await DbProvider.get().$queryRaw`
        SELECT SUM(completedAt - createdAt) AS time
        FROM standard
        WHERE completedAt >= ${periodStart} AND completedAt <= ${periodEnd}
` : 0;
        // Calculate the average standard completion time
        data.standardCompletionTimeAverage = standardsCompleted > 0 ? standardsCompletedSum / standardsCompleted : 0;
        // Find all verified emails created within the period
        data.verifiedEmailsCreated = await DbProvider.get().email.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                verifiedAt: { not: null },
            },
        });
        // Find all verified wallets created within the period
        data.verifiedWalletsCreated = await DbProvider.get().wallet.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                verifiedAt: { not: null },
            },
        });
        // Store in database
        await DbProvider.get().stats_site.create({ data });
    } catch (error) {
        logger.error("logSiteStats caught error", { error, trace: "0423", data });
    }
};
