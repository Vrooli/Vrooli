import { PeriodType, Prisma } from "@prisma/client";
import { PrismaType } from "../../types";
import { batch } from "../../utils/batch";
import { batchGroup } from "../../utils/batchGroup";

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
): Promise<BatchRunRoutinesResult> => batchGroup<Prisma.run_routineFindManyArgs, BatchRunRoutinesResult>({
    initialResult: Object.fromEntries(organizationIds.map(id => [id, {
        runRoutinesStarted: 0,
        runRoutinesCompleted: 0,
        runRoutineCompletionTimeAverage: 0,
        runRoutineContextSwitchesAverage: 0,
    }])),
    processBatch: async (batch, result) => {
        // For each run, increment the counts for the routine version
        batch.forEach(run => {
            const organizationId = run.organization?.id;
            if (!organizationId) return;
            const currResult = result[organizationId];
            if (!currResult) return;
            // If runStarted within period, increment runsStarted
            if (run.startedAt !== null && new Date(run.startedAt) >= new Date(periodStart)) {
                currResult.runRoutinesStarted += 1;
            }
            // If runCompleted within period, increment runsCompleted 
            // and update averages
            if (run.completedAt !== null && new Date(run.completedAt) >= new Date(periodStart)) {
                currResult.runRoutinesCompleted += 1;
                if (run.timeElapsed !== null) currResult.runRoutineCompletionTimeAverage += run.timeElapsed;
                currResult.runRoutineContextSwitchesAverage += run.contextSwitches;
            }
        });
    },
    finalizeResult: (result) => {
        // For the averages, divide by the number of runs completed
        Object.keys(result).forEach(organizationId => {
            const currResult = result[organizationId];
            if (!currResult) return;
            if (currResult.runRoutinesCompleted > 0) {
                currResult.runRoutineCompletionTimeAverage /= currResult.runRoutinesCompleted;
                currResult.runRoutineContextSwitchesAverage /= currResult.runRoutinesCompleted;
            }
        });
        return result;
    },
    objectType: "RunRoutine",
    prisma,
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
) => await batch<Prisma.organizationFindManyArgs>({
    objectType: "Organization",
    processBatch: async (batch, prisma) => {
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
    trace: "0419",
    traceObject: { periodType, periodStart, periodEnd },
});
