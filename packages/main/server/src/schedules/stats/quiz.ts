import { QuizAttemptStatus } from "@local/consts";
import pkg, { PeriodType } from "@prisma/client";
import { logger } from "../../events";
import { PrismaType } from "../../types";

const { PrismaClient } = pkg;

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
): Promise<BatchDirectoryAttemptCountsResult> => {
    // Initialize return value
    const result: BatchDirectoryAttemptCountsResult = Object.fromEntries(quizIds.map(id => [id, {
        timesStarted: 0,
        timesPassed: 0,
        timesFailed: 0,
        scoreAverage: 0,
        completionTimeAverage: 0,
    }]));
    const batchSize = 100;
    let skip = 0;
    let currentBatchSize = 0;
    do {
        // Find all attempts associated with the quizzes
        const batch = await prisma.quiz_attempt.findMany({
            where: {
                quiz: { id: { in: quizIds } },
                OR: [
                    { created_at: { gte: periodStart, lte: periodEnd } },
                    { updated_at: { gte: periodStart, lte: periodEnd } },
                ],
            },
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
            skip,
            take: batchSize,
        });
        // Increment skip
        skip += batchSize;
        // Update current batch size
        currentBatchSize = batch.length;
        // For each attempt, increment the counts for the quiz
        batch.forEach(run => {
            const quizId = run.quiz?.id;
            if (!quizId || !result[quizId]) { return; }
            // If created_at within period and status is not NotStarted, increment timesStarted
            if (run.created_at !== null && new Date(run.created_at) >= new Date(periodStart) && run.status !== QuizAttemptStatus.NotStarted) {
                result[quizId].timesStarted += 1;
            }
            // If updated_at within period and status is either Passed or Failed, increment timesPassed or timesFailed
            // and update averages
            if (run.updated_at !== null && new Date(run.updated_at) >= new Date(periodStart) && [QuizAttemptStatus.Passed, QuizAttemptStatus.Failed].includes(run.status as any)) {
                result[quizId].timesPassed += run.status === QuizAttemptStatus.Passed ? 1 : 0;
                result[quizId].timesFailed += run.status === QuizAttemptStatus.Failed ? 1 : 0;
                result[quizId].scoreAverage += run.pointsEarned;
                if (run.timeTaken !== null) result[quizId].completionTimeAverage += run.timeTaken;
            }
        });
    } while (currentBatchSize === batchSize);
    // For the averages, divide by the number of attempts completed
    Object.keys(result).forEach(quizId => {
        if (result[quizId].timesFailed > 0 || result[quizId].timesPassed > 0) {
            result[quizId].scoreAverage /= (result[quizId].timesFailed + result[quizId].timesPassed);
            result[quizId].completionTimeAverage /= (result[quizId].timesFailed + result[quizId].timesPassed);
        }
    });
    return result;
};

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
) => {
    // Initialize the Prisma client
    const prisma = new PrismaClient();
    try {
        // We may be dealing with a lot of data, so we need to do this in batches
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            // Find all quizzes with at least one attempt
            const batch = await prisma.quiz.findMany({
                where: {
                    attempts: {
                        some: {}, // This is empty on purpose - we don't care about the attempts yet, just that at least one exists
                    },
                },
                select: {
                    id: true,
                },
                skip,
                take: batchSize,
            });
            // Increment skip
            skip += batchSize;
            // Update current batch size
            currentBatchSize = batch.length;
            // Find and count all attempts associated with the quizzes, which 
            // have been started or completed within the period
            const attemptCountsByQuiz = await batchAttemptCounts(prisma, batch.map(quiz => quiz.id), periodStart, periodEnd);
            // Create stats for each routine
            await prisma.stats_quiz.createMany({
                data: batch.map(quiz => ({
                    quizId: quiz.id,
                    periodStart,
                    periodEnd,
                    periodType,
                    timesStarted: attemptCountsByQuiz[quiz.id].timesStarted,
                    timesPassed: attemptCountsByQuiz[quiz.id].timesPassed,
                    timesFailed: attemptCountsByQuiz[quiz.id].timesFailed,
                    scoreAverage: attemptCountsByQuiz[quiz.id].scoreAverage,
                    completionTimeAverage: attemptCountsByQuiz[quiz.id].completionTimeAverage,
                })),
            });
        } while (currentBatchSize === batchSize);
    } catch (error) {
        logger.error("Caught error logging quiz statistics", { trace: "0421", periodType, periodStart, periodEnd });
    } finally {
        // Close the Prisma client
        await prisma.$disconnect();
    }
};
