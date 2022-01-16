import { CODE } from "@local/shared";
import { CustomError } from "../error";
import { Comment, CommentInput, VoteInput } from "schema/types";
import { PrismaType, RecursivePartial } from "types";
import { creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type CommentRelationshipList = 'user' | 'organization' | 'project' |
    'routine' | 'standard' | 'reports' | 'stars' | 'votes';
// Type 2. QueryablePrimitives
export type CommentQueryablePrimitives = Omit<Comment, CommentRelationshipList>;
// Type 3. AllPrimitives
export type CommentAllPrimitives = CommentQueryablePrimitives;
// type 4. Database shape
export type CommentDB = any;
// export type CommentDB = CommentAllPrimitives &
//     Pick<Comment, 'user' | 'organization' | 'project' | 'routine' | 'standard' | 'reports' | 'votes'> &
// {
//     stars: number,
// };

//======================================================================================================================
/* #endregion Type Definitions */
//======================================================================================================================

//==============================================================
/* #region Custom Components */
//==============================================================

/**
 * Component for formatting between graphql and prisma types
 */
const formatter = (): FormatConverter<any, any> => ({
    toDB: (obj: RecursivePartial<Comment>): RecursivePartial<any> => obj as any,
    toGraphQL: (obj: RecursivePartial<any>): RecursivePartial<Comment> => obj as any
})

/**
 * Component for comment authentication
 * @param state 
 * @returns 
 */
const auther = (prisma?: PrismaType) => ({
    /**
     * Determines if the user is allowed to edit/delete the comment
     * @param commentId The comment's ID
     * @param userId The user's ID
     */
    async isAuthenticatedToModify(commentId: string, userId: string): Promise<boolean> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        if (!commentId || !userId) return false;
        // Find rows that match the comment ID and user ID (should only be one or none)
        const comments = await prisma.comment.findMany({ where: { id: commentId, userId } });
        return Array.isArray(comments) && comments.length > 0;
    },
    /**
     * Determines if the user is allowed to vote on a comment
     * @param commentId The comment's ID
     * @param userId The user's ID
     */
    async isAuthenticatedToVote(commentId: string, userId: string): Promise<boolean> {
        // Check for valid arguments
        if (!prisma) throw new CustomError(CODE.InvalidArgs);
        if (!commentId || !userId) return false;
        // Find comment in database
        const comment = await prisma.comment.findFirst({ where: { id: commentId } });
        // Verify that comment was not created by the user. If the user has voted on it before,
        // it's fine (could be switching from upvote to downvote, etc.)
        return comment ? comment.userId !== userId : false;
    }
})

/**
 * Component for comment voting
 * @param state 
 * @returns 
 */
const voter = (prisma?: PrismaType) => ({
        /**
         * Adds a vote to the comment
         * @param input GraphQL vote input
         * @param userId The user's ID
         * @return True if vote counted (even if it was a duplicate)
         */
        async vote(input: VoteInput, userId: string): Promise<boolean> {
            // // Check for valid inputs
            // if (!prisma) throw new CustomError(CODE.InvalidArgs);
            // if (!input.id || !userId) return false;
            // // Check for existing votes (should only be one or none)
            // const existingVotes = await prisma.comment_votes.findMany({
            //     where: { votedId: input.id ?? '', voterId: userId },
            //     select: { id: true, isUpvote: true }
            // });
            // // If only one vote exists, update it (if switching between upvote/downvote)
            // if (Array.isArray(existingVotes) && existingVotes.length === 1) {
            //     // If the vote is the same, return as success
            //     if (existingVotes[0].isUpvote === input.isUpvote) return true;
            //     // Otherwise, update the vote
            //     await prisma.comment_votes.update({
            //         where: { id: existingVotes[0].id },
            //         data: { isUpvote: input.isUpvote }
            //     })
            //     return true;
            // }
            // // If multiple votes exist (shouldn't hit this case, but you never know🤷‍♂️), delete them
            // if (Array.isArray(existingVotes) && existingVotes.length > 0) {
            //     await prisma.comment_votes.deleteMany({
            //         where: { id: { in: existingVotes.map(vote => vote.id) } }
            //     })
            // }
            // // If here, no votes exist, so create a new one
            // await prisma.comment_votes.create({
            //     data: {
            //         isUpvote: input.isUpvote,
            //         voterId: userId,
            //         votedId: input.id
            //     }
            // })
            return true;
        }
    })

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function CommentModel(prisma?: PrismaType) {
    const model = MODEL_TYPES.Comment;
    const format = formatter();

    return {
        prisma,
        model,
        ...format,
        ...auther(prisma),
        ...creater<CommentInput, Comment, CommentDB>(model, format.toDB, prisma),
        ...deleter(model, prisma),
        ...findByIder<Comment, CommentDB>(model, format.toDB, prisma),
        ...reporter(),
        ...updater<CommentInput, Comment, CommentDB>(model, format.toDB, prisma),
        ...voter(prisma),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================