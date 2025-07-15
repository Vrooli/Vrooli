// AI_CHECK: TEST_COVERAGE=1 | LAST: 2025-06-24
// AI_CHECK: TYPE_SAFETY=1 | LAST: 2025-07-04
import { type Prisma } from "@prisma/client";
import { DbProvider, batch, logger } from "@vrooli/server";
import { generatePK, getReactionScore } from "@vrooli/shared";

// Define the specific table names that this job processes
const PROCESSED_REACTION_TABLE_NAMES = [
    "chat_message",
    "comment",
    "issue",
    "resource",
] as const;

// Create a union type from these table names
type ProcessableReactionTableName = typeof PROCESSED_REACTION_TABLE_NAMES[number];

// Map table names to their corresponding ModelType.
const TABLE_NAME_TO_MODEL_TYPE_MAP = {
    "chat_message": "ChatMessage",
    "comment": "Comment",
    "issue": "Issue",
    "resource": "Resource",
} as const;

// Declare select shape and payload type for individual reactions
const reactionSelect = { emoji: true } as const;
type ReactionPayload = Prisma.reactionGetPayload<{ select: typeof reactionSelect }>;

// Declare select shape and payload type for update batch
const objectSelect = {
    id: true,
    score: true,
    reactionSummaries: {
        select: {
            id: true, // This ID is bigint
            emoji: true,
            count: true,
        },
    },
} as const;

type ObjectPayload =
    Prisma.chat_messageGetPayload<{ select: typeof objectSelect }> |
    Prisma.commentGetPayload<{ select: typeof objectSelect }> |
    Prisma.issueGetPayload<{ select: typeof objectSelect }> |
    Prisma.resourceGetPayload<{ select: typeof objectSelect }>;

type UpdateArgs =
    Prisma.chat_messageFindManyArgs |
    Prisma.commentFindManyArgs |
    Prisma.issueFindManyArgs |
    Prisma.resourceFindManyArgs;

/**
 * Processes reactions for a single object, calculating the total score and building a map of reaction summaries.
 * 
 * @param tableName - The name of the table containing the object.
 * @param objectId - The unique identifier of the object.
 * @returns An object containing the total score and a map of reaction summaries.
 */
async function processReactionsForObject(
    tableName: ProcessableReactionTableName,
    objectId: bigint,
): Promise<{ totalScore: number, reactionSummaries: Map<string, number> }> {
    let totalScore = 0;
    const reactionSummaries = new Map<string, number>();

    let reactionWhereClause: Prisma.reactionWhereInput;
    switch (tableName) {
        case "chat_message":
            reactionWhereClause = { chatMessageId: objectId };
            break;
        case "comment":
            reactionWhereClause = { commentId: objectId };
            break;
        case "issue":
            reactionWhereClause = { issueId: objectId };
            break;
        case "resource":
            reactionWhereClause = { resourceId: objectId };
            break;
        default: {
            // This should be unreachable due to ProcessableReactionTableName
            const _exhaustiveCheck: never = tableName;
            logger.error(`Unhandled table name for reaction processing: ${_exhaustiveCheck}`, { trace: "0163A_REACTION_EXHAUSTIVE_CHECK" });
            throw new Error(`Unhandled table name: ${_exhaustiveCheck}`);
        }
    }

    await batch<Prisma.reactionFindManyArgs, ReactionPayload>({
        objectType: "Reaction", // Reactions are always of objectType "Reaction"
        processBatch: async (batch: ReactionPayload[]) => {
            batch.forEach(reaction => {
                totalScore += getReactionScore(reaction.emoji);
                reactionSummaries.set(reaction.emoji, (reactionSummaries.get(reaction.emoji) || 0) + 1);
            });
        },
        select: reactionSelect,
        where: reactionWhereClause,
    });

    return { totalScore, reactionSummaries };
}

/**
 * Represents an operation to update a reaction summary (create, update, or delete).
 */
type ReactionSummaryCreateOp = {
    type: "create";
    emoji: string;
    count: number;
};

// UpdateOp now targets a specific ID
type ReactionSummaryIdentifiedUpdateOp = {
    type: "update";
    id: bigint; // ID of the summary to update (bigint)
    count: number;
};

// DeleteOp now targets a specific ID
type ReactionSummaryIdentifiedDeleteOp = {
    type: "delete";
    id: bigint; // ID of the summary to delete (bigint)
};

type ReactionSummaryUpdateOperation =
    | ReactionSummaryCreateOp
    | ReactionSummaryIdentifiedUpdateOp
    | ReactionSummaryIdentifiedDeleteOp;

/**
 * Updates reaction data for all objects in a specified table.
 * 
 * @param tableName - The name of the table to process.
 */
async function updateReactionsForTable(tableName: ProcessableReactionTableName): Promise<void> {
    try {
        const db = DbProvider.get();
        await batch<UpdateArgs, ObjectPayload>({
            objectType: TABLE_NAME_TO_MODEL_TYPE_MAP[tableName],
            processBatch: async (batch: ObjectPayload[]) => {
                for (const object of batch) {
                    try {
                        const { totalScore, reactionSummaries } = await processReactionsForObject(tableName, object.id);

                        const updates: ReactionSummaryUpdateOperation[] = [];
                        let isReactionSummaryMismatch = false;

                        const newCalculatedSummariesMap = reactionSummaries;
                        const existingSummariesArray = object.reactionSummaries;

                        const existingSummariesByEmoji = new Map<string, Array<{ id: bigint, count: number }>>();
                        existingSummariesArray.forEach(s => {
                            const list = existingSummariesByEmoji.get(s.emoji) || [];
                            list.push({ id: s.id, count: s.count }); // s.id is bigint here
                            existingSummariesByEmoji.set(s.emoji, list);
                        });

                        newCalculatedSummariesMap.forEach((newCount, emoji) => {
                            const currentPersistentSummariesForEmoji = existingSummariesByEmoji.get(emoji);

                            if (!currentPersistentSummariesForEmoji || currentPersistentSummariesForEmoji.length === 0) {
                                updates.push({ type: "create", emoji, count: newCount });
                            } else {
                                // We already know the array has at least one element from the condition above
                                const summaryToKeepAndUpdate = currentPersistentSummariesForEmoji[0];
                                if (summaryToKeepAndUpdate && summaryToKeepAndUpdate.count !== newCount) {
                                    updates.push({ type: "update", id: summaryToKeepAndUpdate.id, count: newCount });
                                }
                                for (let i = 1; i < currentPersistentSummariesForEmoji.length; i++) {
                                    const summaryToDelete = currentPersistentSummariesForEmoji[i];
                                    if (summaryToDelete && summaryToDelete.id) {
                                        updates.push({ type: "delete", id: summaryToDelete.id });
                                    }
                                }
                            }
                        });

                        existingSummariesByEmoji.forEach((listOfExistingForEmoji, emoji) => {
                            if (!newCalculatedSummariesMap.has(emoji)) {
                                listOfExistingForEmoji.forEach(summaryToDelete => {
                                    updates.push({ type: "delete", id: summaryToDelete.id });
                                });
                            }
                        });

                        if (updates.length > 0) {
                            isReactionSummaryMismatch = true;
                        }

                        const isScoreMismatch = object.score !== totalScore;

                        if (isScoreMismatch || isReactionSummaryMismatch) {
                            logger.warning(`Updating reactions for ${tableName} ${object.id}. Score mismatch: ${isScoreMismatch}, Reaction summary mismatch: ${isReactionSummaryMismatch} `, { trace: "0163" });

                            const updateData = {
                                score: totalScore,
                                reactionSummaries: {
                                    create: updates
                                        .filter((u): u is ReactionSummaryCreateOp => u.type === "create")
                                        .map(u => ({ id: generatePK(), emoji: u.emoji, count: u.count })),
                                    update: updates
                                        .filter((u): u is ReactionSummaryIdentifiedUpdateOp => u.type === "update")
                                        .map(u => ({ where: { id: u.id }, data: { count: u.count } })), // Prisma expects array for multi-update
                                    delete: updates
                                        .filter((u): u is ReactionSummaryIdentifiedDeleteOp => u.type === "delete")
                                        .map(u => ({ id: u.id })), // Prisma expects array of WhereUniqueInputs
                                },
                            };

                            const updatePayload = {
                                where: { id: object.id },
                                data: updateData,
                            } as const; // Add 'as const' for stricter type checking if needed by Prisma version

                            switch (tableName) {
                                case "chat_message":
                                    await db.chat_message.update(updatePayload);
                                    break;
                                case "comment":
                                    await db.comment.update(updatePayload);
                                    break;
                                case "issue":
                                    await db.issue.update(updatePayload);
                                    break;
                                case "resource":
                                    await db.resource.update(updatePayload);
                                    break;
                                default: {
                                    const _exhaustiveCheck: never = tableName;
                                    logger.error(`Unhandled table name for update: ${_exhaustiveCheck}`, { trace: "0163B_UPDATE_EXHAUSTIVE_CHECK" });
                                    throw new Error(`Unhandled table name for update: ${_exhaustiveCheck}`);
                                }
                            }
                        }
                    } catch (error) {
                        logger.error(`Failed to process object ${object.id} in table ${tableName}.`, {
                            error,
                            objectId: object.id,
                            tableName,
                            trace: "0163C_OBJECT_PROCESSING_FAILURE",
                        });
                    }
                }
            },
            select: objectSelect,
        });
    } catch (error) {
        logger.error("processTableInBatches caught error", { error, trace: "0164" });
    }
}

export async function countReacts(): Promise<void> {
    const tableNames = PROCESSED_REACTION_TABLE_NAMES;

    for (const tableName of tableNames) {
        await updateReactionsForTable(tableName);
    }
}
