import pkg, { PeriodType, Prisma } from "@prisma/client";
import { logger } from "../../events";
import { PrismaType } from "../../types";
import { batchCollect } from "../../utils/batchCollect";

const { PrismaClient } = pkg;

type BatchRunRoutinesResult = Record<string, {
    runRoutinesStarted: number;
    runRoutinesCompleted: number;
    runRoutineCompletionTimeAverage: number;
    runRoutineContextSwitchesAverage: number;
}>

/**
 * Batch collects run routine stats for a list of organizations
 * @param prisma The Prisma client
 * @param organizationIds The IDs of the organizations to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of organization IDs to run routine stats
 */
const batchRunRoutines = async (
    prisma: PrismaType,
    organizationIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunRoutinesResult> => {
    // Initialize return value
    const result: BatchRunRoutinesResult = Object.fromEntries(organizationIds.map(id => [id, {
        runRoutinesStarted: 0,
        runRoutinesCompleted: 0,
        runRoutineCompletionTimeAverage: 0,
        runRoutineContextSwitchesAverage: 0,
    }]));
    await batchCollect<Prisma.run_routineFindManyArgs>({
        objectType: "RunRoutine",
        prisma,
        processData: async (batch) => {
            // For each run, increment the counts for the routine version
            batch.forEach(run => {
                const organizationId = run.organization?.id;
                if (!organizationId || !result[organizationId]) { return; }
                // If runStarted within period, increment runsStarted
                if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                    result[organizationId].runRoutinesStarted += 1;
                }
                // If runCompleted within period, increment runsCompleted 
                // and update averages
                if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                    result[organizationId].runRoutinesCompleted += 1;
                    if (run.timeElapsed !== null) result[organizationId].runRoutineCompletionTimeAverage += run.timeElapsed;
                    result[organizationId].runRoutineContextSwitchesAverage += run.contextSwitches;
                }
            });
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
        where: {
            organization: { id: { in: organizationIds } },
            OR: [
                { startedAt: { gte: periodStart, lte: periodEnd } },
                { completedAt: { gte: periodStart, lte: periodEnd } },
            ],
        },
    });
    // For the averages, divide by the number of runs completed
    Object.keys(result).forEach(organizationId => {
        if (result[organizationId].runRoutinesCompleted > 0) {
            result[organizationId].runRoutineCompletionTimeAverage /= result[organizationId].runRoutinesCompleted;
            result[organizationId].runRoutineContextSwitchesAverage /= result[organizationId].runRoutinesCompleted;
        }
    });
    return result;
};

/**
 * Creates periodic stats for all organizations
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logOrganizationStats = async (
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) => {
    const prisma = new PrismaClient();
    try {
        await batchCollect<Prisma.organizationFindManyArgs>({
            objectType: "Organization",
            prisma,
            processData: async (batch) => {
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
            },
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
        });
    } catch (error) {
        logger.error("Caught error logging organization statistics", { trace: "0419", periodType, periodStart, periodEnd });
    } finally {
        await prisma.$disconnect();
    }
};
