import { exists, getReactionScore, lowercaseFirstLetter, ReactInput, ReactionFor, removeModifiers } from "@local/shared";
import { reaction_summary } from "@prisma/client";
import { ModelMap } from ".";
import { onlyValidIds } from "../../builders/onlyValidIds";
import { CustomError } from "../../events/error";
import { Trigger } from "../../events/trigger";
import { PrismaType, SessionUserToken } from "../../types";
import { ReactionFormat } from "../formats";
import { ApiModelInfo, ApiModelLogic, ChatMessageModelInfo, ChatMessageModelLogic, CommentModelInfo, CommentModelLogic, IssueModelInfo, IssueModelLogic, NoteModelInfo, NoteModelLogic, PostModelInfo, PostModelLogic, ProjectModelInfo, ProjectModelLogic, QuestionAnswerModelInfo, QuestionAnswerModelLogic, QuestionModelInfo, QuestionModelLogic, QuizModelInfo, QuizModelLogic, ReactionModelLogic, RoutineModelInfo, RoutineModelLogic, SmartContractModelInfo, SmartContractModelLogic, StandardModelInfo, StandardModelLogic } from "./types";

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
    delegate: (prisma) => prisma.reaction,
    display: {
        label: {
            select: () => ({
                id: true,
                api: { select: ModelMap.get<ApiModelLogic>("Api").display.label.select() },
                chatMessage: { select: ModelMap.get<ChatMessageModelLogic>("ChatMessage").display.label.select() },
                comment: { select: ModelMap.get<CommentModelLogic>("Comment").display.label.select() },
                issue: { select: ModelMap.get<IssueModelLogic>("Issue").display.label.select() },
                note: { select: ModelMap.get<NoteModelLogic>("Note").display.label.select() },
                post: { select: ModelMap.get<PostModelLogic>("Post").display.label.select() },
                project: { select: ModelMap.get<ProjectModelLogic>("Project").display.label.select() },
                question: { select: ModelMap.get<QuestionModelLogic>("Question").display.label.select() },
                questionAnswer: { select: ModelMap.get<QuestionAnswerModelLogic>("QuestionAnswer").display.label.select() },
                quiz: { select: ModelMap.get<QuizModelLogic>("Quiz").display.label.select() },
                routine: { select: ModelMap.get<RoutineModelLogic>("Routine").display.label.select() },
                smartContract: { select: ModelMap.get<SmartContractModelLogic>("SmartContract").display.label.select() },
                standard: { select: ModelMap.get<StandardModelLogic>("Standard").display.label.select() },
            }),
            get: (select, languages) => {
                if (select.api) return ModelMap.get<ApiModelLogic>("Api").display.label.get(select.api as ApiModelInfo["PrismaModel"], languages);
                if (select.chatMessage) return ModelMap.get<ChatMessageModelLogic>("ChatMessage").display.label.get(select.chatMessage as ChatMessageModelInfo["PrismaModel"], languages);
                if (select.comment) return ModelMap.get<CommentModelLogic>("Comment").display.label.get(select.comment as CommentModelInfo["PrismaModel"], languages);
                if (select.issue) return ModelMap.get<IssueModelLogic>("Issue").display.label.get(select.issue as IssueModelInfo["PrismaModel"], languages);
                if (select.note) return ModelMap.get<NoteModelLogic>("Note").display.label.get(select.note as NoteModelInfo["PrismaModel"], languages);
                if (select.post) return ModelMap.get<PostModelLogic>("Post").display.label.get(select.post as PostModelInfo["PrismaModel"], languages);
                if (select.project) return ModelMap.get<ProjectModelLogic>("Project").display.label.get(select.project as ProjectModelInfo["PrismaModel"], languages);
                if (select.question) return ModelMap.get<QuestionModelLogic>("Question").display.label.get(select.question as QuestionModelInfo["PrismaModel"], languages);
                if (select.questionAnswer) return ModelMap.get<QuestionAnswerModelLogic>("QuestionAnswer").display.label.get(select.questionAnswer as QuestionAnswerModelInfo["PrismaModel"], languages);
                if (select.quiz) return ModelMap.get<QuizModelLogic>("Quiz").display.label.get(select.quiz as QuizModelInfo["PrismaModel"], languages);
                if (select.routine) return ModelMap.get<RoutineModelLogic>("Routine").display.label.get(select.routine as RoutineModelInfo["PrismaModel"], languages);
                if (select.smartContract) return ModelMap.get<SmartContractModelLogic>("SmartContract").display.label.get(select.smartContract as SmartContractModelInfo["PrismaModel"], languages);
                if (select.standard) return ModelMap.get<StandardModelLogic>("Standard").display.label.get(select.standard as StandardModelInfo["PrismaModel"], languages);
                return "";
            },
        },
    },
    format: ReactionFormat,
    query: {
        /**
         * Finds your reactions on a list of items, or null if you haven't reacted to that item
         */
        async getReactions(
            prisma: PrismaType,
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
            const reactionsArray = await prisma.reaction.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
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
    react: async (prisma: PrismaType, userData: SessionUserToken, input: ReactInput): Promise<boolean> => {
        // Define prisma type for reacted-on object (e.g. chatMessage)
        const reactionForCamel = forMapper[input.reactionFor];
        // Convert to snake case (e.g. chat_message)
        const reactionForSnake = reactionForCamel.replace(/([A-Z])/g, "_$1").toLowerCase();
        // Get prisma type for reacted-on object (e.g. prisma.chat_message)
        const prismaFor = (prisma[reactionForSnake as keyof PrismaType] as any);
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
        const reaction = await prisma.reaction.findFirst({
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
                await prisma.reaction.delete({ where: { id: reaction.id } });
                // Update the corresponding reaction summary table
                const summaryTable = reactingFor.reactionSummaries.find((summary: any) => summary.emoji === reaction.emoji);
                if (summaryTable) {
                    await prisma.reaction_summary.update({
                        where: { id: summaryTable.id },
                        data: { count: Math.max(0, summaryTable.count - 1) },
                    });
                }
            }
            // Otherwise, update the reaction
            else {
                await prisma.reaction.update({
                    where: { id: reaction.id },
                    data: { emoji: input.emoji! },
                });
                // Upsert the corresponding reaction summary table
                const summaryTable = reactingFor.reactionSummaries.find((summary: any) => summary.emoji === input.emoji);
                if (summaryTable) {
                    await prisma.reaction_summary.update({
                        where: { id: summaryTable.id },
                        data: { count: summaryTable.count + 1 },
                    });
                }
                else {
                    await prisma.reaction_summary.create({
                        data: {
                            emoji: input.emoji!,
                            count: 1,
                            [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                        },
                    });
                }
            }
            // Handle trigger
            await Trigger(prisma, userData.languages).objectReact(reaction.emoji, input.emoji, input.reactionFor, input.forConnect, userData.id);
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
            await prisma.reaction.create({
                data: {
                    byId: userData.id,
                    emoji: input.emoji!,
                    [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                },
            });
            // Upsert the corresponding reaction summary table
            const summaryTable = reactingFor.reactionSummaries.find((summary: any) => summary.emoji === input.emoji);
            if (summaryTable) {
                await prisma.reaction_summary.update({
                    where: { id: summaryTable.id },
                    data: { count: summaryTable.count + 1 },
                });
            }
            else {
                await prisma.reaction_summary.create({
                    data: {
                        emoji: input.emoji!,
                        count: 1,
                        [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                    },
                });
            }
            // Handle trigger
            await Trigger(prisma, userData.languages).objectReact(null, input.emoji, input.reactionFor, input.forConnect, userData.id);
            // Update the score
            await prismaFor.update({
                where: { id: input.forConnect },
                data: { score: reactingFor.score + feelingNew },
            });
            return true;
        }
    },
    search: {} as any,
    validate: {} as any,
});
