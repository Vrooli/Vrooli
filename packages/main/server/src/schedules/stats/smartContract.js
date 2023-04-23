import pkg from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
export const logSmartContractStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.smart_contract_version.findMany({
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
            await prisma.stats_smart_contract.createMany({
                data: batch.map(smartContractVersion => ({
                    smartContractId: smartContractVersion.root.id,
                    periodStart,
                    periodEnd,
                    periodType,
                    calls: 0,
                    routineVersions: smartContractVersion._count.calledByRoutineVersions,
                })),
            });
        } while (currentBatchSize === batchSize);
    }
    catch (error) {
        logger.error("Caught error logging smart contract statistics", { trace: "0424", periodType, periodStart, periodEnd });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=smartContract.js.map