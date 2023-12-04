import { PrismaType, logger, withPrisma } from "@local/server";
import { getReactionScore } from "@local/shared";
import { camelCase } from "lodash";

/**
 * Processes reactions for a single object, calculating the total score and building a map of reaction summaries.
 * 
 * @param prisma - The Prisma client instance.
 * @param tableName - The name of the table containing the object.
 * @param objectId - The unique identifier of the object.
 * @returns An object containing the total score and a map of reaction summaries.
 */
const processReactionsForObject = async (
    prisma: PrismaType,
    tableName: string,
    objectId: string,
): Promise<{ totalScore: number, reactionSummaries: Map<string, number> }> => {
    let totalScore = 0;
    const reactionSummaries = new Map<string, number>();
    let reactionSkip = 0;
    const reactionBatchSize = 100;
    let continueProcessing = true;

    while (continueProcessing) {
        const reactions = await prisma.reaction.findMany({
            where: { [`${camelCase(tableName)}Id`]: objectId },
            skip: reactionSkip,
            take: reactionBatchSize,
            select: {
                emoji: true,
            },
        });

        if (reactions.length === 0) {
            continueProcessing = false;
            break;
        }

        reactions.forEach(reaction => {
            totalScore += getReactionScore(reaction.emoji);
            reactionSummaries.set(reaction.emoji, (reactionSummaries.get(reaction.emoji) || 0) + 1);
        });

        reactionSkip += reactionBatchSize;
    }

    return { totalScore, reactionSummaries };
};

/**
 * Represents an operation to update a reaction summary (create, update, or delete).
 */
type ReactionSummaryUpdateOperation = {
    emoji: string;
    count?: number; // Present for create and update
    create?: boolean; // Flag to indicate a create operation
    delete?: boolean; // Flag to indicate a delete operation
};

/**
 * Updates reaction data for all objects in a specified table.
 * 
 * @param prisma - The Prisma client instance.
 * @param tableName - The name of the table to process.
 */
const updateReactionsForTable = async (prisma: PrismaType, tableName: string): Promise<void> => {
    const objectBatchSize = 100;
    let objectSkip = 0;
    let continueProcessing = true;

    while (continueProcessing) {
        const objects = await prisma[tableName].findMany({
            skip: objectSkip,
            take: objectBatchSize,
            select: {
                id: true,
                score: true,
                reactionSummaries: {
                    select: {
                        id: true,
                        emoji: true,
                        count: true,
                    },
                },
            },
        });

        if (objects.length === 0) {
            continueProcessing = false;
            break;
        }

        // Process reactions for each object
        for (const object of objects) {
            const { totalScore, reactionSummaries } = await processReactionsForObject(prisma, tableName, object.id);

            // Compare with existing data and prepare update operations
            const existingSummaries = Object.fromEntries(object.reactionSummaries.map(s => [s.emoji, s.count]));
            const updates: ReactionSummaryUpdateOperation[] = [];
            let isReactionSummaryMismatch = false;

            // Check for updates or creates
            reactionSummaries.forEach((count, emoji) => {
                if (existingSummaries[emoji]) {
                    if (existingSummaries[emoji] !== count) {
                        updates.push({ emoji, count });
                        isReactionSummaryMismatch = true;
                    }
                } else {
                    updates.push({ emoji, count, create: true });
                    isReactionSummaryMismatch = true;
                }
            });

            // Check for deletes
            Object.keys(existingSummaries).forEach(emoji => {
                if (!reactionSummaries.has(emoji)) {
                    updates.push({ emoji, delete: true });
                    isReactionSummaryMismatch = true;
                }
            });

            const isScoreMismatch = object.score !== totalScore;

            if (isScoreMismatch || isReactionSummaryMismatch) {
                logger.warn(`Updating reactions for ${tableName} ${object.id}. Score mismatch: ${isScoreMismatch}, Reaction summary mismatch: ${isReactionSummaryMismatch}`, { trace: "0171" });

                // Construct the update data, including nested writes for reactionSummaries
                const updateData = {
                    score: totalScore,
                    reactionSummaries: {
                        // Use Prisma's nested write syntax for create, update, delete
                        create: updates.filter(u => u.create).map(u => ({ emoji: u.emoji, count: u.count })),
                        update: updates.filter(u => !u.create && !u.delete).map(u => ({ where: { emoji: u.emoji }, data: { count: u.count } })),
                        delete: updates.filter(u => u.delete).map(u => ({ emoji: u.emoji })),
                    },
                };

                await prisma[tableName].update({
                    where: { id: object.id },
                    data: updateData,
                });
            }
        }

        objectSkip += objectBatchSize;
    }
};

export const countReacts = async (): Promise<void> => {
    const tableNames = [
        "api",
        "chat_message",
        "comment",
        "issue",
        "note",
        "post",
        "project",
        "question",
        "question_answer",
        "quiz",
        "routine",
        "smart_contract",
        "standard",
    ];

    await withPrisma({
        process: async (prisma) => {
            for (const tableName of tableNames) {
                await updateReactionsForTable(prisma, tableName);
            }
        },
        trace: "0169",
        traceObject: { tableNames },
    });
};
