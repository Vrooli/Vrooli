import { QuizAttemptStatus } from "@local/shared";
import { PeriodType, Prisma } from "@prisma/client";
import { PrismaType } from "../../types";
import { batch } from "../../utils/batch";
import { batchGroup } from "../../utils/batchGroup";

type BatchDirectoryAttemptCountsResult = Record<string, {
    timesStarted: number;
    timesPassed: number;
    timesFailed: number;
    scoreAverage: number;
    completionTimeAverage: number;
}>

/**
 * Batch collects attempt counts for a list of quizzes
 * @param prisma The Prisma client
 * @param quizIds The IDs of the quizzes to collect attempt counts for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 * @returns A map of quiz IDs to various attempt counts
 */
const batchAttemptCounts = async (
    prisma: PrismaType,
    quizIds: string[],
    periodStart: string,
    periodEnd: string,
): Promise<BatchDirectoryAttemptCountsResult> => batchGroup<Prisma.quiz_attemptFindManyArgs, BatchDirectoryAttemptCountsResult>({
    initialResult: Object.fromEntries(quizIds.map(id => [id, {
        timesStarted: 0,
        timesPassed: 0,
        timesFailed: 0,
        scoreAverage: 0,
        completionTimeAverage: 0,
    }])),
    processBatch: async (batch, result) => {
        // For each attempt, increment the counts for the quiz
        batch.forEach(run => {
            const quizId = run.quiz?.id;
            if (!quizId) return;
            const currResult = result[quizId];
            if (!currResult) return;
            // If created_at within period and status is not NotStarted, increment timesStarted
            if (run.created_at !== null && new Date(run.created_at) >= new Date(periodStart) && run.status !== QuizAttemptStatus.NotStarted) {
                currResult.timesStarted += 1;
            }
            // If updated_at within period and status is either Passed or Failed, increment timesPassed or timesFailed
            // and update averages
            if (run.updated_at !== null && new Date(run.updated_at) >= new Date(periodStart) && [QuizAttemptStatus.Passed, QuizAttemptStatus.Failed].includes(run.status as any)) {
                currResult.timesPassed += run.status === QuizAttemptStatus.Passed ? 1 : 0;
                currResult.timesFailed += run.status === QuizAttemptStatus.Failed ? 1 : 0;
                currResult.scoreAverage += run.pointsEarned;
                if (run.timeTaken !== null) currResult.completionTimeAverage += run.timeTaken;
            }
        });
    },
    finalizeResult: (result) => {
        // For the averages, divide by the number of attempts completed
        Object.keys(result).forEach(quizId => {
            const currResult = result[quizId];
            if (!currResult) return;
            if (currResult.timesFailed > 0 || currResult.timesPassed > 0) {
                currResult.scoreAverage /= (currResult.timesFailed + currResult.timesPassed);
                currResult.completionTimeAverage /= (currResult.timesFailed + currResult.timesPassed);
            }
        });
        return result;
    },
    objectType: "QuizAttempt",
    prisma,
    select: {
        id: true,
        created_at: true,
        updated_at: true,
        pointsEarned: true,
        quiz: {
            select: { id: true },
        },
        status: true,
        timeTaken: true,
    },
    where: {
        quiz: { id: { in: quizIds } },
        OR: [
            { created_at: { gte: periodStart, lte: periodEnd } },
            { updated_at: { gte: periodStart, lte: periodEnd } },
        ],
    },
});

/**
 * Creates periodic stats for all routines
 * @param periodType The type of period to create stats for
 * @param periodStart When the period started
 * @param periodEnd When the period ended
 */
export const logQuizStats = async (
    periodType: PeriodType,
    periodStart: string,
    periodEnd: string,
) => await batch<Prisma.quizFindManyArgs>({
    objectType: "Quiz",
    processBatch: async (batch, prisma) => {
        // Find and count all attempts associated with the quizzes, which 
        // have been started or completed within the period
        const attemptCountsByQuiz = await batchAttemptCounts(prisma, batch.map(quiz => quiz.id), periodStart, periodEnd);
        // Create stats for each routine
        await prisma.stats_quiz.createMany({
            data: batch.map(quiz => {
                const attemptCounts = attemptCountsByQuiz[quiz.id];
                if (!attemptCounts) return;
                return {
                    quizId: quiz.id,
                    periodStart,
                    periodEnd,
                    periodType,
                    ...attemptCounts,
                };
            }).filter((data): data is Exclude<typeof data, undefined> => !!data),
        });
    },
    select: {
        id: true,
    },
    trace: "0421",
    traceObject: { periodType, periodStart, periodEnd },
    where: {
        attempts: {
            some: {}, // This is empty on purpose - we don't care about the attempts yet, just that at least one exists
        },
    },
});
