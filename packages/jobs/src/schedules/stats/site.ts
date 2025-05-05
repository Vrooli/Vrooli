import { DbProvider, logger } from "@local/server";
import { ResourceType } from "@local/shared";
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
    // Initialize stats object with new structure
    const data: Prisma.stats_siteCreateInput = {
        periodStart,
        periodEnd,
        periodType,
        activeUsers: 0,
        teamsCreated: 0,
        verifiedEmailsCreated: 0,
        verifiedWalletsCreated: 0,
        resourcesCreatedByType: {},
        resourcesCompletedByType: {},
        resourceCompletionTimeAverageByType: {},
        routineSimplicityAverage: 0,
        routineComplexityAverage: 0,
        runsStarted: 0,
        runsCompleted: 0,
        runCompletionTimeAverage: 0,
        runContextSwitchesAverage: 0,
    };

    // Use Partial Record for intermediate calculations
    const resourcesCreatedByType: Partial<Record<ResourceType, number>> = {};
    const resourcesCompletedByType: Partial<Record<ResourceType, number>> = {};
    const resourceCompletionTimeSumByType: Partial<Record<ResourceType, number>> = {};
    const resourceCompletionTimeAverageByType: Partial<Record<ResourceType, number | null>> = {};

    try {
        // --- Active Users (Use groupBy for distinct IDs, then filter bots) ---
        const DAYS_90_MS = 1000 * 60 * 60 * 24 * 90;
        const ninetyDaysAgo = new Date(new Date().getTime() - DAYS_90_MS);

        // Get distinct user IDs active in the period
        const activeUserGroups = await DbProvider.get().session.groupBy({
            by: ['user_id'],
            where: {
                last_refresh_at: {
                    gte: periodStart,
                    lte: periodEnd,
                },
            },
        });
        const activeUserIds = activeUserGroups.map(group => group.user_id);

        // Count non-bot users among the active ones
        if (activeUserIds.length > 0) {
            data.activeUsers = await DbProvider.get().user.count({
                where: {
                    id: { in: activeUserIds },
                    isBot: false // Filter out bots
                }
            });
        } else {
            data.activeUsers = 0;
        }

        // --- Teams Created ---
        data.teamsCreated = await DbProvider.get().team.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
            },
        });

        // --- Resources Created (Query 'resource' table by type) ---
        const resourceTypesToCountCreate: ResourceType[] = [
            ResourceType.Api, ResourceType.Code, ResourceType.Project, ResourceType.Routine, ResourceType.Standard // Use PascalCase
        ];
        for (const type of resourceTypesToCountCreate) {
            resourcesCreatedByType[type] = await DbProvider.get().resource.count({
                where: {
                    resourceType: type,
                    createdAt: { gte: periodStart, lte: periodEnd },
                    isDeleted: false,
                    ...(type === ResourceType.Routine || type === ResourceType.Standard ? { isInternal: false } : {}), // Use PascalCase
                },
            });
        }
        data.resourcesCreatedByType = resourcesCreatedByType;

        // --- Resources Completed & Completion Time (Query 'resource' table by type) ---
        const resourceTypesToCountComplete: ResourceType[] = [
            ResourceType.Code, ResourceType.Project, ResourceType.Routine, ResourceType.Standard // Use PascalCase
        ];

        for (const type of resourceTypesToCountComplete) {
            const count = await DbProvider.get().resource.count({
                where: {
                    resourceType: type,
                    completedAt: { gte: periodStart, lte: periodEnd },
                    isDeleted: false,
                    ...(type === ResourceType.Routine || type === ResourceType.Standard ? { isInternal: false } : {}), // Use PascalCase
                },
            });
            resourcesCompletedByType[type] = count;

            if (count > 0) {
                const result: [{ time: bigint | null }] = await DbProvider.get().$queryRaw`
                    SELECT SUM(EXTRACT(EPOCH FROM (completedAt - createdAt))) AS time
                    FROM resource
                    WHERE resourceType = ${type}
                      AND completedAt >= ${periodStart}::timestamptz
                      AND completedAt <= ${periodEnd}::timestamptz
                      AND isDeleted = false
                      ${(type === ResourceType.Routine || type === ResourceType.Standard) ? Prisma.sql`AND isInternal = false` : Prisma.empty}`; // Add isInternal filter conditionally - Use PascalCase

                const sumSeconds = result[0]?.time ? Number(result[0].time) : 0;
                resourceCompletionTimeSumByType[type] = sumSeconds;
                resourceCompletionTimeAverageByType[type] = sumSeconds / count;
            } else {
                resourceCompletionTimeAverageByType[type] = 0;
            }
        }
        data.resourcesCompletedByType = resourcesCompletedByType;
        data.resourceCompletionTimeAverageByType = resourceCompletionTimeAverageByType;

        // --- Routine Complexity/Simplicity (Query 'resource_version' for completed routines) ---
        const completedRoutinesCount = resourcesCompletedByType[ResourceType.Routine] ?? 0;
        if (completedRoutinesCount > 0) {
            const latestCompletedRoutineVersions = await DbProvider.get().resource_version.findMany({
                where: {
                    root: {
                        resourceType: ResourceType.Routine,
                        completedAt: { gte: periodStart, lte: periodEnd },
                        isDeleted: false,
                        isInternal: false,
                    },
                    isLatest: true,
                },
                select: { id: true }
            });

            const routineAggregates = await DbProvider.get().resource_version.aggregate({
                where: {
                    id: { in: latestCompletedRoutineVersions.map(v => v.id) }
                },
                _sum: {
                    complexity: true,
                    simplicity: true,
                },
            });
            data.routineComplexityAverage = (routineAggregates._sum.complexity ?? 0) / completedRoutinesCount;
            data.routineSimplicityAverage = (routineAggregates._sum.simplicity ?? 0) / completedRoutinesCount;
        } else {
            data.routineComplexityAverage = 0;
            data.routineSimplicityAverage = 0;
        }

        // --- Runs Started (Query consolidated 'run' table) ---
        data.runsStarted = await DbProvider.get().run.count({
            where: { startedAt: { gte: periodStart, lte: periodEnd } }
        });

        // --- Runs Completed & Completion Time & Context Switches (Query consolidated 'run' table) ---
        const runsCompleted = await DbProvider.get().run.count({
            where: {
                completedAt: { gte: periodStart, lte: periodEnd },
            },
        });
        data.runsCompleted = runsCompleted;

        let totalRunCompletionTimeSum = 0;
        let totalRunContextSwitchesSum = 0;

        if (runsCompleted > 0) {
            const resultRunTime: [{ time: bigint | null }] = await DbProvider.get().$queryRaw`
                SELECT SUM(EXTRACT(EPOCH FROM (completedAt - startedAt))) AS time
                FROM run
                WHERE completedAt >= ${periodStart}::timestamptz AND completedAt <= ${periodEnd}::timestamptz`;
            totalRunCompletionTimeSum = resultRunTime[0]?.time ? Number(resultRunTime[0].time) : 0;

            const resultRunSwitches = await DbProvider.get().run.aggregate({
                where: {
                    completedAt: { gte: periodStart, lte: periodEnd },
                },
                _sum: {
                    contextSwitches: true,
                },
            });
            totalRunContextSwitchesSum = resultRunSwitches._sum.contextSwitches ?? 0;
        }

        data.runCompletionTimeAverage = data.runsCompleted > 0 ? totalRunCompletionTimeSum / data.runsCompleted : 0;
        data.runContextSwitchesAverage = data.runsCompleted > 0 ? totalRunContextSwitchesSum / data.runsCompleted : 0;

        // --- Verified Credentials ---
        data.verifiedEmailsCreated = await DbProvider.get().email.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                verifiedAt: { not: null },
            },
        });
        data.verifiedWalletsCreated = await DbProvider.get().wallet.count({
            where: {
                createdAt: { gte: periodStart, lte: periodEnd },
                verifiedAt: { not: null },
            },
        });

        // --- Store in Database ---
        await DbProvider.get().stats_site.create({ data });

        logger.info("logSiteStats completed successfully", { periodType, periodStart, periodEnd, trace: "0422" });

    } catch (error) {
        // Enhanced error logging
        logger.error("logSiteStats encountered an error", {
            error: error instanceof Error ? { message: error.message, stack: error.stack } : error,
            periodType,
            periodStart,
            periodEnd,
            calculatedData: data,
            trace: "0423"
        });
    }
};
