import { CODE, VoteFor } from "@shared/consts";
import { CustomError } from "../../error";
import { LogType, Vote, VoteInput } from "../../schema/types";
import { PrismaType } from "../../types";
import { FormatConverter, onlyValidIds } from "./base";
import { genErrorCode, logger, LogLevel } from "../../logger";
import { Log } from "../../models/nosql";
import { GraphQLModelType } from ".";

//==============================================================
/* #region Custom Components */
//==============================================================

export const voteFormatter = (): FormatConverter<Vote, any> => ({
    relationshipMap: {
        '__typename': 'Vote',
        'from': 'User',
        'to': {
            'Comment': 'Comment',
            'Project': 'Project',
            'Routine': 'Routine',
            'Standard': 'Standard',
            'Tag': 'Tag',
        }
    },
    unionMap: {
        'to': {
            'Comment': 'comment',
            'Project': 'project',
            'Routine': 'routine',
            'Standard': 'standard',
        },
    },
})

const forMapper = {
    [VoteFor.Comment]: 'comment',
    [VoteFor.Project]: 'project',
    [VoteFor.Routine]: 'routine',
    [VoteFor.Standard]: 'standard',
    [VoteFor.Tag]: 'tag',
}

/**
 * Casts votes. Makes sure not to duplicate, and to override existing votes. 
 * A user may vote on their own project/routine/etc.
 * @returns True if cast correctly (even if skipped because of duplicate)
 */
const voteMutater = (prisma: PrismaType) => ({
    async vote(userId: string, input: VoteInput): Promise<boolean> {
        // Define prisma type for voted-on object
        const prismaFor = (prisma[forMapper[input.voteFor] as keyof PrismaType] as any);
        // Check if object being voted on exists
        const votingFor: null | { id: string, score: number } = await prismaFor.findUnique({ where: { id: input.forId }, select: { id: true, score: true } });
        if (!votingFor)
            throw new CustomError(CODE.ErrorUnknown, 'Could not find object being voted on', { code: genErrorCode('0118') });
        // Check if vote exists
        const vote = await prisma.vote.findFirst({
            where: {
                byId: userId,
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
                // Log remove vote
                Log.collection.insertOne({
                    timestamp: Date.now(),
                    userId: userId,
                    action: LogType.RemoveVote,
                    object1Type: input.voteFor,
                    object1Id: input.forId,
                }).catch(error => logger.log(LogLevel.error, 'Failed creating "Remove Vote" log', { code: genErrorCode('0204'), error }));
            }
            // Otherwise, update the vote
            else {
                await prisma.vote.update({
                    where: { id: vote.id },
                    data: { isUpvote: input.isUpvote }
                })
                // Log vote/unvote
                Log.collection.insertOne({
                    timestamp: Date.now(),
                    userId: userId,
                    action: input.isUpvote ? LogType.Upvote : LogType.Downvote,
                    object1Type: input.voteFor,
                    object1Id: input.forId,
                }).catch(error => logger.log(LogLevel.error, 'Failed creating "Upvote/Downvote" log', { code: genErrorCode('0205'), error }));
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
            if (input.isUpvote === null) return true;
            // Create the vote
            await prisma.vote.create({
                data: {
                    byId: userId,
                    isUpvote: input.isUpvote,
                    [`${forMapper[input.voteFor]}Id`]: input.forId
                }
            })
            // Log vote
            Log.collection.insertOne({
                timestamp: Date.now(),
                userId: userId,
                action: input.isUpvote ? LogType.Upvote : LogType.Downvote,
                object1Type: input.voteFor,
                object1Id: input.forId,
            }).catch(error => logger.log(LogLevel.error, 'Failed creating "Upvote/Downvote" log', { code: genErrorCode('0206'), error }));
            // Update the score
            const voteCount = input.isUpvote ? 1 : input.isUpvote === null ? 0 : -1;
            await prismaFor.update({
                where: { id: input.forId },
                data: { score: votingFor.score + voteCount }
            })
            return true;
        }
    },
})

const voteQuerier = (prisma: PrismaType) => ({
    async getIsUpvoteds(
        userId: string | null,
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

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export const VoteModel = ({
    prismaObject: (prisma: PrismaType) => prisma.vote,
    format: voteFormatter(),
    mutate: voteMutater,
    query: voteQuerier,
    type: 'Vote' as GraphQLModelType,
})

//==============================================================
/* #endregion Model */
//==============================================================