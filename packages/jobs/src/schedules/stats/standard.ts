import { batch, logger, prismaInstance } from "@local/server";
import { PeriodType, Prisma } from "@prisma/client";

/**
 * Creates periodic stats for all standard in use
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export async function logStandardStats(
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) {
    try {
        await batch<Prisma.standard_versionFindManyArgs>({
            objectType: "StandardVersion",
            processBatch: async (batch) => {
                await prismaInstance.stats_standard.createMany({
                    data: batch.map(standardVersion => ({
                        standardId: standardVersion.root.id,
                        periodStart,
                        periodEnd,
                        periodType,
                        linksToInputs: standardVersion._count.routineVersionInputs,
                        linksToOutputs: standardVersion._count.routineVersionOutputs,
                    })),
                });
            },
            select: {
                id: true,
                root: {
                    select: { id: true },
                },
                _count: {
                    select: {
                        routineVersionInputs: true,
                        routineVersionOutputs: true,
                    },
                },
            },
            where: {
                routineVersionInputs: {
                    some: {}, // This is empty on purpose - we don't care about the routine version, just that at least one exists
                },
                routineVersionOutputs: {
                    some: {}, // This is empty on purpose - we don't care about the routine version, just that at least one exists
                },
                isDeleted: false,
                isLatest: true,
                root: { isDeleted: false },
            },
        });
    } catch (error) {
        logger.error("logStandardStats caught error", { error, trace: "0424", periodType, periodStart, periodEnd });
    }
};
