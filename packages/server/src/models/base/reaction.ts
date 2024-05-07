import { exists, getReactionScore, GqlModelType, lowercaseFirstLetter, ReactInput, ReactionFor, removeModifiers } from "@local/shared";
import { reaction_summary } from "@prisma/client";
import { ModelMap } from ".";
import { onlyValidIds } from "../../builders/onlyValidIds";
import { prismaInstance } from "../../db/instance";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { PrismaType, SessionUserToken } from "../../types";
import { ReactionFormat } from "../formats";
import { ReactionModelLogic } from "./types";

const forMapper: { [key in ReactionFor]: string } = {
    Api: "api",
    ChatMessage: "chatMessage",
    Comment: "comment",
    Issue: "issue",
    Note: "note",
    Post: "post",
    Project: "project",
    Question: "question",
    QuestionAnswer: "questionAnswer",
    Quiz: "quiz",
    Routine: "routine",
    SmartContract: "smartContract",
    Standard: "standard",
};

const __typename = "Reaction" as const;
export const ReactionModel: ReactionModelLogic = ({
    __typename,
    dbTable: "reaction",
    display: () => ({
        label: {
            select: () => ({
                id: true,
                ...Object.fromEntries(Object.entries(forMapper).map(([key, value]) =>
                    [value, { select: ModelMap.get(key as GqlModelType).display().label.select() }])),
            }),
            get: (select, languages) => {
                for (const [key, value] of Object.entries(forMapper)) {
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
            const reactionsArray = await prismaInstance.reaction.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
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
        // Define prisma type for reacted-on object (e.g. chatMessage)
        const reactionForCamel = forMapper[input.reactionFor];
        // Convert to snake case (e.g. chat_message)
        const reactionForSnake = reactionForCamel.replace(/([A-Z])/g, "_$1").toLowerCase();
        // Get prisma type for reacted-on object (e.g. prismaInstance.chat_message)
        const prismaFor = (prismaInstance[reactionForSnake as keyof PrismaType] as any);
        // Check if object being reacted on exists
        const reactingFor: null | { id: string, score: number, reactionSummaries: reaction_summary[] } = await prismaFor.findUnique({
            where: { id: input.forConnect },
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
        if (!reactingFor)
            throw new CustomError("0118", "NotFound", userData.languages, { reactionFor: input.reactionFor, forId: input.forConnect });
        // Find sentiment of current and new reaction
        const isRemove = !exists(input.emoji);
        const feelingNew = getReactionScore(input.emoji!);
        // Check if reaction exists
        const reaction = await prismaInstance.reaction.findFirst({
            where: {
                byId: userData.id,
                [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
            },
        });
        // If reaction already existed
        if (reaction) {
            const feelingExisting = getReactionScore(reaction.emoji);
            const isSame = removeModifiers(input.emoji!) === removeModifiers(reaction.emoji);
            // If reaction is the same as the one we're trying to cast, skip
            if (isSame) return true;
            // If removing reaction, delete it
            if (isRemove) {
                await prismaInstance.reaction.delete({ where: { id: reaction.id } });
                // Update the corresponding reaction summary table
                const summaryTable = reactingFor.reactionSummaries.find((summary: any) => summary.emoji === reaction.emoji);
                if (summaryTable) {
                    await prismaInstance.reaction_summary.update({
                        where: { id: summaryTable.id },
                        data: { count: Math.max(0, summaryTable.count - 1) },
                    });
                }
            }
            // Otherwise, update the reaction
            else {
                await prismaInstance.reaction.update({
                    where: { id: reaction.id },
                    data: { emoji: input.emoji! },
                });
                // Upsert the corresponding reaction summary table
                const summaryTable = reactingFor.reactionSummaries.find((summary: any) => summary.emoji === input.emoji);
                if (summaryTable) {
                    await prismaInstance.reaction_summary.update({
                        where: { id: summaryTable.id },
                        data: { count: summaryTable.count + 1 },
                    });
                }
                else {
                    await prismaInstance.reaction_summary.create({
                        data: {
                            emoji: input.emoji!,
                            count: 1,
                            [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                        },
                    });
                }
            }
            // Handle trigger
            await Trigger(userData.languages).objectReact(reaction.emoji, input.emoji, input.reactionFor, input.forConnect, userData.id);
            // Update the score
            const deltaVoteCount = feelingNew - feelingExisting;
            await prismaFor.update({
                where: { id: input.forConnect },
                data: { score: reactingFor.score + deltaVoteCount },
            });
            return true;
        }
        // If reaction did not already exist
        else {
            // If removing reaction, skip. There's nothing to remove
            if (isRemove) return true;
            // Create the reaction
            await prismaInstance.reaction.create({
                data: {
                    byId: userData.id,
                    emoji: input.emoji!,
                    [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                },
            });
            // Upsert the corresponding reaction summary table
            const summaryTable = reactingFor.reactionSummaries.find((summary: any) => summary.emoji === input.emoji);
            if (summaryTable) {
                await prismaInstance.reaction_summary.update({
                    where: { id: summaryTable.id },
                    data: { count: summaryTable.count + 1 },
                });
            }
            else {
                await prismaInstance.reaction_summary.create({
                    data: {
                        emoji: input.emoji!,
                        count: 1,
                        [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                    },
                });
            }
            // Handle trigger
            await Trigger(userData.languages).objectReact(null, input.emoji, input.reactionFor, input.forConnect, userData.id);
            // Update the score
            await prismaFor.update({
                where: { id: input.forConnect },
                data: { score: reactingFor.score + feelingNew },
            });
            return true;
        }
    },
    search: {} as any,
    validate: () => ({}) as any,
});
