import pkg from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
export const logStandardStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.standard_version.findMany({
                where: {
                    routineVersionInputs: {
                        some: {},
                    },
                    routineVersionOutputs: {
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
                        select: {
                            routineVersionInputs: true,
                            routineVersionOutputs: true,
                        },
                    },
                },
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            await prisma.stats_standard.createMany({
                data: batch.map(standardVersion => ({
                    standardId: standardVersion.root.id,
                    periodStart,
                    periodEnd,
                    periodType,
                    linksToInputs: standardVersion._count.routineVersionInputs,
                    linksToOutputs: standardVersion._count.routineVersionOutputs,
                })),
            });
        } while (currentBatchSize === batchSize);
    }
    catch (error) {
        logger.error("Caught error logging standard statistics", { trace: "0425", periodType, periodStart, periodEnd });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=standard.js.map