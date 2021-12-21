import { CODE } from "@local/shared";
import { CustomError } from "error";
import { Comment, CommentInput, User, VoteInput } from "schema/types";
import { RecursivePartial } from "types";
import { BaseState, creater, deleter, findByIder, FormatConverter, MODEL_TYPES, reporter, updater } from "./base";

//======================================================================================================================
/* #region Type Definitions */
//======================================================================================================================

// Type 1. RelationshipList
export type CommentRelationshipList = 'user' | 'organization' | 'project' | 'resource' |
    'routine' | 'standard' | 'reports' | 'stars' | 'votes';
// Type 2. QueryablePrimitives
export type CommentQueryablePrimitives = Omit<Comment, CommentRelationshipList>;
// Type 3. AllPrimitives
export type CommentAllPrimitives = CommentQueryablePrimitives;
// type 4. FullModel
export type CommentFullModel = CommentAllPrimitives &
    Pick<Comment, 'user' | 'organization' | 'project' | 'resource' | 'routine' | 'standard' | 'reports'> &
{
    stars: number,
    votes: number,
};

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
 const auther = (state: any) => ({
    /**
     * Determines if the user is allowed to edit/delete the comment
     * @param commentId The comment's ID
     * @param userId The user's ID
     */
    async isAuthenticatedToModify(commentId: string, userId: string): Promise<boolean> {
        // Check for valid inputs
        if (!commentId || !userId) return false;
        // Find rows that match the comment ID and user ID (should only be one or none)
        const comments = await state.prisma.comment.findMany({ where: { id: commentId, userId } });
        return Array.isArray(comments) && comments.length > 0;
    },
    /**
     * Determines if the user is allowed to vote on a comment
     * @param commentId The comment's ID
     * @param userId The user's ID
     */
     async isAuthenticatedToVote(commentId: string, userId: string): Promise<boolean> {
        // Check for valid inputs
        if (!commentId || !userId) return false;
        // Find comment in database
        const comment = await state.prisma.comment.findOne({ where: { id: commentId } });
        // Verify that comment was not created by the user. If the user has voted on it before,
        // it's fine (could be switching from upvote to downvote, etc.)
        return comment.userId !== userId;
    }
})

/**
 * Component for comment voting
 * @param state 
 * @returns 
 */
 const voter = ({ prisma }: BaseState<Comment>) => ({
    /**
     * Adds a vote to the comment
     * @param input GraphQL vote input
     * @param userId The user's ID
     * @return True if vote counted (even if it was a duplicate)
     */
    async vote(input: VoteInput, userId: string): Promise<boolean> {
        // Check for valid inputs
        if (!input.id || !userId || !prisma) return false;
        // Check for existing votes (should only be one or none)
        const existingVotes = await prisma.comment_votes.findMany({
            where: { votedId: input.id ?? '', voterId: userId },
            select: { id: true, isUpvote: true}
        });
        // If only one vote exists, update it (if switching between upvote/downvote)
        if (Array.isArray(existingVotes) && existingVotes.length === 1) {
            // If the vote is the same, return as success
            if (existingVotes[0].isUpvote === input.isUpvote) return true;
            // Otherwise, update the vote
            await prisma.comment_votes.update({
                where: { id: existingVotes[0].id },
                data: { isUpvote: input.isUpvote }
            })
            return true;
        }
        // If multiple votes exist (shouldn't hit this case, but you never know🤷‍♂️), delete them
        if (Array.isArray(existingVotes) && existingVotes.length > 0) {
            await prisma.comment_votes.deleteMany({
                where: { id: { in: existingVotes.map(vote => vote.id) } }
            })
        }
        // If here, no votes exist, so create a new one
        await prisma.comment_votes.create({
            data: {
                isUpvote: input.isUpvote,
                voterId: userId,
                votedId: input.id
            }
        })
        return true;
    }
})

//==============================================================
/* #endregion Custom Components */
//==============================================================

//==============================================================
/* #region Model */
//==============================================================

export function CommentModel(prisma?: any) {
    let obj: BaseState<Comment> = {
        prisma,
        model: MODEL_TYPES.Comment,
        format: formatter(),
    }

    return {
        ...obj,
        ...auther(obj),
        ...findByIder<Comment>(obj),
        ...formatter(),
        ...creater<CommentInput, Comment>(obj),
        ...updater<CommentInput, Comment>(obj),
        ...deleter(obj),
        ...reporter(),
        ...voter(obj),
    }
}

//==============================================================
/* #endregion Model */
//==============================================================