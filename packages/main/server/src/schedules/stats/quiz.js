import { QuizAttemptStatus } from "@local/consts";
import pkg from "@prisma/client";
import { logger } from "../../events";
const { PrismaClient } = pkg;
const batchAttemptCounts = async (prisma, quizIds, periodStart, periodEnd) => {
    const result = Object.fromEntries(quizIds.map(id => [id, {
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
        skip += batchSize;
        currentBatchSize = batch.length;
        batch.forEach(run => {
            const quizId = run.quiz?.id;
            if (!quizId || !result[quizId]) {
                return;
            }
            if (run.created_at !== null && new Date(run.created_at) >= new Date(periodStart) && run.status !== QuizAttemptStatus.NotStarted) {
                result[quizId].timesStarted += 1;
            }
            if (run.updated_at !== null && new Date(run.updated_at) >= new Date(periodStart) && [QuizAttemptStatus.Passed, QuizAttemptStatus.Failed].includes(run.status)) {
                result[quizId].timesPassed += run.status === QuizAttemptStatus.Passed ? 1 : 0;
                result[quizId].timesFailed += run.status === QuizAttemptStatus.Failed ? 1 : 0;
                result[quizId].scoreAverage += run.pointsEarned;
                if (run.timeTaken !== null)
                    result[quizId].completionTimeAverage += run.timeTaken;
            }
        });
    } while (currentBatchSize === batchSize);
    Object.keys(result).forEach(quizId => {
        if (result[quizId].timesFailed > 0 || result[quizId].timesPassed > 0) {
            result[quizId].scoreAverage /= (result[quizId].timesFailed + result[quizId].timesPassed);
            result[quizId].completionTimeAverage /= (result[quizId].timesFailed + result[quizId].timesPassed);
        }
    });
    return result;
};
export const logQuizStats = async (periodType, periodStart, periodEnd) => {
    const prisma = new PrismaClient();
    try {
        const batchSize = 100;
        let skip = 0;
        let currentBatchSize = 0;
        do {
            const batch = await prisma.quiz.findMany({
                where: {
                    attempts: {
                        some: {},
                    },
                },
                select: {
                    id: true,
                },
                skip,
                take: batchSize,
            });
            skip += batchSize;
            currentBatchSize = batch.length;
            const attemptCountsByQuiz = await batchAttemptCounts(prisma, batch.map(quiz => quiz.id), periodStart, periodEnd);
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
    }
    catch (error) {
        logger.error("Caught error logging quiz statistics", { trace: "0421", periodType, periodStart, periodEnd });
    }
    finally {
        await prisma.$disconnect();
    }
};
//# sourceMappingURL=quiz.js.map