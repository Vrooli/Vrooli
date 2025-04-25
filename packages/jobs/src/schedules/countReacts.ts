import { DbProvider, FindManyArgs, batch, logger } from "@local/server";
import { ModelType, camelCase, getReactionScore, uppercaseFirstLetter } from "@local/shared";

/**
 * Processes reactions for a single object, calculating the total score and building a map of reaction summaries.
 * 
 * @param tableName - The name of the table containing the object.
 * @param objectId - The unique identifier of the object.
 * @returns An object containing the total score and a map of reaction summaries.
 */
async function processReactionsForObject(
    tableName: string,
    objectId: string,
): Promise<{ totalScore: number, reactionSummaries: Map<string, number> }> {
    let totalScore = 0;
    const reactionSummaries = new Map<string, number>();

    await batch<FindManyArgs>({
        objectType: "Reaction",
        processBatch: async (batch) => {
            batch.forEach(reaction => {
                totalScore += getReactionScore(reaction.emoji);
                reactionSummaries.set(reaction.emoji, (reactionSummaries.get(reaction.emoji) || 0) + 1);
            });
        },
        select: {
            emoji: true,
        },
        where: { [`${camelCase(tableName)}Id`]: objectId },
    });

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
 * @param tableName - The name of the table to process.
 */
async function updateReactionsForTable(tableName: string): Promise<void> {
    try {
        await batch({
            objectType: uppercaseFirstLetter(camelCase(tableName)) as ModelType,
            processBatch: async (batch) => {
                for (const object of batch) {
                    const { totalScore, reactionSummaries } = await processReactionsForObject(tableName, object.id);

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
                        logger.warning(`Updating reactions for ${tableName} ${object.id}. Score mismatch: ${isScoreMismatch}, Reaction summary mismatch: ${isReactionSummaryMismatch}`, { trace: "0163" });

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

                        await DbProvider.get()[tableName].update({
                            where: { id: object.id },
                            data: updateData,
                        });
                    }
                }
            },
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
    } catch (error) {
        logger.error("processTableInBatches caught error", { error, trace: "0164" });
    }
};

export async function countReacts(): Promise<void> {
    const tableNames = [
        "api",
        "chat_message",
        "code",
        "comment",
        "issue",
        "note",
        "post",
        "project",
        "routine",
        "standard",
    ];

    for (const tableName of tableNames) {
        await updateReactionsForTable(tableName);
    }
};
