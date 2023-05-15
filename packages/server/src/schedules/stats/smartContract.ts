import { PeriodType, Prisma } from "@prisma/client";
import { batch } from "../../utils/batch";

/**
 * Creates periodic stats for all smart contracts in use
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logSmartContractStats = async (
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) => await batch<Prisma.smart_contract_versionFindManyArgs>({
    objectType: "SmartContract",
    processBatch: async (batch, prisma) => {
        await prisma.stats_smart_contract.createMany({
            data: batch.map(smartContractVersion => ({
                smartContractId: smartContractVersion.root.id,
                periodStart,
                periodEnd,
                periodType,
                calls: 0, //TODO no way to track calls yet
                routineVersions: smartContractVersion._count.calledByRoutineVersions,
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
    trace: "0424",
    traceObject: { periodType, periodStart, periodEnd },
    where: {
        calledByRoutineVersions: {
            some: {}, // This is empty on purpose - we don't care about the routine version, just that at least one exists
        },
        isDeleted: false,
        isLatest: true,
        root: { isDeleted: false },
    },
});
