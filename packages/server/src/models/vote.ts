import { CustomError, Trigger } from "../events";
import { SessionUser, Vote, VoteFor, VoteInput, VoteSearchInput, VoteSortBy } from "../endpoints/types";
import { PrismaType } from "../types";
import { ModelLogic } from "./types";
import { ApiModel, CommentModel, IssueModel, NoteModel, PostModel, ProjectModel, QuestionAnswerModel, QuestionModel, QuizModel, RoutineModel, SmartContractModel, StandardModel } from ".";
import { SelectWrap } from "../builders/types";
import { onlyValidIds, padSelect } from "../builders";
import { Prisma } from "@prisma/client";

const forMapper: { [key in VoteFor]: string } = {
    Api: 'api',
    Comment: 'comment',
    Issue: 'issue',
    Note: 'note',
    Post: 'post',
    Project: 'project',
    Question: 'question',
    QuestionAnswer: 'questionAnswer',
    Quiz: 'quiz',
    Routine: 'routine',
    SmartContract: 'smartContract',
    Standard: 'standard',
}

const __typename = 'Vote' as const;
const suppFields = [] as const;
export const VoteModel: ModelLogic<{
    IsTransferable: true,
    IsVersioned: true,
    GqlCreate: undefined,
    GqlUpdate: undefined,
    GqlModel: Vote,
    GqlSearch: VoteSearchInput,
    GqlSort: VoteSortBy,
    GqlPermission: any,
    PrismaCreate: Prisma.voteUpsertArgs['create'],
    PrismaUpdate: Prisma.voteUpsertArgs['update'],
    PrismaModel: Prisma.voteGetPayload<SelectWrap<Prisma.voteSelect>>,
    PrismaSelect: Prisma.voteSelect,
    PrismaWhere: Prisma.voteWhereInput,
}, typeof suppFields> = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.vote,
    display: {
        select: () => ({
            id: true,
            api: padSelect(ApiModel.display.select),
            comment: padSelect(CommentModel.display.select),
            issue: padSelect(IssueModel.display.select),
            note: padSelect(NoteModel.display.select),
            post: padSelect(PostModel.display.select),
            project: padSelect(ProjectModel.display.select),
            question: padSelect(QuestionModel.display.select),
            questionAnswer: padSelect(QuestionAnswerModel.display.select),
            quiz: padSelect(QuizModel.display.select),
            routine: padSelect(RoutineModel.display.select),
            smartContract: padSelect(SmartContractModel.display.select),
            standard: padSelect(StandardModel.display.select),
        }),
        label: (select, languages) => {
            if (select.api) return ApiModel.display.label(select.api as any, languages);
            if (select.comment) return CommentModel.display.label(select.comment as any, languages);
            if (select.issue) return IssueModel.display.label(select.issue as any, languages);
            if (select.note) return NoteModel.display.label(select.note as any, languages);
            if (select.post) return PostModel.display.label(select.post as any, languages);
            if (select.project) return ProjectModel.display.label(select.project as any, languages);
            if (select.question) return QuestionModel.display.label(select.question as any, languages);
            if (select.questionAnswer) return QuestionAnswerModel.display.label(select.questionAnswer as any, languages);
            if (select.quiz) return QuizModel.display.label(select.quiz as any, languages);
            if (select.routine) return RoutineModel.display.label(select.routine as any, languages);
            if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages);
            if (select.standard) return StandardModel.display.label(select.standard as any, languages);
            return '';
        }
    },
    format: {
        gqlRelMap: {
            __typename,
            by: 'User',
            to: {
                api: 'Api',
                comment: 'Comment',
                issue: 'Issue',
                note: 'Note',
                post: 'Post',
                project: 'Project',
                question: 'Question',
                questionAnswer: 'QuestionAnswer',
                quiz: 'Quiz',
                routine: 'Routine',
                smartContract: 'SmartContract',
                standard: 'Standard',
            }
        },
        prismaRelMap: {
            __typename,
            by: 'User',
            api: 'Api',
            comment: 'Comment',
            issue: 'Issue',
            note: 'Note',
            post: 'Post',
            project: 'Project',
            question: 'Question',
            questionAnswer: 'QuestionAnswer',
            quiz: 'Quiz',
            routine: 'Routine',
            smartContract: 'SmartContract',
            standard: 'Standard',
        },
        countFields: {},
    },
    query: {
        async getIsUpvoteds(
            prisma: PrismaType,
            userId: string | null | undefined,
            ids: string[],
            voteFor: keyof typeof VoteFor
        ): Promise<Array<boolean | null>> {
            // Create result array that is the same length as ids
            const result = new Array(ids.length).fill(null);
            // If userId not provided, return result
            if (!userId) return result;
            // Filter out nulls and undefineds from ids
            const idsFiltered = onlyValidIds(ids);
            const fieldName = `${voteFor.toLowerCase()}Id`;
            const isUpvotedArray = await prisma.vote.findMany({ where: { byId: userId, [fieldName]: { in: idsFiltered } } });
            // Replace the nulls in the result array with true or false
            for (let i = 0; i < ids.length; i++) {
                // Try to find this id in the isUpvoted array
                if (ids[i] !== null && ids[i] !== undefined) {
                    // If found, set result index to value of isUpvote field
                    result[i] = isUpvotedArray.find((vote: any) => vote[fieldName] === ids[i])?.isUpvote ?? null;
                }
            }
            return result;
        },
    },
    /**
     * Casts votes. Makes sure not to duplicate, and to override existing votes. 
     * A user may vote on their own project/routine/etc.
     * @returns True if cast correctly (even if skipped because of duplicate)
     */
    vote: async (prisma: PrismaType, userData: SessionUser, input: VoteInput): Promise<boolean> => {
        // Define prisma type for voted-on object
        const prismaFor = (prisma[forMapper[input.voteFor] as keyof PrismaType] as any);
        // Check if object being voted on exists
        const votingFor: null | { id: string, score: number } = await prismaFor.findUnique({ where: { id: input.forConnect }, select: { id: true, score: true } });
        if (!votingFor)
            throw new CustomError('0118', 'NotFound', userData.languages, { voteFor: input.voteFor, forId: input.forConnect });
        // Check if vote exists
        const vote = await prisma.vote.findFirst({
            where: {
                byId: userData.id,
                [`${forMapper[input.voteFor]}Id`]: input.forConnect
            }
        })
        // If vote already existed
        if (vote) {
            // If vote is the same as the one we're trying to cast, skip
            if (vote.isUpvote === input.isUpvote) return true;
            // If setting note to null, delete it
            if (input.isUpvote === null) {
                // Delete vote
                await prisma.vote.delete({ where: { id: vote.id } })
                // Handle trigger
                await Trigger(prisma, userData.languages).objectVote(false, input.voteFor, input.forConnect, userData.id);
            }
            // Otherwise, update the vote
            else {
                await prisma.vote.update({
                    where: { id: vote.id },
                    data: { isUpvote: input.isUpvote }
                })
                // Handle trigger
                await Trigger(prisma, userData.languages).objectVote(input.isUpvote ?? null, input.voteFor, input.forConnect, userData.id);
            }
            // Update the score
            const oldVoteCount = vote.isUpvote ? 1 : vote.isUpvote === null ? 0 : -1;
            const newVoteCount = input.isUpvote ? 1 : input.isUpvote === null ? 0 : -1;
            const deltaVoteCount = newVoteCount - oldVoteCount;
            await prismaFor.update({
                where: { id: input.forConnect },
                data: { score: votingFor.score + deltaVoteCount }
            })
            return true;
        }
        // If vote did not already exist
        else {
            // If setting to null, skip
            if (input.isUpvote === null || input.isUpvote === undefined) return true;
            // Create the vote
            await prisma.vote.create({
                data: {
                    byId: userData.id,
                    isUpvote: input.isUpvote,
                    [`${forMapper[input.voteFor]}Id`]: input.forConnect
                }
            })
            // Handle trigger
            await Trigger(prisma, userData.languages).objectVote(input.isUpvote, input.voteFor, input.forConnect, userData.id);
            // Update the score
            const voteCount = input.isUpvote ? 1 : input.isUpvote === null ? 0 : -1;
            await prismaFor.update({
                where: { id: input.forConnect },
                data: { score: votingFor.score + voteCount }
            })
            return true;
        }
    },
})