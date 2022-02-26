import { CODE, VoteFor } from "@local/shared";
import { CustomError } from "../error";
import { Vote, VoteInput } from "schema/types";
import { PrismaType } from "../types";
import { deconstructUnion, FormatConverter } from "./base";
import _ from "lodash";
import { standardDBFields } from "./standard";
import { commentDBFields } from "./comment";
import { projectDBFields } from "./project";
import { routineDBFields } from "./routine";

//==============================================================
/* #region Custom Components */
//==============================================================

export const voteFormatter = (): FormatConverter<Vote> => ({
    constructUnions: (data) => {
        let { comment, project, routine, standard, ...modified } = data;
        if (comment) modified.to = comment;
        else if (project) modified.to = project;
        else if (routine) modified.to = routine;
        else if (standard) modified.to = standard;
        console.log('voteFormatter.toGraphQL finished', modified);
        return modified;
    },
    deconstructUnions: (partial) => {
        let modified = deconstructUnion(partial, 'to', [
            ['comment', {
                ...Object.fromEntries(commentDBFields.map(f => [f, true]))
            }],
            ['project', {
                ...Object.fromEntries(projectDBFields.map(f => [f, true]))
            }],
            ['routine', {
                ...Object.fromEntries(routineDBFields.map(f => [f, true]))
            }],
            ['standard', {
                ...Object.fromEntries(standardDBFields.map(f => [f, true]))
            }],
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
        if (!votingFor) throw new CustomError(CODE.ErrorUnknown);
        // Check if vote exists
        const vote = await prisma.vote.findFirst({
            where: {
                byId: userId,
                [`${forMapper[input.voteFor]}Id`]: input.forId
            }
        })
        console.log('existing vote', vote)
        // If vote already existed
        if (vote) {
            // If vote is the same as the one we're trying to cast, skip
            if (vote.isUpvote === input.isUpvote) return true;
            // If setting note to null, delete it
            if (input.isUpvote === null) {
                // Delete vote
                await prisma.vote.delete({ where: { id: vote.id } })
            }
            // Otherwise, update the vote
            else {
                await prisma.vote.update({
                    where: { id: vote.id },
                    data: { isUpvote: input.isUpvote }
                })
            }
            // Update the score
            const oldVoteCount = vote.isUpvote ? 1 : vote.isUpvote === null ? 0 : -1;
            const newVoteCount = input.isUpvote ? 1 : input.isUpvote === null ? 0 : -1;
            const deltaVoteCount = newVoteCount - oldVoteCount;
            console.log('deltaVoteCount', deltaVoteCount)
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
        voteFor: VoteFor
    ): Promise<Array<boolean | null>> {
        const fieldName = `${voteFor.toLowerCase()}Id`;
        const isUpvotedArray = await prisma.vote.findMany({ where: { byId: userId, [fieldName]: { in: ids } } });
        return ids.map(id => {
            return isUpvotedArray.find((x: any) => x[fieldName] === id)?.isUpvote ?? null
        });
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
        prismaObject,
        ...format,
        ...voter(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================