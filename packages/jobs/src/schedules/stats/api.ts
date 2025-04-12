import { DbProvider, batch, logger } from "@local/server";
import { PeriodType, Prisma } from "@prisma/client";

/**
 * Creates periodic stats for all apis in use
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export async function logApiStats(
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) {
    try {
        await batch<Prisma.api_versionFindManyArgs>({
            objectType: "ApiVersion",
            processBatch: async (batch) => {
                await DbProvider.get().stats_api.createMany({
                    data: batch.map(apiVersion => ({
                        apiId: apiVersion.root.id,
                        periodStart,
                        periodEnd,
                        periodType,
                        calls: 0, //TODO no way to track calls yet
                        routineVersions: apiVersion._count.calledByRoutineVersions,
                    })),
                });
            },
            select: {
                id: true,
                root: {
                    select: { id: true },
                },
                _count: {
                    select: { calledByRoutineVersions: true },
                },
            },
            where: {
                calledByRoutineVersions: {
                    some: {}, // This is empty on purpose - we don't care about the routine version, just that at least one exists
                },
                isDeleted: false,
                isLatest: true,
                root: { isDeleted: false },
            },
        });
    } catch (error) {
        logger.error("logApiStats caught error", { error, trace: "0418", periodType, periodStart, periodEnd });
    }
};
