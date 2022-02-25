import { CODE, VoteFor } from "@local/shared";
import { CustomError } from "../error";
import { Vote, VoteInput } from "schema/types";
import { PartialSelectConvert, PrismaType, RecursivePartial } from "../types";
import { BaseType, deconstructUnion, FormatConverter, FormatterMap, infoToPartialSelect, InfoType, MODEL_TYPES, relationshipFormatter } from "./base";
import _ from "lodash";
import { vote } from "@prisma/client";
import { standardDBFields } from "./standard";
import { commentDBFields } from "./comment";
import { projectDBFields } from "./project";
import { routineDBFields } from "./routine";

//==============================================================
/* #region Custom Components */
//==============================================================

type VoteFormatterType = FormatConverter<Vote, vote>;
/**
 * Component for formatting between graphql and prisma types
 */
export const voteFormatter = (): VoteFormatterType => {
    return {
        dbShape: (partial: PartialSelectConvert<Vote>): PartialSelectConvert<vote> => {
            let modified = partial;
            console.log('vote toDB', modified);
            // Deconstruct GraphQL unions
            modified = deconstructUnion(modified, 'to', [
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
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['from', FormatterMap.User.dbShapeUser],
            ]);
            return modified;
        },
        dbPrune: (info: InfoType): PartialSelectConvert<vote> => {
            // Convert GraphQL info object to a partial select object
            let modified = infoToPartialSelect(info);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['from', FormatterMap.User.dbPruneUser],
            ]);
            return modified;
        },
        selectToDB: (info: InfoType): PartialSelectConvert<vote> => {
            return voteFormatter().dbShape(voteFormatter().dbPrune(info));
        },
        selectToGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Vote> => {
            console.log('voteFormatter.toGraphQL start', obj);
            // Create unions
            let { comment, project, routine, standard, ...modified } = obj;
            if (comment) modified.to = FormatterMap.Comment.selectToGraphQL(comment);
            else if (project) modified.to = FormatterMap.Project.selectToGraphQL(project);
            else if (routine) modified.to = FormatterMap.Routine.selectToGraphQL(routine);
            else if (standard) modified.to = FormatterMap.Standard.selectToGraphQL(standard);
            // Convert relationships
            modified = relationshipFormatter(modified, [
                ['from', FormatterMap.User.selectToGraphQLUser],
            ]);
            console.log('voteFormatter.toGraphQL finished', modified);
            return modified;
        },
    }
}

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
        const prismaFor = (prisma[forMapper[input.voteFor] as keyof PrismaType] as BaseType);
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
    const model = MODEL_TYPES.Vote;
    const format = voteFormatter();

    return {
        prisma,
        model,
        ...format,
        ...voter(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================