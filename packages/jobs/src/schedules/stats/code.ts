import { batch, logger, prismaInstance } from "@local/server";
import { PeriodType, Prisma } from "@prisma/client";

/**
 * Creates periodic stats for all codes in use
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logCodeStats = async (
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) => {
    try {
        await batch<Prisma.code_versionFindManyArgs>({
            objectType: "CodeVersion",
            processBatch: async (batch) => {
                await prismaInstance.stats_code.createMany({
                    data: batch.map(codeVersion => ({
                        codeId: codeVersion.root.id,
                        periodStart,
                        periodEnd,
                        periodType,
                        calls: 0, //TODO no way to track calls yet
                        routineVersions: codeVersion._count.calledByRoutineVersions,
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
        logger.error("logCodeStats caught error", { error, trace: "0103", periodType, periodStart, periodEnd });
    }
};
