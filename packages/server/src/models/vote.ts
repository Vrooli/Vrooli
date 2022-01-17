import { CODE, VoteFor } from "@local/shared";
import { CustomError } from "../error";
import { VoteInput } from "schema/types";
import { PrismaType, RecursivePartial } from "../types";
import { FormatConverter, MODEL_TYPES } from "./base";
import { UserDB } from "./user";
import { CommentDB, ProjectDB, RoutineDB, StandardDB, TagDB } from "../models";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type VoteRelationshipList = 'comment' | 'project' | 'routine' | 'standard' | 'tag' | 'user';
// Type 2. QueryablePrimitives
export type VoteQueryablePrimitives = {};
// Type 3. AllPrimitives
export type VoteAllPrimitives = VoteQueryablePrimitives & {
    id: string;
    isUpvote: boolean;
    userId: string;
    commentId?: string;
    projectId?: string;
    routineId?: string;
    standardId?: string;
    tagId?: string;
};
// type 4. Database shape
export type VoteDB = VoteAllPrimitives & {
    user: UserDB;
    comment?: CommentDB;
    project?: ProjectDB;
    routine?: RoutineDB;
    standard?: StandardDB;
    tag?: TagDB;
}

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

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
const voter = (prisma?: PrismaType) => ({
    async castVote(userId: string, input: VoteInput): Promise<boolean> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Check if vote exists
        const vote = await prisma.vote.findFirst({ where: {
            userId,
            [forMapper[input.voteFor]]: input.forId
        }})
        if (vote) {
            // If vote is the same as the one we're trying to cast, skip
            if (vote.isUpvote === input.isUpvote) return true;
            // Otherwise, update the vote
            await prisma.vote.update({
                where: { id: vote.id },
                data: { isUpvote: input.isUpvote }
            })
            return true;
        }
        else {
            // Create the vote
            await prisma.vote.create({
                data: {
                    userId,
                    isUpvote: input.isUpvote,
                    [forMapper[input.voteFor]]: { connect: { id: input.forId } }
                }
            })
            return true;
        }
    },
    async removeVote(userId: string, input: any): Promise<boolean> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        // Check if vote exists
        const vote = await prisma.vote.findFirst({ where: {
            userId,
            [forMapper[input.voteFor]]: input.forId
        }})
        if (!vote) return false;
        // Delete the vote
        await prisma.vote.delete({ where: { id: vote.id } });
        return true;
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function VoteModel(prisma?: PrismaType) {
    return {
        prisma,
        ...voter(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================