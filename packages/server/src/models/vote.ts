import { VoteFor } from "@shared/consts";
import { isObject } from "@shared/utils";
import { CustomError, Trigger } from "../events";
import { resolveUnion } from "../endpoints/resolvers";
import { SessionUser, Vote, VoteInput } from "../endpoints/types";
import { PrismaType } from "../types";
import { readManyHelper } from "../actions";
import { Displayer, Formatter, GraphQLModelType } from "./types";
import { CommentModel, ProjectModel, RoutineModel, StandardModel } from ".";
import { PartialGraphQLInfo, SelectWrap } from "../builders/types";
import { onlyValidIds, padSelect } from "../builders";
import { Prisma } from "@prisma/client";

const __typename = 'Vote' as const;

const suppFields = ['to'] as const;
const formatter = (): Formatter<Vote, typeof suppFields> => ({
    relationshipMap: {
        __typename,
        from: 'User',
        to: ['Api', 'Comment', 'Issue', 'Note', 'Post', 'Project', 'Question', 'QuestionAnswer', 'Quiz', 'Routine', 'SmartContract', 'Standard']
    },
    supplemental: {
        graphqlFields: suppFields,
        toGraphQL: ({ languages, objects, partial, prisma, userData }) => [
            ['to', async () => {
                if (!userData) return new Array(objects.length).fill([]);
                // Query for data that star is applied to
                if (isObject(partial.to)) {
                    const toTypes: GraphQLModelType[] = objects.map(o => resolveUnion(o.to)).filter(t => t);
                    const toIds = objects.map(x => x.to?.id ?? '') as string[];
                    // Group ids by types
                    const toIdsByType: { [x: string]: string[] } = {};
                    toTypes.forEach((type, i) => {
                        if (!toIdsByType[type]) toIdsByType[type] = [];
                        toIdsByType[type].push(toIds[i]);
                    })
                    // Query for each type
                    const tos: any[] = [];
                    for (const objectType of Object.keys(toIdsByType)) {
                        const validTypes: GraphQLModelType[] = [
                            'Comment',
                            'Organization',
                            'Project',
                            'Routine',
                            'Standard',
                            'Tag',
                            'User',
                        ];
                        if (!validTypes.includes(objectType as GraphQLModelType)) {
                            throw new CustomError('0321', 'InternalError', languages, { objectType });
                        }
                        const paginated = await readManyHelper({
                            info: partial.to[objectType] as PartialGraphQLInfo,
                            input: { ids: toIdsByType[objectType] },
                            objectType: objectType as GraphQLModelType,
                            prisma,
                            req: { languages, users: [userData] }
                        })
                        tos.push(...paginated.edges.map(x => x.node));
                    }
                    // Apply each "to" to the "to" property of each object
                    for (const object of objects) {
                        // Find the correct "to", using object.to.id
                        const to = tos.find(x => x.id === object.to.id);
                        object.to = to;
                    }
                }
                return objects;
            }],
        ],
    },
})

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

/**
 * Casts votes. Makes sure not to duplicate, and to override existing votes. 
 * A user may vote on their own project/routine/etc.
 * @returns True if cast correctly (even if skipped because of duplicate)
 */
const vote = async (prisma: PrismaType, userData: SessionUser, input: VoteInput): Promise<boolean> => {
    // Define prisma type for voted-on object
    const prismaFor = (prisma[forMapper[input.voteFor] as keyof PrismaType] as any);
    // Check if object being voted on exists
    const votingFor: null | { id: string, score: number } = await prismaFor.findUnique({ where: { id: input.forId }, select: { id: true, score: true } });
    if (!votingFor)
        throw new CustomError('0118', 'NotFound', userData.languages, { voteFor: input.voteFor, forId: input.forId });
    // Check if vote exists
    const vote = await prisma.vote.findFirst({
        where: {
            byId: userData.id,
            [`${forMapper[input.voteFor]}Id`]: input.forId
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
            await Trigger(prisma, userData.languages).objectVote(false, input.voteFor, input.forId, userData.id);
        }
        // Otherwise, update the vote
        else {
            await prisma.vote.update({
                where: { id: vote.id },
                data: { isUpvote: input.isUpvote }
            })
            // Handle trigger
            await Trigger(prisma, userData.languages).objectVote(input.isUpvote ?? null, input.voteFor, input.forId, userData.id);
        }
        // Update the score
        const oldVoteCount = vote.isUpvote ? 1 : vote.isUpvote === null ? 0 : -1;
        const newVoteCount = input.isUpvote ? 1 : input.isUpvote === null ? 0 : -1;
        const deltaVoteCount = newVoteCount - oldVoteCount;
        await prismaFor.update({
            where: { id: input.forId },
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
                [`${forMapper[input.voteFor]}Id`]: input.forId
            }
        })
        // Handle trigger
        await Trigger(prisma, userData.languages).objectVote(input.isUpvote, input.voteFor, input.forId, userData.id);
        // Update the score
        const voteCount = input.isUpvote ? 1 : input.isUpvote === null ? 0 : -1;
        await prismaFor.update({
            where: { id: input.forId },
            data: { score: votingFor.score + voteCount }
        })
        return true;
    }
}

const querier = () => ({
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
})

const displayer = (): Displayer<
    Prisma.voteSelect,
    Prisma.voteGetPayload<SelectWrap<Prisma.voteSelect>>
> => ({
    select: () => ({
        id: true,
        // api: padSelect(ApiModel.display.select),
        comment: padSelect(CommentModel.display.select),
        // issue: padSelect(IssueModel.display.select),
        // post: padSelect(PostModel.display.select),
        project: padSelect(ProjectModel.display.select),
        // question: padSelect(QuestionModel.display.select),
        // questionAnswer: padSelect(QuestionAnswerModel.display.select),
        // quiz: padSelect(QuizModel.display.select),
        routine: padSelect(RoutineModel.display.select),
        // smartContract: padSelect(SmartContractModel.display.select),
        standard: padSelect(StandardModel.display.select),
    }),
    label: (select, languages) => {
        // if (select.api) return ApiModel.display.label(select.api as any, languages);
        if (select.comment) return CommentModel.display.label(select.comment as any, languages);
        // if (select.issue) return IssueModel.display.label(select.issue as any, languages);
        // if (select.post) return PostModel.display.label(select.post as any, languages);
        if (select.project) return ProjectModel.display.label(select.project as any, languages);
        // if (select.question) return QuestionModel.display.label(select.question as any, languages);
        // if (select.questionAnswer) return QuestionAnswerModel.display.label(select.questionAnswer as any, languages);
        // if (select.quiz) return QuizModel.display.label(select.quiz as any, languages);
        if (select.routine) return RoutineModel.display.label(select.routine as any, languages);
        // if (select.smartContract) return SmartContractModel.display.label(select.smartContract as any, languages);
        if (select.standard) return StandardModel.display.label(select.standard as any, languages);
        return '';
    }
})

export const VoteModel = ({
    __typename,
    delegate: (prisma: PrismaType) => prisma.vote,
    display: displayer(),
    format: formatter(),
    query: querier(),
    vote,
})