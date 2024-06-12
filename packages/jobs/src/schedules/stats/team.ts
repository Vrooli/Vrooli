import { batch, batchGroup, logger, prismaInstance } from "@local/server";
import { PeriodType, Prisma } from "@prisma/client";

type BatchRunRoutinesResult = Record<string, {
    runRoutinesStarted: number;
    runRoutinesCompleted: number;
    runRoutineCompletionTimeAverage: number;
    runRoutineContextSwitchesAverage: number;
}>

/**
 * Batch collects run routine stats for a list of teams
 * @param teamIds The IDs of the teams to collect stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of team IDs to run routine stats
 */
const batchRunRoutines = async (
    teamIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchRunRoutinesResult> => {
    const initialResult = Object.fromEntries(teamIds.map(id => [id, {
        runRoutinesStarted: 0,
        runRoutinesCompleted: 0,
        runRoutineCompletionTimeAverage: 0,
        runRoutineContextSwitchesAverage: 0,
    }]));
    try {
        return await batchGroup<Prisma.run_routineFindManyArgs, BatchRunRoutinesResult>({
            initialResult,
            processBatch: async (batch, result) => {
                // For each run, increment the counts for the routine version
                batch.forEach(run => {
                    const teamId = run.team?.id;
                    if (!teamId) return;
                    const currResult = result[teamId];
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
                Object.keys(result).forEach(teamId => {
                    const currResult = result[teamId];
                    if (!currResult) return;
                    if (currResult.runRoutinesCompleted > 0) {
                        currResult.runRoutineCompletionTimeAverage /= currResult.runRoutinesCompleted;
                        currResult.runRoutineContextSwitchesAverage /= currResult.runRoutinesCompleted;
                    }
                });
                return result;
            },
            objectType: "RunRoutine",
            select: {
                id: true,
                team: {
                    select: { id: true },
                },
                completedAt: true,
                contextSwitches: true,
                startedAt: true,
                timeElapsed: true,
            },
            where: {
                team: { id: { in: teamIds } },
                OR: [
                    { startedAt: { gte: periodStart, lte: periodEnd } },
                    { completedAt: { gte: periodStart, lte: periodEnd } },
                ],
            },
        });
    } catch (error) {
        logger.error("batchRunRoutines caught error", { error });
    }
    return initialResult;
};

/**
 * Creates periodic stats for all teams
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logTeamStats = async (
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) => {
    try {
        await batch<Prisma.teamFindManyArgs>({
            objectType: "Team",
            processBatch: async (batch) => {
                const runRoutineStats = await batchRunRoutines(batch.map(team => team.id), periodStart, periodEnd);
                await prismaInstance.stats_team.createMany({
                    data: batch.map(team => ({
                        teamId: team.id,
                        periodStart,
                        periodEnd,
                        periodType,
                        ...team._count,
                        ...runRoutineStats[team.id],
                    })),
                });
            },
            select: {
                id: true,
                _count: {
                    select: {
                        apis: true,
                        codes: true,
                        members: true,
                        notes: true,
                        projects: true,
                        routines: true,
                        standards: true,
                    },
                },
            },
        });
    } catch (error) {
        logger.error("logTeamStats caught error", { error, trace: "0419", periodType, periodStart, periodEnd });
    }
};
