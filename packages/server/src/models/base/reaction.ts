import { exists, getReactionScore, lowercaseFirstLetter, ReactInput, ReactionFor, removeModifiers } from "@local/shared";
import { reaction_summary } from "@prisma/client";
import { ApiModel, ChatMessageModel, CommentModel, IssueModel, NoteModel, PostModel, ProjectModel, QuestionAnswerModel, QuestionModel, QuizModel, RoutineModel, SmartContractModel, StandardModel } from ".";
import { onlyValidIds } from "../../builders";
import { CustomError, Trigger } from "../../events";
import { PrismaType, SessionUserToken } from "../../types";
import { ReactionFormat } from "../formats";
import { ModelLogic } from "../types";
import { ApiModelLogic, ChatMessageModelLogic, CommentModelLogic, IssueModelLogic, NoteModelLogic, PostModelLogic, ProjectModelLogic, QuestionAnswerModelLogic, QuestionModelLogic, QuizModelLogic, ReactionModelLogic, RoutineModelLogic, SmartContractModelLogic, StandardModelLogic } from "./types";

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
const suppFields = [] as const;
export const ReactionModel: ModelLogic<ReactionModelLogic, typeof suppFields> = ({
    __typename,
    delegate: (prisma) => prisma.reaction,
    display: {
        label: {
            select: () => ({
                id: true,
                api: { select: ApiModel.display.label.select() },
                chatMessage: { select: ChatMessageModel.display.label.select() },
                comment: { select: CommentModel.display.label.select() },
                issue: { select: IssueModel.display.label.select() },
                note: { select: NoteModel.display.label.select() },
                post: { select: PostModel.display.label.select() },
                project: { select: ProjectModel.display.label.select() },
                question: { select: QuestionModel.display.label.select() },
                questionAnswer: { select: QuestionAnswerModel.display.label.select() },
                quiz: { select: QuizModel.display.label.select() },
                routine: { select: RoutineModel.display.label.select() },
                smartContract: { select: SmartContractModel.display.label.select() },
                standard: { select: StandardModel.display.label.select() },
            }),
            get: (select, languages) => {
                if (select.api) return ApiModel.display.label.get(select.api as ApiModelLogic["PrismaModel"], languages);
                if (select.chatMessage) return ChatMessageModel.display.label.get(select.chatMessage as ChatMessageModelLogic["PrismaModel"], languages);
                if (select.comment) return CommentModel.display.label.get(select.comment as CommentModelLogic["PrismaModel"], languages);
                if (select.issue) return IssueModel.display.label.get(select.issue as IssueModelLogic["PrismaModel"], languages);
                if (select.note) return NoteModel.display.label.get(select.note as NoteModelLogic["PrismaModel"], languages);
                if (select.post) return PostModel.display.label.get(select.post as PostModelLogic["PrismaModel"], languages);
                if (select.project) return ProjectModel.display.label.get(select.project as ProjectModelLogic["PrismaModel"], languages);
                if (select.question) return QuestionModel.display.label.get(select.question as QuestionModelLogic["PrismaModel"], languages);
                if (select.questionAnswer) return QuestionAnswerModel.display.label.get(select.questionAnswer as QuestionAnswerModelLogic["PrismaModel"], languages);
                if (select.quiz) return QuizModel.display.label.get(select.quiz as QuizModelLogic["PrismaModel"], languages);
                if (select.routine) return RoutineModel.display.label.get(select.routine as RoutineModelLogic["PrismaModel"], languages);
                if (select.smartContract) return SmartContractModel.display.label.get(select.smartContract as SmartContractModelLogic["PrismaModel"], languages);
                if (select.standard) return StandardModel.display.label.get(select.standard as StandardModelLogic["PrismaModel"], languages);
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
