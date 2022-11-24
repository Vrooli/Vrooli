import { VoteFor } from "@shared/consts";
import { isObject } from "@shared/utils";
import { CustomError, Trigger } from "../events";
import { resolveVoteTo } from "../schema/resolvers";
import { SessionUser, Vote, VoteInput } from "../schema/types";
import { PrismaType } from "../types";
import { readManyHelper } from "./actions";
import { ObjectMap, onlyValidIds } from "./builder";
import { Formatter, GraphQLModelType, ModelLogic, PartialGraphQLInfo } from "./types";

const formatter = (): Formatter<Vote, 'to'> => ({
    relationshipMap: {
        __typename: 'Vote',
        from: 'User',
        to: {
            Comment: 'Comment',
            Project: 'Project',
            Routine: 'Routine',
            Standard: 'Standard',
            Tag: 'Tag',
        }
    },
    supplemental: {
        graphqlFields: ['to'],
        toGraphQL: ({ languages, objects, partial, prisma, userData }) => [
            ['to', async () => {
                if (!userData) return new Array(objects.length).fill([]);
                // Query for data that star is applied to
                if (isObject(partial.to)) {
                    const toTypes: GraphQLModelType[] = objects.map(o => resolveVoteTo(o.to)).filter(t => t);
                    const toIds = objects.map(x => x.to?.id ?? '') as string[];
                    // Group ids by types
                    const toIdsByType: { [x: string]: string[] } = {};
                    toTypes.forEach((type, i) => {
                        if (!toIdsByType[type]) toIdsByType[type] = [];
                        toIdsByType[type].push(toIds[i]);
                    })
                    // Query for each type
                    const tos: any[] = [];
                    for (const type of Object.keys(toIdsByType)) {
                        const validTypes: GraphQLModelType[] = [
                            'Comment',
                            'Organization',
                            'Project',
                            'Routine',
                            'Standard',
                            'Tag',
                            'User',
                        ];
                        if (!validTypes.includes(type as GraphQLModelType)) {
                            throw new CustomError('0321', 'InternalError', languages, { type });
                        }
                        const model: ModelLogic<any, any, any, any> = ObjectMap[type] as ModelLogic<any, any, any, any>;
                        const paginated = await readManyHelper({
                            info: partial.to[type] as PartialGraphQLInfo,
                            input: { ids: toIdsByType[type] },
                            model,
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
    Comment: 'comment',
    Project: 'project',
    Routine: 'routine',
    Standard: 'standard',
    Tag: 'tag',
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

export const VoteModel = ({
    delegate: (prisma: PrismaType) => prisma.vote,
    format: formatter(),
    query: querier(),
    type: 'Vote' as GraphQLModelType,
    vote,
})