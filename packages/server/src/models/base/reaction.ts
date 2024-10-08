import { GqlModelType, MaxObjects, ReactInput, ReactionFor, ReactionSearchInput, ReactionSortBy, camelCase, exists, getReactionScore, lowercaseFirstLetter, removeModifiers, uuid } from "@local/shared";
import { Prisma } from "@prisma/client";
import { ModelMap } from ".";
import { onlyValidIds } from "../../builders/onlyValidIds";
import { permissionsSelectHelper } from "../../builders/permissionsSelectHelper";
import { useVisibility, useVisibilityMapper } from "../../builders/visibilityBuilder";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { logger } from "../../events/logger";
import { Trigger } from "../../events/trigger";
import { emitSocketEvent } from "../../sockets/events";
import { PrismaDelegate, SessionUserToken } from "../../types";
import { defaultPermissions, oneIsPublic } from "../../utils";
import { calculatePermissions } from "../../validators/permissions";
import { ReactionFormat } from "../formats";
import { ReactionModelInfo, ReactionModelLogic } from "./types";

const forMapper: { [key in ReactionFor]?: keyof Prisma.reactionUpsertArgs["create"] } = {
    Api: "api",
    ChatMessage: "chatMessage",
    Code: "code",
    Comment: "comment",
    Issue: "issue",
    Note: "note",
    Post: "post",
    Project: "project",
    Question: "question",
    QuestionAnswer: "questionAnswer",
    Quiz: "quiz",
    Routine: "routine",
    Standard: "standard",
};
const reversedForMapper = Object.fromEntries(
    Object.entries(forMapper).map(([key, value]) => [value, key]),
);

function reactionForToRelation(reactionFor: keyof typeof ReactionFor): string {
    if (!ReactionFor[reactionFor]) {
        throw new CustomError("0597", "InvalidArgs", ["en"], { reactionFor });
    }
    // Convert to camelCase and sanitize
    return camelCase(reactionFor);
}

const __typename = "Reaction" as const;
export const ReactionModel: ReactionModelLogic = ({
    __typename,
    dbTable: "reaction",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.keys(ReactionFor).map((key) =>
                    [reactionForToRelation(key as keyof typeof ReactionFor), { select: ModelMap.get(key as GqlModelType).display().label.select() }]),
            }),
            get: (select, languages) => {
                for (const key of Object.keys(ReactionFor)) {
                    const value = reactionForToRelation(key as keyof typeof ReactionFor);
                    if (select[value]) return ModelMap.get(key as GqlModelType).display().label.get(select[value], languages);
                }
                return "";
            },
        },
    }),
    format: ReactionFormat,
    query: {
        /**
         * Finds your reactions on a list of items, or null if you haven't reacted to that item
         */
        async getReactions(
            userId: string | null | undefined,
            ids: string[],
            reactionFor: keyof typeof ReactionFor,
        ): Promise<Array<string | null>> {
            // Create result array that is the same length as ids
            const result = new Array(ids.length).fill(null);
            // If userId not provided, return result
            if (!userId) return result;
            // Filter out nulls and undefineds from ids
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${lowercaseFirstLetter(reactionFor)}Id`;
            const reactionsArray = await prismaInstance.reaction.findMany({
                where: { byId: userId, [fieldName]: { in: idsFiltered } },
                select: { [fieldName]: true, emoji: true },
            });
            // Replace the nulls in the result array with true or false
            for (let i = 0; i < ids.length; i++) {
                // Try to find this id in the reactions array
                if (exists(ids[i])) {
                    // If found, set result index to value of emoji field
                    result[i] = reactionsArray.find((vote: any) => vote[fieldName] === ids[i])?.emoji ?? null;
                }
            }
            return result;
        },
    },
    /**
     * Casts reaction. Makes sure not to duplicate, and to override existing reactions. 
     * A user may react on their own project/routine/etc.
     * @returns True if cast correctly (even if skipped because of duplicate)
     */
    react: async (userData: SessionUserToken, input: ReactInput): Promise<boolean> => {
        const { dbTable: reactedOnDbTable, validate: reactedOnValidate } = ModelMap.getLogic(["dbTable", "validate"], input.reactionFor as string as GqlModelType, true, `ModelMap.react for ${input.reactionFor}`);
        const reactedOnValidator = reactedOnValidate();
        // Chat messages get additional treatment, as we need to send the updated reaction summaries to the chat room to update open clients
        const isChatMessage = input.reactionFor === "ChatMessage";
        // Check if object being reacted on exists, and if the user has already reacted on it
        const reactedOnObject = await (prismaInstance[reactedOnDbTable] as PrismaDelegate).findUnique({
            where: { id: input.forConnect },
            select: {
                id: true,
                score: true,
                reactions: {
                    where: { byId: userData.id },
                    select: {
                        id: true,
                        emoji: true,
                    },
                },
                reactionSummaries: {
                    select: {
                        id: true,
                        emoji: true,
                        count: true,
                    },
                },
                ...(isChatMessage ? { chatId: true } : {}),
                ...permissionsSelectHelper(reactedOnValidator.permissionsSelect, userData.id, userData.languages),
            },
        });
        if (!reactedOnObject) {
            throw new CustomError("0118", "NotFound", userData.languages, { reactionFor: input.reactionFor, forId: input.forConnect });
        }
        // Check if the user has permission to react on this object
        const authData = { __typename: input.reactionFor, ...reactedOnObject };
        console.log("permissions select", JSON.stringify(permissionsSelectHelper(reactedOnValidator.permissionsSelect, userData.id, userData.languages)));
        console.log("auth data", JSON.stringify(authData));
        const permissions = await calculatePermissions(authData, userData, reactedOnValidator);
        const ownerData = reactedOnValidator.owner(authData, userData.id);
        const objectOwner = ownerData.Team ? { __typename: "Team" as const, id: ownerData.Team.id } : ownerData.User ? { __typename: "User" as const, id: ownerData.User.id } : null;
        if (!permissions.canReact) {
            throw new CustomError("0637", "Unauthorized", userData.languages, { reactionFor: input.reactionFor, forId: input.forConnect });
        }
        const reaction = Array.isArray(reactedOnObject.reactions) && reactedOnObject.reactions.length > 0
            ? reactedOnObject.reactions[0] as { id: string, emoji: string }
            : null;
        // Find sentiment of current and new reaction
        const isRemove = !exists(input.emoji);
        const deltaScore = getReactionScore(input.emoji) - getReactionScore(reaction?.emoji);
        const updatedScore = reactedOnObject.score + deltaScore;
        // If we're setting the reaction to the same as the current one, skip
        const isSame = (isRemove && !reaction) || (reaction && removeModifiers(input.emoji!) === removeModifiers(reaction.emoji));
        if (isSame) return true;
        // Determine reaction query
        const reactionQuery = !reaction
            ? { create: { byId: userData.id, emoji: input.emoji || "" } } as const
            : isRemove
                ? { delete: { id: reaction.id } } as const
                : { update: { where: { id: reaction.id }, data: { emoji: input.emoji || "" } } } as const;
        // Prepare transaction operations
        const transactionOps: Prisma.PrismaPromise<object>[] = [];
        // Update the score and reaction
        transactionOps.push((prismaInstance[reactedOnDbTable] as PrismaDelegate).update({
            where: { id: input.forConnect },
            data: {
                score: reactedOnObject.score + deltaScore,
                reactions: reactionQuery,
            },
        }) as Prisma.PrismaPromise<object>);
        // Prepare in-memory reaction summaries for updating chat room
        const updatedReactionSummaries = [...reactedOnObject.reactionSummaries];
        // Determine reaction summary query
        const reactedOnIdField = `${reactionForToRelation(input.reactionFor)}Id` as keyof Prisma.reaction_summaryWhereInput;
        type EmojiAdjustment = { emoji: string, delta: number };
        const emojisToAdjust: EmojiAdjustment[] = [];
        const oldEmoji = reaction?.emoji;
        const newEmoji = input.emoji;
        // If emoji is changing, adjust the summary counts for both old and new emojis
        if (oldEmoji && newEmoji && oldEmoji !== newEmoji) {
            emojisToAdjust.push({ emoji: oldEmoji, delta: -1 });
            emojisToAdjust.push({ emoji: newEmoji, delta: +1 });
        }
        // If emoji is being removed, adjust the summary count for the old emoji
        else if (oldEmoji && !newEmoji) {
            emojisToAdjust.push({ emoji: oldEmoji, delta: -1 });
        }
        // If emoji is being added, adjust the summary count for the new emoji
        else if (!oldEmoji && newEmoji) {
            emojisToAdjust.push({ emoji: newEmoji, delta: +1 });
        }
        // Update reaction summary for each emoji
        for (const { emoji, delta } of emojisToAdjust) {
            const whereClause = {
                emoji,
                [reactedOnIdField]: input.forConnect,
            } as const;

            if (delta > 0) {
                // Try to update existing summary
                const createId = uuid();
                transactionOps.push(
                    prismaInstance.$executeRawUnsafe(`
                    INSERT INTO "reaction_summary" (id, emoji, count, "${reactedOnIdField}")
                    VALUES ($1::UUID, $2, $3, $4::UUID)
                    ON CONFLICT (emoji, "apiId", "chatMessageId", "codeId", "commentId", "issueId", "noteId", "postId", "projectId", "questionId", "questionAnswerId", "quizId", "routineId", "standardId")
                    DO UPDATE SET count = reaction_summary.count + $5
                  `, createId, emoji, delta, input.forConnect, delta) as unknown as Prisma.PrismaPromise<object>,
                );
                // Add/update in-memory summaries
                const summary = updatedReactionSummaries.find((s) => s.emoji === emoji);
                if (summary) {
                    summary.count += delta;
                } else {
                    updatedReactionSummaries.push({ id: createId, emoji, count: delta });
                }
            } else if (delta < 0) {
                // Decrement count
                transactionOps.push(
                    prismaInstance.reaction_summary.updateMany({
                        where: whereClause,
                        data: {
                            count: {
                                decrement: -delta,
                            },
                        },
                    }) as Prisma.PrismaPromise<object>,
                );
                // Delete summaries where count <= 0
                transactionOps.push(
                    prismaInstance.reaction_summary.deleteMany({
                        where: {
                            ...whereClause,
                            count: {
                                lte: 0,
                            },
                        },
                    }) as Prisma.PrismaPromise<object>,
                );
                // Update in-memory summaries
                const summaryIndex = updatedReactionSummaries.findIndex((s) => s.emoji === emoji);
                if (summaryIndex !== -1) {
                    updatedReactionSummaries[summaryIndex].count += delta;
                    if (updatedReactionSummaries[summaryIndex].count <= 0) {
                        // Remove the summary from in-memory array
                        updatedReactionSummaries.splice(summaryIndex, 1);
                    }
                } else {
                    // No summary found to decrement; this should not happen
                    logger.error("Reaction summary not found for decrement.", { emoji, trace: "0569" });
                }
            }
        }
        // Execute transaction
        await prismaInstance.$transaction(transactionOps);
        // If we reacted to a chat message, send the updated reaction summaries to the chat room
        if (isChatMessage && reactedOnObject.chatId) {
            emitSocketEvent("messages", reactedOnObject.chatId, { updated: [{ id: input.forConnect, reactionSummaries: updatedReactionSummaries }] });
        }
        // Handle trigger
        await Trigger(userData.languages).objectReact({
            deltaScore,
            objectType: input.reactionFor,
            objectId: input.forConnect,
            updatedScore,
            userId: userData.id,
            objectOwner,
            languages: userData.languages,
        });
        return true;
    },
    search: {
        defaultSort: ReactionSortBy.DateUpdatedDesc,
        sortBy: ReactionSortBy,
        searchFields: {
            apiId: true,
            chatMessageId: true,
            codeId: true,
            commentId: true,
            excludeLinkedToTag: true,
            issueId: true,
            noteId: true,
            postId: true,
            projectId: true,
            questionId: true,
            questionAnswerId: true,
            quizId: true,
            routineId: true,
            standardId: true,
        },
        searchStringQuery: () => ({}), // Intentionally empty
    },
    validate: () => ({
        isDeleted: () => false,
        isPublic: function getIsPublic(data, ...rest) {
            if (!data.by) return false;
            if (data.by.isPrivateVotes) return false;
            return oneIsPublic<ReactionModelInfo["PrismaSelect"]>([
                ["api", "Api"],
                ["chatMessage", "ChatMessage"],
                ["code", "Code"],
                ["comment", "Comment"],
                ["issue", "Issue"],
                ["note", "Note"],
                ["post", "Post"],
                ["project", "Project"],
                ["question", "Question"],
                ["questionAnswer", "QuestionAnswer"],
                ["quiz", "Quiz"],
                ["routine", "Routine"],
                ["standard", "Standard"],
            ], data, ...rest);
        },
        isTransferable: false,
        maxObjects: MaxObjects[__typename],
        owner: (data) => ({
            User: data?.by,
        }),
        permissionResolvers: defaultPermissions,
        permissionsSelect: () => ({
            id: true,
            by: "User",
        }),
        visibility: {
            own: function getOwn(data) {
                return {
                    by: { id: data.userId },
                    // Any non-public, non-owned objects should be filtered out
                    // Can use OR because only one relation will be present
                    OR: [
                        ...useVisibilityMapper("OwnOrPublic", data, forMapper, false),
                    ],
                };
            },
            // Not useful for this object type
            ownOrPublic: null,
            // Not useful for this object type
            ownPrivate: function getOwnPrivate(data) {
                return useVisibility("View", "Own", data);
            },
            // Not useful for this object type
            ownPublic: function getOwnPublic(data) {
                return useVisibility("View", "Own", data);
            },
            public: function getPublic(data) {
                const searchInput = data.searchInput as ReactionSearchInput;
                // If the search input has a relation ID, return that relation only
                const forSearch = Object.keys(searchInput).find(searchKey =>
                    searchKey.endsWith("Id") &&
                    reversedForMapper[searchKey.substring(0, searchKey.length - "Id".length)],
                );
                if (forSearch) {
                    const relation = forSearch.substring(0, forSearch.length - "Id".length);
                    return { [relation]: useVisibility(reversedForMapper[relation] as GqlModelType, "Public", data) };
                }
                // Otherwise, use an OR on all relations
                return {
                    OR: [
                        ...useVisibilityMapper("Public", data, forMapper, false),
                    ],
                };
            },
        },
    }),
});
