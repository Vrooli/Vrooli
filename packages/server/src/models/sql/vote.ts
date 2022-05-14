import { CODE, VoteFor } from "@local/shared";
import { CustomError } from "../../error";
import { LogType, Vote, VoteInput } from "../../schema/types";
import { PrismaType } from "../../types";
import { deconstructUnion, FormatConverter, GraphQLModelType } from "./base";
import _ from "lodash";
import { genErrorCode, logger, LogLevel } from "../../logger";
import { Log } from "../../models/nosql";

//==============================================================
/* #region Custom Components */
//==============================================================

export const voteFormatter = (): FormatConverter<Vote> => ({
    relationshipMap: {
        '__typename': GraphQLModelType.Vote,
        'from': GraphQLModelType.User,
        'to': {
            'Comment': GraphQLModelType.Comment,
            'Project': GraphQLModelType.Project,
            'Routine': GraphQLModelType.Routine,
            'Standard': GraphQLModelType.Standard,
            'Tag': GraphQLModelType.Tag,
        }
    },
    constructUnions: (data) => {
        let { comment, project, routine, standard, tag, ...modified } = data;
        if (comment) modified.to = comment;
        else if (project) modified.to = project;
        else if (routine) modified.to = routine;
        else if (standard) modified.to = standard;
        else if (tag) modified.to = tag;
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = deconstructUnion(partial, 'to', [
            [GraphQLModelType.Comment, 'comment'],
            [GraphQLModelType.Project, 'project'],
            [GraphQLModelType.Routine, 'routine'],
            [GraphQLModelType.Standard, 'standard'],
        ]);
        return modified;
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
const voter = (prisma: PrismaType) => ({
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
                console.log('before log vote a')
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
                console.log('before log vote b')
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
            console.log('before log vote c')
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
    async getIsUpvoteds(
        userId: string,
        ids: string[],
        voteFor: keyof typeof VoteFor
    ): Promise<Array<boolean | null>> {
        // Create result array that is the same length as ids
        const result = new Array(ids.length).fill(null);
        // Filter out nulls and undefineds from ids
        const idsFiltered = ids.filter(id => id !== null && id !== undefined);
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

export function VoteModel(prisma: PrismaType) {
    const prismaObject = prisma.vote;
    const format = voteFormatter();

    return {
        prisma,
        prismaObject,
        ...format,
        ...voter(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================