import { exists, getReactionScore, removeModifiers } from "@local/utils";
import { ApiModel, ChatMessageModel, CommentModel, IssueModel, NoteModel, PostModel, ProjectModel, QuestionAnswerModel, QuestionModel, QuizModel, RoutineModel, SmartContractModel, StandardModel } from ".";
import { onlyValidIds, selPad } from "../builders";
import { CustomError, Trigger } from "../events";
const forMapper = {
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
const __typename = "Reaction";
const suppFields = [];
export const ReactionModel = ({
    __typename,
    delegate: (prisma) => prisma.reaction,
    display: {
        select: () => ({
            id: true,
            api: selPad(ApiModel.display.select),
            chatMessage: selPad(ChatMessageModel.display.select),
            comment: selPad(CommentModel.display.select),
            issue: selPad(IssueModel.display.select),
            note: selPad(NoteModel.display.select),
            post: selPad(PostModel.display.select),
            project: selPad(ProjectModel.display.select),
            question: selPad(QuestionModel.display.select),
            questionAnswer: selPad(QuestionAnswerModel.display.select),
            quiz: selPad(QuizModel.display.select),
            routine: selPad(RoutineModel.display.select),
            smartContract: selPad(SmartContractModel.display.select),
            standard: selPad(StandardModel.display.select),
        }),
        label: (select, languages) => {
            if (select.api)
                return ApiModel.display.label(select.api, languages);
            if (select.chatMessage)
                return ChatMessageModel.display.label(select.chatMessage, languages);
            if (select.comment)
                return CommentModel.display.label(select.comment, languages);
            if (select.issue)
                return IssueModel.display.label(select.issue, languages);
            if (select.note)
                return NoteModel.display.label(select.note, languages);
            if (select.post)
                return PostModel.display.label(select.post, languages);
            if (select.project)
                return ProjectModel.display.label(select.project, languages);
            if (select.question)
                return QuestionModel.display.label(select.question, languages);
            if (select.questionAnswer)
                return QuestionAnswerModel.display.label(select.questionAnswer, languages);
            if (select.quiz)
                return QuizModel.display.label(select.quiz, languages);
            if (select.routine)
                return RoutineModel.display.label(select.routine, languages);
            if (select.smartContract)
                return SmartContractModel.display.label(select.smartContract, languages);
            if (select.standard)
                return StandardModel.display.label(select.standard, languages);
            return "";
        },
    },
    format: {
        gqlRelMap: {
            __typename,
            by: "User",
            to: {
                api: "Api",
                chatMessage: "ChatMessage",
                comment: "Comment",
                issue: "Issue",
                note: "Note",
                post: "Post",
                project: "Project",
                question: "Question",
                questionAnswer: "QuestionAnswer",
                quiz: "Quiz",
                routine: "Routine",
                smartContract: "SmartContract",
                standard: "Standard",
            },
        },
        prismaRelMap: {
            __typename,
            by: "User",
            api: "Api",
            chatMessage: "ChatMessage",
            comment: "Comment",
            issue: "Issue",
            note: "Note",
            post: "Post",
            project: "Project",
            question: "Question",
            questionAnswer: "QuestionAnswer",
            quiz: "Quiz",
            routine: "Routine",
            smartContract: "SmartContract",
            standard: "Standard",
        },
        countFields: {},
    },
    query: {
        async getReactions(prisma, userId, ids, reactionFor) {
            const result = new Array(ids.length).fill(null);
            if (!userId)
                return result;
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${reactionFor.toLowerCase()}Id`;
            const reactionsArray = await prisma.reaction.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
            for (let i = 0; i < ids.length; i++) {
                if (exists(ids[i])) {
                    result[i] = reactionsArray.find((vote) => vote[fieldName] === ids[i])?.emoji ?? null;
                }
            }
            return result;
        },
    },
    react: async (prisma, userData, input) => {
        const prismaFor = prisma[forMapper[input.reactionFor]];
        const reactingFor = await prismaFor.findUnique({
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
        const isRemove = !exists(input.emoji);
        const feelingNew = getReactionScore(input.emoji);
        const reaction = await prisma.reaction.findFirst({
            where: {
                byId: userData.id,
                [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
            },
        });
        if (reaction) {
            const feelingExisting = getReactionScore(reaction.emoji);
            const isSame = removeModifiers(input.emoji) === removeModifiers(reaction.emoji);
            if (isSame)
                return true;
            if (isRemove) {
                await prisma.reaction.delete({ where: { id: reaction.id } });
                const summaryTable = reactingFor.reactionSummaries.find((summary) => summary.emoji === reaction.emoji);
                if (summaryTable) {
                    await prisma.reaction_summary.update({
                        where: { id: summaryTable.id },
                        data: { count: Math.max(0, summaryTable.count - 1) },
                    });
                }
            }
            else {
                await prisma.reaction.update({
                    where: { id: reaction.id },
                    data: { emoji: input.emoji },
                });
                const summaryTable = reactingFor.reactionSummaries.find((summary) => summary.emoji === input.emoji);
                if (summaryTable) {
                    await prisma.reaction_summary.update({
                        where: { id: summaryTable.id },
                        data: { count: summaryTable.count + 1 },
                    });
                }
                else {
                    await prisma.reaction_summary.create({
                        data: {
                            emoji: input.emoji,
                            count: 1,
                            [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                        },
                    });
                }
            }
            await Trigger(prisma, userData.languages).objectReact(reaction.emoji, input.emoji, input.reactionFor, input.forConnect, userData.id);
            const deltaVoteCount = feelingNew - feelingExisting;
            await prismaFor.update({
                where: { id: input.forConnect },
                data: { score: reactingFor.score + deltaVoteCount },
            });
            return true;
        }
        else {
            if (isRemove)
                return true;
            await prisma.reaction.create({
                data: {
                    byId: userData.id,
                    emoji: input.emoji,
                    [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                },
            });
            const summaryTable = reactingFor.reactionSummaries.find((summary) => summary.emoji === input.emoji);
            if (summaryTable) {
                await prisma.reaction_summary.update({
                    where: { id: summaryTable.id },
                    data: { count: summaryTable.count + 1 },
                });
            }
            else {
                await prisma.reaction_summary.create({
                    data: {
                        emoji: input.emoji,
                        count: 1,
                        [`${forMapper[input.reactionFor]}Id`]: input.forConnect,
                    },
                });
            }
            await Trigger(prisma, userData.languages).objectReact(null, input.emoji, input.reactionFor, input.forConnect, userData.id);
            await prismaFor.update({
                where: { id: input.forConnect },
                data: { score: reactingFor.score + feelingNew },
            });
            return true;
        }
    },
});
//# sourceMappingURL=reaction.js.map