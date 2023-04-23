import pkg from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
export const logApiStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.api_version.findMany({
                where: {
                    calledByRoutineVersions: {
                        some: {},
                    },
                    isDeleted: false,
                    isLatest: true,
                    root: { isDeleted: false },
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
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            await prisma.stats_api.createMany({
                data: batch.map(apiVersion => ({
                    apiId: apiVersion.root.id,
                    periodStart,
                    periodEnd,
                    periodType,
                    calls: 0,
                    routineVersions: apiVersion._count.calledByRoutineVersions,
                })),
            });
        } while (currentBatchSize === batchSize);
    }
    catch (error) {
        logger.error("Caught error logging api statistics", { trace: "0418", periodType, periodStart, periodEnd });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=api.js.map