import pkg from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
const batchRunRoutines = async (prisma, organizationIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(organizationIds.map(id => [id, {
            runRoutinesStarted: 0,
            runRoutinesCompleted: 0,
            runRoutineCompletionTimeAverage: 0,
            runRoutineContextSwitchesAverage: 0,
        }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        const batch = await prisma.run_routine.findMany({
            where: {
                organization: { id: { in: organizationIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
            select: {
                id: true,
                organization: {
                    select: { id: true },
                },
                completedAt: true,
                contextSwitches: true,
                startedAt: true,
                timeElapsed: true,
            },
            skip,
            take: batchSize,
        });
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(run => {
            const organizationId = run.organization?.id;
            if (!organizationId || !result[organizationId]) {
                return;
            }
            if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                result[organizationId].runRoutinesStarted += 1;
            }
            if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                result[organizationId].runRoutinesCompleted += 1;
                if (run.timeElapsed !== null)
                    result[organizationId].runRoutineCompletionTimeAverage += run.timeElapsed;
                result[organizationId].runRoutineContextSwitchesAverage += run.contextSwitches;
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(organizationId => {
        if (result[organizationId].runRoutinesCompleted > 0) {
            result[organizationId].runRoutineCompletionTimeAverage /= result[organizationId].runRoutinesCompleted;
            result[organizationId].runRoutineContextSwitchesAverage /= result[organizationId].runRoutinesCompleted;
        }
    });
    return result;
};
export const logOrganizationStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.organization.findMany({
                select: {
                    id: true,
                    _count: {
                        select: {
                            apis: true,
                            members: true,
                            notes: true,
                            projects: true,
                            routines: true,
                            smartContracts: true,
                            standards: true,
                        },
                    },
                },
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            const runRoutineStats = await batchRunRoutines(prisma, batch.map(organization => organization.id), periodStart, periodEnd);
            await prisma.stats_organization.createMany({
                data: batch.map(organization => ({
                    organizationId: organization.id,
                    periodStart,
                    periodEnd,
                    periodType,
                    ...organization._count,
                    ...runRoutineStats[organization.id],
                })),
            });
        } while (currentBatchSize === batchSize);
    }
    catch (error) {
        logger.error("Caught error logging organization statistics", { trace: "0419", periodType, periodStart, periodEnd });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=organization.js.map