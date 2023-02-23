import pkg, { PeriodType } from '@prisma/client';
const { PrismaClient } = pkg;

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
) => {
    // Initialize the Prisma client
    const prisma = new PrismaClient();
    // We may be dealing with a lot of data, so we need to do this in batches
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all latest (so should only be associated with one smart contract) smart contract versions that are used by at least one routine
        const batch = await prisma.smart_contract_version.findMany({
            where: {
                calledByRoutineVersions: {
                    some: {} // This is empty on purpose - we don't care about the routine version, just that at least one exists
                },
                isDeleted: false,
                isLatest: true,
                root: { isDeleted: false },
            },
            select: {
                id: true,
                root: {
                    select: { id: true }
                },
                _count: {
                    select: { calledByRoutineVersions: true }
                }
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // Create stats for each smart contract
        await prisma.stats_smart_contract.createMany({
            data: batch.map(smartContractVersion => ({
                smartContractId: smartContractVersion.root.id,
                periodStart,
                periodEnd,
                periodType,
                calls: 0, //TODO no way to track calls yet
                routineVersions: smartContractVersion._count.calledByRoutineVersions,
            }))
        });
    } while (currentBatchSize === batchSize);
    // Close the Prisma client
    await prisma.$disconnect();
}