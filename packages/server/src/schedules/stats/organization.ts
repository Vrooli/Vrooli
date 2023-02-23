import pkg, { PeriodType } from '@prisma/client';
import { PrismaType } from '../../types';
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
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all runs associated with the organizations
        const batch = await prisma.run_routine.findMany({
            where: {
                organization: { id: { in: organizationIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ]
            },
            select: {
                id: true,
                organization: {
                    select: { id: true }
                },
                completedAt: true,
                contextSwitches: true,
                startedAt: true,
                timeElapsed: true,
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each run, increment the counts for the routine version
        batch.forEach(run => {
            const organizationId = run.organization?.id;
            if (!organizationId || !result[organizationId]) { return }
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
    } while (currentBatchSize === batchSize);
    // For the averages, divide by the number of runs completed
    Object.keys(result).forEach(organizationId => {
        if (result[organizationId].runRoutinesCompleted > 0) {
            result[organizationId].runRoutineCompletionTimeAverage /= result[organizationId].runRoutinesCompleted;
            result[organizationId].runRoutineContextSwitchesAverage /= result[organizationId].runRoutinesCompleted;
        }
    });
    return result;
}

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
    // Initialize the Prisma client
    const prisma = new PrismaClient();
    // We may be dealing with a lot of data, so we need to do this in batches
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all organizations
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
                    }
                }
            },
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // Batch collect run stats
        const runRoutineStats = await batchRunRoutines(prisma, batch.map(organization => organization.id), periodStart, periodEnd);
        // Create stats for each organization
        await prisma.stats_organization.createMany({
            data: batch.map(organization => ({
                organizationId: organization.id,
                periodStart,
                periodEnd,
                periodType,
                ...organization._count,
                ...runRoutineStats[organization.id],
            }))
        });
    } while (currentBatchSize === batchSize);
    // Close the Prisma client
    await prisma.$disconnect();
}